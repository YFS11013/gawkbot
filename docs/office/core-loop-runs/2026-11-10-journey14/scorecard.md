# Journey 14 独立评分卡

> 判分基准：`prereg.md` 的 V1–V7 与 F-a…F-i。证据仅取 `artifacts/` 下的归档与 `internal/team/` 代码；`observations.md` 只作时间线索，不作证据。所有时间为 UTC；`broker-log.txt`/`broker-log-tail.txt`/`log-latency.txt` 原文为 +08:00，已换算。

## 0 总览

| 编号 | 结论 | 证据锚点（文件 + 时间 + 字段） |
|---|---|---|
| V1 | 未观察 | 全部 artifacts 无 `is a container` 文本；父 Issue 无 task 端 `artifact_path` 变更（`final-state.json` JOURN-1 `artifact=null`，`updated_at=2026-11-10T11:30:45Z`） |
| V2 | 成立 | `final-state.json` JOURN-1 `"artifact": null`（`updated_at=2026-11-10T11:30:45Z`）；`parent-decision-packet.json` 无 artifact 字段（`updatedAt=2026-11-10T11:43:46.4591776Z`） |
| V3 | 成立 | `messages.json` msg-91 `payload.to_state=decision` `timestamp=2026-11-10T11:31:07Z`；`final-state.json` request-92 `dedupe_key=decision:JOURN-1` `created_at=2026-11-10T11:31:07Z` `blocking=null` `required=null` |
| V4 | 成立 | `final-state.json` request-92 `question`（`created_at=2026-11-10T11:31:07Z`）原文列出两条 lane 路径，无 `The deliverable is` |
| V5 | 成立 | `final-state.json` request-92 `answered.answered_at=2026-11-10T11:43:46Z`；JOURN-1 `status=done` `lifecycle_state=approved` `hollow_completion=null`；`parent-decision-packet.json` `lifecycleState=approved` `updatedAt=2026-11-10T11:43:46.4591776Z` |
| V6 | 成立 | `final-state.json` request-8 `dedupe_key=plan-approval:JOURN-1` 唯一；`broker-state.json` 分解检查 `last_run=2026-11-10T10:18:14Z` `status_reason="decomposed into 2 sub-issue(s); 2 still open"`；JOURN-12/14 `artifact` 与 wiki 路径逐字一致；`team-tree.txt` 有 `team/decisions/JOURN-1.md` |
| V7 | 未观察 | 全部 artifacts 无容器关闭 409 与 `submit_for_review` 文案；父 Issue 由人 decision 卡关闭（request-92，`answered_at=2026-11-10T11:43:46Z`） |

| 编号 | 结论 | 证据锚点（文件 + 时间 + 字段） |
|---|---|---|
| F-a | 不成立 | 两次 409 后模型均继续：父 Issue 11:30:45Z 进 review、11:43:46Z approved（`final-state.json` JOURN-1 `status=done`） |
| F-b | 不成立 | 写范围拒绝逐 lane（`messages.json` msg-17/18），无换用另一条 lane 路径重试的记录 |
| F-c | 不成立 | `final-state.json` JOURN-1 `"artifact": null`；`broker-log.txt` 仅 JOURN-12/14 写盘，无 JOURN-1 写入 |
| F-d | 不成立 | `final-state.json` request-92 `dedupe_key=decision:JOURN-1` `created_at=2026-11-10T11:31:07Z` |
| F-e | 不成立 | request-92 `question` 列出 `team/accounts/acme-renewal-quote.md` 与 `team/emails/acme-renewal-followup.md` 两条 |
| F-f | 不成立 | 父 Issue 点击成功：request-92 `answered` 11:43:46Z，JOURN-1 `done/approved`，`hollow_completion=null`（lane 卡点击 409 见 §2/§3） |
| F-g | 不成立 | 计划卡唯一（request-8）；无 `task_not_decomposed`；lane `artifact` 与落盘一致；`team/decisions/JOURN-1.md` 存在 |
| F-h | 不成立 | 父 Issue 经人 decision 卡 request-92 关闭（11:43:46Z），非任务端点自关闭 |
| F-i | 不成立 | `messages.json` msg-13/15：JOURN-12 `10:13:42Z`、JOURN-14 `10:13:45Z` 两条子任务创建 |

