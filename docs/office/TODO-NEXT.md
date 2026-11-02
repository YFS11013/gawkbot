# 待办清单（2026-10-29 19:40 北京）

> 与 `DESIGN-DECISIONS.md` 配套。每条都有验收标准，不用“看起来正常”结案。
> 时区：**UTC 为主，括号内为北京时间**。

## 开放项汇总（2026-11-01 21:00 核对）

所有未关闭条目，逐条给"谁卡着"。凡是需要人拍板的，先列在这里，不要凭印象挑一个做。

| # | 条目 | 需要什么 | 卡在哪 |
|---|---|---|---|
| B | P2 #5 digest 的 `actual` 实际值来源 | **人拍板来源** | 一手（央行/统计局）/ FF 发布后页面 / 新闻标题，成本与准确性成反比 |
| D | ~~P3 #7c Windows 包级测试红~~ | ✅ 2026-11-02 拍板「Windows 一等公民」；判据扩到全仓，`go test ./...` 45 包 0 红（§7c 收口）。遗留：多 workspace spawn 未实现，见 E | — |
| E | P3 #7e 后续：Windows 用 Job Object 收口 | 无，纯工程 | 现在只能杀进程树，工具自派生的进程仍可能逃逸 |
| F | P3 #8 清理：空仓库 `YFS11013/gawkbot-office`、3 个 exe 备份 | 前者需 `gh auth refresh -s delete_repo` | 备份 ~88 MB/个，删前确认不需要回滚点 |
| G | P0 #1 forex-watch 16:30 自动简报 | **等时间** | `next_run=2026-11-02T08:30:00Z`（周一 16:30 北京），现在无法验 |
| I | P3 7g 空壳完成链的 ① 入口分流 / ③ 同题合并 | **人拍板** | ② 已做（done 且无 artifact → `hollow_completion=true`）；①③ 改用户可见流程 |
| H | ~~P0 #0 基金周报（11-07）会被 `wiki_artifact` 校验打回~~ ✅ 已修（`b7231a0`） | — | 校验器现在把 `ra_<16 hex>`（含 `.html` 形式）解析成 `visual-artifacts/<id>.html` 再 stat（`task_verification.go` wiki_artifact 分支，`rich_artifact.go` 新增 `visualArtifactWikiRelPath`/`visualArtifactIDRef`）；测试 `TestRunTaskVerification_WikiArtifact` 增加 3 例。端到端待 11-07 首次触发确认 |
| J | ~~候选场景（独立审阅回合 / 广度研究拆只读子代理 / 数值断言检查单 / 按技能路由模型）~~ 
| K | **`memory_backend=gbrain` 时 wiki 写入全挂** | **人拍板**：补 gbrain 写路径，还是在启动时拒绝 gbrain+wiki 并存 | `ensureWikiWorker` 首行 `if ResolveMemoryBackend() != markdown { return }`，worker 恒为 nil → `/wiki/write*` 一律 503；gbrain MCP 不可达（`initialize: EOF`）时 bot 只能绕开写门禁直接写盘，产物不 commit（`git status` 里是 `??`）。同一个二进制换个父进程环境（PATH 里有 gbrain + embedding key）就会从 markdown 变 gbrain，表现为「重启后 wiki 突然不能写」。本轮实测：未设变量重启 → 503；`WUPHF_MEMORY_BACKEND=markdown` 重启 → 三端点恢复。见 `RUNTIME-API.md` §10 |
**✅ 审阅回合已落地（2026-11-01）**；其余三项仍待人挑 | — | 判据见 `DESIGN-DECISIONS.md` §10 判据 2；来源与推导链见 `RESEARCH-2026-11-01-multi-agent.md`（S3 的「每 PR 2 bug、58% 严重」）。实现与接口见 `RUNTIME-API.md` §1.2：done + 机检通过 → 自动开审阅任务（非作者，`review:<任务ID>` 幂等），交付 `team/reviews/<任务ID>.md`；写授权走 `POST /wiki/write-task`（任务声明的 `artifact_path` 即范围）。实测 `ACMER-2847`→`ACMER-2852`（0/3 通过、2 阻断）、`ACMER-2893`→`ACMER-2899`（新写入口落盘 5551 字节、`verification.pass=true`）。关闭开关 `WUPHF_TASK_REVIEWER=off`。其余候选场景的展开（画面/证据/卡点/优先级）见 `USE-CASES-2026-11-01.md` |

已完成、只在正文留档的：P0 #2 停用门禁、P2 #6 `/notebook/*`、P3 #7 测试包编译、P3 7d/7e 两个根因。

## 已完成 — 工作树回归修复（2026-10-30 22:05）
- `/apps/integrations/call` 路由被误删 → 恢复注册；新增的 `/apps/fetch` 独立注册并带 `requireAuth`。
- `web/package.json` 误删 `remark-gfm`（8+ 文件仍 import）→ 恢复，与 `remark-breaks` 并存。
- `CustomAppFrame` GET 白名单误删 `/apps/integrations/catalog` → 恢复。
- `applyLifecycleStateLocked` 误删 `onLifecycleTransitionLocked`（Decision Packet flush）→ 恢复，保留新增的“终态把 building 草案标 failed”。
- `/health` 误删 `build` 字段（StatusBar / HealthCheckApp 依赖）→ 恢复，与新增 `dispatch` 并存。
- `/apps/fetch` 补 SSRF 硬化：拒绝 URL 内凭据、拒绝 30x 跳转（否则可跳出白名单）。
- `docs/agents/INSTRUCTIONS.md` 断句修复（新增规则另起一条，未吞掉原“Own bugs”条）。
- `internal/team` 测试包 Windows 编不过 → 见 P3 #7。

## 已完成 — 工作流与规则重梳（2026-10-30 20:20）
- `AGENTS.md` 重写为唯一工作流入口：三种任务→三条路径（巡检/排障/变更）、五层判据、四条处置原则、六条硬规则、六条禁止项。
- `RULES.md` 修正过时基线（单轮 10–15 分钟 → 18–60 秒）并指向 `AGENTS.md`。
- `DESIGN-DECISIONS.md` §5 改为“机制清单”，不再重复流程。

