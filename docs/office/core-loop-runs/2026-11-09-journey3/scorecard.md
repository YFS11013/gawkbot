# Journey 3 评分卡 — 2026-11-09（core-loop 九步）

> 评分时点：**02:06Z**（北京 10:06）停表 — 连续 17 分钟无新消息/无 active 回合命中 prereg 停表条件。场景起点 **01:02:49Z**（北京 09:02:49）`JOURN-18` 创建，**01:42:27Z** 人工批准 planning，**02:05Z** 停表，**02:06Z** 快照终态。

---

## 1. 判定摘要（≤10 行）

- **整体：6 / 10**（步骤合计 **25 / 36**；预注册 **命中 4 / 7，部分命中 3 / 7**：P2/P3/P5/P6 命中，P1/P4/P7 部分命中）。
- **一句话根因：** `opencode` runner 没有 `RaisePlanApproval` 实现，人给的 Issue 在 `lifecycle_state=planning` 空转 **40 分钟**，需按 prereg 唯一例外人工 `approve` 一次才进入拆活；拆活后 5 子线正确落位、2/4 交付物在正确 wiki 路径可读，但研究简报卡在 `analyst__human` promotion、HTML 未产出，父 Issue 永远 `running`。
- **硬失败：** F2–F6 均 **未触发**（无 hollow done、无假完成、无自耗循环、无超限干预）。
- **人工干预：1 次阻塞批准**（`01:42:27Z POST /tasks {"action":"approve","id":"JOURN-18"}`）+ 安装期 5 条 starter lane `cancel`（`install cleanup`，不计入 F6）。
- **最大进步：** wiki 直写门 `artifact_path` 403 校验、审批卡不阻塞投递、watchdog 可见化 + 短重投、playbook→skill 编译全部在本轮验证通过；最大退步：**planning 自动呈现链路在 opencode 上完全缺失**。

---

## 2. 九步评分表

