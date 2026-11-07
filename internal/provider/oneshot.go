package provider

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/nex-crm/wuphf/internal/config"
)

// RunConfiguredOneShot runs a single-shot generation using the active LLM
// provider's OneShot implementation. Providers without a one-shot path
// (Capabilities.SupportsOneShot == false) and unregistered kinds fall back
// to Opencode (stdin prompt, never argv — Windows CreateProcess 32,767 cap,
// issue #1240). Claude was the previous fallback but its --append-system-prompt
// argv path overflows on Windows and its one-shot auth needs an API key the
// subscription login does not provide.
func RunConfiguredOneShot(systemPrompt, prompt, cwd string) (string, error) {
	return RunConfiguredOneShotCtx(context.Background(), systemPrompt, prompt, cwd)
}

// RunConfiguredOneShotCtx is like RunConfiguredOneShot, but cancellation is
// propagated into providers that expose a context-aware one-shot hook.
func RunConfiguredOneShotCtx(ctx context.Context, systemPrompt, prompt, cwd string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	kind := config.ResolveLLMProvider("")
	if e := Lookup(kind); e != nil && e.Capabilities.SupportsOneShot {
		if e.OneShotCtx != nil {
			return e.OneShotCtx(ctx, systemPrompt, prompt, cwd)
		}
		if e.OneShot != nil {
			return runLegacyOneShotCtx(ctx, e.OneShot, systemPrompt, prompt, cwd)
		}
	}
	// The active provider has no one-shot of its own. When its HTTP endpoint is
	// configured, ask it directly: the alternative below starts the opencode
	// coding agent, which answers a 20-token classification with 8-33s of agent
	// turn and 11k-25k input tokens, and sometimes with tool calls and no text.
	// A configured HTTP provider never had repo tools, so there is no context
	// to lose by not spawning one — the caller gets the model they selected.
	if cfg := OneShotHTTPConfigFor(kind); cfg.Valid() {
		return RunOneShotHTTPCtx(ctx, cfg, systemPrompt, prompt)
	}
	oneShotFallbackOnce.Do(func() {
		envKind := strings.ToUpper(strings.ReplaceAll(kind, "-", "_"))
		slog.Warn("provider: no one-shot for this kind, using the opencode CLI",
			"kind", kind,
			"hint", "set WUPHF_"+envKind+"_BASE_URL and WUPHF_"+envKind+"_MODEL to answer over HTTP instead")
	})
	return RunOpencodeOneShotCtx(ctx, systemPrompt, prompt, cwd)
}

// oneShotFallbackOnce keeps the warning to one line per process: the fallback
// is a standing configuration fact, not a per-call event.
var oneShotFallbackOnce sync.Once

// runLegacyOneShotCtx unblocks the caller on context cancellation, but the
// wrapped legacy fn and any subprocess it spawned continue until natural
// completion. All currently registered providers implement OneShotCtx, so this
// branch is effectively unreachable and only avoids a regression if a legacy
// provider is reintroduced.
func runLegacyOneShotCtx(ctx context.Context, fn func(systemPrompt, prompt, cwd string) (string, error), systemPrompt, prompt, cwd string) (string, error) {
	if ctx.Done() == nil {
		return fn(systemPrompt, prompt, cwd)
	}

	type result struct {
		text string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		t, e := fn(systemPrompt, prompt, cwd)
		ch <- result{t, e}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		return r.text, r.err
	}
}
