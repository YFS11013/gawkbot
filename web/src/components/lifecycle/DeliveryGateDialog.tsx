import { useEffect, useRef, useState } from "react";

import type { DeliveryGateValue } from "../../lib/deliveryGate";

/**
 * The human's answer to the "done means delivered" refusal (J2-5).
 *
 * The broker refuses an Approve/Complete that would land a task done with
 * nothing to open, and its 409 tells the caller to publish a deliverable or
 * pass `verification=not_required` with a reason. The Approve button in this
 * UI posts neither, so without this dialog the sentence pointed at an
 * affordance that did not exist anywhere. Two ways out, exactly the two the
 * broker accepts:
 *
 *  - "Attach the deliverable": a wiki-relative path (or a visual-artifact
 *    id). The broker stats it, so the path must be real.
 *  - "This ships no file": a mandatory one-line reason, stamped on the task
 *    as `artifact_not_required` so the done record and the parent Issue can
 *    tell an accounted-for empty completion from a forgotten one.
 */
export interface DeliveryGateDialogProps {
  open: boolean;
  /** The broker's refusal sentence, shown verbatim. */
  message: string;
  /** Verb being retried, e.g. "Approve" — used in the title only. */
  actionLabel: string;
  busy?: boolean;
  onSubmit: (value: DeliveryGateValue) => void;
  onClose: () => void;
}

type Mode = "artifact" | "none";

export function DeliveryGateDialog({
  open,
  message,
  actionLabel,
  busy = false,
  onSubmit,
  onClose,
}: DeliveryGateDialogProps) {
  const [mode, setMode] = useState<Mode>("artifact");
  const [artifactPath, setArtifactPath] = useState("");
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const firstFieldRef = useRef<HTMLInputElement>(null);
  const reasonRef = useRef<HTMLTextAreaElement>(null);

  // Fresh dialog per refusal: never carry a stale path/reason across tasks.
  useEffect(() => {
    if (!open) return;
    setMode("artifact");
    setArtifactPath("");
    setReason("");
    setError(null);
    firstFieldRef.current?.focus();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  function submit() {
    setError(null);
    if (mode === "artifact") {
      const path = artifactPath.trim();
      if (!path) {
        setError(
          "Enter the deliverable's wiki path (e.g. team/playbooks/launch.md).",
        );
        return;
      }
      onSubmit({ artifactPath: path });
      return;
    }
    const why = reason.trim();
    if (!why) {
      setError("Say why this work ships no file — it is stamped on the task.");
      reasonRef.current?.focus();
      return;
    }
    onSubmit({ verification: "not_required", verificationReason: why });
  }

  return (
    <div
      className="confirm-overlay"
      onClick={(e) => {
        if (e.target === e.currentTarget && !busy) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="delivery-gate-title"
    >
      <div className="confirm-card card">
        <h3 id="delivery-gate-title" className="confirm-title">
          {actionLabel} needs a deliverable
        </h3>
        <p className="confirm-message">{message}</p>

        <div
          className="delivery-gate-options"
          role="radiogroup"
          aria-label="What does this completion ship?"
        >
          <label className="delivery-gate-option">
            <input
              type="radio"
              name="delivery-gate-mode"
              checked={mode === "artifact"}
              onChange={() => {
                setMode("artifact");
                setError(null);
              }}
              disabled={busy}
            />
            <span>Attach the deliverable</span>
          </label>
          <label className="delivery-gate-option">
            <input
              type="radio"
              name="delivery-gate-mode"
              checked={mode === "none"}
              onChange={() => {
                setMode("none");
                setError(null);
                setTimeout(() => reasonRef.current?.focus(), 0);
              }}
              disabled={busy}
            />
            <span>This ships no file</span>
          </label>
        </div>

        {mode === "artifact" ? (
          <div className="delivery-gate-field">
            <label htmlFor="delivery-gate-artifact">Wiki path</label>
            <input
              id="delivery-gate-artifact"
              ref={firstFieldRef}
              className="input"
              value={artifactPath}
              onChange={(e) => setArtifactPath(e.target.value)}
              placeholder="team/playbooks/launch.md"
              disabled={busy}
            />
            <p className="delivery-gate-hint">
              The broker checks the file exists — publish it to the wiki first.
            </p>
          </div>
        ) : (
          <div className="delivery-gate-field">
            <label htmlFor="delivery-gate-reason">Why nothing ships</label>
            <textarea
              id="delivery-gate-reason"
              ref={reasonRef}
              className="input"
              rows={3}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="The answer lived in chat; nothing to hand over."
              disabled={busy}
            />
            <p className="delivery-gate-hint">
              Recorded on the task as <code>artifact_not_required</code>.
            </p>
          </div>
        )}

        {error ? (
          <p className="delivery-gate-error" role="alert">
            {error}
          </p>
        ) : null}

        <div className="confirm-actions">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={busy}
          >
            Cancel
          </button>
          <button
            type="button"
            className="btn btn-primary btn-sm"
            onClick={submit}
            disabled={busy}
          >
            {busy ? "Saving…" : `${actionLabel} anyway`}
          </button>
        </div>
      </div>
    </div>
  );
}
