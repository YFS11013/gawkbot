package team

import (
	"strings"
	"testing"
)

// markTaskDelivered flips a task into the shipped shape the dedupe guards read:
// status done, approved lifecycle, an artifact on record. Seeding directly is
// deliberate — the guards must not depend on HOW a task got delivered.
func markTaskDelivered(t *testing.T, b *Broker, id string) {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	task := b.findTaskByIDLocked(id)
	if task == nil {
		t.Fatalf("task %s not found", id)
	}
	task.status = "done"
	task.LifecycleState = LifecycleStateApproved
	task.Artifact = "team/playbooks/delivered.md"
}

// J2-6: re-issuing a top-level task whose work already shipped is refused and
// names the delivered task, instead of silently creating a second one.
func TestCreateRefusesRedoOfDeliveredTask(t *testing.T) {
	b := newPlanApprovalBroker(t)
	const title = "Send the welcome email"
	first, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: "cos",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	markTaskDelivered(t, b, first.Task.ID)

	_, err = b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: "cos",
	})
	if err == nil {
		t.Fatal("expected a re-issue of delivered work to be refused")
	}
	if !strings.Contains(err.Error(), "already shipped as "+first.Task.ID) {
		t.Fatalf("error should name the delivered task %s, got %v", first.Task.ID, err)
	}
	b.mu.Lock()
	before := len(b.tasks)
	b.mu.Unlock()
	if before != 1 {
		t.Fatalf("refused create must not append a task, have %d", before)
	}

	dup, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: "cos",
		AllowDuplicate: true,
	})
	if err != nil {
		t.Fatalf("allow_duplicate create: %v", err)
	}
	if dup.Task.ID == first.Task.ID {
		t.Fatal("allow_duplicate must create a new task, not reuse the delivered one")
	}
}

// A canceled task is NOT delivered: rebuilding the same work after a cancel
// stays legal (J2-3 semantics), and a different owner is a different task.
func TestCreateAllowsRedoAfterCancelAndForOtherOwner(t *testing.T) {
	b := newPlanApprovalBroker(t)
	const title = "Publish the launch note"
	first, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: "cos",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	b.mu.Lock()
	b.findTaskByIDLocked(first.Task.ID).status = "canceled"
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: "cos",
	}); err != nil {
		t.Fatalf("rebuild after cancel should be allowed, got %v", err)
	}

	second, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: "cos",
	})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	markTaskDelivered(t, b, second.Task.ID)
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: title, Owner: "designer", CreatedBy: "cos",
	}); err != nil {
		t.Fatalf("a different owner should not be blocked by a delivered task, got %v", err)
	}
}

// J2-6: a sub-issue that redoes a sibling which already shipped is refused and
// points at the shipped sibling; allow_duplicate is the explicit opt-out.
func TestSubIssueRefusesRedoOfDeliveredSibling(t *testing.T) {
	b := newPlanApprovalBroker(t)
	parent, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Build the billing system",
		Owner: "eng", CreatedBy: "cos",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	startPlanning(t, b, parent.Task.ID)
	sibling, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Wire Stripe webhooks",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
	})
	if err != nil {
		t.Fatalf("create sibling: %v", err)
	}
	markTaskDelivered(t, b, sibling.Task.ID)

	_, err = b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "wire the stripe webhook",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
	})
	if err == nil {
		t.Fatal("expected a sub-issue redoing a delivered sibling to be refused")
	}
	if !strings.Contains(err.Error(), "already shipped") || !strings.Contains(err.Error(), sibling.Task.ID) {
		t.Fatalf("error should name the delivered sibling %s, got %v", sibling.Task.ID, err)
	}

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "wire the stripe webhook",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
		AllowDuplicate: true,
	}); err != nil {
		t.Fatalf("allow_duplicate sub-issue: %v", err)
	}
}

