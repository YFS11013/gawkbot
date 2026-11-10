# Changelog

All notable changes to WUPHF will be documented in this file.

## [Unreleased]

### Security

- Bumped the Go toolchain directive to `go 1.25.11` so CI and release workflows
  using `go-version-file: go.mod` pick up the patch release's stdlib fixes for
  CVE-2026-42504, CVE-2026-42507, and x509 certificate parsing resource
  consumption. This is separate from the Time Machine cache exclusion change.

### Fixed

- **An Issue no longer adopts one of its lanes' files as its own artifact.** Journey 11 (`core-loop-runs/2026-11-10-journey11`): cos submitted `JOURN-2` for review with `artifact_path=team/accounts/acme-renewal-quote.md` — `JOURN-22`'s deliverable — and from then on the Issue carried one of its two files as "what this produced": the decision card named only that one, and the promotion followed. Both write paths (`MutateTask` and the decision approve) now look up whether a delivered child already owns the path and refuse with `task <id> is a container: <path> is <lane>'s deliverable, not this Issue's own. Leave artifact_path empty — its lanes' files are already on record and the decision card names every one of them — or pass a file <id> wrote itself.` A container may still record a file no lane claims. Tests `TestContainerRefusesALanePathAsItsOwnArtifact`, `TestDecisionApproveRefusesALanePathAsTheIssueArtifact`, and `TestLaneStillRecordsItsOwnArtifact` for the lane side.

- **An Issue whose lanes shipped files can no longer be closed by a bot — it closes through the human's decision card.** Journey 12 (`core-loop-runs/2026-11-10-journey12`): the container guard fired correctly and cos repeated its rule back ("父任务是容器，不能引用子任务的产物"), then closed the Issue anyway from the task endpoint with `complete` + `verification=not_required` — no decision card was raised, no human was asked, and the delivered Issue was stamped `hollow_completion=true` with `artifact=null`. That path was the deliberately-left asymmetry J10 noted; J12 shows it is a sovereignty bypass. `MutateTask` now refuses bot-initiated `complete`/`approve` on a container with delivered lanes and names the route: `task <id> is a container with N delivered lane(s): a bot may not close it from here — that would skip the human. Call submit_for_review on it with no artifact_path; the card names every lane file and the human's approve closes the Issue.` A human may still close their own Issue from the task endpoint, and an empty container (lanes canceled, nothing shipped) closes normally. `HollowCompletion` on this path now counts aggregate delivery instead of looking only at the Issue's own artifact. Tests `TestContainerWithDeliveredLanesCannotCloseFromTheTaskEndpoint`, `TestEmptyContainerStillClosesFromTheTaskEndpoint`.

- **A self-heal lane's file no longer becomes the container Issue's own artifact.** `attachSelfHealCompletionToParentLocked` stamped the finished self-heal lane's path onto the stalled parent whenever the parent's `artifact` was empty — the third writer of that field (after `MutateTask` and the decision approve), and the one Journey 11/12's guards did not cover. The lane keeps its artifact; the Issue's stays empty so the decision card aggregates from its lanes instead of naming one file. Test `TestSelfHealLaneDoesNotStampItsArtifactOnTheContainer`.

- **A container cannot declare a lane's file as its write scope.** Journey 14 (`core-loop-runs/2026-11-10-journey14` §3.1): the Issue's `definition.artifact_path` was pinned to a lane's deliverable — the container guard covered the task's own `artifact` but not the write authorization `define` hands its owner, so one define let the Issue overwrite the very file it exists to aggregate. `define` now refuses a scope owned by a delivered lane with `task <id> is a container: <path> is <lane>'s deliverable, not a file this Issue writes itself. Leave artifact_path out of the definition — the decision card names every lane file.` Test `TestContainerCannotDeclareALanesFileAsItsWriteScope`.

- **A container no longer claims the file a pending lane is about to write.** Guard exercise (`core-loop-runs/2026-11-10-guards`): the lookup behind the container write guards matched only a lane's *recorded* `artifact`, so between a lane's creation (which declares `definition.artifact_path`) and its completion the Issue could still `define` that path as its own write scope — the Journey 14 bypass, one step earlier. The lookup now also matches a child's declared artifact scope and skips `canceled`/`cancelled`/`archived`/`rejected` children, which own nothing. Both write paths carry a test: `TestContainerCannotClaimAPendingLanesWriteScope` / `TestContainerCannotCompleteWithAPendingLanesDeclaredPath` (both red without the clause), and the retired-case counterpart `TestCanceledLaneWriteScopeDoesNotBlockTheContainer` / `TestCanceledLaneDeclaredPathDoesNotTripTheContainerGuard`.

- **A re-define no longer wipes the write scope it was not asked to change.** `normalizeTaskDefinition` builds a fresh definition, so a define that only meant to fix the verification check cleared the task's `artifact_path` — Journey 14: an operator define left both lanes with no declared scope. An omitted `artifact_path` now keeps the existing one; naming a new path replaces it. Test `TestDefineKeepsAnOmittedArtifactScope`.

- **A decision card whose approve is refused stays open.** Journey 14, live: clicking `approve_decision` on a lane whose definition-of-done check failed returned 409 — but the answer had already been persisted, so the card was consumed, the "Approved" line was posted, and the lane had no card left to retry (two lanes ended `answered` at the same second as their refused clicks). A refusal means the decision did not land: `answerRequestFromActor` now reopens the request (pending, answer cleared), removes the announcement, and logs `request_reopened`, so the human's click leaves them exactly where they were — including the reminder and follow-up deadlines `applyRequestAnswerLocked` had already cleared. Test `TestRefusedDecisionCardAnswerStaysOpen`.

- **A lane's own approve now checks the artifact the lane recorded.** Journey 14: the human's answer to a lane's decision card carries no `artifact_path` — the card names the deliverable the task already recorded — and the `wiki_artifact` check read only the request body, so the very lane that had recorded `team/accounts/acme-renewal-quote.md` failed its own approve with `no artifact_path on the completion`. The check now uses the task's recorded artifact when the request declares none (an explicit `artifact_path` still wins, since that is the value the running mutation is about to apply). Tests `TestWikiArtifactApproveUsesTheRecordedArtifact`, `TestWikiArtifactApproveWithoutAnyArtifactFailsClosed`.

- **The task tool now advertises `wiki_artifact`, and a failed definition-of-done can be corrected.** Journey 14 (`core-loop-runs/2026-11-10-journey14`): both lanes declared `verification_kind=artifact` with a wiki-relative spec — the schema only offered `command | artifact | url | none`, so the model could not choose the wiki-aware kind. The artifact check globs the task worktree, so a file published to the team wiki could never satisfy it: the human's approve on the lane's own decision card 409'd with `no non-empty artifact matches "team/accounts/acme-renewal-quote.md"` while the file sat in the wiki. The model diagnosed it exactly and tried to fix it with `action=define`, which was silently ignored because a re-define could not touch an established gate — leaving the lane permanently unclosable. The schema now names `wiki_artifact` for wiki deliverables, and a gate that has already failed can be replaced (or cleared) by a re-define, recorded in the action log; a passing gate still cannot be weakened. Tests `TestTeamTaskSchemaAdvertisesWikiArtifactVerification`, `TestDefineReplacesAFailedVerificationGate`, `TestDefineDoesNotReplaceAPassingVerificationGate`.

- **`team_wiki_search` accepts `query`, and zero hits now says what it searched.** Journey 11 (`core-loop-runs/2026-11-10-journey11`, JOURN-26): a QA lane called the tool with `query` — the argument name its sibling retrieval tools use — got `pattern is required`, and reported to the human that both deliverables "exist nowhere" while `/wiki/read` answered 200 for them. The tool now treats `query` as an alias (`pattern` still wins when both are sent), the empty-argument error names both fields, and a search with no matches answers `No hits for the literal substring "X" in <reader>. This is a substring match over article bodies, not a list of what exists…` instead of `[]`, so silence cannot be read as absence. Tests `TestTeamWikiSearchAcceptsQueryAlias`, `TestTeamWikiSearchPrefersPatternOverQuery`, `TestTeamWikiSearchZeroHitsSaysWhatItSearched`, `TestTeamWikiSearchEmptyArgsNameBothFields`.
- **A refused task comment now says why and where.** `POST /tasks/{id}/comment` on a task living in a bot↔bot DM returned `500 {"error":"failed to post message"}` and dropped the text — the operator had no way to learn the channel was the problem (Journey 11: the cos↔reviewer lane). The bot-DM guard is a refusal, not a server error, so it now answers `403 {"error":"failed to post message","channel":"cos__reviewer","detail":"bot DMs are observer-only for humans; open the thread from its consult marker instead"}`, and every other failure carries the underlying reason in `detail` too. Test `TestTaskCommentOnBotDMReturnsTheReason` (also asserts a refused comment is not stored).

