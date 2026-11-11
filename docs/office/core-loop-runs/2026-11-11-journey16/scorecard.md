# Journey 16 scorecard — 2026-11-11，被测 `897da73` / 二进制 `c0ff6f76…`

独立子代理按 prereg `a4b3b2f` 判分；不采信 `observations.md` 的结论化措辞。所有引用取自 `artifacts/`。409 原文未落盘，守卫拒绝以 `actions[].summary` 为准（与 J15 同一口径）。

## 判据结果

| 编号 | 判定 | 证据 |
|---|---|---|
| V1 | 成立 | `artifacts/requests.json` 全集（request-3/7/22/25/35）无 `dedupe_key=decision:JOURN-18` 或 `decision:JOURN-20`；JOURN-18/JOURN-20 `lifecycle_state=approved`（`artifacts/tasks.json:JOURN-18.lifecycle_state`、`JOURN-20.lifecycle_state`）从未对应 lane 卡；F-k 未触发。 |
| V2 | 成立 | `artifacts/requests.json[id=request-35].question` 逐字含两条路径："This Issue's lanes delivered 2 file(s):\n\n- team/accounts/acme-renewal-quote.md\n- team/emails/acme-renewal-followup.md"；`requests[id=request-35].created_at=2026-11-11T01:34:32Z`，父 Issue 进 review 为 `actions[id=action-33] created_at=2026-11-11T01:34:27Z`（间隔 5s ≤3 min）。 |
| V3 | 不成立（② 部分生效，非硬失败） | `artifacts/requests.json[id=request-22].question` 与 `[id=request-25].question` 仅含 "success criterion: \"公司名与发件人使用占位符 [COMPANY NAME] / [SENDER NAME]\""，是占位符检测器命中，不是 wiki 写入权限/access 追问；`artifacts/tasks.json:JOURN-18.definition.access_needed` 与 `JOURN-20` 均为空，`JOURN-1.definition.access_needed=["无外部系统访问需求；续约条款已由人类直接提供"]`；因此决策 ② 的触发面（access 追问）本轮未出现，② 无输入可证，但占位符检测器仍发了两张 definition-gap 卡。 |
| V4 | 成立 | `artifacts/actions.json[id=action-30].summary`="container artifact: team/accounts/acme-renewal-quote.md is JOURN-18's deliverable"（related_id=JOURN-1）；`action-32.summary`="container close: cos tried to close it with 2 delivered lane(s)"（related_id=JOURN-1）；`guard_refusals.json` 同两条；两次拒绝后未出现换另一条 lane 路径重试的 guard_refused（仅 action-30/32 两条）。 |
| V5 | 成立 | `artifacts/broker-state.json:tasks[id=JOURN-1].status=done`、`lifecycle_state=approved`、`artifact=null`；`requests[id=request-35].answered.choice_id=approve_decision`、`answered_at=2026-11-11T01:36:16Z`；`JOURN-1.sessionReport.metadata=null`（无 hollow_completion）；`actions[id=action-37]` request_answered。 |
| V6 | 成立 | 计划卡唯一：`requests[]` 中 `dedupe_key=plan-approval:JOURN-1` 仅 request-7 一条；`task_not_decomposed` 0 条（actions 无该 kind）；lane 声明与落盘逐字一致：`JOURN-18.artifact=team/accounts/acme-renewal-quote.md` vs `definition.artifact_path=team/accounts/acme-renewal-quote.md`；`JOURN-20.artifact=team/emails/acme-renewal-followup.md` vs `definition.artifact_path=team/emails/acme-renewal-followup.md`（`artifacts/tasks.json`）；wiki 决策页：`artifacts/wiki-git-log.txt` 含 `99ba193 decision: promote JOURN-1 on approval` 且 `artifacts/decision-JOURN-1.md` 存在。 |
| V7 | 成立（未误报） | `artifacts/actions.json` 无 `kind=artifact_overwritten` 行（observations §1.1 计数 `artifact_overwritten 0`）；`artifacts/guard_refusals.json` 仅两条 guard_refused，均非覆盖。 |
| V8 | 成立 | 人类实际作答的卡：request-3（interview, answered.custom_text）、request-7（approval, choice_id=approve）、request-35（approval, choice_id=approve_decision）= 3 条；request-22/25 为 canceled 且无 `answered` 字段（人未作答）。≤3，J15 是 4（多出两张 lane 卡 request-31/34）。 |

## 硬失败

