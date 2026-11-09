package team

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestArtifactGate_BindsOnSubIssueOfDefinedParent pins the Journey 1 hole:
// only the parent Issue carried a Definition, so JOURN-24 closed done with the
// three renewal emails still in a notebook. A sub-issue inherits the parent's
// contract, so its done transition must carry a delivered artifact too.
func TestArtifactGate_BindsOnSubIssueOfDefinedParent(t *testing.T) {
	b := newCompletionHookBroker(t)
	parentID := completionHookCreateDefinedTask(t, b)

	child, err := b.MutateTask(TaskPostRequest{
		Action:        "create",
		Channel:       "team",
		Title:         "Draft the renewal emails",
		Owner:         "eng",
		CreatedBy:     "cos",
		ParentIssueID: parentID,
		// A lead-authored lane declares its write scope at creation (§9.15).
		// The test is about the completion gate, so the lane must exist first.
		Definition: &TaskDefinition{
			Goal:         "Draft the renewal emails",
			ArtifactPath: "team/tasks/renewal-emails.md",
		},
	})
	if err != nil {
		t.Fatalf("create sub-issue: %v", err)
	}
	startPlanning(t, b, child.Task.ID)

	err = finishTask(t, b, child.Task.ID, "")
	var mutationErr *TaskMutationError
	if !errors.As(err, &mutationErr) || mutationErr.Kind != TaskMutationArtifactRequired {
		t.Fatalf("expected TaskMutationArtifactRequired for the sub-issue, got %v", err)
	}
	// The gate binds every task now (J2-1), so the message no longer has to
	// explain an inheritance rule; it must name both ways forward.
	if !strings.Contains(mutationErr.Message, "artifact_path") || !strings.Contains(mutationErr.Message, "verification=not_required") {
		t.Fatalf("gate error must name artifact_path and the not_required escape hatch: %q", mutationErr.Message)
	}
	if task := b.TaskByID(child.Task.ID); strings.EqualFold(strings.TrimSpace(task.status), "done") {
		t.Fatalf("gate must roll back the sub-issue's done transition, got %q", task.status)
	}
}