- **A decision card for an Issue now names every file its lanes shipped, even
  when the Issue carries one of those paths as its own `artifact`.** Journey 11
  (`core-loop-runs/2026-11-10-journey11`): the parent Issue's card appeared 24
  seconds after `JOURN-2` entered `decision` and the click landed, but the card
  read `The deliverable is team/accounts/acme-renewal-quote.md.` — one file,
  while `team/emails/acme-renewal-followup.md` sat in the same Issue,
  unreported. Cause: cos submitted the Issue for review with one child's path as
  the Issue `artifact`, and the card preferred that field over the lanes. The
  card now treats a task with delivered children as a container: it lists every
  lane's file, mentions the Issue's own path only when no lane claims it, and
  never names the same file twice. Test
  `TestDecisionCardNamesEveryLaneEvenWhenIssueCarriesOneArtifact`; with the old
  branch order restored it fails with the live wording
  (`the card must name every delivered lane file, missing
  "team/emails/acme-renewal-followup.md"`).
- **An Issue whose lanes all shipped now has something to click, and approving
  it no longer demands a file the Issue does not have.** Journey 10
  (`core-loop-runs/2026-11-10-journey10`, F-I): the parent Issue `JOURN-14`
  reached `decision` at 02:54:20Z four seconds after `JOURN-23` landed its email
  draft — both lanes had files on disk (`dce4efc`, `5c3c6b3`) — yet the Issue's
  own `artifact` and `artifact_not_required` were both null, so
  `hasApprovableDeliverable` said "nothing to accept" and no `decision:` card
  was raised. Ten minutes later the run's stop table fired on a delivered Issue
  nobody had been asked about. Approving by hand did not help either: with no
  `artifact_path` the same run's probe got
  `409 … cannot be approved into done without a delivered artifact. Have the
  owner publish the deliverable…` — advice aimed at a bot, for work that was
  finished. An Issue is a container, so the predicate now counts what its lanes
  landed (`Broker.childDeliverablesLocked`): a parent with ≥1 delivered child
  carrying an artifact (or an explicit `not_required`) is approvable, the card
  names **every** file rather than one arbitrary path, and the decision approve
  gate accepts aggregate delivery while still refusing vapor. Approving records
  the Issue as delivered, not `hollow_completion`. Tests
  `TestDecisionCardRaisedForParentIssueWithDeliveredLanes` (drives the real path
  — `submit_for_review` → no reviewers → convergence → card → approve) and
  `TestDecisionCardSkippedForParentIssueWithHollowLanes`; the first fails without
  the predicate with the wire symptom `a parent Issue whose lanes delivered owes
  the human a card, got 0`, and fails without the gate with the same 409 text the
  live probe returned.
- **A task's declared artifact now lands on the path the task declared.**
  `/wiki/write-task` answered `200 {path: team/reviews/dec-smoke-answer.md,
  bytes_written: 15}` while appending those bytes to the similar-slug neighbour
  `team/reviews/dec-smoke.md` and creating nothing at the declared path — caught
  by the smoke for the decision card above, which then refused the lane's own
  approve with `artifact "…" does not exist in the wiki`. The culprit was the
  update-first reroute (`routeBotCreateToSimilarSlug`), which exists to stop bots
  minting near-duplicate articles and fired on task-scoped writes too: a bot
  `create` whose slug is ≥ 0.85 similar to a sibling becomes an `append_section`
  on that sibling. Task writes are now **pinned**
  (`WikiWorker.EnqueueTaskArtifact` → `wikiWriteRequest.PinnedPath` →
  `resolveWikiWriteTarget` returns the requested path and mode untouched), so the
  acknowledgement, the file on disk, and the completion gate's existence check
  describe one string. Free writes (`/wiki/write`, the MCP tools) still reroute.
  Test `TestTaskArtifactWriteKeepsItsDeclaredPath` fails without the pin with the
  exact wire symptom (`declared artifact was not written: … cannot find the
  file`) and also asserts the unpinned similar create still reroutes.
- **A lane that lands in `decision` now asks the human where they work — and
  answering the ask does the thing it says.** Journey 9's stop table: `JOURN-16`
  finished its email draft at 13:48:03, reviewer convergence moved it to
  `decision`, and it sat there with `pending_interview=null` and no request
  pointing at it. The only surface saying a call was owed was a web inbox row;
  the channel carried prose, so the office read as silent for the rest of the
  run. Convergence now raises one **non-blocking** card per lane — in the lane's
  channel, anchored to the task, naming the deliverable it is asking the human
  to accept — deduped on `decision:<taskID>` so the 30-second sweeper cannot
  stack copies, and retired automatically when the lane closes. It skips lanes
  with nothing to accept (no artifact and no `not_required` declaration): asking
  someone to approve nothing only teaches them to ignore these cards.
  Answering performs the decision through `RecordTaskDecisionWithComment` — the
  same chokepoint the Inbox's Approve / Request-changes buttons use, not a
  `/tasks` approve, which would have set `status=done` while leaving the task in
  the decision bucket. "Approve" lands `approved`; "send back" lands
  `changes_requested` with the human's note stamped on the task for the owner.
  A free-text thread reply on the card counts as "send back", not consent.
  Applying the same answer twice (a double click, or the same answer arriving
  from two entry points) is a no-op against a lane already in that state, and a
  refused decision is reported — 409 with the reason on the click path, a
  channel note on the reply path — instead of a 200 that changed nothing.
  Blocking stays false: the card is a request to the human, not a reason to park
  a bot.

- **A declared scope must be the file that lands — not merely "some non-visual
  file".** Journey 9 reproduced two escapes from the Journey 8 gate.
  `JOURN-18` declared `team/reviews/acme-renewal-check.md`, never wrote it, and
  closed against `JOURN-32`'s `acme-renewal-final-check.md`; the gate passed it
  because the artifact was non-empty and did not look visual, so the lane's own
  deliverable silently did not exist, and a probe reproduced the same 200. The
  landing is now checked against `definition.artifact_path` by `sameWikiPath`,
  with the refusal naming both paths and the three ways forward (publish at the
  declared path, have the lead redefine the lane, or declare `not_required` with
  a reason). Second escape: a re-sent `complete` on an already-done lane rewrote
  its artifact to `ra_4e8fa813a190faa3` (200), because the gate hung off the
  transition to done; any mutation that records an artifact is now held to the
  scope, terminal or not.

- **The CJK duplicate rule now catches the restatement it missed by 0.025.**
  Journey 9's planner wrote `Acme 续约数字与事实终审` beside
  `子任务 C：数字与事实终审（Acme 续约）` — containment 0.875 against the 0.9
  threshold — and renamed the write path by one word so the artifact_path rule
  did not fire, shipping one review twice. The threshold is now 0.85: still
  above the draft/final pair at 0.67, and the short `审阅 JOURN-N 的交付` shape
  is stopped by the six-shingle floor rather than by the threshold.

- **Two lanes can no longer be one deliverable, and a declared scope can no
  longer be closed by a visual artifact.** Journey 8's independent scorecard
  found three ways the new write-scope contract leaked. (1) The planner fired
  each lane twice, one second apart, re-prefixing the title
  (`JOURN-6-A: Acme 一页报价说明初稿` then `子任务 A — 一页报价说明初稿`); token
  similarity scored the pair 0.20 because CJK has no spaces, so the sibling
  guard let both through. Sibling dedupe now also compares CJK character
  shingles (containment ≥ 0.85 over a shared phrase of ~7+ characters, measured
  against the pairs it must and must not merge; 0.85 is Journey 9's tightening,
  see below), and two open lanes under one
  parent may no longer declare the same `definition.artifact_path` — one file is
  one deliverable. (2) `JOURN-17` declared
  `team/research/acme-renewal-quote.md` and closed with the visual artifact
  `ra_4904a7a50a981735`, which never lands in `team/**`; the completion gate now
  refuses that (`artifact_required`, naming the declared path and both ways
  forward) while `verification=not_required` + reason still clears it and
  unscoped lanes keep the old behaviour. (3) The plan-decomposition check
  reported `decomposed into 1 sub-issue(s)` for a parent that had grown six,
  because it counted only the still-open ones; `TaskSubIssueTotals` now reports
  lanes and open lanes separately, and a parent whose lanes all landed is still
  closed quietly instead of being woken for being childless.

- **A landing lane pulls the rest of its issue forward instead of waiting out
  the hour.** The only timer a running task carries is the hourly follow-up, so
  Journey 8's issue went silent from 09:35:57 — five of six lanes done, the sixth
  waiting on them, nothing left to wake anyone until 10:31 — and the run read as
  wedged for 26 minutes until the stop condition. Completions now re-arm the
  parent's and every still-running sibling's existing follow-up to fire ~2
  minutes out (never later than their current due; closed jobs stay closed,
  other tasks untouched). The promotion approval card pending in that same
  window (`request-117`) was not the cause and is pinned as such: it is created
  `Blocking=false` with `kind=approval`, so it neither gates chat nor parks a
  turn.

