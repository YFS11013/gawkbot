/**
 * The "done means delivered" gate (broker: TaskMutationArtifactRequired)
 * refuses a complete/approve that would land a task done with nothing to
 * open. Before this module existed the refusal was a dead end in the UI:
 * the 409 told the human to "pass verification=not_required" and no screen
 * had a field for it. Everything below exists to turn that sentence into two
 * buttons.
 */
import { ApiError } from "../api/client";

/** What a human may attach to the retried mutation. */
export interface DeliveryGateValue {
  /** Wiki-relative deliverable (e.g. `team/playbooks/launch.md`) or a visual-artifact id. */
  artifactPath?: string;
  /** Only `not_required` is accepted by the broker. */
  verification?: string;
  /** Why this completion ships no file. Required with `verification`. */
  verificationReason?: string;
}

export interface DeliveryGateRefusal {
  /** The broker's own sentence, shown verbatim — it names the exact reason. */
  message: string;
}

/**
 * True when a task mutation was refused because the completion had no
 * artifact and no declaration. Anything else (forbidden, conflict, network)
 * is not this gate and must keep its normal error rendering.
 */
export function deliveryGateRefusal(err: unknown): DeliveryGateRefusal | null {
  if (!(err instanceof ApiError) || err.status !== 409) return null;
  const message = String(err.message ?? "");
  // Both refusals say "artifact"; the missing-reason refusal says
  // "verification_reason" instead.
  if (!/artifact|verification_reason/i.test(message)) return null;
  if (!/cannot (reach|be approved)|needs verification_reason/i.test(message)) {
    return null;
  }
  return { message };
}