// TestScheduleNoTraceRetry_ArmsShortRecheck checks the short re-arm: a task
// whose turn left no durable trace gets a recheck minutes away, not the hourly
// follow-up. A task with an open blocking request is left alone.
func TestScheduleNoTraceRetry_ArmsShortRecheck(t *testing.T) {
	b := newCompletionHookBroker(t)
	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:           "JOURN-91",
		Channel:      "team",
		Title:        "Draft renewal emails",
		Owner:        "eng",
		status:       "in_progress",
		StalledTurns: 1,
		Definition:   &TaskDefinition{Goal: "draft"},
	}}
	b.mu.Unlock()

	// A lifecycle recheck already on the task must survive the retry: the
	// retry has its own job identity, it does not tighten or overwrite it.
	b.mu.Lock()
	b.scheduler = []schedulerJob{{
		Slug:       normalizeSchedulerSlug("recheck", "team", "task", "JOURN-91"),
		Kind:       "recheck",
		TargetType: "task",
		TargetID:   "JOURN-91",
		Channel:    "team",
		Status:     "scheduled",
		DueAt:      "2026-11-07T07:30:00Z",
		NextRun:    "2026-11-07T07:30:00Z",
	}}
	b.mu.Unlock()

	if err := b.ScheduleNoTraceRetry("JOURN-91"); err != nil {
		t.Fatalf("ScheduleNoTraceRetry: %v", err)
	}
	b.mu.Lock()
	var found *schedulerJob
	rechecks := 0
	for i := range b.scheduler {
		if b.scheduler[i].TargetID != "JOURN-91" {
			continue
		}
		switch b.scheduler[i].Kind {
		case "task_retry":
			found = &b.scheduler[i]
		case "recheck":
			rechecks++
			if b.scheduler[i].NextRun != "2026-11-07T07:30:00Z" {
				t.Errorf("lifecycle recheck was moved: %q", b.scheduler[i].NextRun)
			}
		}
	}
	b.mu.Unlock()
	if found == nil {
		t.Fatal("no task_retry job armed for the no-trace turn")
	}
	if rechecks != 1 {
		t.Fatalf("lifecycle recheck job disappeared (got %d)", rechecks)
	}

	// A blocking human request means the owner is not the blocker.
	b.mu.Lock()
	b.requests = append(b.requests, humanInterview{
		ID:       "req-1",
		Kind:     "approval",
		Status:   "pending",
		Channel:  "team",
		From:     "eng",
		Question: "Approve the send?",
		Blocking: true,
	})
	b.scheduler = nil
	b.mu.Unlock()
	if err := b.ScheduleNoTraceRetry("JOURN-91"); err != nil {
		t.Fatalf("ScheduleNoTraceRetry with request: %v", err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, job := range b.scheduler {
		if job.Kind == "task_retry" && job.TargetID == "JOURN-91" {
			t.Fatal("armed a retry while a blocking request waits on the human")
		}
	}
}

// TestProcessTaskJob_TaskRetryIsOneShot pins the retry's lifecycle: it wakes
// the owner once and closes itself, instead of re-firing on the reminder
// cadence (the next empty turn arms a longer retry through the ledger).
func TestProcessTaskJob_TaskRetryIsOneShot(t *testing.T) {
	b := newCompletionHookBroker(t)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID:      "JOURN-92",
		Channel: "team",
		Title:   "Draft renewal emails",
		Owner:   "eng",
		status:  "in_progress",
	})
	b.mu.Unlock()

	delivered := 0
	s := &watchdogScheduler{
		broker: b,
		clock:  newManualClock(time.Date(2026, 11, 7, 7, 0, 0, 0, time.UTC)),
		deliverTask: func(action officeActionLog, task teamTask) bool {
			delivered++
			return true
		},
	}
	s.processTaskJob(schedulerJob{
		Slug:       normalizeSchedulerSlug("task_retry", "team", "task", "JOURN-92"),
		Kind:       "task_retry",
		TargetType: "task",
		TargetID:   "JOURN-92",
		Channel:    "team",
	})
	if delivered != 1 {
		t.Fatalf("retry delivered %d wakes, want 1", delivered)
	}
	for _, job := range b.Scheduler() {
		if job.Slug == normalizeSchedulerSlug("task_retry", "team", "task", "JOURN-92") {
			if !isClosedSchedulerStatus(job.Status) {
				t.Fatalf("retry job still open after firing: status=%q next=%q", job.Status, job.NextRun)
			}
		}
	}
}

// A retry that finds the task blocked closes instead of re-arming: blockers
// clear on their own schedule and the task's recheck still covers it.
func TestProcessTaskJob_TaskRetryClosesOnBlocked(t *testing.T) {
	b := newCompletionHookBroker(t)
	b.mu.Lock()
	b.tasks = append(b.tasks, teamTask{
		ID:      "JOURN-93",
		Channel: "team",
		Title:   "Draft renewal emails",
		Owner:   "eng",
		status:  "in_progress",
		blocked: true,
		Details: "waiting on provider",
	})
	b.mu.Unlock()

	delivered := 0
	s := &watchdogScheduler{
		broker: b,
		clock:  newManualClock(time.Date(2026, 11, 7, 7, 0, 0, 0, time.UTC)),
		deliverTask: func(action officeActionLog, task teamTask) bool {
			delivered++
			return true
		},
	}
	slug := normalizeSchedulerSlug("task_retry", "team", "task", "JOURN-93")
	b.mu.Lock()
	b.scheduler = append(b.scheduler, schedulerJob{
		Slug:       slug,
		Kind:       "task_retry",
		TargetType: "task",
		TargetID:   "JOURN-93",
		Channel:    "team",
		Status:     "scheduled",
	})
	b.mu.Unlock()

	s.processTaskJob(schedulerJob{
		Slug:       slug,
		Kind:       "task_retry",
		TargetType: "task",
		TargetID:   "JOURN-93",
		Channel:    "team",
	})
	if delivered != 0 {
		t.Fatalf("blocked task got %d wakes, want 0", delivered)
	}
	for _, job := range b.Scheduler() {
		if job.Slug == slug && !isClosedSchedulerStatus(job.Status) {
			t.Fatalf("retry job survived a blocked task: status=%q next=%q", job.Status, job.NextRun)
		}
	}
}
