package action

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nex-crm/wuphf/internal/config"
)

func writeFakeOne(t *testing.T) string {
	t.Helper()
	return fakeOneCLI(t, "default")
}

func TestOneCLIHappyPath(t *testing.T) {
	oneBin := writeFakeOne(t)
	client := &OneCLI{Bin: oneBin, WorkDir: t.TempDir(), Env: []string{"ONE_SECRET=test-secret"}}
	ctx := context.Background()

	connections, err := client.ListConnections(ctx, ListConnectionsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(connections.Connections); got != 1 {
		t.Fatalf("expected 1 connection, got %d", got)
	}

	search, err := client.SearchActions(ctx, "gmail", "send email", "execute")
	if err != nil {
		t.Fatal(err)
	}
	if got := search.Actions[0].ActionID; got != "act-send" {
		t.Fatalf("unexpected action id %q", got)
	}

	knowledge, err := client.ActionKnowledge(ctx, "gmail", "act-send")
	if err != nil {
		t.Fatal(err)
	}
	if knowledge.Method != "POST" {
		t.Fatalf("unexpected method %q", knowledge.Method)
	}

	executed, err := client.ExecuteAction(ctx, ExecuteRequest{
		Platform:      "gmail",
		ActionID:      "act-send",
		ConnectionKey: "live::gmail::default::abc123",
		Data: map[string]any{
			"to": "a@example.com",
		},
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !executed.DryRun || executed.Request.Method != "POST" {
		t.Fatalf("unexpected execute result %+v", executed)
	}

	created, err := client.CreateWorkflow(ctx, WorkflowCreateRequest{
		Key:        "welcome-flow",
		Definition: []byte(`{"key":"welcome-flow","name":"Welcome","version":"1","inputs":{},"steps":[]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Created {
		t.Fatalf("expected created workflow, got %+v", created)
	}

	workflow, err := client.ExecuteWorkflow(ctx, WorkflowExecuteRequest{KeyOrPath: "welcome-flow"})
	if err != nil {
		t.Fatal(err)
	}
	if workflow.RunID != "run-1" || workflow.Status != "success" {
		t.Fatalf("unexpected workflow result %+v", workflow)
	}

	eventTypes, err := client.RelayEventTypes(ctx, "gmail")
	if err != nil {
		t.Fatal(err)
	}
	if len(eventTypes.EventTypes) != 1 {
		t.Fatalf("unexpected event types %+v", eventTypes)
	}

	relay, err := client.CreateRelay(ctx, RelayCreateRequest{
		ConnectionKey: "live::gmail::default::abc123",
		Description:   "mail relay",
		EventFilters:  []string{"message.received"},
		CreateWebhook: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if relay.ID != "relay-1" {
		t.Fatalf("unexpected relay %+v", relay)
	}

	relay, err = client.ActivateRelay(ctx, RelayActivateRequest{
		ID:      "relay-1",
		Actions: []byte(`[{"type":"passthrough"}]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !relay.Active {
		t.Fatalf("expected active relay, got %+v", relay)
	}

	events, err := client.ListRelayEvents(ctx, RelayEventsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(events.Events); got != 1 {
		t.Fatalf("expected 1 relay event, got %d", got)
	}

	detail, err := client.GetRelayEvent(ctx, "evt-1")
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != "evt-1" {
		t.Fatalf("unexpected relay detail %+v", detail)
	}
}

func TestNewOneCLIFromEnvUsesConfiguredIdentity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	if err := config.Save(config.Config{
		OneAPIKey: "one-secret",
		Email:     "cos@example.com",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	client := NewOneCLIFromEnv()
	got := strings.Join(client.Env, " ")
	if !strings.Contains(got, "ONE_SECRET=one-secret") {
		t.Fatalf("expected ONE_SECRET env, got %q", got)
	}
	if !strings.Contains(got, "ONE_IDENTITY=cos@example.com") {
		t.Fatalf("expected ONE_IDENTITY env, got %q", got)
	}
	if !strings.Contains(got, "ONE_IDENTITY_TYPE=user") {
		t.Fatalf("expected ONE_IDENTITY_TYPE env, got %q", got)
	}
}

func TestOneCLIRunsWithoutManagedProvisioning(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	oneBin := writeFakeOne(t)
	client := &OneCLI{Bin: oneBin, WorkDir: t.TempDir()}
	result, err := client.ListConnections(context.Background(), ListConnectionsOptions{})
	if err != nil {
		t.Fatalf("expected local One config/bin fallback to run, got %v", err)
	}
	if got := len(result.Connections); got != 1 {
		t.Fatalf("expected 1 connection, got %d", got)
	}
}

func TestNewOneCLIFromEnvFallsBackToNpx(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("WUPHF_ONE_BIN", "")
	dir := t.TempDir()
	t.Setenv(fakeOneEnv, "npx")
	installFakeOne(t, dir, "npx")
	t.Setenv("PATH", dir)

	client := NewOneCLIFromEnv()
	if client.Bin != "npx" {
		t.Fatalf("expected npx fallback, got %q", client.Bin)
	}
	if got := strings.Join(client.ArgsPrefix, " "); got != "-y @withone/cli" {
		t.Fatalf("unexpected args prefix %q", got)
	}

	result, err := client.ListConnections(context.Background(), ListConnectionsOptions{})
	if err != nil {
		t.Fatalf("expected npx-backed one cli to run, got %v", err)
	}
	if got := len(result.Connections); got != 1 {
		t.Fatalf("expected 1 connection, got %d", got)
	}
}

func TestOneCLIListConnectionsUsesSafeActionWorkDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	workDir := t.TempDir()
	traceFile := filepath.Join(t.TempDir(), "pwd.txt")
	t.Setenv(fakeOneEnv, "trace")
	t.Setenv(fakeOneTraceEnv, traceFile)
	oneBin := installFakeOne(t, t.TempDir(), "one")
	client := &OneCLI{Bin: oneBin, WorkDir: workDir}

	result, err := client.ListConnections(context.Background(), ListConnectionsOptions{})
	if err != nil {
		t.Fatalf("ListConnections returned error: %v", err)
	}
	if len(result.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(result.Connections))
	}

	usedDirRaw, err := os.ReadFile(traceFile)
	if err != nil {
		t.Fatalf("read trace file: %v", err)
	}
	usedDir := strings.TrimSpace(string(usedDirRaw))
	expectedDir, err := filepath.EvalSymlinks(homeDir)
	if err != nil {
		t.Fatalf("resolve home dir: %v", err)
	}
	if usedDir != expectedDir {
		t.Fatalf("expected ListConnections to run from home dir %q, got %q", expectedDir, usedDir)
	}
}

func TestOneCLIExecuteWorkflowKeepsFlowWorkDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	workDir := t.TempDir()
	traceFile := filepath.Join(t.TempDir(), "workflow-pwd.txt")
	t.Setenv(fakeOneEnv, "trace")
	t.Setenv(fakeOneTraceEnv, traceFile)
	oneBin := installFakeOne(t, t.TempDir(), "one")
	client := &OneCLI{Bin: oneBin, WorkDir: workDir}

	workflow, err := client.ExecuteWorkflow(context.Background(), WorkflowExecuteRequest{KeyOrPath: "welcome-flow"})
	if err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}
	if workflow.RunID != "run-1" || workflow.Status != "success" {
		t.Fatalf("unexpected workflow result %+v", workflow)
	}

	usedDirRaw, err := os.ReadFile(traceFile)
	if err != nil {
		t.Fatalf("read trace file: %v", err)
	}
	usedDir := strings.TrimSpace(string(usedDirRaw))
	expectedDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("resolve workdir: %v", err)
	}
	if usedDir != expectedDir {
		t.Fatalf("expected ExecuteWorkflow to run from flow workdir %q, got %q", expectedDir, usedDir)
	}
}

func TestOneCLIExecuteActionAutoResolvesConnectionViaTempFlow(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	oneBin := writeFakeOne(t)
	client := &OneCLI{Bin: oneBin, WorkDir: t.TempDir()}

	result, err := client.ExecuteAction(context.Background(), ExecuteRequest{
		Platform: "slack",
		ActionID: "post-message",
		Data: map[string]any{
			"channel": "#ops",
			"text":    "hello",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteAction returned error: %v", err)
	}
	if result.DryRun {
		t.Fatalf("expected live temp-flow execution result, got dry-run %+v", result)
	}
	if !strings.Contains(string(result.Response), `"posted":true`) {
		t.Fatalf("expected flow response payload, got %s", string(result.Response))
	}
}
