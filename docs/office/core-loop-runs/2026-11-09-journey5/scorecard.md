# Journey 5 — 评分卡（独立复核，2026-11-09）

> 依据 `prereg.md §4 V1–V6 / §5 F-A..F-D` 逐条判分。证据以快照文件+字段+时间戳为准；`observations.md` 仅作时间线参考，结论以快照为准。全部时间 UTC。

## 1 判定摘要

### V1–V6

| 编号 | 判定 | 结果 | 核心证据 |
|---|---|---|---|
| V1 | 计划卡自动出现 | **命中** | `0452/broker-state.json:requests[request-9]` `dedupe_key=plan-approval:JOURN-6` `status=pending` `created_at=2026-11-09T04:52:24Z`，计划帖 `msg-8` `04:52:23Z`，间隔 ~1s，≤10min |
| V2 | 拒写文案点名 creator | **命中（仅探针；循环内未观察）** | 探针 `0523/broker-state.json:messages[msg-64]` + `actions[action-55]` 印证 403 链路；`observations §3` 称两次 403 均含 `@cos`/`definition.artifact_path`/`cannot succeed`；快照内无 bot 触发的 403，循环内无此行为 |
| V3 | 提示只喊一次 | **命中** | `0523/messages` `kind=artifact_path_missing` 计数 1（`msg-64` `analyst__cos` `05:23:08Z`），两次调用仅 1 条；`0452/0454/0455/0518` 计数均为 0 |
| V4 | 提示落在对的地方 | **命中** | `0523/msg-64` `channel=analyst__cos`（= `JOURN-15` 任务频道），内容同时含 `@cos`（creator）与 `@analyst`（owner） |
| V5 | 关掉的 job 有理由 | **命中** | `0454→0523/scheduler.json` 全部 `status=done` 的 job 均有 `status_reason` 非空：`target closed` / `superseded by recheck` / `superseded by task_follow_up` |
| V6 | 不回归 | **部分命中（字面未通过，实质不回归）** | `watchdog_alert` 消息全程 0；`watchdogs[]` 在 `0452–0518` 为空，在 `0523` 为 1 条 `request_waiting`（非回归告警，见 §3.2） |

### F-A..F-D 硬失败

| 编号 | 判定 | 结果 | 说明 |
|---|---|---|---|
| F-A | 同任务 ≥2 条 `artifact_path_missing` | **未命中（无失败）** | 2 次调用仅 1 条，去重生效 |
| F-B | 任一 403 不含 creator 名 | **未命中（无失败）** | 探针 nudge 含 `@cos`，`observations` 称 403 含 creator |
| F-C | 计划卡没出现 | **未命中（无失败）** | `request-9` 已出现 |
| F-D | 批准后 5min 无 lane | **未命中（无失败）** | 批准 `04:53:01Z` 后约 18s 拆出 4 条 lane |

### 人工干预

| 项 | 计数 | 说明 |
|---|---|---|
| 循环内人工干预 | **1 次** | `POST /interview/answer {id:request-9,choice:approve}` `04:53:01Z`（`0523/broker-state.json:requests[request-9].answered.choice_id=approve`） |
| 停表后操作员探针 | 2 次 `POST /wiki/write-task`（`analyst` 身份对 `JOURN-15`） | 不计入循环内干预，见 §3.1 |

---

## 2 逐条证据

### V1 计划卡自动出现 — 命中

- **判据**：计划帖之后 ≤10 分钟、无人工动作，`requests` 出现 `dedupe_key=plan-approval:<issue>` 且 `status=pending`。
- **证据**：
  - `snapshots/0452/broker-state.json:messages[msg-8]` `channel=cos__human` `from=cos` `timestamp=2026-11-09T04:52:23Z` — 计划帖正文四段拆分，含 `JOURN-6.1..6.4`。
  - `snapshots/0452/broker-state.json:requests[0]` `id=request-9` `kind=approval` `status=pending` `channel=cos__human` `dedupe_key=plan-approval:JOURN-6` `issue_id=JOURN-6` `created_at=2026-11-09T04:52:24Z` `updated_at=2026-11-09T04:52:24Z`。`question` 含完整计划正文。
  - `snapshots/0454/broker-state.json:requests[request-9].status=answered` `answered.answered_at=2026-11-09T04:53:01Z` — 批准前已为 pending，说明非人工创建。
  - 时间差：`04:52:23Z`→`04:52:24Z` ≈ 1s，远 ≤10min。
  - 幂等：全程仅 1 张 `plan-approval:JOURN-6`（`0452:1` → `0454:1` → `0455:1` → `0518:1` → `0523:1`），无重复。
