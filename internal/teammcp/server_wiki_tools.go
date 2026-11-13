package teammcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nex-crm/wuphf/internal/team"
)

const humanWikiDelegationMaxAge = 24 * time.Hour

const directWikiIntentMaxGapWords = 5

var (
	directWikiVerbRE     = regexp.MustCompile(`(?i)\b(write|add|put|save|record|preserve|publish|create|update)\b`)
	directWikiTargetRE   = regexp.MustCompile(`(?i)\b(wiki|kb|knowledge\s+base)\b`)
	directWikiNegationRE = regexp.MustCompile(`(?i)\b(do not|don't|dont|never)\b`)
)

// handleTeamWikiWrite posts the article to the broker's wiki worker queue.
// Queue saturation surfaces as a tool error so the bot sees it and retries
// on the next turn — no hidden retries.
func handleTeamWikiWrite(ctx context.Context, _ *mcp.CallToolRequest, args TeamWikiWriteArgs) (*mcp.CallToolResult, any, error) {
	slug, err := resolveSlug(args.MySlug)
	if err != nil {
		return toolError(err), nil, nil
	}
	if !systemSkillEnabledFor(ctx, systemSkillWikiMaintenance, slug) {
		return toolError(systemSkillDisabledError(systemSkillWikiMaintenance, slug)), nil, nil
	}
	// The Librarian owns the wiki (Phase 4): writing, formatting, and organizing
	// canonical articles is its job, so it writes directly without the per-write
	// human-delegation gate that other bots need. Other bots still go
	// through notebook_write -> notebook_promote -> @librarian review, or pass a
	// human_request for a one-off direct write.
	// A task-scoped write skips the human-delegation gate: the broker created	the task and pinned its Definition.ArtifactPath, so authorization already	happened upstream and /wiki/write-task enforces the path match.
	path := strings.TrimSpace(args.ArticlePath)
	if path == "" {
		return toolError(fmt.Errorf("article_path is required")), nil, nil
	}
	taskID := strings.TrimSpace(args.TaskID)
	// attributedTaskID is set only on the human-delegation path, and only for
	// the ledger: /wiki/write stays authorizable by the human's message alone.
	attributedTaskID := ""
	if taskID == "" && !adminDirectWikiWriteBypassEnabled() && !strings.EqualFold(strings.TrimSpace(slug), team.LibrarianSlug) {
		// The authorization the task already granted is the one the bot least
		// often notices. Journey 18: the parent Issue declared no artifact_path,
		// /wiki/write-task refused, and the office spent two rounds asking the
		// human for "another message that says write to the wiki" — while its
		// own lane tasks had declared exactly these paths. Resolve it here so a
		// task-scoped write needs no argument and no human round trip; the
		// broker still enforces owner + declared path + open task.
		taskID = resolveTaskArtifactAuthorization(ctx, slug, path)
	}
	if taskID == "" && !adminDirectWikiWriteBypassEnabled() && !strings.EqualFold(strings.TrimSpace(slug), team.LibrarianSlug) {
		if err := verifyHumanWikiWriteDelegation(ctx, slug, args.HumanRequest); err != nil {
			return toolError(err), nil, nil
		}
		// Journey 25: the human authorized the write, the bot wrote it, and the
		// task that the human was actually accepting never learned about it —
		// its card kept naming the one file the task-scoped door had seen.
		// Attribution only; the human's message is still what authorized it.
		attributedTaskID = resolveSoleLiveTask(ctx, slug)
	}
	mode := strings.TrimSpace(args.Mode)
	if mode == "" {
		mode = "create"
	}
	switch mode {
	case "create", "replace", "append_section":
	default:
		return toolError(fmt.Errorf("mode must be one of create | replace | append_section; got %q", mode)), nil, nil
	}
	if strings.TrimSpace(args.Content) == "" {
		return toolError(fmt.Errorf("content is required")), nil, nil
	}
	var result struct {
		Path         string `json:"path"`
		CommitSHA    string `json:"commit_sha"`
		BytesWritten int    `json:"bytes_written"`
	}
	endpoint := "/wiki/write"
	body := map[string]any{
		"slug":           slug,
		"path":           path,
		"mode":           mode,
		"content":        args.Content,
		"commit_message": args.CommitMsg,
	}
	if taskID != "" {
		endpoint = "/wiki/write-task"
		body["task_id"] = taskID
	} else if attributedTaskID != "" {
		body["task_id"] = attributedTaskID
	}
	err = brokerPostJSON(ctx, endpoint, body, &result)
	if err != nil {
		return toolError(err), nil, nil
	}
	payload, _ := json.Marshal(map[string]any{
		"path":          result.Path,
		"commit_sha":    result.CommitSHA,
		"bytes_written": result.BytesWritten,
	})
	return textResult(string(payload)), nil, nil
}

