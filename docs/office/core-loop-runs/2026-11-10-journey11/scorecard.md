# Journey 11 独立评分卡（2026-11-10）

> 判分基准：`prereg.md` V1–V7 与 F-a…F-f。证据仅取 `artifacts/` 与 `snapshots/`；`observations.md` 仅作时间线索。

## §0 总览

### V1–V7

| 验证项 | 结论 | 证据锚点 |
|---|---|---|
| V1 父 Issue 的卡出现了 | 通过 | `snapshots/0425/broker-state-slice.json` JOURN-2 `lifecycle_state=decision` `updated_at=04:24:27Z`；`artifacts/broker-state.json` `request-83` `dedupe_key=decision:JOURN-2` `issue_id=JOURN-2` `created_at=04:24:51Z` `status=answered` |
| V2 卡点名全部交付物 | 失败 | `snapshots/parent-decision-card.json` question: `The deliverable is team/accounts/acme-renewal-quote.md.`（仅一条；缺 `team/emails/acme-renewal-followup.md`） |
| V3 卡不阻塞 | 通过 | `snapshots/parent-decision-card.json` 无 `blocking`/`required` 字段；`broker-state.json` JOURN-2 `pending_interview=null`；JOURN-24/32 在卡期间有推进 |
| V4 点击即落地 | 通过 | `broker-state.json` `request-83` `answered.answered_at=04:25:00Z`；JOURN-2 `status=done` `lifecycle_state=approved` `updated_at=04:25:20Z`；`wiki-log.txt` commit `002deee` `decision: promote JOURN-2 on approval` 04:25:01Z |
| V5 approve 不再 409 | 通过 | `broker-state.json` `request-83` `answered.choice_id=approve_decision`；JOURN-2 `status=done`；无 409 证据 |
| V6 不撒谎 | 部分 | `hollow_completion=null`（通过）；但 `artifact=team/accounts/acme-renewal-quote.md`（失败，被写成某子任务路径） |
| V7 回归不破 | 通过 | 计划卡 `request-18` 唯一；`task_not_decomposed` 0；`plan-decomposition-check` 3 子 Issue（JOURN-22/24/26）与实际一致；声明路径与落盘路径一致 |

### F-a…F-f

| 硬失败 | 是否发生 | 锚点 |
|---|---|---|
| F-a 修复不成立 | 否 | 卡于 04:24:51Z 出现，距 decision 04:24:27Z 仅 24 秒 |
| F-b 状态分裂 | 否 | 点击 04:25:00Z → approved 04:25:20Z（20 秒内）；卡 `answered` |
| F-c 门没生效 | 否 | 点击返回成功（JOURN-2 `done/approved`） |
| F-d 新谎 | 是 | `broker-state.json` JOURN-2 `artifact=team/accounts/acme-renewal-quote.md`（子任务路径） |
| F-e 阻塞回归 | 否 | `pending_interview=null`；JOURN-24/32 在卡期间推进 |
| F-f 回归 | 否 | 计划卡唯一；lane 路径与落盘一致（`wiki-log.txt` commits `0f9d1af`/`8e8f10c`） |

## §1 逐条判定

### V1 — 父 Issue 的卡出现了

prereg 要求：父 Issue 进 `lifecycle_state=decision` 后 ≤3 分钟内出现 `dedupe_key=decision:<父IssueID>` `status=pending` 卡，`issue_id` 指向父 Issue。

artifacts：`snapshots/0425/broker-state-slice.json` JOURN-2 `lifecycle_state=decision` `updated_at=2026-11-10T04:24:27Z`；`artifacts/broker-state.json` `request-83` `dedupe_key=decision:JOURN-2` `issue_id=JOURN-2` `created_at=2026-11-10T04:24:51Z` `status=answered`。

结论：通过。

### V2 — 卡点名全部交付物

prereg 要求：卡正文包含每条已交付 lane 落地的路径（期望两条）。

