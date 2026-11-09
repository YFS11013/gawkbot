# Journey 9 独立评分卡（2026-11-09）

> 独立判分者依据 `prereg.md` §4–§7，回到 `artifacts/`、`snapshots/`、`broker-log-tail.txt` 原始证据逐条判定，不采信观测记录的结论化表述。隔离办公室 `journey9`（broker :7910 / web :7911），被测提交 `690b601` / `000c4b7`，蓝图 `ai-revops`（cos/revops/analyst/ae/reviewer），父 Issue `JOURN-6`（`cos__human`、owner=cos、created_by=human、无 `definition`）。
>
> 判分基准：`docs/office/core-loop-runs/2026-11-09-journey9/prereg.md`（V1–V10、F-A…F-H），观测记录 `observations.md` 仅作时间线索。快照目录 `snapshots/2145-aborted/`（第一次作废轮次）判分时不计入场景。`broker.log` 时间为 `+0800`（21:xx），UTC 为 13:xx，差 8 小时，上文以 UTC 为主。
>
> 证据锚点：`artifacts/broker-state.json`（`tasks` 15 条、`requests` 4 条、`scheduler` 24 条、`actions` 98 条）、`artifacts/messages.json`（103 条）、`artifacts/broker-log-tail.txt`、`artifacts/wiki-log.txt`、`artifacts/team-tree.txt`、`artifacts/wiki-tree.txt`、`artifacts/broker-log-slice.txt`、`snapshots/{1348,1352,1357}/scheduler.json|tasks.json`。

## 0. 判定总览

| 验证项 | 结论 | 一句话证据 |
|---|---|---|
| **V1** 每条 lane 都带写范围 | **通过** | `artifacts/broker-state.json` 中 JOURN-6 的 4 条子 Issue（14/16/18/32）与 3 条 system 审阅 lane（35/56/77）均含 `definition.artifact_path`，0 条 `artifact_not_required` |
| **V2** 共享 wiki 真有交付 | **通过（带保留）** | `artifacts/broker-log-tail.txt` 21:47:39 `JOURN-14 bytes=477`、21:47:47 `JOURN-16 bytes=499`，`team-tree.txt` 两文件在盘（477b/499b），`slug=analyst/ae` 与 owner 一致；但 JOURN-18 声明路径未落地（见 §4 D1） |
| **V3** 守卫是提示，不是墙 | **未观察** | 场景内 0 条 `no write scope` create 拒绝；规划者 4 条 lane 一次即带路径，无撞后改做法轨迹 |
| **V4** 不回归 | **未通过** | 计划卡唯一（`request-10`）、`task_not_decomposed` 0 条、分解检查带 `status_reason`，但 `actions` 里 `kind=watchdog_alert` 3 条（场景内 2 条：action-59 13:51:21、action-84 13:53:42），判据要求 0 条 |
| **V5** 反向臂探针 | **通过（带保留）** | P1 403 拒绝（`broker-log-tail.txt` 22:20:04）、`artifact_path_missing` 无新增（messages 0 条），但文案未见点名 creator（仅 `task=JOURN-16 slug=ae`） |
| **V6** 不刷屏 | **通过（未锻炼）** | 场景内 create 拒绝 0 条 ≤5；但与 V3 同因，未经历压力 |
| **V7** 同父 lane 不重复（D2） | **通过（带保留）** | 无两条 open lane 声明同一 `artifact_path`；JOURN-18 vs JOURN-32 包含度 0.875（<0.9，未识别）；探针 409 均命中 exact-title 而非 CJK shingle，CJK 正例未观察 |
| **V8** 视觉 artifact 不能冒充交付（D3） | **通过（带保留）** | 探针 P2-b 以 `artifact_path=ra_…` 首转 done → 409；但完成门仅查首次转 done：P2-c 以既有他路 200、已 done 后的重复 complete 以 `ra_4e8fa813a190faa3` 改写 artifact → 200（`observations.md` §3 14:21:46，见 §4 D1/D1b） |
| **V9** 落地唤醒（D4） | **未通过** | 唤醒提前一半成立（due 13:53:39 = 落地 13:51:39 + 2 分钟）；但同一 Issue 静默 26.1 分钟（13:53:57→14:20:02），超 ≤10 分钟且触发 F-H |
| **V10** 分解计数（D5） | **通过** | `status_reason="decomposed into 4 sub-issue(s); 1 still open"`（`snapshots/1352/scheduler.json`），与实际 4 条子 Issue（14/16/18/32）、1 条 open（16）一致 |
| 硬失败 | 结论 | 一句话证据 |
|---|---|---|
| **F-A** 写范围拒 ≥3 次同内容 | 未触发 | 场景内 create 拒绝 0 次；`broker-log-tail.txt` 全日志 `write-task refused` 仅 2 条（1 场景路径不匹配 + 1 停表后探针） |
| **F-B** 路径敷衍 / 误用 `artifact_not_required` | 未触发 | 7 条 `artifact_path` 均为 `team/accounts|emails|reviews/acme-*` 有语义路径；`artifact_not_required` 0 条 |
| **F-C** 计划卡缺席 | 未触发 | `request-10`（approval，`cos__human`）13:45:49 出现、13:47:04 answered |
| **F-D** 60 分内 `team/**` 仍 0 份 | 未触发 | 13:47:17 建 lane → 13:47:39 首写成功（`team/accounts/acme-renewal-quote.md`，477B），间隔 22 秒 |
| **F-E** 批准后 10 分内无子任务且无报警 | 未触发 | 批准 13:47:04 → 子任务 13:47:17（13 秒）；`messages.json` 中 `task_not_decomposed` 0 条 |
| **F-F** 同父两条 open 同 `artifact_path` | 未触发 | JOURN-18 声明 `acme-renewal-check.md`、JOURN-32 声明 `acme-renewal-final-check.md`，路径不同（近重复见 V7/D4） |
| **F-G** `ra_` 收 done 被放行且声明文件不在盘 | 未触发（字面） | 首次转 done 的 `ra_` 探针（P2-b）被 409 拒；但已 done 后的重复 complete 以 `ra_` 改写 artifact → 200（探针 JOURN-111，14:21:46，`observations.md` §3），同属「声明路径不兑现」缺口（见 §4 D1/D1b） |
**硬失败回退结论**：F-A、F-B 未触发 → 不切 B；F-F、F-G（字面）未触发 → D2/D3 判据不被宣告「不成立」；但 **F-H 已触发**，按 prereg §5 回退条款「D4 的改动不足以覆盖真实唤醒需求」，须按产品决策重评（sibling-completion wake 仍未覆盖 decision 态阻塞）。**人工干预 1 次**（13:47:04 approve `request-10`）；探针 P1/P2/P3 均停表后（14:20:02 后）执行，`observations.md` §3，不计入场景。

