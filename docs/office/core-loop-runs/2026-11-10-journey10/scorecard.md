# Journey 10 独立评分卡（2026-11-10）

> 独立判分者依据 `prereg.md` §4–§7，回到 `artifacts/`、`snapshots/`、`broker-log-tail.txt` 原始证据逐条判定，不采信观测记录的结论化表述。隔离办公室 `journey10`（broker :7910 / web :7911），被测提交 `52456d3`（+ `76e6171`、`fdb6fa8`），蓝图 `ai-revops`（cos/revops/analyst/ae/reviewer），父 Issue `JOURN-14`（`cos__human`、owner=cos、created_by=human、无 `definition`）。
>
> 判分基准：`docs/office/core-loop-runs/2026-11-10-journey10/prereg.md`（V1–V15、F-A…F-M、§1.1 模型替换），观测记录 `observations.md` 仅作时间线索。`broker.log` 时间为 `+0800`（11:xx），UTC 为 03:xx/02:xx，差 8 小时，下文以 UTC 为主、本地时间加注。
>
> 证据锚点：`artifacts/broker-state.json`（`tasks` 10 条、`requests` 1 条、`actions` 59 条、`scheduler` 11 条）、`artifacts/messages.json`（75 条）、`artifacts/broker-log-tail.txt`、`artifacts/wiki-log.txt`（579 行）、`artifacts/team-tree.txt`、`artifacts/team__*.md`（4 份交付物原文）、`snapshots/{1044,1047,1049,submit-1041}/broker-state-slice.json|messages.json`。

## 0. 判定总览

| 验证项 | 结论 | 一句话证据 |
|---|---|---|
| **V1** 每条 lane 都带写范围 | **通过** | `artifacts/broker-state.json` 中 JOURN-14 的 2 条子 Issue（JOURN-21/23）与 2 条 system 审阅 lane（JOURN-30/46）均含 `definition.artifact_path`，0 条 `artifact_not_required` |
| **V2** 共享 wiki 真有交付 | **通过** | `artifacts/wiki-log.txt` commit `dce4efc`（`team/accounts/acme.md`）与 `5c3c6b3`（`team/emails/acme-renewal-followup.md`），`team-tree.txt` 两文件在盘（617B/910B），`broker-log-tail.txt` 10:46:02/10:50:03 `write-task` 200 且 `slug` 与 owner 一致 |
| **V3** 守卫是提示，不是墙 | **未观察** | 场景内 0 条 `no write scope` create 拒绝；规划者 2 条 lane 一次即带路径，无撞后改做法轨迹 |
| **V4** 不回归 | **未通过** | 计划卡唯一（`request-17`）、`task_not_decomposed` 0 条、分解检查带 `status_reason`，但 `actions` 里 `kind=watchdog_alert` 2 条（02:49:02 action-32/33），判据要求 0 条 |
| **V5** 反向臂探针 | **通过（带保留）** | 停表后 P4 403 拒绝（`broker-log-tail.txt` 11:26:44 `declares … not "team/reviews/not-declared.md"`）、`artifact_path_missing` 0→0；为停表后探针，不计场景干预 |
| **V6** 不刷屏 | **通过（未锻炼）** | 场景内因写范围被拒的 create 0 条 ≤5；与 V3 同因，未经历重试风暴 |
| **V7** 同父 lane 不重复 | **通过** | 无两条同时 open 的 lane 声明同一 `artifact_path`；JOURN-21 `team/accounts/acme.md` vs JOURN-23 `team/emails/acme-renewal-followup.md` 路径不同，标题无 CJK shingle 重复 |
| **V8** 视觉 artifact 不能冒充交付 | **通过（带保留）** | 场景内无 `ra_` 收 done，未出现「声明了路径、`team/**` 无该文件、任务 done」；但 `ra_` 门仅在探针外未被锻炼 |
| **V9** 落地唤醒 | **通过（带保留）** | JOURN-21 落地 02:46:55 → 快照 1047 `task-follow-up:cos--human:journ-14` due `02:48:55Z`（+2 分钟）满足 ≤3 分钟；JOURN-23 落地后未在 3 分钟内再次提前（due 仍 `03:19:02Z`），最长静默 9.6 分钟（02:59:41→03:09:19）≤10 分钟但第二条 lane 唤醒未兑现 |
| **V10** 分解计数 | **通过** | `artifacts/broker-state.json` `plan-decomposition-check:…:journ-14` `last_run=02:49:43Z` `status_reason="decomposed into 2 sub-issue(s); 1 still open"`，与实际 2 条子 Issue（21/23）、1 条 open（23 当时）一致 |
| **V11** decision 态有一张卡（D2-1） | **未通过** | JOURN-14 于 02:54:20 进入 `lifecycle_state=decision`，至 02:57:2x、03:04:20（+10 分钟）、03:11:44、终态 `artifacts/broker-state.json` `requests` 仍仅 `request-17`（`dedupe_key=plan-approval:JOURN-14`、answered），无 `dedupe_key=decision:JOURN-14` |
| **V12** 卡不堆积（D2-2） | **未观察** | 从未出现 `decision:` 卡，无法检验 ≥60 秒跨 sweep 恒为 1、总数不增长 |
| **V13** 点击即落地（D2-3） | **未观察** | 场景内无可点 `decision:` 卡；停表后 P1a 409、P1b 200→`done/approved` 为探针结果，不计场景 |
| **V14** 返工同轮发生（D2-4） | **未观察** | 场景内无 `request_changes_decision` + 返工轨迹，owner 无需在 15 分钟内推进 |
| **V15** 声明路径即落地路径（D6） | **通过** | 每条 `write-task` 200 的路径在 `wiki-log.txt` 有提交且在 `team-tree.txt` 在盘；P3'（`JOURN-73` `pin-probe-v2.md` commit `0a4c053`）未污染邻居 `pin-probe.md`（`355e3b7`），P4 非声明路径 403 且未落盘 |