## 1 逐条证据

### V2 — 父 Issue 的 artifact 始终为空

- `final-state.json` tasks[0]（JOURN-1）：`"artifact": null`、`"artifact_not_required": null`、`"status": "done"`、`"lifecycle_state": "approved"`、`"updated_at": "2026-11-10T11:30:45Z"`。
- `parent-decision-packet.json`：`"taskId": "JOURN-1"`、`"lifecycleState": "approved"`、`"updatedAt": "2026-11-10T11:43:46.4591776Z"`；对象内没有 artifact 字段。
- `broker-log.txt` 第 20/21 行只有 `wiki write-task: task=JOURN-12 ...` 与 `task=JOURN-14 ...`，无 JOURN-1 写盘；`broker-state.json` actions 里与 JOURN-1 相关的只有 `task_created`/`task_updated`/`request_*`，无 artifact 写入。
- 结论：成立。注意父 Issue 的 `definition.artifact_path` 在终态是 lane 路径（见 §3.1），但被测字段 `artifact` 始终为空。

### V3 — 父 Issue 进入 decision 且 ≤3 分钟出现 decision 卡

- `messages.json` msg-91：`"kind": "issue_lifecycle"`、`"content": "Ready for review — @cos submitted JOURN-1: Journey 14: Acme renewal pack"`、`"payload": {"from_state": "review", "to_state": "decision", "transition": "in_review", "task_id": "JOURN-1"}`、`"timestamp": "2026-11-10T11:31:07Z"`。
- `final-state.json` request-92：`"dedupe_key": "decision:JOURN-1"`、`"issue_id": "JOURN-1"`、`"created_at": "2026-11-10T11:31:07Z"`、`"blocking": null`、`"required": null`、`"status": "answered"`。
- 间隔：decision 转换 11:31:07Z，卡创建 11:31:07Z，0 秒 ≤ 3 分钟。
- 结论：成立。

### V4 — 卡正文列出每一条已交付 lane 的路径

`final-state.json` request-92 的 `question` 字段原文（与 `messages.json` msg-93 的 `payload.question` 逐字相同）：

> @cos finished Journey 14: Acme renewal pack (JOURN-1) and the review is in. Nothing else moves until you call it.
>
> This Issue's lanes delivered 2 file(s):
>
> - team/accounts/acme-renewal-quote.md
> - team/emails/acme-renewal-followup.md
>
> The Issue ships no single file of its own; those are what you are accepting.
> Approve and the lane closes. Send it back and the owner picks up your note as the rework.

- 两条 lane 路径都在，且没有 `The deliverable is <单一路径>` 口径；命中 `broker_decision_card.go` 的 `len(lanes) > 0` 分支（`childDeliverablesLocked` 取到 JOURN-12/14 的 `artifact`）。
- 结论：成立。

### V5 — 点 approve_decision 不带 artifact_path → 200，父 Issue done/approved，hollow_completion 不为 true

卡正文同 V4（request-92 `question`，原文引用见上）。点击与落地证据：

- `final-state.json` request-92 `answered`：`"choice_id": "approve_decision"`、`"choice_text": "Approve"`、`"answered_at": "2026-11-10T11:43:46Z"`。
- `messages.json` msg-96（`"from": "you"`，`"content": "Answered @cos's request: Approve"`，`reply_to=msg-93`，`timestamp=2026-11-10T11:43:46Z`）；msg-97（`kind=issue_lifecycle`，`payload.to_state=approved`，`timestamp=2026-11-10T11:43:46Z`）；msg-98（`kind=decision_approved`，`content` 含 `Canonical output: wiki/team/decisions/JOURN-1.md`，`timestamp=2026-11-10T11:43:46Z`）。
- `final-state.json` JOURN-1：`"status": "done"`、`"lifecycle_state": "approved"`、`"review_state": "approved"`、`"hollow_completion": null`、`"artifact": null`、`"artifact_not_required": null`。
- `parent-decision-packet.json`：`"lifecycleState": "approved"`、`"updatedAt": "2026-11-10T11:43:46.4591776Z"`。
- 代码侧：`broker_decision_packet.go:1025` `t.HollowCompletion = effectiveArtifact == "" && aggregateDelivery == 0`；父 Issue `aggregateDelivery=2`，故 false。终态序列化为 `null`（`hollow_completion` 为 `omitempty` 的 bool），语义上不为 true。
- 限制：HTTP 200 本身不在 artifacts 里（只有 observations 声称 `{"ok":true}`）；此处以同秒的状态跃迁、`answered` 记录与 `decision_approved` 消息为证。
- 结论：成立（状态与 hollow 判据有 artifact 直接支持；200 只有状态跃迁的间接支持）。