| 步骤 | 分数 | 判定 | 证据（A=机器可验，B=文件在盘，C=口述） |
|---|---|---|---|
| **1 创建任务** | **4** | delivered | **A：** `snapshots/0112/task-JOURN-18.json` `id=JOURN-18 channel=cos__human owner=cos created_by=human status=in_progress lifecycle_state=planning created_at=2026-11-09T01:02:49Z`（北京 09:02:49）；`ledger[0].agent=cos at=01:05:38Z` 已贴计划。首条子任务 `JOURN-78 created_at=01:42:47Z`。扣 1：频道是 pair DM 而非共享频道，与 Journey 2 同因。 |
| **2 定义清楚** | **2** | broken | **A：** 父 Issue `snapshots/0206/task-JOURN-18.json` 全程无 `definition` 字段（`spec.feedback` 仅有自然语言计划，`definition` 为空）。**A：** 5 条子 Issue 中仅 `JOURN-78 definition.defined_at=2026-11-09T01:47:01Z artifact_path=team/research/smb-retention-2026q4.md` 与 `JOURN-85 definition.defined_at=2026-11-09T01:47:06Z artifact_path=team/emails/smb-renewal-intro.md`（`snapshots/0206/task-JOURN-78/85.json`）有结构化定义；`JOURN-81/89/95` 仍无定义。prereg 要求 `definition.defined_at` 非空且覆盖白名单 ≥2 个路径——**父任务 0/3**，子任务 2/3，部分命中但不满足 CEO 一眼看懂 DoD 的标准。 |
| **3 上下文与授权采访** | **3** | partially | **A：** 本轮无任何 human interview request（`snapshots/0112–0206/requests.json` 在 `cos__human`/`scope=office` 均为空，`observations.md:01:17` 多频道 `requests: []`），符合 constraints「 genuinely undecidable 才问一次」——正确地**没问**。**A：** 上下文检索确有发生：`task-JOURN-18.json ledger.context_used=[wiki:team/getting-started/*×5]`、`task-JOURN-78.json memory_workflow.lookup.updated_at=01:47:27Z query="SMB renewal-retention..."`、`task-JOURN-95.json` 真实执行了 `wiki_list/wiki_search` 并记录 4×404。以 B 计：`agents/cos/notebook/plan-JOURN-18.md`（commit `c3a25c9`）与 `agents/analyst/notebook/2026-11-09-smb-retention-2026q4-简报.md`（commit `20111d2`）均在盘。扣分：采访双落（任务聊天+Inbox）路径**无法验证**，因无采访发生；但若有采访，其路由正确性未被测试。 |
| **4 拉起团队** | **3** | partially | **A：** `snapshots/0206/tasks.json` + 单体 JSON：批准后 **60 秒内**出现 5 条子 Issue——`JOURN-78 analyst__cos analyst 01:42:47Z`、`JOURN-81 ae__cos ae 01:42:52Z`、`JOURN-85 ae__cos ae 01:43:02Z`、`JOURN-89 cos__human cos 01:43:14Z`、`JOURN-95 cos__reviewer reviewer 01:43:21Z`，均 `parent_issue_id=JOURN-18 created_by=cos`。**A：** `log-headless-codex-cos.log:01:43:26Z`「五个子任务已全部派活」与 `01:43:31Z` 分线说明对应。**扣分：** planning 空转 40 分钟（`01:02:49Z→01:42:47Z`）与步骤 2 同根因；若按「人不需要等」口径，实际 spinning 时间远超 Journey 2 的 9 秒，即便派活本身正确，也只能给 partially。 |
| **5 专注执行** | **3** | partially | **A（确有执行）：** `task-JOURN-81.json ledger agent=ae at=01:45:56Z context_used=[wiki:team/inbox/raw-signals.md …]`、`task-JOURN-85.json ledger at=01:48:03Z context_used=[wiki:team/playbooks/smb-renewal-outreach.md, journal:JOURN-85]`、`task-JOURN-78.json ledger at=01:46:38Z`；`log-broker.log:09:47:33 wiki write-task: task=JOURN-85 path=team/emails/smb-renewal-intro.md bytes=733`。**A（执行不全）：** 停表时 3 条 `running`（`JOURN-78/89/95 stalled_since=01:57:59Z/01:53:59Z/01:55:59Z`）、无新日志 `01:48:17Z` 后全员静默 17 分钟。`[INFERENCE]` 未见 `context: learning:X, wiki:Y` 的 turn activity 明文，但 `ledger.context_used` 与 `memory_workflow` 可作为等价证据，降级为 B。 |
| **6 交付物** | **2** | broken | **A：** `snapshots/0206/wiki-tree.txt` 有 `team/playbooks/smb-renewal-outreach.md` 与 `team/emails/smb-renewal-intro.md`，无 `team/research/smb-retention-2026q4.md`、无 `team/visual-artifacts/*.html`。**A：** `task-JOURN-81.json artifact=team/playbooks/smb-renewal-outreach.md lifecycle_state=decision review_state=ready_for_review`、`task-JOURN-85.json artifact=team/emails/smb-renewal-intro.md lifecycle_state=decision`——两条 **不是 `done`**，故无 `hollow_completion` 也不产生 `verification`，门未被绕过但亦未完成。**A：** `task-JOURN-78.json details` 含 `[SUGGESTION] ...已提交晋升 request-129 ...404`，且 `wiki-log-final.txt` 中 `team/research` 无 commit，仅 `agents/analyst/notebook/2026-11-09-smb-retention-2026q4-简报.md` 在 notebook（B）。**A：** `log-broker.log:09:43:46 refuse JOURN-85 ...declares no artifact_path` 与 `09:44:38 refuse JOURN-78` 证明产物门已生效（403），但时序上要求先 `define` 后 `write`，导致两次无谓重试。**结论：** 4 交付物仅 2 在指定 wiki 路径可读，1 在 notebook、1 完全缺失，父 Issue `01:43:36Z` 后无 `done-post` 链接，Verified 徽章诚实地未翻绿——按 DESIGN-DECISIONS §9.5 “人可见交付”口径为 broken。 |
| **7 确定性学习钩子** | **2** | broken | **A：** 无任何 `team/entities/*.facts.jsonl` / `.graph.jsonl` 实体更新，无 `team/learnings/` 新文件（`wiki-tree.txt` 仅 `.gitkeep`）。**B：** 唯一的确定性钩子产出是 `team/skills/smb-renewal-outreach.md`（`wiki-tree.txt` 存在，commit `8fb236c librarian: compile skill smb-renewal-outreach` 09:59:38Z），但它来自 30 分钟 interval 编译而非任务 `done` 后的实体/钩子链路。`artifact-skill.md` 内容正确复用了 playbook 且 `safety_scan= safe`，但不属于步骤 7 定义的 “done 后 entity + playbook draft” 路径，故 **部分可用但判定 broken**。 |
| **8 技能与策略已激活** | **4** | delivered | **A：** `snapshots/0206/wiki-tree.txt` `team/skills/smb-renewal-outreach.md` 存在；`wiki-log.txt:8fb236c` 与 `log-broker.log:09:59:38 INFO writeCompiledSkillLocked: compiled skill created name=smb-renewal-outreach created_by=librarian sha=8fb236c owner_agents=5`。**B：** `artifact-skill.md` 头部 `wuphf.status=active owner_agents=[ae,analyst,cos,reviewer,revops] safety_scan.verdict=safe`，`Steps` 与 `Invariants` 完整引用 `team/playbooks/smb-renewal-outreach.md`。扣 1：策略（policies）原子化分配在快照中无可见 surface，未验证 `Skills/Policies` 分配一致性，但 skill 本体已 delivered。 |
| **9 复用与去重** | **2** | broken | **A：** 本轮无第二个相似任务，按 prereg 以「是否出现重复 lane / `context_used` 是否出现」判。**重复 lane 0 条**（`tasks.json` 10 条含 5 条已 `canceled` starter，5 条 JOURN-* 全 `in_progress`，无同题同 owner 新 id），`create` 冲突未被触发但亦无违规。**A：** `scheduler.json` 中 `JOURN-81/85` 的 `task_retry` 均 `status=done` 且未产生重复任务，符合 P7 观察。**但** 无 “NEXT 任务复用已积累知识（点名、playbook 步骤）并更新而非 duplicating” 的证据——`JOURN-18/78/81/85` 的 `context_used` 虽含 `journal:` 与 wiki，但无跨任务引用，`index/all.md` 未更新。只能给 broken 的低分 partially 替代（2 分），因正确地**没重**但也**没复用**。 |

