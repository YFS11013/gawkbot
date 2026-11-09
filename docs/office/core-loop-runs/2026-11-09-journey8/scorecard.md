# Journey 8 独立评分卡（2026-11-09）

> 独立判分者依据 `prereg.md` §4–§7，回到 `artifacts/`、`snapshots/`、`broker.log` 原始证据逐条判定，不采信观测记录的结论化表述。隔离办公室 `journey8`（broker :7910 / web :7911），被测提交 `5a026e7`（`a lane is created with its write scope, or it is refused`），蓝图 `ai-revops`（cos/revops/analyst/ae/reviewer），父 Issue `JOURN-6`（`cos__human`、owner=cos、created_by=human、无 `definition`）。

判分基准：`docs/office/core-loop-runs/2026-11-09-journey8/prereg.md`（commit `a4ff236`），观测记录 `observations.md` 仅作时间线索，所有“通过/未通过/未观察”均以下列文件为证据锚点：

- `artifacts/tasks-dump.json`、`artifacts/requests-dump.json`、`artifacts/messages-dump.json`
- `artifacts/broker.log`、`artifacts/broker-log-filtered.txt`、`artifacts/write-task-log.txt`
- `artifacts/wiki-commits.txt`、`artifacts/wiki-tree.txt`
- `snapshots/{1730,1731,1734,1802}/broker-state.json`、`requests.json`、`broker-log-slice.txt`

## 0. 判定总览

| 验证项 | 结论 | 一句话证据 |
|---|---|---|
| **V1** 每条 lane 都带写范围 | **通过（带保留）** | 6 条子 Issue 在 `snapshots/1731/broker-state.json`→`snapshots/1802/broker-state.json` 与 `artifacts/tasks-dump.json` 中均含 `definition.artifact_path`，无 `artifact_not_required`（见 `artifacts/tasks-dump.json` `JOURN-15/17/19/21/23/25` 的 `definition`），但 `JOURN-23/25` 同路径、`JOURN-15/17` 同主题异路径，属规划缺陷而非守卫绕过 |
| **V2** 共享 wiki 真有交付 | **通过（部分偏离）** | `artifacts/broker-log-filtered.txt` 09:31:40–09:34:35（CST 17:31:40–17:34:35）9 条 `write-task` 成功，`artifacts/wiki-tree.txt` 列出 `team/reviews/acme-renewal-quote.md`、`team/reviews/acme-renewal-qa.md`、`team/emails/acme-followup-48h.md` 等，`artifacts/wiki-commits.txt` 对应提交作者为 lane owner（analyst/ae/reviewer/cos），但 `JOURN-17` 声明路径未落地，实际 `artifact=ra_4904a7a50a981735` 落 visual artifact |
| **V3** 守卫是提示不是墙 | **未观察** | `artifacts/broker.log` 全程无 `no write scope`/`artifact_path_missing` 的 create 拒绝（09:31:20–09:34 前 0 条），4 次 403 均为“声明了但路径不匹配”（`artifacts/broker-log-filtered.txt` `task declares artifact_path "...", not "..."`），规划者一次未撞 `no write scope` 守卫，故无法判定“撞后改做法” |
| **V4** 不回归 | **通过（带计量偏差）** | 计划卡仅 1 张 `request-9`（`snapshots/1731/requests.json` / `artifacts/requests-dump.json` 09:30:36Z answered 09:31:01Z），`messages-dump.json` 122 条中 0 条 `task_not_decomposed`/`watchdog_alert`，`snapshots/1802/broker-state.json` 的 `plan-decomposition-check:cos--human:task:journ-6` 已 `status=done` 且带 `status_reason="decomposed into 1 sub-issue(s)"`（`last_run 09:36:02Z`），但计数写 1 而实际拆出 6，为计量错误 |
| **V5** 反向臂探针 | **通过** | 停表后探针 `JOURN-127`（`artifacts/tasks-dump.json` `created_by=human` 无 `definition`）在 `artifacts/broker.log` 18:02:48（UTC 10:02:48Z）连续 3 次 403，文案 `@human must set definition.artifact_path ... retrying this call cannot succeed` 点名 creator，`artifacts/messages-dump.json` `msg-129` 10:02:48Z 仅投 1 条 `artifact_path_missing`（`@human/@ae`），后续 2 次重试未新增 |
| **V6** 不刷屏 | **通过（未锻炼）** | `artifacts/broker.log` 中因写范围被拒的 `create` 为 0（`observations.md` §3 与日志一致），≤5 阈值内；但与 V3 同因：无 create 拒绝故“不刷屏”未经压力检验 |

| 硬失败 | 结论 | 证据 |
|---|---|---|
| **F-A** 同 lane 连续 ≥3 次同内容被拒成墙 | **未触发** | `artifacts/broker.log` 无 create 拒绝；`artifacts/tasks-dump.json` 6 条 lane 各一次创建即带 `artifact_path` |
| **F-B** 路径敷衍或 `artifact_not_required` 误用 | **未触发（临界）** | `artifacts/tasks-dump.json` 6 条 `artifact_path` 均为 `team/reviews/…`、`team/emails/…`、`team/research/…` 等有语义路径，非 `team/tasks/x.md` 敷衍值；全程 0 条 `artifact_not_required`，但 `JOURN-17` 声明后未写 declared 路径而走 `ra_4904a7a50a981735` 属实质未兑现 |
| **F-C** 计划卡缺席 | **未触发** | `snapshots/1731/requests.json`、`1802/requests.json`、`artifacts/requests-dump.json` 均有 `request-9` approval answered |
| **F-D** 拆 lane 成功后 60 分内 `team/**` 仍 0 份 | **未触发** | `artifacts/write-task-log.txt` 09:31:40–09:34:35 已有 9 次成功；`artifacts/wiki-commits.txt` 09:31:40 起连续落地 `team/reviews/…`、`team/emails/…`；`artifacts/wiki-tree.txt` 终态 `team/**` 7 个 `write-task` 产物 + 45 个 onboarding/entity/skill 文件 |
| **F-E** 批准后 10 分内既无子任务也无 `task_not_decomposed` | **未触发** | 批准 `request-9` 09:31:01Z 后，`snapshots/1731/broker-state.json`（约 09:31:2xZ）已出现 6 条子任务（`JOURN-15` defined_at 09:31:20Z 等）；`messages-dump.json` 无 `task_not_decomposed`，分解检查已调度 |