## P0-0 — 唤醒/调度链路
- ✅ 2026-10-30 19:30：外部看门狗已上线（`WUPHF-BrokerWatchdog`，每 15 分钟，超期 >10 分钟重启，20 分钟防抖）。
- ✅ 调度器已加心跳日志：`grep "scheduler:" ~/.wuphf/logs/broker.err.log`，正常每 10 分钟一行 `heartbeat`。
  - **日志按进程截断**：`Start-Process -RedirectStandardError` 是覆盖不是追加，重启后文件里只剩新进程的输出。所以重启后 10 分钟内没有 heartbeat 属正常；进程在跑而超过 15 分钟仍无 `loop started` 之后的任何行，才是真断流。
  - 换二进制必须照 watchdog 的启动参数（`-RedirectStandardOutput logs\broker.log -RedirectStandardError logs\broker.err.log` + `WUPHF_OPENCODE_MODEL`）。2026-11-01 手工换二进制时漏了重定向，进程健康但日志 45 分钟全静默，心跳巡检当场失明。
- 每日巡检动作：`tail -3 ~/.wuphf/logs/watchdog.log`（无文件=健康）+ 上面那条心跳 grep。
- ✅ 看门狗 kill/restart 分支已实测（2026-11-01 16:36）：`watchdog.log` 记 `broker unreachable → restarting → restart issued`，3 秒后新进程起来，22 秒后 `scheduler: loop started`，`/health` 200。
- **部署坑（2026-11-01 16:30 实测，会打挂办公室）**：在一次性 bash 调用里 `Stop-Process` + `Move-Item` + `Start-Process`，若该调用在 300s 上限被回收，Windows Job Object 会连**刚启动的 broker** 一起杀掉 → 服务空窗，直到看门狗下一轮（实测空窗 3 分 25 秒）。安全做法二选一：
  1. 只杀不启：`Stop-Process -Id <pid> -Force`，让看门狗按它自己的参数重启（最稳，实测 3 分钟自愈）；
  2. 启动单独一次短调用，确认返回，不与 build/移动混在同一条命令里。
- **2026-11-01 19:00–19:21 全办公室停摆：`opencode-go/beam-free` 本身挂死（不是唤醒链路）**。所有 bot turn `stage=started` 后无任何输出，15 分钟超时后 `status=error detail="exit status 1"`；fund/data-keeper 另有 `database is locked`（同一时刻并发 spawn 的结果）。
  - 定性证据：停掉 broker、清空全部 `opencode.exe` 后单跑 `opencode run --model opencode-go/beam-free "reply OK"` → **90 秒零输出被 KILL**；同一命令换 `--model bear/deepseek-chat` → **7 秒返回 JSON**。二进制 `opencode --version` 正常（1.19.3）。
  - 处置：broker 以 `WUPHF_OPENCODE_MODEL=bear/deepseek-chat` 重启（PID 67688，19:35），turn 恢复（cos 55s/15s `status=ok`）。**beam-free 恢复后要换回来**，否则质量基线变了。
  - 排查顺序教训：先单跑 provider，再怀疑唤醒/队列/DB。`database is locked` 是并发 spawn 的次生现象，不是根因（`~/.local/share/opencode/opencode.db` 已 6.4 GB，值得单独排一次瘦身）。
- 判据：human 消息 2 分钟内日志无 `stage=started`，且消息已在 `broker-state.json` ⇒ 唤醒掉线。
- 处置：重启 broker（命令见附录），重读 token，重发。
- 复发记录：2026-10-30 05:11（空闲 16 小时后首次唤醒失败）。
- ~~待办：定位 notify/dispatch 循环退出的原因（上游级）~~ —— 已定性并修掉（见 P3 7d，`enqueueHeadlessCodexTurnRecord` 丢弃分支漏解锁）；不是上游通知机制问题。
- ✅ 2026-10-30 22:31：找到并修掉代码级根因（入队丢弃分支漏解锁，见 P3 7d），新二进制已部署；重启后 cos turn 12s/62s 正常完成。
- ✅ 2026-11-01 17:34：**核心循环演练跑通**（ACMER-2795，人工建单→define→approve→wiki 工件→机器校验→learning）。四段证据：`definition.defined_at=09:17:59Z`、`verification.pass=true`（2090 字节）、wiki 提交 `learning: pattern/core-loop-drill-2026-11-01`、`scheduler` 里该 follow-up job `status=done` + `closed_at` 已盖章。
- ✅ 2026-11-01 17:58：演练暴露的**同任务中途唤醒被丢弃**已修（见 `DESIGN-DECISIONS.md` 8.8）：approve 落在 planning turn 运行中时唤醒被 `queue-drop` 吞掉，任务空转 60 分钟等 follow-up。现改为挂到活动 turn 的 `Rerun` 槽、回合结束时重放一次；线上 ACMER-2809 无人工催办自启执行并通过校验。

## P0 — 明早必须盯的两次自动运行

### 0. 夜间稳定性 + 09:00 周六基金周报（2026-10-31 09:02 实测）
- broker uptime 10h20m（22:31 重启后未再重启），`scheduler: heartbeat` 每 10 分钟连续，`watchdog.log` 无新记录 → 入队死锁修复后夜间未复发。
- `weekly-saturday-fund-portfolio-ratio-analysis-report` 01:00:03Z 触发（=09:00 北京），fund-invest turn 79.7s `status=ok`，产物 msg-2257，next=11-07。
- **缺口**：该周报只进聊天，无 artifact、无 task → 与 P2 5b 同一问题（routine 直接产出，蒸馏空转）。
- **更新（2026-11-01）**：引擎侧 5b 已落地（`fireRoutineAsTask`），forex routine `5` 同日实测建单成功（`Dispatched as task #ACMER-2779`）；基金周报自 5b 之后**尚未触发**（`last_run=2026-10-31T01:00:03Z`，`next=2026-11-07T01:00:00Z`），所以它的链路仍是未验证状态。
- **预测失败（11-07，见开放项 H）**：该 routine 的交付物是 `visual_artifact_create` 的 `ra_*`，而 5b 给每个 agent_routine 无条件挂 `wiki_artifact + required`，校验只做 `os.Stat(wikiRoot/<artifact_path>)`；payload 第 3 步传的是 `artifact_path=ra_xxx`（id，不是 wiki 相对路径）→ `complete` 会被打回。
- **已修（`b7231a0`，2026-11-01 16:19 部署，PID 36236）**：`wiki_artifact` 校验先解析 `ra_<16 hex>` → `visual-artifacts/<id>.html`；11-07 那次要确认的变成"`complete` 通过且 learning 落库"，不再是"会不会被打回"。

