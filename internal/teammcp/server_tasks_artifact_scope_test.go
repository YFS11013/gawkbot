package teammcp

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

// TestHandleTeamTaskDefineForwardsArtifactScope pins the Journey 1 fix: the
// CEO can declare the one wiki path a task may publish to, and that path
// lands in Definition.ArtifactPath — the field /wiki/write-task authorizes.
// Without it every wiki write bounced back to the human for a go-ahead.
func TestHandleTeamTaskDefineForwardsArtifactScope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()

	b := newTestBroker(t)
	if err := b.StartOnPort(0); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	defer b.Stop()
	t.Setenv("WUPHF_TEAM_BROKER_URL", "http://"+b.Addr())
	t.Setenv("WUPHF_BROKER_TOKEN", b.Token())
	ensureBrokerMembers(t, ctx, "cos", "ae")

	created, _, err := handleTeamTask(ctx, nil, TeamTaskArgs{
		Action:  "create",
		Channel: "general",
		Title:   "Publish the Acme renewal brief",
		Owner:   "ae",
		MySlug:  "cos",
	})
	if err != nil {
		t.Fatalf("handleTeamTask create: %v", err)
	}
	m := regexp.MustCompile(`Task (\S+) in #`).FindStringSubmatch(textFromResult(t, created))
	if m == nil {
		t.Fatalf("no task id in %q", textFromResult(t, created))
	}
	taskID := m[1]

	if _, _, err := handleTeamTask(ctx, nil, TeamTaskArgs{
		Action:        "define",
		ID:            taskID,
		MySlug:        "cos",
		Goal:          "Publish the Acme renewal brief to the wiki.",
		ArtifactScope: "team/accounts/acme-corp.md",
	}); err != nil {
		t.Fatalf("handleTeamTask define: %v", err)
	}

	var got struct {
		Task struct {
			Definition struct {
				ArtifactPath string `json:"artifact_path"`
			} `json:"definition"`
		} `json:"task"`
	}
	if err := brokerGetJSON(ctx, "/tasks/"+taskID, &got); err != nil {
		t.Fatalf("GET task: %v", err)
	}
	if got.Task.Definition.ArtifactPath != "team/accounts/acme-corp.md" {
		t.Fatalf("definition.artifact_path = %q, want team/accounts/acme-corp.md", got.Task.Definition.ArtifactPath)
	}
}

// TestHandleTeamTaskDefineRejectsScopeOutsideTeamRoots keeps the write grant
// narrow: a bot-defined task cannot self-authorize a page outside the
// published roots, so defining the task cannot become a way to rewrite any
// article.
func TestHandleTeamTaskDefineRejectsScopeOutsideTeamRoots(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()

	b := newTestBroker(t)
	if err := b.StartOnPort(0); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	defer b.Stop()
	t.Setenv("WUPHF_TEAM_BROKER_URL", "http://"+b.Addr())
	t.Setenv("WUPHF_BROKER_TOKEN", b.Token())
	ensureBrokerMembers(t, ctx, "cos", "ae")

	created, _, err := handleTeamTask(ctx, nil, TeamTaskArgs{
		Action:  "create",
		Channel: "general",
		Title:   "Rewrite the org chart",
		MySlug:  "cos",
	})
	if err != nil {
		t.Fatalf("handleTeamTask create: %v", err)
	}
	m := regexp.MustCompile(`Task (\S+) in #`).FindStringSubmatch(textFromResult(t, created))
	if m == nil {
		t.Fatalf("no task id in %q", textFromResult(t, created))
	}

	res, _, err := handleTeamTask(ctx, nil, TeamTaskArgs{
		Action:        "define",
		ID:            m[1],
		MySlug:        "cos",
		Goal:          "Rewrite an article the task was not granted.",
		ArtifactScope: "team/people/cos.md",
	})
	if err != nil {
		return // surfaced as a transport error: also a rejection
	}
	if res == nil || !res.IsError {
		t.Fatalf("define accepted a scope outside the allowed roots: %+v", textFromResult(t, res))
	}
	if !strings.Contains(textFromResult(t, res), "artifact_path") {
		t.Fatalf("rejection does not name artifact_path: %q", textFromResult(t, res))
	}
}

// TestHandleTeamTaskCreateForwardsScopeForSubIssue pins the Journey 8 half of
// the write-scope fix: the planner must be able to declare a lane's scope in
// the SAME create call. The broker refuses a lead-authored sub-issue that
// declares neither a path nor "ships nothing", so a create that silently drops
// artifact_scope would leave the planner with no way to satisfy that rule.
func TestHandleTeamTaskCreateForwardsScopeForSubIssue(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()

	b := newTestBroker(t)
	if err := b.StartOnPort(0); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	defer b.Stop()
	t.Setenv("WUPHF_TEAM_BROKER_URL", "http://"+b.Addr())
	t.Setenv("WUPHF_BROKER_TOKEN", b.Token())
	ensureBrokerMembers(t, ctx, "cos", "ae")

	// Parent with no owner: it never enters Planning, so the lane below is
	// judged on the write-scope rule rather than the plan gate.
	parent, _, err := handleTeamTask(ctx, nil, TeamTaskArgs{
		Action:  "create",
		Channel: "general",
		Title:   "Land the Acme renewal",
		MySlug:  "cos",
	})
	if err != nil {
		t.Fatalf("handleTeamTask parent create: %v", err)
	}
	pm := regexp.MustCompile(`Task (\S+) in #`).FindStringSubmatch(textFromResult(t, parent))
	if pm == nil {
		t.Fatalf("no parent id in %q", textFromResult(t, parent))
	}
	// Approve the plan directly: the human actor cannot be forged through the
	// MCP tool (resolveSlug rejects reserved slugs), and this test is about the
	// create payload, not the approval path.
	if err := brokerPostJSON(ctx, "/tasks", map[string]any{
		"action": "approve", "id": pm[1], "created_by": "human", "channel": "general",
	}, nil); err != nil {
		t.Fatalf("approve parent plan: %v", err)
	}

	lane, _, err := handleTeamTask(ctx, nil, TeamTaskArgs{
		Action:        "create",
		Channel:       "general",
		Title:         "Write the Acme renewal one-pager",
		Owner:         "ae",
		MySlug:        "cos",
		ParentIssueID: pm[1],
		ArtifactScope: "team/accounts/acme-corp.md",
	})
	if err != nil {
		t.Fatalf("handleTeamTask lane create with artifact_scope: %v", err)
	}
	lm := regexp.MustCompile(`Task (\S+) in #`).FindStringSubmatch(textFromResult(t, lane))
	if lm == nil {
		t.Fatalf("no lane id in %q", textFromResult(t, lane))
	}

	var got struct {
		Task struct {
			Definition struct {
				Goal         string `json:"goal"`
				ArtifactPath string `json:"artifact_path"`
			} `json:"definition"`
		} `json:"task"`
	}
	if err := brokerGetJSON(ctx, "/tasks/"+lm[1], &got); err != nil {
		t.Fatalf("GET lane: %v", err)
	}
	if got.Task.Definition.ArtifactPath != "team/accounts/acme-corp.md" {
		t.Fatalf("lane definition.artifact_path = %q, want team/accounts/acme-corp.md", got.Task.Definition.ArtifactPath)
	}
	if strings.TrimSpace(got.Task.Definition.Goal) == "" {
		t.Fatal("create with a scope must carry a goal (defaulted from the title)")
	}
}