// resolveSoleLiveTask returns the one non-terminal task this bot owns, or ""
// when there is none or more than one. Used for attribution, never for
// authorization: naming the wrong task would misreport an acceptance surface,
// so the ambiguous case stays unnamed.
func resolveSoleLiveTask(ctx context.Context, slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	var response struct {
		Tasks []struct {
			ID     string `json:"id"`
			Owner  string `json:"owner"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	query := "/tasks?all_channels=true&my_slug=" + url.QueryEscape(slug)
	if err := brokerGetJSON(ctx, query, &response); err != nil {
		return ""
	}
	found := ""
	for _, task := range response.Tasks {
		if !strings.EqualFold(strings.TrimSpace(task.Owner), slug) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(task.Status)) {
		case "done", "failed", "cancelled", "canceled", "archived":
			continue
		}
		if found != "" {
			return ""
		}
		found = task.ID
	}
	return found
}

// resolveTaskArtifactAuthorization returns the caller's open task whose
// definition declares this exact path, or "" when none does. Best-effort: a
// broker that cannot answer leaves the caller on the human-delegation path
// rather than turning a lookup failure into an authorization.
func resolveTaskArtifactAuthorization(ctx context.Context, slug, path string) string {
	slug = strings.TrimSpace(slug)
	path = strings.TrimSpace(path)
	if slug == "" || path == "" {
		return ""
	}
	var response struct {
		Tasks []struct {
			ID         string `json:"id"`
			Owner      string `json:"owner"`
			Status     string `json:"status"`
			Definition *struct {
				ArtifactPath string `json:"artifact_path"`
			} `json:"definition"`
		} `json:"tasks"`
	}
	query := "/tasks?all_channels=true&my_slug=" + url.QueryEscape(slug)
	if err := brokerGetJSON(ctx, query, &response); err != nil {
		return ""
	}
	for _, task := range response.Tasks {
		if !strings.EqualFold(strings.TrimSpace(task.Owner), slug) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(task.Status)) {
		case "done", "failed", "cancelled", "canceled", "archived":
			continue
		}
		if task.Definition == nil || strings.TrimSpace(task.Definition.ArtifactPath) == "" {
			continue
		}
		if team.SameWikiPath(task.Definition.ArtifactPath, path) {
			return strings.TrimSpace(task.ID)
		}
	}
	return ""
}

func verifyHumanWikiWriteDelegation(ctx context.Context, slug, humanRequestID string) error {
	humanRequestID = strings.TrimSpace(humanRequestID)
	if humanRequestID == "" {
		// Human-boundary copy (ten-out-of-ten E1b): bots relay tool errors
		// verbatim, and "the broker requires a direct human message ID" read
		// as raw jargon to a real operator (ICP-eval v3 [18:07]). Lead with
		// words safe to repeat to the human; keep the mechanics for the bot.
		return fmt.Errorf("this wiki update needs the human's direct go-ahead. Ask them in plain words (e.g. \"want me to update the wiki with this?\") — never mention broker internals or message IDs to them. When they reply asking for the write, retry with human_request set to that human message's id. For bot-authored knowledge, use notebook_write then notebook_promote for review instead")
	}

	channels := fetchAccessibleChannels(ctx, slug)
	if len(channels) == 0 {
		channels = []brokerChannelSummary{{Slug: resolveChannel("")}}
	}
	seen := map[string]bool{}
	for _, channel := range channels {
		channelSlug := strings.TrimSpace(channel.Slug)
		if channelSlug == "" || seen[channelSlug] {
			continue
		}
		seen[channelSlug] = true
		messages := fetchChannelMessages(ctx, channelSlug, slug, "all", 100)
		for _, msg := range messages {
			if strings.TrimSpace(msg.ID) != humanRequestID {
				continue
			}
			return validateHumanWikiWriteDelegation(msg)
		}
	}
	return fmt.Errorf("team_wiki_write human_request %q was not found in recent accessible human messages; pass the id of the human's recent message that asked for this wiki write (do not relay this error to the human — ask them plainly for the go-ahead instead)", humanRequestID)
}

func validateHumanWikiWriteDelegation(msg brokerMessage) error {
	if !isVerifiedHumanDelegationSender(msg.From) {
		return fmt.Errorf("team_wiki_write human_request %q is not a human-authored message", msg.ID)
	}
	timestamp := strings.TrimSpace(msg.Timestamp)
	if timestamp == "" {
		return fmt.Errorf("team_wiki_write human_request %q is missing a timestamp", msg.ID)
	}
	createdAt, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return fmt.Errorf("team_wiki_write human_request %q has an invalid timestamp", msg.ID)
	}
	if time.Since(createdAt) > humanWikiDelegationMaxAge {
		return fmt.Errorf("team_wiki_write human_request %q is expired; ask the human to restate the direct wiki request", msg.ID)
	}
	if !hasDirectWikiWriteIntent(msg.Title + "\n" + msg.Content) {
		return fmt.Errorf("team_wiki_write human_request %q does not explicitly ask for a direct wiki write", msg.ID)
	}
	return nil
}

func isVerifiedHumanDelegationSender(sender string) bool {
	sender = strings.ToLower(strings.TrimSpace(sender))
	return sender == "you" || sender == "human" || strings.HasPrefix(sender, "human:")
}

func hasDirectWikiWriteIntent(text string) bool {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return false
	}
	verbs := directWikiVerbRE.FindAllStringIndex(normalized, -1)
	targets := directWikiTargetRE.FindAllStringIndex(normalized, -1)
	if len(verbs) == 0 || len(targets) == 0 {
		return false
	}
	negations := directWikiNegationRE.FindAllStringIndex(normalized, -1)
	for _, verb := range verbs {
		for _, target := range targets {
			if wordsBetween(normalized, verb, target) > directWikiIntentMaxGapWords {
				continue
			}
			if directWikiIntentNegated(normalized, negations, verb, target) {
				continue
			}
			return true
		}
	}
	return false
}

func wordsBetween(text string, left, right []int) int {
	if left[0] > right[0] {
		left, right = right, left
	}
	if left[1] >= right[0] {
		return 0
	}
	return len(strings.Fields(text[left[1]:right[0]]))
}

func directWikiIntentNegated(text string, negations [][]int, verb, target []int) bool {
	for _, negation := range negations {
		if negatesMatch(text, negation, verb) || negatesMatch(text, negation, target) {
			return true
		}
	}
	return false
}

func negatesMatch(text string, negation, match []int) bool {
	return negation[0] < match[0] && wordsBetween(text, negation, match) <= directWikiIntentMaxGapWords
}

// handleTeamWikiRead returns the raw article bytes.
func handleTeamWikiRead(ctx context.Context, _ *mcp.CallToolRequest, args TeamWikiReadArgs) (*mcp.CallToolResult, any, error) {
	path := strings.TrimSpace(args.ArticlePath)
	if path == "" {
		return toolError(fmt.Errorf("article_path is required")), nil, nil
	}
	brokerPath := "/wiki/read?path=" + url.QueryEscape(path)
	// Pass bot slug so the broker can record this read in the attention log.
	// Fall back to the legacy-prefixed spelling for environments still setting it.
	if slug := strings.TrimSpace(os.Getenv("WUPHF_AGENT_SLUG")); slug != "" {
		brokerPath += "&reader=" + url.QueryEscape(slug)
	} else if slug := strings.TrimSpace(os.Getenv("NEX_AGENT_SLUG")); slug != "" {
		brokerPath += "&reader=" + url.QueryEscape(slug)
	}
	bytes, err := brokerGetRaw(ctx, brokerPath)
	if err != nil {
		return toolError(err), nil, nil
	}
	return textResult(string(bytes)), nil, nil
}

// handleTeamWikiSearch runs a literal substring search across the team
// wiki AND the calling bot's own notebook shelf (B4: one retrieval call
// spans wiki + private notes). The reader identity comes from the trusted
// launcher-set WUPHF_AGENT_SLUG env — never from a model-supplied arg — so
// a bot can only widen the search into its OWN notebooks.
func handleTeamWikiSearch(ctx context.Context, _ *mcp.CallToolRequest, args TeamWikiSearchArgs) (*mcp.CallToolResult, any, error) {
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		// `query` is the name the sibling retrieval tools use, so models reach
		// for it here too. Accepting it costs nothing; rejecting it produced a
		// silence a bot reported as "the deliverable does not exist" (Journey
		// 11, JOURN-26), which no human could trace back to an argument name.
		pattern = strings.TrimSpace(args.Query)
	}
	if pattern == "" {
		return toolError(fmt.Errorf(`pattern is required: pass {"pattern": "<literal substring>"} (the name "query" is accepted too)`)), nil, nil
	}
	slug := strings.TrimSpace(trustedEnvBotSlug())
	path := "/wiki/search?pattern=" + url.QueryEscape(pattern)
	if slug != "" {
		path += "&reader=" + url.QueryEscape(slug)
	}
	var result struct {
		Hits []map[string]any `json:"hits"`
	}
	if err := brokerGetJSON(ctx, path, &result); err != nil {
		return toolError(err), nil, nil
	}
	// Zero hits is a statement about one literal substring, not about the wiki.
	// Returning `[]` invited the reader to conclude "this file does not exist"
	// without ever opening it, so say what was searched and what to do next.
	if len(result.Hits) == 0 {
		reader := slug
		if reader == "" {
			reader = "the shared wiki"
		}
		return textResult(fmt.Sprintf("No hits for the literal substring %q in %s. This is a substring match over article bodies, not a list of what exists: try a shorter token (e.g. a single name), or open the path you expect with team_wiki_read.", pattern, reader)), nil, nil
	}
	payload, _ := json.Marshal(result.Hits)
	return textResult(string(payload)), nil, nil
}

// handleTeamWikiList returns the auto-regenerated catalog at index/all.md.
func handleTeamWikiList(ctx context.Context, _ *mcp.CallToolRequest, _ TeamWikiListArgs) (*mcp.CallToolResult, any, error) {
	bytes, err := brokerGetRaw(ctx, "/wiki/list")
	if err != nil {
		return toolError(err), nil, nil
	}
	return textResult(string(bytes)), nil, nil
}

// handleTeamWikiLookup answers a natural-language question with a cited
// response assembled from the team wiki. The broker's /wiki/lookup endpoint
// runs the full QueryHandler pipeline: classify → search → prompt → parse.
// Returns the raw QueryAnswer JSON so the calling bot can render citations.
func handleTeamWikiLookup(ctx context.Context, _ *mcp.CallToolRequest, args TeamWikiLookupArgs) (*mcp.CallToolResult, any, error) {
	q := strings.TrimSpace(args.Query)
	if q == "" {
		return toolError(fmt.Errorf("query is required")), nil, nil
	}
	path := "/wiki/lookup?q=" + url.QueryEscape(q)
	if args.TopK > 0 {
		path += fmt.Sprintf("&top_k=%d", args.TopK)
	}
	bytes, err := brokerGetRaw(ctx, path)
	if err != nil {
		return toolError(err), nil, nil
	}
	return textResult(string(bytes)), nil, nil
}

// ── Lint tools ────────────────────────────────────────────────────────────────

// RunLintArgs is intentionally empty — run_lint takes no input parameters.
type RunLintArgs struct{}

// ResolveContradictionArgs is the contract for resolve_contradiction.
type ResolveContradictionArgs struct {
	ReportDate string `json:"report_date" jsonschema:"YYYY-MM-DD date of the lint report to resolve from"`
	FindingIdx int    `json:"finding_idx" jsonschema:"0-based index into the findings array returned by run_lint"`
	Winner     string `json:"winner"      jsonschema:"A | B | Both — which fact wins; Both acknowledges both as valid"`
}

// handleRunLint calls POST /wiki/lint/run on the broker and returns the
// full LintReport JSON.
func handleRunLint(ctx context.Context, _ *mcp.CallToolRequest, _ RunLintArgs) (*mcp.CallToolResult, any, error) {
	var report any
	if err := brokerPostJSON(ctx, "/wiki/lint/run", nil, &report); err != nil {
		return toolError(err), nil, nil
	}
	payload, _ := json.Marshal(report)
	return textResult(string(payload)), nil, nil
}

// handleResolveContradiction calls POST /wiki/lint/resolve on the broker.
func handleResolveContradiction(ctx context.Context, _ *mcp.CallToolRequest, args ResolveContradictionArgs) (*mcp.CallToolResult, any, error) {
	reportDate := strings.TrimSpace(args.ReportDate)
	if reportDate == "" {
		return toolError(fmt.Errorf("report_date is required")), nil, nil
	}
	if !isLintReportDate(reportDate) {
		return toolError(fmt.Errorf("report_date must be YYYY-MM-DD; got %q", reportDate)), nil, nil
	}
	if args.FindingIdx < 0 {
		return toolError(fmt.Errorf("finding_idx must be non-negative; got %d", args.FindingIdx)), nil, nil
	}
	winner := strings.TrimSpace(args.Winner)
	if winner != "A" && winner != "B" && winner != "Both" {
		return toolError(fmt.Errorf("winner must be A, B, or Both; got %q", winner)), nil, nil
	}

	var resp map[string]string
	body := map[string]any{
		"report_date": reportDate,
		"finding_idx": args.FindingIdx,
		"winner":      winner,
	}
	if err := brokerPostJSON(ctx, "/wiki/lint/resolve", body, &resp); err != nil {
		return toolError(err), nil, nil
	}
	msg := resp["message"]
	if msg == "" {
		msg = fmt.Sprintf("Resolved finding %d from report %s as winner=%s", args.FindingIdx, reportDate, winner)
	}
	return textResult(msg), nil, nil
}

func isLintReportDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