| 硬失败 | 结论 | 一句话证据 |
|---|---|---|
| **F-A** 同 lane 写范围拒 ≥3 次同内容 | **未触发** | `broker-log-tail.txt` 场景内无 `create` 拒因缺 `artifact_path`；`write-task` 拒仅 ownership/已关闭分支 |
| **F-B** 路径全敷衍或误用 `artifact_not_required` | **未触发** | `broker-state.json` 4 条 `artifact_path` 均为 `team/accounts|emails|reviews/*` 有语义路径；`artifact_not_required` 0 条 |
| **F-C** 计划卡缺席 | **未触发** | `artifacts/broker-state.json` `request-17`（`cos__human`、02:42:13Z 创建、02:44:33Z answered、title `Start work on Journey 10…?`） |
| **F-D** 60 分内 `team/**` 仍 0 | **未触发** | 建 lane 02:45:16 → 首写 02:46:02（`broker-log-tail.txt` 10:46:02 `bytes=617`）、02:50:03 第二写，间隔 <5 分钟 |
| **F-E** 批准后 10 分内无子任务且无报警 | **未触发** | 批准 02:44:33 → 首子任务 02:45:16（43 秒，`broker-state.json` `created_at`）；`messages.json` `task_not_decomposed` 0 条 |
| **F-F** 两条 open 同 `artifact_path` | **未触发** | JOURN-21 `team/accounts/acme.md` vs JOURN-23 `team/emails/acme-renewal-followup.md` 路径不同；探针 JOURN-70/73 虽路径近似但 `parent_issue_id` 均为 null，非同父 |
| **F-G** `ra_` 收 done 被放行且声明文件不在盘 | **未触发** | 场景内无 `ra_` 前缀收 done；两条主交付物 `artifact` 与 `definition.artifact_path` 逐字一致且文件在盘 |
| **F-H** 落地后静默 ≥15 分无回合且无 `decision:` 卡 | **未触发** | 有 open lane 期间最长静默 9.6 分钟（02:59:41→03:09:19），且 03:09:03 `recheck` 唤醒打破静默；停表后 03:09:19→03:25:25 的 16 分钟静默发生在全部 lane 已 closed 之后，按 prereg 归入 F-I 而非 F-H 的“唤醒”回退 |
| **F-I** lane 进 `decision` ≥10 分无 `decision:` 卡（D2-1 失败） | **触发** | JOURN-14 `lifecycle_state=decision` 自 02:54:20 起，至 03:04:20 已 10 分钟，`artifacts/broker-state.json` `requests` 仍仅 `request-17`，`pending_interview` 全程 null（`snapshots/1047/1049`、`broker-state.json` 各任务 `pending_interview=null`） |
| **F-J** 点击 200 但 2 分内状态未到目标或卡未撤（D2-3 失败） | **未触发** | 场景内无点击；停表后 P1b `POST /tasks/JOURN-14/decision {artifact_path: team/accounts/acme.md}` 200 后 `status=done/lifecycle=approved`、`artifact=team/accounts/acme.md`，`requests` 清空，状态与卡一致 |
| **F-K** send back 后 15 分内 owner 无回合无消息无 follow-up 提前（D2-4 失败） | **未触发** | 场景内未发生 `request_changes_decision`，无该分支可检验 |
| **F-L** 非阻塞被误用：卡出现后 15 分内其余 open lane 无推进且无 active 回合 | **未触发** | 从未出现 `decision:` 卡，卡被当阻塞闸门的情形无法观察；其余 open lane（JOURN-23）在卡缺席期间仍于 02:50:03 落稿、02:54:16 收 done，有推进 |
| **F-M** `write-task` 200 但声明路径无文件或落进相似名邻居（D6 失败） | **未触发** | 场景 `dce4efc`/`5c3c6b3` 声明路径与落盘路径逐字一致；探针 `0a4c053` 落 `pin-probe-v2.md` 未追加进 `pin-probe.md`（`team__reviews__pin-probe.md` 10B 独立），`not-declared.md` 403 未落盘 |

