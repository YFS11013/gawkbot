package team

import (
	"context"
	"testing"
)

// The fallback is one attempt on a different model, taken only after the
// same-model reconnects are spent, and never chained. These tests pin the
// decision function the queue reads; the wiring itself is exercised by the
// live run recorded in docs/office/RUNTIME-API.md.

func TestHeadlessTurnFallbackModel(t *testing.T) {
	t.Setenv(headlessModelFallbackEnv, "bear/deepseek-chat")

	if got := headlessTurnFallbackModel(headlessCodexTurn{}); got != "bear/deepseek-chat" {
		t.Fatalf("plain turn = %q, want the configured fallback", got)
	}
	// A turn already running on the fallback must not chain to another model.
	already := headlessCodexTurn{ModelOverride: "bear/deepseek-chat"}
	if got := headlessTurnFallbackModel(already); got != "" {
		t.Fatalf("fallback turn = %q, want no second fallback", got)
	}
	t.Setenv(headlessModelFallbackEnv, "  ")
	if got := headlessTurnFallbackModel(headlessCodexTurn{}); got != "" {
		t.Fatalf("unset fallback = %q, want empty (retry disabled)", got)
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
