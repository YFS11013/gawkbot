package team

// Journey 6 (2026-11-09) lost an approved plan to two silences, and these two
// tests pin the pair.
//
//  1. Canceling JOURN-3 at 06:27:32Z left its interview (request-16) pending.
//     A pending interview parks the asking bot's turn
//     (Launcher.turnParkedOnInterview), so cos stayed parked on a question
//     about canceled work and every wake JOURN-8 was owed got dropped.
//  2. The 06:52:15Z reminder tick wrote a `request_waiting` watchdog row and
//     posted nothing: the automation message was gated on req.Blocking, and a
//     bot-raised interview is not blocking. The human never learned that a
//     reply was the only thing that could unstick the office.

import (
	"strings"
	"testing"
	"time"
)

func TestCanceledTaskClosesItsPendingInterview(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)

	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:             "JOURN-3",
		Title:          "[@analyst] Repeated errors blocked",
		Channel:        "cos__human",
		Owner:          "cos",
		status:         "in_progress",
		LifecycleState: LifecycleStateRunning,
		CreatedAt:      now,
	}}
	b.requests = []humanInterview{{
		ID:        "request-16",
		Kind:      "interview",
		From:      "cos",
		Channel:   "cos__human",
		Title:     "Human interview",
		Question:  "Where does the pipeline data live?",
		Status:    "pending",
		IssueID:   "JOURN-3",
		CreatedAt: now,
		UpdatedAt: now,
	}}
	b.scheduleRequestLifecycleLocked(&b.requests[0])
	armed := 0
	for _, job := range b.scheduler {
		if job.TargetType == "request" && job.TargetID == "request-16" {
			armed++
		}
	}
	b.mu.Unlock()
	if armed == 0 {
		t.Fatalf("precondition: no request job armed for request-16")
	}

	if _, changed, err := b.CancelTask("JOURN-3", "human", "cleanup"); err != nil {
		t.Fatalf("cancel: %v", err)
	} else if !changed {
		t.Fatalf("cancel must report changed=true")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	req := b.requests[0]
	if requestIsActive(req) {
		t.Fatalf("interview still active after its task was canceled: %+v", req)
	}
	if req.Status != "canceled" {
		t.Fatalf("request status = %q, want canceled", req.Status)
	}
	for _, job := range b.scheduler {
		if job.TargetType == "request" && job.TargetID == "request-16" && job.Status != "done" {
			t.Fatalf("request job %q survived the task cancel: status=%q", job.Slug, job.Status)
		}
	}
	recorded := false
	for _, a := range b.actions {
		if a.Kind == "request_canceled" && a.RelatedID == "request-16" {
			recorded = true
			if !strings.Contains(a.Summary, "task closed") {
				t.Fatalf("cancel reason lost the cause: %q", a.Summary)
			}
		}
	}
	if !recorded {
		t.Fatalf("no request_canceled action recorded; actions=%+v", b.actions)
	}
}

// A task's interviews also die when it finishes normally — the question is just
// as moot, and the parked turn is just as costly.
func TestDoneTaskClosesItsPendingInterview(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)

	b.mu.Lock()
	b.tasks = []teamTask{{
		ID:             "JOURN-9",
		Title:          "Ship the quote",
		Channel:        "cos__human",
		Owner:          "cos",
		status:         "in_progress",
		LifecycleState: LifecycleStateRunning,
		CreatedAt:      now,
	}}
	b.requests = []humanInterview{{
		ID:        "request-20",
		Kind:      "interview",
		From:      "cos",
		Channel:   "cos__human",
		Title:     "Human interview",
		Question:  "Which formatting do you want?",
		Status:    "pending",
		IssueID:   "JOURN-9",
		CreatedAt: now,
		UpdatedAt: now,
	}}
	b.mu.Unlock()

	if err := b.TransitionLifecycle("JOURN-9", LifecycleStateApproved, "shipped"); err != nil {
		t.Fatalf("transition: %v", err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if requestIsActive(b.requests[0]) {
		t.Fatalf("interview still active after its task was approved: %+v", b.requests[0])
	}
}

// TestRequestReminderPostsForNonBlockingInterview pins the human-visible half:
// a bot's question is not "blocking" in the office sense, but it parks that
// bot's turn just the same, so the reminder must reach the channel.
func TestRequestReminderPostsForNonBlockingInterview(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	b.requests = []humanInterview{{
		ID: "request-16", Channel: "cos__human", Title: "Human interview", From: "cos",
		Question: "Where does the pipeline data live?", Status: "pending", Blocking: false,
	}}
	b.dueJobs = []schedulerJob{{
		Slug: "request-follow-up:cos--human:request-16", Channel: "cos__human",
		TargetType: "request", TargetID: "request-16",
	}}
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Now()), deliverTask: func(officeActionLog, teamTask) bool { return true }}
	sched.processOnce()

	if len(b.alerts) != 1 || b.alerts[0].kind != "request_waiting" {
		t.Fatalf("expected one request_waiting alert; got %+v", b.alerts)
	}
	if len(b.automationPosts) != 1 {
		t.Fatalf("non-blocking interview reminder posted %d messages, want 1", len(b.automationPosts))
	}
	if b.automationPosts[0].channel != "cos__human" {
		t.Fatalf("reminder landed in %q, want cos__human", b.automationPosts[0].channel)
	}
	if !strings.Contains(b.automationPosts[0].body, "pipeline data") {
		t.Fatalf("reminder does not say what is waiting: %q", b.automationPosts[0].body)
	}
}

