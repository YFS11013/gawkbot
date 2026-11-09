# Journey 3 — 原始观察

时区：UTC（括号内为北京 UTC+8）。只记事实与时间戳，结论在 `scorecard.md`。

## 环境（提交任务前）

| 时间（UTC） | 事实 |
|---|---|
| 00:58 | `workspace shred --yes journey2` → 释放 7910/7911；`workspace create --from-scratch journey3`（进程带 `WUPHF_OPENCODE_MODEL=opencode-go/crouton-alpha-free`）→ `Ready (broker :7910, web :7911)` |
| 00:58 | `<space>/.wuphf/config.json` = `{"llm_provider":"opencode","llm_provider_priority":["opencode"],"memory_backend":"markdown"}`（`seedWorkspaceConfig` 从主办公室种入，§9.11 生效：配置在 broker 启动前已就位） |
| 00:59 | `/health`：`provider=opencode, provider_model=opencode-go/crouton-alpha-free, memory_backend_active=markdown, memory_backend_ready=true, wiki_ready=true, dispatch.members=1` |
| 00:59 | broker.log：`WARNING wiki: gbrain unavailable (initialize: EOF); falling back to the in-memory index`（本机无 gbrain，与 Journey 1/2 相同，不影响 markdown wiki 读写） |
| 01:00:14–15 | `POST /onboarding/complete`（`skip_task=true, blueprint=ai-revops, agents=[cos,revops,analyst,ae,reviewer]`）→ `{"ok":true}`；broker.log `onboarding: wiki materialized blueprint=ai-revops dirs=5 articles_created=5`、`wiki bootstrap committed d8fb3cf`、`getting-started pages committed 4d88b40` |
| 01:00:51 | onboarding 立即把蓝图 starter lanes 装成任务：`ai-revops-1`(hygiene/revops)、`ai-revops-2`(pipeline/analyst)、`ai-revops-3`(outreach/ae)、`ai-revops-4`(review/reviewer)、`task-skill-6`(pipeline/cos)，全部 `in_progress` |
| 01:00:51 | `GET /channels` → `{"channels": []}`；`internal/channel/general.go:179` `namedChannelsEnabled = false`（命名频道在本构建里是硬关闭，只有测试能翻） |
| 01:00:07 | cos 在 `cos__human` 自我介绍（onboarding kickoff） |

未产生判定流量前的安装动作，如实登记：`onboarding/complete`（API）、取消 5 条 starter lane（`POST /tasks/{id}/cancel`，reason=`install cleanup: onboarding starter lane`，返回 `changed=true`）。

## 场景时间线

任务：`JOURN-18`「SMB renewal-retention: go/no-go package」，`created_by=human`、`owner=cos`、`channel=cos__human`、`task_type=issue`。

| 时间（UTC） | 事实（原文摘录，不加工） |
|---|---|
| 01:02:49 | `POST /tasks` 建单成功；`status=in_progress`、`follow_up_at=02:02:49`、`recheck_at=02:17:49`、`due_at=02:02:49`；`definition` 字段此刻不存在（detail 里无 definition） |
| 01:03:47 | cos：「办公室频道暂不可达，改用文字拆解说明并等待下一通知。」 |
| 01:04:38 | cos 贴出为 `#ai-revops-2` 拆的 4 个子任务（`2.1 @revops … 2.2 @analyst …`），与 JOURN-18 无关 |
| 01:04:48 | cos：「外部 CRM 还没接上，连接列表为空，先把发现与三条清洁线拆成子任务派出去。」 |
| 01:04:50 | cos：「子任务拆分遇到通道路由问题，我换种方式直接推进序列草稿。」 |
| 01:05:05–01:05:30 | cos：「正在把计分卡拆成三条子任务，分配给 @analyst 与 @reviewer 与 @revops 并行定义。」「已把四个交付物和期限摸清，正在按计划模式核对现有上下文再写方案。」「上下文已核对，正在把计划落到 notebook 上。」 |
| 01:05:37 | cos 在 `cos__human` 贴出 **JOURN-18 计划**（中文，含四个交付物路径、验收标准、风险），标题自述「计划阶段，未开工，**供批准**」 |
| 01:05:38 | `GET /tasks/JOURN-18`：`updated_at=01:05:38Z`，`status=in_progress`；`actions` 0 条；无 definition、无 artifact、无子 Issue |
| 01:05:44 | cos：「当前环境里频道和任务接口暂时不可用，我换成直接通知 owner 起草。」 |
| 01:06:06–01:06:07 | `#ae__cos`、`#analyst__cos`、`#cos__revops` 出现 cos 派工消息，均以 `父任务 ai-revops-3` / `ai-revops-2` 指名 |
| 01:06:38 | ae 在 `#ae__cos` 交付 re-engagement 草稿，并注「任务评论写入失败，改走频道直发」 |
| 01:06:53 | analyst 在 `#analyst__cos` 给出可机读阈值，注「工具参数校验失败，改用正文内 @cos 提及重新发送」 |
| 01:07:07 | revops 在 `#cos__revops` 给出 BLOCKED 汇总（空数据源 + 无 CRM 只读权限） |
| 01:07:23–29 | cos 回复 revops 闭环，称「后续由我向人类确认 CRM 数据源」；`GET /requests?scope=office` 此刻无任何 pending（没有生成审批卡/阻塞卡） |
| 01:07:40± | 记录者取消 5 条 starter lane（安装清理，见上） |
| 01:07:29 之后 | `headless-codex-cos.log` 无新行；`JOURN-18.updated_at` 停在 01:05:38Z |
| 01:09:18 | broker.log `scheduler: heartbeat (uptime=10m0s, poll=20s)` |
| 01:09 | `/scheduler`：`task-follow-up:cos--human:journ-18 next_run=2026-11-09T02:02:49Z`；另有 `task-retry:hygiene:task:ai-revops-1`、`task-retry:outreach:task:ai-revops-3`、`task-retry:review:task:ai-revops-4` 三条 `status=done`（J2 的短重投在这些 lane 上确实被排并已执行） |
| 01:06:29–30 | `headless-codex-cos.log` 连续三条 `error: headless codex turn cancelled so newer queued work can run` |