合计 **4+2+3+3+3+2+2+4+2 = 25 / 36**。

> 十分制 **6 / 10**（Journey 2 为 5/10；22/36→5，本轮 25/36→6；每 +3–4 分约 +1 十分制，按 rubric 整体观感校准，非线性换算）。

---

## 3. 预注册判据逐条

### P1–P7

| # | 判定 | 证据与理由 |
|---|---|---|
| **P1 cos 把任务写成结构化 Definition 并声明产物范围** | **部分命中** | 父 Issue `task-JOURN-18.json` 全程 `definition` 缺失（`spec` 仅 `feedback`，无 `definition.goal/artifact_scope`）。子任务中 `JOURN-78 defined_at=01:47:01Z artifact_path=team/research/smb-retention-2026q4.md`（`snapshots/0206/task-JOURN-78.json:definition`）与 `JOURN-85 defined_at=01:47:06Z artifact_path=team/emails/smb-renewal-intro.md`（`snapshots/0206/task-JOURN-85.json:definition`）在 `cos` 于 `log-headless-codex-cos.log:01:47:17Z`「给 ST-1 补 artifact_scope…顺手把 ST-3 的定义补上」后才出现，且为补救性定义，非批准前一次性声明；`JOURN-81/89` 仍无定义。覆盖白名单 2/3 但**非 cos 在拆活前一次性声明**，故部分命中。 |
| **P2 子 Issue 落在 creator⇄owner 的 pair DM** | **命中** | `tasks.json` `JOURN-78 channel=analyst__cos`、`JOURN-81/85 channel=ae__cos`、`JOURN-95 channel=cos__reviewer` 均为 pair DM；`JOURN-89 channel=cos__human` owner 为 cos 自指，符合 `creator=cos` 的对角线。`log-headless-codex-cos.log` 多次 `posted streamed output to #analyst__cos/#ae__cos/#cos__reviewer` 无 `channel access denied/403`。 |
| **P3 done 不再与产物脱钩（J2-1）** | **命中** | 本轮 **0 条 `done`**，自然 **0 条 `hollow_completion=true`**（`tasks.json` 5 条 JOURN-* 全 `in_progress`）。门本身已生效：`log-broker.log:09:43:46` 与 `09:44:38` 两次 `wiki write-task refused: ...declares no artifact_path; pass human_request instead`；补定义后 `09:47:33 wiki write-task: task=JOURN-85 ...bytes=733` 才放行。未出现 `done+artifact_path 指向不存在文件` 的违规。 |
| **P4 不需要人说「save to wiki」（§9.8+K）** | **部分命中** | `team/playbooks/smb-renewal-outreach.md`（commit `060e338`，`log-broker.log` 未记录但 `wiki-log-final.txt` 有）与 `team/emails/smb-renewal-intro.md`（commit `3e432f0`，`09:47:33`）由 bot 直接写入并产生 git commit，无人类回话即完成。但 `JOURN-78` 简报仅落 `agents/analyst/notebook/2026-11-09-smb-retention-2026q4-简报.md`（commit `20111d2`），并产生 `request-129 @analyst wants to promote ... into the team wiki. Approve?` pending 于 `analyst__human`（`observations.md:01:49`、`scheduler.json:request-follow-up:analyst--human:request-129 next_run=02:15:47Z`），导致 `team/research/smb-retention-2026q4.md` 404。父 Issue `JOURN-18` 亦无完成消息带链接。2/4 自动，1 卡审批，1 完全缺失，故部分命中。 |
| **P5 审批卡不阻塞投递（J2-2）** | **命中** | 停表时唯一 pending 为 `request-129`（`analyst__human`），与交付通道 `ae__cos/cos__human` 分离。`JOURN-81/85` 在 `lifecycle_state=decision` 等审阅期间，`log-headless-codex-cos.log:01:48:17Z` 仍有正文投递，且无 `request pending in this channel; answer required before chat resumes here` 错误；`scheduler.json` 中 `ae__cos` 的 `watchdog_alert` 亦正常投递。未再现 Journey 2 的 `final_chars=0` 空转。 |
| **P6 空转/拒绝唤醒可见化 + 短重投（J2-4）** | **命中** | `log-broker.log:10:00:38/10:02:38/10:02:58/10:04:58 scheduler: dispatched 1 due job(s)` 对应 `scheduler.json` 四条 `task-retry:ae--cos:task:journ-81 last_run=02:02:38Z status=done` 与 `task-retry:ae--cos:task:journ-85 last_run=02:04:58Z status=done`，距空转（`stalled_since 01:53:59Z`）后首次重投 **≤3 分钟可见**，且 `messages.json` 终态与 `observations.md:02:00–02:04` 记录的 `watchdog_alert: "Watchdog: @ae still needs to move ST-2/ST-3 … retrying shortly."` 人可见。 |
| **P7 不再重做已交付的活（J2-6，仅观察）** | **部分命中** | 本轮 5 条子任务无任何 `create` conflict 记录，`tasks.json` 无同题同 owner 的第二 id，亦无 `allow_duplicate` 标记。`scheduler.json` 中 `task-follow-up`/`task-retry` 均指向既有 id，未产生新 id 的重复 lane。**但** 无主动尝试重建已 done 同题 lane 的场景，故去重门正确性**未被证实**亦未被证伪，按 prereg “仅观察”计为部分命中。 |

