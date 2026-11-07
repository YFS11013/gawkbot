package team

// task_ledger.go — U2.3 per-(bot,task) state ledger + U3.3 living task
// brief (docs/specs/sota-uplift.md).
//
// Every headless turn on a task leaves a distilled journal entry: who acted,
// what they said last, which task mutations they made, and how the turn
// ended. The journal is stored on the task and rendered into every
// participant's execution packet, so turn N+1 — by the same bot or a
// teammate — starts from what turn N tried instead of from amnesia. This is
// the deterministic continuity substrate: fresh sessions stay (crash-safe,
// provider-agnostic), but working memory survives between wakes.
//
// Entries are assembled from broker-observable facts (messages posted,
// action log records, turn outcome), not from asking the model to summarize
// itself — no extra LLM call, no self-report to trust.

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// TaskLedgerEntry is one turn's distilled record on a task.
type TaskLedgerEntry struct {
	Bot     string   `json:"agent"`
	At      string   `json:"at"`
	Outcome string   `json:"outcome"`           // "ok" or a short error reason
	Said    string   `json:"said,omitempty"`    // tail of the bot's last message this turn
	Actions []string `json:"actions,omitempty"` // task/action-log mutations made this turn
	// ContextUsed is the manifest of knowledge items the turn's work packet
	// injected ("learning:<id>", "wiki:<ref>", "upstream:<task>",
	// "journal:<task>"). Recorded deterministically at packet-build time —
	// never model-self-reported — so the human can audit exactly what
	// context the bot was handed (B4). Additive wire field.
	ContextUsed []string `json:"context_used,omitempty"`
}

const (
	// taskLedgerMaxEntries bounds the persisted journal per task.
	taskLedgerMaxEntries = 20
	// taskLedgerRenderEntries is how many recent entries packets carry.
	taskLedgerRenderEntries = 6
	// taskLedgerSaidClip bounds the per-entry message tail.
	taskLedgerSaidClip = 600
)

// noDurableTraceOutcome is what recordTaskLedgerEntry writes for a turn that
// left nothing behind. Matching on this exact string is deliberate: it is
// produced by broker-observable facts (empty Said, empty Actions, no error),
// never self-reported, so it is the one honest "this turn accomplished
// nothing" signal in the journal.
const noDurableTraceOutcome = "ok (no durable trace: no message, no task mutation)"

// maxStalledTurns is how many consecutive trace-free turns a task may have
// before the broker takes it out of rotation.
//
// Eight is two working days' worth of half-hour follow-ups, or one day of
// hourly ones. The bar is "nobody is going to supply what this task is waiting
// for" -- not "the bot had a bad turn". task-skill-2923 spent 42 turns proving
// the same point; eight is already generous.
const maxStalledTurns = 8

// AppendTaskLedgerEntry stamps a journal entry onto the task and persists.
//
// Returns the task's new consecutive-stall count and whether this entry just
// crossed maxStalledTurns. Callers that can take the task out of rotation
// (recordTaskLedgerEntry) use the flag; callers that merely journal (eval
// fixtures) ignore it.
func (b *Broker) AppendTaskLedgerEntry(taskID string, entry TaskLedgerEntry) (stalledTurns int, crossedLimit bool) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || strings.TrimSpace(entry.Bot) == "" {
		return 0, false
	}
	if entry.At == "" {
		entry.At = time.Now().UTC().Format(time.RFC3339)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	task := b.taskByIDLocked(taskID)
	if task == nil {
		return 0, false
	}
	// Progress resets the streak. A task that keeps producing messages or
	// mutations may legitimately run for a long time; only "woke, changed
	// nothing, again" accumulates.
	if entry.Outcome == noDurableTraceOutcome {
		task.StalledTurns++
	} else {
		task.StalledTurns = 0
	}
	task.Ledger = append(task.Ledger, entry)
	if len(task.Ledger) > taskLedgerMaxEntries {
		task.Ledger = task.Ledger[len(task.Ledger)-taskLedgerMaxEntries:]
	}
	task.UpdatedAt = entry.At
	_ = b.saveLocked()
	// == not >= so a replayed entry cannot re-trigger the escalation after
	// the task has already been canceled (canceling clears nothing here, but
	// the caller must not act twice on one crossing).
	return task.StalledTurns, task.StalledTurns == maxStalledTurns
}