### 1. forex-watch 08:30 UTC（16:30 北京）自动简报
- **事实修正（2026-10-31 10:30）**：该 routine 是 `30 8 * * 1-5`（周一到周五），**周六不跑**；`last_run=2026-10-30T11:15:20Z`、`next_run=2026-11-02T08:30:00Z`。所以今天 16:30 没有自动简报，下一次是周一 11-02 16:30 北京。此前「今天 16:30 要盯」的假设作废。
- **要验证三件事同时成立**：
  1. per-agent 配置含 `mcp: ['wuphf-office']`（MCP 修复未回退）
  2. turn 正常结束，无 `exit status 0xffffffff`
  3. 聊天内容与 `wiki/artifacts/forex/<date>-headline.md` **逐字一致**
- **验收命令**：
  - `grep "bot=forex-watch" ~/.wuphf/logs/headless-codex-latency.log | tail -2`
  - `cat ~/.wuphf/wiki/artifacts/forex/<date>-headline.md`
  - 对比同一时刻 `forex-watch__human` 的最后一条消息
- **失败时**：抓当轮 opencode 原始 stdout（`~/.wuphf/logs/headless-codex-forex-watch.log` 的 `opencode_result` / `error` 行），不要先杀进程。

### 2. ~~market-analyst 07:30 UTC 是否被停用门禁拦住~~ ✅ 已验证（2026-11-01）
- 事实：`GET /scheduler` 中 `market-analyst-daily-direction-rating-after-forex-brief` 为 `enabled=false`，`last_run=2026-10-29T07:30:17Z`，**10-30 与之后没有再点火**。
- 注：`next_run` 仍显示 `2026-10-30T07:30:00Z` —— 停用的 job 不推进 `next_run`，这个字段不能当"将要运行"读，只能看 `last_run` 判断是否真跑过。

## P1 — 需要决定的两件事

### 3. ~~branch-tracer 工具缺文件~~ ✅ 已完成（2026-10-29 19:39）
- 定性：omp-evolve A/B harness 的实验脚手架（分支执行追踪器），opencode 自动加载 `tools/*.ts`，无配置引用 → 可删。
- 处置：`branch-tracer.ts` 与 `branch-tracer-stub.ts` 一并移到 `~/.config/opencode/tools.disabled-20261029/`（可回滚，未硬删）。
- 验证：19:41 cos turn 12 秒成功、`team_*` 工具齐全；`Cannot find module` 最后一条停留在 16:56:54（移动之前）。
- 同族清理：`tools/tsc-analyzer.ts` 已于 21:45 一并移入同一目录（子代理复核：omp-evolve 脚手架）。
- 保留：`tools/probe-bg.ps1`——`.ps1` 不被 opencode 当 tool 加载，属惰性文件。

### 3b. ~~CoS 假转交~~ ✅ 已完成（20:50）
- policy 1 已写入（scope: cos）：必须真派发 + 回执。验证：CoS 新建 `cos__forex-watch` 频道、发 msg-1794、forex 71 秒作答。
- policy 2 已写入（scope: forex-watch / market-analyst）：只许转述当日 artifact，缺数据写「digest 未收录」。
- 残留：forex 复述时把 Core PCE 的前值/预期漏成「未单独列出」（digest 里其实有 0.2% → 0.3%），属保守方向的错误，暂不处理。

### 4. ~~market-analyst 的去留~~ ✅ 选 A，已删成员（2026-11-01）
- 背景：routine 已停用；10-30 07:30 那份简报含三处编造（Cook 反对票 / RBA 今日升息 4.85% / Warsh GDP 表态）。
- 处置：选 A（删除成员，forex 已覆盖事件层）——`POST /office-members {"action":"remove","slug":"market-analyst"}` 返 200，成员现为 `cos,fund-invest,forex-watch,data-keeper`；`market-analyst-daily-direction-rating-after-forex-brief` 早已 `enabled=false`，不再点火。
- 遗留：该成员留下的 routine / DM 频道 slug 仍在调度表与消息历史里（只是不再有 owner），属正常历史数据，不清理。
- 遗留（2026-11-01 巡检新发现）：`GET /channels?type=dm` 里 `human__market-analyst` 有 **3 行同 slug**（members 只剩 `human`，created 10-27 05:39、10-27 05:40、10-31 02:39）。`findChannelLocked` 按 slug 建索引，只有最后一行可被解析，另两行是孤儿；side bar 会列出三条同名死房间。**没有安全的 HTTP 清理路径**：`POST /channels {"action":"remove","slug":...}` 每次只删首个匹配，且会连带删掉该频道的消息/任务/请求（`ACMER-1119`/`ACMER-1150` 的历史就在里面）。要么等有人拍板连历史一起删，要么留给状态修复脚本，本轮不动。
- 顺带确认：`GET /channels` 默认返回 `{"channels": []}` 不是故障——默认只列非 DM 的命名房间，而命名频道已退役（`namedChannelsEnabled()` 为 false，`internal/team/broker_office_channels.go:744-753`）；`?type=dm` 能看到 7 个 DM。

## P2 — 数据缺口（决定“预期差”能不能恢复）

### 5. 已公布事件的实际值来源
- 现状：digest 只有前值/预期，没有 actual；预期差段在 artifact 里是占位。
- 候选来源（按可靠性排序）：
  1. 央行/统计局文本（一手，但每个国家一条抓取逻辑）
  2. FF 发布后页面的 actual 字段（需要新的解析路径，XML 不提供）
  3. 新闻标题里的数字（当前模型就用这个拼，正是单位混用的来源）
- **验收**：digest 里出现 `actual` 字段，且与一手来源逐条对齐 ≥ 3 个事件。

### 5b. ~~简报挂到 team_task（打通蒸馏）~~ ✅ 已完成（2026-11-01）
- 做法（引擎侧，不依赖 CoS）：
  - `internal/team/broker_routine_tasks.go`：`kind=agent_routine` 到点后建 task（`created_by=system`，owner/title/details/channel 取自 job），`task_created` 的分发就是唤醒，**不再发 routine 消息**；`source_signal_id=routine:<slug>:<next_run>` 防重放；建单失败回落旧消息路径并在 Runs 记 `failed`。
  - 同一处给 task 挂 `verification_kind=wiki_artifact`（required）：判定 complete 自报的 `artifact_path` 是 wiki 内非空文件。没有机器校验就没有蒸馏（`task_distill.go` 只蒸馏 verified-done），这一条是"打通"的另一半。
