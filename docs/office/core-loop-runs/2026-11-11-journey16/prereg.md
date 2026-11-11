# Journey 16 prereg — 三条产品决策的端到端验证（lane 卡让位 / 已授予 access 不问 / 覆盖留痕）

写于 2026-11-11，源码 `897da73`，二进制 `c0ff6f76…`（已部署主办公室，`DRIFT: none`）。**本文件先单独 commit 再开跑**（`core-loop-runs/README.md` §1）。

## 1. 为什么要跑这一轮

J15 的三条遗留项已由操作员拍板并实现（`6a3f137` + `66f05e5`，`DESIGN-DECISIONS.md` §9.26），三条都有单测和红证：

| 决策 | 代码位置 | 测试 |
|---|---|---|
| ① 父 Issue 仍开着的 lane 不发自己的决策卡 | `broker_decision_card.go` `raiseDecisionCardLocked` | `TestLaneDecisionCardDefersToItsLiveIssue` |
| ② 任务已结构的 wiki 写权限不再当 `definition-gap` 追问 | `broker_requests_interviews.go` `accessNeedGrantedByWriteScope` | `TestDefinitionGapSkipsAccessTheTaskAlreadyHolds` |
| ③ 覆盖他人已交付页面写 `artifact_overwritten` 审计行（不拦） | `wiki_handlers.go` `recordForeignArtifactOverwrite` | `TestOverwritingAnotherTasksDeliverableLeavesARow` |

单测证明「给定输入必给输出」，证明不了**模型还会不会走到这些输入上**。J15 之所以值得修，是因为模型真的把已授予的权限问了两次、又自己发了两张 lane 卡；本轮要回答的是：在同一模型、同一 prompt 下，这三处噪音是否真的消失，且没有把该有的提醒一起吞掉（决策 ①/② 都可能误伤）。

## 2. 场景（与 J15 逐字相同，便于跨构建比较）

种子 prompt（Issue 正文，`created_by=human`，频道 `cos__human`，owner `cos`）：

> Journey 16: Acme renewal pack（收尾版）
> Acme 续约（VP Ops：Dana）要两份**各自独立**的英文交付物：① 一页续约报价单（独立文件）② 一封跟进邮件（独立文件）。请拆成**两条子任务**，每条交付自己的文件。
> 这个 Issue 自己不写文件——它只是把两件事办完、然后**收尾**：我要看到它进 done。方案/价格/周期、续约调整、联系人都不在办公室里，走采访或直接问我。

**不预告**任何 409、不预告卡、不提 `submit_for_review`、不提 `not_required`、不提本轮改动。

事实采访答法与 J10–J15 相同（死数字）：Acme，40 席位，$18,000/年，2026-12-31 到期，30 天付款；续约报 $21,600/年，14 天有效期；决策人 Dana（VP Ops）。

## 3. 验证项

| 编号 | 断言 | 读哪里 |
|---|---|---|
| V1 | `requests` 里**没有** `dedupe_key=decision:<laneID>`（JOURN-x 两条 lane 均不得出卡）；lane 进入 `decision` 也不出卡 | `broker-state.json:requests[]`、`tasks[].lifecycle_state` |
| V2 | 父 Issue 进 `decision` 后 ≤3 分钟出现 `dedupe_key=decision:<父ID>` 的卡，正文列出**两条** lane 路径 | `requests[].question` |
| V3 | 无 `definition-gap:` 卡；若模型重述「wiki 写入权限（artifact_scope 已授予）」而卡仍未出现，记「决策 ② 生效」 | `requests[]` |
| V4 | 回归：模型若把 lane 路径填进父 Issue 的 `artifact_path` 或从任务端点关容器 → 409 + `guard_refused` 行；且不换另一条 lane 路径重试 | bot 贴出的工具错误、`actions[]` |
| V5 | 人 `approve_decision`（不带 `artifact_path`）→ 200；父 Issue `done/approved`；`artifact=null`；`hollow_completion` 不为 true | `broker-state.json` |
| V6 | 计划卡唯一（`plan-approval:<父ID>` 1 条）；`task_not_decomposed` 0 条；两条 lane 的声明路径与落盘路径逐字一致；wiki 有 `team/decisions/<父ID>.md` | `requests`、wiki `git log` |
| V7 | 无人为覆盖时**没有** `artifact_overwritten` 行（决策 ③ 不得误报） | `actions[]` |
| V8 | 人类需要作答/点击的卡总数 ≤3：事实采访、计划卡、父 Issue 卡（J15 是 4，多出的是两张 lane 卡） | `requests[]` 计数 |

## 4. 失败模式（硬失败）

| 编号 | 含义 |
|---|---|
| F-k | 出现 `decision:<laneID>` 的 lane 卡 → 决策 ① 失效 |
| F-l | 出现 `definition-gap:` 且内容只包含「wiki 写入权限」→ 决策 ② 失效（若还包含其它真实需求，记「② 部分生效」不算硬失败） |
| F-m | 无覆盖行为却出现 `artifact_overwritten` → 决策 ③ 误报 |
| F-a…F-j | 沿用 J15：卡没发 / 只点名一条路径 / `hollow_completion=true` / 守卫拒绝无留痕 / 模型找到第三条关闭路径 等 |

## 5. 干预预算与停表

- 预算：**3 次**（事实采访、计划卡、父 Issue 卡）。任何额外点击记 `observations.md` 并算超预算。
- 停表：连续 20 分钟无新消息、无 active 回合、`requests` 无未答 `decision:` 卡；或累计 90 分钟。
- 停表后归档证据并 `workspace shred --yes journey16`。

## 6. 判分

独立子代理按本文件写 `scorecard.md`；不采信 `observations.md` 的结论化措辞。V1–V3、V7 允许「未观察」（模型没给出触发输入），不得因为「结果正确」倒推成「决策生效」。

## 7. 环境

| 项 | 值 |
|---|---|
| runtime_home | `C:\Users\ireke\.wuphf-spaces\journey16` |
| broker / web | `:7910` / `:7911` |
| provider / model | `opencode` / `opencode-go/longcat-2.5-free`（`WUPHF_OPENCODE_MODEL`） |
| 记忆后端 | markdown（`wiki_ready` 必须 true） |
| members | 1（未走 onboarding；与 J15 同） |
| 启动方式 | 手工 `Start-Process`（`WUPHF_RUNTIME_HOME` + 两端口 + env），**不是** `workspace create` —— `workspace pause --force` 对该 workspace 报 `context deadline exceeded`、`resume` 报 `already running`，记入 observations 判据外发现 |
| 主办公室 | `:7892` 同机运行，watchdog 已改为只重启主办公室进程树（README §6 的修法已落地，两端口同时 200 验证通过） |

## 8. 与 J15 的可比性

场景、种子正文、事实数字、频道、owner、members 数全部相同；差异只有二进制（含三条决策）与 `WUPHF_HEADLESS_MODEL_FALLBACK`（J15 未设，本轮设为 `iceman-contributor-free`，仅当主模型不可达才用）。因此 V8 的卡数、V1/V3 的有无可以逐条对比 J15 的 `request-31/34`（两张 lane 卡）与 `request-19/22`（两次 access 追问）。
