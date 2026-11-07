package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// gateServer answers the decisions endpoint with a fixed noul probability and
// reports how many text-path prompts the scanner sent afterwards.
func gateServer(t *testing.T, pYes float64) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var textCalls atomic.Int64
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	mux.HandleFunc("/alpha/decisions", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"g","model":"m","answers":{"is_skill":{"type":"noul","noul":` +
			strings.TrimSpace(jsonNumber(pYes)) + `}},"usage":{"input_tokens":10,"cost":0.0001}}`))
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		textCalls.Add(1)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"is_skill\": false}"}}]}`))
	})
	return srv, &textCalls
}

func jsonNumber(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func withGate(t *testing.T, srvURL, chatURL string) {
	t.Helper()
	t.Setenv("WUPHF_SKILL_GATE_MODEL", "pplx-decider-flash-v1.2")
	t.Setenv("WUPHF_SKILL_GATE_API_KEY", "sk-or-test")
	t.Setenv("WUPHF_SKILL_GATE_BASE_URL", srvURL)
	t.Setenv("WUPHF_SKILL_JUDGE_BASE_URL", chatURL)
	t.Setenv("WUPHF_SKILL_JUDGE_MODEL", "text-judge")
	t.Setenv("WUPHF_SKILL_JUDGE_API_KEY", "k")
}

func TestSkillGate_RejectsWithoutTextCall(t *testing.T) {
	srv, textCalls := gateServer(t, 0.05)
	defer srv.Close()
	withGate(t, srv.URL, srv.URL)

	isSkill, _, _, _, err := NewDefaultLLMProvider("").AskIsSkill(context.Background(), "team/x.md", "status note", "")
	if err != nil {
		t.Fatalf("AskIsSkill: %v", err)
	}
	if isSkill {
		t.Fatalf("gate said no")
	}
	if textCalls.Load() != 0 {
		t.Fatalf("rejected article must not reach the text model, got %d calls", textCalls.Load())
	}
}

func TestSkillGate_AcceptsThenTextAuthors(t *testing.T) {
	srv, textCalls := gateServer(t, 0.95)
	defer srv.Close()
	withGate(t, srv.URL, srv.URL)

	if _, _, _, _, err := NewDefaultLLMProvider("").AskIsSkill(context.Background(), "team/x.md", "procedure", ""); err != nil {
		t.Fatalf("AskIsSkill: %v", err)
	}
	if textCalls.Load() != 1 {
		t.Fatalf("accepted article must reach the text model once, got %d", textCalls.Load())
	}
}

// 0.3 <= p < 0.5 is "no verdict": error, so the article stays out of the cache
// and is asked again next pass instead of being filed as "not a skill".
func TestSkillGate_UncertainIsError(t *testing.T) {
	srv, textCalls := gateServer(t, 0.40)
	defer srv.Close()
	withGate(t, srv.URL, srv.URL)

	_, _, _, _, err := NewDefaultLLMProvider("").AskIsSkill(context.Background(), "team/x.md", "ambiguous note", "")
	if err == nil {
		t.Fatalf("uncertain gate must be an error")
	}
	if textCalls.Load() != 0 {
		t.Fatalf("uncertain gate must not spend a text call, got %d", textCalls.Load())
	}
}

func TestSkillGate_DisabledWithoutModel(t *testing.T) {
	srv, textCalls := gateServer(t, 0.05)
	defer srv.Close()
	withGate(t, srv.URL, srv.URL)
	t.Setenv("WUPHF_SKILL_GATE_MODEL", "")

	if _, _, _, _, err := NewDefaultLLMProvider("").AskIsSkill(context.Background(), "team/x.md", "body", ""); err != nil {
		t.Fatalf("AskIsSkill: %v", err)
	}
	if textCalls.Load() != 1 {
		t.Fatalf("gate disabled must fall through to the text model, got %d", textCalls.Load())
	}
}

// The daily model snapshot is generated boilerplate whose hash changes every
// day; a prefix match keeps it out without naming the date.
func TestScanSkipsGeneratedModelSnapshots(t *testing.T) {
	for _, rel := range []string{
		"team/opencode-models-2026-11-07.md",
		"team/opencode-models-2026-10-31.md",
	} {
		if !shouldSkipPath(rel) {
			t.Fatalf("%s must be skipped", rel)
		}
	}
	// A hand-written note that merely mentions the snapshots is still scanned.
	if shouldSkipPath("team/learnings/opencode-models-are-stale.md") {
		t.Fatalf("unrelated article must not be skipped")
	}
	if !shouldSkipPath("team/prep/packet.md") || !shouldSkipPath("team/skills/x.md") {
		t.Fatalf("existing directory skips must still apply")
	}
}
