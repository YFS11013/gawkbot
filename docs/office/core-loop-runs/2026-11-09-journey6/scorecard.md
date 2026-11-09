# Journey 6 Scorecard — 独立判分（2026-11-09）

> 评审范围：`prereg.md` §4（V1–V6）与 §5（F-A..F-D）唯一来源，快照 `snapshots/0622,0624,0627,0631,0650,0700,0701,0706` 四件套（`broker-state.json` / `tasks.json` / `scheduler.json` / `broker-log-slice.txt`），校核 `observations.md` 陈述。
> 模型 `opencode-go/crouton-alpha-free`，隔离办公室 `journey6`（broker :7910）。停表 07:06（45 分钟上限）。只做只读判分，不动任何进程/端口。
> 语言：中文；每条结论均锚定快照目录 + 字段 + 时间戳。未观察即写未观察。

---

## 0. 运行总览

- 父 Issue `JOURN-8`（`Journey 6: Acme 续约报价与跟进`，`channel=cos__human`，`owner=cos`，`created_by=human`，创建 06:20:59）在 `0622` 为 `lifecycle=planning / stage=plan`，批准后（06:23:19）转 `lifecycle=running / stage=breakdown`，至 `0706` 仍未分解出任何子 Issue，且无 `definition`。
- 全程 8 份快照 `tasks` 数量恒为 6（`ai-revops-1..4` 已 `canceled` + `JOURN-3` + `JOURN-8`），无新增 lane 任务。
- `broker-log-slice.txt` 8 份均为 0 字节，无任何 `write-task` / `wiki` / `403` 行。
- `messages` 中 `kind=artifact_path_missing`、`kind=watchdog_alert`（聊天可见）全程 0 条；`task_stalled` 1 条（06:33:35 可见），`human_request_raised` 2 条（`request-12/16`）。
- `watchdogs` / `actions` 在 06:52:15 后出现 1 条 `request_waiting` + `watchdog_alert`（`request-16`），但聊天无对应可见消息——与 prereg §9 外的接线现象一致。

---

## 1. 判定摘要表

### 1.1 V1–V6（验证项）

| 编号 | 名称 | 判定 | 一句话证据 |
|------|------|------|------------|
| V1 | lane 真的尝试写 wiki | **未观察** | 8 份 `broker-log-slice.txt` 长度 0，无 `wiki write-task refused`；`broker-state.messages` 8 份均无 `artifact_path_missing`；`tasks.json` 8 份子任务数 0 |
| V2 | 文案可执行（@creator / definition.artifact_path / cannot succeed） | **未观察** | 无 403 文本可检；隶属 V1 的前置未触发 |
| V3 | 每个任务只喊一次（去重） | **未观察** | 无被拒任务，故去重逻辑未被锻炼；`messages` 中 `artifact_path_missing` 计数全程 0（≤1 空真） |
| V4 | bot 会改做法（≤15 min 内转交/改提交/停重试） | **未观察** | 无首次 403 时间锚点，后续行为无从考核 |
| V5 | 不回归（单张计划卡 + watchdog 0 + 新关 job 带 status_reason） | **部分未命中** | 计划卡 ✅；`status_reason` ✅；`watchdog_alert`（actions/watchdogs）=1 ❌（消息层为 0）——详见 §2.5 |
| V6 | 不刷屏（同任务被拒 ≤3 次） | **未观察**（空真上可记“命中”，但无锻炼价值） | 被拒次数 0，满足 ≤3 但未经历重试路径 |

> 按 `prereg.md` §9 第一条：模型可能贴正文而不调写工具，45 min 内无写尝试则 V1–V4 记「未观察」。本轮完全命中该预告情形。

### 1.2 F-A..F-D（硬失败）

