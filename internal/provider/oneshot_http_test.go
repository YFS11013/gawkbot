package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunOneShotHTTPCtx_SendsPlainCompletion(t *testing.T) {
	var gotPath, gotAuth string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"is_skill\": false}"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	out, err := RunOneShotHTTPCtx(context.Background(), OneShotHTTPConfig{
		BaseURL: srv.URL + "/v1",
		APIKey:  "sk-test",
		Model:   "deepseek-chat",
	}, "SYSTEM", "USER")
	if err != nil {
		t.Fatalf("RunOneShotHTTPCtx: %v", err)
	}
	if out != `{"is_skill": false}` {
		t.Fatalf("content: %q", out)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path: %q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth: %q", gotAuth)
	}
	// The whole point versus the CLI path: no tools, no streaming, no
	// temperature wobble.
	if _, ok := body["tools"]; ok {
		t.Fatalf("one-shot must not send tools: %v", body["tools"])
	}
	if body["stream"] != false {
		t.Fatalf("stream must be false, got %v", body["stream"])
	}
	if body["temperature"] != float64(0) {
		t.Fatalf("temperature must be 0, got %v", body["temperature"])
	}
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("want system+user messages, got %v", msgs)
	}
	if body["model"] != "deepseek-chat" {
		t.Fatalf("model: %v", body["model"])
	}
}

func TestRunOneShotHTTPCtx_EndpointAlreadyComplete(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	if _, err := RunOneShotHTTPCtx(context.Background(), OneShotHTTPConfig{
		BaseURL: srv.URL + "/v1/chat/completions",
		Model:   "m",
	}, "", "hi"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path doubled: %q", gotPath)
	}
}

func TestRunOneShotHTTPCtx_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	_, err := RunOneShotHTTPCtx(context.Background(), OneShotHTTPConfig{BaseURL: srv.URL, Model: "m"}, "", "x")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want 401 error, got %v", err)
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer empty.Close()
	_, err = RunOneShotHTTPCtx(context.Background(), OneShotHTTPConfig{BaseURL: empty.URL, Model: "m"}, "", "x")
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Fatalf("want no-choices error, got %v", err)
	}

	if _, err := RunOneShotHTTPCtx(context.Background(), OneShotHTTPConfig{Model: "m"}, "", "x"); err == nil {
		t.Fatalf("invalid config must error")
	}
}
