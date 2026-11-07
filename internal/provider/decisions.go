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
)

// Decisions API: OpenRouter's typed-decision endpoint.
//
//	POST https://openrouter.ai/api/alpha/decisions
//
// A decision model reads a `state` (text or JSON) and answers typed questions.
// It returns a probability for a yes/no question (`noul`), a chosen option
// (`choice`), or an ordinal score — never free text. That makes it a cheap
// gate in front of a text model: on the skill-judge prompt a decision model
// answers in ~0.9s for ~1.7k input tokens, versus 7-120s and 100-800 output
// tokens for a reasoning chat model, and nothing has to be JSON-parsed out of
// prose afterwards.
//
// Endpoint shape differs from /chat/completions, hence a separate transport.
// The catalog is `GET /api/v1/models?output_modalities=decisions`.
type DecisionsConfig struct {
	BaseURL string // defaults to https://openrouter.ai/api
	APIKey  string
	Model   string
}

// Valid reports whether a decision model can be addressed.
func (c DecisionsConfig) Valid() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.Model) != ""
}

// DecisionQuestion is one typed question. Type is "noul" (criteria true/false),
// "choice" (criteria maps option -> description), or "score" (criteria is an
// ordered list, lowest first).
type DecisionQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

// DecisionAnswer is one typed answer. Only the fields implied by Type carry
// meaning; the API documentation is explicit that a missing key or an
// unexpected type is an error, not a default.
type DecisionAnswer struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        string             `json:"choice,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
}

// NoulProbability returns the yes-probability of a noul answer, or an error
// when the model answered a different question type.
func (a DecisionAnswer) NoulProbability() (float64, error) {
	if a.Type != "noul" {
		return 0, fmt.Errorf("provider: decisions: expected noul answer, got %q", a.Type)
	}
	if a.Noul == nil {
		return 0, errors.New("provider: decisions: noul answer has no probability")
	}
	return *a.Noul, nil
}

// DecisionsResult is the decoded response plus its billing/usage metadata.
type DecisionsResult struct {
	ID          string
	Model       string
	Answers     map[string]DecisionAnswer
	InputTokens int
	CostUSD     float64
}

// Answer returns the answer for a question ID.
func (r DecisionsResult) Answer(id string) (DecisionAnswer, error) {
	a, ok := r.Answers[id]
	if !ok {
		return DecisionAnswer{}, fmt.Errorf("provider: decisions: response has no answer for %q", id)
	}
	return a, nil
}

// RunDecisionsCtx posts one typed-decision request.
func RunDecisionsCtx(ctx context.Context, cfg DecisionsConfig, state any, questions map[string]DecisionQuestion) (DecisionsResult, error) {
	if !cfg.Valid() {
		return DecisionsResult{}, errors.New("provider: decisions needs an api key and a model")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = "https://openrouter.ai/api"
	}
	endpoint := strings.TrimSuffix(base, "/") + "/alpha/decisions"

	payload, err := json.Marshal(struct {
		Model     string                      `json:"model"`
		State     any                         `json:"state"`
		Questions map[string]DecisionQuestion `json:"questions"`
	}{Model: cfg.Model, State: state, Questions: questions})
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("provider: decisions: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("provider: decisions: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))

	resp, err := oneShotHTTPClient().Do(req)
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("provider: decisions: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DecisionsResult{}, fmt.Errorf("provider: decisions: HTTP %d: %s", resp.StatusCode, clipRaw(raw, 300))
	}
	var decoded struct {
		ID      string                    `json:"id"`
		Model   string                    `json:"model"`
		Answers map[string]DecisionAnswer `json:"answers"`
		Usage   struct {
			InputTokens int     `json:"input_tokens"`
			Cost        float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return DecisionsResult{}, fmt.Errorf("provider: decisions: decode: %w (%s)", err, clipRaw(raw, 200))
	}
	if len(decoded.Answers) == 0 {
		return DecisionsResult{}, fmt.Errorf("provider: decisions: no answers (%s)", clipRaw(raw, 200))
	}
	return DecisionsResult{
		ID:          decoded.ID,
		Model:       decoded.Model,
		Answers:     decoded.Answers,
		InputTokens: decoded.Usage.InputTokens,
		CostUSD:     decoded.Usage.Cost,
	}, nil
}
