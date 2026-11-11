package teammcp

// Journey 16: the interview tool aborted on the first failed poll. The broker
// was still inside the caller's own turn, /interview/answer exceeded the 15s
// client timeout three times, and each error ended the tool call — so the model
// reported "采访卡片工具连续超时" and asked the same question again in chat
// while the card it had already raised sat pending, unanswered. The card is the
// durable artifact; a transient poll error must not retire the wait.

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func interviewResultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestHumanInterviewSurvivesATransientPollError(t *testing.T) {
	polls := 0
	srv, _ := stubBroker(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/requests":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"request-9"}`))
		case r.URL.Path == "/interview/answer":
			polls++
			if polls == 1 {
				// The broker is busy with the caller's turn: a 503 here used to
				// end the whole tool call.
				http.Error(w, "broker busy", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"answered","answered":{"choice_id":"","choice_text":"40 seats, $21,600/yr","answered_at":"2026-11-11T01:27:06Z"}}`))
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()
	withBrokerURL(t, srv.URL)
	t.Setenv("WUPHF_AGENT_SLUG", "cos")

	res, _, err := handleHumanInterview(context.Background(), nil, HumanInterviewArgs{
		Question: "What are the renewal terms?",
		Channel:  "cos__human",
	})
	if err != nil {
		t.Fatalf("tool returned a transport error: %v", err)
	}
	if isToolError(res) {
		t.Fatalf("a single slow poll must not fail the interview: %s", toolErrorText(res))
	}
	text := interviewResultText(t, res)
	if !strings.Contains(text, "40 seats, $21,600/yr") {
		t.Fatalf("the human's answer must come back, got %q", text)
	}
	if polls < 2 {
		t.Fatalf("expected the poll to retry after the 503, saw %d poll(s)", polls)
	}
}
