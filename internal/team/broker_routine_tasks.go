package team

import (
	"fmt"
	"strings"
)

// routeRoutineTaskActor creates the task on the engine's behalf. It must be an
// internal actor: internal actors skip the CEO plan-approval gate (a routine has
// no human to approve a plan at 08:30) and are unrestricted in task mutation.
const routineTaskActor = "system"

// routineTaskSignal identifies ONE scheduled fire. The due slot (not "now") is
// the stamp so a broker restart that replays the same tick reuses the task it
// already created instead of stacking a second one. The slot arrives as the
// job's persisted RFC3339 next_run, so callers pass it straight through.
func routineTaskSignal(routineSlug, slot string) string {
	stamp := strings.TrimSpace(slot)
	if stamp == "" {
		stamp = "unknown-slot"
	}
	return "routine:" + strings.TrimSpace(routineSlug) + ":" + stamp
}

// routineTaskForSignal returns the task a previous attempt created for the same
// fire, if any. Done/canceled/archived rows do not count: a routine that ran,
// finished, and is asked to fire again at the same slot (clock skew after a
// restart) has genuinely nothing left to dispatch.
func (b *Broker) routineTaskForSignal(signal string) (string, bool) {
	signal = strings.TrimSpace(signal)
	if signal == "" {
		return "", false
	}
	for _, task := range b.AllTasks() {
		if strings.TrimSpace(task.SourceSignalID) != signal {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(task.Status())) {
		case "done", "canceled", "archived":
			continue
		}
		return task.ID, true
	}
	return "", false
}

// fireRoutineAsTask runs one agent routine as a task instead of a bare chat
// message.
//
// Why: a routine used to post a message and nothing else. The bot did the work,
// wrote an artifact, and no task existed — so nothing could carry the artifact
// and core-loop step 7 (distill → skill) had no input for daily briefs. The
// engine now creates the task; the task_created dispatch IS the wake, which is
// why this path does not also post the routine message (that would wake the
// owner twice, once with no task attached).
//
// Returns the schedulerRun trace the caller records, so a failure here is a
// failed run in the Runs tab, never a silent no-op.
func (w *watchdogScheduler) fireRoutineAsTask(job schedulerJob, botSlug, channel, prompt, title, slot, startedAt string) schedulerRun {
	run := schedulerRun{
		Slug:        job.Slug,
		StartedAt:   startedAt,
		TriggeredBy: "scheduler",
		TargetType:  job.TargetType,
		TargetID:    job.TargetID,
	}
	signal := routineTaskSignal(job.Slug, slot)
	if id, ok := w.broker.RoutineTaskForSignal(signal); ok {
		run.Status = "ok"
		run.Message = fmt.Sprintf("Already dispatched as #%s for this slot", id)
		run.OutputSummary = fmt.Sprintf("No new task: #%s already covers %s", id, slot)
		run.Events = []string{"Deduped: an open task already exists for this fire"}
		return run
	}
	resp, err := w.broker.CreateRoutineTask(TaskPostRequest{
		Action:         "create",
		Channel:        channel,
		Title:          title,
		Details:        prompt,
		Owner:          botSlug,
		CreatedBy:      routineTaskActor,
		SourceSignalID: signal,
		// A routine run is only worth distilling when it shipped something.
		// wiki_artifact + required makes "done" mean "the deliverable you
		// named is a real file in the wiki", which is exactly the gate
		// task_distill.go needs before it writes a learning record.
		VerificationKind:     taskVerificationKindWikiArtifact,
		VerificationRequired: true,
	})
	if err != nil {
		// Fall back to the message post so the routine still runs, and record
		// WHY it fell back — a routine that quietly stops producing because the
		// task path broke is the failure mode this whole change exists to end.
		run.Status = "failed"
		run.Message = fmt.Sprintf("Could not create the routine task: %v", err)
		run.ErrorDetail = err.Error()
		run.Events = []string{
			fmt.Sprintf("Task create failed at %s", startedAt),
			"Falling back to the plain routine message so the owner still runs",
		}
		if postErr := w.postRoutineFallbackMessage(job, botSlug, channel, prompt, title, slot); postErr != nil {
			run.Message += fmt.Sprintf("; fallback message also failed: %v", postErr)
			run.ErrorDetail += "; fallback: " + postErr.Error()
			run.Events = append(run.Events, fmt.Sprintf("PostAutomationMessage failed: %v", postErr))
		} else {
			run.Events = append(run.Events, fmt.Sprintf("Posted fallback message to #%s tagging @%s", channel, botSlug))
		}
		return run
	}
	run.Status = "ok"
	run.Message = fmt.Sprintf("Dispatched as task #%s (owner @%s, #%s)", resp.Task.ID, botSlug, channel)
	run.OutputSummary = fmt.Sprintf("Created #%s in #%s and dispatched @%s — artifact can attach to the task", resp.Task.ID, channel, botSlug)
	run.Events = []string{
		fmt.Sprintf("Scheduler tick at %s", startedAt),
		fmt.Sprintf("Created task #%s owned by @%s", resp.Task.ID, botSlug),
		"Routine message not posted: the task dispatch is the wake",
	}
	return run
}

// postRoutineFallbackMessage is the pre-task behaviour: post the routine body as
// an automation message tagged at the owner. Kept only for the failure path.
func (w *watchdogScheduler) postRoutineFallbackMessage(job schedulerJob, botSlug, channel, prompt, title, slot string) error {
	body := strings.TrimSpace(prompt)
	if !isDirectChannelSlug(channel) {
		mention := "@" + botSlug
		if !strings.Contains(body, mention) {
			body = mention + " " + body
		}
	}
	_, _, err := w.broker.PostAutomationMessage(
		"routine",
		channel,
		title,
		body,
		fmt.Sprintf("routine:%s:%s", job.Slug, slot),
		"routine-scheduler",
		"Routine scheduler",
		[]string{botSlug},
		"",
	)
	return err
}

// CreateRoutineTask and RoutineTaskForSignal are the scheduler-facing face of
// the routine-as-task path. They stay on the interface (not the concrete
// broker) so the scheduler keeps its narrow dependency: creating a task is the
// only mutation it makes, and the dedupe lookup is the only read it makes.
func (b *Broker) CreateRoutineTask(req TaskPostRequest) (TaskResponse, error) {
	return b.MutateTask(req)
}

func (b *Broker) RoutineTaskForSignal(signal string) (string, bool) {
	return b.routineTaskForSignal(signal)
}
