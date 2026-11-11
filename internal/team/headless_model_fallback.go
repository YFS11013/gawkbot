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

// headlessModelFallbackEnv names the model(s) a turn retries on when the
// primary keeps failing: a comma-separated ring, tried left to right, once
// each. Unset means no fallback: the queue behaves exactly as before.
//
// The ring exists because free pools throttle per MODEL, not per key: when
// longcat is 429 the same key still answers on iceman and beam (probe,
// 2026-11-11). A single fallback therefore turns one throttled model into a
// dead office the moment the fallback shares the throttle. Each model is tried
// at most once — a ring that loops forever would spend the whole budget
// cycling through models that are all refusing.
const headlessModelFallbackEnv = "WUPHF_HEADLESS_MODEL_FALLBACK"

// headlessModelFallbackRing returns the configured fallback models in order.
func headlessModelFallbackRing() []string {
	raw := config.Getenv(headlessModelFallbackEnv)
	var ring []string
	seen := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		model := strings.TrimSpace(part)
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if seen[key] {
			continue
		}
		seen[key] = true
		ring = append(ring, model)
	}
	return ring
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

// headlessTurnFallbackModel decides which model this failed turn retries on,
// and the depth to stamp on the retry. It reads the turn's own record, so the
// decision survives the queue cycle that re-enqueues the turn. The model the
// turn is currently pinned to is skipped: retrying a turn on the model that
// just refused it teaches nothing.
func headlessTurnFallbackModel(turn headlessCodexTurn) (string, int) {
	ring := headlessModelFallbackRing()
	current := strings.TrimSpace(turn.ModelOverride)
	for i := turn.FallbackDepth; i < len(ring); i++ {
		if current != "" && strings.EqualFold(ring[i], current) {
			continue
		}
		return ring[i], i + 1
	}
	return "", 0
}

// headlessTurnHasFallback reports whether any model is left in the ring for
// this turn. Callers need the predicate before they have a retry to build.
func headlessTurnHasFallback(turn headlessCodexTurn) bool {
	model, _ := headlessTurnFallbackModel(turn)
	return model != ""
}
