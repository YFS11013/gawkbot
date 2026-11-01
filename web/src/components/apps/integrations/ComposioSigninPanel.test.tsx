import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ComposioSigninPanel } from "./ComposioSigninPanel";

// Two platforms, one component. Where an installer exists the flow asks
// permission; where none exists (Windows) it must not offer a button that
// cannot succeed, and must not describe a failure that never happened.
// OFFICE-3043 is why: the offer-and-fail loop put a blocking dialog in front
// of someone on every broker start, with a "Try again" that always failed the
// same way.

function renderPanel(overrides: Partial<Parameters<typeof ComposioSigninPanel>[0]>) {
  const props = {
    phase: "install_required" as const,
    authUrl: "",
    installCommand: "npm install -g @composio/cli",
    starting: false,
    onStart: vi.fn(),
    ctaLabel: "Connect integrations",
    ...overrides,
  };
  return render(<ComposioSigninPanel {...props} />);
}

describe("<ComposioSigninPanel> platform capability", () => {
  it("offers the setup button where the computer can run it", () => {
    renderPanel({ autoInstall: true });
    expect(
      screen.getByRole("button", { name: /set it up/i }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/runs from a terminal/i)).not.toBeInTheDocument();
  });

  it("shows the command instead of the button where it cannot", () => {
    renderPanel({ autoInstall: false });
    expect(
      screen.queryByRole("button", { name: /set it up/i }),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/runs from a terminal/i)).toBeInTheDocument();
  });

  it("does not claim a failed setup that never started", () => {
    renderPanel({ phase: "cli_missing", autoInstall: false });
    expect(screen.queryByText(/did not finish/i)).not.toBeInTheDocument();
    expect(screen.getByText(/cannot install the helper by itself/i)).toBeInTheDocument();
  });

  it("keeps the original wording where a real attempt can fail", () => {
    renderPanel({ phase: "cli_missing", autoInstall: true });
    expect(screen.getByText(/setup did not finish/i)).toBeInTheDocument();
  });
});
