package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunDecisionsCtx_TypedAnswers(t *testing.T) {
	var body map[string]any
	var path, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, auth = r.URL.Path, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"id":"gen-1","model":"m-1","answers":{
			"is_skill":{"type":"noul","noul":0.96},
			"action":{"type":"choice","choice":"enhance","confidence":0.67}},
			"usage":{"input_tokens":476,"cost":0.00002}}`))
	}))
	defer srv.Close()

	res, err := RunDecisionsCtx(context.Background(), DecisionsConfig{
		BaseURL: srv.URL,
		APIKey:  "sk-or-test\r\n",
		Model:   "pplx-decider-flash-v1.2",
	}, map[string]string{"article": "body"}, map[string]DecisionQuestion{
		"is_skill": {Type: "noul", Instructions: "is it a skill?", Criteria: map[string]string{"true": "yes", "false": "no"}},
	})
	if err != nil {
		t.Fatalf("RunDecisionsCtx: %v", err)
	}
	if path != "/alpha/decisions" {
		t.Fatalf("path: %q", path)
	}
	if auth != "Bearer sk-or-test" {
		t.Fatalf("auth not trimmed: %q", auth)
	}
	if body["model"] != "pplx-decider-flash-v1.2" {
		t.Fatalf("model: %v", body["model"])
	}
	if _, ok := body["state"]; !ok {
		t.Fatalf("state missing")
	}
	ans, err := res.Answer("is_skill")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	p, err := ans.NoulProbability()
	if err != nil || p != 0.96 {
		t.Fatalf("noul=%v err=%v", p, err)
	}
	if res.InputTokens != 476 || res.CostUSD != 0.00002 {
		t.Fatalf("usage: %+v", res)
	}
}

// The API documents that a missing key or an unexpected type is an error, not
// a default. Getting that wrong would read as "not a skill".
func TestDecisionAnswer_RejectsWrongShape(t *testing.T) {
	res := DecisionsResult{Answers: map[string]DecisionAnswer{
		"is_skill": {Type: "choice", Choice: "enhance"},
	}}
	ans, err := res.Answer("is_skill")
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if _, err := ans.NoulProbability(); err == nil {
		t.Fatalf("choice answer must not yield a probability")
	}
	if _, err := res.Answer("missing"); err == nil {
		t.Fatalf("missing answer must error")
	}
	if _, err := RunDecisionsCtx(context.Background(), DecisionsConfig{Model: "m"}, nil, nil); err == nil {
		t.Fatalf("missing key must error")
	}
}

func TestRunDecisionsCtx_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"prompt too long"}`))
	}))
	defer srv.Close()
	_, err := RunDecisionsCtx(context.Background(), DecisionsConfig{BaseURL: srv.URL, APIKey: "k", Model: "m"}, nil, nil)
	if err == nil {
		t.Fatalf("want error")
	}
}