- **A lane is created with its write scope, or the planner is told to fix it
  before the lane exists.** `/wiki/write-task` authorizes exactly
  `definition.artifact_path`, and the tool description says so up front, so a
  sub-issue created without one never even attempts the write: Journey 6 and
  Journey 7 both ended with zero `write-task` calls and zero `team/**`
  markdown, every deliverable left in bot-private notebooks and visual
  artifacts. `team_task action=create` now accepts `definition` (goal +
  `artifact_scope`) in the same call — restricted to the office lead and
  human, so no bot can self-authorize a path by creating a task that names it
  — and a lead-authored sub-issue declaring neither `artifact_scope` nor
  `artifact_not_required_reason` is refused with the allowed roots, the exact
  field to set, and "retrying this create unchanged will be refused again".
  Specialists are exempt: they cannot pin a shared path, so refusing there
  would be a dead end rather than a fix. The `/wiki/write-task` refusal
  (J3-3) stays as the fallback for a declared-but-mismatched path.

- **An approved plan gets one short look before the hourly follow-up.**
  Approving a plan dispatches the owner exactly once; if that activation turn
  spends itself on anything else, nothing wakes the parent again until the
  hourly `task_follow_up`. Journey 6 (2026-11-09): `JOURN-8` was approved at
  06:23:19Z, its activation turn answered an unrelated pipeline question, and
  the task sat with zero sub-issues and zero wiki writes until the run's
  45-minute cap — next wake 07:23:19Z. Approval now arms a one-shot
  `plan_decomposition_check` five minutes out: it closes silently when the
  parent has a sub-issue (`decomposed into N sub-issue(s)`) or left Running,
  and otherwise raises a `task_not_decomposed` alert and wakes the owner once,
  escalating in-channel if every delivery path refuses. The check is exempt
  from the lifecycle supersede sweep: any task mutation re-arms
  `task_follow_up` and used to close the check with "superseded by
  task_follow_up" (Journey 7: 20 seconds after approval), which would have
  removed the guard exactly when the task stays Running while channel traffic
  looks like progress.

- **A question dies with the task it was about.** Canceling or finishing a task
  now cancels its active requests (`task closed (<state>)`). A pending
  interview parks the asking bot's turn (`Launcher.turnParkedOnInterview`
  drops every wake that bot is owed), so an orphaned card outlives the work and
  quietly wedges the office: Journey 6 canceled `JOURN-3` at 06:27:32Z and its
  `request-16` stayed pending, able to suppress cos's wakes for `JOURN-8` for
  the rest of the run.

- **A pending interview reminder is visible even when it is not blocking.**
  The `request_follow_up` tick posted its automation message only for
  `Blocking` requests, and a bot-raised interview is not blocking — so the
  06:52:15Z tick left a `request_waiting` watchdog row and no chat message, and
  the human never learned that a reply was the only thing that could unstick
  the office. Every pending request now says so in its channel ("Waiting on
  your answer" for non-blocking, unchanged "Waiting on human decision"
  otherwise), still once per alert lifetime.

- **A refused artifact write now reaches the person who can fix it.** When a
  task has no `definition.artifact_path`, `/wiki/write-task` used to answer the
  bot with "pass human_request instead" and stop there; the bot silently fell
  back to a notebook and the human only found out at review (Journey 3 and 4
  both hit this through sub-issues the planner created without a definition).
  The refusal now names the task's creator, points at
  `definition.artifact_path`, and says retrying cannot succeed — plus one
  `artifact_path_missing` system message in the task's channel addressing both
  the creator and the owner. Deduped per task off the audit log, so a bot that
  retries every turn still produces exactly one nudge.

- **Closed scheduler jobs say why they closed.** `status_reason` joins the
  scheduler wire shape: the review gate writes "awaiting review (state)", a
  superseded job writes "superseded by task_follow_up", a bulk close writes
  "target closed". Journey 4 could not tell a review-gate close from a
  supersede close on `recheck:…journ-16` because both paths stamp `last_run`.
  The supersede close also stops overwriting `last_run`: that field means "was
  dispatched at", and a state transition is not a dispatch.

### Added

- **Agents on this machine.** gawkbot now scans for the agent CLIs installed
  or running locally — Claude Code, Codex, Opencode, Gemini CLI, Cursor CLI,
  Qwen Code, Copilot CLI, Amp, Goose, Aider, Crush, Factory Droid, Ollama,
  MLX-LM and Exo, plus OpenClaw/Hermes gateways and the Cursor/Windsurf IDEs —
  and shows each one with its running sessions in Settings → Agents on this
  machine. One click (or "Adopt all") turns an agent into an office bot
  created by the Chief of Staff, who routes work to it. Native runtimes bind
  to their existing provider; the rest run on a new `cli-agent` provider that
  drives each CLI's non-interactive mode on the user's own sign-in. The Chief
  of Staff can do the same through the `team_local_agents` tool, behind a
  human approval card.

- **Routines app — Calendar replaced.** The sidebar Calendar entry is now
  Routines, with a Paperclip-style List view (default) and a Google-style
  monthly Calendar view. Every routine has an owning agent picked from a
  dropdown, a human-readable schedule (frequency chips + time/weekday
  pickers, no cron syntax), and a "Run in" channel selector that defaults
  to the owner's DM. A new `+ New routine` composer at `/routines/new`
  builds them from scratch; an inline editor on the detail page saves
  edits as new revisions.
- **Routine detail page with tabs.** Full-screen surface at
  `/routines/$slug` with Overview, Runs, Activity, Revisions, and
  Triggers tabs. Runs are expandable with status glyph, event trace,
  output summary, error block, and a link to the routine's target.
  Revisions list every saved snapshot with one-click Restore and an
  inline diff against the current state. Triggers shows the schedule
  card plus Webhook and Context-change cards flagged Coming soon.
- **Routine lifecycle in the broker.** Per-slug ring buffers for runs,
  activity, and revisions persist across restarts. New endpoints:
  `POST /scheduler` (derives slug, enforces uniqueness, seeds first
  revision), `PATCH /scheduler/{slug}` (extended to accept the full
  editable shape and snapshot a revision on content edits),
  `GET /scheduler/{slug}/{runs,activity,revisions}`, and
  `POST /scheduler/{slug}/revisions/{n}/restore`. Restoring records
  the restored state as the new top revision so history stays
  append-only and reversible.
- **Agent routine dispatcher.** `processAgentJob` fires routines into
  the owner's DM (or an explicit channel) with an `@mention` prefix in
  shared channels, tagging the agent's loop via the `Tagged[]` field.
  Each fire records a detailed `schedulerRun` (events, output summary,
  target snapshot, error block on failure) via `CompleteSchedulerRun`,
  which re-arms the job with status `scheduled` so the schedule keeps
  ticking. Existing routines stuck in terminal `done` state are
  auto-healed on broker load. Routines that fire more often than every
  15 minutes are rejected at create and edit time.

- **Paper-manual rich HTML artifact guidance and tutorials.** Agents now get
  default guidance for producing self-contained HTML companions in a warm
  technical-manual style, including exact Making Software cobalt figure ink,
  paper-like layouts, mono metadata, copy/export controls, and browser-checkable
  notebook, wiki, and chat tutorial fixtures.
- **Per-agent OS keychain substrate.** New v1 package for
  opaque credential handles backed by macOS Keychain, Linux libsecret, and
  Windows Credential Manager adapters. Handles serialize only an opaque
  versioned id, use that id as the keychain account capability, reject Linux
  plaintext `basic_text` collections, and keep broker construction explicit
  instead of a global singleton.
- **Broker-mediated credential handles.** The credential substrate now treats the
  handle id as the keychain account capability and requires a `BrokerIdentity`
  token for read/write/delete calls. `CredentialHandle` uses private runtime
  slots, serializes as `{version:1,id}`, and can be rehydrated only with
  broker-trusted `agentId`/scope context.
- **Agent runner core and Claude CLI adapter.** New v1 runner package
  freezes `RunnerSpawnRequest`, `RunnerEvent`, and `AgentRunner`, streams Claude
  CLI `--print` JSON lines into broker-journaled events, records cost usage,
  writes authoritative receipts, and fails the run when receipt storage fails.
  Broker runner routes add `POST /api/runners` plus
  `GET /api/runners/:id/events` SSE with bearer-to-agent gating.
- **Codex CLI adapter for agent runners.** The runner package now ships
  `createCodexCliRunner()` for `codex exec`, with trusted absolute binary
  resolution, minimal locale-stable credential injection, graceful-then-hard
  termination, token usage events, receipt-write failure handling, and a single
  parser summary when Codex output includes unrecognized non-final lines.
