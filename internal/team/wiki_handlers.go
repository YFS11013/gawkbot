package team

// wiki_handlers.go owns the broker HTTP endpoints for the team wiki. These
// handlers stay in the same package as WikiWorker but live outside
// wiki_worker.go so the worker file remains under the repo file-size budget.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// handleWikiWrite is the broker HTTP endpoint the MCP subprocess posts to
// when a bot calls team_wiki_write. POST /wiki/write with
// {slug, path, content, mode, commit_message}. Responses: 200 {path,
// commit_sha, bytes_written}; 429 saturated; 500 generic; 503 no worker.
func (b *Broker) handleWikiWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		http.Error(w, `{"error":"wiki backend is not active"}`, http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Slug          string `json:"slug"`
		Path          string `json:"path"`
		Content       string `json:"content"`
		Mode          string `json:"mode"`
		CommitMessage string `json:"commit_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	isNew := wikiArticleIsNew(worker.Repo(), body.Path)
	sha, n, err := worker.Enqueue(r.Context(), body.Slug, body.Path, body.Content, body.Mode, body.CommitMessage)
	if err != nil {
		if errors.Is(err, ErrQueueSaturated) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if isNew {
		b.emitNewWikiArticleCard(body.Path, body.Content, r.Header.Get(botRateLimitHeader))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":          body.Path,
		"commit_sha":    sha,
		"bytes_written": n,
	})
}

// handleWikiWriteTask is the task-scoped write: a bot may write the one
// artifact path its own open task declares in Definition.ArtifactPath without
// a per-write human message. This is what lets a review task produce its
// findings file — the broker created the task and pinned the path, so the
// authorization already happened upstream and the path is not free-form.
//
// POST /wiki/write-task with {slug, task_id, path, content, mode,
// commit_message}. Responses: 200 {path, commit_sha, bytes_written}; 403 the
// task is not yours or the path is not its declared artifact; 404 unknown
// task; 409 the task is already closed; 429/500 as /wiki/write.
func (b *Broker) handleWikiWriteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		http.Error(w, `{"error":"wiki backend is not active"}`, http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Slug          string `json:"slug"`
		TaskID        string `json:"task_id"`
		Path          string `json:"path"`
		Content       string `json:"content"`
		Mode          string `json:"mode"`
		CommitMessage string `json:"commit_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	slug := strings.TrimSpace(body.Slug)
	taskID := strings.TrimSpace(body.TaskID)
	path := strings.TrimSpace(body.Path)
	if slug == "" || taskID == "" || path == "" {
		http.Error(w, `{"error":"slug, task_id and path are required"}`, http.StatusBadRequest)
		return
	}
	if err := b.authorizeTaskArtifactWrite(slug, taskID, path); err != nil {
		status := http.StatusForbidden
		switch {
		case errors.Is(err, errTaskArtifactUnknownTask):
			status = http.StatusNotFound
		case errors.Is(err, errTaskArtifactTaskClosed):
			status = http.StatusConflict
		}
		log.Printf("wiki write-task refused: task=%s slug=%s path=%s: %v", taskID, slug, path, err)
		if errors.Is(err, errTaskArtifactUndeclared) {
			// Say it where the two people who can fix it are looking. One nudge
			// per task: the bot retries, and a retry is not a second incident.
			b.nudgeUndeclaredArtifactPath(taskID, slug, path)
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	isNew := wikiArticleIsNew(worker.Repo(), path)
	sha, n, err := worker.EnqueueTaskArtifact(r.Context(), slug, path, body.Content, body.Mode, body.CommitMessage)
	if err != nil {
		if errors.Is(err, ErrQueueSaturated) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if isNew {
		b.emitNewWikiArticleCard(path, body.Content, r.Header.Get(botRateLimitHeader))
	}
	// append_section adds to a page on purpose (that is the sanctioned way to
	// share one file); create/replace put the content back and drop whatever
	// was there.
	if !isNew && !strings.EqualFold(strings.TrimSpace(body.Mode), "append_section") {
		b.recordForeignArtifactOverwrite(taskID, slug, path)
	}
	log.Printf("wiki write-task: task=%s slug=%s path=%s bytes=%d", taskID, slug, path, n)
	writeJSON(w, http.StatusOK, map[string]any{
		"path":          path,
		"commit_sha":    sha,
		"bytes_written": n,
	})
}

var (
	errTaskArtifactUnknownTask = errors.New("task not found")
	errTaskArtifactTaskClosed  = errors.New("task is already closed")
	// errTaskArtifactUndeclared is the "nobody pinned a path" case: the task is
	// open and owned by the caller, but its definition carries no
	// artifact_path, so there is nothing to authorize against. Journey 3 and
	// Journey 4 both hit it through sub-issues the planner created without a
	// definition; the owner then wrote into a notebook and the human only saw
	// the downgrade at review.
	errTaskArtifactUndeclared = errors.New("artifact path not declared")
)

// authorizeTaskArtifactWrite is the whole gate: the task exists, the caller
// owns it, it is still open, and the path is exactly the artifact path its
// definition declared. Anything else is refused — the point of the gate is
// that a bot cannot name a path the broker did not already pin on its task.
func (b *Broker) authorizeTaskArtifactWrite(slug, taskID, path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	task := b.taskByIDLocked(taskID)
	if task == nil {
		return fmt.Errorf("%w: %s", errTaskArtifactUnknownTask, taskID)
	}
	if owner := strings.TrimSpace(task.Owner); !strings.EqualFold(owner, slug) {
		return fmt.Errorf("task %s is owned by %s, not %s", taskID, owner, slug)
	}
	switch strings.ToLower(strings.TrimSpace(task.status)) {
	case "done", "failed", "cancelled", "canceled", "archived":
		return fmt.Errorf("%w: %s is %s", errTaskArtifactTaskClosed, taskID, task.status)
	}
	declared := ""
	if task.Definition != nil {
		declared = strings.TrimSpace(task.Definition.ArtifactPath)
	}
	if declared == "" {
		creator := strings.TrimSpace(task.CreatedBy)
		if creator == "" {
			creator = "whoever created it"
		} else {
			creator = "@" + creator
		}
		// Name the fix and its owner. "pass human_request instead" told the bot
		// what not to do and left the human out of the loop entirely.
		return fmt.Errorf("%w: %s declares no artifact_path, so no wiki path is authorized for it. %s must set definition.artifact_path on the task (or route this deliverable through a human request); retrying this call cannot succeed",
			errTaskArtifactUndeclared, taskID, creator)
	}
	if !SameWikiPath(declared, path) {
		return fmt.Errorf("task %s declares artifact_path %q, not %q", taskID, declared, path)
	}
	return nil
}

// SameWikiPath compares two wiki paths the way a filesystem would here:
// separators normalized, no leading "./" or "/", case-insensitive. Exported
// because the MCP server resolves "does one of my tasks declare this path"
// against the same rule the broker enforces when it authorizes the write;
// two copies would drift into an authorization mismatch.
func SameWikiPath(a, b string) bool {
	canon := func(s string) string {
		s = strings.ReplaceAll(strings.TrimSpace(s), "\\", "/")
		s = strings.TrimPrefix(s, "./")
		s = strings.TrimPrefix(s, "/")
		return strings.ToLower(path.Clean(s))
	}
	return canon(a) == canon(b)
}

// sanitizeReader validates a ?reader= query param before writing it to
// reads.jsonl. Returns the original value when it is safe, or "" to suppress
// tracking when it is not. Allowed: lowercase letters, digits, hyphens,
// underscores, max 64 chars. The special value ReaderHuman ("web") passes.
func sanitizeReader(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 64 {
		return ""
	}
	for _, ch := range raw {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return ""
		}
	}
	return raw
}

// handleWikiRead returns raw article bytes.
//
//	GET /wiki/read?path=team/people/nazz.md
func (b *Broker) handleWikiRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		// No markdown worker → gbrain-backed deployment. Route the read through
		// gbrain (or degrade gracefully when gbrain is unreachable).
		b.serveWikiReadFromGBrain(w, r)
		return
	}
	relPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if err := validateArticlePath(relPath); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	bytes, err := readArticle(worker.Repo(), relPath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	// Track bot reads. The ?reader= param is set by the MCP layer using
	// WUPHF_AGENT_SLUG. Human reads go through /wiki/article, not here.
	// Return 400 if the caller passes the reserved human reader ("web"):
	// a bot slug named "web" would silently inflate human_read_count.
	if raw := r.URL.Query().Get("reader"); raw == ReaderHuman {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `reader "web" is reserved for human browser access`})
		return
	} else if reader := sanitizeReader(raw); reader != "" {
		b.WikiReadLog().Append(relPath, reader)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(bytes)
}