**硬失败回退结论**：F-A、F-B 未触发 → 不切 B；F-F、F-G 未触发 → 对应判据不被宣告「不成立」；但 **F-I 已触发**，按 `prereg.md` §5 回退条款「D2 判『真机不成立』，回到 §9.19 重新论并考虑把应答改为阻塞闸门（那会让每轮被人工闸门挂起，故仅限于『lane 已交付、只差签字』的终态）」。F-H 未触发 → 不重新评估唤醒。F-M 未触发 → D6 路径钉子保持，不回滚为诊断日志。**人工干预 1 次**（02:44:33 approve `request-17`）；探针 P1a/P1b/P2/P3/P3'/P4 均停表后（03:26:xx）执行，不计入场景。

---

## 1. V1–V15 逐条判定

### V1 — 每条 lane 都带写范围

**判定：通过**

- `artifacts/broker-state.json` 的 `tasks`：JOURN-21 `definition.artifact_path="team/accounts/acme.md" defined_at 2026-11-10T02:45:16Z`；JOURN-23 `team/emails/acme-renewal-followup.md defined_at 2026-11-10T02:45:22Z`；system 审阅 lane JOURN-30 `team/reviews/JOURN-21.md defined_at 2026-11-10T02:46:55Z`；JOURN-46 `team/reviews/JOURN-23.md defined_at 2026-11-10T02:54:16Z`。4 条 lane 全部带 `artifact_path`，`artifact_not_required` 0 条。
- 敷衍判断：4 条路径均为 `team/{accounts,emails,reviews}/acme-*.md` 或 `JOURN-*.md` 业务路径，非 `team/tasks/x.md` 类无信息路径；`F-B` 的敷衍分支未成立。
- 父 Issue JOURN-14 本身无 `definition`（场景提交时即无，`created_by=human`、`task_type=issue`），属 prereg 允许的“让规划者自己决定 lane 写范围”形态，不计入“子 Issue 带写范围”考核。

### V2 — 共享 wiki 真有交付

**判定：通过**

- `artifacts/broker-log-tail.txt`：`10:46:02 wiki write-task: task=JOURN-21 slug=revops path=team/accounts/acme.md bytes=617`；`10:50:03 … task=JOURN-23 slug=ae path=team/emails/acme-renewal-followup.md bytes=910`；`10:49:18 … task=JOURN-30 slug=cos path=team/reviews/JOURN-21.md bytes=2668`；`10:56:32 … task=JOURN-46 slug=cos path=team/reviews/JOURN-23.md bytes=3584`。
- `artifacts/wiki-log.txt`：`dce4efc revops 2026-11-10 10:46:02 +0800 Add Acme renewal quote one-pager (JOURN-21) team/accounts/acme.md`；`5c3c6b3 ae 10:50:02 +0800 Add Acme renewal follow-up email draft … team/emails/acme-renewal-followup.md`；`6d9cc8e cos 10:49:18 +0800 JOURN-30 独立审阅结论 … team/reviews/JOURN-21.md`；`06ab721 cos 10:56:32 +0800 审阅结论 … team/reviews/JOURN-23.md`。`artifacts/team-tree.txt` 复核四文件均在盘（`team/accounts/acme.md` `team/emails/acme-renewal-followup.md` `team/reviews/JOURN-21.md` `team/reviews/JOURN-23.md`）。
- 作者即 lane owner：`broker-log-tail.txt` `slug` 分别为 `revops`/`ae`/`cos`，与 `broker-state.json` 中 JOURN-21 `owner=revops`、JOURN-23 `owner=ae`、JOURN-30/46 `owner=cos` 一致。`bytes=` 本身不算证据，按 README §6 第 4 条此处仅作辅助，落盘以 `git log` + `team-tree.txt` 为准。
- 两件主交付物均满足“≥1 份 markdown 且 git 有提交且在盘”。

### V3 — 守卫是提示，不是墙

**判定：未观察**

- `artifacts/broker-log-tail.txt` 全日志 `write-task refused` 3 类：`10:53:29 task JOURN-23 is owned by ae, not cos`、`10:57:43 task is already closed: JOURN-46 is done`、`11:26:08` 探针组（ownership / 已关闭 / 路径不匹配），无一条 `create` 因 `no write scope` 被拒。`artifacts/broker-state.json` 2 条 cos 创建的 lane（02:45:16/02:45:22）一次即带 `artifact_path`。
- 理由：prereg V3 要求的两可观测面——“撞后改做法”与“原样重试 ≥2 次”——本轮均无触发（同 J9 的 0 撞结论）。唯一一次路径不匹配是停表后探针 P4（见 V5），7 秒后无重试（探针单次），不可外推为 create 守卫的改做法证据。

### V4 — 不回归

**判定：未通过**