// TestPlanDecompositionCheckWakesOwnerWhenNoLanes pins the Journey 6 gap: an
// approved plan whose activation turn spent itself elsewhere leaves the parent
// childless until the hourly follow-up. The check turns that hour into five
// minutes, and says why it fired in the job's close reason.
func TestPlanDecompositionCheckWakesOwnerWhenNoLanes(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	b.tasks = []teamTask{{
		ID: "JOURN-8", Title: "Journey 6: Acme renewal quote", Channel: "cos__human",
		Owner: "cos", status: "in_progress", LifecycleState: LifecycleStateRunning,
	}}
	b.dueJobs = []schedulerJob{{
		Slug: "plan_decomposition_check:cos__human:task:JOURN-8", Channel: "cos__human",
		Kind: planDecompositionCheckKind, TargetType: "task", TargetID: "JOURN-8",
	}}
	delivered := 0
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Now()),
		deliverTask: func(officeActionLog, teamTask) bool { delivered++; return true }}
	sched.processOnce()

	if delivered != 1 {
		t.Fatalf("childless approved plan must wake its owner once; delivered=%d", delivered)
	}
	if len(b.alerts) != 1 || b.alerts[0].kind != "task_not_decomposed" {
		t.Fatalf("expected one task_not_decomposed alert; got %+v", b.alerts)
	}
	if len(b.jobStateUpdates) != 1 || b.jobStateUpdates[0].status != "done" {
		t.Fatalf("check must be one-shot; got %+v", b.jobStateUpdates)
	}
}

// A parent that did grow lanes must not be nudged: the check closes silently
// and names the count, so the run record explains itself without a wake.
func TestPlanDecompositionCheckClosesWhenLanesExist(t *testing.T) {
	b := newSchedulerFixtureBroker(t)
	b.tasks = []teamTask{{
		ID: "JOURN-8", Title: "Journey 6: Acme renewal quote", Channel: "cos__human",
		Owner: "cos", status: "in_progress", LifecycleState: LifecycleStateRunning,
	}}
	b.subIssueTotals = map[string][2]int{"JOURN-8": {2, 0}}
	b.dueJobs = []schedulerJob{{
		Slug: "plan_decomposition_check:cos__human:task:JOURN-8", Channel: "cos__human",
		Kind: planDecompositionCheckKind, TargetType: "task", TargetID: "JOURN-8",
	}}
	delivered := 0
	sched := &watchdogScheduler{broker: b, clock: newManualClock(time.Now()),
		deliverTask: func(officeActionLog, teamTask) bool { delivered++; return true }}
	sched.processOnce()

	if delivered != 0 {
		t.Fatalf("a parent with lanes must not be woken; delivered=%d", delivered)
	}
	if len(b.alerts) != 0 {
		t.Fatalf("a parent with lanes must not raise an alert; got %+v", b.alerts)
	}
	if len(b.jobStateUpdates) != 1 || b.jobStateUpdates[0].status != "done" {
		t.Fatalf("check must close; got %+v", b.jobStateUpdates)
	}
	// The reason has to read as "this parent grew lanes", not as a childless
	// count: Journey 8's parent had six lanes and five already landed when this
	// check fired, so an open-only count recorded "decomposed into 1
	// sub-issue(s)" and the run record looked like the planner never split.
	if got := b.jobStateUpdates[0].reason; !strings.Contains(got, "2 sub-issue(s)") || !strings.Contains(got, "0 still open") {
		t.Fatalf("reason should report lanes and open lanes, got %q", got)
	}
}