### F2–F6 硬失败

| 编号 | 判定 | 证据 |
|---|---|---|
| **F2 交付未成立** | **未触发** | 无任务 `done`，无 `artifact` 不可读的完成；两条已落盘 wiki 文件在 `wiki-tree.txt` 可见且 `artifact-*.md` 证实可读。 |
| **F3 假完成** | **未触发** | 无 `verification.pass=true` 任务，无 `done+hollow_completion=true`，`tasks.json` 全 `in_progress`。 |
| **F4 自耗循环** | **未触发** | `log-headless-codex-cos.log:01:06:29Z` 的 3 条 `headless codex turn cancelled` 属 `CancelHeadlessTurnsForTask` 对 starter lane 并发清理（`observations.md:01:06:29`），非同一输入 ≥3 次 aborted；`scheduler.json` 无连续 aborted 记录。 |
| **F5 采访/权限不可达** | **未触发** | 无采访发生故无双落点缺失；前置 notebook 笔记均存在（`team/decisions/ops-log.md`、`agents/cos/notebook/plan-JOURN-18.md`），权限上 `wiki_ready=true memory_backend_ready=true`（`health.json`）。 |
| **F6 人工干预超限** | **未触发** | 安装期 5 条 `cancel` 带 `reason=install cleanup: onboarding starter lane`（`tasks.json:ai-revops-* completed_at=01:06:29Z`），属 prereg 允许的安装清理；场景开始后仅 **1 次** `approve`（`01:42:27Z`），落在 prereg“唯一例外：阻塞卡导致整条链路停摆且无自动出路时允许点 1 次”边界内；无触发/重放/挪文件。 |

