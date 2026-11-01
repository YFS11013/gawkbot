import { ExternalLink } from "lucide-react";

import { CommandRow } from "../../ui/CommandRow";
import type { SigninPhase } from "./useComposioSignin";

// The primary sign-in surface — one panel per flow phase. Shared by the
// first-run onboarding hero and by the connect-time recovery panel, so both
// render the same thing for the same broker state.
//
// Copy rule, applied to every sentence here: no status code, slug, request id,
// environment variable, or shell command. None of it is actionable by the
// person reading. The install command is the one technical artefact the flow
// still needs to surface, and it lives behind a disclosure addressed to
// whoever set the computer up — never inside a sentence.

export interface ComposioSigninPanelProps {
  phase: SigninPhase;
  authUrl: string;
  installCommand: string;
  /**
   * Whether this computer can run the one-time setup itself. False hides every
   * affordance that would start an install, because none of them can finish.
   */
  autoInstall?: boolean;
  starting: boolean;
  onStart: () => void;
  /** Label for the idle-state button: "Connect integrations", "Sign in again". */
  ctaLabel: string;
  /**
   * Stop a sign-in the user did not ask for. Passing it is what makes an
   * automatic sign-in acceptable rather than a surprise, so surfaces that
   * auto-start must supply it.
   */
  onCancel?: () => void;
  /** Answer the one-time-setup prompt with yes. Defaults to onStart. */
  onConfirmInstall?: () => void;
}

/** The install command, addressed to whoever can run it, out of the prose. */
function InstallCommandDetails({ command }: { command: string }) {
  if (!command) return null;
  return (
    <details className="composio-onb-install-details">
      <summary>For whoever set up this computer</summary>
      <CommandRow command={command} />
    </details>
  );
}

function CancelRow({ onCancel }: { onCancel?: () => void }) {
  if (!onCancel) return null;
  return (
    <div className="composio-onb-actions">
      <button type="button" className="btn btn-ghost btn-sm" onClick={onCancel}>
        Cancel
      </button>
    </div>
  );
}

export function ComposioSigninPanel({
  phase,
  authUrl,
  installCommand,
  autoInstall = true,
  starting,
  onStart,
  ctaLabel,
  onCancel,
  onConfirmInstall,
}: ComposioSigninPanelProps) {
  if (phase === "install_required") {
    // The destructive-action carve-out: connecting an app does not authorise
    // putting software on someone's computer, so this one asks.
    //
    // Where no installer exists (Windows), asking is pointless: the button
    // could only fail, and its failure screen offers "try again", which fails
    // the same way. Show the command instead and skip the question.
    if (!autoInstall) {
      return (
        <div className="composio-onb-panel" role="status">
          <p className="composio-onb-panel-title">Setup runs from a terminal</p>
          <p className="composio-onb-panel-note">
            Connecting apps needs a small helper, and this computer cannot
            install it by itself. Run the command below once, then come back
            and sign in.
          </p>
          <InstallCommandDetails command={installCommand} />
          {onCancel ? <CancelRow onCancel={onCancel} /> : null}
        </div>
      );
    }
    return (
      <div className="composio-onb-panel" role="status">
        <p className="composio-onb-panel-title">One-time setup needed</p>
        <p className="composio-onb-panel-note">
          Connecting apps needs a small helper installed on this computer. It
          takes about a minute, it runs only when you connect an app, and
          nothing else on this computer changes.
        </p>
        <div className="composio-onb-actions">
          <button
            type="button"
            className="btn btn-primary"
            onClick={onConfirmInstall ?? onStart}
            disabled={starting}
          >
            Set it up
          </button>
          {onCancel ? (
            <button
              type="button"
              className="btn btn-ghost"
              onClick={onCancel}
              disabled={starting}
            >
              Not now
            </button>
          ) : null}
        </div>
        <InstallCommandDetails command={installCommand} />
      </div>
    );
  }
  if (phase === "installing") {
    return (
      <div className="composio-onb-panel" role="status">
        <p className="composio-onb-panel-title">Setting up integrations…</p>
        <p className="composio-onb-panel-note">
          One-time setup. The sign-in page opens by itself as soon as it is
          ready.
        </p>
        <p className="composio-onb-wait">Working on it…</p>
        <CancelRow onCancel={onCancel} />
      </div>
    );
  }
  if (phase === "cli_missing") {
    // "Did not finish" describes an attempt that failed. Where no installer
    // exists, nothing was attempted — saying otherwise sends people looking
    // for a failure that never happened.
    return (
      <div className="composio-onb-panel" role="status">
        <p className="composio-onb-panel-title">
          {autoInstall ? "Setup did not finish" : "Setup runs from a terminal"}
        </p>
        <p className="composio-onb-panel-note">
          {autoInstall
            ? "The one-time setup for integrations could not finish on this computer. You can try again, or pass the step below to whoever set this computer up."
            : "This computer cannot install the helper by itself. Run the command below once, then try again."}
        </p>
        <InstallCommandDetails command={installCommand} />
        <div className="composio-onb-actions">
          <button
            type="button"
            className="btn btn-primary"
            onClick={onStart}
            disabled={starting}
          >
            Try again
          </button>
        </div>
      </div>
    );
  }
  if (phase === "awaiting_login") {
    return (
      <div className="composio-onb-panel" role="status">
        <p className="composio-onb-panel-title">
          Finish signing in in your browser
        </p>
        {authUrl ? (
          <>
            <p className="composio-onb-panel-note">
              A sign-in page should have opened in a new tab. If it did not,{" "}
              <a
                className="composio-onb-getkey"
                href={authUrl}
                target="_blank"
                rel="noopener noreferrer"
              >
                open the sign-in page
                <ExternalLink size={13} aria-hidden="true" />
              </a>{" "}
              or copy the link and open it on any device.
            </p>
            <CommandRow command={authUrl} />
          </>
        ) : (
          <p className="composio-onb-panel-note">
            Waiting for the sign-in page. If nothing opens, cancel and try
            again.
          </p>
        )}
        <p className="composio-onb-wait">Waiting for you to finish…</p>
        <CancelRow onCancel={onCancel} />
      </div>
    );
  }
  if (phase === "provisioning" || phase === "done") {
    return (
      <div className="composio-onb-panel" role="status">
        <p className="composio-onb-panel-title">Connecting your account…</p>
        <p className="composio-onb-panel-note">
          Saving your credentials to this workspace.
        </p>
      </div>
    );
  }
  return (
    <div className="composio-onb-actions">
      <button
        type="button"
        className="btn btn-primary composio-onb-submit"
        onClick={onStart}
        disabled={starting}
      >
        {starting ? "Connecting…" : ctaLabel}
      </button>
    </div>
  );
}
