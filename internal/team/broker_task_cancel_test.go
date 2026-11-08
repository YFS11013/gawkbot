package team

// POST /tasks/{id}/cancel exists because BlockTask cannot stop a task.
//
// Blocked is a RECOVERABLE state: scheduleTaskLifecycleLocked's default branch
// arms a recheck job for it, so a blocked task wakes itself up again. Measured
// on task-skill-2923 -- blocked by hand, it ran two more times before anyone
// noticed, because blocking only swapped task_follow_up for recheck.
//
// Canceled is terminal, so it lands in the branch that clears every lifecycle
// deadline and completes the scheduler entries. These tests pin that
// difference, not the mechanics of the transition itself.

import (
	"errors"
	"testing"
	"time"
)

func TestCancelTaskIsTerminalAndDisarmsScheduling(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:        "task-runaway",
		Title:     "Run skill: meeting-prep",
		Channel:   "cos__human",
		Owner:     "cos",
		status:    "in_progress",
		CreatedAt: now,
	}}
	// Arm the schedule the way a live in_progress task has it, so the test
	// observes a real clearing rather than a no-op on empty fields.
	b.scheduleTaskLifecycleLocked(&b.tasks[0])
	armedFollowUp := b.tasks[0].FollowUpAt
	armedRecheck := b.tasks[0].RecheckAt
	b.mu.Unlock()

	if armedFollowUp == "" || armedRecheck == "" {
		t.Fatalf("precondition: schedule not armed (follow_up=%q recheck=%q)", armedFollowUp, armedRecheck)
	}

	got, changed, err := b.CancelTask("task-runaway", "human", "42 aborted rounds")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if !changed {
		t.Fatalf("canceling an in_progress task must report changed=true")
	}
	if got.status != "canceled" {
		t.Fatalf("status = %q, want canceled", got.status)
	}
	if got.LifecycleState != LifecycleStateCanceled {
		t.Fatalf("lifecycle_state = %q, want canceled", got.LifecycleState)
	}
	// The point of the whole endpoint: no deadline survives, so nothing can
	// wake the task again.
	for name, v := range map[string]string{
		"follow_up_at": got.FollowUpAt,
		"recheck_at":   got.RecheckAt,
		"reminder_at":  got.ReminderAt,
		"due_at":       got.DueAt,
	} {
		if v != "" {
			t.Fatalf("%s = %q, want empty after cancel", name, v)
		}
	}
}

func TestCancelTaskIsIdempotent(t *testing.T) {
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:      "task-already-done",
		Title:   "Finished work",
		Channel: "cos__human",
		Owner:   "cos",
		status:  "done",
	}}
	b.mu.Unlock()

	got, changed, err := b.CancelTask("task-already-done", "human", "noop")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if changed {
		t.Fatalf("canceling a terminal task must report changed=false")
	}
	if got.status != "done" {
		t.Fatalf("status = %q, want done (a terminal task must not be rewritten)", got.status)
	}
}

func TestCancelTaskSurvivesRestartUnlikeBlock(t *testing.T) {
	// The split between the two status fields is what let a restart resurrect
	// a blocked task as in_progress while lifecycle_state still said blocked.
	// Cancel writes both, so neither field can disagree after a reload.
	b := newTestBroker(t)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:      "task-drifted",
		Title:   "Run skill: meeting-prep",
		Channel: "cos__human",
		Owner:   "cos",
		status:  "blocked",
		blocked: true,
	}}
	b.mu.Unlock()

	if _, _, err := b.CancelTask("task-drifted", "human", "restart-survival probe"); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	b.mu.Lock()
	canceled := b.tasks[0]
	// Replay the load path the boot reconciler uses: derive the lifecycle
	// state from the persisted tuple.
	derived := deriveLifecycleStateFromLegacy(canceled.pipelineStage, canceled.reviewState, canceled.status, canceled.blocked)
	b.mu.Unlock()

	if derived != LifecycleStateCanceled {
		t.Fatalf("reloading canceled task derives %q, want canceled (tuple: stage=%q review=%q status=%q blocked=%v)",
			derived, canceled.pipelineStage, canceled.reviewState, canceled.status, canceled.blocked)
	}
	if isExecutableTeamTaskStatus(derived) {
		t.Fatalf("a canceled task must not be executable, or it re-dispatches on boot")
	}
}

