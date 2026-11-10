# Journey 12 评分卡（独立评分）

被测 commit `4c53083`。证据仅来自 raw artifacts（`broker-state.json`、`messages.json`、`wiki-log.txt`、`broker-log.txt`、`log-latency.txt`）与代码 `internal/team/broker_tasks_mutation_service.go`；`observations.md` 仅作线索，结论全部以 artifact 为准。

## §0 总览

### V1–V6

| 编号 | 判定 | 锚点 |
|---|---|---|
| V1 | 部分 | `broker-state.json` tasks `JOURN-1` 无 `artifact` 写入 lane 路径；`messages` msg-42 cos 自述「父任务是容器，不能引用子任务的产物」；代码 `broker_tasks_mutation_service.go:1434-1445` 守卫文案 `task %s is a container: %s is %s's deliverable`。未在 artifacts 中找到守卫 409 的原始返回文本，仅能从 msg-42 推断守卫曾触发。 |
| V2 | 通过 | `broker-state.json` `JOURN-1`：无 `artifact` 字段（等价 null），`artifact_not_required="not_required: Container issue: its two lanes JOURN-13 (team/accounts/acme-renewal-quote.md) and JOURN-15 (team/emails/acme-renewal-followup.md) already hold the deliverables. (cos, 2026-11-10T08:00:39Z)"`。父 Issue 的 `artifact` 未被写成 lane 路径。 |
| V3 | 失败 | `broker-state.json` requests 全集：dedupe_key 仅 `plan-approval:JOURN-1`（07:38:16Z）与两个 `knowledge-promote:*`（07:46:21Z/07:46:23Z）；**无** `decision:JOURN-1`。父 Issue 在 08:00:39Z 终态时没有任何未答 decision 卡。 |
| V4 | 未观察 | 从未生成 `decision:JOURN-1` 卡，故卡正文不存在。要成立需：父 Issue 进 `decision` 后 ≤3 分钟出现 `dedupe_key=decision:JOURN-1` 卡且 `blocking`/`required` 为 null，正文列出两条 lane 路径。 |
| V5 | 失败 | `broker-state.json` `JOURN-1`：`status=done` `lifecycle_state=approved`，但 `hollow_completion=true`。prereg 要求 `hollow_completion` 不为 true。无人工点击 `approve_decision`（无 `decision:JOURN-1` request）。 |
| V6 | 部分 | 计划卡唯一（`plan-approval:JOURN-1` 一条，`broker-state.json` requests）；`task_not_decomposed` 无证据；lane 路径与落盘一致：`JOURN-13.artifact=team/accounts/acme-renewal-quote.md` 对应 wiki commit `d9a7c79`；`JOURN-15.artifact=team/emails/acme-renewal-followup.md` 对应 `f044319`。**但** wiki-log.txt 中无 `team/decisions/JOURN-1.md` 提交。 |

### F-a…F-g

| 编号 | 是否发生 | 锚点 |
|---|---|---|
| F-a | 发生 | 父 Issue 08:00:39Z 被 cos 自己关闭（`broker-state.json` `JOURN-1.status=done`，无 `decision:JOURN-1` request，无人工点击）；`hollow_completion=true`。模型未走 submit_for_review 路径，直接 complete+not_required。 |
| F-b | 未发生 | `JOURN-1.artifact` 未被写成任一 lane 路径（`broker-state.json` 无 artifact 字段）；msg-42 仅复述守卫结论，无「换另一条 lane 路径重试」的证据。 |
| F-c | 未发生 | 同上，`JOURN-1` 无 `artifact` 字段。 |
| F-d | 发生 | 父 Issue artifact 为空时未发卡：requests 中无 `decision:JOURN-1`（`broker-state.json`）。 |
| F-e | 不适用/未观察 | 无父 Issue 决策卡，无法评估单/双路径。 |
| F-f | 发生 | 点击路径未发生，但 `hollow_completion=true` 已记录（`broker-state.json` `JOURN-1`）；prereg F-f 包含 `hollow_completion=true`。 |
| F-g | 未发生 | 计划卡唯一；wiki 路径逐字一致（`d9a7c79`/`f044319`）；无 `task_not_decomposed` 证据。 |

## §1 逐条判定

### V1 容器写入被拒
prereg 要求：`submit_for_review`（或任何带 `artifact_path` 的变更）对父 Issue 使用任一 lane 路径时返回错误，文案含 `is a container` 与该 lane 的 ID。
artifact 显示：`broker-state.json` `JOURN-1` 无 `artifact` 字段；`messages` msg-42（08:00:39Z）cos 原话「父任务是容器，不能引用子任务的产物——用 not_required 重新关闭」。代码 `broker_tasks_mutation_service.go:1433-1445` 守卫：`if childID, ok := b.childOwningArtifactLocked(task, artifactPath); ok` → 返回 `TaskMutationConflict` 文案 `task %s is a container: %s is %s's deliverable`。证据链支持守卫触发过，但 artifacts 中未保存 409 原始返回文本。判定：**部分**。

### V2 父 Issue artifact 为空
prereg 要求：模型重试后父 Issue `artifact` 为空。
artifact 显示：`broker-state.json` `JOURN-1` 无 `artifact` 字段；`artifact_not_required` 如上。判定：**通过**。

### V3 父 Issue 进 decision 后 ≤3 分钟出现 decision 卡
prereg 要求：`dedupe_key=decision:<父ID>` 卡，`blocking`/`required` 为 null。
artifact 显示：requests 列表 dedupe_key 仅 `plan-approval:JOURN-1` 与两个 `knowledge-promote:*`；无 `decision:JOURN-1`。`JOURN-1` 08:00:39Z 直接 done。判定：**失败**。

