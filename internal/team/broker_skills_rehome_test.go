package team

// createSkillRunTaskLocked used to keep whatever channel the invocation came
// from. Skills owned by one bot are invoked from that bot's own
// #<bot>__human DM, while the run is assigned to the office lead — a different
// bot. A DM admits exactly its two participants, so the lead could not post in
// the channel its own task lived in, and could not be promoted into it either
// (ensureTaskOwnerChannelMembershipLocked returns early for a foreign DM).
//
// That is the shape task-skill-2923 was stuck in: 38 wakeups, each correctly
// aborting for a missing input, each unable to report it anywhere.
//
// The fix is to route the task through preferredTaskChannelLocked, the single
// place that already decides a task's home and already re-homes a task out of
// a human DM into the invoker⇄owner pair DM.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedSkillInvokedFromOwnerDM(t *testing.T) *Broker {
	t.Helper()
	b := newTestBroker(t)
	b.mu.Lock()
	b.members = []officeMember{
		{Slug: "cos", Name: "CEO", Role: "lead"},
		{Slug: "data-keeper", Name: "Data Keeper", Role: "worker"},
	}
	b.skills = append(b.skills, teamSkill{
		ID: "skill-mp", Name: "meeting-prep", Title: "Meeting prep",
		Status: "active", Channel: "data-keeper__human", Content: "## Steps\n1. do it",
	})
	b.mu.Unlock()
	return b
}

func invokeSkillInChannel(t *testing.T, b *Broker, name, invoker, channel string) string {
	t.Helper()
	body := bytes.NewBufferString(`{"invoked_by":"` + invoker + `","channel":"` + channel + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/skills/"+name+"/invoke", body)
	rec := httptest.NewRecorder()
	b.handleInvokeSkill(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("invoke %s: status %d body %s", name, rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	id, _ := out["task_id"].(string)
	if id == "" {
		t.Fatalf("invoke %s: no task_id in %+v", name, out)
	}
	return id
}

// The task must not stay in the invoker's DM when the lead owns it: the lead
// has to be able to write where its work lives.
func TestSkillRunTaskReHomedOutOfInvokerDM(t *testing.T) {
	b := seedSkillInvokedFromOwnerDM(t)
	id := invokeSkillInChannel(t, b, "meeting-prep", "data-keeper", "data-keeper__human")

	b.mu.Lock()
	task := b.findTaskByIDLocked(id)
	b.mu.Unlock()
	if task == nil {
		t.Fatalf("task %q not found", id)
	}

	if task.Channel == "data-keeper__human" {
		t.Fatalf("task stayed in the invoker's DM %q; owner %q cannot post there",
			task.Channel, task.Owner)
	}
	// The point of the re-home: the owner can actually write in the home.
	b.mu.Lock()
	ok := b.canAccessChannelLocked(task.Owner, task.Channel)
	b.mu.Unlock()
	if !ok {
		t.Fatalf("owner %q cannot access its own task channel %q", task.Owner, task.Channel)
	}
}

// A skill invoked in a channel everybody can write to must not be moved: the
// re-home exists for DMs, not as a general relocation.
func TestSkillRunTaskKeepsWritableChannel(t *testing.T) {
	b := seedSkillInvokedFromOwnerDM(t)
	b.mu.Lock()
	b.channels = append(b.channels, teamChannel{
		Slug: "eng", Name: "eng",
		Members: []string{"cos", "data-keeper"},
	})
	b.mu.Unlock()

	id := invokeSkillInChannel(t, b, "meeting-prep", "data-keeper", "eng")

	b.mu.Lock()
	task := b.findTaskByIDLocked(id)
	b.mu.Unlock()
	if task == nil {
		t.Fatalf("task %q not found", id)
	}
	if !strings.EqualFold(task.Channel, "eng") {
		t.Fatalf("channel = %q, want eng (writable channels are left alone)", task.Channel)
	}
}
