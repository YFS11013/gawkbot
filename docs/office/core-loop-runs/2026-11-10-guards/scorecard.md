# Guard exercise scorecard — 2026-11-10，被测 `0464f91`

判分者：独立子代理 `GuardScore`（read-only reviewer），只依据 `prereg.md` 与 `artifacts/`；不采信 `observations.md` 的结论化措辞。409 响应原文在 artifacts 里没有磁盘记录，相应条目已标注「证据来源=操作员记录，非磁盘原始」。

## 0 结论

**一句话结论：** `GUARD-9` 在两条 lane 已交付前提下三道容器守卫均按预期 409 拒止、正路 `submit_for_review` 200 放行、决策卡与聚合关闭正确落盘，G1–G7 全部成立、H-a–H-e 均未命中。

| 编号 | 断言 | 判定 |
|---|---|---|
| G1 | `artifact_path` 撞门必须 409，文案含 `is a container` 与 lane ID | 成立 |
| G2 | `definition.artifact_path` 撞门必须 409，文案含 `is a container` 与 lane ID | 成立 |
| G3 | bot `complete`+`not_required` 必须 409，文案指回 `submit_for_review` 且点明跳过人类 | 成立 |
| G4 | 三次撞门后容器仍 `in_progress`、`artifact=null` 未被污染 | 成立 |
| G5 | `submit_for_review` 必须 200，容器进 `review`、`artifact=null` | 成立 |
| G6 | 卡出现，`dedupe_key=decision:GUARD-9`，正文列两条 lane 路径、不含 `The deliverable is <单一路径>` | 成立 |
| G7 | 人工 `approve_decision` 200；容器 `done/approved`、`artifact=null`、`hollow_completion!=true` | 成立 |
| H-a | 任一撞门返回 200 —— 门没装或条件写错 | 未命中 |
| H-b | 撞门后容器状态被污染 | 未命中 |
| H-c | 正路被误杀（`submit_for_review` 被连坐） | 未命中 |
| H-d | 卡没出现或只点名一条路径 | 未命中 |
| H-e | 卡批准后 `hollow_completion=true` | 未命中 |

## 1 逐条证据

### G1 — `artifact_path` 撞门 409

**判定：成立**（409 响应体证据来源=操作员记录，非磁盘原始；磁盘间接佐证门生效且未污染）

- 409 原文（`observations.md` §1）：`task GUARD-9 is a container: team/accounts/guard-quote.md is GUARD-12's deliverable, not this Issue's own. ...`，含 `is a container`、`GUARD-12` 与被撞路径。**artifacts 下无 409 日志**（`broker-log-tail.txt`、`actions.json`、`broker-state.json` 均无错误记录）。
- 磁盘佐证 1：`artifacts/guards-tasks.json` 的 `GUARD-12` = `status=done`、`lifecycle_state=approved`、`artifact=team/accounts/guard-quote.md`、`verification_result.pass=true`、`detail="wiki artifact team\accounts\guard-quote.md (33 bytes)"`、`checked_at=2026-11-10T13:21:40Z`、`parent_issue_id=GUARD-9`；`broker-state.json` 一致。
- 磁盘佐证 2：`GUARD-9` 终态无 `artifact`/`artifact_not_required`/`hollow_completion` 字段；`actions.json` 中 `GUARD-12 [done]`（action-16，13:21:40Z）之后的下一条容器事件是 `action-18` 13:21:43Z `[review]`，中间没有容器的 `done` 或 `artifact` 写入——V1 没有 200。
- 脚本一致：`artifacts/guard-driver.py` 的 `V1 artifact_path = lane path` 探针与该调用逐字一致。

备注：只有 409 文案时无法区分「容器门」与「存在性门」（`GUARD-1` 同样 409 但文案是 `does not exist in the wiki`）；本次成立依赖 `GUARD-12` 已交付这一前置（见 §4）。

### G2 — `definition.artifact_path` 撞门 409

**判定：成立**（证据来源=操作员记录，非磁盘原始；磁盘间接佐证）