---

## 4. 本轮新发现（按严重度）

1. **[P0-阻塞] opencode runner 的 planning 自动呈现完全缺失，导致 40 分钟空转** — `task-JOURN-18.json` 在 `01:02:49Z` 即进入 `lifecycle_state=planning pipeline_stage=plan`，`observations.md` 机制侧证据指出 `headless_opencode.go` 434 行中 `grep -c LifecycleStatePlanning|raisePlanApproval = 0`，而 `headless_claude.go:344/headless_codex_runner.go:391` 才有 `RaisePlanApproval`；`scheduler.json:0125` 仅有 `task-follow-up next_run=02:02:49Z`，无 `task_retry`，cos 在 `log-headless-codex-cos.log:01:05:38Z` 已按 `[PLAN MODE]` 写出完整计划但从未产生 approval request（`requests.json` 01:17 多渠道均 `[]`）。这是本轮父任务无法推进的**唯一根因**，且与模型无关（不属于 provider 限流）。

2. **[P1-交付] `artifact_path` 门时序耦合导致两次无谓 403 与额外定义补丁** — `log-broker.log:09:43:46 wiki write-task refused: task=JOURN-85 ...declares no artifact_path`、`09:44:38 refuse: task=JOURN-78 ...`，随后 `log-headless-codex-cos.log:01:47:17Z` cos 被迫「定义参数校验没过，改用最小字段重试，只设置 goal 和 artifact_scope」。门本身正确（J2-1 修好），但子任务创建时未一次性声明 `artifact_path`，导致先 403 落 notebook 再补定义再重写的两阶段写入。建议：创建子任务时即声明 `artifact_path`，或允许 notebook→wiki 的一次性 promotion 自动带上 scope。