// TaskSubIssueTotals is what the reason is built from: it counts the lanes the
// parent grew (canceled/archived excluded, since a cancel → rebuild cycle is
// legal) and, separately, the ones still open. A delivered lane must still count
// toward the total — otherwise a parent whose lanes all landed before the check
// fires reads as childless and gets woken for nothing.
func TestTaskSubIssueTotalsCountsDeliveredLanes(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{
		{ID: "JOURN-6", Title: "Journey 8: Acme renewal", Channel: "team", Owner: "cos", status: "in_progress", CreatedAt: now},
		{ID: "JOURN-15", Title: "Quote draft", Channel: "team", Owner: "eng", ParentIssueID: "JOURN-6", status: "done", CreatedAt: now},
		{ID: "JOURN-17", Title: "Email draft", Channel: "team", Owner: "eng", ParentIssueID: "JOURN-6", status: "in_progress", CreatedAt: now},
		{ID: "JOURN-19", Title: "Canceled slice", Channel: "team", Owner: "eng", ParentIssueID: "JOURN-6", status: "canceled", CreatedAt: now},
	}
	b.mu.Unlock()

	total, open := b.TaskSubIssueTotals("JOURN-6")
	if total != 2 || open != 1 {
		t.Fatalf("totals = (%d,%d), want (2,1): delivered lanes count, canceled ones do not", total, open)
	}
}

// Approving a plan is what arms the check; without that the whole mechanism is
// dead code in production.
func TestApprovedPlanArmsDecompositionCheck(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)

	b.mu.Lock()
	b.tasks = []teamTask{{
		ID: "JOURN-8", Title: "Journey 6: Acme renewal quote", Channel: "cos__human",
		Owner: "cos", status: "in_progress", LifecycleState: LifecycleStatePlanning,
		CreatedAt: now,
	}}
	b.raisePlanApprovalInterviewLocked("JOURN-8", "cos", "the plan")
	var reqID string
	for i := range b.requests {
		if strings.TrimSpace(b.requests[i].IssueID) == "JOURN-8" {
			reqID = b.requests[i].ID
		}
	}
	b.mu.Unlock()
	if reqID == "" {
		t.Fatalf("precondition: no plan-approval interview raised")
	}

	b.mu.Lock()
	for i := range b.requests {
		if b.requests[i].ID != reqID {
			continue
		}
		b.applyRequestAnswerLocked(&b.requests[i], &interviewAnswer{ChoiceID: "approve"}, "human")
	}
	if err := b.saveLocked(); err != nil {
		t.Fatalf("save: %v", err)
	}
	b.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	var found *schedulerJob
	for i := range b.scheduler {
		if b.scheduler[i].Kind == planDecompositionCheckKind && b.scheduler[i].TargetID == "JOURN-8" {
			found = &b.scheduler[i]
		}
	}
	if found == nil {
		t.Fatalf("approval did not arm the decomposition check; jobs=%+v", b.scheduler)
	}
	if found.Status != "scheduled" {
		t.Fatalf("check status = %q, want scheduled", found.Status)
	}
	due, err := time.Parse(time.RFC3339, found.DueAt)
	if err != nil {
		t.Fatalf("check due_at unparseable: %q", found.DueAt)
	}
	if delta := due.Sub(time.Now()); delta > planDecompositionCheckDelay+time.Minute || delta < time.Minute {
		t.Fatalf("check due in %v, want about %v", delta, planDecompositionCheckDelay)
	}
}