### V4 卡正文列出两条 lane 路径
prereg 要求：卡正文列出每一条已交付 lane 的路径（期望两条），且不出现单文件口径。
artifact 显示：无 `decision:JOURN-1` 卡，正文不存在。判定：**未观察**。要成立需先有 V3 的卡。

### V5 点 approve_decision 不带 artifact_path → 200，done/approved，hollow_completion 不为 true
prereg 要求：点击后 200，父 Issue done/approved，`hollow_completion` 不为 true。
artifact 显示：`broker-state.json` `JOURN-1`：`status=done` `lifecycle_state=approved` `hollow_completion=true`；无人工点击记录（无 `decision:JOURN-1` request）。判定：**失败**（hollow_completion=true）。

### V6 回归
prereg 要求：计划卡唯一、`task_not_decomposed` 0 条、每条 lane 声明路径与落盘路径逐字一致、wiki 有 `team/decisions/<父ID>.md`。
artifact 显示：计划卡 `plan-approval:JOURN-1` 仅一条（`broker-state.json` requests）；lane 路径 `JOURN-13.artifact=team/accounts/acme-renewal-quote.md` 对应 wiki commit `d9a7c79`（07:55:28Z）；`JOURN-15.artifact=team/emails/acme-renewal-followup.md` 对应 `f044319`（07:55:28Z）；wiki-log.txt 中无 `team/decisions/JOURN-1.md` 提交。判定：**部分**。

## §2 硬失败与回退

- F-a 发生：父 Issue 被 cos 自己关闭，无决策卡、无人工点击，`hollow_completion=true`（`broker-state.json` `JOURN-1`，08:00:39Z）。
- F-d 发生：父 Issue artifact 为空时未发卡（requests 无 `decision:JOURN-1`）。
- F-f 发生：`hollow_completion=true` 已记录。
- 回退：无回退动作；cos 在被拒后改用 `complete` + `verification=not_required` 关闭父 Issue（msg-42；`artifact_not_required` 08:00:39Z）。
- 干预预算：prereg 预算 3 次（语言采访、计划卡、父 Issue 卡）。实际干预 4 次：request-4 答复（07:39:30Z）、request-8 批准（07:40:13Z）、request-21 批准（07:55:28Z）、request-23 批准（07:55:28Z）。**超预算 1 次**。其中 request-4 与 request-8 是 prereg 预算内的两类；request-21/23 是知识提升卡，因父 Issue 决策卡从未出现、lane 无法经决策卡闭环，cos 转而走知识提升路径，故这两张卡由产品流程本身（无决策卡）迫使产生，而非操作员急躁。

## §3 判据外的新发现

1. **父 Issue 自关闭路径允许 terminalize 容器任务**（严重：高）。`broker_tasks_mutation_service.go` 在 artifact 赋值守卫（:1434-1445，`childOwningArtifactLocked`）只堵了带 `artifact_path` 的写路径；同一函数后续 `complete`/`verification=not_required` 路径（:1520-1529 记录 `not_required`，:1565-1578 当时允许）让 cos 以 `not_required: Container issue...` 关闭父 Issue，`hollow_completion=true`。代码注释（:1565-1571）明确承认「this path lets any actor terminalize a task. No decision card was ever raised」。判定：**产品缺陷（code 允许了这条路径）**，非纯模型行为——模型只是在守卫拒绝 artifact_path 后选择了 code 提供的另一条合法路径 `complete+not_required`。

2. **lane 自身从 review→decision 时无决策卡**（严重：中）。`JOURN-13`/`JOURN-15` 07:47:56Z 进 `decision`，但 requests 中无对应 `decision:` 卡；lane `artifact` 当时仍为空（知识提升未完成），谓词判「没什么可批」。证据：`broker-state.json` tasks `JOURN-13`/`JOURN-15` `lifecycle_state=approved` `updated_at=07:59:38Z`，requests 无 `decision:JOURN-13/15`。

3. **lane 无 reviewer**（严重：低）。`broker.log`：`task review: no reviewer available for JOURN-13/JOURN-15 (owner "cos")`。lane 最终由 cos 自己标通过。

## §4 与 Journey 11 的机制差异

- J11：父 Issue 卡出现且可点，但聚合谓词未被真机锻炼（cos 把 lane 路径盖到父 Issue artifact，卡片命中「自己已有 artifact」分支）。
- J12：写入侧守卫生效（父 Issue artifact 未被写成 lane 路径），但聚合分支仍未被锻炼——因为父 Issue 走了 `complete+not_required` 自关闭路径，从未进入 decision，决策卡从未生成。J12 的新失败点从「聚合谓词」转移到「容器任务的 terminalize 路径」。
- J12 lane 无 QA lane（J11 有），且 lane 无 reviewer（`broker.log`）。
- J12 两条 lane 最终 artifact 路径与 wiki 提交逐字一致（`d9a7c79`/`f044319`），J11 的 hollow_completion 问题在 lane 层未复发。

## 勘误（操作员补注，不改动上面的独立判定）

- §3.1 引用的代码注释（「this path lets any actor terminalize a task」）是**本轮修复时**写的，不在被测 commit `4c53083` 里。`4c53083` 的事实是：`MutateTask` 的 `complete`/`approve` 对容器没有任何门，`HollowCompletion` 只看 `artifact` 不看子任务，所以「cos 用 `complete` + `verification=not_required` 关掉父 Issue」是被测代码允许的路径——结论（产品缺陷、非纯模型行为）不变，证据要换成「该 commit 没有这道门」。
- §3.2 与 §4 提到「父 Issue 07:47:56Z 进 decision」：artifact 不支持。`JOURN-1` 的 `updated_at` 在 08:00:39Z 之前是 07:45:35Z（`in_progress/running`），它从未进入 `decision`；07:47 进 `decision` 的是两条 lane。
