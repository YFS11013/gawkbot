package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nex-crm/wuphf/internal/config"
)

// OneShotHTTPConfig points a single-shot completion at any OpenAI-compatible
// /chat/completions endpoint.
//
// Why this exists: RunConfiguredOneShotCtx dispatches on the *active* provider,
// and only the CLI agents (claude, codex, opencode) implement a one-shot. With
// an HTTP provider active, a one-shot call silently fell back to
// RunOpencodeOneShotCtx — i.e. asking a 20-token classification question
// started a coding agent with tools in the repo. Measured on the skill-judge
// prompt (2026-11-06): 8-33s and 11k-25k input tokens through the agent, versus
// 0.6-0.8s and ~1.6k tokens over plain HTTP. The agent also sometimes answered
// with tool calls and no text at all, which read as "no verdict".
//
// The endpoint is resolved by the caller (see provider.EndpointFor) so this
// helper stays a transport, not a policy holder.
type OneShotHTTPConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Valid reports whether the config can address a model.
func (c OneShotHTTPConfig) Valid() bool {
	return strings.TrimSpace(c.BaseURL) != "" && strings.TrimSpace(c.Model) != ""
}

// RunOneShotHTTPCtx sends one chat completion and returns the assistant text.
// No tools, no streaming: the caller wants an answer, not a conversation.
func RunOneShotHTTPCtx(ctx context.Context, cfg OneShotHTTPConfig, systemPrompt, prompt string) (string, error) {
	if !cfg.Valid() {
		return "", errors.New("provider: one-shot http needs base_url and model")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Accept either a base URL (a server's listening address) or the complete
	// endpoint; appending blindly turns the latter into /chat/completions
	// /chat/completions.
	endpoint := strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint = normalizeOpenAICompatEndpoint(cfg.BaseURL)
	}
	messages := make([]openaiMessage, 0, 2)
	if s := strings.TrimSpace(systemPrompt); s != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: s})
	}
	messages = append(messages, openaiMessage{Role: "user", Content: prompt})
	// Temperature 0: the verdict must not wobble between passes, and the
	// verdict cache persists the answer until the article changes.
	payload, err := json.Marshal(struct {
		Model       string          `json:"model"`
		Messages    []openaiMessage `json:"messages"`
		Stream      bool            `json:"stream"`
		Temperature float64         `json:"temperature"`
	}{Model: cfg.Model, Messages: messages, Stream: false, Temperature: 0})
	if err != nil {
		return "", fmt.Errorf("provider: marshal one-shot request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("provider: build one-shot request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := oneShotHTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("provider: one-shot %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return "", fmt.Errorf("provider: one-shot HTTP %d from %s: %s", resp.StatusCode, endpoint, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("provider: read one-shot response: %w", err)
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("provider: decode one-shot response: %w (%s)", err, clipRaw(raw, 200))
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("provider: one-shot response had no choices (%s)", clipRaw(raw, 200))
	}
	content := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("provider: one-shot returned empty content (finish_reason=%q)", decoded.Choices[0].FinishReason)
	}
	return content, nil
}

func oneShotHTTPClient() *http.Client {
	// Generous per-request timeout; callers wrap the context with their own
	// deadline (the skill scanner uses 30s).
	return &http.Client{Timeout: 120 * time.Second}
}

func clipRaw(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if max > 0 && len(s) > max {
		return s[:max] + "…"
	}
	return s
}

// OneShotHTTPConfigFor builds a one-shot config from a registered provider
// kind, reusing the same env > config > defaults chain the streaming path uses
// (WUPHF_<KIND>_BASE_URL / _MODEL / _API_KEY and ~/.wuphf/config.json
// provider_endpoints). Returns an invalid config when the kind has no endpoint,
// so callers can fall back instead of guessing.
func OneShotHTTPConfigFor(kind string) OneShotHTTPConfig {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return OneShotHTTPConfig{}
	}
	baseURL, model := config.ResolveProviderEndpoint(kind, "", "")
	return OneShotHTTPConfig{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  resolveOpenAICompatAPIKey(kind),
	}
}