- **OpenAI-compatible HTTP agent runner adapter.** The runner package now
  includes `createOpenAICompatRunner()` for streaming chat-completions endpoints
  that speak OpenAI-style SSE. The adapter uses `AbortController` for
  termination and timeouts, selects bearer vs `x-api-key` auth from
  `CredentialScope`, writes authoritative receipts, records provider usage as
  cost events, and emits a zero-cost usage-missing note when providers omit
  usage data.
- **Channel participant rail for conversations.** Channel views now include a Slack-like participants list that shows which agents are part of the current channel, opens an agent panel from each row, filters out human seats, and keeps lead agents pinned. The rail supports adding available office agents, disabling or enabling specific channel participants, removing agents from only the current channel, and undoing a remove from the toast within five seconds.
- **Skills app reskinned as pixel-art trading cards.** Every entry in the Skills tab is now a TCG-style card with a procedurally-generated 144px pixel-art portrait (using the existing `drawPixelAvatar` system), status-driven type palette (active = electric, proposed = psychic with "NEEDS REVIEW" stamp, disabled = dark, archived = steel), and a 3D card flip (700ms, ease-out-expo, respects `prefers-reduced-motion`).
  - **Front face:** Title + creator byline, procedural portrait, status/owner stat strip, promoted "Triggers on" row, and scrollable flavor-text description.
  - **Back face:** Full skill metadata (`<dl>` with status, owners, created by, created/updated dates, source) plus "View full SKILL.md →" CTA into the SidePanel preview.
  - **Visual treatment:** TCG aspect ratio (63:88), rounded outer corners fused with the pixel-border treatment, scanline overlay for CRT feel. Press Start 2P + VT323 fonts preloaded.
  - **Accessibility:** Card faces use `inert` (off-axis face removed from tab order); stat strip wrapped in `<dl>` with visually-hidden Status/Owners labels for screen readers; flip button uses `aria-pressed` for toggle semantics, not disclosure; doubled black-then-cardstock focus ring; touch targets ≥ 36px desktop / 44px on coarse-pointer devices.
  - **Performance:** `contain: paint` on the card wrap caps repaint cost on grids of 20+ cards.
  - **Continuity:** All in-app skill flows (approve, reject, suggest changes, archive, disable, invoke) preserved below the card. Skill metadata pinned to UTC so created/updated dates render the same calendar day for every viewer.
- **Live agent event bubbles on the WUPHF roster.** Each agent row in the office sidebar now shows a per-agent pill that pulses on activity (halo for routine and milestone events), holds the most recent action for 60s (120s for milestones), dims into a 60s grace window, then rotates through role-aware Office voice idle copy ("watching tests", "doodling in Figma", "looking at memes") so the rail always feels alive even before any agent has done work. A new `Kind` field on the activity SSE event lets the backend tag events as `routine`, `milestone`, or `stuck`; a `headlessActivityClassifier` table maps tool/status/detail inputs onto the right kind. Stuck detection has two paths into the same UI surface: a 90-second stale-while-active reaper for agents that go quiet mid-task, and a watchdog hook for explicit alert escalation. Stuck pills get bordered chrome (works under `prefers-reduced-motion`) and fire an `aria-live="assertive"` announcement once per transition. A first-run nudge under the first agent row points new users at the chat ("→ tag `@<agent>` in #general") and disappears the moment any human posts; the broker tracks `humanHasPosted` durably (in-memory bool seeded from a one-time message-log scan on bootstrap) and surfaces it on `/office-members.meta` so the frontend hook needs no extra round-trip. EventSource disconnect grace is 5 seconds before the rail dims and a "Reconnecting…" badge appears at the bottom of the agent list; polling continues underneath so state recovers automatically without a banner.
- **Private-network team-member sharing with `wuphf share`.** A running office can now mint a one-use invite, start a private Tailscale/WireGuard web listener, and let one team member join the shared office without exposing the local broker bearer token. Public binds are blocked by default; existing single-user `npx wuphf` launches are unchanged.
- **Wiki-backed team learnings — agents can now record and retrieve compact reusable lessons separately from procedural playbooks.** New `team_learning_record` and `team_learning_search` MCP tools write validated, typed learnings into the team wiki, regenerate a human-readable `team/learnings/index.md`, surface playbook-linked lessons in the wiki UI, and inject high-confidence prior learnings into future agent prompts.
- **Multi-workspace support — run multiple isolated WUPHF teams side by side, switch between them with one click, and pause unused workspaces to stop burning tokens.** A new left-rail switcher shows every workspace, the create modal pre-fills from your current workspace's blueprint and config, and pause/resume drains in-flight agent work cleanly. Each workspace gets its own broker process on its own port pair (7910–7999), its own agent roster, and its own state under `~/.wuphf-spaces/<name>/.wuphf/`. API keys are forked at create time so you can mix providers across workspaces without one workspace overwriting another's config.
- **`wuphf workspace` CLI — list, create, switch, pause, resume, shred, restore, doctor.** Full lifecycle from the terminal, JSON output for automation (`--json`), and a `wuphf --workspace=<name> <cmd>` global override for single-shot commands without flipping the active workspace. `doctor` reconciles the registry against reality: orphan trees, zombie running state, port conflicts, partial migration, broken symlinks — each fix is opt-in.
- **Page-reload-on-switch architecture for the web rail.** Clicking a workspace in the rail navigates to that broker's URL (full reload). The SPA never sees more than one broker's auth token, CORS stays restricted to the served origin, and the UI never has to manage cross-broker state. The simpler architecture replaces an earlier smooth-switch design that codex outside-voice flagged as overbuilt and a real XSS-blast-radius regression.
- **Shred is reversible for 30 days.** `wuphf workspace shred` moves the workspace to `~/.wuphf-spaces/.trash/<name>-<timestamp>/` instead of deleting. Restore via `wuphf workspace restore <trash-id>` (CLI) or the 30-second undo toast (web). Auto-cleanup runs on the next CLI invocation after 30 days.
- **Symmetric workspace layout — every workspace lives at `~/.wuphf-spaces/<name>/.wuphf/`, including main.** First launch on a workspace-aware build atomically renames `~/.wuphf` to `~/.wuphf-spaces/main/.wuphf` and leaves a compatibility symlink for one release cycle so external scripts referencing `~/.wuphf/...` keep working. No special-cased main, no asymmetry tax compounding into future features.

### Changed

- **Pause now drains every Launcher subsystem, not just tmux pane dispatch.** A new `Launcher.Drain(ctx)` cancels the headless dispatch context, scheduler loop, watchdog loop, notify-poll loop, and pane dispatch in lockstep. The previous drain only covered tmux pane sends, which left web-mode's default headless dispatch path running through pause. Codex outside-voice caught this: pause was advertising drain semantics while leaving most of the dispatch surface untouched.
- **Phase-0 audit migrated 18 sites from `os.UserHomeDir()` to `config.RuntimeHomeDir()`.** Every WUPHF-state path now respects `WUPHF_RUNTIME_HOME`, so a broker spawned with `WUPHF_RUNTIME_HOME=/some/path` writes its state under that path instead of bleeding into the user's real `~/.wuphf/`. LLM CLI auth paths (`~/.codex`, `~/.claude`, `~/.config/opencode`) intentionally stay user-global so re-login is not required per workspace. A greppable regression test enforces the boundary on every CI run; an integration leak test spawns a real broker with an isolated runtime home and asserts zero writes under `$HOME/.wuphf/`.
- **`killStaleBroker` now verifies the detected PID's binary path matches `wuphf` before sending SIGKILL.** With workspaces reusing the 7910–7999 range, the previous behavior could have killed an unrelated process bound to a previously-vacated port. Codex outside-voice flagged this as a real safety regression that scope-aware port allocation introduces.
- **Per-agent opencode config now writes under `<runtime_home>/.wuphf/opencode-configs/<slug>.json` instead of `~/.config/opencode/opencode.<slug>.json`.** Two workspaces with the same agent slug would have raced on the shared opencode config directory; namespacing under runtime home eliminates the collision while preserving the read of the user's base opencode config (themes, providers).
- **Human approval cards now show what the action will actually do.** When an agent calls `team_action_execute` for a mutating external action (sending a Gmail email, posting to Slack, updating a HubSpot contact, etc.), the human approval card now surfaces a structured "What this will do" panel with the recipient, subject, body excerpt, and other decision-relevant payload fields, plus a faint footer with the action ID, account, and channel. Before this, the card said only "Approve gmail action: GMAIL_SEND_EMAIL" and the human had to dig into the agent transcript to see what was actually being sent. The new layout uses a recessed inset panel for the payload bullets (theme-aware via `--overlay-soft` so it reads correctly in both light and dark themes), label/value typography (faint `--text-secondary` label, prominent `--text` value), a `(truncated)` chip when an email body exceeds 240 runes, and a "No additional details available" fallback line for empty-payload cases. An `EXTERNAL ACTION` badge sits next to `BLOCKING` so the user sees at a glance that this is a side-effect-causing call. Both the modal `HumanInterviewOverlay` and the inline `InterviewBar` render through the shared `ApprovalContextView` component for consistency. Long bodies clip with rune-aware truncation so multi-byte characters (CJK, emoji) cannot land mid-codepoint and produce invalid UTF-8.