// J2-6: once every lane a parent had has shipped, a re-split is refused — the
// reported Journey 2 shape (NEX-148/NEX-150 created under a parent whose lanes
// NEX-100/102 had already delivered, with rephrased titles the similarity
// matcher cannot see). A canceled lane keeps the parent open.
func TestSubIssueRefusesNewLaneUnderDeliveredParent(t *testing.T) {
	b := newPlanApprovalBroker(t)
	parent, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Run the Q3 renewal push",
		Owner: "cos", CreatedBy: "human",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	startPlanning(t, b, parent.Task.ID)
	lanes := []string{"Draft the renewal emails", "Assemble the account briefs"}
	createdLanes := make([]string, 0, len(lanes))
	for _, title := range lanes {
		created, err := b.MutateTask(TaskPostRequest{
			Action: "create", Channel: "team", Title: title,
			Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
		})
		if err != nil {
			t.Fatalf("create lane %q: %v", title, err)
		}
		createdLanes = append(createdLanes, created.Task.ID)
	}
	for _, id := range createdLanes {
		markTaskDelivered(t, b, id)
	}

	_, err = b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Write the renewal playbook",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
	})
	if err == nil {
		t.Fatal("expected a new lane under a fully delivered parent to be refused")
	}
	if !strings.Contains(err.Error(), "already delivered every lane") {
		t.Fatalf("error should explain the parent is fully delivered, got %v", err)
	}

	// Canceled is not delivered: rebuilding under that parent must stay legal.
	b.mu.Lock()
	b.findTaskByIDLocked(parent.Task.ID)
	for i := range b.tasks {
		if strings.TrimSpace(b.tasks[i].ParentIssueID) == parent.Task.ID && b.tasks[i].Title == lanes[0] {
			b.tasks[i].status = "canceled"
		}
	}
	b.mu.Unlock()
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Write the renewal playbook",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
	}); err != nil {
		t.Fatalf("a canceled lane should leave the parent open, got %v", err)
	}
}

// The cost of the delivered-parent guard, pinned deliberately: once a parent's
// only lane has shipped, adding another lane needs the explicit opt-in. Lanes
// for one Issue normally land together right after plan approval, so this is
// the rare sequential case — and the error says exactly what to send.
func TestNewLaneAfterDeliveryRequiresOptIn(t *testing.T) {
	b := newPlanApprovalBroker(t)
	parent, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Ship the partner onboarding kit",
		Owner: "cos", CreatedBy: "human",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	startPlanning(t, b, parent.Task.ID)
	lane, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Write the partner checklist",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
	})
	if err != nil {
		t.Fatalf("create lane: %v", err)
	}
	markTaskDelivered(t, b, lane.Task.ID)

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Record the onboarding video",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
	}); err == nil {
		t.Fatal("expected the next lane after full delivery to need the opt-in")
	}
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Record the onboarding video",
		Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
		AllowDuplicate: true,
	}); err != nil {
		t.Fatalf("opt-in lane create: %v", err)
	}
}

// Scope guard: the dedupe only targets bot-authored re-planning. A routine
// fires the same title every due slot from the internal "system" actor, and the
// human new-task form posts as "human" — neither may be blocked by a delivered
// task with that title, or scheduled work stops after its first run.
func TestCreateAllowsRepeatFromSystemAndHumanActors(t *testing.T) {
	b := newPlanApprovalBroker(t)
	const title = "Send the daily ops brief"
	for _, actor := range []string{"system", "human"} {
		first, err := b.MutateTask(TaskPostRequest{
			Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: actor,
		})
		if err != nil {
			t.Fatalf("first create as %s: %v", actor, err)
		}
		markTaskDelivered(t, b, first.Task.ID)
		if _, err := b.MutateTask(TaskPostRequest{
			Action: "create", Channel: "team", Title: title, Owner: "eng", CreatedBy: actor,
		}); err != nil {
			t.Fatalf("repeat create as %s must stay allowed, got %v", actor, err)
		}
	}
}
