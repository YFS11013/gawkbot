# 待办清单（2026-10-29 19:40 北京）

> 与 `DESIGN-DECISIONS.md` 配套。每条都有验收标准，不用“看起来正常”结案。
> 时区：**UTC 为主，括号内为北京时间**。

## 开放项汇总（2026-11-01 21:00 核对）

所有未关闭条目，逐条给"谁卡着"。凡是需要人拍板的，先列在这里，不要凭印象挑一个做。

| # | 条目 | 需要什么 | 卡在哪 |
|---|---|---|---|
| B | P2 #5 digest 的 `actual` 实际值来源 | ✅ **已完成（2026-11-08，选一手来源）** | 桥直连 BLS v1 API 取官方指数算 m/m，与发布表逐条核对（CPI 0.6 / Core CPI 0.3 / PPI 1.3 / Core PPI 0.7）；见 `RUNTIME-API.md` §13.7 与 §5 |
| D | ~~P3 #7c Windows 包级测试红~~ | ✅ 2026-11-02 拍板「Windows 一等公民」；判据扩到全仓，`go test ./...` 45 包 0 红（§7c 收口）。多 workspace spawn 已在 Windows 实现（`spawn_common.go` + `spawn_windows.go`，DETACHED_PROCESS 起子进程），见 §7c 收口条目 | — |
| E | ~~P3 #7e 后续：Windows 用 Job Object 收口~~ ✅ 2026-11-03 | — | 见 §7e「同类隐患已收口」：`startHeadlessProcess` 把每个 headless 子进程塞进 `KILL_ON_JOB_CLOSE` 的 job，`terminateHeadlessProcessPID` 先 `taskkill /T` 再 `TerminateJobObject` |
| F | P3 #8 清理：空仓库 `YFS11013/gawkbot-office`、3 个 exe 备份 | **exe 已删（2026-11-08）**；仓库仍需 `gh auth refresh -s delete_repo` | 删了 `code/gawkbot/gawkbot-new.exe`（84.3 MB, 11-01）与 `Downloads/gawkbot_0.240.1_windows_amd64/gawkbot.exe`（64.5 MB, 10-23），保留最近一个 `code/gawkbot/gawkbot-fix.exe`（跑着的那个）。空仓库未动 |
| G | P0 #1 forex-watch 16:30 自动简报 | **等时间** | `next_run=2026-11-02T08:30:00Z`（周一 16:30 北京），现在无法验 |
| I | P3 7g 空壳完成链的 ① 入口分流 / ③ 同题合并 | ✅ **① 与 ③ 均已收口（2026-11-08）** | ② done 且无 artifact → `hollow_completion=true`；① `/tasks/new` 表单对「Details 为空 + 疑问句标题」先拦下，给「Ask in chat」（发到办公室 DM）与「File it anyway」两个出口；③ 由 J2-6 的「已交付拒绝 + allow_duplicate」覆盖，不再另做窗口期复用 |
| H | ~~P0 #0 基金周报（11-07）会被 `wiki_artifact` 校验打回~~ ✅ 已修（`b7231a0`） | — | 校验器现在把 `ra_<16 hex>`（含 `.html` 形式）解析成 `visual-artifacts/<id>.html` 再 stat（`task_verification.go` wiki_artifact 分支，`rich_artifact.go` 新增 `visualArtifactWikiRelPath`/`visualArtifactIDRef`）；测试 `TestRunTaskVerification_WikiArtifact` 增加 3 例。端到端待 11-07 首次触发确认 |
| J | ~~候选场景（独立审阅回合 / 广度研究拆只读子代理 / 数值断言检查单 / 按技能路由模型）~~ 
| L | **核心循环 Journey 1 首次实测**（人给一条带 Goal/Deliverables/Success criteria 的任务，逐段按 rubric 记分） | ✅ **2026-11-07 已跑（6/10，v1 基线 5/10）** | 结果见 `DESIGN-DECISIONS.md` §9.7；证据在 `core-loop-runs/2026-11-07/`（预注册 `prereg.md`、观察 `observations.md`、评分卡 `scorecard.md`、快照 `snapshots/`）。F2–F6 未命中；三个结构性问题：①人给任务 → wiki 的授权往返（`artifact_path` 为空 + composer 消息是 `system`）②回合结束未收口时无短周期重投（65 分钟空转）③子任务 bot 不能向父频道报告（403，阻塞信息不可见）。下一轮按 §9.7 的 P0/P1/P2 修完再跑 Journey 2 |
| K | **`memory_backend=gbrain` 时 wiki 写入全挂** | ✅ **2026-11-08 拍板「解耦 wiki worker 与 memory backend」并已修**（`ensureWikiWorker` 不再看 backend；新增 `/health.wiki_ready`；见 `RUNTIME-API.md` §14.17；实机：gbrain 后端探针 `/health.wiki_ready=true`、`/wiki/write` 200 且 commit `b444658` 作者 `cos`；同轮补上提示词那半——`prompt_builder.go` 的 `markdownMemory` 拆出 `wikiAvailable`，否则 worker 起来了而提示词仍写「没有知识图谱」；再补上工具面那半——broker 按 worker 状态发布 `WUPHF_WIKI_AVAILABLE`，`teammcp` 据此在 gbrain 下同时注册 `team_wiki_*` 与 `team_memory_*`（MCP 是独立进程，看不到 broker 的 worker），否则提示词会指名不存在的工具；见 §14.17 两个表格） | `ensureWikiWorker` 首行 `if ResolveMemoryBackend() != markdown { return }`，worker 恒为 nil → `/wiki/write*` 一律 503；gbrain MCP 不可达（`initialize: EOF`）时 bot 只能绕开写门禁直接写盘，产物不 commit（`git status` 里是 `??`）。同一个二进制换个父进程环境（PATH 里有 gbrain + embedding key）就会从 markdown 变 gbrain，表现为「重启后 wiki 突然不能写」。本轮实测：未设变量重启 → 503；`WUPHF_MEMORY_BACKEND=markdown` 重启 → 三端点恢复。见 `RUNTIME-API.md` §10 |
**✅ 审阅回合已落地（2026-11-01）**；其余三项仍待人挑 | — | 判据见 `DESIGN-DECISIONS.md` §10 判据 2；来源与推导链见 `RESEARCH-2026-11-01-multi-agent.md`（S3 的「每 PR 2 bug、58% 严重」）。实现与接口见 `RUNTIME-API.md` §1.2：done + 机检通过 → 自动开审阅任务（非作者，`review:<任务ID>` 幂等），交付 `team/reviews/<任务ID>.md`；写授权走 `POST /wiki/write-task`（任务声明的 `artifact_path` 即范围）。实测 `ACMER-2847`→`ACMER-2852`（0/3 通过、2 阻断）、`ACMER-2893`→`ACMER-2899`（新写入口落盘 5551 字节、`verification.pass=true`）。关闭开关 `WUPHF_TASK_REVIEWER=off`。其余候选场景的展开（画面/证据/卡点/优先级）见 `USE-CASES-2026-11-01.md` |

已完成、只在正文留档的：P0 #2 停用门禁、P2 #6 `/notebook/*`、P3 #7 测试包编译、P3 7d/7e 两个根因。

## 核心循环 Journey 2 暴露的四个缺陷（2026-11-07，待排期）

来源：`core-loop-runs/2026-11-08-journey2/scorecard.md` §4。五条都有人可见后果，按严重度排。**J2-1 / J2-3 / J2-6 已于 2026-11-08 修复**，复现与测试见下方「2026-11-08 复核」。

| # | 缺陷 | 现象（实测） | 代码位置 | 修复判据 |
|---|---|---|---|---|
| J2-1 | **done 与产物脱钩** | `NEX-100/102` 与重做后的 `NEX-148/150` 全 `done + hollow_completion=true`；系统已识别「无产物」，仍然放行并让父 Issue 等它 | `internal/team/broker_tasks_mutation_service.go:1346`（`artifact_path` 为空直接跳过校验）、`task_completion_hook.go:43` | `done` 必须要么带已校验的 `artifact_path`，要么显式 `verification=not_required` + 理由；hollow 不能让父 Issue 结算 |
| J2-2 | **交付通道与审批通道互锁** | `cos__human` 有 pending 审批卡时 `live-chat-relay-post-error: request pending in this channel`，cos 回合 `status=ok final_chars=0`；答一张卡后同频道立刻恢复（`request-44/46` 仍 pending） | `internal/team/headless_live_chat_relay.go`（闸门）、`broker_requests*.go`（卡片来源） | 审批卡不阻塞非审批类投递；或回合结束时把「未投递的正文」落盘并告诉人卡在哪 |
| J2-3 | **取消不撤销已排的派活** | `NEX-95` 取消（`{"changed":true}`）后 `10:57:04Z` 回到 `in_progress`、`10:57:10Z` 起回合（11 次工具调用）；触发者尚未定位（`scheduler.json` 只剩 `task_follow_up`，无 `task_retry`） | `internal/team/broker_scheduler.go`（`cancelSupersededTaskJobsLocked` 只关同名 slug）、`broker_tasks_lifecycle.go`（cancel 分支） | 复现并指认触发者；cancel 终态后不存在任何指向该 task 的 `task_follow_up`/`recheck`/`task_retry` job，且状态不再倒退 |
| J2-4 | **recheck 只报警不派活** | `19:11:54 dispatched 1 due job(s)` → 只写 `signals` 一条 `task_stalled`，无回合、无消息；人可见的最后一条停在 `11:08:39Z`，任务挂到停表 | `internal/team/scheduler.go:278-345`（`processTaskJob` 非 blocked 分支只 `CreateWatchdogAlert`+`RecordAction`+`deliverTask`） | recheck 对「有 owner、非 blocked、无 active 回合」的任务必须派一次回合（或升格为 `task_retry`），并把结果写成人可见消息 |
| J2-6 | **重做已交付的活** | 审批放行后 `11:21:58Z` 在同一父 Issue 下建 `NEX-148/150`，而 `NEX-100/102` 早已 `done`；四条新任务同样 `hollow_completion=true` | `broker_tasks_lifecycle.go`（`findReusableTaskLocked` 跳过终态）、`broker_tasks_subissue_guards.go`（兄弟去重跳过终态兄弟） | 同题已交付的活再 create 不再产生新 id：拒绝并点名已有任务，或显式 `allow_duplicate` |

补充事实（同一轮）：办公室被放行后**重做已完成的任务**（`NEX-148/150` 对应已 done 的 `NEX-100/102`）——重投路径不查同题任务是否已完成，去重只覆盖同一 active 回合。**2026-11-08 已修（J2-6），见下方复核。**

### 2026-11-08 复核（模型切换后）