### Security

- **Closed PR #850 round-2 security gaps in agent runners.** Runner spawn
  options are now part of the protocol wire shape, OpenAI-compatible endpoints
  require an explicit broker allowlist before credentials are attached, CLI
  prompts are separated from flags with `--`, runner output redaction is
  streaming across event chunks, and adapter input buffers fail closed before
  unbounded line/SSE/error-body accumulation.
- **Agent runner spawn boundaries now prove credential ownership and constrain cwd.** Runner credential reads use a new ownership-aware keychain read that verifies stored `agentId` and scope before returning a secret, and broker runner routes resolve `cwd` under `<workspaceRoot>/<agentId>/` with symlink-aware `realpath` checks. `RunnerSpawnRequest` and `RunnerEvent` now serialize `schemaVersion: 1`, accept legacy unversioned JSON as v1, and expose an optional `providerRoute` slot for branch 10 routing without changing current adapter policy.
- **Hardened agent-runner adapters and broker SSE against forged output and stuck consumers.** Claude CLI, Codex CLI, and OpenAI-compatible runners now validate provider-reported cost before forwarding it, redact full and partial secrets from output, chunk stdio at the protocol byte budget, and use shared terminal cleanup for sink failures. Broker runner shutdown now terminates active runners, retains terminal streams only briefly for reconnects, caps runner capacity, emits event-log LSNs as SSE ids, honors `Last-Event-ID`, and disconnects overfilled SSE consumers with a structured reason.
- **Closed a confused-deputy bypass in the human approval gate.** Agent-controlled fields (`Summary`, `ConnectionKey`, `ActionID`) now flow through `sanitizeContextValue` before they enter the approval card's context string. Without this, a prompt-injected agent could craft a `Summary` containing fake `What this will do:` / `Action:` / `Channel:` sections at line starts; the web parser's first-match-wins regexes would surface the FORGED structure to the human, hiding the real action. The sanitizer collapses every newline variant (LF, CRLF, U+2028, U+2029) to a space and replaces the bullet glyph U+2022 with the middle dot U+00B7, so forged section headers cannot land at a line start where the parser's `^Section:` regexes match. The forged tokens still appear as inline text inside the rendered Why — visible to the human as a long run-on sentence that looks suspicious by construction. Adversarial regression tests (`TestBuildActionApprovalSpecRejectsForgedSummary`, `TestBuildActionApprovalSpecRejectsForgedConnectionKey`) pin the defense.
- **Hardened OS keychain adapters.** Keychain subprocesses now resolve trusted absolute command paths, run with a minimal locale-stable environment, and have bounded timeouts. Linux libsecret writes require a positive encrypted Secret Service collection check, inspect `basic_text` warnings on every probe/readback path, and clear entries if post-write verification sees a plaintext backend. Windows Credential Manager I/O now treats secrets as UTF-8 explicitly, and command failures preserve system causes and recovery hints instead of collapsing to generic empty-stderr errors.

### Deprecated

- **`wuphf --tui` is renamed to `wuphf --legacy-tui`.** The old flag remains as a warning-only alias for now; the legacy bubbletea TUI is slated for removal after the desktop replacement lands.

### Fixed

- **Windows turns now own their process tree through a kernel job object, so a
  tool that exits cannot leave node/bun behind.** `terminateHeadlessProcessPID`
  killed with `taskkill /F /T`, which walks parent→child links as they are at
  kill time — a descendant whose intermediate parent already exited (the normal
  shape of the `opencode.cmd` → node/bun shim) was in nobody's tree, and a turn
  that ended normally killed nothing at all. Every headless spawn now goes
  through `startHeadlessProcess`, which puts the child in a
  `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` job: terminating the job reaches
  reparented descendants, and closing it when the tool exits reaps whatever the
  tool started. `taskkill` still runs first, for a process created in the
  microseconds between `Start` and the job assignment. If the job cannot be
  created (a restrictive parent job) the turn keeps its old kill semantics.
  Regression: `TestHeadlessJobKillsLeftoversWhenTheToolExits` fails in 18s
  without the attach and passes in 2.5s with it.
- **Task/Issue ids are now personal to each workspace instead of always reading
  `OFFICE-N`.** The Linear-style prefix is minted from the workspace's company
  name, but that field is optional and stays blank unless the human picks a
  brand during onboarding — so `refreshIDPrefixFromWorkspaceLocked` fell through
  to the hardcoded `OFFICE` default for every workspace, even though each
  workspace always has a required Name. The prefix is now derived from the
  company name, then the workspace name (`acme-revops` → `ACMER-1`), then the
  local config's company name, only falling back to `OFFICE` when none is
  usable. A new `workspaceIDPrefix` helper skips letter-free candidates so a
  symbol-only company name can't mask a usable workspace name. Adds table tests
  plus a live-path regression test that seeds the workspace registry.
- **Agent runner correctness across provider routing, event ordering, terminal
  races, resume expiry, and receipt reconciliation.** Broker-resolved
  `providerKind` now flows into runner cost and receipt attribution, runner
  events append through a serialized durable-emission queue, terminal success
  and failure paths are CAS-gated to emit exactly one terminal event, expired
  runner SSE resumes return structured `410 runner_resume_window_expired`
  responses, failed runs write best-effort error receipts at the pre-minted
  receipt id, and the Go protocol verifier now rejects runner wire drift with
  TypeScript-parity vectors.
- **Public tunnel start now retries through transient `trycloudflare.com` failures instead of bouncing straight to the user.** When Cloudflare's free QuickTunnel API returns a 5xx or HTML body (the `error code: 1101` / "Error unmarshaling QuickTunnel response" / "failed to unmarshal quick Tunnel" chain), `cloudflared` exits in under a second before publishing a URL. Previously, that surfaced as a hard failure on the first click. The tunnel controller now respawns `cloudflared` up to 3 times with a 1.5s/3s backoff on a recognized transient signature (5xx + QuickTunnel unmarshal errors), keeping the loopback listener and share server stable across attempts so only the subprocess cycles. Non-transient failures (timeout, missing binary, context cancel, pipe error) skip the retry path so the user is not waiting through pointless respawns. If every attempt fails, the surfaced error gains a hint that `trycloudflare.com` itself looks unhealthy.
- **First-run agent nudge now reads as a high-contrast retro speech bubble instead of black text on the purple sidebar.** The "→ tag @&lt;agent&gt; in #general" line that points new users at the office chat had no CSS rule attached, so it inherited the body text color and rendered at roughly 3:1 contrast on the Nex sidebar. It now ships as an NES-style pixel dialog box: olive-yellow fill, near-black mono text (≈16:1 contrast, WCAG AAA), four zero-blur stacked `box-shadow` strokes for a sharp pixel border, a soft drop-shadow underneath for CRT depth, and a chunky 4×4 pixel tail that steps up-left from the bubble toward the agent avatar. A 1.6s `translateY` bob draws the eye without moving any layout-bound properties; `prefers-reduced-motion: reduce` falls back to a static bubble. Matches the existing PixelAvatar aesthetic in the agent rail.
- **Workspace endpoints are wired before the web broker starts serving.** Creating a workspace from the web UI no longer hits `503 {"error":"workspaces not configured"}` because `/workspaces/*` routes now receive the orchestrator during broker construction instead of after `LaunchWeb` blocks forever.
- **Newly created agents now use the office avatar sprite system instead of falling back to the deprecated legacy procedural sprites.** Operation-generated slugs like `planner`, `builder`, `growth`, `reviewer`, and `operator` now resolve to generated office portraits, and arbitrary custom slugs get deterministic office-style procedural portraits in both web and TUI surfaces.
- **Passing `--pack`/`--blueprint` no longer deletes broker state on launch.** Restarting the same office after a crash can pass the selected blueprint again; the launcher treated that as an authoritative reseed and removed `broker-state.json`, making the office look wiped even though `wuphf shred` was never run.
- **Fresh wikis no longer warn during Stage B skill synthesis because `team/agents/` does not exist yet.** The notebook signal scanner now treats the missing agents directory as an empty signal source.
- **Caret no longer drifts when typing past an `@agent` chip in the composer.** The mirror-overlay treatment rendered mention chips with 4–5px horizontal padding and `font-size: 0.9em`, so each chip took more width than the raw text in the textarea behind it. The caret fell behind the typed characters by a few pixels after every chip. Composer chips now inherit the textarea's font metrics exactly (15px, no padding, `display: inline`) and only apply a background highlight — layout-neutral, so character-for-character alignment with the textarea is preserved. Message-bubble chips keep the original styled pill treatment.
- **Self-heal task explosion is capped at 3 active tasks per agent.** An agent that failed on N distinct task IDs previously opened N self-heal entries because the dedupe key was `(agent, taskID)` and each new taskID landed a new entry. On long-running installs this produced hundreds of stale incident tasks for a single agent. Once an agent reaches `maxActiveSelfHealsPerAgent` (default 3, override with `WUPHF_SELF_HEAL_MAX_ACTIVE_PER_AGENT`), additional escalations merge their incident detail into the most recently updated active self-heal task instead of opening a new one. The agent slug match is anchored on the title's trailing space (the literal `@<slug>` followed by a space) so prefix-overlapping slugs like `eng` and `engineering` keep their repair lanes isolated. Adds four targeted tests covering the burst cap, the substring-collision guard, per-agent independence, and the exact-reuse path winning over overflow merge.