## cos 回合日志摘录（`logs/headless-codex-cos.log`）

```
[09:05:44] live-chat-relay-post: posted streamed output to #cos__human as msg-38
[09:05:58] queue-drop: lead already handling same task
[09:06:22] queue-park: at concurrency cap, deferring lane until a slot frees
[09:06:23] live-chat-relay-post: posted streamed output to #cos__human as msg-50
[09:06:25] queue-replace: refreshed pending lead turn for same task
[09:06:29] error: headless codex turn cancelled so newer queued work can run   (×3)
[09:06:38..09:06:57] queue-drop: lead already handling same task               (×7)
[09:07:00] opencode_result: 已确认卫生扫描与 CRM 数据源缺失…
[09:07:13] wake-lead: re-delivering specialist completion from @revops (msg msg-73)
```

## 空转窗口（01:08Z – 02:02Z）

| 时间（UTC） | 事实 |
|---|---|
| 01:17 | `GET /tasks/JOURN-18`：`status=in_progress`, `updated_at=01:05:38Z`, `actions=0`；`cos__human` 最后一条正文停在 01:06:23Z |
| 01:17 | `GET /requests?channel=cos__human|cos__revops|ae__cos|analyst__cos` 全部 `requests: []`；`scope=office` 同样为空 —— 办公室自称「向人类确认 CRM 数据源」，但没有任何 pending 卡 |
| 01:17 | `GET /channels` → `{"channels": []}`。蓝图 starter lanes 的频道（hygiene/pipeline/outreach/review）在本构建里不存在：`internal/channel/general.go:179` `namedChannelsEnabled = false`（生产代码没有开关，只有 `SetNamedChannelsEnabledForTest`） |
| 01:06:29 | 取消 4 条 starter lane 的同一秒，`headless-codex-cos.log` 出现 3 条 `error: headless codex turn cancelled so newer queued work can run` —— 与 §9.9 的 `CancelHeadlessTurnsForTask` 行为一致（取消会停回合）；此后没有任何 `ai-revops-*` 的回合 |
| 01:00:51 起 | `task_retry` 三条（`ai-revops-1/3/4`）在 `/scheduler` 里 `status=done`，说明 §9.8 的 `no durable trace` 短重投在这些 lane 上被排过并执行过 |
| 01:09 | `task-follow-up:cos--human:journ-18 next_run=02:02:49Z`；无 `task_retry` 指向 `journ-18`（cos 的回合有正文投递，不属于 `no durable trace`） |

## 机制侧证据（读代码，不是推断情绪）

办公室把这条人给的 Issue 放进了结构化规划态，规划态的「交给人批准」那一步只接在两个 runtime 上：

| 事实 | 位置 |
|---|---|
| 顶层 Issue 默认先规划（`issueShouldPlanFirstLocked` 对非子任务、非内部 actor、非 app-builder 一律 true） | `internal/team/broker_plan_approval.go:39`；创建分支 `broker_tasks_mutation_service.go:839` |
| 规划态的子任务创建被拒：「parent %s is still in planning — its plan has not been approved yet」 | `internal/team/broker_tasks_subissue_guards.go:61-69` |
| 规划态完成/提交被拒：「its plan must be approved before work can be …」 | `broker_tasks_mutation_service.go:1002-1008` |
| 工作包前置 `[PLAN MODE]` 指令，告诉 bot「计划会自动呈现给人批准」 | `internal/team/plan_mode.go:68`，经 `notifier_delivery.go:210` 注入 |
| 真正的「呈现给人」只有两个调用点：Claude runner 与 Codex runner | `headless_claude.go:344`、`headless_codex_runner.go:391` → `headless_runtime.go:88` `RaisePlanApproval` |
| **opencode runner 没有任何 plan 相关调用**（`grep -c LifecycleStatePlanning\|raisePlanApproval internal/team/headless_opencode.go` = 0，文件 434 行） | `internal/team/headless_opencode.go` |
| 人类批准规划的唯一入口是 `action=approve`（Drafting/Planning→Running，仅人类或内部 actor） | `broker_tasks_mutation_service.go:1019-1030` |