- **对照**：`snapshots/0410`（pre-issue）`requests` 0，`tasks` 仅 4 条 canceled starter，排除预置。

### V2 拒写文案点名 creator — 命中（仅探针；循环内未观察）

- **判据**：任一 lane 的 `/wiki/write-task` 403 文本包含 `@<creator>`、`definition.artifact_path`、`cannot succeed`。
- **循环内事实**：`snapshots/0452,0454,0455,0518` 的 `messages` 中 `kind=artifact_path_missing` 计数均为 0；`broker-state.json:actions` 在 `0452–0518` 无 `kind=artifact_path_missing`；`tasks.json` 全程 `definition=null`（见下），但无任何 bot 尝试写 wiki 的记录。`observations §2-3` 与快照一致：“整轮 `broker.log` 里没有一条 `wiki write-task` 记录”。
- **探针证据**（停表后，非循环行为）：
  - `snapshots/0523/broker-state.json:messages[msg-64]` `id=msg-64` `from=system` `channel=analyst__cos` `kind=artifact_path_missing` `timestamp=2026-11-09T05:23:08Z` `content="@cos: JOURN-15 has no definition.artifact_path, so @analyst's write to team/research/probe.md was refused. Pin the path on the task (or have @analyst deliver through a human request) — until then the lane can only leave notes in a notebook. Owner: @analyst."` — 含 `@cos`（`JOURN-15.creator=cos`）与 `definition.artifact_path`，`channel` 匹配 `JOURN-15.channel=analyst__cos`。
  - `snapshots/0523/broker-state.json:actions[action-55]` `id=action-55` `kind=artifact_path_missing` `channel=analyst__cos` `actor=system` `related_id=JOURN-15` `summary="artifact write refused on JOURN-15: team/research/probe.md"` `created_at=2026-11-09T05:23:08Z` — 与 `msg-64` 同步，证明 403 链路已执行 `nudgeUndeclaredArtifactPath`。
  - `observations §3` 声称两次 `POST /wiki/write-task {slug:analyst,task_id:JOURN-15,path:team/research/probe.md}` 均 403 且文案含 `@cos`、`definition.artifact_path`、`cannot succeed`。快照不存 HTTP 响应体，无法独立复核 `cannot succeed` 字样；但 `msg-64` 的措辞（`was refused. Pin the path...`）与 `authorizeTaskArtifactWrite` 的 403 文案同源，且包含两要素，可印证修复已部署。
- **判定**：按 `prereg §4 V2` 字面，循环内无 lane 触发写 wiki，故**循环内为“未观察”**；若计入停表后探针，则文案要素齐全（`@cos`+`definition.artifact_path` 快照可证，`cannot succeed` 仅有叙述作证），去重与落点另见 V3/V4。此证据**不能**证明 bot 在正常回合会读提示（见 §4）。

### V3 提示只喊一次 — 命中

- **判据**：每个任务在任务频道里最多 1 条 `kind=artifact_path_missing`；同一任务第 2、3 次被拒不再新增。
- **证据**：
  - `snapshots/0452/broker-state.json:messages` kinds=`{issue_lifecycle:4,issue_created:1,human_request_raised:1}` `artifact_path_missing=0`
  - `snapshots/0454` kinds `artifact_path_missing=0`
  - `snapshots/0455` kinds `artifact_path_missing=0`
  - `snapshots/0518` kinds `{task_stalled:3,...}` `artifact_path_missing=0`（`watchdogs=0`）
  - `snapshots/0523/broker-state.json:messages` kinds `{...,artifact_path_missing:1}` 仅 `msg-64` 一条。
  - `0523/broker-state.json:channel_store` 与 `messages` 全量检索：按 `channel=analyst__cos` 过滤，仅 `msg-64` 为 `artifact_path_missing`，同任务无第二条。
  - `observations §3` 记录对 `JOURN-15` 调 `POST /wiki/write-task` ×2，两次均 403 但频道仅 1 条 — 与快照计数一致，证明按任务去重生效（`prereg §9` 所述“同一任务换路径重试仍只有一条是设计”）。
