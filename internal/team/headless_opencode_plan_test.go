package team

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/nex-crm/wuphf/internal/provider"
)

// A planning task on the Opencode runtime must end its turn by raising the
// plan-approval human_interview. Opencode has no plan/read-only mode and does
// not call ExitPlanMode, so without this the task sits in Planning forever —
// sub-task creation and completion are both refused and no card exists for the
// human to click (Journey 3, 2026-11-09: 36 min idle until a manual approve).
func TestOpencodePlanningTurnRaisesPlanApproval(t *testing.T) {
	oldLookPath := headlessOpencodeLookPath
	oldCommandContext := headlessOpencodeCommandContext
	headlessOpencodeLookPath = func(string) (string, error) { return "/usr/bin/opencode", nil }
	headlessOpencodeCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, os.Args[0], "-test.run=TestHeadlessOpencodeHelperProcess", "--")
	}
	t.Cleanup(func() {
		headlessOpencodeLookPath = oldLookPath
		headlessOpencodeCommandContext = oldCommandContext
	})

	t.Setenv("GO_WANT_HEADLESS_OPENCODE_HELPER_PROCESS", "1")
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("WUPHF_RUNTIME_HOME", tmpHome)

	l := launcherWithActiveTask(t, "eng",
		provider.ProviderBinding{Kind: "opencode", Model: "opencode-go/test"},
		teamTask{ID: "plan-task-1", Title: "plan me", LifecycleState: LifecycleStatePlanning})

	if err := l.runHeadlessOpencodeTurn(t.Context(), "eng", "You have new work in #team."); err != nil {
		t.Fatalf("runHeadlessOpencodeTurn: %v", err)
	}

	b := l.broker
	b.mu.Lock()
	raised := 0
	var planBody string
	for i := range b.requests {
		if requestIsPlanApproval(b.requests[i]) && b.requests[i].IssueID == "plan-task-1" {
			raised++
			planBody = b.requests[i].Question
		}
	}
	b.mu.Unlock()
	if raised != 1 {
		t.Fatalf("planning opencode turn raised %d plan-approval interviews, want 1", raised)
	}
	if planBody == "" {
		t.Fatal("plan-approval interview carried no plan text; the turn's final message must be the plan")
	}
}

// A running (post-approval) task must NOT raise a second approval gate: the
// owner is executing, not planning.
func TestOpencodeRunningTurnRaisesNoPlanApproval(t *testing.T) {
	oldLookPath := headlessOpencodeLookPath
	oldCommandContext := headlessOpencodeCommandContext
	headlessOpencodeLookPath = func(string) (string, error) { return "/usr/bin/opencode", nil }
	headlessOpencodeCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, os.Args[0], "-test.run=TestHeadlessOpencodeHelperProcess", "--")
	}
	t.Cleanup(func() {
		headlessOpencodeLookPath = oldLookPath
		headlessOpencodeCommandContext = oldCommandContext
	})

	t.Setenv("GO_WANT_HEADLESS_OPENCODE_HELPER_PROCESS", "1")
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("WUPHF_RUNTIME_HOME", tmpHome)

	l := launcherWithActiveTask(t, "eng",
		provider.ProviderBinding{Kind: "opencode", Model: "opencode-go/test"},
		teamTask{ID: "run-task-1", Title: "run me", LifecycleState: LifecycleStateRunning})

	if err := l.runHeadlessOpencodeTurn(t.Context(), "eng", "You have new work in #team."); err != nil {
		t.Fatalf("runHeadlessOpencodeTurn: %v", err)
	}

	b := l.broker
	b.mu.Lock()
	raised := 0
	for i := range b.requests {
		if requestIsPlanApproval(b.requests[i]) && b.requests[i].IssueID == "run-task-1" {
			raised++
		}
	}
	b.mu.Unlock()
	if raised != 0 {
		t.Fatalf("running task raised %d plan-approval interviews, want 0", raised)
	}
}

// TestHeadlessOpencodeHelperProcess stands in for the `opencode` binary: it
// emits one JSON text event holding the plan, then exits 0.
func TestHeadlessOpencodeHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HEADLESS_OPENCODE_HELPER_PROCESS") != "1" {
		return
	}
	os.Stdout.WriteString(`{"type":"text","part":{"text":"PLAN: 1) research 2) playbook 3) email"}}` + "\n")
	os.Exit(0)
}
