package team

// Decision-field cards: when a lane finishes review and lands in the
// `decision` lifecycle state, the ball is with the human — and until Journey 9
// nothing told them so.
//
// Journey 9's stop table (core-loop-runs/2026-11-09-journey9): JOURN-16 finished
// its email draft at 13:48:03, the reviewer convergence moved it to `decision`,
// and it sat there for the rest of the run. `pending_interview` was null and no
// request pointed at the task: the only way to learn that a decision was owed
// was cos's prose in the channel ("待你确认发送"), and the human's web inbox row
// was the only affordance. The office then read as silent for 26 minutes
// (F-H) because the one party who could move was never asked in the place they
// work — the exact failure `postRequestRaisedChatMessageLocked` exists to
// prevent for every other kind of ask ("a blocking interview sat buried in the
// Inbox for 44 minutes with no push").
//
// So: convergence raises ONE non-blocking approval card per lane, in the lane's
// channel, carrying the deliverable being approved. Answering it performs the
// task action the choice means — approve closes the lane, "send back" bounces it
// to its owner with the note. A card that only recorded an answer and left the
// task in `decision` would be worse than no card at all, which is why the answer
// mapping lives here and not in a separate follow-up.

import (
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	// decisionCardDedupePrefix marks a request as a decision-field card and
	// pins it to one lane. The prefix is also how the answer path recognises
	// which requests own a task mutation.
	decisionCardDedupePrefix = "decision:"

	decisionChoiceApprove = "approve_decision"
	decisionChoiceChanges = "request_changes_decision"
)

// requestIsDecisionCard reports whether a request was raised by reviewer
// convergence for a lane that now needs a human call.
func requestIsDecisionCard(req humanInterview) bool {
	return strings.HasPrefix(strings.TrimSpace(req.DedupeKey), decisionCardDedupePrefix)
}

// childDeliverablesLocked lists what a parent Issue's lanes actually shipped:
// one entry per finished child that recorded an artifact, plus an explicit
// "ships no file" note for children that declared that. Empty for a lane (no
// children) and for a parent whose lanes have not delivered yet.
//
// Journey 10 (core-loop-runs/2026-11-10-journey10): the parent Issue JOURN-14
// reached `decision` at 02:54:20Z with `artifact=null`, even though both of its
// lanes had shipped a file four seconds earlier. `hasApprovableDeliverable` read
// only the Issue's own two fields, so the one party who could sign the work off
// was never asked — F-I fired at 03:04:20Z, ten minutes of a delivered Issue
// with no click available. An Issue is a container: what the human accepts is
// the set of files its lanes landed. Caller holds b.mu.
func (b *Broker) childDeliverablesLocked(task *teamTask) []string {
	if b == nil || task == nil || strings.TrimSpace(task.ID) == "" {
		return nil
	}
	var out []string
	for i := range b.tasks {
		child := &b.tasks[i]
		if strings.TrimSpace(child.ParentIssueID) != task.ID || child.ID == task.ID {
			continue
		}
		if !isDeliveredTeamTaskStatus(child.status) {
			continue
		}
		if artifact := strings.TrimSpace(child.Artifact); artifact != "" {
			out = append(out, artifact)
			continue
		}
		if declared := strings.TrimSpace(child.ArtifactNotRequired); declared != "" {
			out = append(out, fmt.Sprintf("%s (ships no file)", declared))
		}
	}
	return out
}

