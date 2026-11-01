package team

import (
	"testing"
	"time"
)

// TestPruneRetiredSchedulerJobs pins the boot sweep: closed one-shot jobs past
// retention go away; anything that can still fire, or that we cannot date,
// stays.
func TestPruneRetiredSchedulerJobs(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-8 * 24 * time.Hour).Format(time.RFC3339)
	fresh := now.Add(-30 * time.Minute).Format(time.RFC3339)

	jobs := []schedulerJob{
		{Slug: "task-follow-up:cos--human:acmer-1", Kind: "task_follow_up", Status: "done", LastRun: stale},
		{Slug: "recheck:cos--human:task:acmer-2", Kind: "recheck", Status: "done", LastRun: stale},
		{Slug: "request-follow-up:cos--human:request-9", Kind: "request_follow_up", Status: "canceled", LastRun: stale},
		{Slug: "task-follow-up:cos--human:acmer-3", Kind: "task_follow_up", Status: "done", LastRun: fresh},
		{Slug: "recheck:cos--human:task:acmer-6", Kind: "recheck", Status: "done", LastRun: now.Add(-2 * time.Hour).Format(time.RFC3339)},
		{Slug: "task-follow-up:cos--human:acmer-4", Kind: "task_follow_up", Status: "done", LastRun: stale, NextRun: now.Format(time.RFC3339)},
		{Slug: "5", Kind: "agent_routine", Status: "done", LastRun: stale},
		{Slug: "task_recheck", Kind: "system", Status: "scheduled", LastRun: stale},
		{Slug: "recheck:cos--human:task:acmer-5", Kind: "recheck", Status: "done", LastRun: "not-a-time"},
	}
	kept, dropped := pruneRetiredSchedulerJobs(jobs, now)

	wantDropped := map[string]bool{
		"task-follow-up:cos--human:acmer-1":      true,
		"recheck:cos--human:task:acmer-2":        true,
		"request-follow-up:cos--human:request-9": true,
		"recheck:cos--human:task:acmer-6":        true,
	}
	if len(dropped) != len(wantDropped) {
		t.Fatalf("dropped %v, want %d entries", dropped, len(wantDropped))
	}
	for _, slug := range dropped {
		if !wantDropped[slug] {
			t.Errorf("dropped %q, which must be kept", slug)
		}
	}
	if len(kept) != len(jobs)-len(wantDropped) {
		t.Fatalf("kept %d jobs, want %d", len(kept), len(jobs)-len(wantDropped))
	}
	for _, slug := range []string{
		"task-follow-up:cos--human:acmer-3", // inside retention
		"task-follow-up:cos--human:acmer-4", // can still fire
		"5",                                 // routine cron
		"task_recheck",                      // system cron
		"recheck:cos--human:task:acmer-5",   // unparsable last run
	} {
		found := false
		for _, job := range kept {
			if job.Slug == slug {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q must survive the sweep", slug)
		}
	}
}