- **F-A 关联**：未出现 ≥2 条，故 F-A 未触发。

### V4 提示落在对的地方 — 命中

- **判据**：消息出现在任务自己的频道（如 `analyst__cos`/`ae__cos`），且同时 @creator 与 @owner。
- **证据**：
  - 任务归属：`0523/broker-state.json:tasks[JOURN-15]` `id=JOURN-15` `channel=analyst__cos` `owner=analyst` `created_by=cos`（creator）`definition=null`；`tasks.json` 同步；`0454/0455/0518/0523` 均一致。
  - 消息位置：`0523/messages[msg-64]` `channel=analyst__cos` — 与 `JOURN-15.channel` 完全一致，非 `cos__human` 或其他 lane。
  - @ 覆盖：`msg-64.content` 含 `@cos: JOURN-15 has no definition.artifact_path, so @analyst's write...` 与尾部 `Owner: @analyst.` — 同时 @creator（`cos`）与 @owner（`analyst`）。
  - 反例排除：全量 `0523/messages` 中仅此一条 `artifact_path_missing`，无跨频道错投。

### V5 关掉的 job 有理由 — 命中

- **判据**：`/scheduler` 里本轮新关的 task job，`status=done` 且 `status_reason` 非空。
- **证据**（以 `scheduler.json` 为主，`broker-state.json:scheduler` 同步）：
  - `0454/scheduler.json`：`slug=request-follow-up:cos--human:request-9` `kind=request_follow_up` `status=done` `closed_at=2026-11-09T04:53:01Z` `status_reason="target closed"`（`request-9` 被回答后关闭）；`slug=task-follow-up:ae--cos:journ-17` `kind=task_follow_up` `target_id=JOURN-17` `status=done` `closed_at=2026-11-09T04:53:44Z` `status_reason="superseded by recheck"`。
  - `0455/scheduler.json`：同 `0454`，新增关闭仍为上述 2 条，`recheck:ae--cos:task:journ-17` 仍 `scheduled`。
  - `0518/scheduler.json`：`recheck:ae--cos:task:journ-17` `kind=recheck` `target_id=JOURN-17` `status=done` `status_reason="superseded by task_follow_up"`；新增 `recheck:analyst--cos:task:journ-21` `status=done` `status_reason="superseded by task_follow_up"`；`request-follow-up:request-9` 仍 `done/target closed`。
  - `0523/scheduler.json`：与 `0518` 一致（`task-follow-up:ae--cos:journ-17` 已在 `0518` 变为 `scheduled`，`recheck` 保持 `done`），所有 `done` 项 `status_reason` 非空；新增 `request-follow-up:analyst--human:request-43` `scheduled`（未关）。
  - `broker-state.json:scheduler` 在 `0454`（15 jobs）、`0518`（17 jobs）、`0523`（17 jobs）与 `scheduler.json` 完全一致。
- **覆盖度**：本轮实际观察到的 `status_reason` 仅 `target closed`（request job）、`superseded by recheck`、`superseded by task_follow_up` 三种；`prereg` 所述的 `awaiting review (<state>)`、`closed`、`target closed`（task 维度）等分支本轮未触发，故 V5 仅证明“已关的都有理由”，不能证明分支全覆盖（见 §4）。

### V6 不回归 — 部分命中（`watchdog_alert` 0，未出现 J3-2 唤醒被拒；`watchdogs[]` 终态非空但非回归）