- **J2-2 已修（`87e1d9a`）**：闸门改为只停人自己的发言。复现（`request-217` blocking 卡在 `cos__human`）：回合 `final_chars=1013`，relay 两次 flush 与 fallback 全部 409，连“我被卡住了”那句也被吞。修复后在同样三张 blocking 卡（`request-44/46/217`）待答的状态下，回合正文成功投递 `msg-229`；人（或冒名 `from=you` 的 broker 令牌）仍回 409。测试：`TestBlockingRequestGateIsChannelScoped`、`TestBlockingRequestGateLetsBotReportsThrough`、`TestBlockingRequestGateLetsBotHTTPPostsThrough`。
- **J2-4 已修（`fix(team): escalate a refused watchdog wake instead of going silent`）**：`deliverTask` / `deliverTaskNotification` / `sendTaskUpdate` 全部改为返回「这次唤醒到底派出去没有」，`processTaskJob` 在投递路径存在却拒绝时不再静默——`escalateRefusedWake` 往任务频道写一条 `kind=watchdog_alert` 系统消息（`Watchdog: @<owner> still needs to move <title> in #<channel>. No turn could be dispatched to wake @<owner> — retrying shortly.`）并调 `ScheduleNoTraceRetry` 重投。`deliverTask == nil`（eval 夹具、裸 fixture）不升级，避免噪音。实测（journey2，`WUPHF_TASK_FOLLOWUP_MINUTES=2`）：cos 被审批卡挡住唤醒时 `10:43:00Z`、`10:46:40Z` 两轮都出现该消息（旧行为是纯静默）；同一轮里 owner 可达的 job 正常唤醒且**不**出现该消息。测试：`TestSchedulerRefusedWakePostsVisibleEscalation`、`TestSchedulerDeliveredWakeStaysQuiet`、`TestSendTaskUpdateReportsLifecycleGate`、`TestDeliverTaskNotificationReportsDispatchedWake`。遗留：消息只说「唤不到」，不区分「被审批卡挡住」还是「owner 不在编制内」，措辞可再细化。
- **J2-1 已修（`e0da217`）**：`done` 门槛去掉「只绑有 Definition 的任务」。现在**所有任务**要么带**已存在**的 `artifact_path`（存在性检查也不再对无 Definition 任务早退——J2 里 `NEX-104` 指向从不存在的 `team/playbooks/…md` 被放行），要么显式 `verification=not_required` + `verification_reason`（落在 `artifact_not_required`，随 wire 暴露）；两者都没有 → 409 `artifact_required`。`hollow_completion` 保留原义（done 且无 artifact），但父 Issue 结算时若仍有「done + 无 artifact + 无声明」的子任务 → 409 并点名子任务。例外只有一条：本次 complete/approve 上**声明过的机检（command/artifact/url/wiki_artifact）通过**，本身就是交付证据，不必再声明；`skill_run` 这类内部记账任务在创建时自带 `not_required: skill run — the invocation transcript is the record`。决策卡 approve（`RecordTaskDecision`，绕过 MutateTask 的第二个入 done 入口）绑同一规则。MCP `team_task` 新增 `verification` / `verification_reason` 入参并透传。测试：`TestCompleteWithoutArtifactIsRefused`、`TestArtifactGate_BindsUndefinedTask`、`TestArtifactGate_RejectsPhantomArtifactWithoutDefinition`、`TestParentSettlementBlockedByHollowChild`、`TestDecisionApproveRefusesDoneWithoutArtifact`。
- **J2-3 已修（同 `e0da217`）**：触发链指认——scheduler 取到到期 follow-up job 后在锁外派回合，同一刻 cancel 落锁关 job；已派出的回合仍按旧快照跑，调 `claim` 把任务写回 `in_progress`，`scheduleTaskLifecycleLocked` 又装了一条新的 `task_follow_up`（`scheduler.json` 里那唯一一条正是重装的）。两个洞都封：终态收尾改为**不分频道**关掉所有指向该任务的 job（`completeSchedulerJobsLocked("task", id, "")`）；`MutateTask` 加终态回退守卫——`canceled/cancelled/archived` 只允许 `comment`/`cancel`/`reopen`，`done/completed` 拒绝 `claim/assign/reassign/resume/block/review/submit_for_review`，`reopen` 是唯一复活路径；计划刷新（`refreshPlannedTaskBlockStateLocked`）遇终态直接不动。测试：`TestCancelClosesJobsAcrossChannels`（去掉频道修复即失败）、`TestCanceledTaskRefusesRevivalActions`。
- **J2-5 已修（J2-1 的尾巴，2026-11-08）**：J2-1 只接了 `POST /tasks` 与 MCP，人点的「Approve」走 `POST /tasks/{id}/decision`，那个请求体没有 `verification`/`artifact_path`，服务端内核也没有这两个参数——人批准一件真的没有文件的工作会拿到 409，而错误里写的两条出路在这个端点上无处可放（`web/src` 里 `artifact_path` 出现 0 次）。现在决策端点接受 `artifact_path`/`verification`/`verification_reason`，前端加 `DeliveryGateDialog`（409 后两选一：贴 wiki 路径 / 声明「这次没有文件」+ 必填理由），接在 `TaskActionToolbar` 与 `DecisionPacketRoute` 上。测试：`TestDecisionApproveCarriesDeliveryFromHTTP`、`TestDecisionApproveRefusesPhantomSuppliedArtifact`、`TaskActionToolbar.test.tsx`、`lifecycle.test.ts` 的 `postDecision delivery`。
- **J2-6 已修（补充事实那条：重做已交付的活，2026-11-08）**：`NEX-100/102` done 之后，`11:21:58Z` 又在同一父 Issue 下建了 `NEX-148/150` 重做一遍。根因是三处「终态即忽略」：`findReusableTaskLocked` 跳过终态、子任务兄弟去重跳过终态兄弟、父 Issue 没有「已交付面」概念。不能直接放宽复用（合并进 done 任务等于静默吞掉新请求），所以改成**拒绝 + 点名已有的那条 + 唯一 opt-out**：顶层 create 命中「同题 + 同 owner + 同频道已 done」→ `conflict` 并给出 `reopen <id>` 或 `allow_duplicate=true`；子任务命中 done 兄弟 → 同样；父 Issue 下**每一条** lane 都已 done → 拒绝再拆一轮（canceled/archived 不算交付，取消后重建照旧合法）。真实两条重述是换词 + 换 owner 的中文标题（token 相似度 0.57 / 0.1，`titlesAreSimilar` 挡不住），所以**真正挡住 J2 形状的是父级「每条 lane 都交付完了」**这一条。代价（测试钉住）：lane 一条条建、前一条先 done 时，下一条要显式 opt-in。测试：`TestCreateRefusesRedoOfDeliveredTask`、`TestSubIssueRefusesRedoOfDeliveredSibling`、`TestSubIssueRefusesNewLaneUnderDeliveredParent`、`TestNewLaneAfterDeliveryRequiresOptIn`、`TestCreateAllowsRedoAfterCancelAndForOtherOwner`、`TestTeamTaskCreateForwardsAllowDuplicateOnlyWhenSet`。文档：`RUNTIME-API.md` §14.16。
- **全量 `go test ./internal/team/` 已超过默认 10m 超时**：2026-11-08 连跑三次都在 `602s` 处报 `FAIL`（`panic: test timed out after 10m0s`，最后一次卡在 `TestArtifactGate_DonePostAndNoInboxCard`，隔离开 8 次全过），加 `-timeout 30m` 后 **ok 774s**。后续全量必须显式给超时，否则会被误读成测试故障。
- **主办公室未隔离 opencode 数据目录**：`isolateOpencodeDataHome` 依赖 `WUPHF_RUNTIME_HOME`，主办公室取不到 → 回退共享 `~/.local/share/opencode/log/opencode.log`（120 MB，回合日志与用户自己的 CLI 混写）。journey1/2 因有独立 runtime home 而隔离。是否给主办公室也开隔离待定：单实例下无竞态风险，只影响日志可读性与 auth 轮换边界。
- **`bear/deepseek-chat` 自述“plan 阶段、需批准”**：主办公室核验回合（`ACMER-3396`）正文投递成功，但模型自称在 plan 阶段等批准，而 `GET /requests` 为空——无对应审批卡。若后续以“等批准”为由不出活，属模型行为而非闸门，需单独记录。
- **模型实测（真实办公室回合）**：`bear/deepseek-chat` 21s/220 字、48s/871 字、46s/203 字，工具调用 6.6–34.7s；`openrouter/qwen/qwen3.5-122b-a10b` 12–20s 文本；`local/qwen3.5-122b` 93s；`opencode-go/crouton-alpha-free` 42–142s。bear 的 qwen 系模型统一报 `System message must be at the beginning`（模型级，deepseek 系正常）。

## 核心循环 Journey 3 暴露的缺陷（2026-11-09）

来源：`core-loop-runs/2026-11-09-journey3/scorecard.md` §4（6/10，25/36，P1/P4/P7 部分命中、F2–F6 未触发）。**J3-1 / J3-2 已于 2026-11-09 修复**，复现与测试见下方「2026-11-09 复核」。

| # | 缺陷 | 现象（实测） | 代码位置 | 修复判据 |
|---|---|---|---|---|
| J3-1 | **opencode 没有 planning 呈现** | `JOURN-18` 01:02:49Z 进 `lifecycle_state=planning`，cos 01:05:38Z 按 `[PLAN MODE]` 贴出完整计划后无卡可点：`/requests` 全程空；子任务创建被 `parent … still in planning` 拒、`complete` 被拒；空转 40 分钟，下一条唤醒是 60 分钟后的 `task_follow_up`。人 01:42:27Z 手工 `approve` 才拆活 | `internal/team/headless_opencode.go`（0 处 `raisePlanApprovalAfterTurn` / `emitHeadlessPlan`；对照 `headless_claude.go:344`、`headless_codex_runner.go:391`） | planning 回合结束必须发出 plan 事件 + `RaisePlanApproval`（幂等、空计划不发），人类有一张可点的卡；非 planning 任务不得多发卡 |
| J3-2 | **watchdog 把「等审阅」当「owner 不动」** | 02:00:38/02:02:38/02:02:58/02:04:58 四条 `watchdog_alert`：「No turn could be dispatched to wake @ae」，而 `JOURN-81/85` 早已 `lifecycle_state=decision`（`ready_for_review`）、artifact 在盘。唤醒按 `isExecutableTeamTaskStatus` 被拒，报警描述的停滞并不存在，且会按 cadence 一直重复 | `internal/team/scheduler.go`（`processTaskJob` 只跳过 done/blocked，不看生命周期） | 等审阅类生命周期（`review`/`decision`/`changes_requested`/`rejected`）的 task job 直接收尾、不派活不报警；任务状态变更后由 `scheduleTaskLifecycleLocked` 重新装回；`planning` 与 parked/intake 仍受监视（parked 有 owner 但不该被唤醒，属真人待办，仍要可见） |
| J3-3 | ~~子任务创建时不带定义 → 只能等 creator 补~~ **已修（`artifact_path_missing` 提示，2026-11-09）** | J3/J4 稳定复现：子 Issue 无 `definition` → 两个 bot 撞 403 → 退到 notebook，人类到评审才知道降级。修法**不放开写门**：（a）拒写文案改成点名 creator + 指到 `definition.artifact_path` + 明说「重试不会成功」；（b）首次拒绝时往任务频道投一条 `kind=artifact_path_missing` 的 system 消息，@creator 与 @owner 都在场；（c）按任务去重（审计行 `artifact_path_missing` 存在即不再投），bot 每回合重试也不会刷屏。测试 `TestUndeclaredArtifactPathNudgesCreatorOnce`、`TestUndeclaredArtifactPathRefusalNamesCreator`（去掉去重/去掉点名即失败） | `internal/team/wiki_handlers.go`（`errTaskArtifactUndeclared`、`nudgeUndeclaredArtifactPath`） | 未做：建子任务时强制要求 `definition.artifact_path`（会让「无固定产物」的 lane 建不出来）；若未来要做，走 `teamTask.ArtifactNotRequired` 那条既有声明 |
| J3-4 | ~~**`actions` 不记录「计划被批准」**~~ **判错了，不是缺陷（2026-11-09 复核，两次）** | J3 观察里写的「`GET /tasks/JOURN-18` 的 `actions` 为 0 条」是我按不存在的字段推的：task wire（`snapshots/0206/task-JOURN-18.json`）只有 `ledger`（逐回合），没有 `actions`。批准事实记在**办公室全局动作账本**：`broker_tasks_mutation_service.go` 的 approve 激活分支会 `appendActionLocked("task_updated", …, "…[approved]", task.ID)` | 无代码改动。补充：**任务自己的 `ledger` 是人不能进去的 bot 回合日志**——`AppendTaskLedgerEntry` 强制要 `Bot` 字段，且 `Outcome` 参与 `StalledTurns` 连续计数（非 `noDurableTraceOutcome` 即清零），塞一条人类 approve 进去会污染停摆计数。人点卡的事实已在两处留痕：答卡时 `answerRequestFromActor` 写频道消息 + 审计行；`POST /tasks approve` 写全局 `task_updated … [approved]`。若要「任务页显示批准时间线」，正确做法是前端把全局账本按 `related_id` 过滤后展示，不是往回合日志里塞人 |

### Journey 4 复核（2026-11-09，J3-1/J3-2 端到端）

- **J3-1 通过**：隔离办公室 `journey4`（HEAD `364511c`）里提交人给的 Issue `JOURN-9`（03:21:55Z）→ 31 秒后 `request-12` 自己出现，`dedupe_key=plan-approval:JOURN-9`；全办公室仅此一张；只答这张卡（`POST /interview/answer`）同一秒 `lifecycle_state=running`，1 分钟内拆出 `JOURN-16`/`JOURN-18`。证据 `core-loop-runs/2026-11-09-journey4/`。
- **J3-2 通过**：`JOURN-16` 03:30 进 `decision`，其 `recheck` job 在 03:31:11 真跑过并收为 `done`（`next_run=null`），全轮 `watchdog_alert` 0 条、`watchdogs[]` 空。对照（planning 仍受监视）窗口内未观察到，由单测 `TestSchedulerStillWatchesPlanningTasks` / `TestSchedulerStillAlarmsParkedTask` 承担。
- **J3-5 结案（2026-11-09 晚，`61667e0` / `852dc3d` / 本次清理）**：当初的疑问是「同为 `in_progress`，为什么一条 lane 有 `recheck`、另一条没有，以及为什么关掉的 job 说不清理由」。三件事都已经有答案：
  1. **job 种类由状态决定，不是遗漏**：`scheduleTaskLifecycleLocked` 的 `in_progress` 分支只装 `task_follow_up`（小时级），非 `in_progress`（blocked/queued/drafting 等）分支装短周期 `recheck`。Journey 5 的快照直接拍到两者互相顶替（`superseded by recheck` / `superseded by task_follow_up`）。
  2. **关闭有理由了**：`schedulerJob.status_reason`（审阅门 `awaiting review (<state>)`、顶替 `superseded by <kind>`、连带关闭 `target closed`、其它 `closed`），重新排期清空。
  3. **`last_run` 不再被覆盖**：顶替关闭只写 `closed_at`；`last_run` 只表示「派发过」。
  另：`ScheduleRecheck`（ws 里那个 export）**没有任何调用点**，本轮删除——`recheck` job 只由生命周期分支产生。
- **J3-3 稳定复现**：J4 里两条子 Issue 同样创建时无 `definition`，同样撞 `wiki 直写 403`、改走 notebook + 请 @cos 补 `artifact_path`。

### Journey 5 复核（2026-11-09，J3-3 修复 + 关闭理由）

- **V1–V5 命中，F-A..F-D 未触发**（判分见 `core-loop-runs/2026-11-09-journey5/scorecard.md`）：计划卡 04:52:24 自行出现（创建后 22 秒）；只点这一张卡 → 18 秒内拆出 4 条 lane；403 → 一条 `artifact_path_missing` 提示落在任务频道、同时点名 creator/owner，两次调用只喊一次；已关 job 全带 `status_reason`。
- **V6 字面未过**：`watchdogs[]` 终态非空，但那是 `request_waiting`（人类没答「缺资料」采访卡），不是 J3-2 的「唤醒被拒」；`watchdog_alert` 消息全程 0。
- **J3-5 的疑问就此结案**：`recheck` 不是被审阅门关的，是被状态变更顶掉的——快照里能看到 `task-follow-up … superseded by recheck` 与 `recheck … superseded by task_follow_up` 交替。`JOURN-15/19` 全程无 `recheck`，因为它们没离开过 `in_progress`（该分支只装 `task_follow_up`）。
- **仍未证明**：bot 会不会把 403 文案读进去并改走 human request——本轮四条 lane 一次都没尝试写 wiki，都在等源数字（notebook 全 `[NEEDS CONFIRMATION]`），403→提示的链路只由停表后操作员探针证明。

### Journey 6 复核（2026-11-09，批准后的两个静默）

来源：`core-loop-runs/2026-11-09-journey6/`（prereg `9dfa3b8`）。判分：V1–V4 **未观察**、V5 部分未命中、V6 空真、F-A..F-D 未触发。场景给了全部数字（Acme 40 席位、$18,000/年、2026-12-31 到期、30 天付款、续约 $21,600、14 天有效期、决策人 Dana），期望逼出写 wiki；实际 `write-task` 全程 0 次，连一次 403 都没有，所以 **J3-3 那套修复的端到端效果仍然没有被验证**。

| # | 缺陷 | 现象（实测） | 代码位置 | 修复判据 |
|---|---|---|---|---|
| J6-1 | **批准后的首次分解没有短周期检查点** | `JOURN-8` 06:23:19Z 批准；激活回合 06:23:26Z 的正文全在回答另一条任务（`JOURN-3`）的管道问题；此后 `tasks` 只有父任务自己，`write-task` 0 行，06:33:35 一条 `task_stalled`，下一次唤醒 `task-follow-up:cos--human:journ-8` 是 **07:23:19Z**。45 分钟上限内零进展 | `internal/team/broker_plan_approval.go`（`startApprovedPlanTaskLocked` 只 `scheduleTaskLifecycleLocked`，即小时级 follow-up）、`internal/team/scheduler.go` | 批准后 5 分钟内检查一次：有子 Issue → 静默收尾并写明条数；仍 Running 且无子任务 → 写 `task_not_decomposed` 告警 + 唤醒 owner 一次；唤醒被拒走 `escalateRefusedWake` |
| J6-2 | **被取消任务的采访卡留着，继续压住 bot 的唤醒** | `request-16`（`issue_id=JOURN-3`、`blocking` 为空）在 06:27:32Z 任务取消后仍 `pending`；它的 `request_follow_up` 06:52:15Z 触发，只写 `watchdogs`/`actions`，`messages` 18 条不变——聊天里 0 条。而 `Launcher.turnParkedOnInterview` 会丢掉该 bot 的所有唤醒，所以这张没人知道要答的卡能一直压住 cos | `internal/team/broker_lifecycle_transition.go`（终态转换不碰 requests）、`internal/team/scheduler.go`（`processRequestJob` 的 `req.Blocking` 门） | 任务进终态时取消它名下所有 active request（理由 `task closed (<state>)`）；`processRequestJob` 对任何 pending request 都投一条可见提醒（非阻塞用「Waiting on your answer」） |