### V6 — 回归：计划卡唯一、task_not_decomposed 0、lane 路径一致、wiki decisions 文件

- 计划卡唯一：`final-state.json` request-8 `"dedupe_key": "plan-approval:JOURN-1"`、`"created_at": "2026-11-10T10:11:27Z"`、`"answered.choice_id": "approve"`；全部 9 条 request 中只有这一条 plan-approval。
- `task_not_decomposed` 0：`broker-state.json` `scheduler_runs["plan-decomposition-check:cos--human:task:journ-1"][0]`：`"started_at": "2026-11-10T10:18:14Z"`、`"status": "done"`；对应 watchdog：`"last_run": "2026-11-10T10:18:14Z"`、`"status_reason": "decomposed into 2 sub-issue(s); 2 still open"`；`signals` 里只有 4 条 `request_waiting`，无 `task_not_decomposed`。
- lane 声明路径与落盘路径逐字一致：`final-state.json` JOURN-12 `"artifact": "team/accounts/acme-renewal-quote.md"`、`verification.spec` 同路径、`wiki_refs[0]` 同路径；JOURN-14 `"artifact": "team/emails/acme-renewal-followup.md"`、`verification.spec` 同路径、`wiki_refs[0]` 同路径。`messages.json` msg-20/21（`kind=wiki_article_created`）的 `payload.path` 分别同为这两条路径（10:15:59Z / 10:16:03Z）；`team-tree.txt` 第 11/21 行有这两个文件。
- wiki decisions 文件：`team-tree.txt` 第 20 行 `team/decisions/JOURN-1.md`；`wiki-log.txt` 第 3 行 `fe59de5 decision: promote JOURN-1 on approval`。
- 注：两条 lane 的 `definition.artifact_path` 在终态为 `null`（见 §3.2）；本项的「声明路径」以 `task.artifact` + `verification.spec` + `wiki_refs` 与落盘路径比对，全部逐字一致。
- 结论：成立。

### F-a — 模型被 409 后不重试，父 Issue 停在既无 artifact 也没提交的状态

- 409 发生了两次：10:19:14Z（`broker-log.txt` 第 23/24 行，request-24/27 批准被 `artifact` DoD 拒绝）、11:28:26Z（`broker-log-tail.txt` 第 30/31 行，request-37/40 批准被 `wiki_artifact` DoD 拒绝）。
- 第一次之后模型继续：`messages.json` msg-33（10:19:46Z「批准两条子任务，随后聚合关闭 JOURN-1」）、msg-42（10:20:59Z「关联已建立，重新批准两条子任务」）、msg-47（10:22:51Z「尝试用 submit_for_review 把 artifact 记录进任务，再批准」）。
- 第二次之后：JOURN-12/14 于 11:28:59Z / 11:29:43Z 到达 done；父 Issue 11:30:45Z 进 review、11:31:07Z 进 decision、11:43:46Z approved（`final-state.json`）。
- 结论：不成立。父 Issue 最终 `status=done`、`lifecycle_state=approved`，未停在「无 artifact 也没提交」。

### F-b — 模型换一条 lane 的路径再试

- 写范围拒绝是逐 lane 的：msg-17（10:15:10Z）针对 JOURN-12 的 `team/accounts/acme-renewal-quote.md`，msg-18（10:15:17Z）针对 JOURN-14 的 `team/emails/acme-renewal-followup.md`；两条都是「declares no artifact_path」，不是「路径不属于本 lane」。
- `broker-log.txt` 第 18/19 行同样的两条拒绝，无第三条路径尝试。
- 结论：不成立。

### F-c — 父 Issue 的 artifact 仍被写成某条 lane 的路径