// handleWikiSearch returns literal-substring matches across team/. When a
// ?reader=<bot-slug> is supplied (set by the MCP layer from the trusted
// WUPHF_AGENT_SLUG env), the same call ALSO searches that bot's OWN
// notebook shelf, so a single retrieval spans wiki + private notes (B4).
// Permissioned boundary: only the reader's own notebooks are merged -
// cross-bot notebook access stays on the explicit notebook_read path.
//
//	GET /wiki/search?pattern=launch[&reader=eng]
func (b *Broker) handleWikiSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		// No markdown worker → gbrain-backed deployment.
		b.serveWikiSearchFromGBrain(w, r)
		return
	}
	pattern := strings.TrimSpace(r.URL.Query().Get("pattern"))
	if pattern == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pattern is required"})
		return
	}
	hits, err := searchArticles(worker.Repo(), pattern)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

// handleWikiList returns the contents of index/all.md.
//
//	GET /wiki/list
func (b *Broker) handleWikiList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		// No markdown worker → gbrain-backed deployment.
		b.serveWikiListFromGBrain(w, r)
		return
	}
	bytes, err := readIndexAll(worker.Repo())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(bytes)
}

// handleWikiCatalog returns the full catalog as structured JSON for the UI.
//
//	GET /wiki/catalog
//
// Response shape matches web/src/api/wiki.ts { articles: WikiCatalogEntry[] }.
// Distinct from /wiki/list (which returns raw markdown from index/all.md) -
// bots read the markdown index, the UI reads this JSON.
func (b *Broker) handleWikiCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		// No markdown worker → gbrain-backed deployment.
		b.serveWikiCatalogFromGBrain(w, r)
		return
	}
	sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	entries, err := worker.Repo().BuildCatalog(r.Context(), sortParam, b.WikiReadLog(), includeArchived)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []CatalogEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"articles": entries})
}