- **判据**：全轮 `kind=watchdog_alert` 计数 0；`watchdogs[]` 为空。
- **证据**：
  - `kind=watchdog_alert` 消息：`0452` 0、`0454` 0、`0455` 0、`0518` 0、`0523` 0（全量 `messages` `kinds` 统计均无此 kind；`grep` 全快照无命中）。
  - `watchdogs[]`：`0452` `[]`、`0454` `[]`、`0455` `[]`、`0518` `[]`、`0523` `[watchdog-1]`。
  - `0523/broker-state.json:watchdogs[0]` `id=watchdog-1` `kind=request_waiting` `channel=cos__human` `target_type=request` `target_id=request-13` `owner=cos` `status=active` `created_at=2026-11-09T05:23:25Z` `updated_at=2026-11-09T05:23:25Z` `summary="Still waiting on Human interview in #cos__human: 续约报价执行前置..."`。
  - `0518/broker-state.json:requests[request-13]` `kind=interview` `channel=cos__human` `status=pending` `created_at=04:53:15Z` — 自 `0454` 起持续 `pending` 未被回答；`request-43`（`analyst__human`）同为 `pending`。
- **判定**：按字面“`watchdogs[]` 为空”在终态 `0523` 不成立；但该条目 `kind=request_waiting` 目标 `request-13`，是“人类没答采访题”的挂起提醒，非 J3-2 那种 `watchdog_alert` 唤醒被拒。`watchdog_alert` 消息全程 0，且 `0518`（循环结束前最后一次 5min 快照，`05:18Z`，距末条正文 `05:08:08Z` 10min）`watchdogs` 仍为空，证明循环停转期间未产生回归告警。`0523` 的 `watchdog-1` 创建于探针后 `05:23:25Z`，属停表后定时器到期产生的 `request_waiting`，不应判为回归。若以 `0518` 为停表时刻快照，则 V6 两子项均成立；以 `0523` 为终态则字面第二子项不成立。本卡记为“部分命中/实质不回归”。
- **补充**：`observations §2-6` 称“`watchdog_alert` 0 条、`watchdogs[]` 为空、`task_stalled` 0 条”与快照不完全一致——快照显示 `0518/0523` 有 3 条 `task_stalled`：`msg-61 cos__human 05:05:08Z`、`msg-62 analyst__cos 05:08:08Z`、`msg-63 cos__reviewer 05:08:08Z`，内容为静默 10–11min 提醒。此为 `task_stalled`（任务静默），非 `watchdog_alert`，不计入 V6，但说明 `observations` 该句以 `0518` 前或定义不同为准，快照为准。

### F-A 同任务 ≥2 条 `artifact_path_missing` — 未命中（无硬失败）

- 同 V3：`0523` 仅 1 条 `msg-64`，`0452–0518` 0 条；2 次探针调用未产生第二条，去重生效。

### F-B 任一 403 不含 creator — 未命中（无硬失败）

- 探针 nudge `msg-64` 含 `@cos`（creator）与 `definition.artifact_path`；`observations §3` 称两次 403 文案均含三要素；快照无反例。`tasks[JOURN-15].created_by=cos` 与 `@cos` 对应。

### F-C 计划卡没出现 — 未命中（无硬失败）

- `0452/request-9` 已出现（见 V1），`answered` 于 `04:53:01Z`。

### F-D 批准后 5min 无 lane — 未命中（无硬失败）

- 批准 `0454/requests[request-9].answered_at=04:53:01Z`；`0454/tasks` 已含 `JOURN-15/17/19/21` 4 条 `parent_issue_id=JOURN-6`，创建 `msg-16/18/20/22` `timestamp=04:53:19Z`（批准后 ~18s）；全部 `definition=null` 符合“无产物路径”触发条件，拆分路径完好。

### 人工干预计数 — 1 次（合规）

- **时间线**（`observations §1` + 快照印证）：
  - `04:51:54Z` `POST /onboarding/complete` + `cancel` 4 条 starter（环境准备，不计为 §6 干预）
  - `04:52:02Z` 父 Issue `JOURN-6` 创建（`cos__human` `owner=cos` `created_by=human`）
  - `04:52:23Z` cos 计划帖 `msg-8`
  - `04:52:24Z` 自动 `request-9` `pending`
  - `04:53:01Z` **唯一人工操作** `POST /interview/answer {id:request-9,choice:approve}` → `0454/requests[request-9].status=answered`
  - `04:53:15Z` 自动 `request-13` `interview` `pending`（`msg-14`）
  - `04:55:38Z` 自动 `request-43` `analyst__human` `pending`（`msg-44`+`msg-45` pointer）
  - 之后至 `05:08:08Z` 仅 bot 自主消息，无人工回复；`05:08:08Z` 后 15min 无新消息停表。