// childOwningArtifactLocked reports the child that already owns `path` — as a
// delivered `artifact`, or as the write scope the lane declared when it was
// created — so a container mutation can refuse to restate one lane's file as
// the Issue's own. Canceled, rejected, and archived children own nothing.
// Journey 11 (core-loop-runs/2026-11-10-journey11):
// cos submitted JOURN-2 for review with artifact_path set to JOURN-22's
// deliverable, the Issue carried that single path from then on, and the
// promotion + the channel summary both described a two-file Issue by one of its
// files. The read side now lists every lane (see the card), but the write side
// is where the lie was authored. Caller holds b.mu.
func (b *Broker) childOwningArtifactLocked(task *teamTask, path string) (string, bool) {
	if b == nil || task == nil || strings.TrimSpace(path) == "" {
		return "", false
	}
	want := strings.TrimSpace(path)
	for i := range b.tasks {
		child := &b.tasks[i]
		if strings.TrimSpace(child.ParentIssueID) != task.ID || child.ID == task.ID {
			continue
		}
		if isRetiredTeamTaskStatus(child.status) {
			continue
		}
		if isDeliveredTeamTaskStatus(child.status) && strings.TrimSpace(child.Artifact) == want {
			return child.ID, true
		}
		// A lane's declared write scope counts before it delivers. Otherwise
		// the Issue can claim the file a lane is about to write in the window
		// between lane creation and completion, and then overwrite it.
		if child.Definition != nil && strings.TrimSpace(child.Definition.ArtifactPath) == want {
			return child.ID, true
		}
	}
	return "", false
}

// containsPath reports whether path already appears in a list of deliverables.
// The parent card uses it to avoid naming the same file twice: an Issue's own
// `artifact` is often just one of its lanes' paths, restated.
func containsPath(paths []string, path string) bool {
	for _, candidate := range paths {
		if strings.TrimSpace(candidate) == path {
			return true
		}
	}
	return false
}

// hasApprovableDeliverableLocked reports whether a task has anything for the
// human to accept: its own artifact, its own not_required declaration, or the
// deliverables its lanes already landed. A task with none of those has nothing
// to approve, and a card asking to "accept" nothing would only train the human
// to ignore these cards. Caller holds b.mu.
func (b *Broker) hasApprovableDeliverableLocked(task *teamTask) bool {
	if task == nil {
		return false
	}
	if strings.TrimSpace(task.Artifact) != "" || strings.TrimSpace(task.ArtifactNotRequired) != "" {
		return true
	}
	return len(b.childDeliverablesLocked(task)) > 0
}