### Changed

- **Headless `claude --print` is now the default dispatch path for both `wuphf` (web) and `wuphf --legacy-tui`.** Anthropic re-sanctioned headless CLI reuse in the 2026-04 OpenClaw policy note, and it runs on the user's normal subscription quota — no separate extra-usage charge. Every turn now dispatches as a fresh `claude --print` invocation, matching how the Codex runtime already worked and unifying dispatch across both modes. The previous per-agent long-lived interactive tmux pane path is preserved as an internal fallback primitive (reachable if dispatch ever needs to promote to panes at runtime) but is no longer invoked at startup. tmux is still required for `--legacy-tui` since the channel-view TUI runs in tmux; the web UI no longer needs it.
- **Pane-fallback messaging updated to reflect the new default.** The stderr banner and `#general` system post no longer frame headless as "extra-usage quota" — it isn't, anymore. Messaging now reads as: pane-backed fallback attempted but unavailable → continuing with the default headless path on your normal subscription.
- **Inline Enter-key hint on wizard CTAs drops the redundant `Enter` word.** The hint rendered both a `↵` key pill and the word `Enter` next to it — same information twice. Now it's just the pill. On the task step (⌘+Enter), the modifier is rendered as its own pill so the cue stays two key pills instead of one pill plus a string.

## [0.0.7.1] - 2026-04-23

### Fixed

- **Rapid `@agent` tags now coalesce into one reply instead of cutting claude off mid-turn.** The previous per-slug queue served bursts serially but the 3-second minimum gap between sends was shorter than a typical claude turn (~30s), so the second `/clear` still wiped the first reply mid-generation. The queue now tracks a 60-second coalesce window per slug: if a new notification arrives while claude's previous turn is still in flight, the new prompt is MERGED into the pending next dispatch instead of triggering a fresh `/clear`. Claude sees one combined prompt with every question separated by a divider and answers them together. Single tags still dispatch immediately — coalesce only engages when there's a live turn to protect.

## [0.0.7.0] - 2026-04-22

### Added

- **Run the whole web app without a mouse.** Press `?` anywhere (outside a text field) to open a keyboard reference with every shortcut grouped by Global / Command palette / Onboarding wizard / Slash commands / Composer / Feed navigation.
- **`⌘1`–`⌘9` jump to channels 1–9.** The first nine channels in the sidebar now advertise their shortcut with a faint kbd badge that lights up on hover/focus/active. Dark-sidebar legible via `currentColor` inheritance.
- **Enter advances every wizard step.** Identity, Templates, Team, Setup, and Ready all move forward on Enter when their own gate is satisfied. The Task step keeps Enter for newlines and uses `⌘/Ctrl+Enter` to advance, with an inline kbd hint spelling out the difference. Every primary CTA carries a visible `↵ Enter` badge.
- **`? shortcuts` button** in the status bar opens the help modal for mouse users who don't know the shortcut exists.
- **Shared `<Kbd>` + `<KbdSequence>` component** with platform-aware modifier (`⌘` on macOS, `Ctrl` elsewhere). One visual treatment across sidebar hints, wizard CTAs, help modal, status bar, and command palette footer.
- **`:focus-visible` rings** on template tiles, runtime tiles, team tiles, and task suggestions — keyboard users can see where they are without imposing an outline on mouse users.

### Fixed

- **Escape no longer cascades through every open panel.** Pressing Escape with the help modal + command palette + thread panel open used to close all three at once. The modal now uses capture-phase `stopImmediatePropagation` so one press closes only the topmost surface.
- **Wizard Enter stops hijacking focused buttons.** Enter on Back/Skip/runtime-reorder buttons used to advance the step instead of firing the button. On the Ready step, Enter on Skip submitted the task anyway. Enter is now ignored when focus is on a `<button>`, `<a>`, or `<select>`.
- **Hold-Enter on the Ready step no longer corrupts config.json.** The wizard's Enter handler now bails on `e.repeat`, preventing parallel writes to the broker's /config endpoint (which has a known race under concurrent requests).
- **Focus restoration no longer targets detached DOM nodes.** Closing the help modal checks `isConnected` before calling `focus()` on the previously-focused element — if that element was unmounted while the modal was open, focus falls back cleanly.
- **Wizard gate logic and platform detection deduplicated.** `cmdEnterLabel()` now reuses the shared `MOD_KEY` from the Kbd component, and the inconsistent `hasAtLeastOneKey` / `hasAnyApiKey` names are unified.

## [0.0.6.6] - 2026-04-22

### Added

- **@-mentions render as colored chips in both the composer and your own messages.** Typing `@pm` in the textbox now shows a pink chip inline via a mirror-overlay (same font metrics as the textarea, transparent text on top, caret still visible). Human message bubbles apply the same treatment so `@pm what are you doing?` reads with the chip, not as a literal `@pm` token. Unknown slugs like `@joedoe` stay plain text — the chip only fires for registered agents. Shared helper `parseMentions` powers both surfaces and uses the same pattern as the broker's routing regex so what the UI highlights is exactly what the backend will route. Mirror scroll-syncs with the textarea so chips stay aligned once a message exceeds the 120px visible cap.

## [0.0.6.5] - 2026-04-22

### Fixed

- **Rapid `@agent` tags no longer clobber each other mid-turn.** Two `@pm` tags in quick succession used to type two `/clear` + prompt sequences into the same tmux pane back-to-back; claude's TUI finished one turn (answering whichever input it had fully parsed) and silently ate the other. A per-agent dispatch queue now serialises pane notifications with a 3-second minimum gap between `/clear` cycles, so each question gets its own turn. Different agents still dispatch in parallel — the gap is per-slug, not global.

## [0.0.6.3] - 2026-04-22

### Fixed

- **Web UI no longer thrashes with `Maximum update depth exceeded` on every render.** `useCommands` returned a freshly-built array on every call, so the Autocomplete effect watching `commands + items` re-fired on every Composer render, calling `onItems(items)` which re-set `acItems`, which re-rendered Composer. The hook now memoizes the derived array against the underlying query data, giving stable referential identity across renders.
- **Dead tmux panes no longer silently break the live-output stream forever.** When an office reseed (or an agent crash, or an overflow-pane recreation) invalidates a pane target, the capture loop used to log "stopped after 5 failures" and exit permanently. It now sleeps 30s, re-resolves the current pane target from the launcher, and resumes — so long-running sessions recover from pane churn without a restart.
- **Headless codex queue no longer spins in a stale-cancel livelock.** Prod logs showed dozens of `stale-turn: cancelling active turn after 0s` lines per session because an enqueue could preempt an active turn that had just started. Cancellation now requires both the configured staleness threshold AND a 2-second minimum turn age, floors out tight cancel loops without blocking legitimate preemption.

## [0.0.6.2] - 2026-04-22

### Fixed

- **Typing `@pm` in a human message now tags PM.** The web composer does not always commit typed `@slug` text into an explicit tag chip, so the POST body arrived with `tagged: []`. Routing then treated the message as untagged and woke CEO instead. The broker now auto-promotes `@slug` body mentions to `tagged` for every sender (not only agents), provided the slug matches a registered agent. Conversational `@` references to non-agents stay untouched.
- **`office_reseeded: respawn panes failed: tmux: no server running` no longer logs as an error.** Web/headless mode never has a tmux server; attaching to it is expected to fail and the headless dispatch path handles delivery. The no-session error class is now silenced there, so the console stops surfacing a recurring error for a normal code path. Real failures (permission denied, corrupt state) keep logging.

## [0.0.6.1] - 2026-04-22

### Fixed