| 编号 | 名称 | 判定 | 证据 |
|------|------|------|------|
| F-A | 同任务 ≥2 条 `artifact_path_missing`（去重失效） | **未触发** | 全程 `messages` 0 条该 kind |
| F-B | 同 lane 同路径连续 >3 次被拒且无变化 | **未触发** | 0 次被拒，无路径可累计 |
| F-C | 计划卡缺席（J3-1 回归） | **未触发** | `0622` 起即有 `msg-11` 计划正文 + `request-12`（`dedupe=plan-approval:JOURN-8`），且仅一张 |
| F-D | lane 因 403 直接放弃、无交付或上报 | **未触发**（前提未发生） | 无 403，无放弃行为；`JOURN-8` 维持 `running` 未被取消 |

### 1.3 人工干预计数

| 统计口径 | 次数 | 明细 |
|----------|------|------|
| 对 `JOURN-8` 场景的辅助性干预 | **1** | 06:23:19 批准 `request-12`（`approve`）—— `prereg §6` 唯一允许 |
| 对 broker 的一切人类写操作 | **2** | 上述 1 次 + 06:27:32 取消 `JOURN-3`（环境清理，见 §5 理由） |

---

## 2. V1–V6 逐条判定与证据

### V1 — lane 真的尝试写 wiki

- **判定：未观察**
- **判据（prereg §4）：** 隔离办公室 `broker.log` 出现至少 1 条 `wiki write-task refused: task=<id> slug=<bot>`，且 slug 属于本轮 lane owner（非操作员探针）。
- **证据：**
  - `snapshots/*/broker-log-slice.txt`（0622/0624/0627/0631/0650/0700/0701/0706）—— 逐份 `read` 长度 0 字节，`grep` 关键词 `write-task` / `wiki` / `refused` / `403` 命中 0。`0701` 与 `0706` 仍为 0（最后快照已覆盖停表前 45 min 全程）。
  - `snapshots/0622..0706/broker-state.json` → `messages`（长度 12→18）全量 `kind` 直方图：`None / issue_lifecycle / issue_created / human_request_raised / task_stalled`，无 `artifact_path_missing`，无含 `refused / artifact_path / wiki write` 的 `body`。抽检 `0706` 的 `messages[msg-23]=task_stalled` 为唯一非计划消息。
  - `snapshots/0622..0706/tasks.json` → 恒 6 条，无以 `JOURN-8` 为 `parent_issue_id` 的子任务；`scheduler.json` 亦无为 lane 创建任务的调度痕迹。
  - `snapshots/0622..0706/broker-state.json` → `actions` 的 `kind` 直方图：`task_created / task_canceled / task_updated / automation / request_created / request_answered / task_stalled / watchdog_alert`，无 `wiki`/`write-task` 相关 action。
- **为什么未观察：**
  - 直接原因：cos 在批准后未执行分解，`JOURN-8` 本身无 `definition`，但 lane 未被创建，写作前置链断裂。前述批准后的唯一回合（06:23:26 的 `msg-21`）产出指向 `JOURN-3` 的管道数据追问，未触及 `JOURN-8` 的两件交付物。
  - 间接原因：onboarding 自建 `JOURN-3` 抢占注意力（`cos__human` 同频道），且调度上批准后无短周期重投（见 §4.1），导致 45 min 内无写调用。
  - **是否属于 prereg §9 预告：** 属于。`prereg §9` 第一条原文预告“模型可能选择先把正文贴在聊天里、完全不调用写工具。若 45 分钟内没有一次写尝试，V1–V4 记「未观察」”——本轮即该分支。不能据此判定 J3-3 修复有效或无效。

### V2 — 文案可执行（三要素）

- **判定：未观察**
- **判据：** 403 error 文本含 `@<creator>`、`definition.artifact_path`、`cannot succeed` 三要素。
- **证据：**
  - 承 V1，无任何 403 行，`broker-log-slice.txt` 与 `messages` 均无错误文本可审。
  - `tasks.json` 中 `JOURN-8` 的 `created_by=human`，按设计若发生拒写应提示 `@human`，但本轮无实例可验。
- **为什么未观察：** 前置 V1 未触发，无文本可检。属于 §9 同一预告分支。
- **可核实的外围事实：** 基线提交 `d52293e` 的文案实现不在本轮日志中体现，不在本轮结论内。