- **探针**：`05:23:08Z` 两次 `POST /wiki/write-task`（`analyst` 身份）发生在停表后 `workspace shred` 前的验证脚本，非循环内回合，不计入 `prereg §6` 上限；若计入则为停表后操作，不影响“循环内仅 1 次”结论。
- **结论**：`prereg §6` 允许 1 次点批准卡，本轮循环内恰好 1 次，合规；其余未有人工答题（`request-13/43` 始终 `pending`）。

---

## 3 特别核实

### 3.1 `artifact_path_missing` 是谁触发的、去重是否成立、能否证明 bot 会读

- **谁触发**：`0523/messages[msg-64]` `timestamp=2026-11-09T05:23:08Z` 与 `0523/actions[action-55]` `created_at=05:23:08Z` 的时间戳均远晚于循环末条正文 `05:08:08Z`，且 `observations §3` 明确声明该消息由“操作员在停表后用手工脚本以 `analyst` 身份对 `JOURN-15` 调两次 `/wiki/write-task`”产生，非任何 bot 回合内调用。快照印证：`0452–0518` 4 个循环内快照的 `artifact_path_missing` 计数均为 0，`0518→0523` 间才出现 1 条，且 `action-55.actor=system` `related_id=JOURN-15` 符合 `nudgeUndeclaredArtifactPath` 按任务去重的写入路径，无对应 headless 回合消息。
- **去重是否成立**：`observations` 称调 2 次，两次均 403；快照显示频道 `analyst__cos` 仅 1 条 `msg-64`，`actions` 仅 1 条 `artifact_path_missing`，`messages` 全局仅 1 条该 kind。符合 `prereg §9`“按任务去重，同一任务换路径重试仍只有一条是设计”——去重**成立**。
- **能否证明 bot 会读**：**不能**。证据链：
  1. 循环内 4 条 lane（`JOURN-15/17/19/21`）全程 `definition=null`（`0454/0455/0518/0523/tasks.json` 及 `broker-state.json:tasks` 均 `def=null`），且 `actions`/`messages` 无任何 `wiki write-task` 尝试；两个 bot 选择“留 notebook + 等源数字”（`0518/scheduler.json` `payload` 含 checklist/notebook 路径；`artifacts/notebooks/journ-6-1-checklist.md` 与 `2026-11-09-journ-62-...md` 均为 `[NEEDS CONFIRMATION]` 草稿，未走 wiki）。
  2. 探针消息出现于停表后，bot 已无回合可读；快照中无 bot 对 `msg-64` 的回复或后续 `human request` 修正。
  3. 因此本轮仅证明“提示到达正确位置且去重正确”，未证明“bot 在正常回合会读提示并改为 human request/补 `artifact_path`”。

### 3.2 `watchdogs` 非空与 `watchdog_alert` 计数

- `0523/broker-state.json:watchdogs` 1 条 `watchdog-1` `kind=request_waiting` `target=request-13` `created_at=05:23:25Z` — 目标为 `cos__human` 的 `interview` `request-13`（`04:53:15Z` 起 `pending`），是“人类没答采访题”的挂起提醒，非 J3-2 那种任务唤醒被拒。
- `kind=watchdog_alert` 的 `messages` 计数：`0452:0, 0454:0, 0455:0, 0518:0, 0523:0`（全快照 `kinds` 统计无此 kind，`grep` 无命中）。
- `task_stalled` 消息：`0518/0523` 各有 3 条 `msg-61/62/63`（`05:05:08Z, 05:08:08Z`），为任务静默 10–11min 提醒，非 `watchdog_alert`，不计入 V6。
- 结论见 V6 判定。

---

## 4 本轮**不能**证明什么