---

## 1. V1–V10 逐条判定

### V1 — 每条 lane 都带写范围

**判定：通过**

- `artifacts/broker-state.json` 的 `tasks`：JOURN-14 `definition.artifact_path="team/accounts/acme-renewal-quote.md" defined_at 13:47:17Z`；JOURN-16 `team/emails/acme-renewal-followup.md`；JOURN-18 `team/reviews/acme-renewal-check.md`；JOURN-32 `team/reviews/acme-renewal-final-check.md`（`defined_at 13:47:42Z`）；system 审阅 lane JOURN-35/56/77 亦分别含 `team/reviews/JOURN-{14,32,18}.md`。7 条 lane 全部带路径，`artifact_not_required` 0 条。
- 敷衍判断：7 条路径均含 `team/{accounts,emails,reviews}/acme-*` 业务词，非 `team/tasks/x.md` 类无信息路径；`F-B` 面的敷衍分支未成立（JOURN-18 声明路径未兑现属交付缺口，见 §4 D1，不属"敷衍值"）。
- 理由：形式判据（"每个子 Issue 都有 `definition.artifact_path`"）逐条满足；语义偏离（JOURN-18/32 近重复、JOURN-18 声明未兑现）按 prereg 分在 V7/§4 D1 计，不扣 V1。

### V2 — 共享 wiki 真的有交付

**判定：通过（带保留）**

- `artifacts/broker-log-tail.txt`：`21:47:39 wiki write-task: task=JOURN-14 slug=analyst path=team/accounts/acme-renewal-quote.md bytes=477`；`21:47:47 … task=JOURN-16 slug=ae … bytes=499`。`artifacts/team-tree.txt` 复核两文件在盘（`team\accounts\acme-renewal-quote.md 477b`、`team\emails\acme-renewal-followup.md 499b`）。
- 作者即 lane owner：日志 `slug` 分别为 `analyst`、`ae`，与 `broker-state.json` 中 JOURN-14 `owner=analyst`、JOURN-16 `owner=ae` 一致（场景两件主交付物满足）。
- 保留项：JOURN-18 声明的 `team/reviews/acme-renewal-check.md` **从未出现在盘上**（`team-tree.txt` 无此文件；`broker-log-tail.txt` 无 JOURN-18 的 `write-task` 成功行），其 `artifact` 指向 JOURN-32 的 `acme-renewal-final-check.md`（1931B）。V2 的判据是"≥1 份 markdown"且已满足，但单 lane 交付对账在此失效——见 §4 D1。

### V3 — 守卫是提示，不是墙

**判定：未观察**

- `artifacts/broker-log-tail.txt` 全日志 `write-task refused` 仅 2 条：21:47:40 JOURN-16 路径不匹配（场景内，`follow-up` vs `followup`）与 22:20:04 JOURN-16 探针；场景内 `no write scope` / create 拒绝 0 条。4 条 cos 创建的 lane（13:47:17 ×3、13:47:42）一次即带 `artifact_path`。
- 理由：prereg V3 要求的两可观测面——"撞后改做法"与"原样重试 ≥2 次"——本轮均无触发（同 J8 的 0 撞结论）。唯一一次路径不匹配是 `write-task` 的 403（D6 分支），7 秒后纠正重试成功，属 V5/D6 的证据，不可外推为 create 守卫的改做法证据。

