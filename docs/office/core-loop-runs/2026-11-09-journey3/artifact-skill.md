---
name: smb-renewal-outreach
description: Run a 5-step SMB renewal outreach sequence from T-60 to T-7 with objection handling
version: 1.0.0
license: MIT
metadata:
  wuphf:
    title: smb-renewal-outreach
    source_articles:
      - team/playbooks/smb-renewal-outreach.md
    created_by: librarian
    status: active
    safety_scan:
      verdict: safe
      trust_level: community
      summary: safe (no findings)
    owner_agents:
      - ae
      - analyst
      - cos
      - reviewer
      - revops
---

## When this fires
Use when an SMB SaaS renewal is within T-60 and a written outreach sequence is needed to get an explicit renew / adjust / pause reply before the renewal date T.

## Inputs
- Renewal date T, contact name, company, current seats/plan — all from CRM fields, no manual guesses.
- CRM hygiene passed: dedup, owner filled, do-not-contact list synced.
- Analyst pipeline-truth baseline: stage definitions plus real renewal/churn denominator.
- Reviewer approval for the exact email version to send.
- Official price list / adjustable seats/period rules link.

## Steps
1. Verify prerequisites: CRM hygiene, analyst baseline, reviewer approval, complete contact/T/seats/plan. If any field or number is missing, write [NEEDS CONFIRMATION: ...] and stop — do not send.
2. Check ICP fit: 15-30 seat SMB, day-to-day admin as champ, owner/finance as economic buyer, usage declining, price-sensitive on predictability not lowest price, silence usually means no time or internal alignment needed. Treat as H1-H5 until ST-1 validates each; if any is falsified, pause expansion and re-profile first.
3. Enforce global rules: unsubscribe footer on every email; exit within 24h on opt-out; stop sequence immediately on negative reply; pause on hard bounce or 2 soft bounces; max 2 emails/week/contact; any real reply triggers human takeover; after Step 5 silence, auto-pause to quarterly nurture with no auto-restart.
4. Step 1 at T-60: send one self-check email (max 180 words): greet contact, recall plan/seats/T, ask whether usage is aligned, whether to continue as-is, and whether to adjust seats. CTA is a 15-minute alignment meeting where replying keep as-is counts as confirmation. Always include a pause opt-out. Positive reply: book 15 min and skip Steps 2-3. Internal alignment: confirm time, follow up at T-45. Silence: wait to T-45, no extra nudges.
5. Step 2 at T-45: send one value alignment email: ask the top 1-2 priorities and whether the value proposition still aligns; ask for a referral to the final decision-maker. Neutral replies like send info get a single document follow-up only, never a Step 3 push.
6. Step 3 at T-30: multi-thread rescue via email plus one call. Restate the T-30 deadline factually, offer two concrete options (keep as-is / micro-adjust seats) with prices only from the official price link. Log second-contact coverage and call outcome in CRM.
7. Step 4 at T-15: lock options by email plus meeting; meeting notes serve as confirmation. List official plan/seat/period options with terms link, request written reply by [NEEDS CONFIRMATION: deadline] choosing confirm renewal, adjust seats, or pause. Paste verbatim reply to the deal; never self-set the stage.
8. Step 5 at T-7: send final break-up email with a binary choice: confirm as-is, or pause to next quarter. State this is the last send, keep unsubscribe/pause links, then close the sequence.
9. Handle objections with skeletons only, never invented numbers: A Busy means acknowledge, ask for the decision-maker contact, offer keep as-is by reply. B Too expensive means acknowledge, share only official price and adjustable rules, recalc on written request, pause for reviewer on any out-of-policy promise. C Not now means acknowledge, offer pause now for T-7 close, never push as a positive renewal.
10. After each send, classify the reply (positive / internal alignment / referral / price / delay / negative / silence), write it to the deal comment, set the next T-milestone task, and verify frequency and stop rules.

## Output
- Reviewer-approved email draft for the current step with every [NEEDS CONFIRMATION: ...] cleared.
- CRM updates: reply classification, next-step date, pause/stop reason.
- For Steps 4-5: verbatim customer confirmation attached, or nurture transfer if paused.

## Invariants
<!-- INVARIANT-START -->
- NEVER send an email containing [NEEDS CONFIRMATION: ...]; clear every placeholder or pause.
- NEVER fabricate price, discount, case, or penalty terms; use only the official price list and public rules.
- NEVER send before reviewer approval and with CRM hygiene + analyst baseline in place.
- MUST include an unsubscribe link; MUST stop within 24h on opt-out or negative reply; MUST NOT exceed 2 emails/week per contact.
<!-- INVARIANT-END -->

## Examples
Step 1 for a 20-seat SaaS company at T-60: subject is company-name pre-renewal self-check plus 2-min confirm; body recalls current plan, seats, and renewal date, asks if usage is aligned and whether to renew as-is or adjust seats, offers a 15-minute meeting or a keep as-is reply, and adds a pause opt-out. If the customer replies internal alignment, confirm a T-45 follow-up time and send nothing else before T-45.