- `final-state.json` JOURN-1 `"artifact": null`；`parent-decision-packet.json` 无 artifact 字段。
- `broker-log.txt` 只有 JOURN-12/14 的 `wiki write-task`，无 JOURN-1 写盘；actions 里无 JOURN-1 的 artifact 写入。
- 结论：不成立。父 Issue 的 `definition.artifact_path` 被 define 钉成 lane 路径（§3.1），但被测字段 `artifact` 未被写。

### F-d — 父 Issue 进 decision 后卡没发

- `final-state.json` request-92：`"dedupe_key": "decision:JOURN-1"`、`"issue_id": "JOURN-1"`、`"created_at": "2026-11-10T11:31:07Z"`、`"status": "answered"`。
- 结论：不成立。

### F-e — 卡发了但只点名一条路径

- request-92 `question` 列出两条路径（见 V4 原文）。
- 结论：不成立。

### F-f — 点击后 409/500，或 hollow_completion=true

- 父 Issue 的批准点击：request-92 `answered.choice_id=approve_decision`，`answered_at=2026-11-10T11:43:46Z`；JOURN-1 `status=done`、`lifecycle_state=approved`、`hollow_completion=null`；msg-97/msg-98 同秒出现，无 409/500 证据。
- 需要单独点名的硬失败：两条 lane 卡的批准点击在 10:19:14Z 与 11:28:26Z 都返回了 409（`broker-log.txt` 第 23/24 行、`broker-log-tail.txt` 第 30/31 行）。这是 lane 级点击失败，不是 V5 的父 Issue 点击；详见 §2 与 §3.3。
- 结论：不成立（父 Issue 点击未 409、hollow 不为 true；lane 级 409 见 §2/§3）。

### F-g — 计划卡/写范围/路径钉出现回归

- 计划卡唯一（request-8）；分解检查 0 条 `task_not_decomposed`；两条 lane 的 `artifact`/`verification.spec`/`wiki_refs` 与落盘路径逐字一致；`team/decisions/JOURN-1.md` 存在。
- 写范围方面确实发生了两次拒绝（msg-17/18，10:15:10Z/10:15:17Z），但那是 lane 创建时未钉 `artifact_path`，模型随后补定义并成功写盘；终态 lane 的交付路径与落盘一致。这属于新摩擦，不构成 V6 判据的回归。
- 结论：不成立。写范围拒绝与 `definition.artifact_path` 被清空见 §3.2。

### F-h — 模型找到第三条关闭容器的路径，人类仍然没被问

- 父 Issue 的关闭路径有人工参与：msg-96（`"from": "you"`，11:43:46Z）→ msg-97（`to_state=approved`）→ msg-98（`decision_approved`）；request-92 `answered.choice_id=approve_decision`。
- 结论：不成立。容器（父 Issue）是经人 decision 卡关闭的。lane 层在批准被拒后另有关闭路径（§3.4），但不是容器。

### F-i — 钉死形状后仍然不拆 lane

- `messages.json` msg-13：`"kind": "issue_created"`、`"title": "Acme renewal one-page quote"`、`payload.task_id=JOURN-12`、`timestamp=2026-11-10T10:13:42Z`。
- msg-15：`"kind": "issue_created"`、`"title": "Acme renewal follow-up email"`、`payload.task_id=JOURN-14`、`timestamp=2026-11-10T10:13:45Z`。
- `final-state.json` JOURN-12/14 `parent_issue_id=JOURN-1`，两条独立 lane，各自独立文件。
- 结论：不成立。

## 2 硬失败与回退

硬失败（按时间）：