本轮 provider 就是 opencode（`/health` 的 `provider=opencode`），于是：规划态 → 没有审批卡 → 子任务被拒 → 完成被拒 → 下一次唤醒是 60 分钟后的 `task_follow_up`。cos 在 01:05:37 贴出的计划就是按 `[PLAN MODE]` 指令写的（「计划阶段，未开工，供批准」），它不是没照做，是照做了但「自动呈现」这一环在 opencode 上不存在。

## 停表前的一次人工干预（prereg §「允许的人工干预」唯一例外）

| 时间（UTC） | 事实 |
|---|---|
| 01:42:27 | 记录者对 `JOURN-18` 执行一次 `POST /tasks {"action":"approve","id":"JOURN-18","channel":"cos__human"}`（人类批准规划）。响应后 `lifecycle_state=running`、`pipeline_stage=breakdown`、`task_follow_up` 重排到 `02:42:27Z` |
| 01:43:26 | cos：「五个子任务已全部派活」；01:43:31 「JOURN-18 已拆成 5 条并行子线开工：@analyst 做研究简报（JOURN-78）、@ae 做 playbook（JOURN-81）和邮件（JOURN-85）、我做单屏 go/no-go（JOURN-89）、@reviewer 守证据 QA（JOURN-95）」 |
| 01:43:31 起 | 子 Issue 落位：`JOURN-78 analyst__cos analyst`、`JOURN-81 ae__cos ae`、`JOURN-85 ae__cos ae`、`JOURN-89 cos__human cos`、`JOURN-95 cos__reviewer reviewer`，`parent_issue_id=JOURN-18` |
| 01:44 | `headless-codex-ae.log`：ae 先要求「待 @cos 补 artifact_path 定义后落盘」；随后 `JOURN-81` 的 `artifact=team/playbooks/smb-renewal-outreach.md`，wiki 出现 13 KB 文件（commit `060e338 wiki: external edit to …`） |
| 01:45:56 | ae：「JOURN-81 已交付并送审」 |
| 01:47:06–17 | cos：「给 ST-1 补 artifact_scope … 顺手把 ST-3 的定义补上。定义参数校验没过，改用最小字段重试，只设置 goal 和 artifact_scope」 |
| 01:47 | `JOURN-85` 的 `artifact=team/emails/smb-renewal-intro.md`，文件 733 B（commit `3e432f0`）；`team/research/` 目录仍不存在，`JOURN-78` 的简报此前只落在 `agents/analyst/notebook/2026-11-09-smb-retention-2026q4-简报.md`（commit `20111d2`） |
| 01:48:17 | cos：「ST-3…已落盘、未发送，当前状态为待 ST-5 质检…ST-1 仍在跑」 |
| 01:50 | `JOURN-81`/`JOURN-85` 均为 `in_progress` + `lifecycle_state=decision`（等决策包）；`JOURN-78/89/95` 仍 `running`；`/requests?scope=office` 仍为空 |

## 01:48Z – 02:06Z：安静下来，靠 watchdog 报警

| 时间（UTC） | 事实 |
|---|---|
| 01:48:17 | cos 最后一条正文；此后所有 bot 日志无新行（ae/analyst/cos/reviewer 各自停在 01:45–01:48） |
| 01:49 | `broker-state.json` 里 `request-129`：`@analyst wants to promote "ST-1 研究简报…" into the team wiki`，`channel=analyst__human`，`status=pending`（这是 ST-1 落 wiki 的唯一闸门；`/requests?scope=office` 与 `?channel=analyst__cos` 都看不到它，只有落盘状态里看得见） |
| 01:47:01 / 01:47:06 | `JOURN-78` 与 `JOURN-85` 出现 `definition.defined_at`，`artifact_path` 分别是 `team/research/smb-retention-2026q4.md` / `team/emails/smb-renewal-intro.md`；`JOURN-81`/`JOURN-89` 仍无 definition |
| 02:00:38 / 02:02:38 / 02:02:58 / 02:04:58 | 四次 `scheduler: dispatched 1 due job(s)` → `ae__cos` 出现 4 条 `watchdog_alert`：「Watchdog: @ae still needs to move ST-2/ST-3 … No turn could be dispatched to wake @ae — retrying shortly.」（J2-4 的可见化与 `task_retry` 生效：`task-retry:…journ-81/85` 均 `status=done`） |
| 02:02:58 起 | `JOURN-81`/`JOURN-85` 仍是 `in_progress + lifecycle_state=decision`；`artifact` 已填且文件在盘 |
| 02:05 | 17 分钟无新正文、无 active 回合 → 命中 prereg 停表条件（「连续 15 分钟无新消息且无 active 回合」），**停表 02:05Z**（本地 10:05） |
| 02:06 | wiki 新 commit `8fb236c librarian: compile skill smb-renewal-outreach` —— 第 8 步（playbook→skill 编译）在停表前自己发生 |