- 计划卡唯一：`artifacts/broker-state.json` 的 `requests` 仅 `request-17`（`kind=approval, channel=cos__human, dedupe_key=plan-approval:JOURN-14, status=answered, created_at 02:42:13Z, answered 02:44:33Z`）为计划卡；无第二张 `plan-approval`。
- `task_not_decomposed` 0 条：`artifacts/messages.json` 全文检索 0 命中。
- 分解检查正常收尾带 `status_reason`：`artifacts/broker-state.json` `plan-decomposition-check:cos--human:task:journ-14 status=done last_run=02:49:43Z status_reason="decomposed into 2 sub-issue(s); 1 still open"`；`snapshots/1049/broker-state-slice.json` 同值复核。
- **未通过项**：`artifacts/broker-state.json` 的 `actions` 中 `kind=watchdog_alert` 共 2 条 —— action-32（02:49:02Z，`@cos still needs to move Journey 10 …` related JOURN-14）、action-33（02:49:02Z，`@ae still needs to move Acme 续约 48 小时跟进邮件草稿 …` related JOURN-23），判据“0 条”不成立。按 README §6 第 1 条唯一计数源为 `actions[].kind`，`messages.json` 按 `kind`/全文检索 `watchdog_alert` 为 0 条符合预期（两源本就不一致），但 V4 判据明确以 `actions` 为准故不过。
- 附因：同一时刻两人都在跑回合——`observations.md` §2 记 cos 回合 02:49:03 起、ae 于 02:50:03 落稿；`broker-log-tail.txt` `11:26:08` 前后亦有心跳。watchdog 在有 active 回合时告警为误鸣，但判据字面为 0 条，故 V4 仍记未过，口径不一致另列 §3。

### V5 — 反向臂探针（停表后，不计场景干预）

**判定：通过（带保留）**

- 403 发生：`artifacts/broker-log-tail.txt` `11:26:44 wiki write-task refused: task=JOURN-73 slug=cos path=team/reviews/not-declared.md: task JOURN-73 declares artifact_path "team/reviews/pin-probe-v2.md", not "team/reviews/not-declared.md"`；`observations.md` §5 记为探针 P4 的 403，body 同时给出声明路径与尝试路径，符合 prereg “点名两条路径”分支（README §6 第 2 条：路径不匹配分支只点名两条路径、不点名 creator）。
- 无新增 `artifact_path_missing`：`artifacts/messages.json` 全文检索 0 命中（0→0），满足 D6 口径（该次不匹配未在频道投提示，与场景内 `11:26:08` 的 ownership 拒绝同理）。
- **保留项**：该 403 为停表后探针（03:26:44Z，`artifacts/broker-state.json` `JOURN-73` `created_at 03:26:43Z`），不计场景干预；场景内无路径不匹配 `write-task`，V5 “守卫能自愈”的改做法证据未在场景内产生。

### V6 — 不刷屏

**判定：通过（未锻炼）**

- 场景内因写范围被拒的 `create` 为 0 条（日志仅 `write-task` 拒绝，见 V3/V5），≤5 阈值内。
- 与 V3 同因：无一次撞 create 守卫，故“不刷屏”未经重试风暴压力检验。

### V7 — 同父 lane 不重复

**判定：通过**

- 同路径未出现：`broker-state.json` 两条子 Issue 的 `artifact_path` 互异（`team/accounts/acme.md` vs `team/emails/acme-renewal-followup.md`），无两条同时 open 声明同一路径 → F-F 未触发。
- 标题重复对（open 集合内）：JOURN-21「Acme 续约一页报价说明（落 team/accounts/acme.md）」vs JOURN-23「Acme 续约 48 小时跟进邮件草稿（落 team/emails/acme-renewal-followup.md）」，无 CJK shingle 可识别的重复，守卫未拦截属正确。
- 探针相似名：`JOURN-70` `team/reviews/pin-probe.md` 与 `JOURN-73` `team/reviews/pin-probe-v2.md`（`artifacts/broker-state.json` `defined_at 03:26:43Z`）路径仅差后缀，但二者 `parent_issue_id` 均为 null（非同父），且为停表后探针，不计入 V7 考核。

### V8 — 视觉 artifact 不能冒充交付

**判定：通过（带保留）**

- 场景内无 `ra_` 前缀 artifact 提交，`broker-state.json` 中 JOURN-21/23/30/46 的 `artifact` 均与 `definition.artifact_path` 逐字一致且文件在盘（见 V2），“声明了路径却用别处现存文件关闭”未发生。
- **保留项**：完成门对 `ra_` 的拦截未在场景内被锻炼；仅能证明“未出现冒名关闭”，不能证明“若用 `ra_` 会被 409”。按 prereg 字面锚定的 `ra_` 分支本轮无正例，亦无反例，故记通过但带保留。

### V9 — 落地唤醒

**判定：通过（带保留）**