### V4 — 不回归

**判定：未通过**

- 计划卡唯一：`artifacts/broker-state.json` 的 `requests` 4 条中仅 `request-10`（`kind=approval, channel=cos__human, title="Start work on Journey 9…", status=answered`）为计划卡；request-71/91/96 均为 notebook→wiki 推广卡（`blocking` 为空，不 gate）。
- `task_not_decomposed` 0 条：`artifacts/messages.json` 全文检索 0 命中。
- 分解检查正常收尾带 `status_reason`：`snapshots/1352/scheduler.json` `plan-decomposition-check:…:journ-6 status=done last_run=13:52:22Z status_reason="decomposed into 4 sub-issue(s); 1 still open"`。
- **未通过项**：`artifacts/broker-state.json` 的 `actions` 中 `kind=watchdog_alert` 共 3 条 —— action-59（13:51:21，JOURN-18）、action-84（13:53:42，JOURN-6）、action-97（14:21:22，request-71，停表后）。场景时段内至少 2 条不为 0，判据"0 条"不成立。注：`messages.json` 按 `kind`/全文检索 `watchdog_alert` 为 0 条，`actions` 又为 3 条，两源口径冲突；J8 评分按 messages 口径判过，本轮 `observations.md` §2 亦明记 "actions 里出现 `watchdog_alert`"。按字面与最强证据（`actions` 的 `kind` 字段）判定，本项未过；impl 口径不一致另列 §4 D3。

### V5 — 反向臂探针（停表后，不计场景干预）

**判定：通过（带保留）**

- 403 发生：`artifacts/broker-log-tail.txt` `22:20:04 wiki write-task refused: task=JOURN-16 slug=ae path=team/emails/wrong-acme.md: task JOURN-16 declares artifact_path "team/emails/acme-renewal-followup.md", not "team/emails/wrong-acme.md"`；`observations.md` §3 记为探针 P1 的 403，body 同时给出声明路径与尝试路径。
- 无新增 `artifact_path_missing`：`artifacts/messages.json` 全文检索 0 命中（0→0），满足 D6 口径（该次不匹配未在频道投提示，与场景内 21:47:40 的不匹配同理）。
- **保留项**：prereg 要求"403 文案点名 creator"。已落盘的拒绝行仅含 `task=JOURN-16 slug=ae`（执行者 `ae`，非 creator `cos`），**证据中无 creator 点名**。不能因 J8 的同类探针有 `@human must set…` 而外推——本轮该要素未被证据支持。

### V6 — 不刷屏

**判定：通过（未锻炼）**

- 场景内因写范围被拒的 `create` 为 0 条（日志仅 1 条场景内 `write-task` 路径不匹配拒绝 + 停表后探针拒绝），≤5 阈值内。
- 与 V3 同因：无一次撞 create 守卫，故"不刷屏"未经重试风暴压力检验（同 J8 结论形态）。

### V7 — 同父 lane 不再重复（D2）

**判定：通过（带保留）**

- 同路径未出现：`broker-state.json` 四条子 Issue 的 `artifact_path` 互异（`acme-renewal-quote.md` / `acme-renewal-followup.md` / `acme-renewal-check.md` / `acme-renewal-final-check.md`），无两条同时 open 声明同一路径 → F-F 未触发。
- 标题重复对（open 集合内）：JOURN-18「子任务 C：数字与事实终审（Acme 续约）」vs JOURN-32「Acme 续约数字与事实终审」、路径分别为 `acme-renewal-check.md` 与 `acme-renewal-final-check.md`，`observations.md` §2 重算包含度 0.875 < 0.9 阈值，守卫未拦截。判据字面为"被 CJK shingle 判据识别的重复标题成对出现"——未识别即不构成违反，但 0.875 仅差 0.025 过阈值，属近 miss（记 §4 D4）。
- 守卫命中面：操作员探针 P3 两次 409（`observations.md` §3：重建 JOURN-32 标题、"子任务 C：…"标题均 409，指向 JOURN-32 / JOURN-18），但记录明确写明命中的是既有的 `titlesAreSimilar` 精确标题路径，**不是 CJK shingle 判据**。故 CJK 真阳性在本轮未被观察到。

### V8 — 视觉 artifact 不能冒充交付（D3）

**判定：通过（带保留）**

