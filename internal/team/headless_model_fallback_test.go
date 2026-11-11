package team

import (
	"context"
	"testing"
)

// The fallback is a ring: one attempt per configured model, taken only after
// the same-model reconnects are spent, and each model at most once. These
// tests pin the decision function the queue reads; the wiring itself is
// exercised by the queue-level tests and the live office.

func TestHeadlessTurnFallbackModel(t *testing.T) {
	t.Setenv(headlessModelFallbackEnv, "bear/deepseek-chat")

	if got, depth := headlessTurnFallbackModel(headlessCodexTurn{}); got != "bear/deepseek-chat" || depth != 1 {
		t.Fatalf("plain turn = %q/%d, want the configured fallback at depth 1", got, depth)
	}
	// A turn already running on the only configured model has nowhere to go.
	already := headlessCodexTurn{ModelOverride: "bear/deepseek-chat", FallbackDepth: 1}
	if got, _ := headlessTurnFallbackModel(already); got != "" {
		t.Fatalf("spent ring = %q, want no further fallback", got)
	}
	t.Setenv(headlessModelFallbackEnv, "  ")
	if got, _ := headlessTurnFallbackModel(headlessCodexTurn{}); got != "" {
		t.Fatalf("unset fallback = %q, want empty (retry disabled)", got)
	}
}

// A free-model office needs the ring: quotas throttle per model, so the second
// throttled model must not end the turn while a third still answers.
func TestHeadlessTurnFallbackRingAdvancesPastSpentModels(t *testing.T) {
	t.Setenv(headlessModelFallbackEnv, " opencode-go/iceman-free , opencode-go/beam-free ,, opencode-go/ICEMAN-free ")

	if got := headlessModelFallbackRing(); len(got) != 2 {
		t.Fatalf("ring = %v, want two deduped models", got)
	}

	first, depth := headlessTurnFallbackModel(headlessCodexTurn{})
	if first != "opencode-go/iceman-free" || depth != 1 {
		t.Fatalf("first fallback = %q/%d", first, depth)
	}
	second, depth2 := headlessTurnFallbackModel(headlessCodexTurn{ModelOverride: first, FallbackDepth: depth})
	if second != "opencode-go/beam-free" || depth2 != 2 {
		t.Fatalf("second fallback = %q/%d", second, depth2)
	}
	if got, _ := headlessTurnFallbackModel(headlessCodexTurn{ModelOverride: second, FallbackDepth: depth2}); got != "" {
		t.Fatalf("spent ring = %q, want empty", got)
	}

	// The model the turn is pinned to is skipped, not retried: a model that
	// just refused this turn will refuse it again.
	skip, skipDepth := headlessTurnFallbackModel(headlessCodexTurn{ModelOverride: "opencode-go/iceman-free", FallbackDepth: 0})
	if skip != "opencode-go/beam-free" || skipDepth != 2 {
		t.Fatalf("skip-current = %q/%d, want beam at depth 2", skip, skipDepth)
	}
}

func TestHeadlessModelOverrideRoundTrip(t *testing.T) {
	ctx := withHeadlessModelOverride(context.Background(), " openrouter/backup ")
	if got := headlessModelOverride(ctx); got != "openrouter/backup" {
		t.Fatalf("override = %q, want trimmed model", got)
	}
	if got := headlessModelOverride(context.Background()); got != "" {
		t.Fatalf("carrier-less ctx = %q, want empty", got)
	}
	if got := headlessModelOverride(nil); got != "" {
		t.Fatalf("nil ctx = %q, want empty", got)
	}
	// An empty model must not create a carrier that later reads as "set".
	if withHeadlessModelOverride(context.Background(), "  ") != nil {
		// ctx identity check: no-op value must return the same ctx
		base := context.Background()
		if withHeadlessModelOverride(base, "") != base {
			t.Fatal("empty override must return ctx unchanged")
		}
	}
}

func TestCodexModelForBotHonorsRetryOverride(t *testing.T) {
	ctx := withHeadlessModelOverride(context.Background(), "openrouter/backup-model")
	if got := (&Launcher{}).codexModelForBot(ctx, "cos"); got != "openrouter/backup-model" {
		t.Fatalf("codexModelForBot = %q, want the retry override", got)
	}
}
