# Journey 4 观察记录（2026-11-09）

> 目的见 `prereg.md`：只验 J3-1（opencode planning 出卡）与 J3-2（等审阅不误报）。时间均为 UTC（北京 +8）。

## 1. 环境（实测）

| 项 | 值 | 证据 |
|---|---|---|
| 二进制 | `gawkbot-fix.exe`，10:58 本地构建（HEAD `364511c`，含 `0a91081`+`d01b103`） | `snapshots/0323/health.json` |
| 隔离办公室 | `journey4`，broker `:7910`、web `:7911` | `workspace create --from-scratch journey4` → `Ready (broker :7910, web :7911)` |
| provider / model | `opencode` / `opencode-go/crouton-alpha-free` | `health.json: provider_model` |
| memory backend | `markdown`（`memory_backend_ready=true`、`wiki_ready=true`） | `health.json` |
| 蓝图 / 成员 | `ai-revops`：cos / revops / analyst / ae / reviewer | `onboarding/complete` 返回 `{"ok":true}` |
| 主办公室 | pid 3044（7891/7892），本轮未动 | 部署前 `/health` 与本轮分离 |

## 2. 安装（不计分）

- 03:2x `POST /onboarding/complete` → `{"ok":true,"redirect":"/"}`。
- 取消 4 条 starter lane：`ai-revops-1..4`，`POST /tasks/{id}/cancel` body `{"reason":"install cleanup: onboarding starter lane"}`，四条全 `changed:true`；`snapshots/0323/tasks.json` 里它们已是 `canceled/canceled`。

## 3. 时间线

| 时间 | 事实 |
|---|---|
| 03:21:55 | 提交人给的 Issue（prereg §3 原文逐字）：`POST /tasks {"action":"create","channel":"cos__human","owner":"cos","created_by":"human","task_type":"issue"}` → **`JOURN-9`**，`status=in_progress`、`lifecycle_state=planning`、`follow_up_at=04:21:55Z`、`recheck_at=04:36:55Z` |
| 03:22:26 | 出现 **`request-12`**：`kind=approval`、`channel=cos__human`、`from=cos`、`dedupe_key=plan-approval:JOURN-9`、`issue_id=JOURN-9`、`recommended_id=approve`、`due_at=03:52:26Z`；`question` 里带 cos 的计划正文（「## 计划：JOURN-9 一页式验证包…」）。**距任务创建 31 秒，无人工动作** |
| ~03:24 | `log-headless-codex-cos.log` 出现计划正文，末句「本轮计划模式：未建子任务、未写文件，待批准后执行。」 |
| 03:25:21 | **唯一一次人工干预**（prereg §6 允许的那次）：`POST /interview/answer {"id":"request-12","choice_id":"approve"}` → `{"ok":true}`；同一秒 `GET /tasks/JOURN-9` → `lifecycle_state=running`、`status=in_progress` |
| 03:26–03:27 | 拆出两条子 Issue：**`JOURN-16`**、**`JOURN-18`**（均为 `in_progress`/`running`），`snapshots/0327/tasks.json` |
| 03:28:05 | 监控首次采样：三条 JOURN 全 `running`，`watchdog_refused_alerts=0` |

## 4. 机制侧（与观察分列）

- 卡片由 `headless_opencode.go` 的 planning 分支在回合末尾 raise：本轮的 `dedupe_key=plan-approval:JOURN-9` 与 `broker_plan_approval` 的幂等键一致，说明走的是 `RaisePlanApproval`，不是任何兜底消息。
- 答卡走 `POST /interview/answer` → `answerRequestFromActor` → `applyPlanApprovalAnswerLocked` → `Planning→Running`；**未**调用 `POST /tasks approve`（prereg 禁止的替代路径）。
- `watchdog_refused_alerts` 采样口径：`GET /messages?limit=400` 中 `kind=watchdog_alert` 且 `content` 含 `No turn could be dispatched` 的条数。

（后续段落随观察补充：decision 态与告警对照、V4/V5 判定、停表时点。）

## 5. 03:30Z – 03:48Z：decision 态与告警对照

| 时间 | 事实 |
|---|---|
| 03:30:05 | `JOURN-16` 进 `lifecycle_state=decision`，`artifact=team/research/icp-hypothesis-2026q4.md`（该任务 `definition` 仍为 `null` —— 见 §6） |
| 03:31:11 | `recheck:analyst--cos:task:journ-16` 变成 `status=done`、`next_run=null`、`last_run=03:31:11Z`（`snapshots/0347/scheduler.json`）。**收尾理由分辨不出**：新门（J3-2）与 `cancelSupersededTaskJobsLocked` 都写 `last_run` |
| 03:31:47 | cos 最后一条正文；此后无新消息 |
| 03:28:05 – 03:48:06 | 监控每 2 分钟采样 `watchdog_refused_alerts`，**11 次全为 0** |
| 03:48:15 | 停表：自 03:31:47 起 16.5 分钟无新消息、无 active 回合（prereg §7.2） |

终态（`snapshots/0347/`）：