// Cancel must close every scheduler job pointing at the task, whatever channel
// the job was armed in (J2-3). Journey 2 found a surviving task_follow_up after
// cancel because the close matched on the task's current home channel while the
// job still carried an older one — the task woke itself and ran again.
func TestCancelClosesJobsAcrossChannels(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID: "task-nex-95", Title: "Audit the renewal pipeline", Channel: "team",
		Owner: "cos", status: "in_progress", CreatedAt: now, UpdatedAt: now,
	}}
	b.scheduleTaskLifecycleLocked(&b.tasks[0])
	// Jobs from an earlier home channel, as a live run accumulates.
	b.scheduler = append(b.scheduler,
		schedulerJob{
			Slug: "task-follow-up:cos--human:task-nex-95", Kind: "task_follow_up",
			TargetType: "task", TargetID: "task-nex-95", Channel: "cos__human",
			Status: "scheduled", NextRun: now,
		},
		schedulerJob{
			Slug: "recheck:designer--human:task-nex-95", Kind: "recheck",
			TargetType: "task", TargetID: "task-nex-95", Channel: "designer__human",
			Status: "scheduled", NextRun: now,
		},
	)
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "cancel", ID: "task-nex-95", Channel: "team", CreatedBy: "human",
		Details: "stop spending on this",
	}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for _, job := range b.scheduler {
		if job.TargetType != "task" || job.TargetID != "task-nex-95" {
			continue
		}
		if !isClosedSchedulerStatus(job.Status) {
			t.Errorf("job %s (channel %q) still %q after cancel — it will wake a canceled task", job.Slug, job.Channel, job.Status)
		}
	}
}

// A canceled task is not a work surface: the actions that would push it back
// into execution are refused until someone reopens it on purpose (J2-3).
// Journey 2's cancel was undone the same second by a claim from an in-flight
// turn, which rearmed the follow-up and bought another 11-tool-call round.
func TestCanceledTaskRefusesRevivalActions(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID: "task-canceled", Title: "Rewrite the onboarding email", Channel: "team",
		Owner: "cos", status: "in_progress", CreatedAt: now, UpdatedAt: now,
	}}
	b.scheduleTaskLifecycleLocked(&b.tasks[0])
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "cancel", ID: "task-canceled", Channel: "team", CreatedBy: "human",
	}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	for _, action := range []string{"claim", "assign", "reassign", "resume", "block", "review", "complete", "approve"} {
		_, err := b.MutateTask(TaskPostRequest{
			Action: action, ID: "task-canceled", Channel: "team", CreatedBy: "human", Owner: "eng",
		})
		var mutationErr *TaskMutationError
		if !errors.As(err, &mutationErr) || mutationErr.Kind != TaskMutationConflict {
			t.Fatalf("%s on a canceled task must be a conflict, got %v", action, err)
		}
		if task := b.TaskByID("task-canceled"); task == nil || task.status != "canceled" {
			t.Fatalf("%s must leave the task canceled, got %+v", action, task)
		}
	}

	b.mu.Lock()
	for _, job := range b.scheduler {
		if job.TargetID == "task-canceled" && !isClosedSchedulerStatus(job.Status) {
			t.Errorf("refused action rearmed job %s (%q)", job.Slug, job.Status)
		}
	}
	b.mu.Unlock()

	// Notes on closed work stay open, and reopen is the sanctioned way back.
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "comment", ID: "task-canceled", Channel: "team", CreatedBy: "cos",
		Details: "canceled per human",
	}); err != nil {
		t.Fatalf("comment on a canceled task must be allowed: %v", err)
	}
	if _, err := b.MutateTask(TaskPostRequest{
		Action: "reopen", ID: "task-canceled", Channel: "team", CreatedBy: "human",
	}); err != nil {
		t.Fatalf("reopen by human must be allowed: %v", err)
	}
	if task := b.TaskByID("task-canceled"); task == nil || task.status == "canceled" {
		t.Fatalf("reopen must revive the task, got %+v", task)
	}
}