### V3 — 每个任务只喊一次（去重）

- **判定：未观察**（若按“最多 1 条”空真则计数通过，但无锻炼意义，评分口径记未观察）
- **判据：** 每个被拒任务在 `messages` 里最多 1 条 `kind=artifact_path_missing`；同一任务重复被拒不新增。
- **证据：**
  - 8 份快照 `broker-state.json` → `messages` 中 `kind=artifact_path_missing` 全 0（`0622:12条, 0624:16条, 0650:18条含task_stalled, 0701/0706:18条`）。
  - `actions` 中亦无 `artifact_path_missing` kind。
  - 去重逻辑（按任务 ID）的正/反例均未出现，`watchdogs` 计数 0→1 与此无关。
- **为什么未观察：** 无被拒任务，去重分支未被执行。属于 §9 预告的衍生：§9 末句“同一任务换路径重试仍只有一条消息，这是设计”在本轮无样本可验。
- **边界说明：** 若硬套“≤1”则 0 条满足，但不应解读为“去重有效”，应解读为“未锻炼”。

### V4 — bot 会改做法（≤15 min 内）

- **判定：未观察**
- **判据：** 首次 403 之后 ≤15 min，同一 lane 至少出现 ① 频道内明确说要把产物交给 human request / 请 creator 钉路径；② 以 `artifact_path`（或 `verification=not_required` + 理由）成功 `submit_for_review`/`complete`；③ 不再重复同一路径的写入。
- **证据：**
  - 无首次 403 时间戳，锚点缺失，后续 15 min 窗口无从划定。
  - 全程 `messages` 无含 `artifact_path`、`verification`、`submit_for_review`、`complete` 的 lane 产出；`tasks.json` 中 `JOURN-8` 的 `review_state` 全程 `pending_review` 未变，`lifecycle` 自 `0624` 起 `running/breakdown` 冻结。
  - `0701`/`0706` 的 `watchdog_alert` 与 `JOURN-8` 无关（指向 `request-16`）。
- **为什么未观察：** V1 未触发则 V4 无输入。属于 §9 预告。另，调度上 `JOURN-8` 的下一次 `task_follow_up` 为小时级（07:23:19），即使有 403，15 min 内也无自动唤醒可承载“改做法”。

### V5 — 不回归

- **判定：部分未命中（2/3 子项通过，1 项在 actions/watchdogs 层未通过）**
- **判据三子项：** ① 计划卡自动出现且只有一张；② `kind=watchdog_alert` 计数 0；③ 本轮新关的 job 都带 `status_reason`。
- **证据：**
  - **子项① 计划卡：通过**
    - `0622/broker-state.json` → `messages msg-11`（06:21:31）为 `JOURN-8 计划（已进入 plan 阶段…）` 四段式正文；`requests request-12`（`kind=approval, dedupe_key=plan-approval:JOURN-8, channel=cos__human`）在 `0622` 为 `pending`，`0624` 起为 `answered`；`actions action-13/14` 对应 `automation` 与 `request_created` 时间戳 06:21:31/32。
    - 全程仅此一张计划卡，无重复：`scheduler.json` 中 `request-follow-up:cos--human:request-12` 在 `0622` 为 `scheduled/next_run=06:51:32`，`0624` 起为 `done/last_run=06:23:19/closed_at=06:23:19/status_reason=target closed`，无第二张卡生成。
  - **子项② watchdog_alert 计数：消息层 0，通过；系统层 1，未通过**
    - `0706` 之前（含 `0650`）`broker-state.messages` 中 `kind=watchdog_alert` 全 0，`actions` 中 `watchdog_alert` 全 0——满足“聊天不刷屏”本意。
    - `0701/broker-state.json` 新增：`watchdogs[0]`（`id=watchdog-1, kind=request_waiting, target_id=request-16, channel=cos__human, created_at=06:52:15Z`）+ `actions[25]`（`kind=watchdog_alert, source=watchdog, channel=cos__human, actor=watchdog, related_id=request-16, created_at=06:52:15Z, signal_ids=[signal-1]`）。`0706` 保持不变。
    - 该 `watchdog_alert` 源于 `request-16`（`JOURN-3` 的非阻塞 `interview`），非任务失败 watchdog，但按 prereg 字面“`kind=watchdog_alert` 计数 0”则不满足 0。`messages` 层仍为 0，`actions/watchdogs` 层为 1。
    - **判定立场：** 若按“聊天可见 `watchdog_alert`”口径则通过；按“系统任一层”口径则未命中。本 scorecard 如实记录分层结果，并在“顺带暴露问题”中归因到非阻塞采访的静默-转-有痕设计（见 §4.2）。
  - **子项③ 新关 job 带 status_reason：通过**
    - `scheduler.json` 与 `broker-state.scheduler` 一致：
      - `recheck:pipeline:task:ai-revops-2` 在 `0622` 起即 `done/closed_at=06:20:57/status_reason=target closed`
      - `request-follow-up:cos--human:request-12` 在 `0624` 起 `done/status_reason=target closed/closed_at=06:23:19`
      - `task-follow-up:cos--human:journ-3` 在 `0631` 起 `done/status_reason=target closed/closed_at=06:27:32`
    - 本轮内新关闭的 job 均含 `status_reason`，`0706` 仍保持。
