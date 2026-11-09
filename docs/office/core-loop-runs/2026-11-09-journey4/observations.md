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
| 03:31:11 | `recheck:analyst--cos:task:journ-16` 触发（`last_run`），随后 `status=done`、`next_run=null`（`snapshots/0347/scheduler.json`）。这正是 J3 里每 tick 误报的那条路径 |
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
2. **recheck 覆盖不一致**：同为 `running`，`JOURN-16` 有 `recheck`，`JOURN-18` 只有 60 分钟的 `task_follow_up`。
3. **`task_stalled` 只记账不报警**：`running` 态任务超时只写 `actions`，不写 `watchdog_alert`、不进 `watchdogs[]`。J3-2 的改动只覆盖 review 态，这条留给下一步判断。
4. **`/requests?scope=office` 依然看不到 DM 域的卡**（`request-12` 在 `cos__human`）。本轮靠 `broker-state.json` 才拿到全量；UI 侧是否有入口未验证。