## 停表时的事实清单（供判分）

- 父 Issue `JOURN-18`：`in_progress`、`lifecycle_state=running`、`artifact` 空、无 `verification`。
- 子 Issue 5 条，全部 `in_progress`；`JOURN-81`/`JOURN-85` 为 `lifecycle_state=decision`（等审阅），`JOURN-78/89/95` 为 `running`。
- wiki 交付：`team/playbooks/smb-renewal-outreach.md`（13086 B）、`team/emails/smb-renewal-intro.md`（733 B）在盘；`team/research/` 不存在（简报只在 `agents/analyst/notebook/`，且 promotion 卡 pending）；`team/visual-artifacts/` 无 HTML（ST-4 未产出）。
- `team/skills/smb-renewal-outreach.md` 由 librarian 编译（commit `8fb236c`）。
- 未决人工卡只有 `request-129`（promotion，pending）；另有 `decision-1`（watchdog remind_owner，指向已取消的 `#outreach`）。
- 复核：`GET /tasks/JOURN-18` 的 `actions` 仍为 0 条（任务级动作列表未记录被 approve 的事实，`lifecycle_state` 却已 running）——两处状态面不一致，判分时按 wire 的 `lifecycle_state` 取「人类批准发生过」。

## 停表后的机制复核与修复（不改变上面的观察）

判分卡由独立判分者按上面的快照写成（`scorecard.md`，25/36 = 6/10）。停表后我复核了两处被观察到的机制，结论与修复：

1. **`watchdog_alert` 的文案在 `decision` 态上是错的，而且根因不只是文案。** 02:00–02:04 那四条告警指向的 `JOURN-81/85` 当时是 `lifecycle_state=decision`（`ready_for_review`）。唤醒被拒不是链路故障，而是 `notifier_delivery.go:183-186` 的可执行状态门：非 Running/Approved/Planning 的任务本来就不该被派回合。也就是说 watchdog 把「等审阅」判成了「owner 不动」，并且会按 recheck/reminder 的 cadence 一直重复。修复：`scheduler.go` 的 `processTaskJob` 在 done/blocked 判定后，对**等审阅**状态（`review`/`decision`/`changes_requested`/`rejected`）直接收尾 job（不派活、不写 `task_stalled`、不发 `watchdog_alert`）；状态变更后 `scheduleTaskLifecycleLocked` 会重新装回；`planning`、`drafting`、`intake` 仍受监视（那是有 owner 却没动，人该看见）。测试 `TestSchedulerQuietForTasksAwaitingReview`（去掉门槛即 `delivered 1` 失败）、`TestSchedulerStillAlarmsParkedTask`、`TestSchedulerStillWatchesPlanningTasks`。
2. **opencode 的 planning 缺卡。** `headless_opencode.go` 现在在回合成功且有正文时 `emitHeadlessPlan` + `raisePlanApprovalAfterTurn`。测试 `TestOpencodePlanningTurnRaisesPlanApproval`（去掉调用即 `raised 0 plan-approval interviews` 失败）、`TestOpencodeRunningTurnRaisesNoPlanApproval`。

两条修复都用单测证伪/证实，**没有重跑 Journey 3**：本轮证据保持不变，修复的端到端效果需要 Journey 4 验证。

### 判分卡 §4 的一条自我修正（2026-11-09，代码复核）

判分卡 §4.6 与 TODO 的 J3-4 都写了「`approve` 分支没 `RecordAction`」。**这条判错了**：task wire 根本没有 `actions` 字段（`snapshots/0206/task-JOURN-18.json` 的键只有 `ledger` 等 20 个），批准事实记在办公室全局动作账本里——`broker_tasks_mutation_service.go` 的 Planning/Drafting→Running 激活分支以 `appendActionLocked("task_updated", …, "… [approved]", task.ID)` 写入。结论改为：账本没缺，缺的是「任务页面内联全局账本」这个产品决定，不计为缺陷。J3-3 的判据也同步改成「定义来得晚 + 写门拒后只有 promotion 一条路」，写门本身保留。