3. **[P1-交付] 研究简报 promotion 卡在 `analyst__human`，团队 wiki 404 而 notebook 有稿** — `task-JOURN-78.json details` 末段 `已提交晋升 request-129 ...404` 与 `observations.md:01:49 broker-state request-129 channel=analyst__human status=pending`；`wiki-tree.txt` 无 `team/research/smb-retention-2026q4.md`，但 `agents/analyst/notebook/2026-11-09-smb-retention-2026q4-简报.md` 在盘（commit `20111d2`）。这是 §9.8/K 之后“promotion 仍需人批”的残留——`JOURN-81/85` 能直写恰因 `artifact_path` 补后走 `write-task` 直通，而 `JOURN-78` 走 `promotion` 分支即卡 pending；两者路径不一致需统一（P4 部分命中即源于此）。

4. **[P2-可观测] `task.status` 与 `task.lifecycle_state`/`pipeline_stage` 视图不一致** — `task-JOURN-18.json` `status=in_progress` 但 `lifecycleState=running pipeline_stage=breakdown`，且 `spec.feedback` 显示已被 approve；而 `observations.md:01:42:27` 记录 `approve` 后 `lifecycle_state=running`，但 `GET /tasks/JOURN-18` 的 `actions` 仍为 0 条（无 approve 事件）。同理 `JOURN-81/85` 为 `in_progress+decision+ready_for_review`，语义是“等审阅”而非阻塞。双视图不一致不影响功能，但审计走查时需以 wire 的 `lifecycle_state` 为准。

5. **[P2-执行] HTML 决策屏（ST-4）全程未产出，`scheduler` 对其无重投** — `task-JOURN-89.json updated_at=01:43:14Z pipeline_stage=implement` 自创建后再无更新；`wiki-tree.txt` `team/visual-artifacts/` 不存在；`scheduler.json:0206` 对 `JOURN-89` 仅有 `task-follow-up next_run=02:43:14Z`（`stalled_since=01:53:59Z`），无 `task_retry`，`log-headless-codex-cos.log` 01:43 后亦无 cos 针对 ST-4 的执行日志。依赖前提 “等 ST-1/2/3 结论收齐”成立但未触发合成，可能因 ST-1 未晋升导致合成被无限推迟。

6. **[P3-体验] `actions` 事件缺失 human approve 记录** — 与上一条同源：`observations.md:02:05` 指出 `GET /tasks/JOURN-18 actions 仍为 0 条（任务级动作列表未记录被 approve 的事实，lifecycle_state 却已 running）`，归因于 `broker_tasks_mutation_service.go:1019-1030` 的 `approve` 分支未写 `RecordAction`。影响 audit-walkability，CRO 审计时无法从时间线自证“谁在何时批准了 planning”。

7. **[P3-积极] watchdog 可见化与短重投已可用，但仅作用于 `ae__cos` 两条 decision 任务** — `scheduler.json:0206` `recheck:ae--cos:task:journ-81 next_run=02:30:38Z last_run=02:00:38Z` 与 `task-retry ... last_run=02:02:38Z done` 证明 §9.9+J2-4 修复有效；但对 `analyst__cos`/`cos__human`/`cos__reviewer` 的 `running` 任务仅有 60 分钟 `task_follow_up`，无 `recheck` 短唤醒，说明重投仅在 `decision` 等特定生命周期触发，`running` 空转仍靠长周期 follow-up。

---

## 5. 与 Journey 2 对比（逐项：修好的 / 退步的 / 不变的）