**已修（2026-11-09）**：
- J6-1：`broker_scheduler.go` 新增 `planDecompositionCheckKind` / `planDecompositionCheckDelay`(5m) / `schedulePlanDecompositionCheckLocked` / `TaskSubIssueCount`；`startApprovedPlanTaskLocked` 末尾装 check；`scheduler.go` 的 `processPlanDecompositionCheck` 处理收尾/唤醒/升级。测试 `TestPlanDecompositionCheckWakesOwnerWhenNoLanes`、`TestPlanDecompositionCheckClosesWhenLanesExist`、`TestApprovedPlanArmsDecompositionCheck`（去掉装 job 即失败：`approval did not arm the decomposition check`）。
- J6-2：`broker_requests_interviews.go` 新增 `cancelRequestsForClosedTaskLocked`，由 `transitionLifecycleLocked` 在目标状态为终态时调用；`processRequestJob` 去掉 `req.Blocking` 门。测试 `TestCanceledTaskClosesItsPendingInterview`（去掉钩子即失败：`interview still active after its task was canceled`）、`TestDoneTaskClosesItsPendingInterview`、`TestRequestReminderPostsForNonBlockingInterview`（去掉门即失败：`posted 0 messages, want 1`）。
- **仍欠**：J3-3 链路的端到端证据（403 → 三要素文案 → 去重 → 改做法）。Journey 6 因环境遗留任务抢走注意力，没走到写路径；需要独占频道的一轮重新取证。

### Journey 7 复核（2026-11-09，干净环境下的同一个问题）

prereg `f3b62d4`（其中 §2.1 更正：专用频道不可行——`namedChannelsEnabled()` 硬编码 false，`POST /channels` 409，只能靠清理隔离）。判分见 `core-loop-runs/2026-11-09-journey7/scorecard.md`。

- **环境这次是干净的**：onboarding 后 `ai-revops-1..4` 立刻取消，提交场景前 `tasks` 只剩 4 条 canceled、`requests` 为空；Journey 6 的污染源（`[@analyst] Repeated errors blocked` 落进 `cos__human`）没有出现。
- **分解正常**：07:44:30 批准，30 秒内拆出 6 条 lane（`JOURN-13/16/18/21/23/25`，全部无 `definition`）。`F-E` 未触发，`task_not_decomposed` 0 条。
- **仍然没有一次 `write-task`**：`write-task` / `403` / `refused` / `artifact_path_missing` 全部 0；bot 确实写了 wiki 仓库，但走的是 `visual artifact`（`ra_…` 三条）与 `fact`/`entity` 通道，`team/**` markdown 交付 0 份；三条 lane 把稿子写在 `agents/*/notebook/`。
- **为什么 403 路径打不通（代码证据）**：`team_wiki_write` 不带 `task_id` 时返回的是「需要人类直接授权……bot 侧知识请用 notebook_write → notebook_promote」（`internal/teammcp/server_wiki_tools.go:44-52,100-110`）。`/wiki/write-task` 的 `artifact path not declared` 只有在 bot **主动带 `task_id`** 时才会命中。J3/J4 撞上 403 是那几轮 lane 自己选了带任务 ID 写入；J6/J7 的 lane 选了 notebook + visual artifact——两条路都合法，模型默认选后者。
- **结论**：**J3-3 的端到端（403 → 三要素文案 → 改做法）连续两轮没有被锻炼**，而且现在知道原因不是环境，而是投递路径的分叉。要么让规划者在拆 lane 时就钉 `definition.artifact_scope`（这样 lane 才有资格走 write-task），要么承认 notebook/visual-artifact 就是默认交付、把 J3-3 的修复当防御性兜底。**这是产品决策，不是缺陷**，列在这里等人拍板。
  - **2026-11-09 拍板：选 A**（钉写范围），理由与实现见 `DESIGN-DECISIONS.md` §9.15。改动：`create` 接受 `definition`（lead/human 才能设）、lead 建 lane 无 `artifact_scope` 且无 `artifact_not_required_reason` 即拒绝、专家豁免（否则成了点不了名的死路）、403 文案保留作「路径不匹配」兜底。测试 `broker_subissue_write_scope_test.go`（6 条；去掉两处门槛即失败）。
  - **下一步**：Journey 8 取证（prereg 先 commit）。判据 V1 lane 都带写范围 / V2 `team/**` ≥1 份 markdown 且 `write-task` 的 commit 落在 cos/ae 名下 / V3 规划者撞守卫 ≤ 每 lane 1 次且撞后改做法；反向臂：操作员用错路径，403 仍点名 creator 且只喊一次。**回退**：若规划者反复撞守卫或路径全是敷衍值（`team/tasks/x.md`），切 B，本修复降级为「防路径不匹配」。

### Journey 8 复核（2026-11-09，写范围守卫的第一次真机检验）

prereg `a4ff236`，被测 `5a026e7`，判分 `core-loop-runs/2026-11-09-journey8/scorecard.md`（独立 reviewer）。

- **机制成立**：批准后 30 秒内拆出 6 条 lane，**全部一次带 `definition.artifact_path`**；`write-task` 成功 9 次，`team/**` 新增 `team/emails/` 2 份、`team/reviews/` 5 份，作者就是 lane owner。守卫拒绝 **0** 次——规划者读了系统提示就照做了。
- **J3-3 的 403 链终于被锻炼**：4 次拒绝全是「声明了但路径不匹配」，每次都在 2–5 秒内改成声明路径并成功（`JOURN-15/19/21/23`）。反向臂（`JOURN-127`，human 建的未声明任务）3 次 403、只投 1 条 `artifact_path_missing`，文案点名 `@human` + `definition.artifact_path` + 重试无用。V1/V2/V4/V5/V6 通过或带保留，V3 未观察（没人撞守卫）。
- **不触发回退**：F-A/B/C/D/E 全未触发，§9.15 的 A 方案保持。
- **新缺陷（评分卡 §4）→ 2026-11-09 已修（`DESIGN-DECISIONS.md` §9.16）**：
  - **D1 promotion 卡 park 住回合：证伪。** 推广卡 `Blocking=false`、`kind=approval`，既不过 `requestBlocksMessages`（要求 Blocking）也不过 `BotAwaitingInterviewAnswer`（只认 `kind=interview`）。停表的真因是下面的 D4。测试 `TestPromotionApprovalCardDoesNotParkOrGate` 钉住这条，别再拿兜底卡当嫌疑人，也别给它加自动放行。
  - **D2 同主题 lane 重复：已修。** CJK 没有空格，整句是一个 token，两对重复的相似度只有 0.20 / 0.43。兄弟去重加两条：CJK shingle 包含度 ≥ 0.9 且 ≥ 6 个共享 shingle；同一父任务下两个 open lane 不得声明同一 `artifact_path`。`allow_duplicate=true` 可越过这两条。
  - **D3 声明了路径却用 `ra_` 收 done：已修。** 完成门对「声明了写范围」的 lane 拒绝视觉 artifact；`verification=not_required` + 理由仍是出口；未声明范围的老任务行为不变。
  - **D4/D1 落地没人叫醒：已修。** 一条 lane 落地会把父 Issue 和仍 open 的兄弟的既有 follow-up 拉到 ~2 分钟后（只提前，不新建）。
  - **D5 分解计数：已修。** `TaskSubIssueTotals` 报「lane 总数 + 仍 open 数」，判断仍用总数（全落地时不能当空壳唤醒）。
  - **D6 路径不匹配的 403 不投提示：结案，设计如此（§9.17）。** 403 body 已并排给出声明路径与尝试路径，随工具响应回到发起调用的 bot；J8 四次不匹配全部在 2–5 秒内用正确路径重试成功。频道提示只会把同样两个路径再说给不能改的人。只有「未声明」（需要创建者动手）才 nudge。测试 `TestWriteTaskMismatchStaysOnTheCall` 钉住：body 含两条路径、频道消息/action 数不变、按声明路径重试落盘。

- **`336bbbc` 在真机上的表现**：批准时装了 `plan-decomposition-check:cos--human:task:journ-6`；20 秒后父任务因拆出子 Issue 触发生命周期重排，把它以 `superseded by task_follow_up` 关掉（本轮无害，因为确实分解了）。但这个机制意味着**任何后续 mutation 都会取消这道保险**，而它要盯的恰恰是「任务一直 running、频道里却像在动」的状态。已修：`cancelSupersededTaskJobsLocked` 对 `plan_decomposition_check` 免于顶替（仍会在自身触发、终态 bulk close、取消时关闭）。测试 `TestDecompositionCheckSurvivesLifecycleRearm`（去掉豁免即失败：`status = "done" (reason "superseded by task_follow_up"), want scheduled`）。

### Journey 9 复核（2026-11-09，写范围守卫的第二次真机检验）

prereg `4863c50`，被测 `690b601`（运行时等同 `000c4b7`），判分 `core-loop-runs/2026-11-09-journey9/scorecard.md`（独立 reviewer）→ **核心循环 5/10**。

- **站住的**：D5 计数 `decomposed into 4 sub-issue(s); 1 still open` 与实际完全一致；D4 提前唤醒精确命中（父 follow-up `14:45:26` → `13:53:39`，13 秒后父 Issue 在频道推进）；D3 的 `ra_` 分支探针返回 409；D6 的路径不匹配仍是 7 秒自愈、不刷频道。
- **没站住的**：
  - **D1（重）声明路径被别人的文件顶替 → 已修。** `JOURN-18` 声明 `team/reviews/acme-renewal-check.md`（从未写），用 `JOURN-32` 的 `acme-renewal-final-check.md` 收 done，门限放行（探针复现 200）。现在 `sameWikiPath(declared, recorded)` 不等即 409。
  - **D1b（重）落地后还能改 artifact → 已修。** 对已 done 的探针 lane 再发 `complete`（`ra_…`）返回 200 并改写 artifact，因为门挂在 `reachedDone` 上。现在「记录 artifact」即受检（`reachedDone || body.ArtifactPath != ""`）。测试 `TestForeignArtifactCannotCloseADeclaredScope`、`TestLandingLaneCannotSwapArtifactForVisual`（去掉修复各自红）。
  - **D4（中）CJK 去重差 0.025 → 已修。** `Acme 续约数字与事实终审` vs `子任务 C：数字与事实终审（Acme 续约）` = 0.875，阈值从 0.9 降到 0.85；测试 `TestSiblingLaneRestatedWithoutLaneLetterIsRefused`。
  - **D2（重）decision 态没有人类能点的卡 → 已修（§9.19）。** `raiseDecisionCardLocked` 在三个收敛出口各发一张非阻塞卡（`dedupe_key=decision:<taskID>`，随 lane 关闭自动撤），应答走 `RecordTaskDecisionWithComment`：approve → `approved`，send back → `changes_requested` + 原话落到 owner 的返工说明；线程回复按 send back 处理。测试 `broker_decision_card_test.go` 七条（缺卡 / 无交付不发卡 / approve / send back / 关单撤卡 / 重复应用 / 过期卡报错），关掉收敛钩子或应答落地各自会红。原记录： `JOURN-16` 停在 `ready_for_review`／`decision` 13 分钟，`pending_interview=None`，全库无请求指向它；父 Issue 下一次唤醒 `14:23:42` 晚于 15 分钟停表点 `14:20:02`，F-H 由此触发（26.1 分钟静默）。**这是产品决策**：要么在 lane 进入 `ready_for_review` 时物化一张指向自身、非阻塞的确认卡；要么承认「等人不算静默」，把停表判据改成「无未决请求」。未定之前不动调度器。
  - **D6（重，冒烟新发现）`/wiki/write-task` 200 但文件落在相似名的邻居上 → 已修（§9.20）。** `DECSM-36/49` 声明 `team/reviews/dec-smoke-answer.md`，写返回 `200 {bytes_written: 15}`、broker.log 也有该路径，但 commit `1cc1786` 只含 `team/reviews/dec-smoke.md`，内容被追加进邻居，声明路径全盘不存在；完成门随即拒绝 lane 自己的 approve（`does not exist in the wiki`）。根因是 update-first（`routeBotCreateToSimilarSlug`，相似度 ≥ 0.85 的 bot `create` 改道 `append_section`）也作用在任务声明路径上。现在 `EnqueueTaskArtifact` + `PinnedPath` 让任务写逐字落盘，自由写仍改道；测试 `TestTaskArtifactWriteKeepsItsDeclaredPath` 去掉 pin 即红（现象与真机逐字相同）。
  - **D3（中）`watchdog_alert` 计数源两份。** `actions.kind` 计 2 条（`task_stalled`），`messages` 计 0 条，导致 V4 在 J8 过、J9 不过。下一轮 prereg 必须写死计数源。
  - **D5（中，判据问题非代码）** V5 照抄 J8 的「403 点名 creator」，但 J9 探针命中的是路径不匹配分支，那一支本来就只点名两条路径（见 §9.17）。判据应按探针实际命中分支写。
- **本轮证不了**：CJK 去重的真阳性（场景里只有 0.875 的近 miss 与精确标题拦截）；规划者在场景内被 `ra_` 拒绝后自修正；人类确认链闭环（父 Issue 停表时仍 `in_progress`）。

### Journey 11 复核（2026-11-10，父 Issue 的卡出现了，但只点名一个文件）

prereg `0c2d1ac`，被测 `54ad984`，判分 `core-loop-runs/2026-11-10-journey11/scorecard.md`（独立子代理）。场景同 J10，模型 `opencode-go/longcat-2.5-free`。

- **站住的**：
  - **父 Issue 的卡真的会发、真的能点**：`JOURN-2` 04:24:27Z 进 `decision`，`request-83` 04:24:51Z 出现（24 秒），04:25:00 点 `approve_decision` → 200，04:25:06 `done/approved`，wiki 落 `team/decisions/JOURN-2.md`（`002deee`）。J10 的 F-I 没有复现。
  - **lane 也会进 decision 并停人**：`JOURN-22`/`JOURN-24` 各自出现 `decision:` 卡并被点击批准；J10 一条都没有。
  - **回归**：计划卡唯一、`task_not_decomposed` 0 条、声明路径与落盘逐字一致。
