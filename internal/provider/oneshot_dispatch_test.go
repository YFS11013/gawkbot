package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// A provider without a one-shot (ollama, mlx, exo, openai-compat) must answer
// over its own HTTP endpoint when one is configured. The fallback it replaces
// is the opencode coding agent: 8-33s and 11k-25k input tokens for a 20-token
// question, with repo tools the caller never asked for.
func TestRunConfiguredOneShotCtx_UsesHTTPWhenConfigured(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"http answer"}}]}`))
	}))
	defer srv.Close()

	t.Setenv("WUPHF_LLM_PROVIDER", "ollama")
	t.Setenv("WUPHF_OLLAMA_BASE_URL", srv.URL+"/v1")
	t.Setenv("WUPHF_OLLAMA_MODEL", "qwen-test")

	out, err := RunConfiguredOneShotCtx(context.Background(), "sys", "prompt", "")
	if err != nil {
		t.Fatalf("one-shot: %v", err)
	}
	if out != "http answer" {
		t.Fatalf("got %q, want the HTTP answer", out)
	}
	if hits.Load() != 1 {
		t.Fatalf("endpoint hits = %d, want 1", hits.Load())
	}
}

// A locally installed provider answers at its own address without any env
// beyond the provider selection. Before this, ollama/mlx-lm/exo resolved to an
// invalid one-shot config and the caller silently got the opencode agent.
func TestOneShotHTTPConfigFor_DefaultsToTheProviderEndpoint(t *testing.T) {
	t.Setenv("WUPHF_LLM_PROVIDER", "ollama")
	t.Setenv("WUPHF_OLLAMA_BASE_URL", "")
	t.Setenv("WUPHF_OLLAMA_MODEL", "")

	cfg := OneShotHTTPConfigFor("ollama")
	if !cfg.Valid() {
		t.Fatal("ollama must resolve to its default endpoint")
	}
	if cfg.BaseURL != defaultOllamaBaseURL || cfg.Model != defaultOllamaModel {
		t.Fatalf("got %s / %s, want %s / %s", cfg.BaseURL, cfg.Model, defaultOllamaBaseURL, defaultOllamaModel)
	}

	// An explicit endpoint still wins over the default.
	t.Setenv("WUPHF_OLLAMA_BASE_URL", "http://127.0.0.1:9999/v1")
	t.Setenv("WUPHF_OLLAMA_MODEL", "chosen")
	if got := OneShotHTTPConfigFor("ollama"); got.BaseURL != "http://127.0.0.1:9999/v1" || got.Model != "chosen" {
		t.Fatalf("env must override the default, got %s / %s", got.BaseURL, got.Model)
	}

	// A kind with no StreamFn has nothing to default to.
	if OneShotHTTPConfigFor("no-such-provider").Valid() {
		t.Fatal("an unknown kind must not invent an endpoint")
	}
}