- **为什么部分未命中：** 非场景任务 `JOURN-3` 的 `request-16` 触发了 `request_waiting` 看门狗，属于环境噪声对 V5 文面计数的污染，并非 `JOURN-8` 回归。

### V6 — 不刷屏（同任务被拒 ≤3 次，含 1 次允许重试）

- **判定：未观察**（空真层面“0≤3”通过，但无重试样本，不应记为验证通过）
- **判据：** 同一任务被拒次数 ≤3。
- **证据：** 承 V1，`broker-log-slice.txt` 0 行，被拒次数 0，`messages` 中无重复拒写。`F-A/B` 同理 0。
- **为什么未观察：** 无拒写实例，重试上限未被锻炼。属于 §9 预告的衍生未观察。

---

## 3. F-A..F-D 逐条判定与证据

### F-A — 同任务 ≥2 条 `artifact_path_missing`（去重失效）

- **判定：未触发**
- **证据：** `0622..0706/broker-state.json` → `messages` 中 `artifact_path_missing` 全 0；`actions` 亦 0。触发阈值（≥2）未达到。
- **说明：** 前置 V1 未触发，F-A 无输入；不能证去重失效，也不能证其有效。

### F-B — 同 lane 同路径连续 >3 次被拒且无变化

- **判定：未触发**
- **证据：** 被拒次数 0，连续 >3 的计数器未启动；`broker-state` 无任何 `write-task refused`。
- **说明：** 同属 V1 未触发的衍生；prereg §5 所述“文案没起作用，只会重试”在本轮无样本。

### F-C — 计划卡缺席（J3-1 回归）

- **判定：未触发**
- **证据：** 见 V5 子项①。`0622` 的 `msg-11` 计划正文、`request-12` 批准卡、`scheduler` 的 `request-follow-up:request-12` 三处互证。自 `0624` 后 `request-12.status=answered`，无第二张计划卡，回退未发生。
- **说明：** 不回归项中唯一完全正向的证据。

### F-D — lane 因 403 直接放弃、无任何交付或上报

- **判定：未触发**（前提缺失）
- **证据：** 无 403，无 `task_canceled` 指向 `JOURN-8`；`JOURN-8` 全程 `status=in_progress`，`0706` 仍 `lifecycle=running/stage=breakdown`，未被放弃。`messages` 中无 lane 宣告放弃的正文。
- **说明：** 本轮行为是“未分解、未尝试”，而非“尝试后放弃”，与 F-D 描述的更差情形不同。

---

## 4. 三件事核实（按 context 第 3 点）