- **没站住的**：
  - **D2 的正文口径（中）→ 已修。** 卡只写了 `The deliverable is team/accounts/acme-renewal-quote.md.`，因为 cos 在 `submit_for_review` 时把一条 lane 的路径盖成了 Issue 的 `artifact`，而卡片先读这个字段。现在「容器」优先于「自己的 artifact」：有 lane 交付就逐条列出，自己的路径只在没有 lane 认领时追加一句。回归 `TestDecisionCardNamesEveryLaneEvenWhenIssueCarriesOneArtifact`（反向改回分支顺序即失败）。**注意：§9.22 的聚合谓词这一轮没被真机锻炼**（父 Issue 进 decision 时已有 artifact），只有进程内测试证明。
  - **`submit_for_review(artifact_path)` 给容器盖单文件戳 → 已修**：两条写路径都查 `childOwningArtifactLocked`，命中已交付 lane 的路径就 409 并点名那个 lane；容器自己写的文件仍可记录。测试 `TestContainerRefusesALanePathAsItsOwnArtifact`、`TestDecisionApproveRefusesALanePathAsTheIssueArtifact`、`TestLaneStillRecordsItsOwnArtifact`。**下一轮真机应能看到 409 与随后的无 artifact 重试**，父 Issue 的 `artifact` 保持为空，聚合谓词才会第一次被真机锻炼。
  - **`/tasks/{id}/comment` 对 bot 私有频道 500 → 已修**：守卫本来就该拒（人类在 bot↔bot DM 只读），但响应既不说原因也不说频道。现在 403 + `{"channel":…,"detail":…}`，底层原因一律带在 `detail` 里；测试 `TestTaskCommentOnBotDMReturnsTheReason`（含「被拒的评论不得入库」）。
  - **`team_wiki_search` 的契约陷阱 → 已修**：接受 `query` 作 `pattern` 的别名（两者都传时 `pattern` 优先），错误文案点名两个字段，零命中回「No hits for the literal substring "…" in <reader>；这是子串匹配，不是存在性清单」而不是 `[]`；测试 `internal/teammcp/wiki_search_alias_test.go` 四条。**留下的仍是模型行为**：`JOURN-26` 全程没打开声明路径。
  - **干预 7 次 vs 预算 2 次**：多出来的是两张 lane 卡、更正 QA 前提、取消 QA、催交终审。下一轮预算按「每 lane 一张卡」重算。
- **顺手修的两处**：`team_wiki_search` 接受 `query` 别名 + 零命中不再回 `[]`；`/tasks/{id}/comment` 对 bot↔bot DM 改回 403 并带 `channel`/`detail`。见 §9.23。
- **测试基建的已知 flake（未修，与业务无关）**：`TestObsidianWatcher_*` 在整包并行跑时偶发 `TempDir RemoveAll cleanup: … .wuphf\agent-scratch\cos: The process cannot access the file`（2026-11-10 连续两次，分别是 `DebounceCoalescesRapidWrites` 与 `StopDuringPendingDebounce`）。单独跑 `-run TestObsidianWatcher` 14.6s 全绿，整包 `-skip TestObsidianWatcher` 全绿。假设：包内 256 处 `t.Parallel()` 共享进程环境，`WUPHF_RUNTIME_HOME` 指向另一个（已结束的）测试的 `t.TempDir()`，并行测试按它写 `agent-scratch/<slug>`，句柄没关就在别人的 RemoveAll 里撞上。要修得先审计哪些并行测试写 scratch 目录，不是本轮范围。

### Journey 12 复核（2026-11-10，容器守卫生效了，模型转去用 not_required 自己关父 Issue）

prereg `core-loop-runs/2026-11-10-journey12/prereg.md`，被测 `4c53083`，判分 `core-loop-runs/2026-11-10-journey12/scorecard.md`（独立子代理）。场景同 J10/J11，模型 `opencode-go/longcat-2.5-free`，无 QA lane。

- **站住的**：
  - **容器守卫真的拦住了**：cos 自述「父任务是容器，不能引用子任务的产物」（msg-42），父 Issue 的 `artifact` 最终为空、`artifact_not_required` 里点名的两条路径与 lane 的 `artifact` 逐字一致（`broker-state.json`）。J11 的 `artifact_path` 盖戳没有复现。
  - **lane 交付闭合**：`JOURN-13`/`JOURN-15` 各自 `artifact` 与 wiki 提交（`d9a7c79`/`f044319`）逐字一致；计划卡唯一、`task_not_decomposed` 0 条。
- **没站住的**：
  - **F-a 的变体：人类主权的旁路（重）→ 已修。** 守卫拒了 lane 路径，但 `MutateTask` 的 `complete` + `verification=not_required` 仍然允许 **bot** 把有已交付 lane 的容器关掉：不产生决策卡、人类没被问、`hollow_completion=true`（`JOURN-1`，08:00:39Z）。这正是 J10 记下的「故意的不对称」，J12 证明它不是对称问题而是旁路。现在 bot 关这类容器一律 409 并指回 `submit_for_review`（无 artifact）→ 决策卡 → 人工批准；人类自己仍可从任务端点关，空容器照常关。`HollowCompletion` 在该路径也按聚合交付算。测试 `TestContainerWithDeliveredLanesCannotCloseFromTheTaskEndpoint`、`TestEmptyContainerStillClosesFromTheTaskEndpoint`。**§9.22 的聚合谓词仍然没被真机锻炼**——父 Issue 从未进入 `decision`，卡一次没发（V3/V4/V5 判失败/未观察）。
  - **lane 自己也没有卡**：两条 lane 07:47 进 `decision`，`requests` 里没有任何 `decision:JOURN-13/15`；当时 `artifact` 仍空，谓词判「没什么可批」。真正让文件上线的是两张 `knowledge-promote` 卡（干预 3、4）之后 cos 自己把 lane 标通过——**lane 的交付未经人类**，这与 §9.22 只保证「有交付物的容器有卡」一致，但意味着「lane 层的人工确认」在 office 模式仍是空的。
  - **lane 无 reviewer**：日志 `task review: no reviewer available for JOURN-13/JOURN-15 (owner "cos")`（J11 有 QA lane）。
- **干预 4 次 vs 预算 3 次**：request-4 事实、request-8 计划卡、request-21/23 知识提升。后两张是产品流程自己逼出来的——没有父 Issue 卡，lane 的文件只能走知识提升上线；不是操作员急躁。下一轮预算按「事实 + 计划 + 可能的知识提升」重算。
- **顺手修的一处**：`HollowCompletion` 从「只看 artifact」改成「artifact 空 **且** 无已交付子任务」；两条关闭路径（决策端点、任务端点）现在口径一致。
- **这一轮证不了**：父 Issue 决策卡正文（V4）、人工点击闭环（V5）、`decision:` 卡的去重、容器 409 之后模型是否转走 `submit_for_review`（本轮它走了 not_required，代码允许；修复后才有下一轮）。**J13 才可能第一次锻炼 §9.22 的聚合谓词**：要断言父 Issue 进 `decision` 且 `artifact=null`，并加一条「bot 对这类容器的 `complete` 必须 409」。

### Journey 14 复核（2026-11-10，lane 卡出现了，但 DoD 的 artifact kind 让 lane 永久关不掉）

prereg `core-loop-runs/2026-11-10-journey14/prereg.md`，被测 `6caf6cf`（+自愈修复），判分 `…/journey14/scorecard.md`（独立子代理）。场景同 J13，但 Issue 正文钉死「两条子任务、两份独立文件」。

- **站住的**：
  - **钉死形状有效（F-i 未发生）**：cos 拆出 `JOURN-12`/`JOURN-14` 两条 lane（msg-13/15），各自写自己的文件（wiki `8ab0cc2` 报价 36 行、`135f9dc` 邮件 22 行），父 `JOURN-1` 的 `artifact` 保持为空。
  - **lane 带 artifact 时会出自己的 decision 卡**：`request-24`/`request-27`（`decision:JOURN-12/14`）——J12 的 lane 因 artifact 为空而无卡，这次证明了 §9.22 的谓词对 lane 也成立。
- **没站住的**：
  - **lane 卡点不动（重，产品缺陷）→ 已修**：人工点 `approve_decision` 两张都 409 `definition-of-done check failed (artifact): no non-empty artifact matches "team/accounts/acme-renewal-quote.md"`。lane 的 `verification.kind=artifact`、spec 是 wiki 相对路径，而 `artifact` 只 glob 工作区/bot scratch；schema 从没列出 `wiki_artifact`（`server_types.go` 的 `VerificationKind`），模型选不到对的 kind。现在 schema 点名 `wiki_artifact`。
  - **关不掉的 lane 无法补救（重，产品缺陷）→ 已修**：cos 自己诊断出「校验卡在 artifact（工作区文件）上，而交付物是 wiki 文章。改用 define 把校验方式修正为 none」，但 `define` 的 `task.Verification != nil` 分支静默拒绝替换，lane 的 `verification` 始终是 `artifact`，**没有任何人能关掉它**。现在**已失败过的 gate 允许被 re-define 替换/清除**（action log 记 `verification_replaced`），通过的 gate 仍不可削弱。测试 `TestDefineReplacesAFailedVerificationGate`、`TestDefineDoesNotReplaceAPassingVerificationGate`。
  - **`definition-gap` 卡又重复追问（模型行为）**：`request-48` 问 `access needed: wiki write access`——wiki 写入本来就是 lane 的默认能力，且文件已经写进去了。
- **干预 3 次 vs 预算 3 次**：request-3 事实、request-8 计划、两张 lane 卡的 approve 尝试（同一类动作，算 1 次干预的两次点击）。
- **这一轮证不了**：父 Issue 的容器 409（V1）、父决策卡（V3/V4）、人工批准闭环（V5）、bot 关闭门（V7）——**都被 lane 卡死拖住了**。修复后才有 J15。
- **处置**：lane 卡死属于产品缺陷而非场景问题，J14 停机归档；修复（schema + 失败 gate 可替换）经全量测试后部署，**J15 用修好的二进制重跑同一场景**。

- **第三处修补(同轮真机)**:重启 broker + `define` 换 `wiki_artifact` 之后,人工点 lane 卡仍 409 `no artifact_path on the completion` —— 卡批准不带 `artifact_path`,校验只读请求体。已改为回落到任务已记录的 `Artifact`(显式值仍优先);什么都没记录仍 fail closed。**J14 的 lane 卡在部署这一版后才可能真正通过。**
- **结果(同轮)**:部署第三处修复后重启,两 lane 自动闭环(`done/approved`、`wiki_artifact` 校验 `pass=true`);父 `JOURN-1` 进 `in_progress/decision`、`artifact=null`,卡 `request-92` 正文列出两条 lane 路径并写明「The Issue ships no single file of its own」;人工 `approve_decision` -> 200,父 `done/approved`、`artifact=null`、`hollow_completion` 不出现(=false)。**聚合谓词第一次在真机上被触发并通过。**
- **这一轮证不了**:V1/V7(模型这轮没有试图把 lane 路径塞给父 Issue,也没有用任务端点关容器——它是被 DoD 卡住的,不是被容器守卫拦住的);所以「容器只能经决策卡关闭」的新门在 J14 里仍是**未被锻炼**的代码,等 J15+ 的措辞把模型推上去才知道。
- **新发现(判据外)**:lane 的 `definition.artifact_path` 没被设置时,第一条 wiki 写入被拒并提示「Pin the path on the task」;模型绕道「请人类提升知识到 wiki」,交付物先以 knowledge draft 落库、再由人工 promotion 落到正式路径。这条写入范围提示本身可用,但它把交付路径的分叉成本推给了人。

- **第五处修补(评分卡 §3 判据外)**:容器守卫漏了 `definition.artifact_path` 这条写范围(父 Issue 被 define 钉上 lane 路径=授权覆写);`define` 省略 `artifact_path` 会清空已有范围(操作员的 define 把两条 lane 清成 null)。都已修并有双向测试。
- **评分卡结论**(独立子代理,`.../journey14/scorecard.md`):V2–V6 成立、**V1/V7 未观察**(本轮模型没去碰容器守卫)、F-a…F-i 全不成立。即:**聚合谓词第一次真机成立**;容器写入/关闭门仍是未被锻炼的代码。
### Journey 15 复核（2026-11-10，模型自己撞上两道容器守卫，两条都拦住了）

prereg `6c91bfd`（开跑前单独提交），被测 `8f0cee0`，判分 `core-loop-runs/2026-11-10-journey15/scorecard.md`（独立子代理）。场景同 J14（两条 lane、两份独立文件），Issue 正文写明「这个 Issue 自己不写文件，我要看到它进 done」——J12 里 cos 用 `not_required` 关容器的自我说服原话。模型 `opencode-go/longcat-2.5-free`，工作区 `journey15`（broker :7910）。

- **站住的**：
  - **模型自发撞写入门（V1 的对象第一次出现）**：`14:56:26` `guard_refused` / `related_id=JOURN-1` / `container artifact: team/accounts/acme-renewal-quote.md is JOURN-15's deliverable`。把 lane 的交付物当父 Issue 自己的 artifact，被拒。
  - **模型转头去任务端点关容器（V7 的对象第一次出现）**：`14:57:11` `container close: cos tried to close it with 2 delivered lane(s)`，第二次拒绝。**J14 修的那条人类主权旁路被模型主动踩到**。
  - **拒绝之后模型自己改走正路**：`14:57:22` `JOURN-1 → review`、`14:57:45` 父卡 `request-46` 发出、人批 → `done/approved`、`artifact=null`、无 `hollow_completion`（F-a/F-c/F-h 都没发生）。
  - **判据 V2–V6 全部成立**：父 `artifact` 全程 `null`；`decision` 与卡同一分钟内（≤3 分钟）；卡列出**两条** lane 路径且无 `The deliverable is <单一路径>` 口径；计划卡唯一、`task_not_decomposed` 0 条、lane 声明路径与落盘逐字一致；wiki 有 `team/decisions/JOURN-1.md`。