- `ra_` 分支成立（探针）：P2-b（`observations.md` §3）以 `definition.artifact_path=team/accounts/probe-d3.md` 建 JOURN-111、用 `artifact_path=ra_0123456789abcdef` 收 done → **409** "declared the write scope … a visual artifact does not close it"，D3 门在线上生效。
- 保留项：完成门只查首次转 done 的 `ra_` 前缀。场景内 JOURN-18 声明 `acme-renewal-check.md` 从未写入、以 JOURN-32 的 `acme-renewal-final-check.md` 收 done（`broker-state.json` `JOURN-18.artifact` 证）；探针 P2-c 同型复现 JOURN-111 以 `team/accounts/acme-renewal-quote.md` 收 done → 200。两处均未被 D3 拦——判据字面锚定的 `ra_` 未被绕过，但"声明了路径却用别处现存文件关闭"的同族缺口在本轮真机存在（§4 D1）。另：探针 JOURN-111 在 14:21:46 **已 done 后**再次 `complete`（`artifact_path=ra_4e8fa813a190faa3`）→ 200 并把 `artifact` 改写为视觉 artifact（`observations.md` §3），说明完成门仅在首次「转 done」时判定，`reachedDone=false` 的重复 mutation 不受门限约束（§4 D1b）；随后 cos 14:22:10–29 把探针 JOURN-111 当真实任务收束，同一汇总已污染（`observations.md` §3）。

### V9 — 落地唤醒（D4）

**判定：未通过**

- 提前唤醒一半成立：`snapshots/1348/scheduler.json` 父 Issue `task-follow-up:…:journ-6 due_at=14:45:26Z` → `snapshots/1352/scheduler.json` `due_at=13:53:39Z`。JOURN-18 于 13:51:39 `completed_at`，13:53:39 恰为落地 + 2 分钟（`broker-state.json` `JOURN-18.completed_at=13:51:39Z`），满足"≤3 分钟内提前到 +2 分钟附近"；13:53:42 `broker-log-tail.txt` `scheduler: dispatched 1 due job(s)`，cos 13:53:52 发布进展（`messages.json` 13:53:52）。
- **静默一半不成立**：父 Issue 最后活动 `JOURN-6.updated_at=13:53:57Z`（`broker-state.json`）→ 停表 14:20:02，**静默 26.1 分钟**，超 "最长静默 ≤10 分钟"；14:04:43 仅系统 `task_stalled`/watchdog 提示（`actions` action-87 + 频道消息），无 agent active 回合。按 prereg §5 → **F-H 触发**："一条 lane 落地后同一 Issue 仍静默 ≥15 分钟且无 active 回合"。
- 附因（机制层）：JOURN-16 自 13:48:03 起卡 `decision`/`ready_for_review`，14:12 快照 `pending_interview=None`、全库无指向它的 interview/approval 卡（`observations.md` §2），recheck 于 14:03:02 收尾无推进（`broker-state.json` `recheck:ae--cos:task:journ-16 status_reason="awaiting review (decision) — the ball is not with the owner"`）。D4 的提前唤醒已消费一次，之后父 Issue 回到 30 分钟节拍（`due_at=14:23:42Z`，`snapshots/1357/scheduler.json`），而停表规则（15 分钟静默）在 14:20:02 先触发。—— 即 F-H 的近因是 decision 态无人类可操作卡，而非 D4 提前未生效（§4 D2）。

### V10 — 分解计数（D5）

**判定：通过**

- `snapshots/1352/scheduler.json`（1357/1400 复核一致）：`plan-decomposition-check:cos--human:task:journ-6 status=done status_reason="decomposed into 4 sub-issue(s); 1 still open"`，`last_run 13:52:22Z`。
- 与实际对账：`broker-state.json` 中 `parent_issue_id=JOURN-6` 的子 Issue 为 JOURN-14/16/18/32 共 4 条，其中 1 条 open（JOURN-16 `status=in_progress`，在 13:52 时点 `completed_at` 为空，最终 `completed_at=14:20:36Z` 系停表后探针所为）；system lane 35/56/77 无 `parent_issue_id`，不计入。J8 的"报 1 实 6"计数错误已修复。

---

## 2. F-A … F-H 硬失败判定

