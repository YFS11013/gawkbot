package provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nex-crm/wuphf/internal/bot"
)

// TestOpenAICompatStreamRetriesGatewayErrors pins the backoff fix: a
// transient 503 followed by a 200 must produce a text chunk, not an error.
func TestOpenAICompatStreamRetriesGatewayErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"upstream overloaded"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"recovered\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	// "test-retry-kind" is unregistered: ResolveProviderEndpoint falls back
	// to env, then to the passed defaults. Point the default at the test
	// server so no env/config is needed.
	kind := "test-retry-kind"
	ch := make(chan bot.StreamChunk, 16)
	go runOpenAICompatStream(t.Context(), ch, kind, srv.URL, "test-model", nil, nil, "", "")
	var texts, errs []string
	timeout := time.After(90 * time.Second)
loop:
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				break loop
			}
			if m.Type == "error" {
				errs = append(errs, m.Content)
			} else if m.Type == "text" {
				texts = append(texts, m.Content)
			}
		case <-timeout:
			t.Fatal("timed out waiting for stream result")
		}
	}
	if len(errs) > 0 {
		t.Fatalf("expected recovery after 503, got errors: %v", errs)
	}
	if !strings.Contains(strings.Join(texts, ""), "recovered") {
		t.Fatalf("expected recovered text, got %q", texts)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected >=2 upstream calls (fail + retry), got %d", calls.Load())
	}
}