// recordTaskLedgerEntry assembles the journal entry for a finished headless
// turn from broker-observable facts. Called from the queue worker after the
// turn settles (no broker lock held on entry).
func (l *Launcher) recordTaskLedgerEntry(slug string, turn headlessCodexTurn, startedAt time.Time, turnErr error) {
	if l == nil || l.broker == nil {
		return
	}
	taskID := strings.TrimSpace(turn.TaskID)
	if taskID == "" {
		return
	}
	entry := TaskLedgerEntry{
		Bot:         slug,
		At:          time.Now().UTC().Format(time.RFC3339),
		Outcome:     "ok",
		ContextUsed: append([]string(nil), turn.ContextUsed...),
	}
	if turnErr != nil {
		entry.Outcome = truncate(turnErr.Error(), 200)
	}
	since := startedAt.UTC().Format(time.RFC3339)

	// Last thing the bot said in the task's channel this turn.
	if ch := normalizeChannelSlug(turn.Channel); ch != "" {
		msgs := l.broker.ChannelMessages(ch)
		for i := len(msgs) - 1; i >= 0; i-- {
			m := msgs[i]
			if m.From != slug || m.Timestamp < since {
				continue
			}
			entry.Said = tailClip(m.Content, taskLedgerSaidClip)
			break
		}
	}

	// Task mutations the bot made this turn (action log is append-only;
	// RelatedID carries the task id for task_* records).
	for _, a := range l.broker.ActionsSince(since) {
		if a.Actor != slug || strings.TrimSpace(a.RelatedID) != taskID {
			continue
		}
		entry.Actions = append(entry.Actions, a.Kind+": "+truncate(a.Summary, 160))
	}

	if entry.Said == "" && len(entry.Actions) == 0 && entry.Outcome == "ok" {
		// A turn that left no observable trace records only its outcome —
		// still worth a line: "bot woke and produced nothing durable" is
		// exactly the signal reviewers and the next turn need.
		entry.Outcome = noDurableTraceOutcome
	}
	stalledTurns, crossedLimit := l.broker.AppendTaskLedgerEntry(taskID, entry)
	if entry.Outcome == noDurableTraceOutcome {
		// A turn that changed nothing should not sleep a full follow-up
		// interval: arm a short retry (2m, doubling) so the owner wakes again
		// while the context is still fresh. Skipped when the task is done,
		// blocked, or waiting on an open human request.
		if err := l.broker.ScheduleNoTraceRetry(taskID); err != nil {
			log.Printf("broker: schedule no-trace retry for %s: %v", taskID, err)
		}
	}
	if !crossedLimit {
		return
	}
	// This turn was the Nth consecutive one that changed nothing. Take the
	// task out of rotation instead of letting the schedule wake it again --
	// the 42-turn lesson, applied before it can repeat.
	//
	// Canceled, not blocked: blocked is recoverable and re-arms a recheck, so
	// it would resume the loop. The reason lands in Details, which every
	// packet renders, so the next reader sees why without opening a notebook.
	reason := fmt.Sprintf("Auto-canceled after %d consecutive turns with no durable trace "+
		"(no message, no task mutation). The follow-up schedule was waking the owner "+
		"with the same input and getting nothing back. If more input is needed, "+
		"supply it and reopen.", stalledTurns)
	if _, _, err := l.broker.CancelTask(taskID, "watchdog", reason); err != nil {
		// Never fail the turn on bookkeeping: the ledger entry is already
		// written, so the next run sees the counter and tries again.
		log.Printf("broker: auto-cancel stalled task %q: %v", taskID, err)
	}
}

// ActionsSince returns action-log records stamped at or after the RFC3339
// timestamp. Snapshot copy; safe for concurrent use.
func (b *Broker) ActionsSince(since string) []officeActionLog {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []officeActionLog
	for _, a := range b.actions {
		if a.CreatedAt >= since {
			out = append(out, a)
		}
	}
	return out
}

// taskLedgerContext renders the living task brief for packets (U3.3): the
// recent journal so every participant wakes knowing what was already tried.
func taskLedgerContext(task teamTask) string {
	if len(task.Ledger) == 0 {
		return ""
	}
	entries := task.Ledger
	if len(entries) > taskLedgerRenderEntries {
		entries = entries[len(entries)-taskLedgerRenderEntries:]
	}
	var lines []string
	for _, e := range entries {
		line := fmt.Sprintf("- [%s @%s] %s", e.At, e.Bot, e.Outcome)
		if len(e.Actions) > 0 {
			line += " | " + strings.Join(e.Actions, "; ")
		}
		if e.Said != "" {
			line += " | said: " + e.Said
		}
		lines = append(lines, line)
	}
	return "TASK JOURNAL (what previous turns already did — continue from here, do not restart or re-discover):\n" + strings.Join(lines, "\n")
}