// TestDecompositionCheckSurvivesLifecycleRearm pins why the check is exempt from
// the supersede sweep. Journey 7 (2026-11-09) armed it at approval and lost it
// 20 seconds later to "superseded by task_follow_up" as soon as the parent's
// schedule was re-armed — harmless there (the plan did decompose), but the
// stall the check exists for is a task that stays Running while channel
// traffic looks like activity. If any mutation could cancel the guard, the
// guard would be absent exactly when it is needed.
func TestDecompositionCheckSurvivesLifecycleRearm(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)

	b.mu.Lock()
	b.tasks = []teamTask{{
		ID: "JOURN-6", Title: "Journey 7: Acme renewal quote", Channel: "cos__human",
		Owner: "cos", status: "in_progress", LifecycleState: LifecycleStateRunning, CreatedAt: now,
	}}
	b.schedulePlanDecompositionCheckLocked(&b.tasks[0])
	// Any later lifecycle re-arm sweeps competing task jobs.
	b.scheduleTaskLifecycleLocked(&b.tasks[0])
	var check *schedulerJob
	for i := range b.scheduler {
		if b.scheduler[i].Kind == planDecompositionCheckKind {
			check = &b.scheduler[i]
		}
	}
	b.mu.Unlock()

	if check == nil {
		t.Fatalf("decomposition check vanished entirely")
	}
	if check.Status != "scheduled" {
		t.Fatalf("decomposition check status = %q (reason %q), want scheduled",
			check.Status, check.StatusReason)
	}
}

// Journey 15 (core-loop-runs/2026-11-10-journey15): the lane's decision card
// was raised at 14:53:45 and the lane closed itself four minutes later through
// MutateTask, which never touched the card. The operator's approve then landed
// on an already-done lane — a no-op that looked like consent. A card for closed
// work is worse than no card: it asks for a decision nobody can still make.
func TestCompletingALaneClosesItsOrphanedDecisionCard(t *testing.T) {
	b := newTestBroker(t)
	now := time.Now().UTC().Format(time.RFC3339)
	lane := teamTask{
		ID: "JOURN-15", Title: "One-pager quote", Channel: "cos__human", Owner: "cos",
		status: "review", LifecycleState: LifecycleStateDecision, CreatedAt: now, UpdatedAt: now,
		Artifact: "team/accounts/acme-renewal-quote.md",
	}
	sibling := teamTask{
		ID: "JOURN-17", Title: "Follow-up email", Channel: "cos__human", Owner: "cos",
		status: "review", LifecycleState: LifecycleStateDecision, CreatedAt: now, UpdatedAt: now,
	}
	b.mu.Lock()
	b.tasks = []teamTask{lane, sibling}
	b.requests = []humanInterview{
		{ID: "request-31", Kind: "approval", From: "cos", Channel: "cos__human",
			Title: "Approve JOURN-15?", Question: "Approve?", Status: "pending",
			IssueID: "JOURN-15", DedupeKey: "decision:JOURN-15", CreatedAt: now, UpdatedAt: now},
		{ID: "request-34", Kind: "approval", From: "cos", Channel: "cos__human",
			Title: "Approve JOURN-17?", Question: "Approve?", Status: "pending",
			IssueID: "JOURN-17", DedupeKey: "decision:JOURN-17", CreatedAt: now, UpdatedAt: now},
	}
	b.mu.Unlock()

	if _, err := b.MutateTask(TaskPostRequest{
		Action: "complete", ID: "JOURN-15", Channel: "cos__human", CreatedBy: "cos",
		ArtifactPath: "team/accounts/acme-renewal-quote.md",
	}); err != nil {
		t.Fatalf("complete lane: %v", err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if requestIsActive(b.requests[0]) {
		t.Fatalf("the lane's own card survived its close: %+v", b.requests[0])
	}
	if b.requests[0].Status != "canceled" {
		t.Fatalf("card status = %q, want canceled", b.requests[0].Status)
	}
	// The sibling's card is about other work and must stay answerable.
	if !requestIsActive(b.requests[1]) {
		t.Fatalf("closing one lane canceled its sibling's card: %+v", b.requests[1])
	}
	canceled := false
	for _, a := range b.actions {
		if a.Kind == "request_canceled" && a.RelatedID == "request-31" {
			canceled = true
			if !strings.Contains(a.Summary, "task closed") {
				t.Fatalf("cancel reason lost the cause: %q", a.Summary)
			}
		}
	}
	if !canceled {
		t.Fatalf("no request_canceled row for the orphaned card; actions=%+v", b.actions)
	}
}