// handleWikiArchiveSweep runs one WikiArchiver.Sweep synchronously and
// returns the SweepResult as JSON. POST is required because this endpoint
// mutates state (archives articles, commits to git).
//
//	POST /wiki/archive/sweep
func (b *Broker) handleWikiArchiveSweep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		http.Error(w, `{"error":"wiki backend is not active"}`, http.StatusServiceUnavailable)
		return
	}
	b.archiveSweepMu.Lock()
	defer b.archiveSweepMu.Unlock()
	// Cap the manual sweep with the same 10-minute timeout as the scheduled
	// path so a hung git process cannot hold archiveSweepMu indefinitely
	// while the caller holds the connection open.
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	result, err := worker.EnqueueArchiveSweep(ctx, b.WikiReadLog(), 0)
	if err != nil && isHardArchiveSweepError(err) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// startArchiveSweepLoop runs WikiArchiver.Sweep on the wiki-archive-sweep
// cron schedule (default: daily, MinFloor: 60 min).
func (b *Broker) startArchiveSweepLoop(ctx context.Context) {
	const defaultInterval = 1440 * time.Minute
	go func() {
		for {
			enabled, interval := b.SchedulerJobControl("wiki-archive-sweep", defaultInterval)
			now := time.Now().UTC()
			b.updateSchedulerHeartbeat("wiki-archive-sweep", "Wiki archive sweep",
				int(interval/time.Minute), now.Add(interval),
				disabledOrSleeping(enabled), "")
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
			if !enabled {
				continue
			}
			runStatus := b.runArchiveSweepTick()
			b.updateSchedulerHeartbeat("wiki-archive-sweep", "Wiki archive sweep",
				int(interval/time.Minute), time.Now().UTC().Add(interval),
				"sleeping", runStatus)
		}
	}()
}