| 硬失败 | 触发 | 证据（文件 + 值） |
|---|---|---|
| **F-A** 回退触发：同 lane 写范围拒 ≥3 次且 create 内容不变 | 未触发 | `broker-log-tail.txt` 场景内 `write-task refused` 仅 1 条（JOURN-16 路径不匹配 21:47:40）；create 拒绝 0 条；`broker-state.json` 4 条 lane 均一次创建即带 `artifact_path`（`defined_at` 13:47:17/13:47:42） |
| **F-B** 回退触发：路径全敷衍，或 `artifact_not_required` 误用 | 未触发 | 7 条 `artifact_path`（`broker-state.json` 各 `definition`）均为业务路径；`artifact_not_required` 0 条。JOURN-18 声明未兑现属交付缺口（§4 D1），不满足"所有 lane 敷衍"字面 |
| **F-C** 计划卡缺席 | 未触发 | `broker-state.json` `request-10`（13:45:49 创建、13:47:04 `answered`） |
| **F-D** 60 分内 `team/**` 仍 0 | 未触发 | 建 lane 13:47:17 → 首写 13:47:39（`broker-log-tail.txt` 21:47:39 `bytes=477`），`team-tree.txt` 终态 `team/**` 35 文件 |
| **F-E** 批准后 10 分内无子任务且无报警 | 未触发 | 批准 13:47:04 → 首子任务 13:47:17（13 秒，`snapshots/1352/tasks.json` / `broker-state.json` `created_at`）；`messages.json` `task_not_decomposed` 0 条 |
| **F-F** 两条 open 同 `artifact_path` | 未触发 | JOURN-18 `acme-renewal-check.md` vs JOURN-32 `acme-renewal-final-check.md`（已见 V7）；探针 P3 409 为精确标题路径命中，事后复建被拒，非场景内出现 |
| **F-G** `ra_` 收 done 被放行且声明文件不在盘 | 未触发（字面） | 首次转 done 的 `ra_` 探针（P2-b）被 409 拒，场景内无 `ra_` 收 done；但已 done 后的重复 complete 以 `ra_4e8fa813a190faa3` 改写 artifact → 200（探针 JOURN-111，14:21:46，`observations.md` §3），属"声明路径不兑现"同族缺口，见 §4 D1/D1b。后者证据以操作员探针落盘（`broker-state.json` 终态 JOURN-111 `artifact` 已回写为 `team/accounts/acme-renewal-quote.md`，故该中间 200 以 `observations.md` §3 为唯一证据） |
| **F-H** 落地后静默 ≥15 分且无 active 回合 | **触发** | `JOURN-6.updated_at=13:53:57Z` → 停表 14:20:02（26.1 分钟）；`actions` action-87 `task_stalled` 14:04:43（"11m0s — marked stalled"）后无回合；见 V9 |

## 3. 环境清理单列

- `ai-revops-1..4` 在重建后 13:45:01–02 取消（`observations.md` §0；`broker-state.json` 四条 `status=canceled`），不计场景干预。
- 第一次作废轮次（`JOURN-2`、starter lane）见 `observations.md` §0，工件在 `snapshots/2145-aborted/`，判分不计。
- 停表后探针（14:20:04 起）：P1（V5/D6）、P3（D2）、P2-a/P2-b/P2-c（D3/D7）均属操作员探针，不计场景干预；其副作用 JOURN-105（JOURN-16 审阅，`team/reviews/JOURN-16.md` 5406B，`broker-log-tail.txt` 22:21:04）、JOURN-111/JOURN-114（探针 lane）为探针引入，场景证据中已剔除（终态 `JOURN-16.completed_at=14:20:36Z`、`JOURN-111.artifact=team/accounts/acme-renewal-quote.md` 系探针后状态）。

---

## 4. 缺陷（按严重度，最重在前）

> 注：除注明"操作员探针引入 / 环境引入"外，均为场景内机制缺陷；判据字面问题另行标注，不冒充已触发的 F*。

### D1（重）完成门只拦 `ra_` 前缀，声明路径可被任意现存文件冒名关闭 —— V8/P2-c/JOURN-18

- **可观测事实**：JOURN-18 `definition.artifact_path="team/reviews/acme-renewal-check.md"`，但盘上该文件从未存在；任务 `status=done` 且 `artifact="team/reviews/acme-renewal-final-check.md"`（JOURN-32 的文件）。`broker-log-tail.txt` 有 JOURN-32 21:49:17 的 `write-task … bytes=1931`，无 JOURN-18 的成功写行。`wiki-log.txt` commit `98af7ad`（reviewer，JOURN-32）与 `fe85fd` 均记 final-check 为 JOURN-32 产物。探针 P2-c 现场复现：JOURN-111 声明 `team/accounts/probe-d3.md` 从未存在，以 `team/accounts/acme-renewal-quote.md` 收 done → 200（`observations.md` §3 14:21:0x；`broker-state.json` 终态 JOURN-111 `artifact` 即该路径）。
- **位置**：`artifacts/broker-state.json`（JOURN-18 `definition` vs `artifact`；JOURN-111 `artifact`）、`artifacts/team-tree.txt`（无 `acme-renewal-check.md`）、`artifacts/broker-log-tail.txt`、`artifacts/wiki-log.txt`；P2-c 200 以 `observations.md` §3 为证据（broker-state 终态已被后续探针改写，见 D1b）。
- **应有机制**：完成校验除 `ra_` 前缀外，还须断言 `task.artifact == definition.artifact_path`（或等值归一化），不等即 409 并指明声明路径与尝试路径（复用 write-task 403 的"点名 + 一次改对"格式）。本轮触发后 F-G 的字面（`ra_`）不构成，但下一轮 D3 验收应补"声明路径 ≠ 提交路径"用例，否则 D3 只是对 `ra_` 的补丁。

### D1b（重）重复 complete 可改写 artifact，完成门只在首次转 done 时判定 —— 探针 JOURN-111（操作员探针引入、用例有效）