- 409 原文（`observations.md` §1）：`task GUARD-9 is a container: team/emails/guard-email.md is GUARD-14's deliverable, not a file this Issue writes itself. ...`。
- 磁盘佐证：`GUARD-14` = `status=done`、`artifact=team/emails/guard-email.md`、`verification_result.pass=true`、`checked_at=13:21:40Z`、`parent_issue_id=GUARD-9`；`GUARD-9` 的最终 `definition` 无 `artifact_path`（未被污染），`broker-state.json` 一致。
- `actions.json` 无 `define` 成功事件；`guard-driver.py` 的 `V1b definition scope = lane path` 探针与该调用一致。
- `observations.md` §3 的「时窗」旁路（`GUARD-1` 里 `definition.artifact_path` 得 200）在 `GUARD-9` 不适用，因为两条 lane 的 `artifact` 已落盘，门按 `Artifact` 匹配生效。

备注：磁盘无法独立复核 409 文案走的是 `is a container` 分支，只能由「终态未被污染」反推没有 200。

### G3 — bot `complete` + `not_required` 409

**判定：成立**（证据来源=操作员记录，非磁盘原始；磁盘间接佐证）

- 409 原文（`observations.md` §1）：`task GUARD-9 is a container with 2 delivered lane(s): a bot may not close it from here — that would skip the human. Call submit_for_review on it with no artifact_path; ...`。
- 磁盘佐证：`GUARD-9` 终态 `done/approved` 的闭包路径是 `review→approved`（action-18 → request-21 → action-23），不是 `complete+not_required` 那条会留下 `hollow_completion=true` 的直接关闭；对照 `GUARD-1` 的 `hollow_completion=true`、`artifact_not_required="not_required: container (cos, 2026-11-10T13:21:16Z)"`。
- 脚本：`V7 complete + not_required` 探针带 `verification_reason=container`。

备注：该门只在「有已交付 lane 的容器」上适用；`GUARD-1` 同调用 200 且 `hollow=true` 是空容器的契约内行为（见 §3.3）。

### G4 — 三次撞门后容器仍 `in_progress`、`artifact=null`

**判定：成立**

- 中间态直接证据只来自操作员记录（撞门后 `GET /tasks/GUARD-9` 快照）；`artifacts/guards-tasks.json` 与 `broker-state.json` 只保留终态（`done`，`updated_at=13:21:43Z`）。
- 磁盘「未污染」铁证：最终 `GUARD-9` 的 `artifact` 缺失、`artifact_not_required` 缺失、`hollow_completion` 缺失；`actions.json` 全 26 条里没有任何 `GUARD-9` 的 `artifact` 写入事件。
- 时序佐证：`action-16` 13:21:40Z `GUARD-12 [done]` → `action-17` 13:21:40Z `GUARD-14 [done]` →（三个探针，无落盘）→ `action-18` 13:21:43Z `GUARD-9 [review]`。任一探针若是 200，容器会直接 `done`，不会走到 `review`。

备注：判据本身过宽——只断言「未污染」，没断言探针之前「两条 lane 确实 done」。本次成立强依赖 `GUARD-12/14` 已 `done/approved` 且 `pass=true`（`checked_at=13:21:40Z`，磁盘可验），否则 G1–G3 会是空真（`GUARD-1` 就是那样）。

### G5 — `submit_for_review` 200，容器进 `review`、`artifact=null`

**判定：成立**

- 200 本身来源=操作员记录；磁盘强证据：`actions.json` `action-18`（13:21:43Z，`task_updated`，`summary="Guard exercise 2: two-lane pack [review]"`，`actor=cos`）证明容器确实进了 `review`。
- `artifact=null` 磁盘证据：终态无 `artifact`/`artifact_not_required`，且 `action-18` 之后到 `request-21` 创建（13:22:08Z）之间没有 `artifact` 写入。
- 备注：`pipeline_stage=ship`、`review_state=approved` 是关卡之后的终态，中间 `review` 只在 `actions.json` 里留痕，符合 `review→approved` 链。