- **压力来源不是操作员**：`14:56:40` 产品自己发 `watchdog_alert`「@cos still needs to move Journey 15 …」——催「推进」的拉力来自场景与产品，模型是自己走上去的。
- **这一轮证不了**：**V9**（容器 `define` 的写范围）——模型从未对 `JOURN-1` 调 `define`，`actions[]` 里父 Issue 只有 `task_created`/`task_updated [review]`，写范围门这次仍无对象。V9 两次真机对象仍只有 guard exercise 的操作员剧本。
- **干预 4 次 vs 预算 3 次**：事实采访 → 计划卡 → 两张 lane 卡 → 父卡。prereg 没给 lane 卡留位，而它是产品固定流程（且两张都批在 lane 自己关闭之后，等于空操作）。下一轮预算按「事实 + 计划 + lane 卡 + 父卡」写。
- **判据外发现**：
  - **裸 `GET /tasks` 落到 `general`（已修 `a7ec9f8`）**：`ListTasks` 把空 channel 洗成 `general`，裸列表永远答 `{"channel":"general","tasks":[]}`；Journey 15 的观察脚本因此连续读到「0 个任务」而 `JOURN-1` 就在 `cos__human`。现在裸列表 400 并点名三种读法，`?parent_issue_id=` 不再被频道过滤。与 `GET /requests` 同一类：**没有默认房间就必须报错**。
  - **lane 决策卡追不上 lane 自己的关闭（已修 `01bfcb8` 之后）**：两条 lane `14:53:50` 进 `decision`、卡已发，`14:54:40`（人批之前）已是 `done/approved`（`task review: no reviewer available (owner "cos")`），人的批准落在已关闭的 lane 上。根因不是「无 reviewer 自动通过」——`TransitionLifecycle` 自 Journey 6 起就在终态时撤卡，而 lane 走的是 `MutateTask` 的 `complete`/`approve`/`reject`/`cancel`，那条路径从没调用过 `cancelRequestsForClosedTaskLocked`。现在终态分支补上，`TestCompletingALaneClosesItsOrphanedDecisionCard`（去掉调用即红：`the lane's own card survived its close`）并断言兄弟 lane 的卡仍可答。**lane 层的人工确认**仍取决于「无 reviewer 自动通过」这条规则，未动。
  - **`definition-gap:` 把已授予的 scope 当缺口再问一遍（未修，第四次复现）**：`request-19`/`request-22` 问「wiki 写入权限」，同一时刻 `artifact_scope` 已授予、文件已写完；两卡至今 `pending`，无人需要。

### Guard exercise（2026-11-10，操作员剧本：容器守卫 V1/V1b/V7 第一次被真正踩到）

prereg `core-loop-runs/2026-11-10-guards/prereg.md`，判据 G1–G7、失败模式 H-a…H-e；证据 `…/2026-11-10-guards/`（`observations.md` + `artifacts/`，含驱动脚本、broker 状态、两份 lane 文件、wiki `git log`）。被测 `0464f91`，工作区 `guards15`（broker :7910）。**不用模型**：操作员直接打 broker HTTP API 扮演抄近路的 agent，因为 J14 的评分卡把 V1/V7 判成「未观察」，等模型自发撞门的概率不可控。

- **站住的（7/7，H-a…H-e 全未命中）**：
  - **G1** 容器 `complete` + lane 路径 → 409 `task GUARD-9 is a container: team/accounts/guard-quote.md is GUARD-12's deliverable, not this Issue's own. …`
  - **G2** 容器 `define` 的 `definition.artifact_path` = lane 路径 → 409 `… is not a file this Issue writes itself. …`
  - **G3** bot `complete` + `verification=not_required` → 409 `… is a container with 2 delivered lane(s): a bot may not close it from here — that would skip the human. Call submit_for_review …`
  - **G4** 三次撞门后容器仍 `in_progress/running`、`artifact=null`、`artifact_not_required=null`（没有被任何一次写入污染）。
  - **G5** 正路 `submit_for_review`（无 `artifact_path`）→ 200，进 `review`。
  - **G6** `request-21`（`dedupe_key=decision:GUARD-9`）正文逐字列出两条 lane 路径，并写明 `The Issue ships no single file of its own`。
  - **G7** 人工 `approve_decision` → 200 `{"ok":true}`；容器 `done/approved`、`artifact=null`、`hollow_completion` 不出现(=false)。
- **新发现（判据外，已修）**：`childOwningArtifactLocked` 只匹配子任务**已记录**的 `Artifact`，不看子任务**声明**的 `Definition.ArtifactPath`。第一次试跑里容器 `define` 指向一条尚未交付的 lane 的声明路径 → **200 接受**：lane 创建到交付之间有一段窗口，容器可以在窗口里把自己授权去写那条 lane 将来要写的文件。现在匹配也看声明写范围，并跳过 `canceled/cancelled/archived/rejected`（退场的 lane 不再拥有旧路径）。测试 `TestContainerCannotClaimAPendingLanesWriteScope`、`TestCanceledLaneWriteScopeDoesNotBlockTheContainer`（前者去掉修复即红）。
- **边界被实测确认（非缺陷）**：空容器（无已交付 lane）从任务端点 `complete` + `not_required` 仍是**合法**的，`hollow_completion=true` 是诚实的——它确实什么都没交付。门只在「有已交付 lane」时存在。
- **未深挖的观察**：① 一个任务可以写另一个任务**已交付**的 `artifact`（`GUARD-4` 覆盖了 `GUARD-12` 的 `team/accounts/guard-quote.md`）；文件是否该独占未定，`mode=append_section` 的共享写法是有意的。② `GET /requests` 在 `0464f91` 上返回 `requests: []`，而同一刻状态文件里有 pending 卡；脚本改用状态文件取卡 id。
- **判分**：独立子代理 `GuardScore`（`…/2026-11-10-guards/scorecard.md`）：**G1–G7 全部成立、H-a…H-e 均未命中、硬失败无**；并指出三处 409 的响应原文没有落盘（只存在于操作员记录），G1/G2 的文案判据区分不了「容器门」和「存在性门」，G4 必须连前置条件一起断言。
- **判据外发现**：
  - **被拒的变更不落 action、也不进 broker 日志（已修）**：`actions.json` 26 条全是成功事件，三次 409 一个痕迹都没有。三条容器守卫（写范围 / artifact / bot 关闭）的拒绝现在各写一条 `guard_refused`（`related_id=任务`，summary 带被我撞的那条路径与 lane id），测试 `TestContainerGuardRefusalsLeaveALedgerRow`（去掉记录即红）。注意：记录必须发生在 `rollbackTask()` **之后**——`brokerTaskMutationSnapshot.restore` 会把 `b.actions` 重置回快照，先记就白记。
  - **一个任务可以写另一个任务已交付的 `artifact`**：`GUARD-4` 覆盖了 `GUARD-12` 的 `team/accounts/guard-quote.md`（33 → 1642 字节）。`mode=create` 没有独占校验；是否该独占需要产品决策（`mode=append_section` 的共享写法是有意的）。
  - **决策批准路径的同一处拒绝原本不留痕（已修 `01bfcb8`）**：`recordTaskDecisionInternal` 在 `childOwningArtifactLocked` 拒绝处不动笔，人从审批界面撞门时什么都不留。该门跑在任何 mutation 之前，没有 `rollbackTask()` 的快照恢复问题，`recordGuardRefusalLocked` 直接落盘即可；`TestDecisionApproveRefusesALanePathAsTheIssueArtifact` 断言这行（去掉记录即红 `got 0`），`TestDecisionApproveWithoutALanePathLeavesNoGuardRow` 钉住干净的 approve 不留行。
- **同轮查出来的缺陷（已修）**：裸 `GET /requests` 静默答「没有请求」——`normalizeChannelSlug` 把 `""` 洗成 `general`，那条「channel is required」的 400 分支读的是洗过的值、永远是死代码，于是裸调用按 `general` 过滤出一条空列表（驱动脚本因此误报 `NO CARD` 两分钟）。现在判断改用原始 query 值，裸调用报 400 并说清该传什么；`?channel=`/`?scope=all` 不受影响。**同一处缺陷还有第二面**：`byID` 也读洗过的值，裸 `?id=<rid>` 同样返回 `200 {"requests":[]}`——而 `web/src/api/botKnowledge.ts` 的 `getPendingPromotion` 正是这样轮询的（只传 `id`+`viewer_slug`），于是凡是落在非 `general` 频道的待批准提升快照都拿不到——而提升请求正是建在 agent 自己的频道里；一并改成原始 query 值，可见性改按每条请求单独判。测试 `TestGetRequestsWithoutAChannelSaysSo`（裸列表 400 / 裸 by-ID 命中 / 未知 id 空列表；去掉修复即红）。详见 `…/2026-11-10-guards/observations.md` §6。



### Journey 13 复核（2026-11-10，形状没钉死，模型把两份交付物塞进一个文件、一条 lane 都没拆）

prereg `core-loop-runs/2026-11-10-journey13/prereg.md`，被测 `6caf6cf`，判分 `…/journey13/scorecard.md`（独立子代理，含操作员勘误）。场景同 J12，模型 `opencode-go/longcat-2.5-free`。

- **站住的**：
  - **单任务交付闭环成立**：`JOURN-1` 自己写 `team/accounts/acme-renewal.md`（wiki `367d955`，64 行），`submit_for_review` → `review` → `decision`（msg-21，09:06:17Z），`request-22`（`decision:JOURN-1`）同秒出现，人工点 `approve_decision` → 200，`done/approved`，wiki 落 `team/decisions/JOURN-1.md`（`29c8463`）。
  - **`6caf6cf` 的 bot 关闭门这一轮没有被触发**（模型根本没走任务端点关容器——它压根没造容器）。
- **没站住的**：
  - **容器守卫 / 聚合谓词 / bot 关闭门全部没有对象（重，场景问题）**：`tasks` 里只有 `JOURN-1`，**零条 lane**，两份交付物合写在一个文件里。V1/V2/V7 判未观察，V3 通过，V4 判失败（卡文为单文件口径 `The deliverable is team/accounts/acme-renewal.md.`），V5 通过，V6 部分。**J11/J12 同样措辞下都拆过两条 lane，说明「拆不拆」是模型自由裁量**。
  - **`definition-gap` 卡重复追问已给事实（高，模型行为）**：`request-14`（09:05:26Z）的「缺失细节」是 `access needed: 用户已提供的合同事实（方案/价格/周期/续约调整/联系人）`，而这些事实 09:01:09Z 已经在 `request-4` 的 `custom_text` 里给全了。代码侧 `raiseDefinitionGapInterviewLocked` 只读 `task.Definition.AccessNeeded`（`broker_requests_interviews.go:528-545`），是 cos 在自己的 `action=define`（msg-18/19）里把已拿到的事实写成了「还缺的访问」。不是 broker 的重复追问，是模型的 define 内容错。
- **干预 4 次 vs 预算 3 次**：request-4 事实、request-9 计划、request-14 definition-gap、request-22 决策卡。第 3 张由模型的 define 内容逼出来。
- **顺手修的一处（第三条写路径）**：`attachSelfHealCompletionToParentLocked` 会把自愈 lane 的文件盖到父 Issue 的 `artifact` 上——J11/J12 的守卫没覆盖的第三个写入者。现在父 Issue 的 `artifact` 一律保持为空，文件留在 lane 上，由决策卡聚合。测试 `TestSelfHealLaneDoesNotStampItsArtifactOnTheContainer`。见 §9.23。
- **这一轮证不了**：容器 409、聚合卡正文、bot 关闭门——**J14 把形状钉死（两条子任务、两个独立文件）就是为这个**；F-i 新增：钉死后仍不拆 lane 就算场景约束无效。



### Journey 10 复核（2026-11-10，父 Issue 的 decision 卡）

prereg `4f161fd`（+ `d6ca54f` 记录赛前模型替换），被测 `52456d3`（+ `76e6171`、`fdb6fa8`），判分 `core-loop-runs/2026-11-10-journey10/scorecard.md`（独立 reviewer）。场景与 J7/J8/J9 逐字相同，模型换成 `opencode-go/longcat-2.5-free`（`crouton-alpha-free` 被上游撤掉，§9.21）。

- **站住的**（这一轮唯一有正向证据的）：
  - **D6 路径钉真机成立**：两条 lane 声明路径与落盘逐字一致（`dce4efc` 617 B、`5c3c6b3` 910 B）；探针 `JOURN-70`/`JOURN-73` 用相似名（`pin-probe.md` / `pin-probe-v2.md`）各自落地（`355e3b7`/`0a4c053`），邻居未被追加；未声明路径 403 点名两条路径且未落盘。**J9 的「200 但落邻居」没有复现。**
  - **D5 计数准确**：`decomposed into 2 sub-issue(s); 1 still open` 与实际一致（规划者这次只拆 2 条）。
  - **D4 提前唤醒**：`JOURN-21` 02:46:55 落地 → 父 Issue 与兄弟 lane 的 `task-follow-up` 都提前到 02:48:55。
  - **计划卡只有一张**、`task_not_decomposed` 0 条、DecompositionCheck 正常收尾。
- **没站住的**：
  - **D2（重）父 Issue 进 decision 后 10 分钟没有卡 → 已修（§9.22）。** `JOURN-14` 02:54:20 进 `decision`，`requests` 全程只有已答的 `request-17`；根因是 `hasApprovableDeliverable` 只读任务自己的 `Artifact`/`ArtifactNotRequired`，而父 Issue 两者皆 null（它的两条 lane 都交付了）。同一处还有第二个后果：人工补点 `POST /tasks/JOURN-14/decision {approve}` 返 409，文案把下一步交给 owner。现在卡的谓词与决策 approve 门都认「父 Issue 有已交付子任务」，卡正文逐条点名各 lane 的文件。测试 `TestDecisionCardRaisedForParentIssueWithDeliveredLanes`（走真实路径 submit_for_review → 无 reviewer → 收敛 → 卡 → approve）、`TestDecisionCardSkippedForParentIssueWithHollowLanes`。
  - **V4 的判据错（判据已改，代码不动）**：02:49:02 `action-32/33` 两条 `@cos/@ae still needs to move …` 是两条到期的 `task_follow_up` 按点叫人的记录，记录后同一函数立刻 `deliverTask`：cos 回合 02:49:03 起、ae 02:50:03 出稿。`actions[].kind=watchdog_alert` 同时表示「我按点叫了人」，所以「0 条」不可能成立（J9 也是这么红的）。口径见 `core-loop-runs/README.md` §6 第 5 条：判「叫不动」，不判条数。
  - **`recheck` 唤醒后重复播报（未修）**：03:09:03 `recheck` 触发，ae 连发 `msg-63/64/65`（03:09:11–03:09:19）复述同一结论，无新事实。
- **留下不对称（未修，故意的）**：`/tasks` 的 approve 路径（`MutateTask` 的 `complete`/`approve`）对父 Issue 仍会 409 并可能标 `hollow_completion=true`。人类批准 Issue 的行进路线是 decision 端点（`RecordTaskDecisionWithComment`，§9.19），这次只修了那条；两条路径的判定要保持一致得等它们合并，或者在同一处抽一个 `deliveredAggregateLocked`。记在这里，不假装已经收口。
- **这一轮证不了**：任何 `decision:` 卡的去重（V12）、点击闭环（V13/V14）——本轮卡从未物化；create 守卫的「撞后改做法」（V3/V6，场景内 0 次撞）；`ra_` 收 done 的拦截（V8，场景内 0 次）。
- **判据本身的教训**：prereg 把 V11 的对象写成「lane 进 decision」，而 office 模式的 lane 交付后直接 `done/approved`（`taskNeedsStructuredReview` 判假），**没有任何 lane 会进 decision**；进 decision 的只有父 Issue。下一轮 prereg 要把对象写成「任一任务（lane 或 Issue）进 `decision`」，否则同一条缺陷会以「未观察」的形式年年躲过去。