- **@-tagging a specialist now wakes the specialist, not CEO.** Tagging any non-CEO agent in `#general` was silently routing to CEO instead, because the channel-membership filter ran before the explicit @-tag check. A newly hired specialist is in the broker's member list but not yet in `ch.Members` for `#general`, so the filter dropped the notification and CEO absorbed the message. Explicit @-tags from humans or CEO now bypass the channel-membership filter, the sender's intent is explicit and trumps domain inference. Both collaborative and focus modes are patched.
- **DMs to specialists now reach the specialist.** Agents hired via the web wizard (`POST /office-members`) were added to the broker's roster but not to the launcher's in-memory pack, so `activeSessionMembers()` silently excluded them from `agentNotificationTargets`. Any DM or explicit @-tag targeting a wizard-hired agent dropped into the void. `activeSessionMembers` now appends broker-only members after pack-listed ones, keeping pack ordering stable while ensuring every hired agent is reachable.

## [0.0.6.0] - 2026-04-21

### Fixed

- **Agents boot into real tmux panes again — no more silent drop to the extra-usage quota.** Pane-backed spawn was failing on every launch with `tmux: command too long` because the launch command embedded the full 5-10 KB system prompt as a single shell argument, exceeding tmux's internal command-parse buffer. The prompt now goes to `$TMPDIR/wuphf-prompt-<slug>.txt` and the launcher passes `--append-system-prompt-file` to `claude`, keeping the tmux command under 4 KB. A regression test pins the length bound so future prompt growth cannot repeat the bug. Interactive panes also mean no more parallel `claude --print` subprocess storm at launch, which removes the CPU-starvation path that was making the web UI appear to boot-timeout on first launch.
- **Pane-spawn failure now tells you what actually broke.** The fallback message used to say "Install tmux to run agents on your normal subscription" whether tmux was missing OR tmux was installed but rejected the command. If you hit the second case, the advice was wrong and misleading. The launcher now distinguishes the two cases: missing → install it; rejected → file a bug with the detail.
- **Broker tokens and system prompts no longer linger in `$TMPDIR` after the session ends.** `Launcher.Kill()` now removes per-agent `wuphf-mcp-*.json` and `wuphf-prompt-*.txt` files for every known office member on shutdown.
- **`npx wuphf` no longer falsely warns about itself.** The PATH shadow detector treated the npm package's `wuphf.js` launcher as a separate binary shadowing the native `wuphf`, even though both ship in the same `node_modules/wuphf/bin` directory. Sibling files in the running binary's own directory are now recognized as one install, not a shadow. Real shadows (a hand-built copy in `~/.local/bin`, a stale homebrew install) still surface.
- **Boot-error diagnostics survive the 10-second watchdog.** If the web UI threw a specific error during bootstrap (`Uncaught error: X`, `React failed to mount`), the 10s watchdog used to destructively replace that overlay with a generic "Boot timeout" message, erasing the actionable signal. First overlay now wins, and the generic watchdog no longer fires once any fatal is already on screen. When it does fire, the detail now includes `document.readyState`, `location.hash`, and which `/assets/index-*` bundles actually finished downloading — so a real boot-timeout can be debugged instead of guessed at.

## [0.0.5.3] - 2026-04-21

### Fixed

- **Messages in #general now reach your team lead instead of silently disappearing.** Five operation blueprints (`niche-crm`, `local-business-ai-package`, `bookkeeping-invoicing-service`, `multi-agent-workflow-consulting`, `paid-discord-community`) and two legacy packs (`coding-team`, `lead-gen-agency`) declared a non-ceo `lead_slug` (`operator`, `tech-lead`, `ae`). Broker code hardcodes `"cos"` for channel-member injection, focus-mode routing, and the one-on-one default, so human messages dispatched to `@cos` — an agent that never existed in those blueprints — and got dropped with no trace. The lead slug is now `cos` everywhere, and `TestAllOperationBlueprintsUseCEOLead` guards against regression. Fresh onboards route correctly. Existing installs that still have `operator` registered in `~/.wuphf/state.yaml` need a reset; a migration for in-place upgrades lands in a follow-up.

## [0.0.5.2] - 2026-04-20

### Fixed

- **Pane spawn no longer races across concurrent WUPHF instances.** Two launches on the same machine used to share a hardcoded tmux socket (`wuphf`) and session name (`wuphf-team`). When a second launch ran `kill-session` between the first launch's `new-session` and `split-window`, tmux tore down the whole server and the first launch's split-window exited 1 with "no server running on /private/tmp/tmux-*/wuphf". That race is what pushed prod into the headless fallback message "Running in headless mode (spawn visible agents failed: spawn first agent: exit status 1). Install tmux and relaunch to use interactive Claude sessions on your normal subscription." even though tmux was installed and healthy. The socket and session names now carry a `-<port>` suffix on any non-default broker port — dev (7899), worktree launches, CI, or any custom `--broker-port` — so prod on the default port still uses the original `wuphf` / `wuphf-team` and concurrent instances can't trample each other. Backward compatible: default-port installs keep historical names.
- **Pane spawn failures now carry the real tmux error.** `spawnVisibleAgents` used `.Run()` which threw away tmux's stderr, so the #general system message was limited to "exit status 1" with no hint of cause. Both split-window calls now use `CombinedOutput()` and append `(tmux: <stderr>)` to the wrapped error — e.g. `spawn first agent: exit status 1 (tmux: no server running on /private/tmp/tmux-501/wuphf)` — so the next failure diagnoses itself from the fallback message alone.

## [Unreleased] — `feat/llm-wiki`

### Added — LLM Wiki (Karpathy-pattern team memory)