**环境清理单列**：`ai-revops-1..4` 在 09:30:19Z 由 system 批量 `canceled`（`artifacts/tasks-dump.json` 四条 `status=canceled` / `messages-dump.json` msg-2..5），不计场景干预；停表后反向臂探针 `JOURN-127` 10:03:15Z `canceled`（`artifacts/requests-dump.json` `request-133 canceled`）不计入场景。

---

## 1. V1–V6 逐条判定（带原始证据）

### V1 — 每条 lane 都带写范围

**判定：通过（带保留）**

- **数量与存在性**：`snapshots/1731/broker-state.json` 的 `tasks` 在 `JOURN-6`（无 `definition`）之下已列出 6 条子 Issue，逐条均含 `definition.artifact_path`：

  ```json
  JOURN-15  definition.artifact_path="team/reviews/acme-renewal-quote.md"  defined_at 2026-11-09T09:31:20Z
  JOURN-17  definition.artifact_path="team/research/acme-renewal-quote.md" defined_at 2026-11-09T09:31:21Z
  JOURN-19  definition.artifact_path="team/emails/acme-followup-48h.md"    defined_at 2026-11-09T09:31:25Z
  JOURN-21  definition.artifact_path="team/emails/acme-renewal-followup.md" defined_at 2026-11-09T09:31:25Z
  JOURN-23  definition.artifact_path="team/reviews/acme-renewal-qa.md"      defined_at 2026-11-09T09:31:29Z
  JOURN-25  definition.artifact_path="team/reviews/acme-renewal-qa.md"      defined_at 2026-11-09T09:31:30Z
  ```
  证据文件：`snapshots/1731/broker-state.json`（tasks 11 条含上述 6 条）、`snapshots/1734/broker-state.json`（tasks 16 条）与终态 `artifacts/tasks-dump.json`（同 6 条 `definition` 字段逐字一致）、`snapshots/1802/broker-state.json`（`JOURN-15/17/19/21/23/25` 仍均含 `artifact_path`）交叉复核一致。

- **敷衍路径检查**：6 条路径均非 `team/tasks/x.md` 类无语义敷衍值，分别为 `team/reviews/acme-*.md`、`team/emails/acme-*.md`、`team/research/acme-*.md`，均含业务关键字（`reviews`/`emails`/`research` + `acme-renewal`），满足“有意义”。

- **artifact_not_required 误用检查**：`artifacts/tasks-dump.json` 全量 `grep artifact_not_required` 0 命中；`snapshots/*/broker-state.json` 同 0 命中。所有非 `canceled` 任务均有 `artifact_path` 且无 `artifact_not_required_reason`，未出现“声明不产出而实际有交付物”的误用。

- **保留项（不是否定 V1，但影响可信度）**：
  1. `JOURN-23` 与 `JOURN-25` `artifact_path` 完全重复 `team/reviews/acme-renewal-qa.md`（`artifacts/tasks-dump.json` 两条 `definition` 逐字相同），同路径竞写；
  2. A 轨 `JOURN-15`（`team/reviews/acme-renewal-quote.md`）与 `JOURN-17`（`team/research/acme-renewal-quote.md`）同题异路径、B 轨 `JOURN-19` 与 `JOURN-21` 同题异文件名，标题去重未生效（见 `artifacts/tasks-dump.json` `JOURN-15 title="JOURN-6-A: Acme 一页报价说明初稿"` vs `JOURN-17 title="子任务 A — 一页报价说明初稿"`）；
  3. 上述重复不构成 F-B，但使 V1 的“每条 lane 有范围”形式通过、语义上冗余。

### V2 — 共享 wiki 真的有交付

**判定：通过（但 `JOURN-17` 为伪兑现，需单列）**

- **write-task 成功 ≥1 条**：`artifacts/broker-log-filtered.txt` 与 `artifacts/broker.log` 自 17:31:40 起连续记录成功，且与 `artifacts/write-task-log.txt` 逐行一致：

  ```
  2026/11/09 17:31:40 wiki write-task: task=JOURN-15 slug=analyst path=team/reviews/acme-renewal-quote.md bytes=570
  2026/11/09 17:31:42 wiki write-task: task=JOURN-21 slug=ae path=team/emails/acme-renewal-followup.md bytes=694
  2026/11/09 17:31:44 wiki write-task: task=JOURN-19 slug=ae path=team/emails/acme-followup-48h.md bytes=1957
  2026/11/09 17:32:10 wiki write-task: task=JOURN-41 slug=cos path=team/reviews/JOURN-15.md bytes=5102
  2026/11/09 17:32:14 wiki write-task: task=JOURN-23 slug=reviewer path=team/reviews/acme-renewal-qa.md bytes=2509
  2026/11/09 17:32:17 wiki write-task: task=JOURN-47 slug=cos path=team/reviews/JOURN-21.md bytes=5149
  2026/11/09 17:33:49 wiki write-task: task=JOURN-88 slug=cos path=team/reviews/JOURN-23.md bytes=8294
  2026/11/09 17:34:08 wiki write-task: task=JOURN-100 slug=cos path=team/reviews/JOURN-19.md bytes=4819
  2026/11/09 17:34:35 wiki write-task: task=JOURN-105 slug=cos path=team/reviews/JOURN-17.md bytes=6925
  ```
  小计 9 条成功（终态 `artifacts/broker-log-filtered.txt` 统计；`observations.md` §2 所述 1734 快照时 12 成功含时段口径差异，但最低 9 条已满足“≥1”）。