- **可观测事实**：探针 lane JOURN-111 已 `done`（14:21:12）后，14:21:46 analyst 再次发 `complete`（`artifact_path=ra_4e8fa813a190faa3`）→ 200，`artifact` 被改写为视觉 artifact；`observations.md` §3（14:21:46）记终态为 `artifact=ra_4e8fa813a190faa3`（文件仍为 `probe-d3.md` 不存在）。`broker-state.json` 终态 JOURN-111 `artifact` 为 `team/accounts/acme-renewal-quote.md`，说明该次改写之后又被回写——即同一探针任务的 `artifact` 被连续两次改动，终态以最后一次为准；中间 200 以 `observations.md` §3 为唯一证据，`broker-log-tail.txt` 无对应 22:21:46 行。
- **归因**：**操作员探针引入的复现**（探针 JOURN-111/JOURN-114 本身是探针任务；`observations.md` §3 14:21:12 副作用派生 JOURN-114），但揭穿的是**完成门只在「转 done」时判定**的机制缺陷——`reachedDone=false` 的重复 mutation 不再过门。cos 14:22:10–29 随后把探针 lane 当真实任务收束（`observations.md` §3），探针污染父 Issue 汇总，另计 D8。
- **应有机制**：`complete` 无论 lane 是否已 `done`，每次都执行与首次机检相同的 artifact 一致性校验（含 `ra_` 前缀与 `artifact == definition.artifact_path`）；重复 `complete` 直接 idempotent 200（不改写 artifact）或再次 409。

### D2（重）decision 态无人类可操作卡，父 Issue 在有人待命时静默 26 分钟而被停表 —— F-H（机制近因，同 J8 的 26 分钟）

- **可观测事实**：JOURN-16 `decision` 自 13:48:03 不再有回合（`broker-state.json` `ledger` 末条 13:48:03）；14:03:02 recheck 收尾无推进（`status_reason="awaiting review (decision)…"`）；14:12 时 `pending_interview=None`、全库无指向它的 interview/approval 卡，人类唯一可见信号是 13:53:52 的频道一句"待你确认发送"（`observations.md` §2）；父 Issue `JOURN-6.updated_at=13:53:57Z` 后静默至停表 14:20:02（26.1 分钟），14:04:43 `task_stalled`（action-87 "11m0s"）无恢复。
- **位置**：`artifacts/broker-state.json`（JOURN-16/JOURN-6 `updated_at`、recheck `status_reason`、`actions` action-87）、`snapshots/1357/scheduler.json`（父 Issue 唤醒后回 30 分钟节拍 `due_at=14:23:42Z`，晚于停表）。
- **应有机制**：`review_state=ready_for_review/decision` 的 lane 应自动物化一张**指向 lane 本身**的审批/确认卡（阻塞在该 lane 上、且 `sibling-completion wake` 之外），或 recheck 在无卡时直接向 `cos__human` 升 request；否则 30 分钟节拍 + 15 分钟停表规则必在"等 HUMAN"处误停（本轮 `due_at=14:23:42Z` > 停表点 14:20:02）。标记：**场景内机制缺陷**（非操作员探针、非环境）。

### D3（中）`watchdog_alert` 口径两份、两边阈值不同 —— V4 未过

- **可观测事实**：`actions`（`kind=watchdog_alert`）场景内 2 条（action-59 13:51:21、action-84 13:53:42），`messages.json` 按 `kind`/全文检索 0 条；prereg "0 条"未说明计数源。
- **位置**：`artifacts/broker-state.json` `actions`；`artifacts/messages.json`；`observations.md` §2（"actions 里出现 `watchdog_alert`"）。
- **应有机制**：判据与 `watchdog` 的落盘路径统一（二者皆以 `actions.kind` 计，或新增 `messages.kind=watchdog_alert`），并明确 `task_stalled` 是否计入 V4；否则每轮结论随口径翻转（J8 按 messages 口径过、本轮按 actions 口径不过）。

### D4（中）D2 兄弟去重仍有 0.025 的绕行窗，CJK 真阳性从未被锻炼 —— V7 保留项

- **可观测事实**：JOURN-18 vs JOURN-32 包含度 0.875（`inter/min = 7/8`，`observations.md` §2 重算）与阈值 0.9 仅差 0.025；两条标题仍均 open 共存（`broker-state.json` 均 `status=done` 且 13:51–13:53 间均 open），两路径相差一词（`check` vs `final-check`）。探针 P3 的两次 409 均系 `titlesAreSimilar` 精确标题命中，非 CJK（`observations.md` §3 自述）。
- **位置**：`artifacts/broker-state.json`（JOURN-18/32 `title`、`definition`）、`observations.md` §2–§3。
- **应有机制**：近同义路径（edit-distance/后缀词）应与标题同查；同时需要在某轮以微改标点/助词的 CJK 重复标题实测 CJK 真阳性（本轮只有"未误杀"和"精确标题拦截"，没有一条 CJK 启用证据）。

### D5（中）V5 的"点名 creator"要素在证据中不可见