1. 10:15:10Z / 10:15:17Z — wiki 写范围拒绝。`broker-log.txt` 第 18/19 行：`wiki write-task refused: task=JOURN-12 ... artifact path not declared: JOURN-12 declares no artifact_path, so no wiki path is authorized for it. ...`；`messages.json` msg-17/msg-18（`kind=artifact_path_missing`）。回退：两次写入未落盘，wiki 在无文件状态，直到模型补定义后 10:15:59Z / 10:16:03Z 才写入（`broker-log.txt` 第 20/21 行）。
2. 10:19:14Z — lane 决策卡批准被 409。`broker-log.txt` 第 23/24 行：`task JOURN-12 definition-of-done check failed (artifact): no non-empty artifact matches "team/accounts/acme-renewal-quote.md"`（JOURN-14 同，`team/emails/acme-renewal-followup.md`）。回退：批准被拒绝，lane 回到 in_progress（`broker-state.json` actions action-30/32，10:20:22Z `[in_progress]`）。
3. 10:29:14Z — bot provider 错误。`log-latency.txt` 第 8 行：`bot=cos status=error provider=opencode total_ms=599779 detail="exit status 1"`（run 起于 10:19:14Z，约 10 分钟）。observations 未记。
4. 11:03:34Z — bot provider 错误。`log-latency.txt` 第 11 行：`status=error total_ms=599734 detail="exit status 1"`（run 起于 10:53:34Z）。observations 未记。
5. 11:28:26Z — lane 决策卡批准被 409。`broker-log-tail.txt` 第 30/31 行：`task JOURN-12 definition-of-done check failed (wiki_artifact): no artifact_path on the completion: pass the wiki-relative path of the deliverable ...`（JOURN-14 同）。回退：批准被拒绝；但 33–97 秒后 lane 仍到达 done（见 §3.4）。
6. 模型自己的 define 改校验未生效：msg-70（10:58:57Z）声称「定义已改为 none」，但归档中没有任何 `kind=none` 的 verification；11:28:26Z 的校验仍是 `wiki_artifact`（操作员的 define 之后）。msg-72（11:00:34Z）也自述「define 改 none 没跳过」。

操作员干预（按时间）：

- 10:10:16Z — 答 request-3（事实采访，`final-state.json` request-3 `answered.choice_id=provide_now`）。预算内。
- 10:13:08Z — 批准 request-8（计划卡，`answered.choice_id=approve`）。预算内。
- 10:19:14Z — 点 request-24/27 的 approve_decision（lane 卡），均 409。超预算。
- 11:26:37Z — broker 重启 #1（`broker-log-tail.txt` 第 18 行 `Launching WUPHF Office web view...` 后第 19 行 19:26:37 +08:00）。超预算。
- 11:27:47Z — 对 JOURN-12/14 各做一次 define，把 verification 从 `artifact` 替换为 `wiki_artifact`（`broker-state.json` actions action-81/83 `kind=verification_replaced`）。超预算；这是干预，不是系统自身行为。
- 11:28:26Z — 点 request-37/40 的 approve_decision（lane 卡），均 409。超预算。
- 11:43:14Z — broker 重启 #2（`broker-log-tail.txt` 第 42/43 行）。超预算。
- 11:43:46Z — 点 request-92 的 approve_decision（父 Issue 卡）。预算内。

预算：prereg §5 为 3 次（事实采访、计划卡、父 Issue 卡）。实际干预 8 次（含两次 lane 卡点击、一次双 lane define、两次重启），超预算 5 次。

## 3 判据外的新发现