### 2026-11-09 复核

- **J3-1 已修**：`runHeadlessOpencodeTurn` 在回合开始时算 `planning := l.resolveTurnPosture(ctx, slug) == posturePlan`，回合成功且有正文时 `emitHeadlessPlan(...)` 出计划卡，并在末尾 `l.raisePlanApprovalAfterTurn(taskID, slug, text)`（空计划不发、非 planning 任务 `RaisePlanApproval` 自会 no-op）。测试：`TestOpencodePlanningTurnRaisesPlanApproval`（去掉 raise 调用即失败：`raised 0 plan-approval interviews, want 1`）、`TestOpencodeRunningTurnRaisesNoPlanApproval`。opencode 仍无原生 plan/read-only 模式，姿态只改结果处理，不改权限——这一点与 Claude/Codex 不同，注释已写清。
- **J3-2 已修**：`processTaskJob` 在 done/blocked 判定之后加一道门——`taskAwaitingReview(lifecycle)`（`review`/`decision`/`changes_requested`/`rejected`，定义在 `broker_lifecycle_transition.go`）且 job 不是 `task_followup` → `UpdateSchedulerJobState(..., "done")` 直接收尾，不派活、不写 `task_stalled`、不发 `watchdog_alert`。任务再次进入 `in_progress` 时 `scheduleTaskLifecycleLocked` 会重新装回 recheck，故不会漏看。测试：`TestSchedulerQuietForTasksAwaitingReview`（四个状态各自跑一遍；去掉门槛即失败：`delivered 1`）、`TestSchedulerStillAlarmsParkedTask`、`TestSchedulerStillWatchesPlanningTasks`。范围刻意不收 `drafting`/`intake`：那两个状态下有 owner 却没动，正是人该看见的松散端。

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

### 5. ~~已公布事件的实际值来源~~ ✅ 已完成（2026-11-08，选一手来源）
- 现状（修复前）：digest 只有前值/预期，没有 actual；预期差段在 artifact 里是占位。**FF 周历 XML 里 `<actual>` 节点数为 0**（实测），所以过去的「实际值」只能从新闻标题拼——就是单位混用的来源。
- 拍板：走**一手来源（央行/统计局）**，一个序列一条抓取逻辑，不再解析 FF 页面、不再拼新闻数字。
- 做法：`~/.wuphf/data/forex-bridge.py` 新增 `primary_actuals()` —— BLS v1 公开 API（无 key，25 序列/天/IP）取官方指数水平，桥自己算 `(cur/prev-1)*100` 并 `ROUND_HALF_UP` 到 0.1；缓存 6h（`bls_actuals.json`）；映射见 `RUNTIME-API.md` §13.7。
- **验收（2026-11-09 实测，`~/.wuphf/wiki/team/forex-bridge-digest.md`）**：四个事件都出现 actual，且与 BLS 发布文本/表格逐条对齐 ——
  - `CPI m/m` 0.6%（`cpi.t01` Aug→Sept 0.6；正文 "increased 0.6 percent"）
  - `Core CPI m/m` 0.3%（正文 "less food and energy rose 0.3 percent"）
  - `PPI m/m` 1.3%（`ppi.t01` FD 4 末列 1.3；正文 "final demand increased 1.3 percent"）
  - `Core PPI m/m` 0.7%（`ppi.t01` FD 49104 末列 0.7）
  四个序列与 FF 前值的口径差全为 0.0（digest 只在非零时才打 `口径差`）。
- 踩过的坑：`WPSFD4131` 看着像「最终需求除食品能源」，实际算 0.2%，发布表是 0.7%；正解是 `WPSFD49104`。序列 ID 必须拿发布表核，不能凭命名猜。
- 未接来源：GBP GDP（ONS）、USD Claims（DoL 539 需要季节性调整）、央行讲话/纪要——这些事件 digest 不写实际值，尾注要求写「实际值未收录」。DoL ETS 的 `ar539.csv` 试过：13 MB 全州明细、无全国行、且是未季调，与新闻口径不可比，放弃。

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
    - **遗留产品缺口（2026-11-02 已补）**：`internal/workspaces` 在 Windows 上 `defaultSpawn` 曾返回「本平台不支持起 broker」的哨兵错误，多 workspace 的创建/恢复不可用。现在 Windows 走真实 spawn：平台无关部分（argv/env/日志/PID/就绪探测）在 `spawn_common.go`，Windows 侧用 `CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS` 起子进程，成功返回后 `Release` 掉进程句柄，子进程在父进程退出后存活；超时/写 PID 失败按 POSIX 语义 `Kill` 并 `Wait`。**没有用 Job Object**：`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` 会在父进程退出（最后一个句柄关闭）时杀掉子进程，正好和"父退出后子进程存活"冲突；不加该标志的 job 又会随父句柄关闭而消失、什么也换不来。代价是 pause/kill 只终止直接子进程，孙进程仍会逃逸（与 §7e 的 P3 #7e 同一批）。
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
- ✅ **同类隐患已收口（2026-11-03）**：`configureHeadlessProcess` 在 Windows 仍是空操作，边界改由 **Job Object** 提供——`startHeadlessProcess` 在 `cmd.Start()` 之后立刻把子进程塞进一个 `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` 的 job（`headless_process_windows.go`），进程句柄退出即关 job、连带清掉它派生的一切。
  - `terminateHeadlessProcessPID` 现在两段杀：先 `taskkill /T`（覆盖 Start→Assign 之间那几微秒里新建、还没进 job 的进程），再 `TerminateJobObject`（覆盖树走不到的形状：中间父进程已退出的后代、pid 已无法解析的孙进程）。job 句柄按 pid 查，不重新遍历树，所以 pid 复用不会指错对象。
  - 全部 headless 起进程点改走 `startHeadlessProcess`：claude / cli-agent / codex / opencode / session runner / `custom_app_dev.pipeAndWait` / `runAppBuildStep`（`cmd.Run()` 拆成 Start+Wait）。
  - 回归测试 `TestHeadlessJobKillsLeftoversWhenTheToolExits`：shim 自己正常退出（不是被杀），它派生的 powershell 必须随之消失。反向验证：把 attach 短路掉，18.6s 后报 `leftover … outlived the tool that started it`；恢复后 2.5s 通过。
  - **行为变化（有意）**：turn 里被工具拉起的后台进程不再活过这一轮——过去它们持着 stdout 管道、sqlite 和调度队列，正是 8.5h 挂死与 `database is locked` 的来源。需要长期存活的东西必须起成 broker 管理的服务，不能是 turn 的子进程。
  - 已知窗口：`CreateProcess` 到 `AssignProcessToJobObject` 之间新生的进程不在 job 里，只能靠 `taskkill /T` 在父链还在时抓到；彻底消掉需要 `CREATE_SUSPENDED` + `ResumeThread`，而 os/exec 不暴露线程句柄，未做。
  - 失败静默：job 建不出来（受限于容器/父 job）时退回纯 `taskkill`，turn 截止语义不变。
  - **已部署（2026-11-03 11:04 北京）**：`ce7221d` 构建后换二进制，broker 以 watchdog 同参数（`--broker-port 7892 --provider opencode --no-open` + 两条 `-RedirectStandard*`）重启；`/health` ok、`scheduler: loop started`、`DRIFT: none`。
  - 线上验证（两次真 turn，`headless-codex-latency.log`）：`ACMER-3290` ok 14.8s、`ACMER-3295` ok 10.5s，两者都是 `created_by=system` 的部署冒烟（`hollow_completion=true`，数空壳完成时忽略）。`Get-CimInstance` 按创建时间比对：**11:04 之后没有任何 bun/node/opencode 进程留下**。
  - **部署顺序的坑（本轮踩到）**：先 build 后 commit，drift 的「二进制落后于代码」立刻红（它只比二进制 mtime 与之后的非测试 Go 提交）。正确顺序是 commit → build → 部署；已经先 build 的，重建一次即可（两次构建的 sha256 不同，因为 `-buildvcs` 把 revision/working-tree 状态打进了二进制）。
  - 现存历史遗留：10-28 至 11-03 10:21 的 bun/node 孤儿仍在（老版本 turn 的遗留），本 fix 只对未来生效；清不清由人决定，watchdog 重启时会顺带清 `opencode.exe`。

### 7d. 唤醒掉线的代码级根因（2026-10-30 修复）
- `enqueueHeadlessCodexTurnRecord` 的同任务丢弃分支在 `headless.mu` 持有状态下 `return`（该函数无 defer，其余分支都显式 Unlock）。丢一次重复 lead 唤醒 → 锁永久泄漏 → 后续每次入队/`stopHeadlessWorkers` 阻塞，表现为「进程在、调度在、消息不进 turn」。
- 与 P0-0 观察一致（05:11 空闲 16h 后首次唤醒失败）。已修并部署（22:31 重启后 cos turn 正常，queue_wait 46ms）。

### 7b. 三条僵尸任务（更新 10-30）
- ✅ ACMER-1742：plan 已批准（`lifecycle: running`），data-keeper 可收尾。
- 仍存：task-skill-1852 / 1857（in_progress，零消息）。

- ACMER-1742（planning, data-keeper）、task-skill-1852 / 1857（in_progress, cos，零消息）。
- 无操作员取消端点；交给 CoS 关闭可能再触发循环，暂记录。
- 若要清：让 CoS「取消 task-skill-1852/1857，无产出；1742 保留」并观察一轮。

### 7f. 跨 DM 唤醒的回答投不出去（2026-11-01 巡检确认；2026-11-02 修了任务侧根因）
- 现象：`wake-lead: re-delivering specialist completion from @forex-watch` 把 cos 唤到 `forex-watch__human` 里的任务上；cos 干完活要回帖时，`live-chat-relay-post` 与 `opencode_fallback-post` 双双 `channel access denied`（11:09:34、11:19:02 各一次），正文只留在 `headless-codex-cos.log`，人看不到。cos 自己也说了一句「@cos 在此 DM 频道无写入权限」。
- 规则本身没错：DM 只认两名参与者（AUDIT-2026-10-31 §2.5 已记为既定规则），拒绝是预期；问题是**回答没有出口**，同时每次唤醒刷两条 error。
- **已修（2026-11-02）：任务不该产生在别人的 DM 里。** `createSkillRunTaskLocked` 直接沿用调用方的频道，而 skill 多半从调用者自己的 `#<bot>__human` 触发，任务 owner 却是 office lead——另一个 bot。DM 只认两名参与者，所以 owner 既写不进去，也不能被 `ensureTaskOwnerChannelMembershipLocked` 拉进去（它对 `botSide != owner` 直接 return）。现在创建时就走 `preferredTaskChannelLocked`，它会 `reHomeTaskOutOfHumanDMLocked` 把任务搬到调用者⇄owner 的 pair DM，两人都能写。回归 `TestSkillRunTaskReHomedOutOfInvokerDM`（去掉 fix 必红）/ `TestSkillRunTaskKeepsWritableChannel`（可写频道不被搬动）。
- 实证：`task-skill-2923` 卡了 38 轮——`channel=data-keeper__human`、`owner=cos`，每轮都正确因缺 `object_name`/`meeting_date` 中止，每轮都无法在任何它能命名的频道里报告，结论只落在自己的 notebook。已 `blocked` 止损。
- **仍未修**：已存在的此类任务（含 `wake-lead` 把 cos 唤进别人 DM 的老任务）。上面 A/B/C 三选一仍然待拍板——修的是「新生」，不是「存量」。

### 7j. `block` 停不住任务，且没有取消端点（2026-11-02 实测）
- 第一次止损用 `POST /tasks/task-skill-2923/block`，返回 200、`status=blocked`，看起来停了。**实际没停**：`scheduleTaskLifecycleLocked` 对除 `in_progress` 之外的所有状态走 `default` 分支，主动排一个 `recheck` job 并重设 `follow_up_at`/`recheck_at`/`reminder_at`。blocked 在设计上就是「可恢复、等上游」，所以它必然被再唤醒——只是把 `task_follow_up` 换成了 `recheck`。实测 block 之后又跑了 2 轮（executions 39 → 41）。
- 真正的终态只有 `done / completed / canceled / cancelled / archived`（`isTerminalTeamTaskStatus`），进入终态才清空调度并 `completeSchedulerJobsLocked`。
- **没有操作员取消端点**：`/tasks/{id}` 只接受 GET，子路径只有 `block|resume|decision|comment|activity`。要停一个任务只能停 broker、手改 `broker-state.json` 的 `status`、再启动。本次即如此（备份 `broker-state.json.bak-cancel-2923`）。
- **重启会把 blocked 改回 in_progress，`lifecycle_state` 仍是 `blocked`——根因已定位并修复**：`reconcileOrphanedBlockedTasksLocked`（`internal/team/broker_defaults.go:622`）本意是解开「依赖已全终态、但没收到 unblock 通知」的历史孤儿。它把「没有未解决依赖」当作症状，而一条**根本没有 `DependsOn`** 的任务同样满足这个条件——于是每次启动都被当成孤儿解开。`task-skill-2923` 就是这样回来的：手工 block、无依赖、重启后 `status=in_progress`，`follow_up_at` 重新排上，循环本会继续。修法是加一条守卫：`len(t.DependsOn) == 0` 直接跳过，没有依赖可「解决」就不是这个函数该管的。回归 `internal/team/broker_blocked_reconcile_test.go` 三个方向（手工 block 存活 / 真孤儿仍被解 / 依赖活跃仍 blocked），撤掉守卫必红。
- **`blocked` 会主动重排 recheck 的说法成立但属设计**：`scheduleTaskLifecycleLocked` 对除 `in_progress` 外的所有非终态走 `default` 分支排 `recheck`。语义上 blocked 是「可恢复、等上游」，所以它会被再唤醒。改它会影响所有等待型任务，未动；若将来要「blocked 即真停」，应新增独立的暂停态而不是改这里。
- **已修（2026-11-02）**：`POST /tasks/{id}/cancel`（`CancelTask`，`internal/team/broker_tasks_lifecycle.go`）。落终态 `canceled`，清空四个调度字段并把 scheduler job 置 `done`；幂等（已终态返回 `changed=false`）。为此新增 `LifecycleStateCanceled`——此前 `status=canceled` 反推成 `approved`，在按 `lifecycle_state` 渲染的看板上「停掉」和「完成」长得一样。回归 `broker_task_cancel_test.go`（终态且清调度 / 幂等 / 重启后仍派生为 canceled 且不可执行）。
- 顺带修了 `normalizeTaskPlan` 的无条件覆盖：它最后一行 `task.pipelineStage = taskStageForStatus(task)` 会把 `applyLifecycleStateLocked` 刚按 forward map 写好的终态 stage 盖成模板的 open stage（`canceled`/`archived` 都变成 `triage`），导致反向映射回不到原状态。现在终态跳过重算。
- **已修（2026-11-02）**：停滞任务自动退出轮转。`teamTask.StalledTurns` 记录**连续**无持久痕迹的轮次数（不封顶、不随 journal 轮转），任何有产出的轮次归零；到 `maxStalledTurns = 8` 时由 `recordTaskLedgerEntry` 调 `CancelTask`，理由写进 `Details`。判据用 `ok (no durable trace: …)` 而不是次数——持续产出改动的长任务不该被杀。回归 `task_ledger_stall_test.go` 四条，短路 escalation 必红。决策记录见 `DESIGN-DECISIONS.md §8.9`。
- 顺带：失败重投不降噪（审计建议 N 次后停），本轮未动。
- **已修（2026-11-02）**：`TestObsidianWatcher_NotifyWriteFiltersOwnEvents` 全量跑稳定红的问题。根因不在产品侧——self-write 过滤一直是对的，是测试**在自己的第二个写入开始前，第一个写入的回调还没提交完**。`waitForHEADChange` 在第一个 commit 就返回；`fire()` 此时可能已通过 `recent` 检查、正在跑 git，而 git 比任何「HEAD 静默」窗口都慢，所以 HEAD 看着稳定、commit 仍在飞行，落地时正好落进 `requireHEADStable` 的观察窗。修复是加 `waitWatcherIdle`（等同包的可观测状态：`timers` 为空 + `pending.Wait()` + 复查），在两个 fixture 的第一步之后调用。先试过「等 HEAD 静默 200ms」的版本，`-count=3` 仍 1/3 红，证明要等的是回调而不是 HEAD。验证：`-count=3` 绿，`go test ./internal/team -count=1` 全量 **ok 555s**（此前两次全量分别红在 `490945f → bedcb3f`、`46692f1 → 65a8bbd`）。
- 同类已修（2026-11-01）：app 提案审批后建 App Builder 任务也把空频道默认成 `#general`（`broker_apps_proposal.go`）。现在按「提案所在房间 → 建者（app-builder）的主场 → 记日志放弃」解析；提案房间在审批前退役（`#general` 退役、成员删除带走 DM）时不再把任务和错误消息塞进死房间。新增 `TestProposeAppApprovalSurvivesARetiredRoom`。