- **可观测事实**：P1 拒绝行（22:20:04）仅 `task=JOURN-16 slug=ae` + 声明/尝试路径；J8 的同类行含 `@human must set definition.artifact_path … retrying this call cannot succeed`，本轮行中无 creator、无"不可重试"声明。
- **位置**：`artifacts/broker-log-tail.txt` 22:20:04；`observations.md` §3（探针 P1）。标记：**操作员探针引入的证据缺口**（探针已用，文案要素缺失计在机制上，不计场景干预）。
- **应有机制**：`write-task` 路径不匹配的拒绝行统一带 creator 点名与"一次改对"指示，与 `artifact_path_missing` 投递一致。

### D6（轻，环境引入）gbrain memory 写入连续失败，记忆落盘降级

- **可观测事实**：`broker-log-tail.txt` 21:49:56 起 `source_capture: … gbrain call put_page … EOF` 多条；启动即 `wiki: gbrain unavailable … falling back to in-memory index`。`wiki-log.txt` 的 skill/playbook/learn 归档照常，但非 wiki 的记忆写入失败。
- **位置**：`artifacts/broker-log-tail.txt`。标记：**环境引入**，未影响 V1–V10 的任务面判定（核心循环未用 gbrain 作门）。

### D7（轻，未锻炼面）V3/V6 的守卫教育与节流仍为未观察

- 与 J8 结论同：本轮规划者一次没撞 `no write scope`（见 V3）；write-task 路径不匹配 7 秒改对证据（D6）不可外推 create 守卫文案。
- **应有机制**：下轮保留"故意不带写范围"的单 lane 用例，产出 create 拒绝行与频道提示各 1 条。

### D8（操作员探针引入，不计机制缺陷）探针 P2-a 用法错误 + 14:21:46 加探针污染与副作用

- **事实**：P2-a 用 `artifact` 字段（被忽略）实为 `artifact_path`，收 done 的 200 不构成门限证据（`observations.md` §3 自述"探针未命中"，判分剔除）。P2-c 的 200 与 14:21:46 重复 complete 分别是有效的 D3 缺口/完成门缺陷复现（计入 D1/D1b）。新增：探针 lane JOURN-111 被 analyst 在 done 后重复 complete（14:21:46，`ra_4e8fa813a190faa3` 200）→ JOURN-114 的派生有 14:21:12 先行（见 D1b）→ cos 14:22:10–29 把探针 lane 当真实任务收束："JOURN-111 已完成并已标记 done（artifact_path=ra_…），结论是 D3 无法通过：本轮 wiki 检索…0 命中"（`observations.md` §3）。即 broker/机器人无法区分探针任务与业务任务，探针污染了父 Issue 的汇总。副作用 JOURN-105/JOURN-111/JOURN-114 与 `JOURN-16.md` 5406B 均为停表后引入，不计 V1–V10 场景证据（§3 已单列；D1/D1b 的机制缺陷标注计）。

---

## 5. 本轮不能证明什么

1. **CJK shingle 在真实重复上能拦住**：本轮只有 0.875 的近 miss 与精确标题拦截；`observations.md` §2 的 1.000 对（JOURN-14 vs 35、32 vs 56）均为 system 审阅 lane，不走规划者守卫。D2 的核心修复（CJK 包含度）没有一条真阳性证据。
2. **规划者在场景流内被 `ra_` 409 拒后用 write-task/not_required 自修正**：D3 的 409 只在探针 P2-b 出现；场景内无 `artifact_required` 命中，更无"撞后改做法"的轨迹（V8 仅证明门存在）。
3. **create 守卫的文案可读性与"不刷屏"的节流能力**：V3 未观察、V6 未锻炼（同 J8）——同 J8 的结论不从本轮借力。
4. **V5 的 creator 点名**：见 D5，证据中无该要素。
5. **人类在环能闭环**：JOURN-16 的 decision 态从未物化为人类卡（`pending_interview=None`）；本轮 human 唯一动作是批准一次计划卡。不能证明"cos 汇总待确认 → 人类确认 → lane 收尾"链路（JOURN-6 终态仍 `in_progress/pending_review`）。
6. **父 Issue 的自然收尾**：停表时 `JOURN-6` 未 done（`broker-state.json` `status=in_progress, lifecycle_state=running, review_state=pending_review`）。不能证明父任务在推广卡 pending（request-71/91/96）下可收尾——同 J8 D1 形态（park）本轮未被实锤也未被证伪。
7. **30 分钟 follow-up 节拍与 15 分钟停表的兼容性**：父 Issue 唤醒后被抬到 `due_at=14:23:42Z`（+30 分钟），停表在 14:20:02 先发生。不能证明该策略组合在"长期等待"场景下是设计还是误停。
8. **reviewer 的视觉 artifact（`ra_01fc3c1f14f1d426`）的定性**：`wiki-log.txt` commit `67b359` 显示 reviewer 写 `JOURN-32` 小结时同步创建 visual artifact，`wiki-tree.txt` 终态 2 组 `ra_*` 存在。本轮可判定它未被用作完成载体，但不能证明该行为在其它运行时不会分流（D3 防的正是这件事）。