- 提前唤醒一半成立：`snapshots/1044/broker-state-slice.json` 父 Issue `task-follow-up:cos--human:journ-14 due_at=03:44:33Z` → `snapshots/1047/broker-state-slice.json` `due_at=02:48:55Z`。JOURN-21 于 02:46:55Z `completed_at`（`broker-state.json` `JOURN-21.completed_at=02:46:55Z`），02:48:55Z 恰为落地 +2 分钟（调度 `broker-log-tail.txt` `10:49:03 scheduler: dispatched 2 due job(s)` 对应 `snapshots/1049` `last_run 02:49:02Z`），满足“≤3 分钟内提前到 +2 分钟附近”。
- 兄弟唤醒同理：`snapshots/1047` `task-follow-up:ae--cos:journ-23 due_at=02:48:55Z`（原 `03:19:03Z` 提前），与父 Issue 同步提前。
- **保留项**：第二条 lane JOURN-23 于 02:54:16Z `completed_at`，但 `snapshots/1049` 与 `artifacts/broker-state.json` 显示 `task-follow-up:cos--human:journ-14` `due_at` 仍为 `03:19:02Z`（`last_run 02:49:02Z` 后未再提前）、`recheck:cos--human:task:journ-14` 下一次 `last_run=03:09:03Z` 距落地 14 分 47 秒，超出“≤3 分钟”窗口。即 V9 的“任一 lane”由 JOURN-21 满足，但“每条 lane”未满足。
- 静默一半：父 Issue `JOURN-14.updated_at=02:54:20Z`（进入 decision）→ 最后场景消息 `msg-62` 02:59:41Z → `recheck` 唤醒 03:09:03Z，最长无消息/无 `issue_lifecycle` 更新区间 02:59:41→03:09:19（9.6 分钟）≤10 分钟；开 open lane 期间（至 02:54:16 JOURN-23 closed）最长静默 02:50:28→02:54:07（3.6 分钟）≤10 分钟。按 `messages.json` 与 `artifacts/broker-state.json` `updated_at` 计，静默条款字面通过。

### V10 — 分解计数

**判定：通过**

- `artifacts/broker-state.json` `plan-decomposition-check:cos--human:task:journ-14 status=done last_run 02:49:43Z status_reason="decomposed into 2 sub-issue(s); 1 still open"`，`snapshots/1049` 复核一致。
- 与实际对账：`broker-state.json` 中 `parent_issue_id=JOURN-14` 的子 Issue 为 JOURN-21/23 共 2 条，其中 1 条在 02:49:43 时 open（JOURN-23 `status=in_progress`，最终 `completed_at=02:54:16Z`）；system 审阅 lane JOURN-30/46 `parent_issue_id` 为 null，不计入。

### V11 — decision 态有一张卡（D2-1）

**判定：未通过**

- `observations.md` §2 记 `JOURN-14` 于 02:54:20Z 进入 `lifecycle_state=decision`（`review_state=ready_for_review`，`pipeline_stage=review`，`artifact=null`，`artifact_not_required=null`）；`artifacts/broker-state.json` 终态 `JOURN-14` `lifecycle_state=approved` 为停表后 P1b 批准后的状态，但 `ledger` 末条仍为 `02:54:20Z #JOURN-14 已进入待审`，且 `scheduler` `recheck:cos--human:task:journ-14` `last_run=03:09:03Z status_reason="awaiting review (decision) — the ball is not with the owner"` 证实曾处 decision。
- 卡缺席：`artifacts/broker-state.json` `requests` 仅 `request-17`（`dedupe_key=plan-approval:JOURN-14`、`status=answered`），无任何 `dedupe_key=decision:JOURN-14`；`snapshots/1047/1049` 同为仅 `request-17`；`observations.md` §2 复核 02:55:42、03:04:20（已满 10 分钟）、03:11:44、终态 03:27 均仅 `request-17`。
- `pending_interview` 仍为 null 符合非阻塞要求（`broker-state.json` 全部任务 `pending_interview=null`、`snapshots/*/broker-state-slice.json` 同），但卡本身未出现故 V11 不满足。根因：`JOURN-14` 进入 decision 时 `artifact` 与 `artifact_not_required` 均为 null → `hasApprovableDeliverable` 为假 → `raiseDecisionCardLocked` 直接返回 `""`（`observations.md` §2 02:55:42 记）。

### V12 — 卡不堆积（D2-2）

**判定：未观察**

- 由于 V11 未产生任何 `dedupe_key=decision:<taskID>` 卡，无法检验“跨 ≥2 个 sweep 间隔（≥60 秒）同一 dedupe_key 卡数恒为 1”“requests 总数不因重复 sweep 增长”。
- `artifacts/broker-state.json` `requests` 数组长度 1（仅已答的 plan 卡），`scheduler` `task_recheck`/`task_follow_up` 均按 15/30 分钟节拍正常调度，无因重复 sweep 产生的增量。

### V13 — 点击即落地（D2-3）

**判定：未观察**