- **验收（实测，两次真跑 forex routine "5"）**：
  - `ACMER-2774`（无校验版）：`done`，`artifact=artifacts/forex/2026-11-01-headline.md`，Runs 记 `Dispatched as task #ACMER-2774`，DM 里没有 `from=routine` 的 routine 消息。
  - `ACMER-2779`（带校验版）：`verification_result={pass:true, detail:"wiki artifact artifacts\forex\2026-11-01.md (2337 bytes)"}`，`git -C ~/.wuphf/wiki log` 出现 `learning: operational/5`（40f046d，作者 forex-watch，task_id=ACMER-2779），`team/learnings/index.jsonl` 新增一条 `source=execution trusted=true`。
  - 该次运行没有 `archivist:` 提交——中文简报标题/正文没有命中已知实体，属数据事实，不是链路断。
- 备注：`learningKeyFromTitle` 把中文标题压成 `5`（ASCII 数字），所以同一条 routine 的每日 learning 在渲染页按 (scope,type,key) 去重成一行，JSONL 每次追加一行。要按天区分得另给 key，当前不做。
- **已知不兼容（巡检 2026-11-01）**：校验器只认 wiki 相对路径（`task_verification.go:195-231` 用 `os.Stat(wikiRoot/<artifact_path>)`），但 `visual_artifact_create` 的交付物以 id `ra_xxx` 被引用（HTML 实际在 `wiki/visual-artifacts/ra_xxx.html`）。payload 里写 `artifact_path=ra_xxx` 的 routine 一律过不了校验。别处已有 `ra_` 解析（`wiki_task_attribution.go:51`），是这一处漏了。受影响：`weekly-saturday-fund-portfolio-ratio-analysis-report`（11-07 首次触发，见开放项 H）。
- 已知失效模式（巡检 2026-11-01 实测）：routine 的 `channel` 已不存在时，建 task 直接 `404 channel not found`，回落的消息也照旧写进那个死房间（消息存储不校验频道注册），于是每天一条 `failed` 运行 + 一次无效唤醒。现在 `channel not found`（`TaskMutationNotFound`）不再回落：直接把本次 fire 记成 `failed` 并在 run 里写明原因，因为回落消息只会写进同一个不存在的房间，还会白唤醒一次负责人；其他建单失败仍保留回落消息。`#general` 已退役，实测 `POST /tasks {channel:"general"}` → 404；已把两条指向 `#general` 的 agent_routine（`check-scheduled-task-status-for-model-snapshot-daily-archive-verification`、`daily-model-snapshot-verification-routine`）停用（可逆，快照流水线已于 10-31 归档）。现存启用的 agent_routine 只剩 `5`（forex DM）与周报（fund DM），都指向活频道。

### 5c. ~~artifact 进索引~~ ✅ 已完成（10-30 05:10）
- 做法：bridge 增加 `git_commit()`；额外输出 `team/forex-brief-<date>.md` 摘要（索引只扫 `team/`）。
- 验收：`git log` 有 `artifact: forex brief 2026-10-30`；`grep forex-brief ~/.wuphf/wiki/index/all.md` 命中。
- 备注：完整表格仍在 `artifacts/`，索引指向摘要文章，双份不同用途。

