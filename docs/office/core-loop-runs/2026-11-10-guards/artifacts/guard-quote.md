# Guard lane: quote

## What this lane covers

The quote guard lane is the checkpoint that validates a commercial quote before it leaves the building. A quote is a pricing commitment: once sent, it is hard to walk back without eroding margin or trust. This lane exists so that every quote is checked for accuracy, authority, and terms before the customer sees it.

## Guardrails

- **Pricing accuracy** — every line item, rate, and total must reconcile with the approved price book. No manual overrides without a named approver.
- **Discount authority** — discounts above the standard band require explicit sign-off. The lane records who approved what, and caps the maximum discount per deal size.
- **Terms and conditions** — payment terms, delivery windows, and liability clauses must match the standard template. Any deviation is flagged before send.
- **Currency and units** — quotes are checked for correct currency, tax treatment, and unit of measure so the customer does not receive an ambiguous number.
- **Expiration** — every quote carries a valid-until date. Expired quotes cannot be re-sent as-is.

## Enforcement

The lane runs on every quote before it is issued. A quote that fails any guardrail is blocked and routed back to the owner with the specific failure named. Passing the lane is a precondition for send — there is no override that skips it silently.

## What this lane does not cover

The quote lane validates the commercial content of a quote. It does not vet the underlying opportunity, the customer's creditworthiness, or the technical feasibility of delivery — those sit in their own lanes upstream.