### 7g. 空壳完成（2026-11-01 诊断；② 已落地，① 已做，③ 并入 J2-6）
- 复发三次同款：`ACMER-2757`（10-31，跑满 10 分钟超时）→ `ACMER-2764`（stuck 重试，补答后关单）→ `ACMER-2785`（11-01，24 秒关单，正文只有一句「已完成」）。
- 链路：**问题被建成了 Issue**（`created_by=human`、`task_type=issue`、`title==details`，没有 Definition）→ 完成零门槛（`spec {}`）→ 一次 `team_task complete` 即 done → 完成 hook 广播 `delivered`，频道里没有正文 → 下一轮把它当「已答」的证据（`ses_e6aa388e4ffe65ZEW1dTv7z8qQ` 的推理原文：*"The answer was already posted in chat."*）。
- 结构原因：artifact 门禁只对**有 Definition** 的任务生效（`broker_tasks_mutation_service.go` 的 artifact gate），人工/对话框建的 Issue 全是 `Definition=nil`；`wiki_artifact` 强制校验只挂在 agent_routine 生成的任务上。
- **已做（②，`broker_types.go` + `broker_tasks_mutation_service.go`）**：任务转到 done 且 `artifact` 为空时，打 `hollow_completion=true` 上线（`teamTask.HollowCompletion`，wire 键 `hollow_completion`）。不改完成语义，只让这类完成可被计数。测试 `TestCompleteWithoutArtifactIsMarkedHollow`（空完成=true、带 artifact=false、字段上线）。
- 巡检口径：`curl /tasks?all_channels=true&include_done=true` 里 `status=done && (hollow_completion || artifact 为空)` 的条数。历史任务没有该字段，等价于 `artifact` 为空——2026-11-01 实测 52 个 done 里 47 个如此。
- 部署冒烟样例：`ACMER-2789`（`smoke: hollow_completion 验证`）是上线验证用的，不是第五次事故，数数时忽略它。二进制 16:30 构建、16:36 起由看门狗拉起（PID 39216，`provider_model=opencode-go/beam-free`）。**同类的部署冒烟还有 `ACMER-3290` / `ACMER-3295`**（2026-11-03 Job Object 上线验证，`created_by=system`），数空壳完成时一并忽略。
- **① 已做（2026-11-08，前端入口分流）**：`/tasks/new` 提交时若 **Details 为空**且标题是疑问句（`?`/`？` 结尾、CJK 疑问开头词或句末「吗/呢」、Latin 首词 who/what/how/is/can…，见 `web/src/lib/questionIntent.ts`）→ 不建 Issue，改出 `issue-new-question-triage` 面板，两个出口：「Ask in chat」把原句 `postMessage` 到 `<one_on_one_agent>__human`（无 One-on-One 时退回调 assignee，再退 `cos`）并跳到该频道；「File it anyway」照旧建单。Details 一填就不再拦——验收标准里合法地会有问句。只做入口，服务端仍允许 question 型 Issue（MCP/API 直接建单不受影响），因为坏的是默认路径而不是能力。测试：`questionIntent.test.ts` + `TaskNewForm.test.tsx`（拦下 / 坚持建 / 转聊天），web 全量 332 文件 3651 passed。实机（7891，2026-11-09）：输入 `What is our Q4 renewal rate?` → 出现 `issue-new-question-triage` 面板 + 两个出口，任务列表里没有它；填 Details 后面板立即消失、正常建单（探针任务 `ACMER-3400` 已 cancel）。
- **③ 不再单独做，由 J2-6 覆盖**：已交付的同题 create 直接 409 点名，父 Issue 每条 lane 都交付后建新 lane 同样拒绝；比「窗口期复用旧任务」更硬，也不再需要窗口概念。
- 备注：`task_type=question` 这条备选未采——Issue 里没有「答题」的终态与产物语义，加一个类型只会把同一问题推向另一条死路。

### 7h. 有 shell 的 bot 可以直接写 wiki 文件，绕过所有写入门禁（2026-11-01 发现）
- 现象：`/wiki/write-task` 明确拒绝了 `team/reviews/…` 以外的路径（`wiki write-task refused` 有日志），但 cos 仍把 `reviews/ACMER-2832.md`、`reviews/ACMER-2847.md` 写进了 `~/.wuphf/wiki/`，git 里以 `recover from crashed write` / `external edit` 形式出现。
- 原因：bot 的 shell 工具能直接对 `~/.wuphf/wiki/` 落盘，`/notebook/*`、`/wiki/write*`、promote 审批这些门禁只在 API 层生效。
- 影响面：任何 bot 都能改 `team/` 正文、伪造 learning；`wiki_artifact` 校验也因此可能被"先落盘再 complete"绕过（本轮的 `ACMER-2852` 就是先被忘接写成 `reviews/` 再由人把文件挪进 `team/reviews/` 才关单）。
- 未做：给 wiki 目录加文件级监控并告警（`external edit` 提交已存在，可作为探针），或收紧 bot 的 shell 工作目录。属安全边界改造，需拍板。