### 4.1 批准（06:23:19）之后有没有子 Issue？`scheduler.json` 里指向 `JOURN-8` 的下一次唤醒是什么类型、什么时候？

- **有没有子 Issue：没有。**
  - `tasks.json` 在 `0622`（批准前）为 `JOURN-8` 1 条父任务；`0624`（批准后首份快照，06:24:00，距批准 +41s）、`0627`、`0631`、`0650`、`0700`、`0701`、`0706`（停表）均为同一 6 条集合，无 `parent_issue_id=JOURN-8` 的任何新增任务。`broker-state.actions` 中无 `task_created` 指向 `JOURN-8` 子任务。
  - `observations.md` 的“没有创建任何子任务”与快照一致，可采信。

- **指向 `JOURN-8` 的下一次唤醒：**
  - 快照：`snapshots/0622/scheduler.json`（批准前）→ `task-follow-up:cos--human:journ-8` 为 `kind=task_follow_up`，`due_at=2026-11-09T07:20:59Z`，`next_run=2026-11-09T07:20:59Z`，`status=scheduled`。
  - 批准后（`0624/scheduler.json` 起，批准时间 06:23:19 同秒 `JOURN-8` 被 `update`）该条目更新为 `due_at=2026-11-09T07:23:19Z`，`next_run=2026-11-09T07:23:19Z`，`kind=task_follow_up`，`status=scheduled`，`enabled=false`，`payload` 为 `JOURN-8` 任务单原文。此值在 `0627/0631/0650/0700/0701/0706` 七份快照中保持不变（`0706` 仍 `next_run=07:23:19`）。
  - **类型：** `task_follow_up`（任务跟进提醒），非 `task_recheck` / `task_reminder` / `request_follow_up`。
  - **时间：** 07:23:19（批准后约 60 min），小时级。批准 → 首次分解之间无短周期重投，仅有该 `task_follow_up`。
  - **交叉证据：** `broker-state.json` 的 `scheduler` 数组同值；`tasks.json` 中 `JOURN-8` 的 `follow_up_at/due_at` 自 `0624` 起同步为 `07:23:19`，`recheck_at=07:38:19` / `reminder_at=07:53:19` 未被触发（停表 07:06 早于三者）。

- **机制含义：** `observations.md` 第 2.1 节所述“批准后没有分解检查点”属实——`ScheduleNoTraceRetry` 仅在回合“什么都没留下”时排，06:23 那轮因产出 `JOURN-3` 内容而不排，系统对“回合有产出但不是本任务要的产出”不可见。

### 4.2 `request-16` 的 `request-follow-up` 在 06:52:15 触发后，`messages` 里有没有对应的人类可见消息？（`watchdogs` / `actions` 里有，聊天里有没有？）

- **结论：没有。**
- **证据：**
  - `snapshots/0700/broker-state.json`（06:52 前最后快照）→ `watchdogs 0`，`actions` 无 `watchdog_alert`，`messages` 18 条（最后一条 `msg-23 task_stalled` 来自 06:33:35）。
  - `snapshots/0701/broker-state.json`（首份覆盖 06:52:15 的快照）→ 新增 `watchdogs[0]=watchdog-1`（`kind=request_waiting, target_id=request-16, created_at=06:52:15Z`）与 `actions[25]=watchdog_alert`（`created_at=06:52:15Z, channel=cos__human, related_id=request-16`），但 `messages` 仍为 18 条，与 `0700` 完全相同，直方图无新增 `watchdog_alert` / `request_waiting` kind，`body` 无新增含 `request-16` 的聊天行。
  - `snapshots/0706/broker-state.json` 复核不变。
  - `scheduler.json` 侧：`request-follow-up:cos--human:request-16` 在 `0650` 为 `next_run=06:52:00/status=scheduled`，`0701` 起为 `last_run=06:52:15/next_run=07:22:15/status=scheduled`，证实确在 06:52:15 执行且重排 30 min，但 `messages` 层无人类可见投递。