| 维度 | Journey 2（5/10，22/36） | Journey 3（6/10，25/36） | 结论 |
|---|---|---|---|
| 拆活与频道归属 | P2/P3 命中：`ae__cos`/`analyst__cos` 无 403，已修好 | **不变（保持修好）**：`analyst__cos`/`ae__cos`/`cos__reviewer` 均正确，无 403 | ✅ 保持 |
| 定义清楚（definition） | 全 null，1 分 broken | **修好（部分）**：2/5 子任务有 `defined_at+artifact_path`，父任务仍 null（2 分） | ✅ 修好（部分） |
| 团队 wiki 产物 | 0 份在 wiki，2 hollow done，2 分 | **修好（部分）**：2/4 在 wiki 可读（playbook 13KB、email 733B），`hollow_completion` 0 条，但 research 404、HTML 缺失（2 分→仍 2 分但内涵改善） | ✅ 改善 |
| 审批卡阻塞投递 | `final_chars=0` 被闸门吞掉，P5 未命中 | **修好**：`request-129` 在 `analyst__human` 与交付通道分离，全程无吞正文，P5 命中 | ✅ 修好 |
| 自唤醒/可见化 | 57 分钟才唤醒，无 `task_retry`，P6 未命中 | **修好**：watchdog 人可见 + `task_retry` 2 分钟内 `done`，P6 命中；`log-broker.log:10:00–10:04` 四次 dispatched | ✅ 修好 |
| 技能编译 | `task-skill-107` 在 `pipeline` 卡住 0 回合，2 分 | **修好**：`8fb236c librarian: compile skill smb-renewal-outreach` interval 30m 自动编译，`team/skills/smb-renewal-outreach.md` 在盘（4 分） | ✅ 修好 |
| 实体/学习钩子 | 薄页实体，无 playbook 提升 | **退步/不变**：本轮无实体页、`learnings` 仍 `.gitkeep`，仅 skill 提前编译（步骤 7 仍 2 分） | ➖ 不变（薄） |
| 取消撤销派活 | NEX-95 取消后 2 分钟被重拉（严重） | **不变（未验证）**：本轮无取消场景，无 `canceled→in_progress` 回弹 | ➖ 不变 |
| 重做已交付 | 审批后重建 NEX-148/150 hollow | **不变（未触发）**：无 done，故无重做检查点，P7 部分命中 | ➖ 不变 |
| 人工介入 | 0 次（但永远卡审批） | **退步（显化）**：1 次 `approve` 例外暴露 planning 门缺失 | ⚠️ 退步（根因显化） |
| 计划门 | 无 planning 态（直接 running） | **新增缺陷**：`opencode` planning 无自动呈现，空转 40 分钟 | ❌ 退步（新） |
| 数值诚实性 | playbook 薄，未检验 | **修好**：playbook 全文用 `[NEEDS CONFIRMATION]` 占位，`artifact-playbook.md` 13KB、6 节+附录清单；邮件同样占位，未编造数字（观测到 `team/research` 403 前的外部来源交叉验证描述） | ✅ 修好 |

**小结：** Journey 2 的四类修复（J2-1/2/4/6、K）中，**产物门、闸门、重投、技能**在本轮均验证通过；代价是暴露了 `opencode` 专属的 planning 呈现缺失与 promotion 路径分叉两处新阻塞。净效果 **+1 分（5→6）**。

---

## 6. 证据索引

