package team

// reconcileOrphanedBlockedTasksLocked exists for one failure mode: a task
// blocked on a dependency that reached a terminal status before the unblock
// notification was wired up. It reads "no unresolved dependencies" as "the
// parent already finished, nobody told this task".
//
// A task with NO dependencies has that same reading for a completely different
// reason — a person blocked it on purpose — and the reconciler used to unblock
// it on every boot. task-skill-2923 was blocked by hand after 38 aborted
// wakeups (meeting-prep needs object_name and meeting_date; nobody supplied
// them) and came back as in_progress after the restart, re-arming the
// follow_up schedule.
//
// Both directions are pinned here: the deliberate block survives, the orphaned
// one is still reconciled.

import (
	"testing"
	"time"
)

func TestReconcileOrphanedBlockedLeavesDeliberateBlockAlone(t *testing.T) {
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:             "task-manual-block",
		Title:          "Run skill: meeting-prep",
		Channel:        "cos__human",
		Owner:          "cos",
		status:         "blocked",
		blocked:        true,
		LifecycleState: LifecycleStateBlocked,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}}
	b.reconcileOrphanedBlockedTasksLocked()
	task := b.findTaskByIDLocked("task-manual-block")
	status, blocked := task.status, task.blocked
	b.mu.Unlock()

	if !blocked || status != "blocked" {
		t.Fatalf("a block with no dependencies must survive a boot reconcile: status=%q blocked=%v", status, blocked)
	}
}

func TestReconcileOrphanedBlockedStillUnblocksFinishedDependency(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{
		{ID: "task-parent", Title: "Parent", Channel: "cos__human", Owner: "cos",
			status: "done", CreatedAt: now},
		{ID: "task-child", Title: "Child", Channel: "cos__human", Owner: "cos",
			status: "blocked", blocked: true, LifecycleState: LifecycleStateBlocked,
			DependsOn: []string{"task-parent"}, CreatedAt: now},
	}
	b.reconcileOrphanedBlockedTasksLocked()
	child := b.findTaskByIDLocked("task-child")
	status, blocked := child.status, child.blocked
	b.mu.Unlock()

	if blocked || status != "in_progress" {
		t.Fatalf("an orphaned block (parent already done) must be reconciled: status=%q blocked=%v", status, blocked)
	}
}

func TestReconcileOrphanedBlockedKeepsBlockWhenDependencyActive(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{
		{ID: "task-parent", Title: "Parent", Channel: "cos__human", Owner: "cos",
			status: "in_progress", CreatedAt: now},
		{ID: "task-child", Title: "Child", Channel: "cos__human", Owner: "cos",
			status: "blocked", blocked: true, LifecycleState: LifecycleStateBlocked,
			DependsOn: []string{"task-parent"}, CreatedAt: now},
	}
	b.reconcileOrphanedBlockedTasksLocked()
	child := b.findTaskByIDLocked("task-child")
	status, blocked := child.status, child.blocked
	b.mu.Unlock()

	if !blocked || status != "blocked" {
		t.Fatalf("a block whose dependency is still active must stay blocked: status=%q blocked=%v", status, blocked)
	}
}
