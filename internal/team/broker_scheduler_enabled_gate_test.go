package team

import (
	"testing"
	"time"
)

func TestSchedulerJobDueRespectsDisabledRoutine(t *testing.T) {
	past := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	now := time.Now()

	disabled := schedulerJob{Kind: "agent_routine", Enabled: false, NextRun: past, Status: "disabled"}
	if schedulerJobDue(disabled, now) {
		t.Fatal("disabled agent routine reported due")
	}
	enabled := schedulerJob{Kind: "agent_routine", Enabled: true, NextRun: past, Status: "scheduled"}
	if !schedulerJobDue(enabled, now) {
		t.Fatal("enabled agent routine not due")
	}
	followUp := schedulerJob{Kind: "task_follow_up", NextRun: past, Status: "scheduled"}
	if !schedulerJobDue(followUp, now) {
		t.Fatal("task follow-up must keep firing regardless of Enabled")
	}
}
