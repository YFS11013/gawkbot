# Journey 14 prereg — 把「容器 + 聚合卡」钉死在场景里再跑一次

写于 2026-11-10，被测 `6caf6cf`（与 J13 同一份代码；J13 未改动代码）。

## 1. 为什么要跑这一轮

J13 的结论：场景写「形状你自己定」时，cos 把两份交付物塞进**同一个文件**、**一条 lane 都不拆**，于是容器守卫、聚合谓词、bot 关闭门（V1/V2/V7）全部没有对象。J11/J12 同样措辞下拆过两条 lane，说明这是模型的自由裁量，不是必然。

所以 J14 只改一处：**在 Issue 正文里钉死形状**——两条子任务、两个独立文件。其余（Acme 续约、英文、office 模式、`opencode-go/longcat-2.5-free`、事实采访、计划卡）与 J10–J13 相同。

## 2. 场景

Issue 正文（种子 prompt）：

> Journey 14: Acme renewal pack
> Acme 续约（VP Ops：Dana）需要两份**各自独立**的英文交付物：
> ① 一页续约报价单（独立文件）
> ② 一封跟进邮件（独立文件）
> 必须拆成**两条子任务**，每条交付自己的文件。现有方案/价格/周期、续约调整、联系人都不在办公室里，走采访或直接问我。

事实采访的答法与 J13 相同（死数字）。**不预告**任何 409 或卡。

## 3. 验证项

| 编号 | 断言 | 读哪里 |
|---|---|---|
| V1 | 容器写入被拒：任何带 lane 路径的 `artifact_path` 变更对父 Issue 返回错误，文案含 `is a container` 与该 lane 的 ID | bot 贴出的工具错误、`ledger` |
| V2 | 父 Issue 的 `artifact` 始终为空（没被写成任一 lane 路径） | `broker-state.json` |
| V3 | 父 Issue **进入 `decision`**，且 ≤3 分钟出现 `dedupe_key=decision:<父ID>` 卡（`blocking`/`required` 为 null） | `tasks[].lifecycle_state`、`requests` |
| V4 | 卡正文列出**每一条**已交付 lane 的路径（期望两条），不出现 `The deliverable is <单一路径>` 口径 | 卡 `question` |
| V5 | 点 `approve_decision` 不带 `artifact_path` → 200，父 Issue `done/approved`，`hollow_completion` 不为 true | `broker-state.json` + 本机请求返回 |
| V6 | 回归：计划卡唯一、`task_not_decomposed` 0 条、每条 lane 声明路径与落盘路径逐字一致、wiki 有 `team/decisions/<父ID>.md` | `requests`、`messages`、wiki `git log` |
| V7 | 若模型尝试用任务端点关容器（`complete`/`approve` + `not_required`），返回 **409** 且文案指回 `submit_for_review`；父 Issue 不得因此被关掉 | bot 贴出的错误、`tasks[].status` |

## 4. 失败模式（硬失败）

| 编号 | 含义 |
|---|---|
| F-a | 模型被 409 后**不重试**，父 Issue 停在既无 artifact 也没提交的状态 |
| F-b | 模型换一条 lane 的路径再试（守卫只堵一条路径） |
| F-c | 父 Issue 的 `artifact` 仍被写成某条 lane 的路径（还有第三条写路径） |
| F-d | 父 Issue 进 `decision` 后卡没发（聚合谓词在真机上不成立） |
| F-e | 卡发了但只点名一条路径 |
| F-f | 点击后 409/500，或 `hollow_completion=true` |
| F-g | 计划卡/写范围/路径钉出现回归 |
| F-h | 模型找到**第三条**关闭容器的路径，人类仍然没被问 |
| F-i | 钉死形状后仍然不拆 lane（场景钉死无效，说明「拆不拆」由模型自由裁量且正文约束无效） |

## 5. 干预预算与停表

- 预算：**3 次**（事实采访、计划卡、父 Issue 卡）。额外干预记进 `observations.md` 并算超预算。
- 若再出现 `definition-gap:` 卡（J13 的 request-14 那种重复追问）：记录后由操作员简短作答，算超预算并注明「产品流程逼出来的」。
- 停表：连续 15 分钟无新消息、无 active 回合、`requests` 无未答 `decision:` 卡；或累计 60 分钟。
- 停表后归档并 `workspace shred --yes journey14`。

## 6. 判分

独立子代理按本文件写 `scorecard.md`（不采信 `observations.md` 的结论化措辞）。重点看 V1/V7 的工具错误原文与 V4 的卡正文。

## 7. 已知的环境限制

- 测试基建的 `TestObsidianWatcher` flake 与本轮无关（`§9.23` 已记）。
- 种子 prompt 的措辞与 J10–J13 **不完全相同**（J14 是唯一钉死形状的一轮），这是 J13 的结论直接要求的改动。