---

## 6. 总评与分数

- **三处修复的真机 verdict**：**D5 完全成立**（计数 4/1 准确）；**D4 机制一半成立**（提前唤醒 +2 分钟精确命中），但第二半（Issue 最长静默 ≤10 分钟）被 F-H 证伪（实际 26.1 分钟），近因在 decision 态无卡（D2），按 prereg 回退条款 D4 不足以覆盖唤醒需求，须产品重评；**D3 的 `ra_` 门成立**（探针 409），但"声明路径可被现存他路关闭"的同族缺口（D1）仍在，D3 判据应补"提交路径 ≠ 声明路径"；**D2 未证伪亦未证实**（无同路径共存，CJK 未锻炼）。
- **回退**：F-A/F-B/F-F/F-G（字面）均未触发 → 不切 B，§9.15 选 A 保持；F-H 已触发 → D4 进入产品重评通道。人工干预 1 次（合规）。
- **场景任务层面**：两件主交付物（477B 报价、499B 邮件）只用任务事实、未编造，按 13:47:39 即落地；但父 Issue 未收尾、JOURN-18 声明未兑现，不得称核心循环已闭环（与 J8 的"有交付、无收尾"同型）。

**核心循环分数：5/10** —— D5 全对、D4 提前唤醒与 D3 的 ra_ 门在真机留下证据，但 F-H 触发使 Issue 仍静默 26.1 分钟、声明路径可被既有他路关闭（JOURN-18/P2-c）、父 Issue 停表时未收尾；循环能开工而不能自愈收尾。

## 7. 证据清单

| 证据文件 | 关键行/字段 | 用于 |
|---|---|---|
| `artifacts/broker-state.json` | `tasks`：JOURN-14/16/18/32（`parent_issue_id=JOURN-6`，各 `definition.artifact_path`）；JOURN-18 `artifact=…final-check.md` vs `definition=…check.md`；JOURN-6 `updated_at=13:53:57Z, status=in_progress, lifecycle_state=running, review_state=pending_review`；JOURN-16 `completed_at=14:20:36Z`（探针后）；JOURN-111 `definition=team/accounts/probe-d3.md` vs `artifact=team/accounts/acme-renewal-quote.md`；`requests` request-10 answered、71/91/96 pending；`actions` watchdog_alert 3 条（59/84/97）、task_stalled action-87；`scheduler` plan check done + `status_reason` | V1–V10, F*, D1–D4 |
| `snapshots/1348/scheduler.json` | 父 `task-follow-up:…:journ-6 due_at=14:45:26Z` | V9 |
| `snapshots/1352/scheduler.json` | 同上 `due_at=13:53:39Z`；plan check `status_reason="decomposed into 4 sub-issue(s); 1 still open" last_run 13:52:22Z` | V9, V10 |
| `snapshots/1357/scheduler.json` | 父 follow-up `due_at=14:23:42Z last_run=13:53:42Z` | V9, D2 |
| `artifacts/messages.json` | 103 条；`task_not_decomposed` 0、`artifact_path_missing` 0；13:53:52 cos 汇总；14:04:43 task_stalled | V4, V5, V9 |
| `artifacts/broker-log-tail.txt` | `21:47:39 JOURN-14 bytes=477`、`21:47:40 JOURN-16 refused(路径不匹配)`、`21:47:47 JOURN-16 bytes=499`；`22:20:04 探针P1 refused`；gbrain EOF 多条；write-task 成功 7 条（14/16/35/32/56/77/105）无 JOURN-18 行 | V2, V3, V5, D6 |
| `artifacts/team-tree.txt` | `team\accounts\acme-renewal-quote.md 477b`、`team\emails\acme-renewal-followup.md 499b`、`team\reviews\acme-renewal-final-check.md 1931b`；无 `acme-renewal-check.md` | V2, D1 |
| `artifacts/wiki-tree.txt` | `team/accounts/acme-renewal-quote.md`、`team/emails/acme-renewal-followup.md`、`team/reviews/{acme-renewal-final-check,JOURN-14,16,18,32}.md`；`wiki/visual-artifacts/ra_01fc…/ra_e732…` | V2, §5-8 |
| `artifacts/wiki-log.txt` | `98af7ad reviewer JOURN-32 final-check`、`fe85fd86 JOURN-32…final-check.md 1931B`、`67b359 reviewer visual artifact ra_01fc…` | V2, D1 |
| `artifacts/broker-log-slice.txt` | 与 broker-log-tail 的 write-task 段一致（交叉复核） | V2, V5 |
| `observations.md` §0–§3 | 作废轮次、唯一干预 13:47:04、探针 P1/P2/P3（停表后）、CJK 0.875 重算表、JOURN-16 无卡（14:12） | 时间线、探针归因、环境单列 |

> 注：本卡判分未改动 `observations.md`、`prereg.md` 或任何 artifacts/snapshots；`snapshots/2145-aborted/` 已按判分约定剔除。
