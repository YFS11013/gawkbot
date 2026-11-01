//go:build windows

package team

import (
	"context"
	"errors"
)

// composioInstallCommand on Windows is the cross-platform npm install: the
// official `curl | bash` script has no Windows equivalent, and npm is Composio's
// recommended install path off Unix. Surfaced to the user as the manual
// fallback (auto-install is not attempted on Windows).
const composioInstallCommand = "npm install -g @composio/cli"

// composioAutoInstallSupported is false here on purpose: the Windows build has
// no installer to run, so the sign-in flow must not enter the "installing"
// phase at all. Entering it could only end in install_failed, whose remedy is
// "try again" — a loop that cannot terminate. OFFICE-3043 observed the cost:
// every broker start replayed the install, failed, and put a blocking dialog
// in front of a person who has nothing to install.
//
// How to restore it: put a real installer in defaultComposioInstaller (below)
// and flip this to true. Nothing else reads it.
const composioAutoInstallSupported = false

// defaultComposioInstaller is a no-op on Windows: the official installer is a
// `curl | bash` pipeline with no Windows equivalent, so auto-install reports
// not-supported and the sign-in flow falls back to surfacing the manual install
// command (cli_missing). The returned error is advisory — composioSigninAutoInstall
// treats the CLI's presence on PATH, not this exit, as the source of truth.
func defaultComposioInstaller(_ context.Context) error {
	return errors.New("automatic Composio CLI install is not supported on Windows; run the install command manually")
}