// raiseDecisionCardLocked raises the decision-field card for a lane that just
// landed in `decision`. Idempotent on the dedupe key, so repeated convergence
// sweeps (the 30-second ticker) never stack cards. Non-blocking by design: the
// lane is not parked on it, no bot's wake is suppressed, and `pendingInterview`
// stays with whichever request really is gating the office. Caller holds b.mu;
// persistence is the caller's responsibility.
func (b *Broker) raiseDecisionCardLocked(task *teamTask) string {
	if b == nil || task == nil || !b.hasApprovableDeliverableLocked(task) {
		return ""
	}
	// A lane under a live Issue does not get its own card: its Issue's card is
	// the acceptance surface, and that card enumerates every lane file
	// ("This Issue's lanes delivered N file(s) … those are what you are
	// accepting"). Journey 15 showed why a second card is worse than none —
	// with no reviewer configured the lane approved itself 38 seconds after
	// raising the card, so the operator's click landed on a closed lane and
	// changed nothing. One acceptance per deliverable set, made by the party
	// who can still move it.
	//
	// A lane whose Issue is already terminal has no such surface left (a
	// container card only exists while the container is open), so it keeps its
	// own card: that is the last place the human can see the work.
	if parentID := strings.TrimSpace(task.ParentIssueID); parentID != "" {
		if parent := b.taskByIDLocked(parentID); parent != nil && !isTerminalTeamTaskStatus(parent.status) {
			return ""
		}
	}
	dedupeKey := decisionCardDedupePrefix + strings.TrimSpace(task.ID)
	for i := range b.requests {
		if !requestIsActive(b.requests[i]) {
			continue
		}
		if strings.TrimSpace(b.requests[i].DedupeKey) == dedupeKey {
			// Already asked. "" (not the existing id) so the convergence hook
			// does not re-persist the whole broker state on every 30-second
			// sweep for a card that has been sitting there all along.
			return ""
		}
	}

	from := strings.TrimSpace(task.Owner)
	if from == "" {
		from = "office"
	}
	channel := normalizeChannelSlug(task.Channel)
	if strings.TrimSpace(task.Channel) == "" {
		if home, err := b.homeChannelForWriterLocked(from, task.Owner, from); err == nil {
			channel = home
		}
	}
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = task.ID
	}

	// Lead with what the human is being asked to accept and what happens either
	// way — the lesson from the plan-approval copy rewrite, where the bot's
	// internal framing ("Plan ready for X. Review it and approve…") left the
	// human unable to tell what it was for.
	var qb strings.Builder
	fmt.Fprintf(&qb, "%s finished %s (%s) and the review is in. Nothing else moves until you call it.\n\n", ownerLabelForPlan(from), title, task.ID)
	lanes := b.childDeliverablesLocked(task)
	own := strings.TrimSpace(task.Artifact)
	switch {
	case len(lanes) > 0:
		// An Issue is a container: name every file its lanes landed. Journey
		// 11 showed why the container case must win over the Issue's own
		// artifact: cos submitted JOURN-2 for review with a single child's
		// path as the Issue artifact, so a card that led with that field told
		// the human "the deliverable is team/accounts/acme-renewal-quote.md"
		// while the follow-up email sat in the same Issue, unreported.
		fmt.Fprintf(&qb, "This Issue's lanes delivered %d file(s):\n\n%s\n", len(lanes), "- "+strings.Join(lanes, "\n- "))
		if own != "" && !containsPath(lanes, own) {
			fmt.Fprintf(&qb, "\nThe Issue itself also records %s.\n", own)
		} else if own == "" {
			qb.WriteString("\nThe Issue ships no single file of its own; those are what you are accepting.\n")
		}
	case own != "":
		fmt.Fprintf(&qb, "The deliverable is %s.\n", own)
	case strings.TrimSpace(task.ArtifactNotRequired) != "":
		fmt.Fprintf(&qb, "The owner declared this lane ships no file (%s).\n", strings.TrimSpace(task.ArtifactNotRequired))
	}
	qb.WriteString("Approve and the lane closes. Send it back and the owner picks up your note as the rework.")

	options := []interviewOption{
		{ID: decisionChoiceApprove, Label: "Approve", Description: "Accept the deliverable and close this lane."},
		{ID: decisionChoiceChanges, Label: "Send back", Description: "Hand it back to the owner with what must change.", RequiresText: true, TextHint: "Type what must change before this can be accepted."},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	b.counter++
	req := humanInterview{
		ID:            fmt.Sprintf("request-%d", b.counter),
		Kind:          "approval",
		Status:        "pending",
		From:          from,
		Channel:       b.requestChannelForLocked(from, channel),
		Title:         "Approve " + title + "?",
		Question:      qb.String(),
		Options:       options,
		RecommendedID: decisionChoiceApprove,
		DedupeKey:     dedupeKey,
		IssueID:       strings.TrimSpace(task.ID),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	b.scheduleRequestLifecycleLocked(&req)
	b.postRequestRaisedChatMessageLocked(&req)
	b.requests = append(b.requests, req)
	if ch := strings.TrimSpace(req.Channel); ch != "" {
		b.appendActionLocked("request_created", "office", ch, from,
			truncateSummary(req.Title+" "+req.Question, 140), req.ID)
	}
	return req.ID
}

// raiseDecisionCardAfterConvergenceLocked is the convergence hook: raise the
// card for a lane that just entered decision, and persist. Caller holds b.mu.
func (b *Broker) raiseDecisionCardAfterConvergenceLocked(taskID string) {
	if b == nil {
		return
	}
	task := b.taskByIDLocked(strings.TrimSpace(taskID))
	if task == nil || task.LifecycleState != LifecycleStateDecision {
		return
	}
	if b.raiseDecisionCardLocked(task) == "" {
		return
	}
	if err := b.saveLocked(); err != nil {
		log.Printf("broker: persist decision card for task %q: %v", taskID, err)
	}
}

// applyDecisionCardAnswer performs the decision a card's answer means. MUST be
// called with b.mu RELEASED: it goes through RecordTaskDecisionWithComment, the
// same chokepoint the web Inbox's Approve / Request-changes buttons use, which
// takes the lock itself and owns every consequence — verification + artifact
// gates, the Review/Decision → Approved transition, the wiki Decision article,
// the channel broadcast, distillation, and the retirement of the requests
// anchored to the lane (which is what makes this card one-shot).
//
// Approving through MutateTask's "approve" instead would have set status=done
// while leaving the task in the decision lifecycle bucket — a split state the
// Inbox would keep listing as "decision required".
//
// Choices map as:
//   - approve_decision         → RecordDecisionApprove (the lane closes)
//   - request_changes_decision → RecordDecisionRequestChanges with the note
//   - a free-text thread reply on the card → request_changes with that text
//     (the human answered in words, which is a rework instruction, not consent)
func (b *Broker) applyDecisionCardAnswer(requestID, actor string) error {
	if b == nil {
		return nil
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil
	}
	var (
		taskID, choice, note string
		state                LifecycleState
		found                bool
	)
	b.mu.Lock()
	for i := range b.requests {
		if b.requests[i].ID != requestID {
			continue
		}
		req := b.requests[i]
		if !requestIsDecisionCard(req) || req.Answered == nil {
			break
		}
		choice = strings.ToLower(strings.TrimSpace(req.Answered.ChoiceID))
		note = strings.TrimSpace(req.Answered.CustomText)
		taskID = strings.TrimSpace(req.IssueID)
		if task := b.taskByIDLocked(taskID); task != nil {
			state = task.LifecycleState
		}
		found = true
		break
	}
	b.mu.Unlock()
	if !found || taskID == "" {
		return nil
	}

	var action recordDecisionAction
	switch choice {
	case decisionChoiceApprove:
		action = RecordDecisionApprove
	case decisionChoiceChanges:
		action = RecordDecisionRequestChanges
	default:
		if choice != "" && choice != "answer_directly" {
			return nil
		}
		action = RecordDecisionRequestChanges
	}
	// This executor is reachable from the HTTP answer endpoint, the Slack gate
	// and both message handlers, so a second click on the same card re-enters it
	// with the answer already recorded. Walking the decision twice would
	// re-broadcast it and re-queue distillation for a task that already shipped;
	// a lane already sitting in the state this action produces means the work is
	// done — report success, touch nothing.
	switch action {
	case RecordDecisionApprove:
		if state == LifecycleStateApproved {
			return nil
		}
	case RecordDecisionRequestChanges:
		if state == LifecycleStateChangesRequested {
			return nil
		}
	}
	// A refusal has nobody else to tell: the answer is already recorded, so hand
	// the reason back to whoever answered. A click that reports success and
	// changes nothing is the exact failure this card exists to end.
	if err := b.RecordTaskDecisionWithComment(taskID, string(action), note, actor); err != nil {
		log.Printf("broker: decision card %s answered %q but the %s decision on task %s was refused: %v", requestID, choice, action, taskID, err)
		return err
	}
	return nil
}

// reportDecisionAnswerFailure tells the channel that a human's reply on a
// decision card answered the card but the decision itself was refused. The
// click path returns the refusal to the caller (409); a thread reply has no
// response body to carry it, and a human who wrote "…send it back" deserves to
// know the lane did not move.
func (b *Broker) reportDecisionAnswerFailure(channel string, err error) {
	if b == nil || err == nil {
		return
	}
	b.PostSystemMessage(channel, "That reply answered the decision card, but the decision could not be applied: "+err.Error(), "decision_card")
}