- **设计归因：** `request-16` 为 `kind=interview`，`blocking` 为空（8 份 `broker-state.requests` 均为 `blocking=None`，`dedupe_key=None`），`processRequestJob` 仅在 `req.Blocking && !existing` 时才投自动化消息；`request-16.blocking` 为空故不投递。此为非阻塞采访提醒的现行接线，与 `observations.md` 第 2.3 节“J5 的 `request-13` 同型现象”一致，可采信。

### 4.3 `broker.log` 里有没有 bot 发起的 `wiki write-task`（区分于操作员探针；本轮应无探针）

- **结论：没有。**
- **证据：**
  - `snapshots/0622..0706/broker-log-slice.txt` 8 份均为 0 字节（`read` 逐份验证），无任何含 `write-task` / `wiki` / `refused` / `403` 的行。`observations.md` 所述“`broker.log` 的 `write-task` 行数 0”可采信。
  - `broker-state.actions` 全程无 `wiki` 相关 kind；`messages` 无 `artifact_path_missing`。
  - 本轮无操作员探针（`observations.md` 记“本轮应无探针”，快照中无探针痕迹，`actions` 无探针写入）。
- **区分说明：** 即使有探针也应在日志中以特定 slug 区分，但本轮日志为空，故无需区分。

---

## 5. 人工干预计数（从时间线自数，含对 06:27:32 取消 `JOURN-3` 的定性）

### 5.1 时间线全量人类写操作

| # | 时间（UTC） | 动作 | 来源 | 证据 |
|---|-------------|------|------|------|
| 1 | 06:23:19 | `POST /interview/answer {"id":"request-12","choice_id":"approve"}` → `{"ok":true}`，`request-12` 置 `answered`，`JOURN-8` 置 `running/breakdown` | `observations.md §1`；`0624/broker-state.json` → `requests request-12 status=answered`，`actions action-20 task_updated [plan approved] created_at=06:23:19`，`scheduler request-follow-up:request-12 last_run=06:23:19 status=done` | 人类对批准卡的唯一答复 |
| 2 | 06:27:32 | 取消 `JOURN-3`（`{"changed":true}`，`task_canceled` 落账） | `observations.md §1`；`0631/broker-state.json` → `tasks JOURN-3 status=canceled/completed_at=06:27:32`，`scheduler task-follow-up:journ-3 done/closed_at=06:27:32/status_reason=target closed`，`actions` 新增 `task_canceled` | 环境清理 |

- 此外无其他 `POST /task/cancel` / `interview/answer` 人类写操作（`request-16` 全程 `pending` 未被回答，`JOURN-8` 未被取消）。

### 5.2 06:27:32 取消 `JOURN-3` 是否算“场景干预”？

**本 scorecard 的判定：不算对 `JOURN-8` 场景的辅助性干预，但算对 broker 状态的人工写操作；按 prereg 文面应计为“其余人工步骤”并如实记录。**

理由：

1. **对象非场景任务。** `JOURN-3`（`[@analyst] Repeated errors blocked…`，`parent_issue_id=ai-revops-2`，`created_by=system`）是 onboarding 失败后系统自建的 incident，与 `prereg §3` 定义的场景任务 `JOURN-8`（`created_by=human` 的 Acme 报价/跟进）无数据与交付物交集。取消它未向 `JOURN-8` 注入任何事实、路径或提示。
2. **意图为隔离噪声，非助推交付。** `JOURN-3` 与 `JOURN-8` 同处 `cos__human`，cos 的注意力被前者占据（`0622` 的 `msg-10/14/15` 与 `0624` 的 `msg-21` 均为管道数据追问）。取消 `JOURN-3` 是为切断跨任务干扰、观察 `JOURN-8` 是否自恢复——效果上 `JOURN-8` 在其后仍无分解（`0631` 起 `JOURN-3=canceled` 但 `JOURN-8` 无新消息、无 `task_stalled` 之外的唤醒），证实问题不在“一次分心”。
3. **prereg 计数口径的两种读法需同时披露：**
   - **严格文面：** `prereg §6`“只允许 1 次：点批准卡。其余人工步骤必须逐条记进 observations”——则 06:27:32 是第 2 次人工步骤，`observations.md` 已逐条记录，满足“记账”要求，但超出“1 次”上限的文面数值为 2。
   - **语义口径（本 scorecard 采用的摘要表口径）：** 对 `JOURN-8` 产出有因果助推的“场景干预”仍为 1（仅批准）；环境清理单列，不与场景干预混计。正文同时给出“一切人类写操作=2”的总数，避免歧义。