1. 父 Issue 的 `definition.artifact_path` 被 define 钉成 lane 路径，容器守卫没有拦。`final-state.json` JOURN-1 `definition.artifact_path = "team/accounts/acme-renewal-quote.md"`、`defined_at = 2026-11-10T10:52:54Z`，而 JOURN-12 的 `artifact` 正是该路径；同任务 `artifact` 为 null。代码 `broker_tasks_mutation_service.go:1462-1480` 的容器守卫只检查 `body.ArtifactPath`（task 的 `artifact` 字段），不检查 `body.Definition.ArtifactPath`；`wiki_handlers.go:172-190` 的写范围守卫只读 `Definition.ArtifactPath`。因此一条 define 就能让父 Issue 获得写某条 lane 路径的授权，且无任何 `is a container` 文案。本轮没有发生父 Issue 写盘，所以 F-c 不成立，但这是 V1 守卫的一个真实旁路。
2. `define` 会重建 definition，省略 `artifact_path` 会清空 lane 的写范围。`final-state.json` JOURN-12/14 `definition.artifact_path = null`、`defined_at = 2026-11-10T11:27:47Z`（操作员 define 的时间）。对比：10:15:10Z/10:15:17Z 因 `declares no artifact_path` 被拒，10:15:59Z/10:16:03Z 写入成功，说明当时两条 lane 的 `definition.artifact_path` 已被模型钉上；11:27:47Z 的 define 把它清成 null。代码 `task_definition.go:80` `out := &TaskDefinition{Goal: goal, ArtifactPath: strings.TrimSpace(in.ArtifactPath), ...}` 是整体替换，不与旧 definition 合并。
3. lane 决策卡批准在 11:28:26Z 被 `wiki_artifact` 校验 409，校验只读请求体 `artifact_path`，不读 lane 自己记录的 artifact。`broker-log-tail.txt` 第 30/31 行原文：`definition-of-done check failed (wiki_artifact): no artifact_path on the completion: pass the wiki-relative path of the deliverable`。当前代码 `task_verification.go:371-387` 已加入回退（`declared == "" { declared = recordedArtifact }`），注释明确写着「Journey 14: a lane that had its artifact recorded still failed its own approve」——即该修复晚于本轮被测构建。
4. 两条 lane 在批准被拒后 33–97 秒到达 done/approved，关闭路径没有留证。`broker-state.json` actions action-88/90：JOURN-12 `task_updated [done]` 11:28:59Z、JOURN-14 11:29:43Z；`final-state.json` JOURN-12/14 `completed_at` 同两位秒、`review_state="not_required"`、`hollow_completion=null`。归档里没有对应的 cos 工具调用或消息：msg-85（11:28:08Z）之后下一条 cos 消息是 msg-94（11:31:08Z）。`messages.json` 里两条 lane 只有 `task_delivered`（msg-88/90），没有 `decision_approved`；request-37/40 的 `answered` 停在 11:28:26Z 的批准被拒。[INFERENCE] 关闭走的是 completion 路径（`complete`/带 `artifact_path` 的提交），绕过了被拒的 decision 卡；精确调用无法从 artifacts 复原。
5. 模型自己的诊断反复出错。msg-50（10:23:26Z）：「校验卡在 artifact（工作区文件）上，而交付物是 wiki 文章。改用 define 把校验方式修正为 none。」msg-63（10:53:28Z）：「批准仍被 artifact 校验拦住 —— 校验器要的是工作区里的非空文件，而交付物是 wiki 文章。我先读 wiki 内容，并在工作区落地同名文件。」msg-66（10:58:16Z）：「Wiki 两份交付物都在，但校验器要的是工作区文件。先用 not_required opt-out 试批准。」msg-67（10:58:20Z）：「Opt-out 没生效——校验器仍按 workspace 文件查。改用 define 把校验方式彻底改为 none，再重新提交审阅。」msg-70（10:58:57Z）：「定义已改为 none，任务回到 in_progress。重新提交审阅。」msg-72（11:00:34Z）：「校验仍按 workspace 文件查，define 改 none 没跳过。试 complete + not_required 绕过。校验器坚持要工作区文件。查一下有没有可用的文件系统类外部动作。」其中 msg-70 的成功声明与归档矛盾（无 `kind=none` 证据；11:28:26Z 仍是 `wiki_artifact`）。
6. 两次 bot provider 错误（10:29:14Z、11:03:34Z，`exit status 1`，各约 10 分钟，`log-latency.txt` 第 8/11 行）未被 observations 记录；这两次崩溃发生在模型反复尝试关闭 lane 的时段，与 10:33:36Z、11:13:36Z 的 `task_stalled` 消息（`messages.json` msg-52/76）时间相邻。
7. 终态数据一致性瑕疵：`final-state.json` JOURN-1 `updated_at = 2026-11-10T11:30:45Z`（进 review 的时间），早于其批准时间 11:43:46Z；`completed_at = null`，但 `status=done`、`lifecycle_state=approved`。`parent-decision-packet.json` 的 `updatedAt` 则是 11:43:46.4591776Z。
8. 分解检查给出了正面证据：`broker-state.json` watchdog `plan-decomposition-check:cos--human:task:journ-1` `status_reason = "decomposed into 2 sub-issue(s); 2 still open"`（`last_run=2026-11-10T10:18:14Z`），与实际两条子任务一致。这不是缺陷，但说明「钉死形状」在分解检查层面被识别。

## 4 与 J11/J12 的机制差异