- 场景内无 `decision:` 卡可点，故两条点击路径（`approve_decision` ≤10 秒 `done/approved` 且卡消失；`request_changes_decision` + 文字 ≤10 秒 `changes_requested` 且原话入 `changes_requested.body`）均未被锻炼。
- 停表后探针对照（不计场景，仅作机制参考）：P1a `POST /tasks/JOURN-14/decision {action:approve}` 不带 `artifact_path` → 409 `task JOURN-14 cannot be approved into done without a delivered artifact…`（`observations.md` §5）；P1b 同路径 + `artifact_path=team/accounts/acme.md` → 200 `{"action":"approve","status":"recorded"}`，`artifacts/broker-state.json` `JOURN-14` 转 `done/approved` `artifact=team/accounts/acme.md`，`artifacts/wiki-log.txt` `aafd3a2 decision: promote JOURN-14 on approval team/decisions/JOURN-14.md`，`messages.json` `msg-66/67/68 decision_approved` 03:26:08Z。两次点击均按预期分支响应，但为停表后证据，不纳入 V13 场景判定。

### V14 — 返工同轮发生（D2-4）

**判定：未观察**

- 场景内未发生 `request_changes_decision` + 返工要求，`artifacts/broker-state.json` 中 `JOURN-14` 从未进入 `lifecycle_state=changes_requested`，`changes_requested` 字段无。
- 无 “send back 之后 ≤15 分钟内 owner 出现 active 回合或新消息并推进（改写文件 / 重新 submit_for_review）” 的轨迹可检；`observations.md` §5 的 P1b 为直接批准路径，未走返工分支。

### V15 — 声明路径即落地路径（D6）

**判定：通过**

- 场景两条主交付物：`artifacts/broker-log-tail.txt` `10:46:02 task=JOURN-21 … path=team/accounts/acme.md bytes=617` 对应 `artifacts/wiki-log.txt` `dce4efc team/accounts/acme.md` 且 `artifacts/team-tree.txt` 在盘；`10:50:03 task=JOURN-23 … path=team/emails/acme-renewal-followup.md bytes=910` 对应 `5c3c6b3 team/emails/acme-renewal-followup.md` 且在盘；声明路径与落盘路径逐字一致。
- 审阅 lane 同理：`10:49:18 task=JOURN-30 … team/reviews/JOURN-21.md bytes=2668` 对应 `6d9cc8e`；`10:56:32 task=JOURN-46 … team/reviews/JOURN-23.md bytes=3584` 对应 `06ab721`。
- 探针对照（停表后，属 V15 反向臂）：P3 `JOURN-70` 声明 `team/reviews/pin-probe.md` → `artifacts/broker-log-tail.txt` `11:26:43 wiki write-task: task=JOURN-70 … path=team/reviews/pin-probe.md bytes=10`、commit `355e3b7`、`artifacts/team__reviews__pin-probe.md` 10B 在盘；P3' `JOURN-73` 声明 `team/reviews/pin-probe-v2.md` → `11:26:44 … task=JOURN-73 … path=team/reviews/pin-probe-v2.md bytes=10`、commit `0a4c053`、`team__reviews__pin-probe-v2.md` 10B 在盘且**落在 v2 自己的路径**，`pin-probe.md` 内容未被追加（比对 `artifacts/team__reviews__pin-probe.md` 与 `team__reviews__pin-probe-v2.md` 各 10B 独立，未出现旧版 `decsmoke` 的“v2 内容追加进 v1”行为）；P4 同一任务 `JOURN-73` 打未声明路径 `team/reviews/not-declared.md` → `11:26:44 wiki write-task refused: … declares artifact_path "team/reviews/pin-probe-v2.md", not "team/reviews/not-declared.md"` 403 且 `team-tree.txt` 无 `not-declared.md`，未出现“200 + bytes 有日志、声明路径不存在”。

---

## 2. 硬失败与回退结论