artifacts：`snapshots/parent-decision-card.json` question 字段仅 `The deliverable is team/accounts/acme-renewal-quote.md.`；缺 `team/emails/acme-renewal-followup.md`。

结论：失败。卡只点名一条路径。

### V3 — 卡不阻塞

prereg 要求：`blocking` 为空、`required` 为假；`pending_interview` null；卡存在期间其余 open lane 仍在推进。

artifacts：`snapshots/parent-decision-card.json` 无 `blocking`/`required`；`broker-state.json` JOURN-2 `pending_interview=null`；JOURN-24 `updated_at=04:13:03Z`、JOURN-32 `updated_at=04:13:41Z`。

结论：通过。

### V4 — 点击即落地

prereg 要求：点 `approve_decision`（不带 `artifact_path`）≤10 秒内父 Issue `status=done` `lifecycle_state=approved`，卡从 requests 消失，`team/decisions/<IssueID>.md` 有提交。

artifacts：`broker-state.json` `request-83` `answered.answered_at=04:25:00Z` `choice_id=approve_decision`；JOURN-2 `status=done` `lifecycle_state=approved` `updated_at=04:25:20Z`；`wiki-log.txt` commit `002deee` `decision: promote JOURN-2 on approval` 04:25:01Z。

结论：通过。

### V5 — approve 不再 409

prereg 要求：同一动作返回 200；不得出现 J10 的 cannot be approved...

artifacts：`broker-state.json` `request-83` answered；JOURN-2 `done/approved`；无 409 字样。

结论：通过。

### V6 — 不撒谎

prereg 要求：`hollow_completion` 不得为 true；`artifact` 不得被写成某子任务路径（允许为空）。

artifacts：`broker-state.json` JOURN-2 `hollow_completion=null`（通过）；`artifact=team/accounts/acme-renewal-quote.md`（失败，被写成 JOURN-22 的路径）。

结论：部分（hollow 通过，artifact 失败）。

### V7 — 回归不破

prereg 要求：计划卡只一张、`task_not_decomposed` 0 条、lane 数与实际子 Issue 数一致、各 lane 声明路径与落盘路径一致。

artifacts：`broker-state.json` `request-18` `dedupe_key=plan-approval:JOURN-2` 唯一；messages 无 `task_not_decomposed`；子 Issue JOURN-22/24/26 共 3 条；`wiki-log.txt` commits `0f9d1af`（team/accounts/acme-renewal-quote.md）、`8e8f10c`（team/emails/acme-renewal-followup.md）。

结论：通过。

## §2 硬失败与回退

- F-a：未触发（卡出现）。
- F-b：未触发（点击后 20 秒 approved）。
- F-c：未触发（无 409）。
- F-d：触发 —— `broker-state.json` JOURN-2 `artifact=team/accounts/acme-renewal-quote.md`（子任务路径）。回退：父 Issue 的 artifact 字段被写入子任务路径，属于新谎。
- F-e：未触发（pending_interview null）。
- F-f：未触发（计划卡唯一、路径落盘）。

恢复：点击后 JOURN-2 `done/approved`；`wiki-log.txt` commit `002deee` 落盘 decision。

## §3 判据外的新发现

1. 卡正文只点名一条交付物，而父 Issue 有两条（严重：中）。
2. QA lane JOURN-26 误报两份交付物不存在，后被取消（严重：中）。
3. 操作员干预次数：6 次（语言采访、计划卡、JOURN-22 卡、JOURN-24 卡、JOURN-26 取消、父 Issue 留言），超出 prereg §6 的 2 次预算。

## §4 与 Journey 10 的机制差异

- J10：父 Issue 进 decision 无卡（F-I 触发），artifact=null 导致 hasApprovableDeliverable 为假。
- J11：修复后卡出现（request-83），点击 approve 成功，JOURN-2 done/approved。
- J11 新缺陷：卡正文只点名一条路径（V2 失败），artifact 被写成子任务路径（F-d）。
- J10 无 lane 进入 decision；J11 中 JOURN-22/24 均走 review→decision→approved。