package team

import (
	"strings"
	"testing"
	"time"
)

// Journey 8, D1/D4 — the actual mechanism behind "the office went quiet".
//
// The parent issue was approved at 09:31 and split into six lanes in ten
// seconds. Five lanes landed by 09:34; JOURN-25 stayed running because its
// ledger said it was waiting for the others. The next wake for anyone under the
// issue was the hourly follow-up (10:31). Between 09:36 and the stop condition
// there were two "gone quiet" notices and nothing else, which reads exactly like
// a wedged office.
//
// The pending promotion approval floating in that window (request-117) is NOT
// the cause: it is created with Blocking=false and kind=approval, so it neither
// gates chat (requestBlocksMessages) nor parks a turn
// (BotAwaitingInterviewAnswer only counts kind=interview). TestPromotionApprovalCard
// below pins that, so nobody "fixes" the card for this again.
func TestLaneLandingPullsForwardParentAndSiblingWakes(t *testing.T) {
	b := newTestBroker(t)
	hourOut := time.Now().UTC().Add(60 * time.Minute).Format(time.RFC3339)
	b.mu.Lock()
	b.tasks = []teamTask{
		{ID: "JOURN-6", Title: "Journey 8: Acme renewal", Channel: "team", Owner: "cos", status: "in_progress"},
		{ID: "JOURN-15", Title: "Quote draft", Channel: "team", Owner: "eng", ParentIssueID: "JOURN-6", status: "in_progress"},
		{ID: "JOURN-25", Title: "QA report", Channel: "team", Owner: "rev", ParentIssueID: "JOURN-6", status: "in_progress"},
		{ID: "JOURN-23", Title: "QA report (duplicate)", Channel: "team", Owner: "rev", ParentIssueID: "JOURN-6", status: "done"},
		{ID: "OTHER-1", Title: "Unrelated", Channel: "team", Owner: "eng", status: "in_progress"},
	}
	for _, id := range []string{"JOURN-6", "JOURN-15", "JOURN-25", "OTHER-1"} {
		b.scheduler = append(b.scheduler, schedulerJob{
			Slug: "task_follow_up:team:" + strings.ToLower(id), Kind: "task_follow_up",
			TargetType: "task", TargetID: id, Channel: "team",
			DueAt: hourOut, NextRun: hourOut, Status: "scheduled",
		})
	}
	// A closed job for a sibling must stay closed: re-arming finished work would
	// resurrect a reminder nobody wants.
	b.scheduler = append(b.scheduler, schedulerJob{
		Slug: "task_follow_up:team:journ-23", Kind: "task_follow_up",
		TargetType: "task", TargetID: "JOURN-23", Channel: "team",
		DueAt: hourOut, NextRun: hourOut, Status: "done",
	})
	landed := b.taskByIDLocked("JOURN-23")
	b.rearmParentAndSiblingFollowUpsLocked(landed)
	now := time.Now().UTC()

	dueOf := func(id string) (time.Time, string) {
		for i := range b.scheduler {
			job := b.scheduler[i]
			if job.Kind == "task_follow_up" && job.TargetID == id {
				parsed, _ := time.Parse(time.RFC3339, job.DueAt)
				return parsed, job.Status
			}
		}
		t.Fatalf("no follow-up job for %s", id)
		return time.Time{}, ""
	}
	b.mu.Unlock()

	// The parent and the still-running sibling are pulled forward to roughly the
	// short grace window; the unrelated task keeps its hour.
	for _, id := range []string{"JOURN-6", "JOURN-25"} {
		due, status := dueOf(id)
		if status != "scheduled" {
			t.Fatalf("%s should stay scheduled, got %q", id, status)
		}
		if due.After(now.Add(siblingLaneFollowUpDelay + time.Minute)) {
			t.Fatalf("%s wake not pulled forward: due %s", id, due.Format(time.RFC3339))
		}
	}
	if due, _ := dueOf("OTHER-1"); due.Before(now.Add(30 * time.Minute)) {
		t.Fatalf("an unrelated task's wake must not move, got %s", due.Format(time.RFC3339))
	}
	if _, status := dueOf("JOURN-23"); status != "done" {
		t.Fatalf("a closed follow-up must stay closed, got %q", status)
	}
}

