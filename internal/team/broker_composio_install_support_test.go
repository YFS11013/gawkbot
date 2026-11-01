package team

import "testing"

// A missing CLI on a platform that cannot install one must never enter
// "installing": that phase ends in install_failed, whose only remedy is "try
// again", so the pair loops forever. OFFICE-3043 saw the cost — every broker
// start replayed the install, failed, and put a blocking dialog in front of
// someone who had nothing to install.
func TestComposioSigninUnsupportedPlatformSkipsInstall(t *testing.T) {
	if composioAutoInstallSupported {
		t.Skip("this platform has an installer to run")
	}
	b := newComposioSigninBroker(t, t.TempDir()) // empty PATH: no CLI anywhere

	state := b.startComposioSignin("human", composioSigninStartOptions{Auto: false})
	if state.Status != composioSigninStatusCLIMissing {
		t.Fatalf("status = %q, want %q: installing without an installer can only fail", state.Status, composioSigninStatusCLIMissing)
	}
	if state.InstallCommand == "" {
		t.Fatal("cli_missing must carry the manual command")
	}
	if state.Reason != composioInstallUnsupportedMessage {
		t.Fatalf("reason = %q, want the unsupported-setup sentence", state.Reason)
	}
	if state.AutoInstall {
		t.Fatal("auto_install must be false where no installer exists")
	}

	// The latch: the next automatic connect offers the button instead of
	// replaying this branch.
	next := b.startComposioSignin("human", composioSigninStartOptions{Auto: true})
	if next.Status != composioSigninStatusIdle {
		t.Fatalf("automatic start after the unsupported path = %q, want idle", next.Status)
	}
}

// The prompt that asks permission to install must say whether installing is
// possible here, or the UI offers a button that cannot succeed.
func TestComposioSigninInstallRequiredAdvertisesCapability(t *testing.T) {
	b := newComposioSigninBroker(t, t.TempDir())

	state := b.startComposioSignin("human", composioSigninStartOptions{Auto: true})
	if state.Status != composioSigninStatusInstallRequired {
		t.Fatalf("status = %q, want %q", state.Status, composioSigninStatusInstallRequired)
	}
	if state.AutoInstall != composioAutoInstallSupported {
		t.Fatalf("auto_install = %v, want %v on this platform", state.AutoInstall, composioAutoInstallSupported)
	}
	if state.InstallCommand == "" {
		t.Fatal("install_required must carry the manual command")
	}
}