| 硬失败 | 是否触发 | 证据（文件 + 值） | 回退条款 |
|---|---|---|---|
| **F-A** | 未触发 | `broker-log-tail.txt` 场景内无 `create` 因缺 `artifact_path` 被拒；`artifacts/broker-state.json` 2 条 lane 均一次创建即带 `definition.artifact_path` | 按 `prereg.md` §5 不切 B |
| **F-B** | 未触发 | `broker-state.json` 4 条 `artifact_path` 均为业务路径；`artifact_not_required` 0 条 | 不切 B |
| **F-C** | 未触发 | `broker-state.json` `request-17` 02:42:13Z 创建、02:44:33Z answered | — |
| **F-D** | 未触发 | 建 lane 02:45:16Z → 首写 10:46:02 +0800（02:46:02Z），`team-tree.txt` 终态 `team/**` 2 主交付 + 2 审阅 | — |
| **F-E** | 未触发 | 批准 02:44:33Z → 首子任务 02:45:16Z（43 秒）；`messages.json` `task_not_decomposed` 0 条 | — |
| **F-F** | 未触发 | `broker-state.json` JOURN-21 vs JOURN-23 `artifact_path` 不同；探针 JOURN-70/73 非同父 | 对应判据不宣告「不成立」 |
| **F-G** | 未触发 | 场景内无 `ra_` 收 done；声明路径均在盘（`dce4efc`/`5c3c6b3`） | 对应判据不宣告「不成立」 |
| **F-H** | 未触发 | 有 open lane 期间最长静默 9.6 分钟 <15 分钟；`broker-state.json` `recheck:…:journ-14 last_run 03:09:03Z` 及时唤醒，无 ≥15 分钟“有落地、无回合、无卡”静默 | 不重新评估唤醒 |
| **F-I** | **触发** | `JOURN-14` `lifecycle_state=decision` 自 02:54:20Z 起，至 03:04:20Z 已 10 分钟，`broker-state.json` `requests` 仍仅 `request-17`（answered），`pending_interview=null`（`broker-state.json` 各任务、`snapshots/1047`） | **D2 判「真机不成立」，回到 `docs/office/DESIGN-DECISIONS.md` §9.19 重新论并考虑把应答改为阻塞闸门（仅限于“lane 已交付、只差签字”的终态）** |
| **F-J** | 未触发 | 场景内无点击；停表后 P1b 200 后 `lifecycle_state=approved` 且 `requests` 清空，无“200 但状态不变”或“卡未撤” | D2 同上，已由 F-I 覆盖 |
| **F-K** | 未触发 | 未发生 `request_changes_decision`，无该分支可触发 | 同上 |
| **F-L** | 未触发 | 未出现 `decision:` 卡，无法形成“卡被当阻塞闸门”——其余 open lane 在卡缺席期间仍推进（JOURN-23 02:50:03 落稿、02:54:16 done） | — |
| **F-M** | 未触发 | `wiki-log.txt` 200 路径均有提交且在盘（`dce4efc`/`5c3c6b3`/`355e3b7`/`0a4c053`）；`not-declared.md` 未落盘 | D6 保持，不回滚为诊断日志 |

**回退执行**：F-A/F-B 未触发 → 不切 B（维持 `team/**` markdown 为交付载体）；F-F/F-G 未触发 → D2/D3 判据不被宣告「不成立」；F-I 触发 → D2（`76e6171` 三出口非阻塞卡）判真机不成立，根因 `JOURN-14` 进 `decision` 时 `artifact=null` 导致 `hasApprovableDeliverable` 为假；F-M 未触发 → D6（`52456d3` `PinnedPath`）保持。人工干预仅 1 次（02:44:33 `POST /interview/answer {id:request-17, choice_id:approve}` 200）。

---

## 3. 本轮新暴露、但 `prereg.md` 没覆盖的东西

> 仅列有证据的（时间戳 + 字段），标明「不在本轮判据内」。

1. **父 Issue 无可批准交付物导致 `decision:` 卡无法物化**（不在 V1–V10 判据内，但触发 F-I）。`JOURN-14` 进 `decision` 时 `artifact=null`、`artifact_not_required=null`（`observations.md` §2 02:54:20、`recheck` `status_reason="awaiting review (decision) — the ball is not with the owner"` 03:09:03Z），`hasApprovableDeliverable` 为假使 `raiseDecisionCardLocked` 返回 `""`，致人类无卡可点。`artifacts/broker-state.json` `JOURN-14` `artifact` 在场景结束前为 null（停表后 P1b 带 `artifact_path=team/accounts/acme.md` 才变 `team/accounts/acme.md`）。

2. **`watchdog_alert` 在有 active 回合时误鸣**（不在 V3/V4 的“提示文案”判据内）。`artifacts/broker-state.json` `actions` action-32/33 均 `created_at 02:49:02Z`，而 `observations.md` §2 记 cos 回合 02:49:03–02:49:38（`total_ms=161659`）与 ae 落稿 02:50:03 正在进行，`broker-log-tail.txt` `10:49:03 scheduler: dispatched 2 due job(s)` 亦在同分钟。告警与执行重叠，非“无推进”。

3. **`task_follow_up` 与 `recheck` 的职责重叠与静默重计时**（不在 V9 判据内）。`artifacts/broker-state.json` `task-follow-up:cos--human:journ-14` `status=done enabled=false closed_at 02:53:59Z status_reason="superseded by recheck"`，随后 `recheck:cos--human:task:journ-14 last_run 03:09:03Z` 接管，`msg-63/64/65`（03:09:11–03:09:19Z `ae__cos` cos）为 `recheck` 唤醒后的重复播报（`artifacts/messages.json` 63–65），无新交付或状态变化，仅重复确认“审阅已落库、任务板无待办”。

4. **`write-task` ownership 门先于 `artifact_path` 门生效**（不在 V5 判据内）。`artifacts/broker-log-tail.txt` `10:53:29 task JOURN-23 is owned by ae, not cos`、`11:26:08 task JOURN-23 is owned by ae, not cos`（探针 P2，`slug=cos` 打 `ae` 的任务）均因作者不符直接 403，未走到“声明路径 vs 尝试路径”分支；`observations.md` §5 自述 P2 为“探针写错作者，非目标分支”。正确分支由 P4 命中。