- **team/** 确有 markdown**：`artifacts/wiki-tree.txt` 列出
  ```
  team/emails/acme-followup-48h.md
  team/emails/acme-renewal-followup.md
  team/reviews/acme-renewal-quote.md
  team/reviews/acme-renewal-qa.md
  team/reviews/JOURN-15.md
  team/reviews/JOURN-17.md
  team/reviews/JOURN-19.md
  team/reviews/JOURN-21.md
  team/reviews/JOURN-23.md
  ```
  至少 7 个 `write-task` 产物在 `team/` 顶级路径，且非 `agents/*/notebook/` 私有目录。

- **commit 作者即 lane owner**：`artifacts/wiki-commits.txt` 中对应提交的 `author` 字段与 `tasks-dump.json` 的 `owner` 一一对应（证据行）：
  ```
  feccaeaf 2026-11-09 17:31:40 +0800 analyst    JOURN-6-A: Acme 一页报价说明初稿（JOURN-15）        → team/reviews/acme-renewal-quote.md  owner=analyst
  430709df 2026-11-09 17:31:43 +0800 ae         JOURN-6-B: Acme 48小时跟进邮件草稿（未发送）       → team/emails/acme-followup-48h.md / acme-renewal-followup.md owner=ae
  04e535dc 2026-11-09 17:32:14 +0800 reviewer   JOURN-23 Acme 交付物质检报告                     → team/reviews/acme-renewal-qa.md owner=reviewer
  8fb7ff3d 2026-11-09 17:32:10 +0800 cos        审阅 JOURN-15 一页报价说明初稿                    → team/reviews/JOURN-15.md owner=cos
  2efa757d 2026-11-09 17:32:17 +0800 cos        JOURN-47：独立审阅 JOURN-21 交付物               → team/reviews/JOURN-21.md owner=cos
  f0b6335f 2026-11-09 17:34:08 +0800 cos        审阅 JOURN-19：…                                → team/reviews/JOURN-19.md owner=cos
  2899fd01 2026-11-09 17:34:35 +0800 cos        审阅：JOURN-17 一页报价说明初稿（JOURN-105）      → team/reviews/JOURN-17.md owner=cos
  ```
  符合“`wiki log` 里对应 commit 的作者是场景 lane 的 owner”。

- **JOURN-17 的例外（V2 的反例）**：`artifacts/tasks-dump.json` 中 `JOURN-17` 的 `artifact` 字段为 `ra_4904a7a50a981735`（visual artifact），而非 `team/research/acme-renewal-quote.md`；`artifacts/wiki-tree.txt` 全量检索 `team/research/` 0 命中该路径；`artifacts/wiki-commits.txt` 在 17:31:38 仅有 `analyst artifact: create visual artifact ra_4904a7a50a981735` 与 `artifacts/acme.md`（`attached_artifact_id: ra_4904a7a50a981735`）佐证其产物落在 `wiki/visual-artifacts/ra_4904a7a50a981735.html|json`（`wiki-tree.txt` 末两行）。即：**V2 要求中“`JOURN-17` 声明了 `artifact_path` 但未向其写入”，其 `artifact` 字段只是引用了 visual artifact**。按 prereg V2 的“区分 write-task 成功与 artifact 字段只是引用”要求，此条不计入 `write-task` 成功；但其余 5 条 lane 与 5 条独立审阅任务的 `write-task` 成功已独立满足 V2。

### V3 — 守卫是提示，不是墙

**判定：未观察**

- **定义**： prereg §4 V3 备注与 `observations.md` §3 已明确“撞守卫指 `no write scope` 拒绝”。本轮的 4 次 403 均非此类：`artifacts/broker-log-filtered.txt` 前 4 条拒绝的 `reason` 字段均为 `task JOURN-xx declares artifact_path "...", not "..."`（路径不匹配），而非 `artifact path not declared`/`no write scope`。
- **计数**：`artifacts/broker.log` 全文检索 `artifact path not declared` / `no write scope` 在探针前（09:31:00–10:02:00Z）0 条；`tasks-dump.json` 与 `snapshots/1731/broker-state.json` 中所有子 Issue 均已带 `definition.artifact_path`，与“全程 `no write scope` 计数 0”（`observations.md` §3）复核一致。
- **改做法缺失**：由于一次未撞守卫，自然不存在“撞后补路径或声明不产出”的改做法轨迹；同时也无“原样重试同一 create ≥2 次”的墙化证据。
- **结论写法**：不能记为“改做法通过”，应记为“**未观察到撞后改做法**”——正如 prereg §4 V3 的“若为 0，要写清这属于‘未观察到撞后改做法’，不能算‘改做法通过’”。本轮规划者（`cos` 在 `snapshots/1731` 已直接为 6 条 lane 钉好路径）属于“一次做对”，守卫文案的可读性与引导性未经检验。

### V4 — 不回归

**判定：通过（分解检查计数错误需修）**

- **计划卡自动出现且只有一张**：
  - `snapshots/1730/requests.json` 0 张 → `snapshots/1731/requests.json` 1 张：`request-9 kind=approval title="Start work on Journey 8: Acme 续约报价与跟进?" channel=cos__human created_at 2026-11-09T09:30:36Z answered 09:31:01Z`；
  - 终态 `artifacts/requests-dump.json` 共 3 张中，仅 `request-9` 为计划卡，其余为 `request-117`（promotion，09:34:53Z pending）与 `request-133`（探针，10:03:01Z canceled），后两者 channel/title 均非计划卡；
  - `snapshots/1734/requests.json` 与 `snapshots/1802/requests.json` 亦仅保留 `request-9`（answered）于场景时段内，无重复计划卡。结论：**计划卡按时出现且唯一**，J3-1 未回归。

- **`task_not_decomposed` 0 条**：`artifacts/messages-dump.json`（122 条）全量检索 `task_not_decomposed` 0 命中；`snapshots/1730–1802/broker-state.json` 的 `messages`（1730:6 条 / 1731:25 条 / 1734:101 条 / 1802:111 条）同 0 命中。分解监督未报警。

- **`watchdog_alert` 0 条**：同上检索 `watchdog_alert` 0 命中。虽有两条系统 `gone quiet` 提示（`artifacts/messages-dump.json` `msg-125` 09:44:44Z `cos__reviewer` / `msg-126` 09:46:44Z `cos__human`），但 `kind` 并非 `watchdog_alert`，且 prereg 的 `watchdog_alert` 指代强告警，故按判据为 0。

- **拆 lane 后 `plan_decomposition_check` 正常收尾带 `status_reason`**：
  - `snapshots/1731/broker-state.json` 的 `scheduler` 中 `plan-decomposition-check:cos--human:task:journ-6` 为 `status=scheduled due_at 09:36:01Z enabled=false`；
  - `snapshots/1734` 同 `scheduled`；
  - `snapshots/1802/broker-state.json` 已转为 `status=done last_run 09:36:02Z last_run_status=done status_reason="decomposed into 1 sub-issue(s)" enabled=false`。
  - 判定：**已收尾且带 `status_reason`**，满足 prereg 表述的“正常收尾带 `status_reason`”。**但计量错误**：`status_reason` 写 1，而实际 `JOURN-6` 拆出 6 条（`JOURN-15/17/19/21/23/25`），`tasks-dump.json` 与 `broker-state.json` 均可复核为 6，分解计数应为 6 而非 1，属 `scheduler` 文案缺陷（见“待修缺陷”）。

### V5 — 反向臂探针（停表后执行，不计场景干预）

**判定：通过**

- **停表时点**：`observations.md` §4 记录 10:02:07Z 连续 15 分无新消息停表（最后消息 09:46:44Z watchdog 提示），与 `snapshots/1802`（09:46 后无新增 lane）及 `artifacts/messages-dump.json` 最后场景消息 `msg-126` 09:46:44Z 一致。

- **探针任务**：`artifacts/tasks-dump.json` `JOURN-127` 条目 `id=JOURN-127 title="Probe: undeclared write scope (reverse arm)" owner=ae channel=ae__cos created_by=human definition=null`（human 豁免新守卫，故可建无 `definition` 任务）；`artifacts/requests-dump.json` `request-133` `channel=ae__human` 对应其计划卡 10:03:01Z。

- **403 文案点名 creator 且声明无法重试**：`artifacts/broker.log` 与 `artifacts/broker-log-filtered.txt` 在 18:02:48（= UTC 10:02:48Z）连续 3 行：
  ```
  wiki write-task refused: task=JOURN-127 slug=ae path=team/research/probe-undeclared.md: artifact path not declared: JOURN-127 declares no artifact_path, so no wiki path is authorized for it. @human must set definition.artifact_path on the task (or route this deliverable through a human request); retrying this call cannot succeed
  ```
  字段验证：`@human` 点名 creator、`declares no artifact_path` 点明原因、`retrying this call cannot succeed` 声明原样重试无效——完全命中 prereg §4 V5 的三要素。

- **只投 1 条 `artifact_path_missing`**：`artifacts/messages-dump.json` 在 10:02:48Z 仅 `msg-129` 一条：
  ```
  @human: JOURN-127 has no definition.artifact_path, so @ae's write to team/research/probe-undeclared.md was refused. Pin the path on the task (or have @ae deliver through a human request) — until then the lane can only leave notes in a notebook. Owner: @ae.  [channel ae__cos] [tagged human, ae, cos]
  ```
  与 `observations.md` §4 “频道里出现 1 条 `artifact_path_missing` 提示（@human + @ae 都被点名），此后两次重试没有新增”逐字对齐；`artifacts/broker.log` 虽有 3 次 403，但 `messages-dump.json` 在 10:02:48–10:03:15 间无新增同类提示，满足“仍只投 1 条”。

### V6 — 不刷屏

**判定：通过（未锻炼）**

- **阈值**：`artifacts/broker.log` 中因写范围被拒的 `create`（`no write scope` 分支）在场景时段内 0 条；`artifacts/broker-log-filtered.txt` 同 0 条（4 条拒绝均为 write-task 路径不匹配，非 create）。≤5 阈值内。
- **补充**：由于 V3 的 0 撞守卫，本轮“不刷屏”亦未经历压力：不存在重试风暴，故 V6 的通过仅表示“无刷屏事实”，不等同“在重试风暴下仍不刷屏”。需后续以逼迫守卫的用例二次检验节流。

---

## 2. F-A..F-E 硬失败逐条判定

### F-A（回退触发）：同一 lane 因写范围被拒 ≥3 次且 create 内容不变

**未触发**

- `artifacts/broker.log` / `broker-log-filtered.txt` 在场景时段（09:31–09:46Z）无任何 `no write scope` 的 create 拒绝；`artifacts/tasks-dump.json` 的 6 条 lane 均一次创建即带 `artifact_path`（`defined_at` 09:31:20–09:31:30Z），无重试链。
- 反向臂的 3 次拒绝为 `write-task` 拒绝（同一 `path=team/research/probe-undeclared.md` 连续 3 次），但其 `task=JOURN-127` 的 `definition` 始终为 `null`，且拒绝的是 `write-task` 而非 `create`，不落此条计数；即便计入，文案已明确“retrying cannot succeed”且频道仅提示一次，未构成“原样重试仍被拒成墙”的产品失败——但按 prereg 口径，F-A 仅统计 `create` 的 `no write scope` 拒绝，故本轮 0 次。

### F-B（回退触发）：所有 lane 的 `artifact_path` 都是敷衍值，或声明 `artifact_not_required` 而实际有交付物

**未触发（临界，需记异常）**

- **敷衍值检查**：`artifacts/tasks-dump.json` 6 条 `artifact_path` 见 V1 列举，均为 `team/{reviews,emails,research}/acme-*.md`，含领域与业务语义，非 `team/tasks/x.md` 类敷衍值。
- **误用检查**：全量 0 条 `artifact_not_required`（见 V1），故“声明不产出而实际有交付物”的第二分支不成立。
- **临界异常（不触发回退，但暴露载体分流）**：`JOURN-17` 声明 `team/research/acme-renewal-quote.md` 却以 `artifact=ra_4904a7a50a981735` 交付（`artifacts/tasks-dump.json` `JOURN-17 artifact=ra_...`），`wiki-tree.txt` 无 `team/research/acme-renewal-quote.md`，`artifacts/acme.md` 头部 `attached_artifact_id: ra_4904a7a50a981735` 与 `wiki/visual-artifacts/ra_4904a7a50a981735.html|json` 为其真载体。按 prereg 文字“所有 lane 都是敷衍值”不成立，但该 lane 形式上有路径、实质未写共享 wiki，属 F-B 担忧的同质问题在 `visual artifact` 维度复现，已单列为缺陷（见 §4）。

### F-C：计划卡缺席（J3-1 回归）

**未触发**

- `snapshots/1731/requests.json` 与 `artifacts/requests-dump.json` 的 `request-9`（`kind=approval channel=cos__human title="Start work on Journey 8: …" created_at 09:30:36Z`）即计划卡，且在 09:31:01Z 被 `approve`（`answered.choice_id=approve`），与 `snapshots/1730`（0 张）到 `1731`（1 张）的增量一致。

### F-D：拆 lane 成功但 60 分钟内 `team/**` 仍为 0 份

**未触发**

- 拆 lane 完成时点：最晚子任务 `JOURN-25` `created_at 09:31:30Z`；首个 `write-task` 成功 09:31:40Z（`JOURN-15`），末个场景内成功 09:34:35Z（`JOURN-105`），间隔 <5 分钟。
- `artifacts/wiki-commits.txt` 自 17:31:40 起连续 `team/reviews/*`、`team/emails/*` 提交；`artifacts/wiki-tree.txt` 终态 `team/**` 52 文件（`team/reviews` 7 产物 + `team/emails` 2 产物 + 其余 onboarding/entity/playbook/skill）。
- 故 60 分阈值内 `team/**` 远非 0。

### F-E：批准后 10 分钟内既无子任务也无 `task_not_decomposed`

**未触发**

- 批准 09:31:01Z → 首条子任务 `JOURN-15` 09:31:20Z（`defined_at`），差 19 秒；`snapshots/1731/broker-state.json`（09:31 附近快照）已含 6 条子任务；`artifacts/messages-dump.json` 无 `task_not_decomposed`，分解检查已调度（`plan-decomposition-check:…:journ-6 due_at 09:36:01Z`，见 V4）。

**回退结论**：F-A 与 F-B 均未触发，按 prereg §5 “触发 F-A 或 F-B → 切 B” 的写死判据，**本轮不触发回退**，§9.15 选 A 保持。

---

## 3. 本轮不能证明什么

以下 4 项为 prereg 要求必覆盖的“未观察”边界，均与 `artifacts/snapshots` 对齐后仍证据不足，须在后续轮次单独立项检验。

### 3.1 守卫文案在真机是否可读（本轮规划者一次没撞）

- **事实**：`artifacts/broker.log` / `broker-log-filtered.txt` 在场景时段内 `no write scope` 拒绝 0 条；`artifacts/tasks-dump.json` 显示 `cos` 在 09:31:20–09:31:30Z 的 6 次 `team_task action=create` 均一次带 `definition.artifact_path`（标题在观测与 `requests-dump.json` 均载有“**建 lane 时当场钉写范围**及其原因”的系统提示语）。
- **不能证明**：prereg 推测的“规划者读取拒因当场补写范围”的教育链未被触发，故**无法证明** `internal/team/broker_tasks_subissue_guards.go` 中 `no write scope` 拒绝的文案（是否点名 creator、是否列允许根、是否声明“原样重试仍会被拒”）在真实模型视野里可读、易懂、能改做法。反向臂虽验证了 `write-task` 的 `artifact path not declared` 文案（18:02:48 3 次 403 均含 `@human must set definition.artifact_path … retrying cannot succeed`，见 `artifacts/broker-log-filtered.txt` 末 3 行），但那是另一条分支（`nudgeUndeclaredArtifactPath`）且发生在 `write-task` 而非 `create`，不能外推至 `create` 守卫的教育效果。
- **缺什么**：一次“故意不带 `artifact_scope` 的 create 被拒 → 模型在同一回合补 `artifact_scope` 重试”的完整 trace；当前缺该 trace 的 `broker.log` 拒绝行、`messages-dump.json` 中的守卫提示、以及重试 `create` 的 `definition` 差异。

### 3.2 重复审批卡 `request-117` 与停表的关系

- **事实**：`artifacts/requests-dump.json` 中 `request-117 kind=approval status=pending title="Promote to the team wiki: JOURN-100 审阅 JOURN-19 结论（补记）" created_at 09:34:53Z channel=cos__human` 与 `artifacts/messages-dump.json` `msg-118` 09:34:53Z 的 promotion 卡一致；`snapshots/1802/requests.json` 仍为 `pending`；`artifacts/messages-dump.json` 的下一条 `msg-119..124` 为 `cos` 的审阅播报，`cos` 最后一条自发消息为 `msg-124` 09:35:57Z，之后仅有 `msg-125/126` 两条 watchdog `gone quiet`（09:44:44Z、09:46:44Z），停表 10:02:07Z。
- **不能证明**：
  1. `request-117` 是否“把 cos 的回合 park 住”导致 `JOURN-6` 永不收尾——`artifacts/tasks-dump.json` 显示 `JOURN-6 pipeline_stage=breakdown lifecycle_state=running updated_at 09:31:41Z` 后无 `ledger` 更新，`JOURN-25` 同 `running`，但 `messages-dump.json` 与 `broker-state.json` 均未记录 `request-117` 对 `JOURN-6` 的阻塞边（如 `blocked_by` 字段或取消流水），只能观察到时序相关，不能证因果；
  2. 停表判据“连续 15 分无新消息”（prereg §7）是否应将 `pending` 的 promotion 卡计入“有事待办”——当前实现显然将其视为“消息静默”而停表，但 prereg 未定义 `request_follow_up` 与 promotion 的停表豁免，无法判定是“机制按设计停”还是“机制把未决审批当安静”。
- **缺什么**：`scheduler` 的 `request_follow_up` 任务对 `request-117` 是否曾调度（`snapshots/1802/broker-state.json` 的 `scheduler` 列表仅示 kind/label/status，未展开 `request-117` 的 follow-up 实例）、`JOURN-6` 与 `request-117` 的关联字段（`blocked_by`/`promotion` 边）、以及对“停表前存在 pending 审批时是否应延时”的设计决议。

### 3.3 `JOURN-25` 停在 `running` 的机制

- **事实**：`artifacts/tasks-dump.json` `JOURN-25`：
  ```json
  id=JOURN-25 status=in_progress pipeline_stage=breakdown owner=reviewer
  created_at 09:31:30Z ledger=[{agent=reviewer at 09:32:00Z "子任务 C 已进入等待态：待 A/B 落地后逐项核对…"}] 
  definition.artifact_path=team/reviews/acme-renewal-qa.md
  ```
  同路径的 `JOURN-23` 已在 09:33:52Z 前 `done`（`artifacts/tasks-dump.json` `JOURN-23 status=done artifact=team/reviews/acme-renewal-qa.md`，`wiki-commits.txt` `04e535dc reviewer JOURN-23 Acme 交付物质检报告` 17:32:14），且 `messages-dump.json` `msg-113` 09:34:11Z 明确播报“A/B 已放行，QC 报告已在 `team/reviews/acme-renewal-quote.md` 中”。
  `msg-125` 09:44:44Z `cos__reviewer` 对 `JOURN-25` 触发 `gone quiet`，10:02 终态仍 `in_progress`。
- **不能证明**：
  - `JOURN-25` 是“等待人工确认”还是“调度器丢失后续唤醒”——`snapshots/1802/broker-state.json` 的 `scheduler` 未见针对 `JOURN-25` 的单任务重放/重试 job，仅有 `plan-decomposition-check:…:journ-6` 已 `done`；
  - 同路径 `JOURN-23/25` 共享 `team/reviews/acme-renewal-qa.md` 是否导致后者被视为“已交付”而提前 park——`artifacts/broker-log-filtered.txt` 在 17:32:14 对 `JOURN-23` 的 `write-task` 成功之外，再无对 `JOURN-25` 的写尝试，亦无取消/完成记录。
- **缺什么**：`JOURN-25` 的 `scheduler_runs` / `actions` 明细（本轮 `artifacts` 未落 `scheduler_runs.json`，仅快照有空 `scheduler_runs` 列表）、以及“出厂质检”双 lane 的设计意图（是备份还是误拆）的产品说明。

### 3.4 同主题 lane 重复（15/17、19/21、23/25）是否属于规划缺陷

- **事实**：`artifacts/tasks-dump.json` 显示三对同主题：
  - A 轨：`JOURN-15 title="JOURN-6-A: Acme 一页报价说明初稿" goal="产出一页报价说明初稿…" owner=analyst` vs `JOURN-17 title="子任务 A — 一页报价说明初稿" goal="子任务 A — 一页报价说明初稿" owner=analyst`；
  - B 轨：`JOURN-19 title="JOURN-6-B: Acme 48 小时跟进邮件草稿" goal="产出 48 小时跟进邮件草稿，不发送。" owner=ae` vs `JOURN-21 title="子任务 B — 48 小时跟进邮件草稿" goal="子任务 B — 48 小时跟进邮件草稿" owner=ae`；
  - C 轨：`JOURN-23/25` 同 `team/reviews/acme-renewal-qa.md`（前已述）。
  `observations.md` §3 称“标题措辞不同，兄弟去重按标题相似度，没拦住”——该解释在 artifacts 中得到侧面印证：四条标题仅前缀/标点差异，而 `artifact_path` 亦各不相同（A 轨 `reviews` vs `research`、B 轨 `acme-followup-48h.md` vs `acme-renewal-followup.md`）。
- **不能证明**：
  1. 去重失效是“阈值过严”还是“未对 `goal`/`artifact_path` 做归一化比对”——`broker-state.json` 未落去重判定日志；
  2. 重复是否造成实质成本——虽可见 `JOURN-17` 走 `visual artifact`、`JOURN-25` 永不收尾、`JOURN-23/25` 路径冲突，但 `index/all.md` 与 `playbook` 归档是否因此产生重复学习噪声未在本轮 `artifacts/` 中体现（`wiki-commits.txt` 的 playbook/skill 增强均沿 `JOURN-41/47/88/100/105` 审阅链，而非 lane 去重）。
- **缺什么**：`internal/team/broker_tasks_mutation_service.go` 兄弟去重的相似度算法日志（`task_title_similarity` 阈值命中分）、以及“同 owner + 同 goal 前缀即视为重复”的修订规约。

---

## 4. 待修缺陷（按严重度）

> 严重度按 prereg 语境与 `DESIGN-DECISIONS.md` §9.15 的“共享 wiki 真有交付”目标标定；每条给出“证据行 → 建议修法”。

### P1 — 严重（阻塞闭环或污染交付）

**D1. `request-117` 的 `pending` promotion 把 `cos` 回合 park 住，父任务永不收尾**

- **证据**：
  - `artifacts/requests-dump.json` `request-117 status=pending created_at 09:34:53Z channel=cos__human title="Promote … JOURN-100 …"`；
  - `artifacts/messages-dump.json` `msg-118` 09:34:53Z 推广卡、`msg-120..124` 后 `cos` 再无自发消息（最后 `msg-124` 09:35:57Z），`msg-126` 09:46:44Z `gone quiet on "Journey 8: Acme…"`；
  - `artifacts/tasks-dump.json` `JOURN-6 lifecycle_state=running pipeline_stage=breakdown updated_at 09:31:41Z`（批准后仅一条“已拆子任务”ledger，此后 30 分钟 0 更新）；
  - `snapshots/1802/broker-state.json` `tasks` 中 `JOURN-6` 仍 `in_progress`，`requests` 中 `request-117` 仍 `pending`，`scheduler` 无 `task_follow_up` 对 `JOURN-6` 的重放。
- **影响**：V4 虽判“计划卡唯一、分解检查已收尾”，但外层 Issue 无法 `done`，`team/**` 虽有交付，却无 machine-verifiable 的父任务完成信号，下游依赖生命周期（如看板统计）的任务会被误判为“仍在做”。
- **建议修法**：
  1. 将 `context_promote`（`internal/teammcp/server_tasks.go` 的 `Promote to team wiki`）的审批卡从“阻塞 owner 回合”的同步模型改为“异步归档”：`JOURN-6` 的 `lifecycle_state` 不应等待 notebook 推广的 `answered`，而应在子任务 `done` 后由 deterministic hook 触发 `review_state=approved`；
  2. 或为 `request-117` 增加 `auto-approve after N min if no human veto` 的团队内提升（human 仍可 `reject_with_steer`），避免无人值守的实验室停表被单张推广卡劫持。

**D2. 同主题 lane 去重失效（A/B/C 各两条，C 轨同路径）**

- **证据**：
  - `artifacts/tasks-dump.json` 6 条 `definition` 标题/goal 见 §1 V1 摘录，A 轨两条同为“Acme 一页报价说明初稿”、B 轨两条同为“48 小时跟进邮件草稿”、C 轨两条同 `artifact_path team/reviews/acme-renewal-qa.md`；
  - `snapshots/1731/broker-state.json` 已同时存在 6 条，说明去重在 `action=create` 入口未拦；
  - 后果：`JOURN-17` 实际未写 `team/research/…`（见 D3）、`JOURN-25` 永久 `running`（见 §3.3）、`team/reviews/acme-renewal-qa.md` 被两任务声明导致审计歧义。
- **影响**：同份工作双记账 → 审阅链被迫对 `JOURN-17/19/21` 分别审（`JOURN-100/105`），浪费 token 且使“去重策略”成为 paper-only。
- **建议修法**：
  1. 在 `internal/team/broker_tasks_mutation_service.go` 的兄弟去重中，对 `goal` 做归一化（去 `JOURN-6-` 前缀、全半角/标点、空格）后比 `artifact_path` 的父目录与 slug 词面量，而非仅比 `title` 相似度；
  2. 对新 lane 的 `artifact_path` 增加同父任务内的唯一性校验：同一 `parent_issue_id` 下声明同路径即拒（文案复用 create 守卫的三要素），并在 `prompt_builder.go` 的规划提示中追加“每份交付物仅一 lane”示例。

### P1 — 严重（交付载体分流）

**D3. `JOURN-17` 声明 `team/research/acme-renewal-quote.md` 却以 `visual artifact` 交付，`write-task` 真空**

- **证据**：
  - `artifacts/tasks-dump.json` `JOURN-17 definition.artifact_path="team/research/acme-renewal-quote.md" artifact="ra_4904a7a50a981735"`；
  - `artifacts/broker-log-filtered.txt` 中 `JOURN-17` 无 `write-task: ... path=team/research/...` 成功行（仅 `JOURN-15/19/21/23` 有）；
  - `artifacts/wiki-commits.txt` `7ac99d03 17:31:38 +0800 analyst artifact: create visual artifact ra_4904a7a50a981735`、`artifacts/wiki-tree.txt` 末尾 `wiki/visual-artifacts/ra_4904a7a50a981735.html|json`，`artifacts/acme.md` `attached_artifact_id: ra_4904a7a50a981735`；
  - `artifacts/wiki-tree.txt` `grep team/research` 0 命中。
- **影响**：§9.15 的“共享 wiki 真有交付”被 visual artifact 绕过——`index/all.md` 与 deterministic learning hook 依赖 `team/**` 的 git 提交与 `team/entities/*.facts.jsonl`，`visual artifact` 不进该链，`archivist` 仅为 `JOURN-17` 记一条 `companies/needs-confirmation` fact（`4a2de2be ... analyst fact: companies/needs-confirmation … Produced artifact: ra_4904a7a50a981735`），不产生 `team/research/` 的可检索交付。
- **建议修法**：
  1. 在 `internal/teammcp/server_tasks.go` 的任务完成校验中，将 `artifact` 字段的 `ra_` 前缀视为“未写 `team/**`”的失败态：若 `definition.artifact_path` 非空而 `artifact` 以 `ra_` 起，`status` 不得置 `done`，文案沿用 write-task 403 的 `@human must set …` 三要素；
  2. 或在 `broker_tasks_subissue_guards.go` 对 `team/research/` 类路径增加白名单提示时，同步在 planner prompt 中说明“报价说明不得走 `research` 且不得走 visual artifact，必落 `team/reviews/` 或 `team/emails/`”。

### P2 — 中等（状态机与计量）

**D4. `JOURN-25` 永久 `running`（`breakdown` 态）与 `gone quiet` 无恢复**

- **证据**：见 §3.3（`JOURN-25` 在 `09:32:00Z` 进入等待后无后续 ledger，`snapshots/1802` 仍 `breakdown`，`msg-125` `gone quiet` 后无 `restart`）。
- **建议修法**：为 `pipeline_stage=breakdown` 的 `running` 任务增加与 `task_follow_up` 同源的 `recheck`：若兄弟 `JOURN-23` 已 `done` 且 `team/reviews/acme-renewal-qa.md` 已存在（`wiki-tree.txt` 可判定），`JOURN-25` 应自动 `canceled` 并附 `status_reason="superseded by JOURN-23 (same artifact_path)"`，避免父任务被单条僵尸子任务拖住。

**D5. `plan_decomposition_check` 计数错误（`1 sub-issue(s)` vs 实际 6）**

- **证据**：`snapshots/1802/broker-state.json` `plan-decomposition-check:…:journ-6 status=done status_reason="decomposed into 1 sub-issue(s)"`；`artifacts/tasks-dump.json` 实际 6 条子任务。
- **建议修法**：`scheduler` 的分解检查在统计时改为以 `parent_issue_id=JOURN-6` 的 `tasks` 计数为准，或将 `status_reason` 改为枚举 `decomposed` 而不带数字，避免审核者误判“只拆 1 条算通过”。

**D6. `write-task` 路径不匹配 4 次（声明后再改路径重试）**

- **证据**：`artifacts/broker-log-filtered.txt` 17:31:35 `JOURN-15 team/accounts/acme-renewal-quote-draft.md ≠ team/reviews/acme-renewal-quote.md`、17:31:38 两条 `team/emails/acme-48h-followup.md ≠ {team/emails/acme-renewal-followup.md, team/emails/acme-followup-48h.md}`、17:32:09 `JOURN-23 team/reviews/acme-deliverables-qa.md ≠ team/reviews/acme-renewal-qa.md`；均在 2–5 秒内重试到声明路径成功（17:31:40 `JOURN-15 570B` 等），4 次拒绝文案均为 `declares artifact_path "…", not "…"`。
- **影响**：虽最终均纠正且未触发刷屏，但首写 42% 失败率（4/9+4）表明 planner 的 `artifact_path` 与执行器的写实现不一致，守卫的“防路径不匹配”能力虽有效，却以试错成本兑现。
- **建议修法**：在 `prompt_builder.go` 的写约束提示中，将 `artifact_path` 以代码块精确回显到执行器上下文，并在 `team_task` 的 `definition` 回包里带 `artifact_path` 的逐字提醒，减少执行器自拟 `acme-48h/deliverables-qa` 变体。

### P3 — 轻微（可观测性与文案）

**D7. `team/emails` 产出两份内容一致的邮件稿（`acme-followup-48h.md` 与 `acme-renewal-followup.md`）未合并**

- **证据**：`artifacts/wiki-tree.txt` 两文件并存；`artifacts/messages-dump.json` `msg-113` 09:34:11Z `reviewer` 已提示“存在两份邮件稿内容一致，建议只保留一份”。
- **建议修法**：归档前由 `archivist` 或 `reviewer` 的第二级去重合并为单一 `acme-followup-48h.md`，并将被合并任务的 `artifact` 指向保留文件。

**D8. 反向臂后 `JOURN-127` 的 plan 仍被允许进入 `plan` 态（虽未执行）**

- **证据**：`artifacts/messages-dump.json` `msg-130..136` 显示 `JOURN-127` 在 `write-task` 3 次 403 后仍进入“制定探测计划→出 plan→起 approval 卡 `request-133`”的正常 intake，虽符合设计（只读探针可 plan），但与“`no write scope` 的 lane 应被 create 守卫直接拒”的直观预期形成文案分岔，需在 `prompt_builder.go` 明确“无声明的 lane 连 plan 也不应被创建”。
- **建议修法**：文档化 `JOURN-127` 的 `created_by=human` 豁免（仅反向臂）与机器 lane 的拒绝边界，避免读者将探针的放行误读为守卫失效。

---

## 5. 总评

**§9.15 的机制是否成立？——形式成立，价值未经压力检验。**

- **“lane 建出来就带写范围”在形式上成立**：`cos` 在 09:31:20–09:31:30Z 的 6 次 `team_task action=create` 均一次携带 `definition.artifact_path`（`artifacts/tasks-dump.json` 与 `snapshots/1731/broker-state.json` 的 `defined_at` 链），`artifacts/broker.log` 在场景时段 0 条 `no write scope` 拒绝，说明 plannning 侧的“当场钉范围”提示（`broker_tasks_subissue_guards.go` 与 `prompt_builder.go` 的文案）至少在“不犯错”路径上生效。
- **“共享 wiki 真有交付”在数量上成立、在单 lane 上失真**：`team/**` 在 `artifacts/wiki-tree.txt` 的 7 个 `write-task` 产物与 `artifacts/wiki-commits.txt` 的 9 次成功（含 5 条独立审阅）足以证“有交付”；但 `JOURN-17` 以 `ra_4904a7a50a981735` 分流到 `wiki/visual-artifacts/`（`artifacts/acme.md` 为证），以及 `JOURN-23/25` 同路径声明，使“每条 lane 的交付均在声明路径上可检索”这一更严的命题不成立。
- **守卫的教育价值未被检验**：V3 的 `0 撞` 意味着 `broker_tasks_subissue_guards.go` 的 `no write scope` 拒绝及其“点名 creator、列允许根、声明原样重试仍会被拒”的完整文案在真机上一次未被模型消费；反向臂对 `write-task` 分支的文案验证（18:02:48 3 次 403 + `msg-129` 单条提示）不能外推至 `create` 守卫。 §9.15 要回答的“规划者会不会在被拒后当场补写范围”因从未被拒而**无答案**。

**是否触发回退条件？——未触发。**

- 按 prereg §5 回退判据“F-A 或 F-B 即切 B（接受 notebook + visual artifact 为交付载体）”：F-A 0 次、F-B 未满足“所有 lane 敷衍或 `artifact_not_required` 误用”（虽有 `JOURN-17` 的 visual artifact 分流，但非全量且非 `artifact_not_required`），**本轮不回退**，§9.15 选 A（lane 建即带写范围 → 共享 wiki 交付）保持有效，但需以 D1–D4 的修复作为下一轮的前置：否则下一轮即便逼迫守卫，也会被 promotion 阻塞与僵尸子任务拖住而复现“有交付、无收尾”的同一结局。

**下一轮必做（否则 §9.15 的下一次判分仍为“未观察”）**：
1. 构造一个“故意不带 `artifact_scope` 的 lane”用例（单 lane、标题唯一）以锻炼 `no write scope` 守卫，产出 `broker.log` 的拒绝行与频道 `artifact_path_missing` 提示各 1 条，并验证同 lane 补 `artifact_scope` 后一次通过；
2. 修 D1（promotion 不 park 父任务）与 D4（`JOURN-25` 类型僵尸 lane 自动 `canceled`），使 60 分钟上限内的 `JOURN-6` 能进 `done`；
3. 在 `tasks-dump.json` 的完成校验中将 `artifact=ra_` 视为未兑现，关闭 `JOURN-17` 类分流。

---

## 6. 证据清单（按文件）

| 证据文件 | 关键行/字段 | 用于 |
|---|---|---|
| `artifacts/tasks-dump.json` | `JOURN-15 definition.artifact_path="team/reviews/acme-renewal-quote.md" defined_at 09:31:20Z` 等 6 条；`JOURN-17 artifact="ra_4904a7a50a981735"`；`JOURN-6 / JOURN-25 status=in_progress pipeline_stage=breakdown`；`JOURN-127 definition=null created_by=human` | V1, V2, V4, F-A..F-E, D1–D4 |
| `snapshots/1731/broker-state.json` | `tasks` 11 条含 6 条子 Issue 各带 `definition`；`requests` 1 条 `request-9 answered` | V1, V4, F-C/E |
| `snapshots/1734/broker-state.json` | `tasks` 16 条，同 6 条定义保持 | V1 |
| `snapshots/1802/broker-state.json` | `tasks` 16 条（`JOURN-6/25` 仍 `running`）；`requests` 2 条（`request-9 answered` + `request-117 pending`）；`scheduler` `plan-decomposition-check:…:journ-6 status=done status_reason="decomposed into 1 sub-issue(s)" last_run 09:36:02Z` | V1, V4, D5, §3.2–3.3 |
| `artifacts/broker.log` / `artifacts/broker-log-filtered.txt` | `17:31:35–17:32:09` 4 条 `declares artifact_path "...", not "..."`；`17:31:40–17:34:35` 9 条 `write-task: … bytes=…`；`18:02:48` 3 条 `artifact path not declared … @human must set definition.artifact_path … retrying this call cannot succeed` | V2, V3, V5–V6, D3/D6 |
| `artifacts/write-task-log.txt` | 同 filtered 前 10 行（场景内 9 成功+4 拒绝） | V2, D6 |
| `artifacts/wiki-tree.txt` | `team/emails/acme-followup-48h.md`、`team/emails/acme-renewal-followup.md`、`team/reviews/acme-renewal-quote.md`、`team/reviews/acme-renewal-qa.md`、`team/reviews/JOURN-*.md`、`wiki/visual-artifacts/ra_4904a7a50a981735.*` | V2, D3 |
| `artifacts/wiki-commits.txt` | `feccaeaf 17:31:40 analyst JOURN-6-A …`、`430709df 17:31:43 ae …`、`04e535dc 17:32:14 reviewer …`、`8fb7ff3d 17:32:10 cos …`、`7ac99d03 17:31:38 analyst artifact: create visual artifact ra_4904…` 等 | V2 |
| `artifacts/acme.md` | `attached_artifact_id: ra_4904a7a50a981735` / `visual-artifact:ra_4904a7a50a981735` | V2, D3 |
| `artifacts/requests-dump.json` | `request-9 answered 09:30:36–09:31:01Z`、`request-117 pending 09:34:53Z`、`request-133 canceled 10:03:01Z` | V4, §3.2, D1 |
| `artifacts/messages-dump.json` | `msg-129 10:02:48Z ae__cos artifact_path_missing`（仅 1 条）、`msg-118 09:34:53Z request-117`、`msg-125 09:44:44Z gone quiet JOURN-25`、`msg-126 09:46:44Z gone quiet JOURN-6`、`msg-113 09:34:11Z QC 播报`、`122 条中 0 条 task_not_decomposed/watchdog_alert` | V3–V6, §3 |
| `snapshots/1730/requests.json` | `[]` | V4 F-C |
| `snapshots/1730/broker-state.json` | `JOURN-6 status=in_progress pipeline_stage=plan definition=null` | V1 起点 |
| `artifacts/broker-log-slice.txt`（各快照） | 与 `broker.log` 切片一致 | 交叉复核 |

> 注：`artifacts/broker.log` 时间为 `+0800`（17:xx），UTC 为 09:xx，两者差 8 小时，上文已对齐标注；`snapshots/1730–1802` 的 `scheduler.json` 因 `405 Method Not Allowed` 未落 `jobs`，故分解检查证据以 `broker-state.json` 的 `scheduler` 数组为准。