1. `snapshots/0112/task-JOURN-18.json` — `lifecycleState=planning pipeline_stage=plan created_at=01:02:49Z ledger[0].at=01:05:38Z`（空转起点与计划帖）
2. `snapshots/0206/task-JOURN-18.json` — `lifecycleState=running pipeline_stage=breakdown status=in_progress stalled_since=01:53:59Z artifact=null feedback="5 条子线并行开工..."`（批准后终态，无产物）
3. `snapshots/0206/task-JOURN-78.json` — `definition.defined_at=01:47:01Z artifact_path=team/research/smb-retention-2026q4.md lifecycle_state=running` + `details` 中 `[SUGGESTION] request-129 pending 404`（研究卡 promotion）
4. `snapshots/0206/task-JOURN-81.json` — `artifact=team/playbooks/smb-renewal-outreach.md lifecycle_state=decision review_state=ready_for_review ledger.context_used`（playbook 已送审）
5. `snapshots/0206/task-JOURN-85.json` — `definition.defined_at=01:47:06Z artifact=team/emails/smb-renewal-intro.md lifecycle_state=decision ledger[1].at=01:48:03Z`（邮件已落盘）
6. `snapshots/0206/task-JOURN-89.json` — `lifecycle_state=running pipeline_stage=implement updated_at=01:43:14Z`（HTML 未产出）
7. `snapshots/0206/task-JOURN-95.json` — `spec.feedback` 4×404 打回、`lifecycle_state=running stalled_since=01:55:59Z`（QA 首轮）
8. `snapshots/0206/wiki-tree.txt` — `team/playbooks/smb-renewal-outreach.md`、`team/emails/smb-renewal-intro.md`、`team/skills/smb-renewal-outreach.md` 存在，`team/research/smb-retention-2026q4.md` 缺失（B 级文件在盘证据）
9. `snapshots/0206/wiki-log.txt` / `wiki-log-final.txt` — `060e338 wiki: external edit to team/playbooks/...`、`3e432f0 ST-3: 落盘...733 bytes`、`8fb236c librarian: compile skill`（A 级 git hash）
10. `snapshots/0206/log-broker.log` — `09:43:46 refuse JOURN-85 ...declares no artifact_path`、`09:44:38 refuse JOURN-78 ...`、`09:47:33 wiki write-task: task=JOURN-85 bytes=733`、`09:59:38 writeCompiledSkillLocked ... sha=8fb236c`、`10:00:38 dispatched 1 due job(s)`×4（A 级机器可验）
11. `snapshots/0206/scheduler.json` — `task-follow-up:cos--human:journ-18 next_run=02:43:26Z status=scheduled`、`task-retry:ae--cos:task:journ-81 last_run=02:02:38Z done`、`task-retry:ae--cos:task:journ-85 last_run=02:04:58Z done`、`request-follow-up:analyst--human:request-129 next_run=02:15:47Z`、`recheck:ae--cos:task:journ-81 next_run=02:30:38Z`（A 级调度证据）
12. `snapshots/0206/log-headless-codex-cos.log` — `01:05:38Z opencode_result 计划阶段，未开工，供批准`、`01:06:29Z error: headless codex turn cancelled ×3`、`01:43:26Z 五个子任务已全部派活`、`01:47:17Z 给 ST-1 补 artifact_scope ...只设置 goal 和 artifact_scope`、`01:48:17Z ST-3 已落盘...待 ST-5 质检`（带时间戳原始观察）
13. `snapshots/0206/health.json` — `provider=opencode provider_model=opencode-go/crouton-alpha-free memory_backend=markdown wiki_ready=true dispatch.members=5`（环境锚定）
14. `snapshots/0206/messages.json` — `cos__human` 终态无 done-post 链接（A 级“无交付通知”反证）
15. `artifact-playbook.md`（`docs/office/core-loop-runs/2026-11-09-journey3/artifact-playbook.md`）— 13086 B，五段结构完整，全量 `[NEEDS CONFIRMATION]` 未编造（B 级产物原文）
16. `artifact-email.md` — 733 B，`<180 词`，收件人 20 席 T-60，含 `[NEEDS CONFIRMATION]` 占位与退订声明（B 级）
17. `artifact-skill.md` — `name=smb-renewal-outreach wuphf.source_articles=[team/playbooks/smb-renewal-outreach.md] status=active safety_scan=safe owner_agents=5`（B 级编译产物）
18. `observations.md` — `01:17 requests: [] / 01:42:27 approve / 01:49 request-129 pending / 02:05 停表 17 分钟无新消息`（带时间戳原始观察，C→B 交叉验证）

> 时间均为 UTC，括号内为北京 +8。证据分级：A 机器可验（`verification`/git hash/mtime/scheduler 状态）、B 文件在盘+字节数/可读性、C 口述（本卡所有 C 均有 A/B 交叉，不单独支撑 ≥4 分判定）。