4. **不计入 F 类硬失败。** 取消 `JOURN-3` 未掩盖 F-A..F-D，亦未制造新的 `watchdog_alert`（后者的 `request-16` 看门狗在 `JOURN-3` 取消后仍因 `request-16` 未答而触发，时间 06:52:15 晚于取消）。

**结论：** 摘要表记“场景干预 1 | 人工写操作 2（含环境清理）”；`observations.md` 已按要求逐条记录，符合 prereg 的记账义务。

---

## 6. 本轮不能证明什么 / 顺带暴露的问题

### 6.1 本轮不能证明什么（至少两项，按要求）

#### ① J3-3 修复（`d52293e` 拒写文案 + `artifact_path_missing` 提示 + 去重）的端到端效果仍未验证

- **未验证链路：** `403 → 文案三要素 → 去重（每任务只喊一次）→ bot 改做法（转交/改提交/停重试）→ 不刷屏（≤3 次）` 全链条在本轮零样本。`prereg §9` 第一条的“不触发写路径”分支被命中，故 V1–V4 均记未观察，V3/V6 的去重/限重试亦未被锻炼。
- **不能得出的结论：**
  - 不能据本轮判定“`d52293e` 的文案可执行/不可执行”（V2 未观察）。
  - 不能据本轮判定“去重有效/失效”（V3 未观察，F-A 未触发）。
  - 不能据本轮判定“bot 会/不会读文案并改变做法”（V4 未观察，F-B 未触发）。
- **证据边界：** 8 份快照 + 0 字节日志 + 0 条 `artifact_path_missing` 共同构成“未触发”的强证据；但“未触发”≠“修复无效”，仅能记为“该场景在此模型/此频道布局下不触发写路径”。

#### ② 批准后缺少分解检查点这一现象的证据边界（一轮观察 ≠ 稳定复现）

- **本轮观察到的现象：**
  - 批准（06:23:19）→ 唯一回合（06:23:26 `msg-21`）产出归 `JOURN-3` → 此后 42 min（至 07:06 停表）`JOURN-8` 无任何新 `messages`、`actions`、`watchdogs`；`scheduler` 的 `task-follow-up:journ-8` 保持 `next_run=07:23:19`（小时级），无 5/15/30 min 级的分解重投；`ScheduleNoTraceRetry` 因“回合留痕”而不排。
  - `0624..0706/tasks.json` 的 `JOURN-8` 保持 `running/breakdown`，`ledger` 无新增；`0706` 仍 `stalled_since=06:33:35` 但无后续唤醒。
- **证据边界：**
  - 单轮、单模型（`crouton-alpha-free`）、单频道布局（`cos__human` 双任务叠加）的观测，不能外推为“批准后分解检查点缺失是稳定缺陷”。`observations.md` 亦提醒“这不是一次分心，而是批准后没有分解检查点”的定性需以多轮复现校核。
  - 反事实未测：若 `JOURN-3` 不存在或 `JOURN-8` 独占频道，是否仍不分解；若模型为其他或改写计划措辞是否触发 lane 创建——本轮均未覆盖。
  - 调度侧 `07:23:19` 的 `task_follow_up` 是否会在停表后才触发分解，超出 45 min 窗口未观测到，需更长时窗或强制唤醒对照。

### 6.2 顺带暴露的问题（不计入 V/F，但值得记录）