1. **Bot 是否真会把 403 文案读进去并改走 human request / 补 `definition.artifact_path`**  
   本轮 4 条 lane 全程未尝试写 wiki（`0452–0518` `artifact_path_missing=0`，`actions` 无写 wiki；`0518` 时 `request-13/43` 已为 pending，但 bot 未再试写），且 notebook 草稿明确“等源数字”（`journ-6-1-checklist.md` 7 项全 `[NEEDS CONFIRMATION]`，`JOURN-6.2` 草稿亦全标待确认）。唯一验证 403→nudge 的链路是停表后操作员探针（`0523/msg-64`），bot 无机会读取并修正。`prereg §9` 预期风险已点明此局限，本轮**只能证明“提示到达”**，**不能证明“bot 据此修正”**。

2. **`status_reason` 是否覆盖所有关闭路径**  
   `scheduler.json` 仅观察到 `superseded by recheck`（`0454 task-follow-up:ae--cos:journ-17 done`）、`superseded by task_follow_up`（`0518 recheck:ae... done, recheck:analyst... done`）、`target closed`（`request-follow-up:request-9 done`）三种。`prereg` 提及的 `awaiting review (<state>)`（审阅门）、`closed`（其它终态）、`target closed`（task 维度）、以及 `schedulerJob` 重新排期清空等分支**未观察**；`status_reason` 非空仅对已关闭的 job 成立，未覆盖的路径仍为**未观察**，不许默认通过。

3. **`awaiting review` 与 `closed` 分支**  
   本轮无任务进入审阅门关闭（`JOURN-15/19` 保持 `running`，`JOURN-17/21` 为 `decision/review` 但对应 job 为 `superseded` 而非 `awaiting review`），亦无 `closed` 收敛关闭；故该两类 `status_reason` 文案未在快照中出现。

4. **同任务多路径去重的边界**  
   探针仅对 `JOURN-15` 以同一 `path=team/research/probe.md` 调 2 次，证明同任务同路径去重；未验证“同任务换路径仍只 1 条”（`prereg §9` 设计）的实际表现，亦未验证不同任务各 1 条的隔离性（循环内无触发）。

5. **Wiki 产物与归档后复核**  
   `artifacts/wiki-tree.txt` 显示 wiki 新增仅 `agents/analyst/notebook/*` 两篇 notebook（`journ-6-1-checklist.md` `644c5fd`、`...journ-62-...md` `5e51a5f`），无 `team/research` 或正式交付物落盘；`wiki git log` 无 `team/research/probe.md`。符合“缺源数字时不落 wiki”的正确行为，但也意味着 `definition.artifact_path` 缺失的修复未在真实交付物路径上被检验。

6. **`watchdogs` 的回归判定**  
   终态 `0523` 的 `request_waiting` 非回归，但 `watchdog_alert` 为 0 仅证明未触发 J3-2 类告警，未证明 `request_waiting` 定时器阈值、去重或提醒文案的正确性。

---

## 5 快照清单与证据索引

| 快照 | 文件 | 关键字段 |
|---|---|---|
| 0410 | `broker-state.json` `scheduler.json` | 基线：`requests 0`, `tasks 4×canceled`, `scheduler 7 jobs` |
| 0452 | `broker-state.json` `tasks.json` `scheduler.json` | `requests[request-9] pending 04:52:24Z`, `msg-8 04:52:23Z`, `tasks[JOURN-6] planning/plan` |
| 0454 | `broker-state.json` `scheduler.json` | `request-9 answered 04:53:01Z`, `tasks JOURN-15/17/19/21 created 04:53:19Z def=null`, `scheduler task-follow-up:ae done superseded by recheck` |
| 0455 | `broker-state.json` `scheduler.json` `tasks.json` | `request-13 pending 04:53:15Z`, `scheduler 同 0454` |
| 0518 | `broker-state.json` `scheduler.json` | `request-43 pending 04:55:38Z`, `messages 55条 含 task_stalled×3`, `recheck×2 done superseded by task_follow_up`, `watchdogs 0` |
| 0523 | `broker-state.json` `scheduler.json` `tasks.json` | `msg-64 artifact_path_missing 05:23:08Z analyst__cos`, `action-55 同`, `watchdog-1 request_waiting 05:23:25Z request-13`, `messages 56条` |

> 未观察项已显式标注“未观察”，不默认通过。所有判定以快照字段为准，`observations.md` 时间线仅作交叉引用。

