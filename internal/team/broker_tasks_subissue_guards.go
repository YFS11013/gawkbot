package team

import (
	"fmt"
	"strings"
)

// applySubIssueCreateRulesLocked enforces the sub-issue invariants at create
// time when a task carries a parent_issue_id. Extracted from MutateTask to keep
// broker_tasks_mutation_service.go under the file-size budget. Caller holds b.mu
// and is responsible for rollback + returning the error. No-op for top-level
// tasks (empty ParentIssueID).
//
// Rules:
//   - Force task_type=issue so every sub-issue renders on the Issue detail
//     surface (same lifecycle, packet, components).
//   - Sub-issues nest one level deep only (reject a parent that is itself a
//     sub-issue, so the UI never has to render sub-sub-issue cascades).
//   - Plan-gate: refuse creation while the parent is still in Planning — its
//     plan is not approved, so this is the premature decomposition the planning
//     phase exists to prevent.
//   - Shallow-subtask guard: refuse a sub-issue that merely restates its parent.
//   - Sibling-dedup guard: refuse a sub-issue that duplicates a sibling under
//     the same parent — including a sibling that already SHIPPED (J2-6). Only
//     canceled/archived siblings stay invisible, so rebuilding after a cancel
//     remains legal.
//   - Delivered-parent guard: refuse a new lane under a parent whose every
//     existing lane already shipped — the re-split that redid NEX-100/102 as
//     NEX-148/150 had nothing left to cover.
//
// Internal recovery actors (system/broker/nex) are exempt from the plan/shallow/
// sibling/delivered-parent guards so migration and fold-in paths are never
// blocked. The two delivered-duplicate guards (J2-6) apply to BOT-authored
// creates only — humans and internal actors re-split by design (routines repeat
// the same title every slot). allowDuplicate is the caller's explicit "this is
// genuinely new work that happens to share the shape" opt-out.
func (b *Broker) applySubIssueCreateRulesLocked(task *teamTask, actor string, allowDuplicate bool) error {
	if task == nil || task.ParentIssueID == "" {
		return nil
	}
	task.TaskType = "issue"
	parent := b.findTaskByIDLocked(task.ParentIssueID)
	if parent == nil {
		// Refuse a sub-issue whose parent does not exist — a dangling
		// parent_issue_id would otherwise create an orphan that bypasses every
		// parent-based guard (plan-gate, shallow-restate, sibling dedup).
		return taskMutationError(
			TaskMutationNotFound,
			fmt.Sprintf("parent issue %s not found — create the parent first or fix the parent_issue_id", strings.TrimSpace(task.ParentIssueID)),
			nil,
		)
	}
	if strings.TrimSpace(parent.ParentIssueID) != "" {
		return taskMutationError(
			TaskMutationConflict,
			"sub-issues can only be one level deep; pick the top-level parent instead",
			nil,
		)
	}
	internal := isInternalTaskActor(actor)
	if parent.LifecycleState == LifecycleStatePlanning && !internal {
		return taskMutationError(
			TaskMutationConflict,
			fmt.Sprintf("parent %s is still in planning — its plan has not been approved yet. Finish the plan and wait for the human to approve it before creating sub-tasks.", parent.ID),
			nil,
		)
	}
	if internal {
		return nil
	}
	if titlesAreSimilar(task.Title, parent.Title) {
		return taskMutationError(
			TaskMutationConflict,
			fmt.Sprintf("sub-task %q just restates parent %s — break the parent into distinct, non-overlapping pieces instead of duplicating it.", strings.TrimSpace(task.Title), parent.ID),
			nil,
		)
	}
	// Sibling dedupe. A sibling that already SHIPPED still counts (J2-6): the
	// re-split that redid NEX-100/102 as NEX-148/150 landed next to done lanes
	// under the same parent. Canceled/archived siblings stay invisible, so
	// rebuilding canceled work remains legal (J2-3).
	// J2-6 delivered checks are for autonomous (bot-authored) re-planning.
	// Humans and internal actors are exempt: a human re-splitting an Issue is
	// an explicit act with no escape hatch in the UI, and routine/incident
	// paths repeat titles by design.
	botAuthored := !isHumanMessageSender(actor)
	for j := range b.tasks {
		sib := &b.tasks[j]
		if strings.TrimSpace(sib.ParentIssueID) != task.ParentIssueID {
			continue
		}
		delivered := isDeliveredTeamTaskStatus(sib.status)
		if !delivered && isTerminalTeamTaskStatus(sib.status) {
			continue
		}
		if !titlesAreSimilar(task.Title, sib.Title) {
			continue
		}
		if delivered {
			if allowDuplicate || !botAuthored {
				continue
			}
			return taskMutationError(
				TaskMutationConflict,
				fmt.Sprintf("sub-task %q duplicates sibling %s (%q), which already shipped under parent %s — reopen %s instead of re-creating it, or pass allow_duplicate=true if this is genuinely new work.", strings.TrimSpace(task.Title), sib.ID, strings.TrimSpace(sib.Title), parent.ID, sib.ID),
				nil,
			)
		}
		return taskMutationError(
			TaskMutationConflict,
			fmt.Sprintf("sub-task %q duplicates existing sibling %s (%q) under parent %s — comment on it or pick a distinct slice instead.", strings.TrimSpace(task.Title), sib.ID, strings.TrimSpace(sib.Title), parent.ID),
			nil,
		)
	}
	if botAuthored && !allowDuplicate {
		if err := b.refuseNewLaneUnderDeliveredParentLocked(parent); err != nil {
			return err
		}
	}
	return nil
}

// refuseNewLaneUnderDeliveredParentLocked blocks re-decomposing a parent whose
// every lane already shipped (Journey 2 J2-6). NEX-148/NEX-150 were created
// under NEX-95 minutes after all of its lanes were done, with rephrased titles
// and different owners — title matching cannot see that, but "this parent has
// nothing left open" can.
//
// A canceled/archived lane is NOT delivered and leaves the parent open, so a
// cancel → rebuild cycle (J2-3) is never blocked; the same applies to a parent
// with no children at all.
func (b *Broker) refuseNewLaneUnderDeliveredParentLocked(parent *teamTask) error {
	if parent == nil {
		return nil
	}
	delivered := make([]string, 0, 4)
	for i := range b.tasks {
		sib := &b.tasks[i]
		if strings.TrimSpace(sib.ParentIssueID) != parent.ID {
			continue
		}
		if !isDeliveredTeamTaskStatus(sib.status) {
			return nil
		}
		delivered = append(delivered, sib.ID)
	}
	if len(delivered) == 0 {
		return nil
	}
	return taskMutationError(
		TaskMutationConflict,
		fmt.Sprintf("parent %s has already delivered every lane it had (%s) — reopen one of those or close the parent instead of re-splitting it; pass allow_duplicate=true if this is genuinely new work discovered after delivery.", parent.ID, strings.Join(delivered, ", ")),
		nil,
	)
}
