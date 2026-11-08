package team

import (
	"testing"

	"github.com/nex-crm/wuphf/internal/onboarding"
)

// TestIsDeterministicPhase2CEODM_GatesCEODMDuringPhase2 locks in the
// regression that an LLM bot must not fire while the deterministic
// Phase 2 onboarding conversation is driving the CEO DM. Prior to the
// gate, deliverMessageNotification enqueued a Claude turn for every
// human reply to a ceo_form_field / ceo_chip_row / ceo_team_trim card,
// which produced a sticky "CEO is typing…" indicator and stalled the
// scripted flow behind a hallucinated LLM response.
func TestIsDeterministicPhase2CEODM_GatesCEODMDuringPhase2(t *testing.T) {
	cases := []struct {
		name    string
		channel string
		phase   string
		onboard bool
		want    bool
	}{
		{"CEO DM mid-phase-blueprint", "cos__human", onboarding.PhaseBlueprint, false, true},
		{"CEO DM mid-phase-team", "cos__human", onboarding.PhaseTeam, false, true},
		{"CEO DM mid-phase-greet", "human__cos", onboarding.PhaseGreet, false, true},
		{"reserved CEO DM slug", onboarding.CEOOnboardingDMSlug, onboarding.PhaseBlueprint, false, true},
		{"CEO DM after onboarding complete", "cos__human", onboarding.PhaseComplete, true, false},
		{"CEO DM in LLM-backed draft phase", "cos__human", onboarding.PhaseDraft, false, false},
		{"non-CEO DM during Phase 2", "engineer__human", onboarding.PhaseTeam, false, false},
		{"non-DM channel during Phase 2", "team", onboarding.PhaseTeam, false, false},
		{"empty channel during Phase 2", "", onboarding.PhaseTeam, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			// Point WUPHF_RUNTIME_HOME at the temp home instead of clearing it:
			// clearing left RuntimeHomeDir to os.UserHomeDir, which ignores HOME
			// on Windows and read — then SAVED — this operator's real
			// ~/.wuphf/onboarded.json (2026-10-30: phase/completed_at clobbered
			// by this test on a Windows dev box).
			t.Setenv("WUPHF_RUNTIME_HOME", home)

			s, err := onboarding.Load()
			if err != nil {
				t.Fatalf("onboarding.Load: %v", err)
			}
			s.Phase = tc.phase
			if tc.onboard {
				s.CompletedAt = "2026-05-28T00:00:00Z"
			}
			if err := onboarding.Save(s); err != nil {
				t.Fatalf("onboarding.Save: %v", err)
			}

			got := isDeterministicPhase2CEODM(tc.channel)
			if got != tc.want {
				t.Fatalf("isDeterministicPhase2CEODM(%q) with phase=%q onboarded=%v = %v, want %v",
					tc.channel, tc.phase, tc.onboard, got, tc.want)
			}
		})
	}
}

// TestSendTaskUpdateReportsLifecycleGate pins the return value the watchdog now
// reads: a wake refused by the lifecycle gate must report false, else the
// scheduler cannot tell "owner was woken" from "nothing happened".
func TestSendTaskUpdateReportsLifecycleGate(t *testing.T) {
	b := newTestBroker(t)
	l := &Launcher{broker: b}
	refused := l.sendTaskUpdate(
		notificationTarget{Slug: "eng"},
		officeActionLog{Kind: "watchdog_alert", Channel: "general", Actor: "watchdog"},
		teamTask{ID: "t1", Channel: "general", Owner: "eng", LifecycleState: LifecycleStateDrafting},
		"body",
	)
	if refused {
		t.Fatal("a lifecycle gate that drops the wake must report false")
	}
}

// TestDeliverTaskNotificationReportsDispatchedWake is the positive half of the
// same contract: when the owner's wake really is queued, the watchdog must see
// true and stay quiet (no escalation message on top of real work).
func TestDeliverTaskNotificationReportsDispatchedWake(t *testing.T) {
	b := newTestBroker(t)
	l := &Launcher{broker: b}
	ok := l.deliverTaskNotification(
		officeActionLog{Kind: "watchdog_alert", Channel: "general", Actor: "watchdog"},
		teamTask{ID: "t1", Channel: "general", Owner: "eng", LifecycleState: LifecycleStateRunning},
	)
	if !ok {
		t.Fatal("a queued owner wake must report true")
	}
	queued := 0
	l.headless.mu.Lock()
	for _, q := range l.headless.queues {
		queued += len(q)
	}
	l.headless.mu.Unlock()
	if queued == 0 {
		t.Fatal("reporting true without a queued turn would let the watchdog skip escalation silently")
	}
}
