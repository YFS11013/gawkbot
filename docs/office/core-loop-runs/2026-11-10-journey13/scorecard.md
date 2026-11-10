# Journey 13 评分卡

## §0 总览

### V1–V7

| 编号 | 判定 | 锚点 |
|---|---|---|
| V1 | 未观察 | broker-state.json tasks：仅 JOURN-1，无 children；无 is a container 错误原文 |
| V2 | 失败 | broker-state.json tasks[0].artifact = "team/accounts/acme-renewal.md"（updated_at 09:12:50Z） |
| V3 | 部分 | lifecycle_state 最终 "approved"；requests 中 dedupe_key="decision:JOURN-1" = request-22（created_at 09:06:17Z） |
| V4 | 失败 | request-22.question："The deliverable is team/accounts/acme-renewal.md."（单一路径口径） |
| V5 | 通过 | request-22.answered.choice_id="approve_decision" answered_at 09:12:33Z；status="done"，lifecycle="approved"，hollow_completion 非 true |
| V6 | 部分 | 计划卡唯一（request-9 dedupe_key="plan-approval:JOURN-1"）；tasks 仅 JOURN-1；wiki team/decisions/JOURN-1.md（29c8463） |
| V7 | 未观察 | 无 complete/approve + not_required 尝试的 409 原文 |

### F-a…F-h

| 编号 | 是否发生 | 锚点 |
|---|---|---|
| F-a | 否 | 父 Issue 最终 status="done"，lifecycle="approved" |
| F-b | 否 | 无换 lane 路径重试记录 |
| F-c | 是 | broker-state.json tasks[0].artifact = "team/accounts/acme-renewal.md" |
| F-d | 否 | request-22（decision:JOURN-1）已发出 09:06:17Z |
| F-e | 是 | request-22.question 仅点名 team/accounts/acme-renewal.md 一条路径 |
| F-f | 否 | approve_decision → status done/approved，hollow_completion 非 true |
| F-g | 否 | 计划卡唯一；definition.artifact_path 与 artifact 一致 |
| F-h | 否 | 未发现第三条关闭路径；人类被 request-22 询问 |

## §1 逐条判定

- V1：prereg 要求容器写入被拒含 is a container。artifact：broker-state.json tasks 仅 JOURN-1 无 children；无拒绝原文。未观察——需先存在 ≥1 child lane，cos 对父 Issue 提交带该 lane 路径的 artifact_path，broker 返回 409 含 is a container 与 lane ID。本 run 无 container。
- V2：prereg 要求父 Issue artifact 始终为空。artifact：broker-state.json tasks[0] "artifact": "team/accounts/acme-renewal.md"（updated_at 09:12:50Z）。失败。
- V3：prereg 要求父 Issue 进入 decision 且 ≤3 分钟出现 decision 卡。artifact：lifecycle_state 最终 "approved"；request-22 dedupe_key="decision:JOURN-1" created_at 09:06:17Z；msg-21 09:06:17Z to_state "decision"。部分。
- V4：prereg 要求卡正文列出每一条已交付 lane 路径（期望两条），不出现单文件口径。artifact：request-22.question 原文 "The deliverable is team/accounts/acme-renewal.md."。失败——仅一条路径；无 children 故 childDeliverablesLocked 为空，走 own != "" 分支。
- V5：prereg 要求 approve_decision 不带 artifact_path → 200，done/approved，hollow_completion 非 true。artifact：request-22.answered.choice_id="approve_decision" answered_at 09:12:33Z；status="done"，lifecycle="approved"；无 hollow_completion true。通过。
- V6：prereg 要求计划卡唯一、task_not_decomposed 0、路径钉一致、wiki 有 decisions 文件。artifact：request-9 dedupe_key="plan-approval:JOURN-1" 唯一；tasks 仅 JOURN-1；definition.artifact_path = artifact = "team/accounts/acme-renewal.md"；wiki team/decisions/JOURN-1.md commit 29c8463。部分（无 lane 可比对）。
- V7：prereg 要求模型用任务端点关容器返回 409 指回 submit_for_review。artifact：无对应 409 原文。未观察。

