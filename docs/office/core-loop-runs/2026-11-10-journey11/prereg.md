# Journey 11 — 预登记（2026-11-10）

> 目的只有一个：把 Journey 10 的唯一缺陷（父 Issue 进 `decision` 却没有卡、以及人工 approve 被 409 顶回）放到真机上看它是否已经修好。
> 场景与 J7/J8/J9/J10 **逐字相同**，不再考察其它判据；V1–V6 全部围绕「父 Issue 的 decision 卡」。

## 1. 环境

| 项 | 值 |
|---|---|
| 被测提交 | `54ad984`（`fix(team): approve a delivered Issue without inventing an artifact`），构建 `go build -o gawkbot-fix.exe ./cmd/wuphf` |
| 对照 | `52456d3`（J10）——同一场景、同一蓝图、同一模型，唯一差异是本次修复 |
| 隔离办公室 | `journey11`（broker :7910，web :7911） |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/longcat-2.5-free`（创建与重启都必须显式带；J10 prereg §1.1 已记录 `crouton-alpha-free` 被上游撤掉的代价） |
| 蓝图 | `ai-revops`，agents `cos, revops, analyst, ae, reviewer` |
| 记忆后端 | markdown + wiki（默认） |
| onboarding | `POST /onboarding/complete`，`skip_task=true`；随后取消 starter lanes `ai-revops-1..4` |
| 主办公室 | 7891/7892，**不得被本轮任何命令指向** |

## 2. 修复内容（本轮唯一变量）

`internal/team/broker_decision_card.go` + `broker_decision_packet.go`：

- `Broker.childDeliverablesLocked(task)`：父 Issue 已交付子任务落下的文件清单（含 `not_required` 的「ships no file」标注）。
- `hasApprovableDeliverableLocked`：自己的 artifact/not_required **或** 子任务交付物非空 → 可批准。
- 卡正文：父 Issue 自己没有文件时逐条列出各 lane 落地的路径。
- 决策 approve 门（`recordTaskDecisionInternal` (2)）：`effectiveArtifact`、`effectiveNotRequired`、子任务交付物三者全空才算 vapor；`HollowCompletion` 同理。

**没有变**：卡仍然只由 reviewer 收敛的三个出口发、仍然非阻塞（不设 `Blocking`、不动 `pending_interview`）、应答仍走 `RecordTaskDecisionWithComment`。

## 3. 场景

与 J10 §3 逐字相同（标题前缀改为 `Journey 11`）：

```
title:   "Journey 11: Acme 续约报价与跟进"
details: "客户 Acme（40 席位，SaaS）。现行合同：年付 $18,000，2026-12-31 到期，付款 30 天。
          拟续约：年付 $21,600（+20%），条款不变，有效期 14 天。对方决策人 Dana（VP Ops）。
          两件交付物：① 一页报价说明（价格/范围/有效期/付款/决策点）② 48 小时内的跟进邮件草稿。
          只认以上数字，不要引入其它假设。"
```

## 4. 验证项

| 编号 | 判定 | 通过判据 |
|---|---|---|
| V1 | **父 Issue 的卡出现了** | 父 Issue 进 `lifecycle_state=decision` 后 ≤3 分钟内，`requests` 出现 `dedupe_key=decision:<父IssueID>`、`status=pending` 的卡；`Iss.issue_id` 指向父 Issue |
| V2 | **卡点名全部交付物** | 卡正文包含每条已交付 lane 落地的路径（本轮期望两条：`team/accounts/*.md` 与 `team/emails/*.md`），不是只写一个 |
| V3 | **卡不阻塞** | 卡 `blocking` 为空、`required` 为假；`pending_interview` 仍为 null；卡存在期间其余 open lane 仍在推进或有 active 回合 |
| V4 | **点击即落地** | 操作员点 `approve_decision`（**不带** `artifact_path`）：≤10 秒内父 Issue `status=done`、`lifecycle_state=approved`，卡从 `requests` 消失，`team/decisions/<IssueID>.md` 在 `git log` 有提交 |
| V5 | **approve 不再 409** | 同一动作返回 200；不得出现 J10 的 `cannot be approved into done without a delivered artifact` |
| V6 | **不撒谎** | 父 Issue 的 `hollow_completion` 不得为 true；父 Issue 的 `artifact` 字段不得被写成某一个子任务的路径（允许为空） |
| V7 | **回归不破** | 计划卡只一张、`task_not_decomposed` 0 条、`plan-decomposition-check` 的 lane 数与实际子 Issue 数一致、各 lane 声明路径与落盘路径一致 |

## 5. 硬失败

- **F-a（修复不成立）**：父 Issue 进 `decision` 后 ≥10 分钟仍无 `decision:` 卡 → 回到 `DESIGN-DECISIONS.md` §9.22 重新论，不得再声称「聚合交付」已修。
- **F-b（状态分裂）**：点击返回 200 但 2 分钟内 `lifecycle_state` 未到 `approved`，或卡未撤。
- **F-c（门没生效）**：点击返回 409（聚合分支没走到）。
- **F-d（新谎）**：父 Issue 被标 `hollow_completion=true`，或 `artifact` 被写成某个子任务的路径。
- **F-e（阻塞回归）**：卡出现后 `pending_interview` 非空，或 15 分钟内其余 open lane 无任何推进且无 active 回合。
- **F-f（回归）**：计划卡缺席或出现两张；lane 声明路径未落盘（J10 的 D6 修复被弄坏）。

## 6. 人工干预

只允许 **2 次点击**：计划卡 approve（1 次）、父 Issue 的 `approve_decision`（1 次）。均由操作员通过 API 完成并记时间戳；不计其它人工动作。

**判分口径**（`core-loop-runs/README.md` §6）：

- `watchdog_alert` 不按条数判（第 5 条）：它包含按点的唤醒。判据是**叫不动**——带 `No turn could be dispatched` 的条目数，或「每条之后 ≤2 分钟该 owner 有回合或消息」。
- 产物只认盘上文件 + `git log`（第 4 条），`bytes=` 不算证据。
- 「球在人手里」看 `requests` 里未答的 `decision:` 卡（第 3 条）。

## 7. 停表

连续 15 分钟无新消息、无 active 回合、且 `requests` 里没有未答的 `decision:` 卡；或 60 分钟上限。停表后**先**归档 `wiki-log.txt`、`team-tree.txt`、`broker-state.json`、`messages.json`、`broker.log` 尾、关键产物原文到 `artifacts/`，**再** `workspace shred`。

## 8. 判分

`observations.md` 由记录者写（时间戳事实，不写结论）；`scorecard.md` 由**独立**子代理按本文件 V1–V7 与 F-a…F-f 写，必须回到 `artifacts/` 原始证据。