- J11（3 个 children）：cos 把父 Issue 的 `artifact` 写成某条 child 的路径，卡命中 `own != ""` 分支，只点名一条路径（J11 卡 V2 失败）；父 Issue 批准成功。聚合谓词没有被真机锻炼。
- J12（2 个 children）：容器写入守卫拒绝了 cos 的 `artifact_path`（msg-42「父任务是容器，不能引用子任务的产物」），随后 cos 用 `complete` + `verification=not_required` 从任务端点自关父 Issue，`hollow_completion=true`，无 decision 卡。J12 锻炼的是容器写入守卫与容器 terminalize 旁路。
- J13：无 children，单文件，V1/V7 不可测。
- J14（2 个 children）：模型拆了两条 lane；父 Issue 卡命中 `len(lanes) > 0` 分支并点名两条（V4 成立）；父 Issue 经人 decision 卡批准、`hollow_completion` 不为 true（V5 成立）。这是聚合谓词第一次在真机上被锻炼并走通。
- J14 新锻炼的机制：`definition.artifact_path` 写范围守卫（msg-17/18 的拒绝）；DoD 校验在 wiki 交付物上的行为（`artifact` 与 `wiki_artifact` 两种 kind、以及 11:28:26Z 的 `wiki_artifact` 拒绝）；操作员 define 替换 verification；lane 经非 decision 路径关闭（§3.4）。
- J14 仍不能锻炼的：容器 `is a container` 写入拒绝（V1 未观察，模型从未对父 Issue 做 task 端 `artifact_path` 变更）；容器任务端点关闭 409（V7 未观察，模型从未尝试关闭父 Issue 容器，只对 lane 试过 `complete + not_required`）；`hollow_completion` 路径（父 Issue 有已交付 lane，未触发）；F-c 的第三条写路径（父 Issue 的 `definition.artifact_path` 虽被钉成 lane 路径，但没有发生父 Issue 写盘）。

## 5 可信度限制

- 二进制切换：`broker-log.txt` 显示初始启动 10:02:36Z；`broker-log-tail.txt` 显示两次重启（11:26:37Z、11:43:14Z）。artifacts 不含任何版本号/构建号，build 归属只能按重启时间推断。V3/V4 的卡产生于 11:26:37Z 之后的构建；V5 的批准发生在 11:43:14Z 之后的构建；两条 lane 的 `verification_result.pass=true` 盖在 11:43:30Z（11:43:14Z 重启之后）。observations 声称最终构建含「三处修复」，但 artifacts 无法证实构建内容；11:28:26Z 的 `wiki_artifact` 拒绝表明 recorded-artifact 回退修复不在该构建里。
- HTTP 状态码未归档：409 文案在 `broker-log*.txt`；父 Issue 批准的 200 只见于 observations，artifacts 只有同秒的状态跃迁与 `answered` 记录。
- observations 与 artifacts 矛盾处：observations 称「11:43:32 停表前快照」，但 request-92 `created_at=2026-11-10T11:31:07Z`；observations 称「11:28 重启」，但 `broker-log-tail.txt` 的重启时间是 11:26:37Z；observations 称 10:23:34Z `verification_result pass=false kind=artifact`，但归档只有终态 `verification_result`，无历史值；observations 称 `hollow_completion`「缺省(=false)」，final-state 实际是 `null`。
- lane 关闭路径无工具调用/消息留证（§3.4），只能从 `review_state=not_required`、无 `decision_approved`、只有 `task_delivered` 推断。
- 操作员干预（11:27:47Z 的 define、两次重启、多次点击）使终态不是系统自身行为；V5 的成功依赖操作员的 define 把 lane 门替换掉，以及 11:43:14Z 重启后的构建。
- 环境：两次重启时 `gbrain unavailable ... Facts will NOT persist across restarts`（`broker-log-tail.txt` 第 19/43 行），wiki 索引退回内存后端；`team-tree.txt`/`wiki-log.txt` 是 wiki 状态的唯一证据。
- 终态 JOURN-1 `updated_at` 早于批准、`completed_at=null`（§3.7），任务时间戳不能直接当批准时刻使用。
- 归档的 `verification_result` 只有终态；10:19:14Z/10:23:34Z 的 `pass=false` 未归档，中间态的 `verification.kind` 只能从 10:19:14Z 的 `artifact` 与 11:28:26Z 的 `wiki_artifact` 两个拒绝点推断。