## §2 硬失败与回退

干预计数：request-4（08:56:05Z）、request-9（08:58:36Z）、request-14（09:05:26Z）、request-22（09:06:17Z）= 4 次，超出预算 3 次 1 次。request-14（dedupe_key="definition-gap:JOURN-1"）question 为 "access needed: 用户已提供的合同事实…"，而 request-4 已于 09:01:09Z 提供全部合同事实（answered.custom_text）。故第 4 次干预由产品 definition-gap 卡重复追问已给事实造成。

## §3 判据外的新发现

(a) definition-gap 卡重复事实：severity 高。证据 request-14.question "access needed: 用户已提供的合同事实（方案/价格/周期/续约调整/联系人）" 与 request-4.answered.custom_text（09:01:09Z）重复。代码路径：broker_tasks_mutation_service.go task definition / access_needed 处理 → 卡 question 生成。判定：code-permitted 的重复追问（模型/流程把已答事实重新列为 access_needed）。

(b) 两交付物 Issue 塌缩为单文件：severity 中。证据 msg-20（09:06:11Z）cos 自述两份交付物写入同一 team/accounts/acme-renewal.md；definition.deliverables 两节均指向同一文件；request-22 单文件口径。代码路径：broker_decision_card.go raiseDecisionCardLocked → childDeliverablesLocked 仅遍历 ParentIssueID==task.ID 的 children，本 run 无 children → lanes 空 → 走 own != "" 分支输出 "The deliverable is %s."。判定：model choice（cos 选单文件、未分解）+ code-permitted（broker 允许无 children 的单 Issue 完成）。

(c) 无 container → V1/V7 不可测：severity 低。代码路径：childOwningArtifactLocked 与 hasApprovableDeliverableLocked（broker_decision_card.go）仅对有 children 的父 Issue 生效。

## §4 与 Journey 11/12 的机制差异

- J11：cos 把单个 child lane 路径盖成父 Issue artifact；J13：无 child lane，父 Issue 自身 artifact = 单文件路径，卡片同样命中 own != "" 分支——表面相同，根因不同（J11 lane 路径复用，J13 无分解）。
- J12：cos 用 complete + verification=not_required 自关父 Issue，hollow_completion=true，无卡；J13：6caf6cf 堵住该路径后，父 Issue 走 submit_for_review → decision → request-22 → human approve，hollow_completion 非 true。
- J13 新增 definition-gap 卡重复追问已给事实（request-14），J11/J12 未出现。
- 聚合谓词（§9.22）在 J13 未被真机锻炼：无 children，childDeliverablesLocked 恒空。
EOF
title: "Journey 13 scorecard"

## 勘误（操作员补注，不改动上面的独立判定）

- **V2 应为「不适用」而非「失败」。** V2 的判据是「artifact 没被写成任一 lane 路径」。本轮没有任何 lane（`tasks` 只有 `JOURN-1`），`team/accounts/acme-renewal.md` 是这个 Issue 自己的交付物，不是某条 lane 的路径。判「失败」是把 V2 误读成「artifact 必须为空」。
- **V3 应为「通过」。** 最终 `lifecycle_state=approved` 是点击之后的状态；`msg-21`（09:06:17Z）记录 `to_state decision`，`request-22` 同秒创建（`broker-state.json`），「进 decision 后 ≤3 分钟出现 decision 卡」成立。
- **§3(a) 的归因要修正。** `raiseDefinitionGapInterviewLocked`（`internal/team/broker_requests_interviews.go:528-545`）只读 `task.Definition` 的 placeholder 与 `AccessNeeded`；它不会去读 `request-4` 的答案。卡上那句「access needed: 用户已提供的合同事实（…）」是 cos 在自己的 `action=define`（msg-18/19「我先定义任务契约」）里写进 `AccessNeeded` 的——**模型把已拿到的事实写成了「还缺的访问」**，不是 broker 重复追问。结论（severity 高、code-permitted）不变，根因是模型的 define 内容。
