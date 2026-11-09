# Journey 4 判定卡 — J3-1 / J3-2 端到端复核（2026-11-09）

> 本轮不打九步分，只判 prereg §4 的 V1–V5 与 §5 的 F-A..F-D。
> 场景起点 `JOURN-9` 创建 **03:21:55Z**；唯一人工干预 **03:25:21Z**（答卡）；停表 **03:48:15Z**（自 03:31:47Z 起 16.5 分钟无新消息/无 active 回合，命中 prereg §7.2）。
> 证据分级：A = 机器可验（接口/状态文件/git/调度器字段）；B = 文件在盘；C = 口述。

## 1. 判定摘要

| 项 | 结果 |
|---|---|
| **J3-1（opencode plan 卡）** | **通过**。计划帖之后 31 秒，`request-12` 自己出现（`dedupe_key=plan-approval:JOURN-9`），无人工动作；全办公室只此一张 |
| **J3-2（等审阅不误报）** | **通过**。`JOURN-16` 于 03:30 进 `decision`，其 `recheck` job 在 03:31:11 真的跑了并收为 `done`；全程 `watchdog_alert` **0 条**、`watchdogs[]` 空、无重投 |
| **V3（凭空卡推进）** | **通过**。只答卡（`POST /interview/answer`，未走 `/tasks approve`）→ 同一秒 `lifecycle_state=running`，~1 分钟内拆出 `JOURN-16`/`JOURN-18` |
| **V5（对照：planning 仍会被看住）** | **未观察**（窗口内没有停在 planning 的任务；按 prereg 记「未观察」，不记通过） |
| 硬失败 | F-A/F-B/F-C/F-D **均未触发** |
| 人工干预 | **1 次**，即 prereg 允许的那次（且点在**卡片**上，不是任务端点） |

## 2. V1–V5 逐条

| 编号 | 判定 | 证据 |
|---|---|---|
| V1 「10 分钟内出现 plan-approval 卡」 | **命中** | A：`snapshots/0327/broker-state.json` 的 `request-12`：`kind=approval`、`status=answered`、`channel=cos__human`、`from=cos`、`issue_id=JOURN-9`、`dedupe_key=plan-approval:JOURN-9`、`created_at=2026-11-09T03:22:26Z`（任务 `created_at=03:21:55Z`）。`bg_137` 轮询在 **03:23:18Z** 首次看到它（`/requests?scope=office` 仍为空，卡在 DM 域——与 J3 同）；`question` 里带 cos 的计划正文，末句「本轮计划模式：未建子任务、未写文件，待批准后执行。」 |
| V2 「只有一张（幂等）」 | **命中** | A：`snapshots/0347/broker-state.json` 的 `requests` 长度 1，唯一元素 `request-12`（`status=answered`）；`dedupe_key` 机制即幂等键。`messages.json` 里 `human_request_raised=1` |
| V3 「只点卡即可推进并拆活」 | **命中** | A：03:25:21Z `POST /interview/answer {"id":"request-12","choice_id":"approve"}` → `{"ok":true}`；同秒 `GET /tasks/JOURN-9` → `lifecycle_state=running`；`snapshots/0327/tasks.json` 出现 `JOURN-16`（`analyst__cos`）、`JOURN-18`（`ae__cos`），`parent_issue_id=JOURN-9` |
| V4 「进 decision 后不再有 refused-wake 告警」 | **命中** | A：`recheck:analyst--cos:task:journ-16` 的 `last_run=2026-11-09T03:31:11Z`、`status=done`、`next_run=null`（`snapshots/0347/scheduler.json`）——正是 J3 里每 tick 误报的那条路径；而 `broker-state.json` 的 `watchdogs` 为 `[]`、`messages` 中 `watchdog_alert` 计数 **0**；`j4-mon.log` 自 03:28:05 到 03:48:06 每 2 分钟采样 `watchdog_refused_alerts=0` 共 11 次 |
| V5 「对照组：planning 仍受监视」 | **未观察** | 窗口内没有停在 `planning` 等卡的任务（唯一 planning 任务在 03:25:21 就被答卡推进）。按 prereg §4 V5 的允许写法记为「未观察，不许默认通过」。对照改由单测 `TestSchedulerStillWatchesPlanningTasks` 与 `TestSchedulerStillAlarmsParkedTask` 承担（含 failing-before） |

## 3. 硬失败

| 编号 | 判定 | 说明 |
|---|---|---|
| F-A（10 分钟无卡） | 未触发 | 31 秒出卡 |
| F-B（≥2 张卡） | 未触发 | `requests` 恒为 1 |
| F-C（decision 后仍报 refused wake） | 未触发 | 0 条 `watchdog_alert`；`watchdogs[]` 空 |
| F-D（答卡不推进） | 未触发 | 同秒 `running`，1 分钟内拆两条 lane |

## 4. 本轮顺带看到的（不判分，进 observations/TODO）

1. **J3-3 复现**：`JOURN-16`/`JOURN-18` 创建时都没有 `definition`（`snapshots/0347/task-JOURN-16.json` 的 `definition=null`），两个 bot 都撞上 `wiki 直写 403`，全部改走「先落 notebook、再上报 @cos 补 artifact_path」。`JOURN-16` 最终以 `artifact=team/research/icp-hypothesis-2026q4.md` 进 `decision`（**definition 仍为 null**），`JOURN-18` 停在 `running` 无产物。与 J3 完全同型 → J3-3 是稳定复现，不是偶发。
2. **`recheck` 只给部分 lane 装**：`JOURN-16` 有 `recheck`（03:31:11 跑过），`JOURN-18` 只有 `task_follow_up`（04:28:36）。同为 `running` 的两条 lane，重投覆盖不一致——J3 评分卡 §4.7 那条观察在 J4 再次出现，可以升格为 TODO（J3-5）。
3. **`task_stalled` 与告警分离**：`actions` 里有 2 条 `task_stalled`（`JOURN-9` 10m、`JOURN-18` 11m），但**没有**对应 `watchdog_alert` 消息，也没有 `watchdogs[]` 条目。J3-2 的改动只关了 review 态；`running` 态仍然「记账不报警」——是设计如此还是缺口，需要下一步判断。

## 5. 证据索引

1. `snapshots/0323/health.json` — provider/model/backend 锚定
2. `snapshots/0323/tasks.json` — 4 条 starter lane 已 `canceled`
3. `snapshots/0327/tasks.json` — `JOURN-9`（`running`）、`JOURN-16`、`JOURN-18`
4. `snapshots/0327/broker-state.json` — `request-12` 全字段（`dedupe_key`/`issue_id`/`reply_to=msg-13`/`created_at=03:22:26Z`）
5. `snapshots/0335/…`、`snapshots/0347/…` — 中途与终态
6. `snapshots/0347/scheduler.json` — `recheck:…journ-16 status=done last_run=03:31:11Z`
7. `snapshots/0347/log-headless-codex-cos.log` — 计划正文与「待批准后执行」原句
8. `snapshots/0347/log-headless-codex-analyst.log` — `wiki 直写被卡…JOURN-16 已核验在审（ready_for_review）`
9. `snapshots/0347/log-headless-codex-ae.log` — 「落盘被权限挡住了…先存到笔记本」
10. `snapshots/0347/log-broker.log` — 调度与 403 行
11. `j4-mon.log`（临时，不随仓库）— 11 次采样 `watchdog_refused_alerts=0`，最后一条 03:48:06Z
12. `observations.md` — 时间线与机制侧分列