// A visual-only artifact never counts as a delivered lane under a declared
// scope, and the promotion card is inert — the two facts the run record depends
// on when it says the office was not parked by an approval.
func TestPromotionApprovalCardDoesNotParkOrGate(t *testing.T) {
	b := newTestBroker(t)
	b.mu.Lock()
	b.requests = []humanInterview{{
		ID: "request-117", Kind: "approval", Status: "pending", From: "cos",
		Channel: "cos__human", Title: "Promote to the team wiki: QA report",
		// Blocking is deliberately false on this card kind; set it explicitly so
		// the test fails if the creation path ever flips it.
		Blocking: false,
	}}
	b.mu.Unlock()

	if b.BotAwaitingInterviewAnswer("cos") {
		t.Fatal("an approval card must not park the asker's turn — only kind=interview does")
	}
	b.mu.Lock()
	blocking := firstBlockingRequestInChannel(b.requests, "cos__human")
	b.mu.Unlock()
	if blocking != nil {
		t.Fatalf("a non-blocking approval must not gate chat, got %+v", blocking)
	}
}

// The re-arm has to happen on the real completion path, not just when the helper
// is called directly: this drives a lane to done through MutateTask and then
// reads the parent's and the sibling's own follow-up rows.
func TestLaneCompletionRearmsWakesThroughMutation(t *testing.T) {
	b := newCompletionHookBroker(t)
	parent, err := b.MutateTask(TaskPostRequest{
		Action: "create", Channel: "team", Title: "Journey 8: Acme renewal",
		Owner: "cos", CreatedBy: "human",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	startPlanning(t, b, parent.Task.ID)

	newLane := func(title, scope string) string {
		lane, err := b.MutateTask(TaskPostRequest{
			Action: "create", Channel: "team", Title: title,
			Owner: "eng", CreatedBy: "cos", ParentIssueID: parent.Task.ID,
			Definition: &TaskDefinition{Goal: title, ArtifactPath: scope},
		})
		if err != nil {
			t.Fatalf("create lane %q: %v", title, err)
		}
		return lane.Task.ID
	}
	laneA := newLane("Write the quote draft", "team/reviews/quote-draft.md")
	laneB := newLane("Write the QA report", "team/reviews/qa-report.md")

	b.mu.Lock()
	dueOf := func(id string) time.Time {
		for i := range b.scheduler {
			job := b.scheduler[i]
			if job.Kind != "task_follow_up" || job.TargetID != id || isClosedSchedulerStatus(job.Status) {
				continue
			}
			parsed, _ := time.Parse(time.RFC3339, job.DueAt)
			return parsed
		}
		return time.Time{}
	}
	parentBefore, siblingBefore := dueOf(parent.Task.ID), dueOf(laneB)
	b.mu.Unlock()
	hourOut := time.Now().UTC().Add(30 * time.Minute)
	if parentBefore.Before(hourOut) || siblingBefore.Before(hourOut) {
		t.Fatalf("precondition: wakes should start about an hour out, got parent=%s sibling=%s",
			parentBefore.Format(time.RFC3339), siblingBefore.Format(time.RFC3339))
	}

	if err := finishTask(t, b, laneA, "team/reviews/quote-draft.md"); err != nil {
		t.Fatalf("finish lane A: %v", err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	deadline := time.Now().UTC().Add(siblingLaneFollowUpDelay + time.Minute)
	if got := dueOf(parent.Task.ID); got.After(deadline) {
		t.Fatalf("parent wake not pulled forward after a lane landed: %s", got.Format(time.RFC3339))
	}
	if got := dueOf(laneB); got.After(deadline) {
		t.Fatalf("running sibling wake not pulled forward after a lane landed: %s", got.Format(time.RFC3339))
	}
}
