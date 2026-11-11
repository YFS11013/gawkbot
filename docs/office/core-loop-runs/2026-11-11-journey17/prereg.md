# Journey 17 prereg — D3 修复的端到端验证（采访工具不再因一次轮询失败而整体失败）

写于 2026-11-11，源码 `a8272ba`，二进制 `376955ac23f5c06f…`（含 `21c7799` 的容忍式轮询，已部署主办公室）。**本文件先单独 commit 再开跑**。

## 1. 为什么要跑这一轮

J16 观察到的 D3：模型建了采访卡后连续三次调用 `wuphf-office_human_interview`（01:22:36 / 01:23:49 / 01:25:06），01:26:17 自述「采访卡片工具连续超时」，把同一问题在频道里手工重问了一遍（`msg-6`），随后又把同一件事包成计划卡。人最终答的是最早那张卡——同一问题在人的视野里出现了两次。

机制：轮询用 15 秒超时的共享 HTTP 客户端打 `/interview/answer`，而 broker 当时正在跑**调用方自己的回合**并持锁；一次超时就把整个工具调用判死。`21c7799` 改为「记录错误并继续，直到 30 分钟截止」，单测 `TestHumanInterviewSurvivesATransientPollError` 有红证（回退即 503 报错）。

单测证明的是「503 之后还会继续轮询」，证明不了**模型这一侧的行为是否真的改掉**：如果工具现在会静默等满 30 分钟，模型可能不再重问，也可能因为等待太久自己放弃并另起炉灶。本轮要回答的就是这个。

## 2. 场景（与 J15/J16 逐字相同，便于三方对比）

种子 prompt（Issue 正文，`created_by=human`，频道 `cos__human`，owner `cos`）：

> Journey 17: Acme renewal pack（收尾版）
> Acme 续约（VP Ops：Dana）要两份**各自独立**的英文交付物：① 一页续约报价单（独立文件）② 一封跟进邮件（独立文件）。请拆成**两条子任务**，每条交付自己的文件。
> 这个 Issue 自己不写文件——它只是把两件事办完、然后**收尾**：我要看到它进 done。方案/价格/周期、续约调整、联系人都不在办公室里，走采访或直接问我。

事实采访答法与 J10–J16 相同（死数字）：Acme，40 席位，$18,000/年，2026-12-31 到期，30 天付款；续约报 $21,600/年，14 天有效期；决策人 Dana（VP Ops）。

**本轮的操作员延迟是刻意的**：采访卡出现后**等 4 分钟再答**（J16 里是 4 分 30 秒，恰好跨过三次超时）。只有让等待跨越原来会失败的时间窗，才能区分「工具不再超时」与「人答得快所以没看见超时」。

## 3. 验证项

| 编号 | 断言 | 读哪里 |
|---|---|---|
| V1 | `interview` 类 request 只有 1 张（同一问题不得出现第二张卡） | `requests[].kind=interview` 计数 |
| V2 | 采访卡 `pending` 期间（≥4 分钟），cos **没有**在频道里手工重问同一个问题（不得出现「采访卡片工具连续超时」或等价的重问消息） | `messages[]` 时间窗内的 cos 发言 |
| V3 | 人答 `request-3` 后，模型在同一回合继续（不重启一轮重新提问），并引用所给死数字 | `messages[]`、`actions[]` |
| V4 | 回归：无 lane 决策卡（`decision:JOURN-x` 不存在）；容器卡列全两条 lane 路径 | `requests[]`、`requests[dedupe_key=decision:<父ID>].question` |
| V5 | 回归：容器守卫若被踩到 → 409 + `guard_refused` 行 | `actions[]` |
| V6 | 结局：父 Issue `done/approved`、`artifact=null`、`hollow_completion` 不为 true；两条 lane 的 `artifact` 与 `definition.artifact_path` 逐字一致；wiki 有 `team/decisions/<父ID>.md` | `tasks[]`、wiki `git log` |
| V7 | 人需作答的卡 ≤3（采访、计划、容器） | `requests[]` 中 status=answered |

## 4. 失败模式

| 编号 | 含义 |
|---|---|
| F-n | 采访工具仍然报超时 / 模型仍在频道里重问（D3 修复无效） |
| F-o | 工具不再超时，但模型因为等不到而自行放弃采访、凭空编条款 |
| F-k…F-j | 沿用 J16：lane 卡出现、只点名一条路径、`hollow_completion=true`、守卫拒绝无留痕 等 |

## 5. 干预预算与停表

- 预算：**3 次**（事实采访、计划卡、父 Issue 卡）。
- 停表：连续 20 分钟无新消息、无 active 回合、`requests` 无未答 `decision:` 卡；或累计 60 分钟。
- 停表后归档证据并 `workspace shred --yes journey17`。

## 6. 判分

独立子代理按本文件写 `scorecard.md`；V2 必须给出时间窗内的逐条消息证据，不接受「没看到」。

## 7. 环境

| 项 | 值 |
|---|---|
| runtime_home | `C:\Users\ireke\.wuphf-spaces\journey17` |
| broker / web | `:7910` / `:7911` |
| provider / model | `opencode` / `opencode-go/longcat-2.5-free` |
| fallback | `opencode-go/iceman-contributor-free` |
| skill gate/judge | 与 watchdog 同块（`perplexity/pplx-decider-flash-v1.2` / `iceman-contributor-free`，key 来自 `auth.json`） |
| 记忆后端 | markdown（`wiki_ready=true`） |
| members | 1（未走 onboarding） |
| 启动方式 | 手工 `Start-Process`（`WUPHF_RUNTIME_HOME` + 两端口 + 完整 env + 两条 Redirect）；`workspace create` 先起再按端口杀掉重起（原因见 J16 D1） |
| 主办公室 | `:7892` 同机运行；watchdog 已按端口/进程树杀 |
| 起跑时任务数 | 0 |

## 8. 与 J16 的可比性

场景、种子、死数字、频道、owner、members 数完全相同；差异只有二进制（`21c7799` 的容忍式轮询 + 部署在 `376955ac`）与「操作员等 4 分钟再答」这一条刻意的延迟。因此 V2 的可比对象是 J16 的 `msg-6`（「采访卡片工具连续超时」）。
