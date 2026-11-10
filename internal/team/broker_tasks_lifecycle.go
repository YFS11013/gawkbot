package team

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// BlockTask transitions taskID to LifecycleStateBlocked and
// records `blockerID` in task.BlockedOn so the unblock cascade fires
// automatically when the blocker merges.
//
// Pass blockerID="" to block without a typed blocker (legacy callers
// that just want to pause a task without naming the dependency).
// Multiple BlockedOn entries are supported; this call appends without
// duplicating an existing entry.
func (b *Broker) BlockTask(taskID, actor, reason, blockerID string) (teamTask, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := strings.TrimSpace(taskID)
	if id == "" {
		return teamTask{}, false, fmt.Errorf("task id required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	reason = strings.TrimSpace(reason)
	blockerID = strings.TrimSpace(blockerID)
	now := time.Now().UTC().Format(time.RFC3339)

	for i := range b.tasks {
		task := &b.tasks[i]
		if task.ID != id {
			continue
		}
		if isTerminalTeamTaskStatus(task.status) {
			return *task, false, nil
		}
		if err := rejectFalseLocalWorktreeBlock(task, reason); err != nil {
			return *task, false, err
		}
		beforeStatus := task.status
		if reason != "" {
			switch existing := strings.TrimSpace(task.Details); {
			case existing == "":
				task.Details = reason
			case !strings.Contains(existing, reason):
				task.Details = existing + "\n\n" + reason
			}
		}
		// Record the blocker ID before the lifecycle transition so the
		// indexed inbox query sees a consistent (state, BlockedOn) tuple
		// the first time it reads. dedup against an existing entry so
		// re-blocking on the same task doesn't grow the list.
		if blockerID != "" && blockerID != task.ID {
			already := false
			// Rename to `dep` so we don't shadow the broker receiver
			// `b` and silently break a future maintainer who adds
			// `b.something(...)` inside this loop.
			for _, dep := range task.BlockedOn {
				if dep == blockerID {
					already = true
					break
				}
			}
			if !already {
				task.BlockedOn = append(task.BlockedOn, blockerID)
			}
		}
		// Route the legacy block path through the lifecycle transition
		// layer so derived fields, the indexed lookup, and the self-heal
		// gate (build-time gate #1) all stay in sync. The transition
		// stamps status/blocked/pipelineStage/reviewState atomically and
		// updates the lifecycleIndex bucket.
		if _, err := b.transitionLifecycleLocked(task.ID, LifecycleStateBlocked, reason); err != nil {
			return *task, false, err
		}
		task.UpdatedAt = now
		if err := rejectTheaterTaskForLiveBusiness(task); err != nil {
			return *task, false, err
		}
		b.scheduleTaskLifecycleLocked(task)
		if err := b.syncTaskWorktreeLocked(task); err != nil {
			return teamTask{}, false, err
		}
		b.appendActionLocked("task_updated", "office", normalizeChannelSlug(task.Channel), actor, truncateSummary(task.Title+" ["+task.status+"]", 140), task.ID)
		// Self-heal gate (build-time gate #1): blocked is a
		// typed legitimate state, not a self-heal trigger. Short-circuit
		// the call site itself so the unit test can observe absence at
		// the call boundary, not just the side-effect downstream.
		if task.LifecycleState != LifecycleStateBlocked {
			b.requestCapabilitySelfHealingLocked(task, actor, reason)
		}
		if err := b.saveLocked(); err != nil {
			return teamTask{}, false, err
		}
		b.emitTaskTransitionAutoNotebook(task, beforeStatus, actor)
		return *task, true, nil
	}

	return teamTask{}, false, fmt.Errorf("task not found")
}

func (b *Broker) ResumeTask(taskID, actor, reason string) (teamTask, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := strings.TrimSpace(taskID)
	if id == "" {
		return teamTask{}, false, fmt.Errorf("task id required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	reason = strings.TrimSpace(reason)
	now := time.Now().UTC().Format(time.RFC3339)

	for i := range b.tasks {
		task := &b.tasks[i]
		if task.ID != id {
			continue
		}
		beforeStatus := task.status
		changed := false
		// Strip the stale rate-limit marker on every ResumeTask call, not
		// only when this call flips Blocked from true to false. A different
		// code path (e.g. unblockDependentsLocked, capability self-healing)
		// may have already cleared Blocked while leaving the marker in
		// Details; without the unconditional strip the watchdog's
		// externalWorkflowRetryAfter check would still detect the stale
		// timestamp on its next tick and re-enter the resume loop. The
		// strip is a no-op when no marker is present.
		if cleaned := stripExternalRetryMarker(task.Details); cleaned != task.Details {
			task.Details = cleaned
			changed = true
		}
		// Honor real dependencies on resume. Clearing a transient block (a
		// rate-limit cooldown, a capability self-heal) must not jump a task
		// ahead of a declared dependency: a task with unresolved DependsOn stays
		// blocked, and unblockDependentsLocked activates it when its dependency
		// completes. This is the inverse of "non-dependent tasks run together" —
		// dependent tasks do NOT. (Previously the now-removed exclusive-owner
		// lane re-blocked here as a side effect; the gate is now explicit.)
		if (task.blocked || strings.EqualFold(strings.TrimSpace(task.status), "blocked")) && !b.hasUnresolvedDepsLocked(task) {
			targetState := LifecycleStateReady
			if strings.TrimSpace(task.Owner) != "" {
				targetState = LifecycleStateRunning
			}
			transitioned, err := b.transitionLifecycleLocked(task.ID, targetState, "task resumed")
			if err != nil {
				return teamTask{}, false, err
			}
			task = transitioned
			changed = true
		}
		if !changed {
			return *task, false, nil
		}
		if reason != "" && !strings.Contains(task.Details, reason) {
			task.Details = strings.TrimSpace(task.Details)
			if task.Details != "" {
				task.Details += "\n\n"
			}
			task.Details += reason
		}
		b.ensureTaskOwnerChannelMembershipLocked(task.Channel, task.Owner)
		b.queueTaskBehindActiveOwnerLaneLocked(task)
		task.UpdatedAt = now
		b.scheduleTaskLifecycleLocked(task)
		if err := b.syncTaskWorktreeLocked(task); err != nil {
			return teamTask{}, false, err
		}
		b.appendActionLocked("task_unblocked", "office", normalizeChannelSlug(task.Channel), actor, truncateSummary(task.Title+" resumed", 140), task.ID)
		if err := b.saveLocked(); err != nil {
			return teamTask{}, false, err
		}
		b.emitTaskTransitionAutoNotebook(task, beforeStatus, actor)
		return *task, true, nil
	}

	return teamTask{}, false, fmt.Errorf("task not found")
}

// CancelTask moves a task to a terminal canceled state: an operator decided
// this work should not happen.
//
// The difference from BlockTask is the whole reason this exists. Both clear
// the current schedule, but blocked is a RECOVERABLE state, so
// scheduleTaskLifecycleLocked's default branch immediately arms a recheck job
// for it. Canceled is terminal, so it lands in the branch that clears
// follow_up_at/recheck_at/reminder_at/due_at and completes the scheduler
// entries. A canceled task therefore cannot wake itself up again, which is
// what "stopped" needs to mean.
//
// Idempotent: canceling a task that is already terminal returns changed=false
// and mutates nothing.
func (b *Broker) CancelTask(taskID, actor, reason string) (teamTask, bool, error) {
	id := strings.TrimSpace(taskID)
	// Same contract as MutateTask's cancel path: registered before the unlock
	// defer so it runs after b.mu is released, and it stops any provider
	// process still working on the task that was just canceled.
	defer func() {
		if id != "" {
			b.cancelHeadlessTurnsForTask(id)
		}
	}()

	b.mu.Lock()
	defer b.mu.Unlock()

	if id == "" {
		return teamTask{}, false, fmt.Errorf("task id required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	reason = strings.TrimSpace(reason)
	now := time.Now().UTC().Format(time.RFC3339)

	for i := range b.tasks {
		task := &b.tasks[i]
		if task.ID != id {
			continue
		}
		if isTerminalTeamTaskStatus(task.status) {
			return *task, false, nil
		}
		beforeStatus := task.status
		// Route through the transition layer so lifecycle_state, the indexed
		// bucket, and the derived fields all move atomically. Writing
		// task.status directly here would leave lifecycle_state pointing at
		// whatever the task used to be -- the exact split that made a restart
		// resurrect this task as in_progress while the lifecycle field still
		// said blocked.
		transitioned, err := b.transitionLifecycleLocked(task.ID, LifecycleStateCanceled, reason)
		if err != nil {
			return teamTask{}, false, err
		}
		task = transitioned
		task.CompletedAt = now
		if reason != "" && !strings.Contains(task.Details, reason) {
			if existing := strings.TrimSpace(task.Details); existing != "" {
				task.Details = existing + "\n\n" + reason
			} else {
				task.Details = reason
			}
		}
		task.UpdatedAt = now
		b.scheduleTaskLifecycleLocked(task)
		if err := b.syncTaskWorktreeLocked(task); err != nil {
			return teamTask{}, false, err
		}
		b.appendActionLocked("task_canceled", "office", normalizeChannelSlug(task.Channel), actor, truncateSummary(task.Title+" canceled", 140), task.ID)
		if err := b.saveLocked(); err != nil {
			return teamTask{}, false, err
		}
		b.emitTaskTransitionAutoNotebook(task, beforeStatus, actor)
		return *task, true, nil
	}

	return teamTask{}, false, fmt.Errorf("task not found")
}

func (b *Broker) handleTaskAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body TaskAckRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := b.AckTask(body)
	if errors.Is(err, errTaskAckInvalid) {
		http.Error(w, "id and slug required", http.StatusBadRequest)
		return
	}
	if errors.Is(err, errTaskAckOwnerOnly) {
		http.Error(w, "only the task owner can ack", http.StatusForbidden)
		return
	}
	if errors.Is(err, errTaskNotFound) {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, errTaskPersistFailed) {
		http.Error(w, "failed to persist", http.StatusInternalServerError)
		return
	}
	if err != nil {
		log.Printf("tasks ack: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (b *Broker) EnsureTask(channel, title, details, owner, createdBy, threadID string, dependsOn ...string) (teamTask, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	channel = b.preferredTaskChannelLocked(channel, createdBy, owner, title, details)
	// "" means the task has no conversation home yet, which is legal. Running
	// these checks on "" would normalise it back to "general".
	if channel != "" {
		if b.findChannelLocked(channel) == nil {
			return teamTask{}, false, fmt.Errorf("channel not found")
		}
		if !b.canAccessChannelLocked(createdBy, channel) {
			return teamTask{}, false, fmt.Errorf("channel access denied")
		}
	}
	title = strings.TrimSpace(title)
	if existing := b.findReusableTaskLocked(taskReuseMatch{
		Channel:  channel,
		Title:    title,
		ThreadID: strings.TrimSpace(threadID),
		Owner:    strings.TrimSpace(owner),
	}); existing != nil {
		now := time.Now().UTC().Format(time.RFC3339)
		beforeStatus := existing.status
		if existing.Details == "" && strings.TrimSpace(details) != "" {
			existing.Details = strings.TrimSpace(details)
		}
		if existing.Owner == "" && strings.TrimSpace(owner) != "" {
			existing.Owner = strings.TrimSpace(owner)
			if !existing.blocked {
				existing.status = "in_progress"
			}
		}
		if existing.ThreadID == "" && strings.TrimSpace(threadID) != "" {
			existing.ThreadID = strings.TrimSpace(threadID)
		}
		b.reindexTaskLifecycleFromLegacyLocked(existing)
		syncTaskMemoryWorkflow(existing, now)
		b.ensureTaskOwnerChannelMembershipLocked(channel, existing.Owner)
		existing.UpdatedAt = now
		b.queueTaskBehindActiveOwnerLaneLocked(existing)
		if err := rejectTheaterTaskForLiveBusiness(existing); err != nil {
			return teamTask{}, false, err
		}
		b.scheduleTaskLifecycleLocked(existing)
		if err := b.syncTaskWorktreeLocked(existing); err != nil {
			return teamTask{}, false, err
		}
		if err := b.saveLocked(); err != nil {
			return teamTask{}, false, err
		}
		b.emitTaskTransitionAutoNotebook(existing, beforeStatus, createdBy)
		return *existing, true, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// Allocate the task ID before choosing the channel so we can name
	// the per-task channel deterministically.
	b.counter++
	taskID := b.allocateIssueIDLocked()
	// Mint a dedicated channel for a task that defaulted to "general" or was
	// created from inside another task's chat (so it never shares one).
	if shouldMintPerTaskChannel(channel, b.channelOwnedByAnotherTaskLocked(channel), &teamTask{
		Title:   title,
		Details: strings.TrimSpace(details),
		Owner:   strings.TrimSpace(owner),
	}) {
		if ch := b.createPerTaskChannelLocked(taskID, title, strings.TrimSpace(owner), strings.TrimSpace(createdBy)); ch != nil {
			channel = ch.Slug
		}
	}
	task := teamTask{
		ID:        taskID,
		Channel:   channel,
		Title:     title,
		Details:   strings.TrimSpace(details),
		Owner:     strings.TrimSpace(owner),
		status:    "open",
		CreatedBy: strings.TrimSpace(createdBy),
		ThreadID:  strings.TrimSpace(threadID),
		DependsOn: dependsOn,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if len(task.DependsOn) > 0 && b.hasUnresolvedDepsLocked(&task) {
		task.blocked = true
	} else if task.Owner != "" {
		task.status = "in_progress"
	}
	syncTaskMemoryWorkflow(&task, now)
	b.ensureTaskOwnerChannelMembershipLocked(channel, task.Owner)
	b.queueTaskBehindActiveOwnerLaneLocked(&task)
	if err := rejectTheaterTaskForLiveBusiness(&task); err != nil {
		return teamTask{}, false, err
	}
	b.scheduleTaskLifecycleLocked(&task)
	if err := b.syncTaskWorktreeLocked(&task); err != nil {
		return teamTask{}, false, err
	}
	b.reindexTaskLifecycleFromLegacyLocked(&task)
	b.tasks = append(b.tasks, task)
	b.appendActionLocked("task_created", "office", channel, createdBy, truncateSummary(task.Title, 140), task.ID)
	if err := b.saveLocked(); err != nil {
		return teamTask{}, false, err
	}
	b.emitTaskTransitionAutoNotebook(&task, "", createdBy)
	return task, false, nil
}

// AppendTaskDetail appends non-duplicate detail text to an existing task without
// changing ownership or status.
func (b *Broker) AppendTaskDetail(taskID, actor, detail string) (teamTask, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := strings.TrimSpace(taskID)
	if id == "" {
		return teamTask{}, fmt.Errorf("task id required")
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return teamTask{}, fmt.Errorf("detail required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}

	for i := range b.tasks {
		task := &b.tasks[i]
		if task.ID != id {
			continue
		}
		_ = appendTaskDetailLocked(task, detail)
		task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		b.appendActionLocked("task_updated", "office", normalizeChannelSlug(task.Channel), actor, truncateSummary(task.Title+" [updated]", 140), task.ID)
		if err := b.saveLocked(); err != nil {
			return teamTask{}, err
		}
		return *task, nil
	}

	return teamTask{}, fmt.Errorf("task not found")
}

func appendTaskDetailLocked(task *teamTask, detail string) error {
	if task == nil {
		return fmt.Errorf("task required")
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return fmt.Errorf("detail required")
	}
	if strings.Contains(task.Details, detail) {
		return nil
	}
	task.Details = strings.TrimSpace(task.Details)
	if task.Details != "" {
		task.Details += "\n\n"
	}
	task.Details += detail
	return nil
}

// hasUnresolvedDepsLocked returns true if any of the task's dependencies
// are still active. Any terminal status — done, completed, canceled,
// cancelled — counts as resolved. This mirrors requestIsResolvedLocked's
// treatment of cancelled humanInterview deps so a parent's cancellation
// no longer permanently orphans every dependent task. Missing deps still
// count as unresolved (dependency doesn't exist yet).
func (b *Broker) hasUnresolvedDepsLocked(task *teamTask) bool {
	for _, depID := range task.DependsOn {
		if requestIsResolvedLocked(b.requests, depID) {
			continue
		}
		found := false
		for j := range b.tasks {
			if b.tasks[j].ID == depID {
				found = true
				if !isTerminalTeamTaskStatus(b.tasks[j].status) {
					return true
				}
				break
			}
		}
		if !found {
			return true // dependency doesn't exist yet — treat as unresolved
		}
	}
	return false
}

// unblockDependentsLocked checks all blocked tasks and unblocks those whose
// dependencies are now resolved. For each newly unblocked task, it appends a
// "task_unblocked" action so the launcher can deliver a notification to the owner.
//
// Lane A extension: the function sweeps the union of (DependsOn, BlockedOn)
// and branches on LifecycleState so harness tasks (blocked) move
// to review on resolution while legacy DependsOn-blocked tasks keep their
// pre-Lane-A in_progress / open behavior.
//
// Returns the list of cascade transitions that the caller must publish to the
// auto-notebook writer AFTER its own saveLocked succeeds. Emitting under
// b.mu before the persist would leak notebook entries for transitions the
// broker subsequently rolled back on save failure (CodeRabbit, major).
func (b *Broker) unblockDependentsLocked(completedTaskID string) []pendingTaskTransition {
	// Dependency release requires COMPLETION, not an approval click
	// (ICP-eval v3 [18:45:40]: OFFICE-253 was released the moment the human
	// approved OFFICE-246, then ran against a one-pager that never existed).
	// OnDecisionRecorded fires this cascade after EVERY decision — including
	// Drafting→Running activations — so verify the named upstream actually
	// reached a terminal status before releasing anything that waited on it.
	// IDs that don't resolve to a task (answered humanInterview requests)
	// pass through: request resolution is checked per-dependent below.
	for i := range b.tasks {
		if b.tasks[i].ID == completedTaskID {
			if !isTerminalTeamTaskStatus(b.tasks[i].status) {
				return nil
			}
			break
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var pending []pendingTaskTransition
	for i := range b.tasks {
		task := &b.tasks[i]
		// A task is "currently blocked" if the legacy `blocked` boolean
		// is set OR the typed LifecycleState reports the block.  Both
		// are accepted so harness tasks (which set LifecycleState before
		// the legacy fields settle) and pre-Lane-A tasks (which only
		// have the legacy flag) cascade the same way.
		if !task.blocked && task.LifecycleState != LifecycleStateBlocked {
			continue
		}
		// Sweep both legacy DependsOn and the new typed BlockedOn list so
		// the same code path resolves harness tasks and pre-Lane-A tasks.
		hasDep := false
		for _, depID := range task.DependsOn {
			if depID == completedTaskID {
				hasDep = true
				break
			}
		}
		if !hasDep {
			for _, depID := range task.BlockedOn {
				if depID == completedTaskID {
					hasDep = true
					break
				}
			}
		}
		if !hasDep {
			continue
		}
		// Snapshot BlockedOn so a transition failure below can roll
		// back the filtered list. Mutating before the transition and
		// then `continue`ing on error used to leave the task in
		// LifecycleStateBlocked with the completed blocker
		// stripped out — the next merge cascade would skip the task
		// (it's no longer in DependsOn/BlockedOn) and the task sat
		// stuck forever.
		originalBlockedOn := append([]string(nil), task.BlockedOn...)
		if len(task.BlockedOn) > 0 {
			filtered := task.BlockedOn[:0]
			for _, depID := range originalBlockedOn {
				if depID == completedTaskID {
					continue
				}
				filtered = append(filtered, depID)
			}
			task.BlockedOn = filtered
		}
		if b.hasUnresolvedDepsLocked(task) || len(task.BlockedOn) > 0 {
			continue
		}
		beforeStatus := task.status
		// Mirror the strip in ResumeTask: if the dependent task was
		// also rate-limited at some earlier point, the stale marker
		// in Details would otherwise trigger the watchdog resume loop
		// even though the dependency completion already unblocked it.
		task.Details = stripExternalRetryMarker(task.Details)
		// Branch on LifecycleState: harness tasks move to review,
		// legacy tasks fall back to the pre-Lane-A behavior. The
		// transition layer stamps every derived field for both paths.
		if task.LifecycleState == LifecycleStateBlocked {
			if _, err := b.transitionLifecycleLocked(task.ID, LifecycleStateReview, "blocker resolved by "+completedTaskID); err != nil {
				// Restore BlockedOn so the next cascade can retry
				// this dependent task instead of silently dropping
				// the unblock signal.
				task.BlockedOn = originalBlockedOn
				log.Printf("broker: unblock cascade transition failed for task %q: %v", task.ID, err)
				continue
			}
		} else {
			targetState := LifecycleStateReady
			if strings.TrimSpace(task.Owner) != "" {
				targetState = LifecycleStateRunning
			}
			transitioned, err := b.transitionLifecycleLocked(task.ID, targetState, "legacy blocker resolved by "+completedTaskID)
			if err != nil {
				// Restore BlockedOn so the next cascade can retry
				// this dependent task instead of silently dropping
				// the unblock signal.
				task.BlockedOn = originalBlockedOn
				log.Printf("broker: unblock cascade legacy transition failed for task %q: %v", task.ID, err)
				continue
			}
			task = transitioned
		}
		b.queueTaskBehindActiveOwnerLaneLocked(task)
		task.UpdatedAt = now
		b.scheduleTaskLifecycleLocked(task)
		_ = b.syncTaskWorktreeLocked(task)
		b.appendActionLocked(
			"task_unblocked",
			"office",
			normalizeChannelSlug(task.Channel),
			"system",
			truncateSummary(task.Title+" unblocked by "+completedTaskID, 140),
			task.ID,
		)
		pending = append(pending, pendingTaskTransition{
			taskID:       task.ID,
			beforeStatus: beforeStatus,
		})
	}
	return pending
}

type taskReuseMatch struct {
	Channel          string
	Title            string
	ThreadID         string
	Owner            string
	PipelineID       string
	SourceSignalID   string
	SourceDecisionID string
}

func (m taskReuseMatch) hasScopedIdentity() bool {
	return strings.TrimSpace(m.PipelineID) != "" ||
		strings.TrimSpace(m.SourceSignalID) != "" ||
		strings.TrimSpace(m.SourceDecisionID) != ""
}

func hasScopedTaskIdentity(task *teamTask) bool {
	if task == nil {
		return false
	}
	return strings.TrimSpace(task.SourceSignalID) != "" ||
		strings.TrimSpace(task.SourceDecisionID) != ""
}

func taskOwnerMatches(task *teamTask, owner string) bool {
	if task == nil {
		return false
	}
	taskOwner := strings.TrimSpace(task.Owner)
	return owner == "" || taskOwner == owner || taskOwner == ""
}

func scopedTaskIdentityMatches(task *teamTask, match taskReuseMatch) bool {
	if task == nil {
		return false
	}
	if match.PipelineID != "" {
		if strings.TrimSpace(task.PipelineID) != match.PipelineID {
			return false
		}
	}
	if match.SourceSignalID != "" && strings.TrimSpace(task.SourceSignalID) != match.SourceSignalID {
		return false
	}
	if match.SourceDecisionID != "" && strings.TrimSpace(task.SourceDecisionID) != match.SourceDecisionID {
		return false
	}
	return true
}

func taskCanMatchScopedIdentity(task *teamTask, match taskReuseMatch) bool {
	if hasScopedTaskIdentity(task) {
		return true
	}
	return strings.TrimSpace(match.PipelineID) != "" && strings.TrimSpace(task.PipelineID) != ""
}

// findReusableTaskLocked looks for an existing non-terminal task that
// matches the given intent (title + owner + optional thread / scoped
// identity).  The search is deliberately channel-agnostic: since each
// new business-objective task now gets its own dedicated channel
// (task-<id>), a duplicate create request that arrives against "general"
// must still find the already-minted task in its per-task channel.
// The Backup & Migration system task has a unique title so it is never
// matched accidentally.
//
// Dedup scope (intentional, bounded): with no thread or scoped identity the
// match is title + owner only, so two genuinely-distinct intents that share a
// title AND owner can collapse into one. That over-match is bounded two ways —
// terminal tasks are skipped (a finished same-title task is never reused; a
// fresh one is minted), and the only collision that survives is between two
// concurrently-active identical-title-and-owner tasks, which in practice is a
// resubmission of the same intent (idempotent re-plan) — exactly the case the
// channel-agnostic search exists to dedup. Callers that need precise identity
// (distinct intents sharing a title) pass a ThreadID or a scoped identity
// (PipelineID / SourceSignalID / SourceDecisionID), which narrows the match.
// TestReuseIsChannelAgnostic and TestFindReusableTaskSkipsTerminal pin both
// halves of this contract.
func (b *Broker) findReusableTaskLocked(match taskReuseMatch) *teamTask {
	title := strings.TrimSpace(match.Title)
	threadID := strings.TrimSpace(match.ThreadID)
	owner := strings.TrimSpace(match.Owner)
	scopedIdentity := match.hasScopedIdentity()
	for i := range b.tasks {
		task := &b.tasks[i]
		if isTerminalTeamTaskStatus(task.status) {
			continue
		}
		// Title match is fuzzy (normalized + token-set similar), not byte-exact:
		// near-duplicate restatements of the same goal ("Ship the MVP" vs "Build
		// the first MVP slice") must collapse onto the existing task instead of
		// spawning a duplicate. See titlesAreSimilar.
		sameTitle := title != "" && titlesAreSimilar(task.Title, title)
		if threadID != "" && strings.TrimSpace(task.ThreadID) == threadID {
			if sameTitle && taskOwnerMatches(task, owner) {
				taskHasScopedIdentity := taskCanMatchScopedIdentity(task, match)
				if scopedIdentity || taskHasScopedIdentity {
					if !scopedIdentity || !taskHasScopedIdentity {
						continue
					}
					if scopedTaskIdentityMatches(task, match) {
						return task
					}
					continue
				}
				return task
			}
			continue
		}
		if !sameTitle || !taskOwnerMatches(task, owner) {
			continue
		}
		taskHasScopedIdentity := taskCanMatchScopedIdentity(task, match)
		if scopedIdentity || taskHasScopedIdentity {
			if !scopedIdentity || !taskHasScopedIdentity {
				continue
			}
			if scopedTaskIdentityMatches(task, match) {
				return task
			}
			continue
		}
		return task
	}
	return nil
}

// verificationNotRequired is the one accepted TaskPostRequest.Verification
// value: the caller states the completion ships no artifact at all (J2-1).
const verificationNotRequired = "not_required"

// isDeliveredTeamTaskStatus reports whether a task actually shipped. Canceled
// and archived work did NOT, which is why the delivered-duplicate guards below
// treat them differently from done (J2-3: after a cancel, rebuilding the same
// work must stay legal).
func isDeliveredTeamTaskStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "completed":
		return true
	default:
		return false
	}
}

// isRetiredTeamTaskStatus reports whether a task stopped being a going concern:
// canceled, rejected, or archived work owns nothing, so guards that hand a file
// or a scope to a child must ignore it.
func isRetiredTeamTaskStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "canceled", "cancelled", "archived", "rejected":
		return true
	default:
		return false
	}
}

// findDeliveredTaskLocked returns the task that ALREADY shipped the work a
// create is about to re-do.
//
// findReusableTaskLocked deliberately skips terminal rows, so a re-issued
// request fell straight through to a brand-new task. Journey 2 J2-6 is exactly
// that: `NEX-100` (renewal emails) and `NEX-102` (account briefs) were `done`,
// the parent was re-split anyway, and `NEX-148`/`NEX-150` redid the same work —
// four hollow completions where two deliveries existed.
//
// Unlike reuse this NEVER merges into the done task (that would swallow the new
// request silently and leave nobody working); the caller refuses the create and
// tells the caller to reopen the delivered task or opt in explicitly.
func (b *Broker) findDeliveredTaskLocked(match taskReuseMatch) *teamTask {
	title := strings.TrimSpace(match.Title)
	if title == "" {
		return nil
	}
	owner := strings.TrimSpace(match.Owner)
	channel := strings.TrimSpace(match.Channel)
	for i := range b.tasks {
		task := &b.tasks[i]
		if !isDeliveredTeamTaskStatus(task.status) {
			continue
		}
		if !titlesAreSimilar(task.Title, title) {
			continue
		}
		// Strict owner equality (unlike reuse's taskOwnerMatches, which also
		// accepts an unowned task): what somebody else delivered must not
		// block a create addressed to a different owner.
		if owner != "" && strings.TrimSpace(task.Owner) != owner {
			continue
		}
		if channel != "" && strings.TrimSpace(task.Channel) != channel {
			continue
		}
		return task
	}
	return nil
}

// rejectTerminalTaskRegressionLocked stops a closed task from being pulled back
// into active work by anything but reopen. Journey 2 J2-3: cancel landed and
// the same second the task was in_progress again, because a queued turn still
// holding the task id claimed it and scheduleTaskLifecycleLocked armed a fresh
// follow-up — the cancel bought nothing. Canceled stays canceled; done stays
// done until a human/CEO reopens it on purpose.
func rejectTerminalTaskRegressionLocked(task *teamTask, action string) error {
	if task == nil {
		return nil
	}
	action = strings.ToLower(strings.TrimSpace(action))
	switch strings.ToLower(strings.TrimSpace(task.status)) {
	case "canceled", "cancelled", "archived":
		// comment: notes on closed work are fine. cancel: idempotent.
		// reopen: the one sanctioned way back.
		switch action {
		case "comment", "cancel", "reopen":
			return nil
		}
		return taskMutationError(TaskMutationConflict,
			fmt.Sprintf("task %s is %s — %s is refused on closed work. A human or the CEO must reopen it first.", task.ID, task.status, action), nil)
	case "done", "completed":
		// Idempotent closes and review verdicts stay allowed; the actions that
		// push a finished task back into execution do not.
		switch action {
		case "claim", "assign", "reassign", "resume", "block", "review", "submit_for_review":
			return taskMutationError(TaskMutationConflict,
				fmt.Sprintf("task %s is already %s — %s is refused on finished work. A human or the CEO must reopen it first.", task.ID, task.status, action), nil)
		}
	}
	return nil
}

// hollowChildrenBlockingParentLocked names a child of task that is done with
// no artifact and no not_required declaration (J2-1): a parent closed on top of
// one aggregates vapor. Returns "" when every child accounts for itself.
func (b *Broker) hollowChildrenBlockingParentLocked(task *teamTask) string {
	if task == nil || strings.TrimSpace(task.ID) == "" {
		return ""
	}
	for i := range b.tasks {
		child := &b.tasks[i]
		if strings.TrimSpace(child.ParentIssueID) != task.ID || child.ID == task.ID {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(child.status), "done") && !strings.EqualFold(strings.TrimSpace(child.status), "completed") {
			continue
		}
		if strings.TrimSpace(child.Artifact) != "" || strings.TrimSpace(child.ArtifactNotRequired) != "" {
			continue
		}
		return child.ID
	}
	return ""
}

func isTerminalTeamTaskStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "completed", "canceled", "cancelled", "archived":
		return true
	default:
		return false
	}
}
