package team

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// When a judge endpoint is configured, the verdict must leave over plain HTTP
// instead of starting a CLI agent with repo tools.
func TestAskIsSkill_UsesJudgeHTTP(t *testing.T) {
	var hits atomic.Int64
	var sawArticle atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Tools json.RawMessage `json:"tools"`
		}
		_ = json.Unmarshal(raw, &body)
		if len(body.Messages) == 0 || !strings.Contains(body.Messages[len(body.Messages)-1].Content, "UNIQUE-ARTICLE-BODY") {
			t.Errorf("article missing from prompt: %s", raw)
		}
		if len(body.Tools) > 0 {
			t.Errorf("tools must not be sent: %s", body.Tools)
		}
		sawArticle.Store(true)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"is_skill\": false}"}}]}`))
	}))
	defer srv.Close()

	t.Setenv("WUPHF_SKILL_JUDGE_BASE_URL", srv.URL)
	t.Setenv("WUPHF_SKILL_JUDGE_MODEL", "judge-model")
	t.Setenv("WUPHF_SKILL_JUDGE_API_KEY", "sk-judge")

	p := NewDefaultLLMProvider("")
	isSkill, _, _, _, err := p.AskIsSkill(context.Background(), "team/x.md", "UNIQUE-ARTICLE-BODY", "")
	if err != nil {
		t.Fatalf("AskIsSkill: %v", err)
	}
	if isSkill {
		t.Fatalf("want is_skill=false")
	}
	if hits.Load() != 1 || !sawArticle.Load() {
		t.Fatalf("judge endpoint not used: hits=%d", hits.Load())
	}
}

// Unreadable judge answers still surface as errors, never as a cached "no".
func TestAskIsSkill_JudgeHTTPGarbageIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"I think it is not a skill."}}]}`))
	}))
	defer srv.Close()
	t.Setenv("WUPHF_SKILL_JUDGE_BASE_URL", srv.URL)
	t.Setenv("WUPHF_SKILL_JUDGE_MODEL", "judge-model")

	if _, _, _, _, err := NewDefaultLLMProvider("").AskIsSkill(context.Background(), "team/x.md", "body", ""); err == nil {
		t.Fatalf("garbage answer must be an error so the article is retried")
	}
}

func TestSkillJudgeHTTPConfig_EmptyWithoutEnv(t *testing.T) {
	t.Setenv("WUPHF_SKILL_JUDGE_BASE_URL", "")
	t.Setenv("WUPHF_SKILL_JUDGE_URL", "")
	t.Setenv("WUPHF_SKILL_JUDGE_MODEL", "")
	t.Setenv("WUPHF_SKILL_JUDGE_PROVIDER", "")
	if cfg := skillJudgeHTTPConfig(); cfg.Valid() {
		t.Fatalf("want invalid config, got %+v", cfg)
	}
}