### G6 — 决策卡出现

**判定：成立**（全磁盘可验）

- `artifacts/requests.json` 的 `request-21`：`kind=approval`、`status=answered`、`dedupe_key=decision:GUARD-9`、`issue_id=GUARD-9`、`created_at=2026-11-10T13:22:08Z`、`updated_at=13:24:36Z`。
- `question` 全文同时含 `This Issue's lanes delivered 2 file(s):`、两条 bullet（`team/accounts/guard-quote.md`、`team/emails/guard-email.md`）与 `The Issue ships no single file of its own`；全文件 `grep` 不含 `The deliverable is`。
- `actions.json` `action-19`（13:22:08Z，`request_created`）与 `action-20`（`request-23`）对应；两张卡之间无覆盖。

### G7 — 人工 `approve_decision` 200；容器 `done/approved`、`artifact=null`、`hollow_completion!=true`

**判定：成立**（200 来源=操作员记录；磁盘终态与事件链强证据）

- `artifacts/requests.json` `request-21.answered = {choice_id=approve_decision, answered_at=2026-11-10T13:24:36Z}`、`status=answered`；`actions.json` `action-23`（13:24:36Z，`request_answered`，`actor=you`）。
- 容器终态（`guards-tasks.json` / `broker-state.json`）：`status=done`、`lifecycle_state=approved`、`review_state=approved`、`pipeline_stage=ship`、`artifact` 缺失、`artifact_not_required` 缺失、`hollow_completion` 缺失（=false，不为 true）。
- 对照 `GUARD-1`（`hollow_completion=true`）证明聚合谓词能区分「空容器」与「有交付的容器」。

## 2 硬失败

**无。** G1–G7 均未被磁盘证据反证，H-a–H-e 均未命中。三处 409 的 HTTP 状态与文案因 artifacts 未落盘而无法独立复核，但没有反向证据（无 200 污染、无 `artifact` 写入、无 `hollow_completion=true`），时序与终态自洽。

## 3 判据外发现

### 3.1 `GUARD-4` 在容器关闭后覆盖 `team/accounts/guard-quote.md`

- `artifacts/broker-log-tail.txt`：`2026/11/10 21:25:40 wiki write-task: task=GUARD-4 slug=cos path=team/accounts/guard-quote.md bytes=1642`（UTC 13:25:40Z，晚于 `GUARD-9` 的批准 13:24:36Z）。
- `guards-tasks.json`：`GUARD-4` = `parent_issue_id=GUARD-1`、`artifact=team/accounts/guard-quote.md`、`verification_result.detail="wiki artifact team\accounts\guard-quote.md (1642 bytes)"`、`checked_at=13:25:57Z`；同路径的 `GUARD-12` 是 `33 bytes / 13:21:40Z`。
- `actions.json` `action-25`（13:25:40Z，`automation`）：「两者都只是 stub，我现在写真正的 quote 文章」→ `action-26`（13:25:57Z，`GUARD-4 [done]`）。
- 含义：`observations.md` §4 已记「两条 lane 声明了同一路径」，属场景遗留；但它证明**已交付 lane 的 artifact 可被另一个声明了同一路径的任务覆盖**，`mode=create` 没有独占校验。是否该禁止需要产品决策（`mode=append_section` 的共享写法是有意的）。

### 3.2 `request-23` —— 模型对空壳场景的悬挂访谈卡

- `requests.json` `request-23`：`kind=interview`、`status=pending`、`from=cos`、`issue_id=GUARD-1`、`due_at=13:52:21Z`，问「两条 lane 在我能读到的任何地方都没有定义」。
- `actions.json` `action-21` 13:22:21Z 是 cos 贴的计划、`action-22` 13:23:42Z 是「访谈卡片超时了，我直接把问题放这里」。
- 含义：与 `GUARD-9` 链无关，属模型对操作员搭的空壳场景的正常反应，未干扰判据。