- **Git-native team wiki at `~/.wuphf/wiki/`.** Every article is a real markdown file in a local git repo. Each agent commits under its own git identity (per-commit `-c user.name=...` flags — never touches your global git config). Your team's memory is explicit, yours, file-over-app, and portable. `cat` it, `git log` it, `git clone` it anywhere.
- **`--memory-backend markdown` as the new default for fresh installs.** Existing Nex/GBrain users keep their current backend via `.wuphf/config.yaml` — no forced migration. `--memory-backend` now accepts `markdown | nex | gbrain | none`, and the MCP tool surface switches accordingly: markdown exposes `team_wiki_*` tools, the knowledge-graph backends keep the existing `team_memory_*` tools. The two never coexist on one server instance.
- **Serialized-write worker with fail-fast backpressure.** All writes flow through a single goroutine-owned queue (buffered 64, 10s per-write timeout). On saturation the MCP tool returns `wiki queue saturated, retry on next turn` — no hidden retries, no silent blocking. Covered by an IRON regression matrix that verifies exact tool-registration per backend.
- **Crash recovery on startup.** If the wiki repo has uncommitted changes from a prior crashed write, startup auto-commits them with a `wuphf-recovery` author. No data loss, full trace in `git log`.
- **Backup mirror + double-fault recovery.** Every commit kicks off a debounced async copy to `~/.wuphf/wiki.bak/`. If the repo corrupts and the backup is healthy, startup restores automatically. If both are corrupt, WUPHF falls back to `--memory-backend none` with a banner rather than crashing.
- **Graceful fallback when git is unavailable.** Detected at startup; WUPHF disables the wiki backend and shows a banner telling you to install git. Never crashes.
- **Transactional blueprint materialization.** Each of the 6 shipped blueprints (`bookkeeping-invoicing-service`, `local-business-ai-package`, `multi-agent-workflow-consulting`, `niche-crm`, `paid-discord-community`, `youtube-factory`) declares a domain-specific `wiki_schema:` with thematic directories and skeleton articles. On blueprint pickup during onboarding, those land in your wiki via temp-dir-then-rename — either all skeletons land or none do. Idempotent: re-picking a blueprint never overwrites existing articles.
- **Wikipedia-style UI at `/wiki`.** Reading-first editorial layout: Fraunces display headings, Source Serif 4 body at 18px/1.72, warm-paper `#FAF8F2` palette. Full Wikipedia information architecture — Article/Talk/History hat-bar, infobox with dark title band, italic strapline ("From Team Wiki, your team's encyclopedia"), hatnote cross-refs, numbered nested TOC with `[hide]`, Page Stats / Cite This Page / Referenced By panels, Categories chip footer, Wikipedia-style page footer with "View git history / Cite / Download as markdown / Clone wiki locally" actions, fixed-bottom live edit-log footer pulsing on every `wiki:write` SSE event. Agent pixel avatars on every byline — ties the wiki visually to the rest of the WUPHF app. See `DESIGN-WIKI.md` for the full spec.
- **18 React components under `web/src/components/wiki/`** with 90%+ test coverage via Vitest + React Testing Library (net-new frontend test infrastructure, also usable by every future feature). `react-markdown` + `remark-gfm` + `remark-wiki-link` + `rehype-slug` + `rehype-autolink-headings` render article content. `dompurify` sanitizes. SSE live-update on `wiki:write` invalidates the affected article's TanStack Query cache in real time.
- **Wikilink parser** with shared Go ⇄ TypeScript test fixture at `web/tests/fixtures/wikilinks.json` — both parsers consume the same canonical grammar cases. Syntax: `[[slug]]` → `team/slug.md`, `[[slug|Display]]` → renders "Display" but links to slug. Broken wikilinks (target doesn't exist) render red with a trailing marker; healthy ones render with a dashed-blue underline that solidifies on hover.
- **`GET /wiki/article?path=...` rich endpoint** returns article content + extracted title (first H1) + revision count + contributors + backlinks (reverse index via tree walk) + word count. Matches `web/src/api/wiki.ts WikiArticle`. Agents read via MCP (`team_wiki_read` — raw markdown); UI reads via the rich endpoint.

### Architecture notes

- **Three design systems, one repo.** `DESIGN.md` covers the pixel-office marketing site (dark, Press Start 2P). `web/src/styles/global.css` covers the general Slack-inspired web app. `DESIGN-WIKI.md` covers the `/wiki` surface (editorial-reference, warm paper, Fraunces + Source Serif 4). Each scope has non-interchangeable rules.
- **Per-agent wikis are deferred to v1.1.** v1 ships team wiki only. Per-agent `agents/{slug}/` introduces a private-on-filesystem access model that isn't load-bearing for the demo moment.
- **LLM merge-resolver is deferred to v1.1.** v1 uses serialized writes — no concurrent commits can conflict. Merge-resolver only worth building once the serialized-write path shows measurable pain at real-world load.
- **Nex compounding intelligence layer (entity briefs, playbook compilation, skill generation) is deferred to v1.1.** These sit additively on top of the markdown files and are disableable — the file-over-app guarantee is preserved forever.

### Internal

- New Go packages touched: `internal/team/wiki_git.go`, `wiki_worker.go`, `wiki_article.go`, `wiki_e2e_test.go` + tests; `internal/operations/wiki_materialize.go` + tests; additions to `internal/teammcp/server.go`, `internal/team/broker.go`, `internal/team/broker_onboarding.go`, `internal/config/config.go`. New env var `WUPHF_MEMORY_BACKEND` drives the tool-surface switch (matches the existing `WUPHF_CHANNEL` / `WUPHF_AGENT_SLUG` propagation pattern from broker to MCP subprocess).
- 33+ new Go tests at 81.6% coverage on wiki files (`wiki_git.go` · `wiki_worker.go` · `wiki_article.go`). 80 new web tests at 90% coverage on `web/src/components/wiki/` and `web/src/lib/`. Cross-lane integration tests in `internal/team/wiki_e2e_test.go` exercise the full HTTP stack.
- Full-repo `go test ./...` green across all 25 packages. `go test -race ./internal/team/... -run TestE2EWiki` clean.

## [0.0.5.1] - 2026-04-20

### Fixed
- **Blueprint channel names no longer leak `{{command_slug}}` as literal text.** Onboarding blank-slate seeding now renders the `{{command_slug}}` template variable alongside `{{brand_name}}` and `{{brand_slug}}`, matching the sibling code paths in `internal/company/blueprints.go` and `internal/team/operation_bootstrap.go`. Default channels created from blueprint starter packs show a real command-room slug (e.g., `acme-co-command`) instead of `{{command-slug}}`.

## [0.0.5.0] - 2026-04-17

### Added
- **Won't Do column in the Tasks board.** Canceled tasks now have their own lane next to Done instead of disappearing silently. Drag a card onto it (or use the task detail modal's "Won't do" action) to record that the work was skipped without deleting it. Empty Won't Do / Blocked / Pending columns stay hidden when idle and reappear as drop targets while you are dragging.
- **Task detail modal with owner reassign and won't-do action.** Click any task card to open a detail view, reassign the owner in place, or mark the work as won't-do without leaving the board.

### Changed
- **"Blocked" stat on the Office Activity view split into two pills.** The single "Blocked" card used to show `blocked tasks + watchdog alerts` combined so a "2" there could mean anything. Now you see "Blocked lanes" and "Watchdog alerts" as separate counts, and clicking either pill smooth-scrolls down to the "Needs attention" list where you can act on the items. Both are keyboard-activatable (Enter/Space) with an accessible label.

## [0.0.4.1] - 2026-04-17

### Added
- **One CLI is now selectable in Settings → Integrations → Action Provider.** The dropdown was missing the option even though the action registry already routed to One CLI by default for connections, action execution, and relays. The React settings UI, the legacy HTML fallback, and the typed API client all expose the option now.

### Fixed
- **Saving `action_provider = one` from the web UI no longer 400s.** The `POST /config` handler's allowlist only accepted `auto` and `composio`, so even though `/config set action_provider one` worked from the CLI, clicking Save in the web UI silently failed with HTTP 400 "unsupported action_provider". Added a regression test covering every provider value the registry supports.

## [0.0.4.0] - 2026-04-17

### Added
- **Shred your workspace from Settings.** New "Danger Zone" section in the web Settings with a `Shred workspace` button that deletes your team, company identity, office task receipts, and workflows, then reopens onboarding on next launch. The card lists exactly what gets deleted vs preserved, and the confirm modal requires typing `i am sure` before firing. Task worktrees, logs, sessions, LLM caches, and `config.json` are always preserved.
- **`wuphf shred` CLI subcommand.** Full workspace wipe that reopens onboarding. Prompts for the verb to confirm, or takes `-y` for scripted teardown.
- **`/shred` slash command in the TUI.** Wipes the workspace in-process, then exits the session so your next `wuphf` boots clean. The existing `/reset` (clear transcript and refresh panes) is unchanged.

## [0.0.3.0] - 2026-04-14

### Added
- **Skill invocations now drop you in the channel where the run is happening.** Click `⚡ Invoke` on the Skills tab, or run `/skill invoke <name>` from anywhere, and the UI jumps to the channel so you can watch the agents pick up the work instead of staring at the Skills list wondering if anything happened.

### Fixed
- **Broker stays up when something panics.** A panic inside a message-notification handler, task-action handler, or headless codex turn used to kill the whole broker (no stack, no logs). Three long-running goroutines now recover panics, write the full stack to `~/.wuphf/logs/panics.log`, and keep the office alive. If you see the broker die silently after this, that file will tell us exactly what blew up.
- **`/skills/<name>/invoke` now returns the resolved channel in its response.** The UI uses this to redirect reliably even when the skill has a default channel that differs from where you invoked from.

## [0.0.2.1] - 2026-04-14

### Removed
- **`docs/` removed from version control.** All planning documents, specs, and analysis files under `docs/` are now gitignored — local-only, never committed. Keeps the repo focused on shipped code.

## [0.0.2.0] - 2026-04-14

### Added
- **Resume in-flight work on restart.** When WUPHF shuts down with tasks in progress or conversations mid-flight, work now automatically resumes when WUPHF comes back up. On startup, agents receive a resume packet listing their active tasks (with stage, status, and working directory for worktree-isolated work) and any unanswered human messages awaiting their response. No more orphaned tasks or dropped conversations after a crash or restart.
- **Spec-compliant routing.** Resume packets route using pack membership: tagged messages go to the tagged agents, untagged messages go to the pack lead. Agents no longer in the current pack are silently skipped. The CEO is always enqueued first in headless mode to bypass the queue-hold guard.
- **29 new tests** covering in-flight detection, reply-chain parsing, pack membership filtering, 1:1 mode, nil-broker safety, terminal status exclusions (including `completed`), nex-sender inclusion, and the full resume flow in both tmux and headless paths.

### Changed
- `RecentHumanMessages` now includes the `nex` sender alongside `you` and `human`, so Nex automation messages that triggered work are correctly captured in resume packets.
- `findUnansweredMessages` now only counts replies from agent senders, so human-to-human thread continuations no longer falsely mark a message as answered.

## [0.0.1.0] - 2026-04-14

### Added
- **Proactive skill suggestions.** CEO agent now detects repeated workflows during normal conversation and proposes reusable skills via `[SKILL PROPOSAL]` blocks. Proposals surface as non-blocking interviews in the Requests panel. One-click accept activates the skill, reject archives it. The system learns from the team's actual work instead of requiring manual prompt editing.
- **Author-gated proposal parsing.** Only the team lead (CEO) can trigger skill proposals via message blocks. Prevents specialists and pasted transcripts from creating false proposals. Empty offices reject all proposals by default.
- **Agent team suggestions via existing tools.** CEO can suggest new specialist agents using the existing `team_member` and `team_channel_member` MCP tools with human approval via `human_interview`. No new data model needed.
- **11 unit tests** covering the full skill proposal lifecycle: CEO happy path, non-CEO rejection, malformed input, dedup, re-proposal after rejection, non-blocking interview creation, accept/reject callbacks, prompt content verification, persistence round-trip.