### 7i. broker 的 run 历史是 20 条环形缓冲，重启即空（2026-11-02，N9 过程中确认）
- 现象：`/scheduler/{slug}/runs` 每个 job 最多留 20 条，broker 重启后全部清空。`last_run` 同理，7 个 `system_managed` system cron 全部为空。
- 影响：任何"成功率/平均间隔/连续失败次数"指标都算不出来——环形缓冲里 20 条对一个日频 job 是 20 天，对一个分钟级 job 是 20 分钟，口径完全不同。N9 因此只报 `runs_seen` 这个**原始事实**，不算比率。
- 已做的规避：记分卡每次运行把观测追加到 `~/.wuphf/data/routines/history.jsonl`，趋势靠这个文件而不是 broker。但它只在脚本跑的时候有数据点（现在每天一条）。
- 未做：把 run 历史落 SQLite（`~/.wuphf/broker.db` 已有表），或至少让环形缓冲按时间而不是按条数保留——后者能保证"最近 7 天"这个查询永远有意义。属存储层改造，需要拍板。

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
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/longcat-2.5-free`（用户级环境变量 + 看门狗脚本 + 启动参数三处一致，2026-11-10 起）；看门狗兜底 `opencode-go/iceman-contributor-free`。`crouton-alpha-free` 已被上游撤掉（§9.21）。；**2026-11-11 用户拍板继续用免费模型，不换付费**（DESIGN-DECISIONS §9.29）。当日探针：longcat 429、iceman 200、beam-free 200，主模型 429 时每回合先付 4 分钟零输出守卫 |

| broker token | `%TEMP%\wuphf-broker-token-7892`，**每次重启都变** |
| 桥 | Windows 计划任务跑 `forex-bridge-run.vbs`，每 6 小时；健康时无 `wiki/team/alerts.md` |
| 记分卡 | 计划任务 `WUPHF-RoutineScorecard` 每天 09:05 跑 `~/.wuphf/data/routine-scorecard.ps1`（collector + `--publish`）；日志 `~/.wuphf/logs/routine-scorecard.log`；判据 `python scripts/office/check_routine_scorecard.py` |
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
   - **2026-11-02 发现 10-31 只接了一半**：`Invoke-DriftCheck` 定义了，但主流程从未调用它——只有 `-DriftOnly` 分支调用，而计划任务的命令不带该参数。`watchdog.log` 里 4 条 drift 记录全部来自 10-31 手工测试（`t-alerts.md`、`t51-alerts.md`），此后 15 分钟一次的定时运行只做 liveness。
   - 已补：抽出 `Invoke-StateChecks`（drift + routine gate），在无 due job 分支与正常分支末尾调用；`Restart-Broker` 改为返回是否真的重启了，重启后跳过状态检查（broker 正在起，检查只会因错误原因失败）。
   - 同时接入 N9 的 `check_routine_scorecard.py`（`Invoke-RoutineGate`）：快照超过 26h 即告警，按输出 SHA256 去重。**这是记分卡的自指闭环**——否则记分卡本身的调度挂了，没有任何东西会红。
   - 实测（2026-11-02 12:15）：快照改到 10-01 → `routine gate FAILED: snapshot is 772.3h old (limit 26h)`；同故障再跑 → 0 行（去重）；恢复 → 一行 `routine gate: recovered` 且状态文件删除。
   - 备份 `~/.wuphf/data/broker-watchdog.ps1.bak-20261102`。
2. ~~**入口裁剪**~~ —— **已做（2026-10-31）**：`docs/agents/INSTRUCTIONS.md` 641 行 / 31.0 KB → **118 行 / 6.2 KB**，只留硬规则（Working Rules、Git/PR、Quality Bar、Wuphf Commands、Landing、Lint）+ 首屏「按任务读哪份」路由表。
   - 散文（Nex Context、Triangulation、Verification agents、When to use which、Screenshots、Storybook、Multi-round review、Sub-agent dispatch contract、共享树 12 节、Worktree parallelism、Demo+oracle）原样迁到 `docs/agents/shared-tree-lessons.md`（544 行 / 26.4 KB），分 A 基座 / B Wuphf 仪式 / C 共享工作树三组。
   - 校验：原文件 511 条非空行中，仅 21 行未出现在拆分后（全部属于被重写的 Landing Changes 段），其余逐行保留；29 个章节标题各自只落在一个文件里。
   - 顺带修掉一处规则冲突：旧入口写「commit to main 并直接 push origin」（原作者仓库规则），对本 fork 是错的（`push.default nothing` + origin 无写权限），改为对齐 `OFFICE-RULES.md` §6 的 `mine`-only 规则。
   - 路由表就是索引，没有另建 `docs/office/INDEX.md` —— 多一份要被记住的文件就多一处腐烂点。
3. **policy 全文本**不入 git（每次 PATCH 生成新 revision + id）。repo 只记意图与 id，生效与否一律 `GET` 验证。
4. ~~**空壳 workflow 收尾**~~ —— scheduler job 无 DELETE 端点，保持 disabled + 清单 `requires_file` 把关。
   skill `daily-opencode-model-snapshot` 于 **2026-11-01 重新 archive 并验证落盘**：10-31 那次归档只改了内存/state，wiki 文件没重写（mtime 停在 10-29、frontmatter 仍 `active`），broker 重启后被 scanner 按文件重新导入 → 复活成 active。重做后 `team/skills/daily-opencode-model-snapshot.md` 的 `metadata.wuphf.status: archived`（提交 `0062abe`），`/skills` 回到 3 个 active。验收命令见 `RUNTIME-API.md` §7。
   **教训**：归档/禁用这类"看起来生效"的操作，必须读回落盘产物（frontmatter、文件 mtime、git log），否则重启就是一次回滚。

### Journey 17（2026-11-11，本轮作废：上游限流）

`core-loop-runs/2026-11-11-journey17/`：prereg `5696d9c` 保留，`ABORTED.md` 记根因。模型一次都没产出——02:07 起两回合都是 `opencode-go/longcat-2.5-free` 的 `stream error`，直连端点复现 `429 rate_limit_error`（02:29 / 02:31 两次）。办公室的自愈路径完整：回合失败 → `JOURN-1 blocked` → 自动止血单 `JOURN-4` → 人类催办后重起回合。**没有判据可判，不计入跨轮对比**。配额恢复后按 prereg 原样重跑（命令写在 ABORTED.md §怎么重跑）。

运维含义（已修正，02:44 实测）：429 是**按模型**算的，不是按 key。同一时刻 `longcat-2.5-free` 429、`iceman-contributor-free` 200、`beam-free` 200、`crouton-alpha-free` 410（已废弃）。所以「换一个免费模型」确实能救单个模型限流，前提是**代码肯去切**——原实现把 429 当普通失败，直接 blocked，fallback 一次都没被调用（见 DESIGN-DECISIONS §9.28 的修法）。

**已修并真机复验（`becd286` + `headless_codex_queue.go`）**：marker 认 `rate limit` / `rate_limit` / `too many requests`；超时分支额外判「这一回合有没有模型输出」（只认 `text`/`tool_use`/`tool_result`/`plan`，runner 死前补发的 `error`/`manifest` 不算），零输出就换 fallback 再试一次。12:14:34 实测：longcat 仍 429 → 跑满 10 分钟 → 切 `iceman-contributor-free` → `status=ok total_ms=76964`。未解决的是那 10 分钟等待。两条路都试过了：① opencode 二进制里没有任何 retry 相关环境变量；② 把 `provider.opencode-go.options.headerTimeout=60000` 写进每机器人配置（schema 明确支持），配置落盘了但真机行为不变——只对自定义 provider 生效，自带 provider 不认，代码已回滚。剩下的唯一杠杆是 broker 侧「零输出预算」：`WUPHF_TURN_SILENT_BUDGET`（默认 4 分钟，`0` 关闭）——回合连续 4 分钟没有任何模型输出（text/tool/plan，记账事件不算）就取消这次尝试、把同一回合交给 fallback 重排一次；只在「有 TaskID + 配了 fallback + 非 fallback 回合」时启用，吐出任何输出后立刻失效。单测 `TestHeadlessQueueSilentStallCancelsEarlyAndRetries` 覆盖（真 worker，600ms 预算，断言正好两次尝试且日志点名 fallback），**已真机验证**：2026-11-11 14:27:41 +08:00 隔离办公室 `:7910`（`longcat` 仍 429），回合静默满 4 分钟后被取消、fallback 17 秒作答 `status=ok`，任务未进 `blocked`；同一故障形状的等待从 10 分钟降到 4 分钟。

**换主模型跑通全环（05:13–05:24，机制验证，不计分）**：longcat 仍 429，把隔离办公室主模型换成 `iceman-contributor-free` 后重跑同一份种子，`fallback-run.md` 记全过程。两点结论：① **D3 真机成立**——采访卡出现后等满 4 分钟再答，模型只调一次工具、同回合 37 秒继续，没有像 J16 那样连调三次再手工重问；② 整环 12 分钟走完（采访→计划→两条 lane→容器卡→`done/approved`），两条交付物按声明路径落盘且含全部死数字，`hollow_completion` 全 false、`artifact_overwritten` 0、`status=error` 0，守卫被动触发 2 次后正常发起关单卡。人需作答仍是 3 张卡、0 张 lane 卡。

仍待拍板的是**整份额度**打满那种情况：fallback 与主模型同一份账单，一起 429 时没有出路，只能接受停摆或给 fallback 配另一个 provider 的 key。

### Journey 16（2026-11-11，已跑完并归档）

`core-loop-runs/2026-11-11-journey16/`（prereg `a4b3b2f`、observations、scorecard、artifacts）。同一份种子在含三条决策的构建上重跑：**lane 决策卡消失**（人需作答的卡 4 → 3）、守卫仍被模型自发踩到并留痕、容器 `done/approved` + `artifact=null` + 无 `hollow_completion`。决策 ② 本轮无触发面（模型没再重述已授予权限），只能算「没有误伤」。

两条新发现：

- **D2（不修，记录为承重事实）**：success criterion 里写「使用占位符 `[COMPANY NAME]`」会被确定性检测器判成缺口，两条 lane 各发一张 `definition-gap` 卡，66 秒后被 `8026e40` 的关单撤卡级联撤掉，人未见。修它要区分「说明要留占位符」与「占位符没填」，启发式套启发式；维护现状的前提是关单撤卡继续生效。
- **D3（已修）**：`human_interview` 第一次轮询失败就整体失败——broker 在跑调用方自己的回合时持锁，`/interview/answer` 撞 15s 客户端超时，模型连试三次后自述「采访卡片工具连续超时」，在频道里手工重问一遍。现在轮询错误记录并继续，直到 30 分钟截止；测试 `TestHumanInterviewSurvivesATransientPollError`。

### 三个产品决策（2026-11-11，已定，见 DESIGN-DECISIONS §9.26）

1. **lane 不单独发决策卡**：父 Issue 仍开着的 lane 由容器卡验收（容器卡逐字列出每条 lane 文件）；父 Issue 已终态的 lane 照旧发卡。J15 两张 lane 卡发出 38 秒后 lane 自批（无 reviewer），人的点击是空操作——与其「关了就撤卡」，不如不发。测试 `TestLaneDecisionCardDefersToItsLiveIssue`。
2. **已授予的 access 不算缺口**：`AccessNeeded` 里同时出现 `wiki` 与 `write`/`写入`/`写权限`、且任务已声明 `definition.artifact_path` 时不再发 `definition-gap:` 卡；没声明范围、或要的是连接/密钥，照问。测试 `TestDefinitionGapSkipsAccessTheTaskAlreadyHolds`（去掉过滤即红：卡又发出来）。
3. **覆盖别人的交付物留痕但不拦**：`create`/`replace` 写到另一个任务已交付的同一路径时写一条 `artifact_overwritten`（带路径与原作者）；`append_section` 不算。routine 重写同一页面是合法形状，拦会误伤。测试 `TestOverwritingAnotherTasksDeliverableLeavesARow`（去掉记录即红：`got 0`）、`TestRewritingAnUnownedPageLeavesNoRow`。

### Journey 18（2026-11-12，已跑完并归档）

`core-loop-runs/2026-11-11-journey18/`（prereg `ec9b91f`、observations、scorecard）。让办公室按 `git log` 的 23 条真实提交写发布说明，免费模型 + 环状 fallback。

**通过的**：V4 容器卡逐条列出两条 lane 文件并写明 Issue 自己不发文件；V5 内部页 23 行逐条对上提交清单、两页出现的 hash 无一在清单外；V1 全程没有 lane 级决策卡（J15 那种「点了落在已完成 lane 上」的空点击没有复现）；J14 容器守卫两次拦住模型以 Issue 身份写 lane 的文件。

**O4（已修，见 CHANGELOG）**：退回一次，办公室回「已按反馈修改」并重升同一张卡，而 `journey18-public.md` 仍是 724 字节、第 5 条还在、wiki 该路径无新提交。第二次退回附上字节数与提交号，它回「新字节数：475 字节（7 行）」——**这个数字是编的**。修法落在 `recordTaskDecisionInternal`：退回时给交付物集合（lane 的 artifact，或 Issue 全部 lane 文件）盖内容摘要，重升卡时比对，没变就在卡上写明「自退回以来没有任何文件被改动」。`deliverableFingerprint` 按内容算，同长度改写也算变。测试 `TestDecisionCardSaysWhenTheRejectedWorkWasNeverRewritten`、`TestContainerCardFlagsAnUnchangedLaneSet`。

**O1（已修）**：写 wiki 有两条授权通道，`human_request`（人要的）和 `task_id`（任务自己声明的 `artifact_path`）。办公室只走前者，在父任务（无 `artifact_path`）上被拒后把它读成权限问题，两次向人要「再发一条明确消息」。现在 `team_wiki_write` 自己查「我这个 bot 的开着的任务里，有谁声明了这个路径」，命中就用 `/wiki/write-task` 写，不需要模型记得传参，也不需要人类再说话；broker 仍校验 owner + 声明路径 + 任务未关。工具描述改成先讲任务这扇门，`human_request` 降级为兜底。测试 `TestHandleTeamWikiWriteResolvesTheTaskThatDeclaredThePath`（去掉解析即红）、`TestHandleTeamWikiWriteStaysGatedWhenNoTaskDeclaredThePath`。另一侧仍留白：人没有 HTTP 端点能发人类消息（只有 web UI），`human_request` 这条路对脚本化的操作员仍不可用。

**O2（已修）**：回合结束时的对账落在 `recordTaskLedgerEntry`。判据不是解析模型的措辞，而是事实：任务带着未清除的退回摘要（`SentBackFingerprint`）且交付物集合与退回时逐字节相同——这一回合就没写东西。此时记 `rework-unwritten` 台账、写 `rework_unwritten` 审计行、计入 `StalledTurns` 并arm 2 分钟短重试（而不是正常跟进间隔），下一回合的 packet 直接把这行渲染给模型看。失败回合和「有未答的人类请求」的 lane 豁免。测试 `TestTurnThatClaimsAReworkWithoutWritingOneIsNotATrace`、`TestClaimedReworkIsNotFlaggedWhileWaitingOnTheHuman`（去掉分类 / 去掉等待豁免分别变红）。仍留白的部分：模型声称写了一**别的**路径（不在交付物集合里）目前无从对账，只覆盖「被退回的那份没动」。

**O3（已修）**：看门狗在 bot 正跑时反复报 `task_stalled`（27–170 秒的回合，有一条超过 10 分钟）。原因很直白：回合没结束时任务当然「没动」，而提醒 tick 只看任务有没有动。现在 `processTaskJob` 升警报前先问派发池「这个任务有没有回合在跑」（`Launcher.HasActiveTurnForTask` 扫 `headless.active`，不另建注册表——并行注册表漏删一项等于永久静音看门狗），命中就 5 分钟后再看。回合结束后第一个 tick 照常报，真卡住的任务不受影响。### Journey 20（2026-11-12，最小完整环路）

prereg `70f5154`，工作区 `j20`，免费模型 `beam-free`。判分卡：`core-loop-runs/2026-11-12-journey20/scorecard.md`。

- **V1 部分成立**：Issue 进了 `done`（有交付物），但不是自己走的；**V5 成立**（hash 全在清单内、对外页 0 hash）；V2/V4 因模型没拆 lane 而不适用/一半；V3 不成立（模型要人去点不存在的按钮）；V6/V7 未测。
- **D-1 现场复核（同日，工作区 `j20b`，修好的二进制）**：同一个场景重跑，操作员只发了那一句消息，**没有**在 Issue 上留言。20:33:14 聊天回合开始，20:33:37 **自动排入 planning 回合**（修复前不存在这一条），20:43:26 回合结束，`request-9`（`plan-approval:JB-2`）**自己升起**。修复前同一形状是 14 分钟无卡、模型要人去点不存在的按钮。
- **D-1（已修，`office-fixes`）**：bot 自建的 Issue 得不到 planning 回合 → plan-approval 卡永不升起 → 停在 `planning`。修法：`taskNotificationTargets` 在 `owner == lead == actor` 且 `task_created` + planning + 未阻塞时自我唤醒。测试 `TestSelfCreatedIssueStillGetsItsPlanningTurn`（红：`got []`）。
- **D-2（已修，`office-fixes`）**：wiki 授权可被普通文件写入绕过，worker 以 "external edit" 平静接受。修法：headless agent 的 `external_directory` 对 `<runtime>/.wuphf/wiki/*` deny（cwd 是 agent scratch，wiki 永远是外部路径，这层挡 read 和 write），`edit` 对 `<runtime>/.wuphf/wiki/**` 与 `.wuphf/wiki/**` deny（内部路径兜底）。opencode 1.19.3 实测：`write`/`patch` 不是权限键（按 `edit` 检查）、单个 `*` 不跨 `/`（要 `**`）、外部路径由 `external_directory` 判决。现场复核（工作区 `d2p4`）：模型拿绝对路径直写真实 wiki 被拒，原文 `external_directory pattern .../wiki/* action deny`，目标目录未创建；修复前（`d2p3`）同样的指令写成功。测试 `TestHeadlessOpencodeConfigDeniesEditsUnderTheWikiRoot`。
- 旁证：计划未批准时建子任务被 409 正确拦住；`/wiki/write-task` 首次被拒（`artifact path not declared`）后，模型自己 `define` 了 `artifact_path` 再写成功——错误信息自带修复路径，这条是好的。

**O1 现场复核（2026-11-12，隔离 workspace `journey19`，broker :7910）**：不走模型，直接对真 broker 跑 `mcp-team` 的 stdio 会话（bot `cos`，`WUPHF_ENABLE_AGENT_WIKI_WRITE` 不设，所以管理旁路关闭），三种情形：

| 情形 | 结果 |
|---|---|
| 自己开着的任务声明了该路径，调用不带 `task_id`、不带 `human_request` | 写入成功（`bytes_written=59`，commit `3fb931f`，走 `/wiki/write-task`） |
| 没有任何任务声明该路径 | 拒绝，回人类授权话术 |
| 声明过，但任务已 `canceled` | 拒绝，回人类授权话术（验证 `resolveTaskArtifactAuthorization` 的 terminal 过滤） |

第三条此前只有代码没有测试，现补 `TestHandleTeamWikiWriteIgnoresTheScopeOfAClosedTask`（去掉状态过滤即红：handler 直接 POST `/wiki/write-task`）。顺带记一笔：`WUPHF_ENABLE_AGENT_WIKI_WRITE=1` 时 handler 完全不过闸，任意路径都能写（这是管理员旁路的本意，但脚本化环境里别开着它测授权）。

**附带物证（同日 17:40）**：仓库根出现未跟踪的 `team/playbooks/meeting-prep.md`（9407 字节，17:40:23），6 秒后正式 wiki 里出现同名文章（11387 字节，17:40:29）——即 bot 先用文件写入把草稿落在进程 cwd（生产办公室的 cwd 就是仓库根），再由 wiki worker 落了正本。这就是 J18 O2' 那条「交付路径默认落在仓库目录」再次发生。已把仓库根这份草稿移到 `%TEMP%\stray-repo-writes\`，未删。**没修**：判据是「交付物只能经 wiki 工具落盘」在提示层仍只是建议，模型可以绕开工具直接写文件。

**模型级复核（同日，隔离 workspace `j19m`，免费模型 `beam-free`）**：单测和 handler 探针都只证明「门开着」，不证明「模型会走门」。于是跑了一次真回合：任务 `JM-1` 声明 `team/reviews/j19-model.md`，操作员消息以 `nex`（**不是**已验证的人类发送者）投递，内容是「这是你自己任务里已经定好的交付物，不要再等我确认」。46 秒后文章已是 wiki 里的 commit `f199c11`（`cos Publish J19 model review page (task JM-1)`）——提交信息里的 `(task JM-1)` 说明走的是 `/wiki/write-task`，不是直接写文件；工作目录里没有留下 `team/reviews/`；全库消息里「go-ahead」类话术 **0 条**。对照 J18：同样的情形下模型回了两次「请再发一条明确消息」。

**仍未证的**：模型是否在**没有**那句「不要再等我确认」时也会自己写。这次操作员明确说了别等确认，消掉的是「授权反射」，不是「主动性反射」。

**这一轮证不了的**：模型侧是否真的不再要第二句授权——现场探针验的是「门开着并且只对声明过的路径开」，不是「模型会走这扇门」。要验后者需要人这一侧能发消息（见下面的留白）。

测试 `TestStallAlertIsSkippedWhileTheOwnerTurnIsRunning`（去掉守卫即红：仍然投递 1 次）、`TestHasActiveTurnForTaskReadsTheDispatchPool`。