func (b *Broker) runArchiveSweepTick() string {
	worker := b.WikiWorker()
	if worker == nil {
		return "inactive"
	}
	b.archiveSweepMu.Lock()
	defer b.archiveSweepMu.Unlock()
	// 10 minutes is generous for 500 eligible articles at ~1s/commit.
	// Without a timeout a hung git process holds archiveSweepMu forever,
	// blocking the POST /wiki/archive/sweep handler with no escape.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	result, err := worker.EnqueueArchiveSweep(ctx, b.WikiReadLog(), 0)
	if err != nil && isHardArchiveSweepError(err) {
		log.Printf("wiki archive: sweep error: %v", err)
		return "error"
	}
	if result.Archived > 0 || result.Errors > 0 {
		log.Printf("wiki archive: sweep complete - archived=%d skipped=%d errors=%d",
			result.Archived, result.Skipped, result.Errors)
	}
	if result.Errors > 0 {
		return "error"
	}
	return "ok"
}

// handleWikiAudit returns the cross-article commit log for audit / compliance.
// Unlike /wiki/history/<path> which scopes to one article, this feed covers
// the whole wiki and includes bootstrap + recovery + system commits so the
// lineage is complete.
//
//	GET /wiki/audit
//	GET /wiki/audit?limit=50
//	GET /wiki/audit?since=2026-04-01T00:00:00Z
//
// Response:
//
//	{
//	  "entries": [
//	    {
//	      "sha": "...", "author_slug": "...", "timestamp": "...",
//	      "message": "...", "paths": ["team/..."]
//	    },
//	    ...
//	  ],
//	  "total": N
//	}
func (b *Broker) handleWikiAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	worker := b.WikiWorker()
	if worker == nil {
		http.Error(w, `{"error":"wiki backend is not active"}`, http.StatusServiceUnavailable)
		return
	}
	// Parse limit (optional, 0 = all). Default cap keeps a runaway caller
	// from dragging in 100k commits; explicit `limit=0` opts out of the cap.
	const defaultLimit = 500
	limit := defaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a non-negative integer"})
			return
		}
		limit = v
	}
	var since time.Time
	if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "since must be RFC3339 (e.g. 2026-04-01T00:00:00Z)"})
			return
		}
		since = t
	}
	entries, err := worker.Repo().AuditLog(r.Context(), since, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// Re-shape to snake_case for the JSON API. Same convention as
	// /wiki/catalog and /wiki/article. `paths` never serialised as null:
	// absent paths (rare, but possible for a signed-only commit) get an
	// empty array so consumers don't have to null-guard.
	type wireEntry struct {
		SHA        string   `json:"sha"`
		AuthorSlug string   `json:"author_slug"`
		Timestamp  string   `json:"timestamp"`
		Message    string   `json:"message"`
		Paths      []string `json:"paths"`
	}
	wire := make([]wireEntry, 0, len(entries))
	for _, e := range entries {
		paths := e.Paths
		if paths == nil {
			paths = []string{}
		}
		wire = append(wire, wireEntry{
			SHA:        e.SHA,
			AuthorSlug: e.Author,
			Timestamp:  e.Timestamp.UTC().Format(time.RFC3339),
			Message:    e.Message,
			Paths:      paths,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": wire,
		"total":   len(wire),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// recordForeignArtifactOverwrite notes that a task replaced a wiki page another
// task already delivered. It does not refuse the write: a shared page (a ledger,
// a running report) is a legitimate shape, and `mode=append_section` is the
// sanctioned way to add to one — but `mode=create` on somebody else's deliverable
// silently replaced 33 bytes of GUARD-12's work with 1642 bytes of GUARD-4's
// during the guard exercise (core-loop-runs/2026-11-10-guards), and nothing
// recorded it. An audit row is the least this can cost: blocking would break
// routines that rewrite the same path every run.
func (b *Broker) recordForeignArtifactOverwrite(taskID, slug, path string) {
	if b == nil {
		return
	}
	taskID = strings.TrimSpace(taskID)
	path = strings.TrimSpace(path)
	if taskID == "" || path == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	ownerID := ""
	for i := range b.tasks {
		other := &b.tasks[i]
		if other.ID == taskID || !isDeliveredTeamTaskStatus(other.status) {
			continue
		}
		if artifact := strings.TrimSpace(other.Artifact); artifact != "" && SameWikiPath(artifact, path) {
			ownerID = other.ID
			break
		}
	}
	if ownerID == "" {
		return
	}
	channel := "general"
	if task := b.taskByIDLocked(taskID); task != nil {
		if ch := normalizeChannelSlug(task.Channel); ch != "" {
			channel = ch
		}
	}
	b.appendActionLocked("artifact_overwritten", "office", channel, slug,
		truncateSummary(fmt.Sprintf("%s overwrote %s (delivered by %s)", taskID, path, ownerID), 140), taskID)
	if err := b.saveLocked(); err != nil {
		log.Printf("broker: artifact overwrite for %s recorded in memory but not persisted: %v", taskID, err)
	}
}

// artifactPathMissingKind marks both the audit row and the chat line a
// refused write leaves behind. One kind keeps the dedupe query and the
// message a reader sees pointing at the same event.
const artifactPathMissingKind = "artifact_path_missing"

// nudgeUndeclaredArtifactPath posts the refusal where it can be acted on: the
// task's channel, addressing the creator who must pin the path and the owner
// who has to work around it meanwhile.
//
// Deduped by the audit row — a bot that retries the same write every turn must
// not produce a chat storm, and the row survives restarts (the in-memory
// message list does not). The log keeps 150 rows, so a busy office can lose
// the marker and nudge twice for one task; a second line is cheaper than a new
// wire field on every task.
func (b *Broker) nudgeUndeclaredArtifactPath(taskID, slug, path string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	task := b.taskByIDLocked(taskID)
	if task == nil {
		return
	}
	for _, a := range b.actions {
		if a.Kind == artifactPathMissingKind && a.RelatedID == taskID {
			return
		}
	}
	creator := strings.TrimSpace(task.CreatedBy)
	owner := strings.TrimSpace(task.Owner)
	channel := normalizeChannelSlug(task.Channel)
	if channel == "" {
		channel = "general"
	}
	who := "this task"
	if creator != "" {
		who = "@" + creator
	}
	writer := "a bot"
	if strings.TrimSpace(slug) != "" {
		writer = "@" + strings.TrimSpace(slug)
	}
	content := fmt.Sprintf("%s: %s has no definition.artifact_path, so %s's write to %s was refused. Pin the path on the task (or have %s deliver through a human request) — until then the lane can only leave notes in a notebook.",
		who, taskID, writer, path, writer)
	if owner != "" {
		content += fmt.Sprintf(" Owner: @%s.", owner)
	}
	b.appendActionLocked(artifactPathMissingKind, "office", channel, "system",
		truncateSummary(fmt.Sprintf("artifact write refused on %s: %s", taskID, path), 140), taskID)
	b.counter++
	b.appendMessageLocked(channelMessage{
		ID:        fmt.Sprintf("msg-%d", b.counter),
		From:      "system",
		Channel:   channel,
		Kind:      artifactPathMissingKind,
		Content:   content,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
