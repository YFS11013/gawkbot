package team

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Journey 11 (core-loop-runs/2026-11-10-journey11): the operator tried to
// correct a QA lane on its own task and got
// `500 {"error":"failed to post message"}`. The task lived in the cos↔reviewer
// DM, where humans are observer-only — a legit refusal — but nothing in the
// response named the channel or the reason, so the only way to route the
// correction was to guess.
func TestTaskCommentOnBotDMReturnsTheReason(t *testing.T) {
	b := newTestBroker(t)
	ensureTestMemberAccess(b, "cos", "cos", "Chief of Staff")
	ensureTestMemberAccess(b, "reviewer", "reviewer", "Reviewer")
	if err := b.StartOnPort(0); err != nil {
		t.Fatalf("failed to start broker: %v", err)
	}
	defer b.Stop()

	base := fmt.Sprintf("http://%s", b.Addr())

	// Land a task in the cos↔reviewer pair DM the way the consult relay does:
	// create the channel, then point a task at it.
	b.mu.Lock()
	if dm := b.ensureDMConversationLocked("cos__reviewer"); dm == nil {
		b.mu.Unlock()
		t.Fatal("could not create the cos↔reviewer pair DM")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	const taskID = "JOURN-900"
	b.tasks = append(b.tasks, teamTask{
		ID: taskID, Title: "QA the renewal pack", Channel: "cos__reviewer",
		Owner: "reviewer", TaskType: "issue", status: "in_progress",
		CreatedBy: "cos", CreatedAt: now, UpdatedAt: now,
	})
	channel := normalizeChannelSlug("cos__reviewer")
	before := len(b.messages)
	b.mu.Unlock()
	if _, _, ok := isBotToBotDM(channel); !ok {
		t.Fatalf("this test needs a bot↔bot DM channel, got %q", channel)
	}

	commentBody, _ := json.Marshal(map[string]string{"body": "Both deliverables are on disk — read the paths."})
	commentReq, _ := http.NewRequest(http.MethodPost, base+"/tasks/"+taskID+"/comment", bytes.NewReader(commentBody))
	commentReq.Header.Set("Authorization", "Bearer "+b.Token())
	commentReq.Header.Set("Content-Type", "application/json")
	commentResp, err := http.DefaultClient.Do(commentReq)
	if err != nil {
		t.Fatalf("post comment: %v", err)
	}
	reply, _ := io.ReadAll(commentResp.Body)
	commentResp.Body.Close()

	if commentResp.StatusCode != http.StatusForbidden {
		t.Fatalf("a human posting into a bot DM is a refusal, not a server error: got %d %s", commentResp.StatusCode, reply)
	}
	var parsed struct {
		Error   string `json:"error"`
		Channel string `json:"channel"`
		Detail  string `json:"detail"`
	}
	if err := json.Unmarshal(reply, &parsed); err != nil {
		t.Fatalf("decode refusal: %v (%s)", err, reply)
	}
	if parsed.Channel != channel {
		t.Fatalf("the refusal must name the channel it refused, got %q want %q", parsed.Channel, channel)
	}
	if !strings.Contains(parsed.Detail, "observer-only") {
		t.Fatalf("the refusal must carry the guard's reason, got %q", parsed.Detail)
	}
	b.mu.Lock()
	after := len(b.messages)
	b.mu.Unlock()
	if after != before {
		t.Fatalf("a refused comment must not be stored: %d messages before, %d after", before, after)
	}
}
