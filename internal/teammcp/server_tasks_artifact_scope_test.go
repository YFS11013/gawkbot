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