| 编号 | 是否命中 | 证据 |
|---|---|---|
| F-k | 否 | 全 `requests[]` 无 `dedupe_key=decision:JOURN-18`/`decision:JOURN-20`；JOURN-18/20 进 done 未出 lane 决策卡。 |
| F-l | 否（部分生效） | definition-gap 卡存在（request-22/25）但内容仅为占位符检测器命中，不包含 wiki 写入权限/access 追问；按 prereg F-l 定义，不算硬失败。 |
| F-m | 否 | 无覆盖行为且无 `artifact_overwritten` 行（actions 计数 0）。 |
| F-a | 否 | 409 后未停滞：action-30 (01:33:56) → action-33 "[review]" (01:34:27) → request-35 (01:34:32) → 人批 (01:36:16) → JOURN-1 done/approved (01:36:03)。 |
| F-b | 否 | 仅一次 container artifact 尝试（team/accounts/acme-renewal-quote.md），无换另一条 lane 路径再试。 |
| F-c | 否 | `tasks[id=JOURN-1].artifact=null`（`artifacts/tasks.json` 与 broker-state 一致），未被写成任一 lane 路径。 |
| F-d | 否 | 父 Issue 进 review (01:34:27) 后 5s 即出现 request-35 decision:JOURN-1。 |
| F-e | 否 | request-35.question 列出 2 条路径，非单条。 |
| F-f | 否 | 人批 request-35 返回 200（action-37），sessionReport.metadata=null 无 hollow_completion。 |
| F-g | 否 | 计划卡仅 request-7 一条；JOURN-18/20 artifact 与 artifact_path 逐字一致；wiki-git-log 有 team/decisions/JOURN-1.md。 |
| F-h | 否 | 容器关闭旁路被 action-32 拦住后，模型改走 submit_for_review → request-35 → 人批，无第三条关闭路径成功。 |
| F-i | 否 | 已拆两条子任务 JOURN-18/JOURN-20（action-15/16 task_created）。 |
| F-j | 否 | 两次守卫拒绝均在 actions/guard_refusals 留有 guard_refused 行，可观测性完好。 |

## 与 Journey 15 的对比

- 卡数：J15 人类作答 4（事实采访 + 计划卡 + 两张 lane 卡 + 父 Issue 卡，实际 4 类含 lane 卡）→ J16 3（事实采访 + 计划卡 + 父 Issue 卡），lane 卡消失，V1/V8 达成。J15 的 request-31/34（两张 lane decision 卡）在 J16 未出现。
- 守卫：J15 有 action-36（container artifact）与 action-39（container close）两条 guard_refused；J16 同样有 action-30（container artifact）与 action-32（container close）两条，守卫仍被模型自发踩到（F-a…F-j 全否）。
- 定义缺口形状：J15 为两次 access 追问（request-19/22，wiki 写入权限）；J16 变为占位符检测器命中（request-22/25，[COMPANY NAME]/[SENDER NAME]），access_needed 为空，说明决策 ② 生效于其触发面（access），但占位符检测器构成新的定义缺口来源（D2）。
- 交付：两轮均 done/approved、artifact=null、无 hollow_completion、wiki 有 decision-JOURN-1.md，可比。

## 结论

本轮证明：决策 ①（父 Issue 开着的 lane 不发自己的决策卡）生效——JOURN-18/20 进 decision 未出 lane 卡，V1 成立，人类卡数从 4 降到 3。决策 ② 对 access 追问生效（access_needed 为空、无 access 类 definition-gap），但占位符检测器仍对 success_criteria 中故意的占位符发卡，属于新触发面，不是 ② 的失效。决策 ③ 无覆盖行为且无 artifact_overwritten 行，V7 成立，未误报。容器守卫（artifact/close）仍被模型自发踩到并留痕，F-a…F-j 全否；父 Issue 最终 done/approved、artifact=null、无 hollow_completion。V3 严格按 prereg §6 不得倒推为「决策生效」，故标不成立（占位符卡未消失）。

## 证据强度说明

409 原文未落盘（broker.err.log 仅 warn 级），守卫拒绝以 `actions[]` 的 summary 为准，与 J15 同一口径；未独立复核工具返回值。所有字段取自 `artifacts/`（requests.json、actions.json、guard_refusals.json、tasks.json、broker-state.json、wiki-git-log.txt、decision-JOURN-1.md）。observations.md 的事实行可采信，结论化措辞未采信。V3 的「不成立」基于占位符卡客观存在，不依赖 observations 的结论。