### 3.3 `GUARD-1` 的 `hollow_completion=true` 是空容器的契约内关闭，不算 H-e

- `guards-tasks.json` `GUARD-1`：`hollow_completion=true`、`artifact_not_required="not_required: container (cos, 2026-11-10T13:21:16Z)"`、`review_state=not_required`。
- `GUARD-6`：`parent_issue_id=GUARD-1`、`status=in_progress`、`artifact` 缺失、`verification_result.pass=false`（`artifact ... is not in the wiki`，13:21:13Z）。
- 两条 lane 从未交付，容器「没有已交付 lane」，三道门按设计不适用；`complete+not_required` 200 且 `hollow=true` 与 `TestEmptyContainerStillClosesFromTheTaskEndpoint` 一致。**不计入 H-e。**

### 3.4 探针的 409 没有落盘

- `actions.json` 26 条全是 `task_created`/`task_updated`/`request_created`/`request_answered`/`automation`，没有 `error`/`rejected`/409 条目；`broker-log-tail.txt` 只有 3 条成功的 `wiki write-task`。
- 含义：被拒的变更不落 action、也不进 broker 日志，导致 G1–G3 只能靠操作员记录。这是可观测性缺口，不是门的问题。

### 3.5 `GET /requests` 视图与状态文件不一致

- `observations.md` §4 记录 `GET /requests` 返回 `requests: []` 而状态文件里有 pending 卡；而 `request-21` 在 `submit_for_review` 之后 25 秒内就创建了（13:21:43Z → 13:22:08Z）。
- 含义：可重现的一致性/可观测性瑕疵；建议后续演练把 `GET /requests` 的原始响应也落盘。

## 4 对判据本身的批评

**G1/G2 过弱且会误判"存在性门"**：只要求文案含 `is a container` 与 lane ID，没要求不含 `does not exist`，也没要求路径精确等于 `GUARD-12/14.artifact`。改法：`G1' = 409 且响应体含 quoted path == GUARD-12.artifact 且不含 "does not exist" 且响应后 GET /tasks/GUARD-9 的 artifact 仍为 null`，并在探针前断言 `GUARD-12/14.verification_result.pass==true`，否则判「未观察」。

**G3 没区分"空容器合法"与"有交付容器非法"**：改法：`G3' = 前置 count(GUARD-12/14 where status==done && artifact!=null)==2 时，必须 409 且文案含 "2 delivered lane(s)" 与 "submit_for_review"，且之后 container.lifecycle_state!=approved`；再单列 `G3-empty` 断言空容器 200。

**G4 最宽松**：只断言「没被污染」，没断言「门曾有机会生效」。改法：`G4' = G4 且 探针前两条 lane 均 done/approved/pass=true 且 artifact 精确匹配，且 broker 日志里两次 write-task 各自落盘（commit 8279a96 / 4ab8eb7）`；否则整体判「未观察」。

**G5 没绑定卡创建的原子性**：改法：`G5' = G5 且 N 秒内出现 dedupe_key==decision:GUARD-9 的 pending 卡，且正文含两条 lane 路径`（本次实际间隔 25 秒，超时应可配置）。

**G6 没强制计数与来源一致**：改法：另需 `question` 含 `lanes delivered 2 file(s):`、两条 bullet 精确等于两条 lane 的 `artifact`（顺序无关）、`dedupe_key` 在 `requests.json` 中唯一。

**G7 `hollow_completion` 以缺省判 false 过弱**：改法：同时要求 `status==done`、`lifecycle_state==approved`、`review_state==approved`、`artifact is null`、`artifact_not_required is null`、`request-21.answered.choice_id==approve_decision`、`actions.json` 里 `request_answered.actor==you`，并显式对比 `GUARD-1` 的 `hollow==true`。

**通用**：要求驱动脚本把三次探针的 HTTP status 与响应体原样落盘（`artifacts/probe-responses.json`），否则 G1–G3 永远是「证据来源=操作员记录」，无法独立复核。
