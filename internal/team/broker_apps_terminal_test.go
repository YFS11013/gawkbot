package team

import (
	"testing"
)

// Terminal transitions retire the build lane: a task that ends
// (approved/archived/rejected) without publishing must not leave its app
// draft in "building" forever (sidebar placeholder). Regression test for
// the 2026-10-28 live incident (AI情报板: task done, app stuck building).
func TestTerminalTransitionFailsBuildingDraft(t *testing.T) {
	t.Setenv("WUPHF_RUNTIME_HOME", t.TempDir())
	b := newTestBroker(t)

	// Building draft bound to the task channel.
	seedAcceptanceApp(t, "app_00000000000000aa", "task-APP-1", 100, customAppStatusBuilding, 0)
	// Ready app on the same channel must survive.
	seedAcceptanceApp(t, "app_00000000000000bb", "task-APP-1", 100, customAppStatusReady, 1)
	// Building draft on another channel must survive.
	seedAcceptanceApp(t, "app_00000000000000cc", "task-OTHER-9", 100, customAppStatusBuilding, 0)

	task := teamTask{ID: "APP-1", Channel: "task-APP-1", LifecycleState: LifecycleStateRunning}
	b.tasks = append(b.tasks, task)

	b.mu.Lock()
	defer b.mu.Unlock()
	stored := &b.tasks[len(b.tasks)-1]
	if err := b.applyLifecycleStateLocked(stored, LifecycleStateApproved); err != nil {
		t.Fatalf("transition: %v", err)
	}

	got := func(id string) string {
		app, _, err := b.appStore().Get(id)
		if err != nil {
			t.Fatalf("get %s: %v", id, err)
		}
		return app.Status
	}
	if s := got("app_00000000000000aa"); s != customAppStatusFailed {
		t.Errorf("building draft on task channel = %q, want failed", s)
	}
	if s := got("app_00000000000000bb"); s != customAppStatusReady {
		t.Errorf("ready app on task channel = %q, want untouched", s)
	}
	if s := got("app_00000000000000cc"); s != customAppStatusBuilding {
		t.Errorf("building draft on other channel = %q, want untouched", s)
	}
}