### 6. ~~`/notebook/catalog` 404（上游 bug）~~ ✅ 已完成（2026-11-01）
- 定性：不是 fork 独有——上游 main 同样没有 `/notebook/*` 的 handler，是**上游 bug**。客户端 `internal/teammcp/context_tools.go` 调 4 个端点（write/catalog/search/promote），broker 一个都没注册。
- 实测损失：`~/.wuphf/logs/headless-codex-*.log` 里 10-24 有 23 条 `/notebook/*` 404；fund-invest 的一条 turn 明确写「notebook capture (backend 404 for several turns)」后放弃，产物标成 "pending on the storage side"。
- 修法（本 fork 自建，无法推上游）：
  - `internal/team/broker_notebook_routes.go`：4 个 `requireAuth` 路由。写只许 `agents/<slug>/notebook/<file>.md`（复用 `validateNotebookWritePath`）；检索按书架隔离；promote 走既有 `requestKnowledgePromotion`，**只起 human 审批卡，从不直接写 team/**。
  - `internal/team/notebook_git.go` + `WikiWorker.EnqueueNotebookNote`：走同一个 drain 循环（单写者不变式），git 提交作者=bot，**不重建 index、不发 `wiki:write`**（私有笔记不属于团队目录）。
  - 顺带修掉一个静默谎言：客户端把 404/405 也标 `retryable:true`，现在归为 `notebook_endpoint_unavailable` 且不可重试。
- 验收：`go test ./internal/team -run TestNotebook -count=1` 9 例全绿（含跨书架写入被拒、catalog 计数、检索不越界、promote 只建卡不落盘、重复 promote 复用同一张卡、无 token 401、错方法 405+Allow）；契约见 `RUNTIME-API.md` §6。

## P3 — 工程债

### 7. ~~`internal/team` 测试包编不过~~ ✅ 已完成（2026-10-30）
- 根因不是「缺 helper」：`newSessionMsgFixture` / `sessionAnswer` / `pollUntil` 定义在 `broker_session_messaging_test.go`，该文件与 `_limits_test.go` 都带 `//go:build darwin || linux`（假 CLI 是 shell 脚本）；`headless_session_typing_test.go` 复用了这些 fixture 却没有同名构建标签，Windows 上独自编译才报 undefined。
- 处置：给 `headless_session_typing_test.go` 补同一构建标签。
- 验收：`go vet ./internal/team/` 干净；`go test ./internal/team -run 'Scheduler|Opencode|Headless' -count=1` 可编译执行。

### 7c. 包级测试在 Windows 的剩余红项（2026-10-30 首次运行）
- 全包 24 红；修掉 3 个真 bug 后仍有 ~21 条，全部是 OS 能力差（非代码回归）：symlink 权限（`MkdirAll` 建链需管理员）、POSIX 0600/chmod 语义、`/Users/me/...` 绝对路径断言、SIGKILL 退出码、box CLI 未安装、临时目录内 git 缺 identity、rename 并发语义、`os.UserHomeDir` 不认 `HOME`。
- 未做：逐条加 `runtime.GOOS == "windows" → t.Skip`。要跑绿得先决定「Windows 是否一等公民」；在此之前只用定向 `-run`。
- 2026-11-01 复测（全包 `go test ./internal/team -count=1`，411s）：**19 条仍红**，清单固定——`TestBoxSigninFlowMintsAndStoresAKey`、`TestOnboardingCompleteMaterializesWiki`、`TestSaveLocked_ConcurrentBrokersSamePathDoNotRace`、`TestDefaultBrokerStatePath_RelativeFallbackWhenHomeMissing`、`TestHandleStudioRunWorkflowExecutesOneDraftAndUpdatesSkill`、`TestHandleStudioRunWorkflowReturnsRateLimitMetadata`、`TestHandleWorkspacesList_MarksActiveWorkspaceWithCanonicalRuntimeHome`、`TestPreflightHeadlessCodexAuthFailsAndPostsSystemMessage`、`TestWriteHeadlessOpencodeMCPConfigConcurrent`、`TestWriteHeadlessOpencodeMCPConfigLogsBaseConfigParseFailure`、`TestSessionCommandsAreExactlyTheThreeObserved`、`TestSessionCommandsCarryNoForbiddenFlag`、`TestSessionCommandEnvDropsTheOfficeAndKeepsThePerson`、`TestChannelPaneLogPaths`、`TestClaudeCommand_WritesPromptFileWithCorrectContent`、`TestClaudeCommand_ErrorSurfaces`、`TestOfficeEvals`、`TestIsTransientProcessKill`、`TestWikiPageMove_CommitFailureRestoresSelfLinkBody`。抽检三类根因：路径分隔符（`ChannelPaneLogPaths` 断言 `.wuphf/logs/x` 前斜杠）、POSIX 绝对路径（`/Users/me/shop`）、Windows 并发 rename `Access is denied`。基线对照：同一份清单在 `c77d978` 的干净 worktree 上完全一致。
- 判据：**未来巡检拿这份清单做差集**——出现清单外的红项才是回归。
- **2026-11-02 复测与修正（本轮）**：
  - `TestOfficeEvals` / `scheduler-truth` 曾被归入"OS 能力差"，**是误判**。它断言「到期 routine 向 `#general` 发一条含 `@eng` 与 prompt 的消息」，而 agent routine 早已改成以任务为唤醒——`broker_routine_tasks.go` 的成功分支明写 `Routine message not posted: the task dispatch is the wake`，所以任何正确实现都必红。实测该 tick 确实建了任务（`OFFICE-1`，owner=`eng`，details 含 prompt），只是没有频道消息。断言改为查这个任务；去掉 `processOnce()` 会红（已反向验证）。`cmd/office-eval` 回到 **147/147**，提交 `2a82152`。
  - `TestWikiPageMove_CommitFailureRestoresSelfLinkBody`（靠 chmod 目录让 commit 失败，Windows 上目录仍可写）与 `TestIsTransientProcessKill/SIGKILL`（本机 git-bash 的 `sh` 会以 MSYS 退出码 2304 结束，不是 SIGKILL 签名）加 Windows skip，各自写明原因：`bff9c74`、`c2bc8f3`。
  - 清单里的两条 `TestObsidianWatcher_*` 属时序 flaky：单独跑 3 次全过，全包并发下红。
  - **教训**：把红项一律归因"OS 能力差"会漏掉真回归，而真回归会一直躲在 19 条噪声里。判据补一条——单独跑仍红、且失败断言描述的行为与代码当前契约不符时，先怀疑契约过时，再怀疑平台。
- **2026-11-02 收口（Windows 一等公民）**：全包 `go test ./internal/team -count=1` → **424s，0 条红**；`cmd/office-eval` **147/147**。
  - 产品修复两处（`1562556`）：
    - **`replaceFile`**：Windows 上 `os.Rename` 只要目标被任何进程打开就报 `Access is denied`（索引器、杀软就是这样打开状态文件的），
      而这是桌面上的常态。只对三种 sharing 错误重试约 0.1 秒，缺目录/权限问题不重试、原错误照常上抛；包内 11 处"写临时文件再改名"全改走它。
      实测：32 个 broker 并发保存同一 state 路径，从 20 个错误到 0。
    - **junction 解析**：`filepath.EvalSymlinks` 在 Windows **不跟随 junction**（实测返回原路径且 `err=nil`），
      workspace 的 canonical 比较因此永远不等。改用 `GetFinalPathNameByHandle` 去掉 `\\?\` 前缀后再比。
  - 测试修复（`12c5cd1`、`ce68b21`）：
    - HOME 隔离 helper `setTestHome`（同时设 `USERPROFILE`；只设 HOME 的用例在 Windows 上会读到真实 `~/.config`、`~/.codex`）。
    - 假 CLI 改成**测试二进制自身**（`WUPHF_FAKE_CLI`，`TestMain` 提前退出）：`/bin/sh` 脚本在 Windows 根本起不来。
    - session 夹具去掉 `/Users/me/shop`；`TestChannelPaneLogPaths` 用 `filepath.Join` 精确比较并 pin 真正的 `WUPHF_RUNTIME_HOME`。
    - 权限断言限定 POSIX；`chmod` 造写失败改为把临时目录指向不存在的路径（两边都失败）。
    - onboarding 测试 pin `WUPHF_MEMORY_BACKEND=markdown`（gbrain 就绪时 worker 不启动），并在用户编辑前 `b.Stop()`（否则排队的骨架写回盖掉编辑）。
    - 相对路径回退同时清 `USERPROFILE`；目录链接回退到 junction（不需要开发者模式/管理员）。
  - 判据改写（见 `OFFICE-RULES.md` R8）：**Windows 上全包必须全绿**，不再维护"已知红项清单"。
  - **2026-11-02 全仓收口**：判据从 `./internal/team` 扩到 `go test ./... -count=1` → **447s，45 个包 ok，0 条红**（此前 16 个包共 65 条红）。
    - 产品修复两处（`cm28`）：`internal/workspaces` 的 flock 在 Windows 是三个空函数，registry 又是 read-modify-write，两个 workspace 同时保存会互相改名失败、丢掉一次更新 —— 改用 `LockFileEx`；临时文件原来是固定 `.tmp`，改成 `os.CreateTemp` 并走 `fsutil.ReplaceFile`。
    - `ReplaceFile`/`WriteFileAtomic` 收敛到 `internal/fsutil`，`internal/team` 的 `replaceFile` 变成一行委派、`atomicWriteFile` 直接调用（`cm27`）。
    - 其余 63 条全部是测试假设，不是产品行为：HOME/`USERPROFILE` 隔离不完整（Windows 的 `os.UserHomeDir` 读 `USERPROFILE`，`config.Path` 另读 `WUPHF_RUNTIME_HOME`，只设 HOME 会让全包共享开发者的真实 `~/.wuphf`，一条测试存的 `Blueprint: "niche-crm"` 改了之后所有 roster 断言看的 manifest）；`/bin/sh` 假 CLI 在 Windows 起不来（改测试二进制自身）；`os.Symlink` 需开发者模式（改能力探测 skip）；NTFS 无 POSIX mode（断言限 POSIX）；JSON 里拼 Windows 路径未转义；`/tmp` 与 `/` 分隔符写死。
    - 两条与平台无关的真错：`approval_context_canonical.txt` 被 git 按 CRLF 检出后与 `\n` 构建的 spec 比较（测试归一化行尾）；skill 使用计数测试要求第二次 +1，但 broker 的「每个 skill 同时只允许一个活跃 skill-run 任务」守卫正在（正确地）去重 —— 测试现在先完成该任务再调用。
    - **遗留产品缺口（不是测试问题）**：`internal/workspaces` 在 Windows 上 `defaultSpawn` 仍返回 `errSpawnNotSupportedOnWindows`，多 workspace 的创建/恢复不可用；`go test ./internal/workspaces` 全绿只说明单 workspace 的注册表/布局/doctor 路径正确。按 R8 这是一等公民判据下的已知缺口，需要 Job Object + 进程组 spawn 才能补（与 §7e 的 P3 #7e 同一批）。
- **危险（2026-11-02 又发生一次，已修根因）**：`go test ./...` 把真实的 `~/.wuphf/config.json` 写成了 `{}`、`onboarded.json` 写成了空骨架，
  于是 UI 把已完成的办公室判成新装、直接弹引导页。数据本身没丢（`team/broker-state.json` 里 4 个成员、77 个任务、500 条消息都在，`.last-good` 同份），
  恢复方式是 `config.json.bak-claude-code-20261024` 合并当前字段 + `onboarded.json` 补 `phase/complete`、`completed_at`、`company_name`（备份在 `~/.wuphf/backups/test-damage-20261102/`）。
  三个写入点：`internal/config` 的 `withTempHome`/`withTempConfig`、`internal/commands` 的 `withTempConfigHome` —— 它们只设了 `HOME`，
  而 `ConfigPath()` 先读 `WUPHF_CONFIG_PATH`、再读 `WUPHF_RUNTIME_HOME`，最后才是 `os.UserHomeDir()`（Windows 读 `USERPROFILE`）。
  现在四个变量一起设。**巡检办法**：跑全仓前后对 `~/.wuphf/config.json`、`onboarded.json` 取 md5，不等就是又漏了隔离。

- **危险（已修）**：只 `t.Setenv("HOME")` 而不设 `WUPHF_RUNTIME_HOME` 的用例，在 Windows 上会读写真实 `~/.wuphf`。`TestIsDeterministicPhase2CEODM` 曾在 22:16 覆写 `~/.wuphf/onboarded.json` 的 `phase`/`completed_at`（已改指临时 home；真实文件已把 phase 恢复成 `complete`，`completed_at` 仍是测试夹具值 `2026-05-28`，日期不可考）。

### 7e. turn 超时不生效 → 8.5h 挂死（2026-10-31 修复）
- 现象：10-30 23:31:14 起三个 cos turn，`first_event_ms=-1`，直到 10-31 08:00:05 才落地 `timed out after 25m0s`，`total_ms≈30.53M`，收尾是 Bun 的 `EPIPE: broken pipe, write`；期间队列被堵，后续 turn `queue_wait_ms` 达 28.4 分钟。
- 根因：`opencode` 解析到 npm shim `opencode.cmd`，`TerminateProcess` 只杀 shim，真正的 Bun 子进程继续持有 stdout 管道，worker 的 `cmd.Wait` 一直等它，直到它自己写失败退出。
- 修法：Windows 侧改 `taskkill /F /T /PID`（杀进程树）+ opencode/codex/claude 三个 runner 设 `cmd.WaitDelay=30s`（孤儿持管道也不能把 Wait 拖过 turn 截止）。
- 回归测试 `TestTerminateHeadlessProcessPIDKillsGrandchild`：旧实现必红（实测 16s 后报 grandchild 存活），新实现 2.3s 通过。
- 同类隐患：`configureHeadlessProcess` 在 Windows 仍是空操作（没有 Job Object），工具自身再派生的进程仍可能逃逸；后续可考虑 Job Object + `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`。

### 7d. 唤醒掉线的代码级根因（2026-10-30 修复）
- `enqueueHeadlessCodexTurnRecord` 的同任务丢弃分支在 `headless.mu` 持有状态下 `return`（该函数无 defer，其余分支都显式 Unlock）。丢一次重复 lead 唤醒 → 锁永久泄漏 → 后续每次入队/`stopHeadlessWorkers` 阻塞，表现为「进程在、调度在、消息不进 turn」。
- 与 P0-0 观察一致（05:11 空闲 16h 后首次唤醒失败）。已修并部署（22:31 重启后 cos turn 正常，queue_wait 46ms）。

### 7b. 三条僵尸任务（更新 10-30）
- ✅ ACMER-1742：plan 已批准（`lifecycle: running`），data-keeper 可收尾。
- 仍存：task-skill-1852 / 1857（in_progress，零消息）。

- ACMER-1742（planning, data-keeper）、task-skill-1852 / 1857（in_progress, cos，零消息）。
- 无操作员取消端点；交给 CoS 关闭可能再触发循环，暂记录。
- 若要清：让 CoS「取消 task-skill-1852/1857，无产出；1742 保留」并观察一轮。

### 7f. 跨 DM 唤醒的回答投不出去（2026-11-01 巡检确认，待拍板）
- 现象：`wake-lead: re-delivering specialist completion from @forex-watch` 把 cos 唤到 `forex-watch__human` 里的任务上；cos 干完活要回帖时，`live-chat-relay-post` 与 `opencode_fallback-post` 双双 `channel access denied`（11:09:34、11:19:02 各一次），正文只留在 `headless-codex-cos.log`，人看不到。cos 自己也说了一句「@cos 在此 DM 频道无写入权限」。
- 规则本身没错：DM 只认两名参与者（AUDIT-2026-10-31 §2.5 已记为既定规则），拒绝是预期；问题是**回答没有出口**，同时每次唤醒刷两条 error。
- 要选一条：
  - A. 唤醒改成落在负责人自己的 DM（`cos__human`），正文里带任务指针——最小改动，回答一定写得进；
  - B. 内部 actor 允许以「转述」身份写入被唤起的 DM（消息带 `from=cos, relayed`），要动访问控制；
  - C. 回答作为任务评论/线程写回任务所在频道（需要评论模型，不只是消息）。
- 顺带：失败重投不降噪（审计建议 N 次后停），本轮未动。
- 同类已修（2026-11-01）：app 提案审批后建 App Builder 任务也把空频道默认成 `#general`（`broker_apps_proposal.go`）。现在按「提案所在房间 → 建者（app-builder）的主场 → 记日志放弃」解析；提案房间在审批前退役（`#general` 退役、成员删除带走 DM）时不再把任务和错误消息塞进死房间。新增 `TestProposeAppApprovalSurvivesARetiredRoom`。

### 7g. 空壳完成（2026-11-01 诊断；② 已落地，①/③ 待拍板）
- 复发三次同款：`ACMER-2757`（10-31，跑满 10 分钟超时）→ `ACMER-2764`（stuck 重试，补答后关单）→ `ACMER-2785`（11-01，24 秒关单，正文只有一句「已完成」）。
- 链路：**问题被建成了 Issue**（`created_by=human`、`task_type=issue`、`title==details`，没有 Definition）→ 完成零门槛（`spec {}`）→ 一次 `team_task complete` 即 done → 完成 hook 广播 `delivered`，频道里没有正文 → 下一轮把它当「已答」的证据（`ses_e6aa388e4ffe65ZEW1dTv7z8qQ` 的推理原文：*"The answer was already posted in chat."*）。
- 结构原因：artifact 门禁只对**有 Definition** 的任务生效（`broker_tasks_mutation_service.go` 的 artifact gate），人工/对话框建的 Issue 全是 `Definition=nil`；`wiki_artifact` 强制校验只挂在 agent_routine 生成的任务上。
- **已做（②，`broker_types.go` + `broker_tasks_mutation_service.go`）**：任务转到 done 且 `artifact` 为空时，打 `hollow_completion=true` 上线（`teamTask.HollowCompletion`，wire 键 `hollow_completion`）。不改完成语义，只让这类完成可被计数。测试 `TestCompleteWithoutArtifactIsMarkedHollow`（空完成=true、带 artifact=false、字段上线）。
- 巡检口径：`curl /tasks?all_channels=true&include_done=true` 里 `status=done && (hollow_completion || artifact 为空)` 的条数。历史任务没有该字段，等价于 `artifact` 为空——2026-11-01 实测 52 个 done 里 47 个如此。
- 部署冒烟样例：`ACMER-2789`（`smoke: hollow_completion 验证`）是上线验证用的，不是第五次事故，数数时忽略它。二进制 16:30 构建、16:36 起由看门狗拉起（PID 39216，`provider_model=opencode-go/beam-free`）。
- **待拍板**：① 疑问句不进 Issue（UI 入口分流或 `task_type=question`）；③ 同题窗口期内复述+引用旧任务而不是新建。两条都改用户可见流程，未动。

### 7h. 有 shell 的 bot 可以直接写 wiki 文件，绕过所有写入门禁（2026-11-01 发现）
- 现象：`/wiki/write-task` 明确拒绝了 `team/reviews/…` 以外的路径（`wiki write-task refused` 有日志），但 cos 仍把 `reviews/ACMER-2832.md`、`reviews/ACMER-2847.md` 写进了 `~/.wuphf/wiki/`，git 里以 `recover from crashed write` / `external edit` 形式出现。
- 原因：bot 的 shell 工具能直接对 `~/.wuphf/wiki/` 落盘，`/notebook/*`、`/wiki/write*`、promote 审批这些门禁只在 API 层生效。
- 影响面：任何 bot 都能改 `team/` 正文、伪造 learning；`wiki_artifact` 校验也因此可能被"先落盘再 complete"绕过（本轮的 `ACMER-2852` 就是先被忘接写成 `reviews/` 再由人把文件挪进 `team/reviews/` 才关单）。
- 未做：给 wiki 目录加文件级监控并告警（`external edit` 提交已存在，可作为探针），或收紧 bot 的 shell 工作目录。属安全边界改造，需拍板。

### 8. 清理
- ✅ **scheduler 死 job 已清（2026-11-01 16:55）**：`task_follow_up` / `request_follow_up` / `recheck` 这类一次性 job 在实体终结后只是被关掉（`status=done`、`next_run` 清空），从来没人删，一天就攒到 145 条，把 160 条里真正活着的 9 条埋掉。
  - 机制：boot 时 `pruneRetiredSchedulerJobs` 丢弃"已关闭 + 无 next_run + 关闭超过 10 分钟"的 job，并连同其 runs/activity/revisions 环形缓冲一起删（`scheduler_lifecycle.go`）。
  - **坑一**：`LastRun` 不能当关闭时间——boot 会为每个任务重跑 `scheduleTaskLifecycleLocked`，把已关闭 job 的 `LastRun` 刷成当前时刻，于是"旧"永远变"新"。改为在**首次关闭时**打 `ClosedAt`，按它算年龄（`4ae3bbe`）。
  - **坑二**：这次升级前写下的行没有 `ClosedAt`，`LastRun` 又不可信 → 视为 Legacy，一次性丢弃（`f98fa4b`）。
  - 结果：**161 → 18**（7 个系统 cron + 8 个 `agent_routine` + 1 个 `one_workflow` + 2 条 market-analyst 的 recheck）；`/scheduler/5/runs` 仍保留 9 次历史，任务表 55 条未受影响。备份在 `~/.wuphf/backups/scheduler-20261101T164540.json` 与 `broker-state-…json`。
  - 判据：`GET /scheduler` 的 job 数应稳定在"活着的调度 + 最近 10 分钟内关闭的一次性 job"，不再随任务数线性增长。
- `YFS11013/gawkbot-office`（空仓库，创建后被 fork 方案取代）：删除需 `delete_repo` scope → `gh auth refresh -h github.com -s delete_repo && gh repo delete YFS11013/gawkbot-office --yes`，或网页删。
- `C:\Users\ireke\code\gawkbot\gawkbot-fix.exe.bak-20261029`（回滚点，确认稳定后再删）
- `C:\Users\ireke\code\gawkbot\gawkbot-fix.exe.old-20261030`（10-30 22:27 前的二进制，含入队死锁；仅在需要复现旧故障时保留）
- `C:\Users\ireke\code\gawkbot\gawkbot-fix.exe~`（更早的备份）

## 附：当前运行参数（避免下次再猜）

| 项 | 值 |
|---|---|
| 启动命令 | `gawkbot-fix.exe --broker-port 7892 --provider opencode --no-open` |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/beam-free`（用户级环境变量，已 `setx` 持久化）；兜底 `bear/deepseek-chat`。
**2026-11-01 19:35 起运行态是 `bear/deepseek-chat`**（beam-free 挂死，见 P0-0） |
| broker token | `%TEMP%\wuphf-broker-token-7892`，**每次重启都变** |
| 桥 | Windows 计划任务跑 `forex-bridge-run.vbs`，每 6 小时；健康时无 `wiki/team/alerts.md` |
| 简报产物 | `~/.wuphf/wiki/artifacts/forex/<date>.md` + `<date>-headline.md` |
| 常规检查 | `curl -s 127.0.0.1:7892/health`（含 model 与 members） |

## 9. 期望态 vs 运行态（2026-10-31 定案）

结论：**git 只放期望态与契约，运行时仍是唯一权威**；两者靠脚本对照，不靠人记得。

已做：
- `docs/office/expected-scheduler.json` — standing job（系统 cron / bot routine / workflow job）+ 计划任务的期望值、依赖文件、日志新鲜度阈值。per-task follow-up 与 recheck 类 job 不列（按任务生命周期增删，列进去会永远漂移）。
- `scripts/office/check-office-drift.ps1` — 只读比对，输出越界项，exit 1。已实测三种探针会红：参数漂移、清单外新 job、计划任务日志停更；当前实跑 `DRIFT: none`。
- `OFFICE-RULES.md` §1 巡检第 8 项改为「必须跑该脚本」。

待办：
1. ~~**自动化**：把 drift 检查挂进 watchdog~~ —— **已做（2026-10-31）**：`~/.wuphf/data/broker-watchdog.ps1`（计划任务 `WUPHF-BrokerWatchdog`，15 分钟）新增 `Invoke-DriftCheck`，`-DriftOnly` 可单跑；备份 `broker-watchdog.ps1.bak-20261031`。关键设计：
   - 计划任务用的是 **Windows PowerShell 5.1**（`powershell -File`，不是 `pwsh`）。5.1 解码子进程的中文会变乱码，所以**告警与去重全在 repo 脚本里（PS7）**，watchdog 只读首行 ASCII 状态（`DRIFT: none|cleared|<n>`）与退出码。
   - **drift 从不重启 broker**（重启只属于 liveness）；新违规集按内容 SHA256 追加到 `wiki/team/alerts.md` 一次，恢复时删状态文件；`watchdog.log` 只在「首次检出」与「恢复」各写一行，持续期间静默。
   - 已实测（5.1 与 7 两个宿主）：干净静默、强制违规告警一次且中文正常、重复运行不重复告警、恢复后状态清除。
2. ~~**入口裁剪**~~ —— **已做（2026-10-31）**：`docs/agents/INSTRUCTIONS.md` 641 行 / 31.0 KB → **118 行 / 6.2 KB**，只留硬规则（Working Rules、Git/PR、Quality Bar、Wuphf Commands、Landing、Lint）+ 首屏「按任务读哪份」路由表。
   - 散文（Nex Context、Triangulation、Verification agents、When to use which、Screenshots、Storybook、Multi-round review、Sub-agent dispatch contract、共享树 12 节、Worktree parallelism、Demo+oracle）原样迁到 `docs/agents/shared-tree-lessons.md`（544 行 / 26.4 KB），分 A 基座 / B Wuphf 仪式 / C 共享工作树三组。
   - 校验：原文件 511 条非空行中，仅 21 行未出现在拆分后（全部属于被重写的 Landing Changes 段），其余逐行保留；29 个章节标题各自只落在一个文件里。
   - 顺带修掉一处规则冲突：旧入口写「commit to main 并直接 push origin」（原作者仓库规则），对本 fork 是错的（`push.default nothing` + origin 无写权限），改为对齐 `OFFICE-RULES.md` §6 的 `mine`-only 规则。
   - 路由表就是索引，没有另建 `docs/office/INDEX.md` —— 多一份要被记住的文件就多一处腐烂点。
3. **policy 全文本**不入 git（每次 PATCH 生成新 revision + id）。repo 只记意图与 id，生效与否一律 `GET` 验证。
4. ~~**空壳 workflow 收尾**~~ —— scheduler job 无 DELETE 端点，保持 disabled + 清单 `requires_file` 把关。
   skill `daily-opencode-model-snapshot` 于 **2026-11-01 重新 archive 并验证落盘**：10-31 那次归档只改了内存/state，wiki 文件没重写（mtime 停在 10-29、frontmatter 仍 `active`），broker 重启后被 scanner 按文件重新导入 → 复活成 active。重做后 `team/skills/daily-opencode-model-snapshot.md` 的 `metadata.wuphf.status: archived`（提交 `0062abe`），`/skills` 回到 3 个 active。验收命令见 `RUNTIME-API.md` §7。
   **教训**：归档/禁用这类"看起来生效"的操作，必须读回落盘产物（frontmatter、文件 mtime、git log），否则重启就是一次回滚。