5. **gbrain 记忆落盘连续失败**（环境，不在 V1–V15 内）。`artifacts/broker-log-tail.txt` `10:40:05 WARNING wiki: gbrain unavailable … falling back to the in-memory index` 起，`10:46:55/10:49:29/10:54:18` 等 `source_capture: … gbrain call put_page … EOF` 多条；启动即 `wiki_index_gbrain_entity: connect … EOF`。未影响任务面判定（核心循环未以 gbrain 为门）。

6. **onboarding 重命名 `Access is denied`**（环境，不在判据内）。`artifacts/broker-log-tail.txt` `10:40:21/10:40:22/10:40:24 onboarding: transition broker callback failed … rename … onboarded-*.json … Access is denied` 三条。

7. **已关闭任务的 `write-task` 被拒**（不在 V15 内）。`artifacts/broker-log-tail.txt` `10:57:43 task is already closed: JOURN-46 is done`、`11:02:14` 同。

---

## 4. 与 Journey 9 的机制差异

> 只列机制层面差异，不列措辞。

| 维度 | Journey 9（`690b601`） | Journey 10（`52456d3` + `76e6171`） | 机制含义 |
|---|---|---|---|
| **模型（prereg §1.1 赛前替换）** | `opencode-go/crouton-alpha-free` | `opencode-go/longcat-2.5-free`（赛前实测 `pong` 16s 一次读对，`beam-free` 22s 两次） | 跨轮差异必须先扣除“模型换了”再归因到机制；主办公室同步换模型 |
| **分解粒度** | `plan-decomposition-check` 报 `4 sub-issue(s)`（JOURN-14/16/18/32） | 报 `2 sub-issue(s)`（JOURN-21/23） | 同一题目、同一蓝本下规划者拆 lane 数从 4 变 2 |
| **D6 路径钉（`52456d3` `PinnedPath`）** | `JOURN-18` 声明 `acme-renewal-check.md` 未落盘、artifact 指向 `JOURN-32` 的 `final-check.md`；探针 P2-c 以他路文件收 done 200 | 场景 `dce4efc`/`5c3c6b3` 声明与落盘逐字一致；探针 `355e3b7` 与 `0a4c053` 各自落地、`not-declared.md` 403 未落盘，邻居未被追加 | `PinnedPath` 在真机上生效，J9 的“200 但落邻居”缺口在本轮未复现 |
| **D2 `decision:` 卡（`76e6171`）** | `JOURN-16` 在 `decision`（`ready_for_review`）无卡，父 Issue `JOURN-6` 静默 26.1 分钟触发 F-H | `JOURN-14`（父 Issue 自身）进 `decision` 无卡，10 分钟触发 F-I；`JOURN-21/23` 均 `review_state=not_required` 直接 `done/approved`，未进 `decision` | 卡机制在两种“谁进 decision”形状下均未物化；J10 根因更明确为 `artifact=null` 导致 `hasApprovableDeliverable` 假 |
| **`watchdog_alert` 计数** | `actions` 2–3 条（场景内 2 条），`messages` 0 条，口径差曾使 J9 在 J8 过、J9 不过 | 同为 `actions` 2 条（02:49:02 双告警）、`messages` 0 条，且告警时两 bot 正在跑回合（同分钟调度 dispatched） | 两轮均以 `actions[].kind` 为唯一源，J10 的误鸣证据更完整 |
| **V5 反向臂文案分支** | P1 拒绝行仅含 `task=… slug=…` + 两条路径，未 点名 creator（保留项） | P4 拒绝行同为“声明 vs 尝试”分支（`declares … not "…"`），仍不点名 creator，符合 README §6 第 2 条“能自愈的错误回到调用点”分支 | 文案按实际命中分支判定，两轮一致，但 J10 命中了正确的路径分支 |
| **决策闭环** | 停表时 `JOURN-6` `running/pending_review` 未收尾 | 停表时 `JOURN-14` `decision/ready_for_review` 未收尾，停表后 P1b 带 `artifact_path` 200 才 `done/approved`（`aafd3a2`） | 两轮均未在场景内自然收尾，J10 的 P1b 展示了“带交付物批准”路径可闭环 |

---

**结语（中文）**：这一轮唯一有证据的是：V2 的 `dce4efc`/`5c3c6b3` 双交付物落盘、V10 的 `2 sub-issue(s)` 计数准确、V15 的 `PinnedPath` 钉住声明路径（`355e3b7`/`0a4c053` 各自落地、探针 `not-declared` 403）、V9 的 JOURN-21 落地后 2 分钟内 `task-follow-up` 提前（02:48:55Z）、以及 V4/V11 的反面证据——`watchdog_alert` 2 条致 V4 未过、JOURN-14 进 `decision` 10 分钟无 `decision:` 卡致 V11 未过并触发 F-I，而 V12/V13/V14 因从未出现 `decision:` 卡而只能记「未观察」，V3/V6 的 create 守卫与不刷屏、V8 的 `ra_` 门亦因零命中而为「未锻炼/未观察」；换言之，本轮既验证了 D6 路径钉，又复现了 D2 非阻塞卡在父 Issue 无 artifact 时的物化失败。
