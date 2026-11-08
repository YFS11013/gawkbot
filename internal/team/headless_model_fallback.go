package team

import (
	"context"
	"strings"

	"github.com/nex-crm/wuphf/internal/config"
)

// Headless model fallback.
//
// Every office turn runs on one model (WUPHF_OPENCODE_MODEL for the Opencode
// runtime; the task/bot binding or Codex config for Codex). When that model's
// upstream is down, rate-limited, or — the case that actually bit us — the id
// itself is unknown, the turn dies and the office goes quiet until a human
// notices. The queue already re-runs a turn twice on the same model for
// connection drops; those retries are useless when the model is what is broken.
//
// This adds one attempt on a configured fallback model, taken after the
// same-model reconnects are exhausted, and only once per turn.

// headlessModelFallbackEnv names the model a turn retries on when the primary
// keeps failing. Unset means no fallback: the queue behaves exactly as before.
const headlessModelFallbackEnv = "WUPHF_HEADLESS_MODEL_FALLBACK"

// headlessModelFallback returns the configured fallback model, or "".
func headlessModelFallback() string {
	return strings.TrimSpace(config.Getenv(headlessModelFallbackEnv))
}

// headlessModelOverrideKey carries the model a turn must use. It is a context
// value, not an environment variable, because several bots run turns
// concurrently in this process: mutating the environment for one retry would
// reroute every other in-flight turn.
type headlessModelOverrideKey struct{}

// withHeadlessModelOverride returns a ctx whose turn must use model. An empty
// model leaves ctx untouched.
func withHeadlessModelOverride(ctx context.Context, model string) context.Context {
	model = strings.TrimSpace(model)
	if ctx == nil || model == "" {
		return ctx
	}
	return context.WithValue(ctx, headlessModelOverrideKey{}, model)
}

// headlessModelOverride returns the model this turn was pinned to, or "".
func headlessModelOverride(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if model, ok := ctx.Value(headlessModelOverrideKey{}).(string); ok {
		return strings.TrimSpace(model)
	}
	return ""
}

// headlessTurnFallbackModel decides whether this failed turn gets its one
// fallback attempt. It reads the turn's own record, so the decision survives
// the queue cycle that re-enqueues the turn.
func headlessTurnFallbackModel(turn headlessCodexTurn) string {
	if strings.TrimSpace(turn.ModelOverride) != "" {
		return "" // already the fallback attempt: no chain
	}
	return headlessModelFallback()
}