- `JOURN-9` `in_progress/running`，`artifact=null`（父 Issue，等 lane；`task_stalled` 记账 10m，但无 `watchdog_alert` 消息）
- `JOURN-16` `in_progress/decision`，`artifact=team/research/icp-hypothesis-2026q4.md`，`definition=null`
- `JOURN-18` `in_progress/running`，`artifact=null`（邮件 lane 卡在 403，草稿在 notebook）
- `broker-state.json`：`requests` 只有 `request-12`（`answered`）；`watchdogs` 为 `[]`；`messages` 里 `kind=watchdog_alert` 计数 **0**；`actions` 末尾 2 条 `task_stalled`（`JOURN-9` 10m、`JOURN-18` 11m）无对应告警消息
- `scheduler.json`：`recheck:…journ-16` `done`（03:31:11）；三条 `task_follow_up` 分别 04:25/04:28/04:31；**`JOURN-18` 没有 recheck job**

## 6. 顺带观察（不判分）

1. **J3-3 同型复现**：两条子 Issue 创建时都无 `definition`，两个 bot 都撞 `wiki 直写 403`，都改走「先落 notebook，再请 @cos 补 `artifact_path`」。`JOURN-16` 后来带 artifact 进 `decision`（definition 仍空），`JOURN-18` 停在 `running` 无产物。→ J3-3 是稳定复现。
2. **recheck 有无不对称，但起源与收尾都不可从状态分辨**：同为 `in_progress`，`JOURN-16` 有一条 `recheck` job（`last_run=03:31:11`、`done`），`JOURN-18` 只有 `task_follow_up`。`scheduleTaskLifecycleLocked` 的 `in_progress` 分支只装 follow-up，`recheck` 只在非 `in_progress` 的分支装；`ScheduleRecheck` 在生产代码里没有调用点。所以这条 job 的来路（哪次状态切换装的）与收尾理由都查不到——**这是可观测性缺口，不是已证实的漏 wake**。
3. **`task_stalled` 是另一条可见线，不是沉默**：03:36:43 `cos__human` 出「@cos has gone quiet on “ICP 假设与首触点邮件：一页式验证包” for about 10 minutes…」，03:40:43 `ae__cos` 出同型一条（`broker-state.json` 的 `messages`，`kind=task_stalled`）。它和 `watchdog_alert` 是两条独立通道：前者由 stall 检测器按任务静默时长发一次，后者是 watchdog 的「唤醒被拒」。J3/J4 的「只记账」说法是我读错 kind——账本（`actions`）里确实有，消息里**也有**。
4. **`/requests?scope=office` 依然看不到 DM 域的卡**（`request-12` 在 `cos__human`）。本轮靠 `broker-state.json` 才拿到全量；UI 侧是否有入口未验证。

## 7. 记录缺口（自我记账）

- `JOURN-16` 落盘的 `team/research/icp-hypothesis-2026q4.md` **没有在 shred 前复制进仓库**：证据只到 `snapshots/0347/wiki-tree.txt`、`task-JOURN-16.json` 的 `artifact` 字段与 wiki git log（B 级：文件曾经在盘），正文内容不可复核。J3 的流程（shred 前 `cp` 产物）本轮漏了一步；下次跑完先归档产物再 shred。
- 本轮 `workspace shred --yes journey4` 已执行，7910/7911 释放；证据目录保留 `prereg.md` / `observations.md` / `scorecard.md` / `snapshots/0323|0327|0335|0347|0348`。

## 8. 修复落地（同日，J3-3 与 J3-5 的可观测性部分）

| 缺陷 | 修法 | 现场验证（隔离办公室 `j5smoke2`，7910） |
|---|---|---|
| J3-3 拒写只说给 bot 听 | `wiki_handlers.go`：拒写文案点名 `created_by`、指到 `definition.artifact_path`、明说重试无效；首次拒绝往任务频道投一条 `kind=artifact_path_missing` 的 system 消息（@creator + @owner），按任务去重 | 对无 `definition` 的任务连打 3 次 `/wiki/write-task` → 3 次 403（同一句文案），`analyst__cos` 里只有 **1 条** nudge：「@cos: JSMOK-28 has no definition.artifact_path, so @analyst's write to team/research/smoke.md was refused… Owner: @analyst.」 |
| J3-5 关掉的 job 说不清为什么 | `schedulerJob.StatusReason`（wire `status_reason`）：审阅门写 `awaiting review (decision)`、被顶掉写 `superseded by task_follow_up`、连带关闭写 `target closed`；重新排期即清空 | `POST /tasks/JSMOK-2/cancel` 后 `/scheduler` 里 `task-follow-up:analyst--cos:jsmok-2` → `status=done`、`status_reason=target closed`、`next_run=null`，而活着的行 reason 为空 |

两处都有 failing-before 单测（去掉去重/去掉点名/去掉 reason 即红）。全量 `go test ./internal/team/ -timeout 30m` 一轮里 `TestObsidianWatcher_StopDuringPendingDebounce` 因 Windows 临时目录句柄报 cleanup 错，单独跑通过——并行执行下的文件锁抖动，与本轮改动无关。