1. **“回合有产出但非本任务要的产出”对调度不可见。**
   - 06:23 那轮因产出 `JOURN-3` 相关内容而被视为“有痕”，`ScheduleNoTraceRetry` 不排；`task_stalled`（06:33:35）仅发可见消息，无自动唤醒；`task_follow_up` 小时级。批准 → 分解之间出现“有痕即不重试”的盲区。

2. **非阻塞 `interview` 的提醒静默。**
   - `request-16`（`kind=interview, blocking=None`）的 `request-follow-up` 在 06:52:15 仅产生 `watchdogs` + `actions` 的 `watchdog_alert`，聊天 0 条（`0701` 的 `messages 18` 与 `0700` 相同）。与 Journey 5 的 `request-13` 同型，属于现行 `processRequestJob` 仅对 `blocking` 请求投递可见消息的接线选择。对人类操作员而言，`request_waiting` 长时间静默。

3. **单频道多任务对 cos 注意力的抢占。**
   - 本轮 Acme 场景被 onboarding 遗留的管道基线任务（`JOURN-3`）在同一 `cos__human` 频道内抢占首轮与批准后首回合的注意力；取消 `JOURN-3` 后仍未自恢复，提示“同频道多任务”布局本身放大了分解失联的概率。后续轮次宜为场景任务独占频道或显式隔离 onboarding 遗留。

4. **调度时间与停表窗口的错位。**
   - 本轮 45 min 上限（07:06）早于 `task_follow_up`（07:23:19）、`recheck`（07:38:19）、`reminder`（07:53:19）三档跟进，导致“批准后是否终会分解”在窗口内不可证伪。评估该机制需延长窗口或注入确定性短周期分解检查点。

---

## 7. 附录：证据索引（供复核）

### 7.1 快照清单

| 快照 | 时间 | 覆盖 | 关键文件 |
|------|------|------|----------|
| 0622 | 06:22 | 批准前 | `JOURN-8 planning/plan`，`request-12 pending`，`request-16 pending` |
| 0624 | 06:24 | 批准后 +41s | `JOURN-8 running/breakdown`，`request-12 answered`，`request-follow-up:request-12 done` |
| 0627 | 06:27 | 取消前 | 同 0624，`JOURN-3 running` |
| 0631 | 06:31 | 取消后 | `JOURN-3 canceled`，`task-follow-up:journ-3 done` |
| 0650 | 06:50 | task_stalled 后 | `msg-23 task_stalled`，`watchdogs 0` |
| 0700 | 07:00 | request-16 跟进前 | `scheduler request-follow-up:request-16 next_run 06:52:00` 待触发 |
| 0701 | 07:01 | request-16 跟进后 | `watchdog-1 + action-25 watchdog_alert @06:52:15`，`messages 仍 18` |
| 0706 | 07:06 | 停表 | 同 0701，`JOURN-8 next_run 07:23:19` 未到 |

### 7.2 关键字段速查

- `tasks.json`：`JOURN-8.id / channel=cos__human / owner=cos / created_by=human / lifecycle / stage / parent_issue_id / stalled_since`
- `broker-state.json`：`requests[].kind/blocking/dedupe_key/status`，`messages[].kind/body`，`watchdogs[]`，`actions[].kind/created_at/related_id`，`scheduler[].slug/kind/next_run/last_run/status/status_reason`
- `scheduler.json`：`task-follow-up:cos--human:journ-8`（`kind=task_follow_up, next_run=07:23:19`），`request-follow-up:cos--human:request-16`（`last_run=06:52:15, next_run=07:22:15`）
- `broker-log-slice.txt`：逐份 0 字节，`grep -i "write-task|wiki|refused|403"` 0 命中

---

*判分人：独立 reviewer（子代理，只读判分）*
*复核建议：下一轮为 `JOURN-8` 独占频道、移除 onboarding 遗留；或为 `task_follow_up` 增加批准后 2–5 min 的分解检查点对照组；并延长观察窗至 ≥90 min 以覆盖 07:23 的跟进唤醒。*
