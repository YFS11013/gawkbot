# Journey 1 评分卡 — 2026-11-07（core-loop 九步）

> 评分时点：**2026-11-07 14:08（本地 UTC+8）**，快照 `snapshots/140420`（14:04:20 UTC+8）。两个任务仍 `running`，定时器 14:58/15:16 未到；邮件仍是 notebook-only。详见 §6。

---

## 1. 判定摘要（≤10 行）

- **整体：6 / 10**，相对 **v1 的 5/10 微升（+1）**。
- **一句话根因：** wiki 可读性与诚实生命周期已修（不再 fake-done / REDACTED），但**人给任务的 wiki 授权缺口**把 4 个工件卡住 20 分钟、并导致两段长空转，续约邮件与实体学习仍未落库。
- **F2–F6：** F2 未命中（未 fake-done）、F3 未命中、F4 未命中、F5 未命中（无采访可分发的空集，按预注册不计硬失败）、F6 未命中（仅一次允许的 `save to wiki` + 已登记的开跑前清理）。
- **升分项：** REDACTED 消失、fake-done 消失、plan-gate 消失；**未升分项：** 采访双投递、实体/引用图、邮件 wiki 化、空转重投。

---

## 2. 九步评分表

| 步骤 | 分数 | 判定 | 证据引用（≥3 分必须含 A 级） |
|---|---|---|---|
| **1 创建任务** | **4** | **delivered** | **A:** `snapshots/140420/tasks.json` 中 `JOURN-22` `created_at=2026-11-07T05:40:54Z` / `status=in_progress` / `owner=cos`；`activity-JOURN-22.json:2-9` `action-34 lifecycle human 05:40:54Z`；`messages-cos__human.json:10-14` `msg-23 system Issue created: JOURN-22 05:40:54Z`。**A:** `snapshots/140420/task-JOURN-22.json:task.created_at=05:40:54Z` 与首次读取 13:41:32 仅差 ~38s，任务立即可见。 扣 1 分：`details` 含逐字重复段（composer 拼接），频道为 `cos__human` DM 而非共享频道。 |
| **2 定义清楚** | **3** | **partially** | **A:** `snapshots/140420/task-JOURN-22.json:task.definition` `goal="Land all three upcoming renewals..."` / `success_criteria[4]` / `defined_at=2026-11-07T05:43:24Z`（机器可验，`definition` 非空）。**C（缺口）：** `messages-cos__human.json` 20 条中无任何一条 bot 复述三要素（Goal / Deliverables+格式 / Success criteria），`msg-39 05:52:26Z` 与 `msg-40 05:52:34Z` 仅复述「briefs+playbook ready」。按 `prereg.md §3 排除项 1` 改判口径：任务模型无 `Definition` 字段，改看是否在聊天复述三要素——**未复述**。未出现合并采访（`observations.md:13:52` 间记录 0 条采访）。 |
| **3 上下文与授权采访** | **2** | **broken** | **C/B:** `messages-cos__human.json` 无采访落「任务聊天 + Inbox」双投递；`observations.md 13:49` 12 条 pending 均为 onboarding 播种期（`05:05–05:08Z`）的陈旧 `approval/freeform/interview`，与 JOURN-22 无关，13:49 清空后亦无新采访产生。**A（前置笔记存在但不足以抵消）：** `wiki-tree.txt:15-17` `agents/cos/notebook/2026-11-07-renewal-briefs-playbook-draft-journ-22.md 1680B 2026-11-07 13:44:55` 在任务创建后；`observations.md 13:45:30` `.reads/reads.jsonl 6394B` 已更新。未吸收 `Dana Whitfield` 等回答（ae 笔记标记 `[NEEDS CONFIRMATION]`）。按 `prereg.md §5` 未出现评审意义上的采访。 |
| **4 拉起团队** | **4** | **delivered** | **A:** `snapshots/140420/task-JOURN-24.json:parent_issue_id=JOURN-22` / `owner=ae` / `created_at=05:43:30Z` / `lifecycle=running`；`activity-JOURN-22.json:18-28` `sub_issue JOURN-24 title="Draft tailored renewal emails..." actor=cos 05:43:30Z`；`health.json:dispatch.members=5`（`cos·revops·analyst·ae·reviewer` 全量，符合 `prereg.md §1 团队`）。**A:** `tasks.json` 中 `JOURN-22 cos running` 与 `JOURN-24 ae running` 依赖边真实（`parent_issue_id`）。扣 1 分：仅拆 1 个子任务覆盖 3 封邮件与 QA，未拆 brief/playbook 子任务，`JOURN-22.spec.feedback 06:01:55Z` 自述「Criteria 1&4 tracked under JOURN-24」但 `JOURN-24` 未再拆 `@reviewer` QA 子任务。 |
| **5 专注执行** | **3** | **partially** | **A（前置笔记）：** `wiki-tree.txt` `agents/cos/notebook/2026-11-07-renewal-briefs-playbook-draft-journ-22.md 1680B 13:44:55` / `agents/ae/notebook/renewal-email-drafts-journ24.md 5126B 13:45:57`（均 `≥ task.created_at 05:40:54Z`）；`broker.log 13:44:55/13:45:57` 间无错误仅后续授权拒。**A（context_used 行）：** `task-JOURN-22.json:task.ledger[0].context_used` `wiki:team/getting-started/* ×4`（`channels.md/connecting-your-work.md/working-with-agents.md/your-context-graph.md`）；`task-JOURN-24.json:ledger[0].context_used` `wiki:team/getting-started/the-wiki.md` 等 + `learning:dd728beb-1936-48df-923f-0aab25e24883`（机器可验）。**C（缺陷）：** 上下文均为通用 onboarding 页，未见 `team/accounts/` 或 `team/playbooks/` 检索行；`observations.md 13:44:23` `wiki write-task refused: task JOURN-22 declares no artifact_path` 导致跨回合退回 notebook；`j1-sampler.log 13:45:33–14:08` 显示 56 分钟 `events=5→6` 无增长的空转期，健康但无工作。 |
| **6 交付物** | **3** | **partially** | **A（4 个 wiki 工件，已落库可读）：** `wiki-tree.txt` `team/accounts/acme-corp.md 1681B 14:01:29` / `team/accounts/brightline.md 1555B 14:01:32` / `team/accounts/corti-labs.md 1838B 14:01:36` / `team/playbooks/renewal-outreach.md 2604B 14:01:41`；**A（git 可验）：** `git log --oneline` `cf96d59 / 78c00e8 / 328f6c3 / 9a7f6f6`（`Publish ... per human 'save to wiki' authorization`）作者 `cos@wuphf.local`，`git show --stat 9a7f6f6` 50 行插入；`index/all.md 14:01` 2340B 已索引，`cat team/accounts/acme-corp.md` 首节含 `Account snapshot / Situation / Risk drivers` 无 `[REDACTED]`。**A（done-post 无忧反而扣分）：** `messages-cos__human.json:msg-47 06:01:55Z` / `msg-48 06:02:00Z` 仅含路径文字 `team/accounts/` 与 `team/playbooks/renewal-outreach.md`，**无可点击 wiki 链接**、无 Inbox 通知；`tasks.json` 两任务仍 `in_progress/running` 未声明 `done`，`Verified` 徽章未翻绿（诚实，未 fake-done）。**B（缺口）：** 3 封续约邮件仅在 `agents/ae/notebook/renewal-email-drafts-journ24.md 5126B 13:45:57`（**不在 wiki**），`JOURN-24 updated_at 05:49:01Z` 后未再更新，`wiki-tree.txt` 14:04 无 `team/emails/` 或 `team/accounts/*-email*.md`。 |
| **7 确定性学习钩子** | **2** | **broken** | **A（半成功：playbook 草稿）：** `team/playbooks/renewal-outreach.md 2604B 14:01:41` 含 `Purpose / Inputs required / Steps 1–6 / Grounding rules / Roles / Lessons learned`，结构化可复用（见 §2 行内抽样）。**A（实体缺口，机器可验）：** `wiki-tree.txt` `team/entities/.graph.jsonl 360B` + `team/entities/companies-*.facts.jsonl` 仅 `crm-pipeline/no-crm` 两条（`ai-revops-2` 遗留，`05:19Z`），**无 `acme-corp/brightline/corti-labs` 事实文件**；`team/companies/` 仍仅 `crm-pipeline.md/no-crm.md`，`index/all.md` 无 `Acme/Brightline/Corti` 实体条目；`git log --oneline 14:01 之后` 无 `archivist: entity article companies/*`；**无** `[[wikilink]]` 互链、**无** infobox、**无** `References/Footnotes→tasks/artifacts`。`observations.md 14:03` 已记「entity 未生成」。B5 维基观感：`team/accounts/*.md` 可读但非 Wikipedia 形态（缺 infobox/TOC/蓝红链/References）。 |
| **8 技能与策略已激活** | **3** | **partially** | **A（技能已编译分发）：** `wiki-tree.txt` `team/playbooks/.compiled/renewal-outreach/SKILL.md 2158B 14:01:42` + `team/skills/renewal-outreach.md 2366B 14:01:42`；**A（git）：** `git log --oneline` `d808541 archivist: compile playbook renewal-outreach` / `0b2f18e librarian: compile skill renewal-outreach`；`broker.log 14:01:43` `writeCompiledSkillLocked: compiled skill created name=renewal-outreach slug=renewal-outreach owner_agents=5`；`team/skills/renewal-outreach.md` 头 `owner_agents: [ae, analyst, cos, reviewer, revops]` 可见于 Skills 面。**B/C（策略缺口）：** 无原子 `policies` 落盘证据（`team/policies/` 未在 `wiki-tree.txt` 出现）；未执行旅程第 7 步的「`never email renewals on Fridays` → 即时策略」验证，`messages-cos__human.json` 20 条无策略反馈与 `policy_created` 事件（`unproven`）。 |
| **9 复用与去重** | **2** | **broken / unproven** | **C（本轮仅 Journey 1，无次生任务，故无复用观测）：** `tasks.json` 仅 `JOURN-22/24 + ai-revops-2`，无第二相似任务；**A（去重初判：未重复但也未验证更新路径）：** `git log --oneline` 中 Acme/Brightline/Corti `Publish ... brief` 各 1 次，无更新提交；`wiki-tree.txt` 无重复 `acme-corp-2.md` 等。若有下一任务，缺 `playbook.executions.jsonl` 追加与 `team/accounts/*.md` 就地更新的演示（`team/playbooks/renewal-outreach.executions.jsonl` 未在 `wiki-tree.txt` 出现）。按 `prereg.md §7` 本次第 9 步仅作「更新而非重复」初判，记 `unproven`。 |

> 证据分级：A=机器可验（git sha / 文件大小+mtime / JSON 字段时间戳），B=文件在盘（路径+字节数），C=口述/截图。`prereg.md §8` 要求 ≥3 分至少一条 A，本表已满足。

---

## 3. F2–F6 逐条（命中/未命中 + 证据 + 为什么）

| 编号 | 判定 | 证据与理由 |
|---|---|---|
| **F2 交付未成立** | **未命中** | 充要条件是 `done` 且无 wiki artifact / 不可读 / `[REDACTED]` / 链接不在 done-post。**事实：** `tasks.json` 与 `task-JOURN-22/24.json` 在 14:04 快照均为 `lifecycle=running` / `status=in_progress`，未声明 `done`；`activity-*.json` 无 `done` 事件。4 个 wiki 工件已可读落库（§2 步骤 6），无 `[REDACTED]`（`cat team/accounts/acme-corp.md` 验证）。done-post 链接虽仅为路径文字而非超链，但任务未 `done`，故 F2 不触发。**注意：** 若以「部分交付」计，3 封邮件仍 `notebook-only`（`agents/ae/notebook/renewal-email-drafts-journ24.md 5126B`）未 wiki 化，属交付不完整但非 F2 硬失败。 |
| **F3 假完成** | **未命中** | 充要条件是 `Verified` 通过而机器校验没过，或无校验却 `done`。**事实：** `task-JOURN-22.json` 与 `task-JOURN-24.json` 均无 `verification` 字段（`verification_kind/spec` 缺失，`TaskPostRequest` 原生缺口见 `observations.md 13:53`），`review_state=pending_review`，无 `Verified` 翻绿；`broker.log` 无 `verified=true`。任务诚实地停在 `in_progress`，未出现 v1 的 40 秒 fake-done。 |
| **F4 自耗循环** | **未命中** | 充要条件是同一输入连续 ≥3 次 `aborted`，或 `stalled=8` 被取消且无可读原因。**事实：** `headless-codex-cos.log` 仅 1 次 `error: headless codex turn cancelled so newer queued work can run 13:49:33`（`human-priority: cancelling active turn after 5.654s so the bot absorbs the human message`，因处理 12 条 stale 请求产生的 `you` 消息触发重排，非自耗）。`j1-sampler.log` 显示 `status=in_progress life=running` 连续 200+ 次采样无 `aborted`。`broker.log` 无 `stalled=8`。 |
| **F5 采访/权限不可达** | **未命中**（但为**空集未命中**） | 硬失败定义：提问没有同时落「任务聊天 + Inbox」，或无任务前置 notebook 笔记。**事实：** `messages-cos__human.json` 在 JOURN-22 生命周期内 **无采访提问**（0 条），故无「提问未双投递」可判；`agents/cos/notebook/2026-11-07-renewal-briefs-playbook-draft-journ-22.md 1680B 13:44:55` 与 `agents/ae/notebook/renewal-email-drafts-journ24.md 5126B 13:45:57` 前置笔记存在。按字面合同 F5 未触发，但按旅程意图 `Journey 1 step 3` 应出现的 `Dana Whitfield / wiki as data source` 合并采访**未出现**（`observations.md 13:45:24` cos 亦自述被 pending 阻塞），记为步骤 3 的 `broken` 而非 F5 硬失败。 |
| **F6 干预超限** | **未命中** | 允许的人工干预仅 `approve×1` + `补参×1`（`prereg.md §5`），其余触发/重放/挪文件即命中。**事实：** 人工仅 `13:49` 12 条 stale request 的批量 `reject/hold/answer_directly`（`messages-cos__human.json:msg-27–38`），`prereg.md §4` 已登记为开跑前清理，不计入 F6；`13:20` 取消 4 条 starter lanes 与 `13:16` API onboarding 亦已登记为已知偏离；任务内仅 `14:00:49 msg-41 you | save to wiki` 一次补参，符合配额。无触发/重放/挪文件操作。 |

---

## 4. Delta vs v1（`.icp-eval/` 5/10 基线）

| v1 失败点 | 本轮判定 | 证据 |
|---|---|---|
| **redacted 指针（artifact 不可读/`[REDACTED]`）** | **FIXED** | `cat team/accounts/acme-corp.md / brightline.md / corti-labs.md` 均含完整 `Account snapshot / Situation / Risk drivers / Recommended approach / Email angle`，`team/playbooks/renewal-outreach.md` 2604B 含 `Steps 1–6` 全量，无 `[REDACTED]`；`index/all.md` 已索引。v1 的 `team/accounts` 不可读已消失。 |
| **dead Accept（评审门点不动）** | **IMPROVED** | JOURN-22 `review_state=pending_review` 未卡死，`spec.feedback` 三次追加（`05:44:56 / 05:52:30 / 06:01:55Z`）均成功写入；`@reviewer` QA 标注于 `spec.feedback` 与 `ae.log`「Ready for @reviewer QA before send」，虽未演示 Accept 点击，但通道未阻塞。v1 的点击无响应未复现。`unproven` 是否一键 Accept 仍待 Journey 2 验证。 |
| **fake-done（无校验却 done，40s 假完成）** | **FIXED** | 两任务在 `14:04:20` 快照仍 `running/in_progress`，`activity-*.json` 无 `done`，`broker.log` 无 `verified` 翻绿。v1 的无校验 done 已不再发生，系统诚实等待 `JOURN-24` 与 `@reviewer`。 |
| **plan gate 混乱（plan 审批卡循环）** | **FIXED** | `task-JOURN-22.json:definition.defined_at=05:43:24Z` 在 `sub_issue 05:43:30Z` 之前即就绪，`ledger` 显示 `ok` 而非 `plan_approval_required`；`observations.md 13:43:24` 两条 `cos action` 已推进，无 plan-gate 阻塞。`prereg.md §3` 的散文+`SuccessCriteria` 模型已替代 v1 的 plan-gate。 |
| **计数漂移（board/dashboard 计数不一致）** | **UNPROVEN（倾向 IMPROVED）** | `health.json dispatch.members=5` 与 `tasks.json` 6 条（含 4 条 canceled）一致；`j1-sampler.log` 连续 `events=1→7` 单调递增无跳变。未执行旅程第 5 步的 `Fleet view + CRO walk` 的 board 计数对照，故记 `unproven`。 |
| **joke-only 状态（空转时只讲笑话）** | **IMPROVED** | 前期 `observations.md 13:45–14:01` 回合均产出实质 notebook（1680B + 5126B）与 wiki 4 提交，非笑话；`headless-codex-cos.log 13:45:24` 与 `13:52:34` 均为阻塞原因说明+下一步指引。扣分：14:01 后进入 56 分钟 `0 个 opencode.exe` 空转期，无笑话但亦无心跳消息，属 `joke-only` 的反向极端——`silent stall`。 |
| **audit 归因（全部 "Human"）** | **UNCHANGED（待验证）** | `activity-JOURN-22.json` 7 条中 `actor=cos 5 条 + human 1 + system 1`，归因已区分 `actor`；但 `observations.md §5 待记录` 的 audit-log 是否仍全 `Human` 未在快照覆盖（`tasks.json` 无 audit 字段导出）。记 `UNCHANGED / unproven`，需 `GET /audit` 才能定论。 |
| **meter 滞后（进度/仪表滞后）** | **IMPROVED** | `j1-sampler.log` 15s 轮询显示 `updated` 时间戳 `05:40:54→05:43:24→05:45:24→05:52:30→06:01:55Z` 与真实提交 `cf96d59 14:01:29` 误差 <1 分钟，无 v1 的数分钟滞后。但 14:01 后 `updated` 冻结在 `06:01:55Z` 至 14:08，反映的是**调度未重投而非仪表滞后**。 |

**新增失分项（v1 无，v2 新）：** `wiki write-task` 授权门（`internal/team/wiki_handlers.go:160-165` 要求 `task.Definition.ArtifactPath != ""`，`broker_intake.go:818` 不填该字段，仅 `office_eval_*` 设值）导致 `broker.log 13:44:23 wiki write-task refused: declares no artifact_path` 并迫使 `13:52 msg-39` 人补 `save to wiki`（`headless-codex-cos.log 13:45:24` 自述双阻塞：system 消息不可引用 + pending 阻塞）。直接 `POST /wiki/write` 服务端无校验，约束仅在 prompt 层（`prompt_builder.go:466`），属**策略层硬门 vs 服务层敞开**的不一致。

---

## 5. Top 3 修复建议（按影响排序，落到文件/函数）

### 1) 打通「人给任务 → wiki 产物」的授权路径（P0，阻塞循环）

- **位置：** `internal/team/wiki_handlers.go:66-165`（`write-task` vs `write` 双路径）、`internal/team/broker_intake.go:818`（`Definition` 构造处）、`internal/team/prompt_builder.go:466` / `internal/team/system_skills.go:66`（`human_request` 引用约束的 prompt 层实现）。
- **现象：** composer 创建的任务不带 `Definition.ArtifactPath`，且 composer 产生的 `msg-23 system: issue_created` 不是 `human-authored` 消息，导致 `13:44:23 wiki write-task refused`，bot 被迫在 `13:52:34` 索要 `save to wiki` 人补句，空转 20 分钟。
- **修法：** 二选一：**(a) 服务层修复**——`broker_intake.go:818` 为 `issue` 任务填充 `Definition.ArtifactPath = team/tasks/<id>/` 或 `team/accounts/` 等默认前缀，或在 `wiki_handlers.go:160` 对 `created_by=human` 的任务豁免 `ArtifactPath` 校验；**(b) 消息层修复**——将 composer 的 `issue_created` 记为 `human` 消息（或同时写入一条 `human_request` 授权锚点），使 `team_wiki_write human_request=<msgId>` 可引用。任一修法后，用 `curl -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/tasks/JOURN-22` 验证 `definition.artifact_path != ""`，并重跑 `broker.log` 不再出现 `declares no artifact_path`。

### 2) 为「回合结束但任务未完成」补调度重投，消除 14:01 后的 56 分钟 silent stall（P1，活性）

- **位置：** `internal/team/scheduler.go`（`poll=20s` 循环与 `follow_up_at / recheck_at / reminder_at` 计时器）、`internal/team/broker.go` 的 turn 调度（`human-priority: cancelling active turn` 后重排逻辑）、`headless-codex-*.log` 的 `opencode_result` 回收处。
- **现象：** `j1-sampler.log 14:01:55–14:08` `0 个 opencode.exe`，`JOURN-22 06:01:55Z / JOURN-24 05:49:01Z` 冻结；定时器分别 `14:58 / 15:16 / 15:31` 才重投，空闲 ~56 分钟。`JOURN-24` 的 3 封邮件已在 notebook 却无人将其 `team_wiki_write` 到 wiki，`JOURN-22` 等待 `JOURN-24 + @reviewer QA` 但无人驱动。
- **修法：** 在 `opencode_result` 落盘时若 `lifecycle=running` 且 `updated_at` 未推进超过 `2×poll`，立即重入调度队列（而非等 `recheck_at`）；或为父任务 `JOURN-22` 在子任务 `JOURN-24` 产出后自动派 `reviewer` QA 子回合。验证：`broker.log` 应在 `14:02` 后 1–2 分钟内出现新的 `opencode.exe` pid，`JOURN-24` 的 wiki 化提交应在 14:10 前出现（而非等 14:58）。

### 3) 补齐确定性学习钩子：邮件 wiki 化 + 实体事实/引用图 + 可点击 done-post（P1，闭环可信度）

- **位置：** `internal/team/wiki_handlers.go`（`team/accounts/` 与 `team/emails/` 路径策略）、`internal/team/entity_index.go` / `team/entities/.graph.jsonl` 与 `team/entities/*.facts.jsonl` 写入处、`internal/team/skill_compiler.go` / `internal/team/playbook_compiler.go`（`playbook.executions.jsonl` 追加）、`web/src/components/TaskChat.tsx` 或 `MessageRenderer.tsx`（done-post 链接渲染）。
- **现象：** 3 封邮件仅 `agents/ae/notebook/renewal-email-drafts-journ24.md 5126B`，`wiki-tree.txt` 无 `team/emails/`；`team/entities/` 无 `acme-corp/brightline/corti-labs` 事实；`msg-47/48` 的 `team/accounts/{acme-corp,brightline,corti-labs}.md` 为纯文本路径，非 `[[wikilink]]` 蓝链，Inbox 无通知；`team/playbooks/renewal-outreach.executions.jsonl` 未出现。
- **修法：** (a) `JOURN-24` 回合模板显式要求 `team_wiki_write` 3 封邮件到 `team/accounts/<slug>-renewal-email.md` 或 `team/emails/<slug>.md`，并将 `team/accounts/<slug>.md` 追加 `## Related artifacts` 章节 `[[team/emails/...]]`；(b) `archivist` 在 `9a7f6f6` 后追加 `fact: companies/acme-corp` 等 3 条 `facts.jsonl` 与 `.graph.jsonl` 双向边，`team/accounts/*.md` 追加 `{{infobox}}` 与 `## References` 指向 `JOURN-22/JOURN-24`；(c) done-post 改为 `All four wiki articles are live: [[team/accounts/acme-corp]] ...` 并同步写 Inbox。验证：`wiki-tree.txt` 出现 `team/emails/*.md ≥3`、`team/entities/*acme*` 事实、`index/all.md` 蓝链可点。

---

## 6. 未决事实（本评分基于 14:08 可观察状态）

- **评分时点：** 2026-11-07 14:08 本地（UTC+8），UTC 06:08。最后快照 `snapshots/140420`（14:04:20）与 `j1-sampler.log` 连续采样至 14:08:26（`status=in_progress life=running events=7 last=/cos updated=06:01:55Z`）。
- **仍 `running` 且尚未定论：** `JOURN-22`（`cos`，`pending_review`，`follow_up_at 07:01:55Z / recheck_at 07:16:55Z` 本地 15:01/15:16）与 `JOURN-24`（`ae`，`pending_review`，`recheck_at 06:58:30Z` 本地 **14:58**）。`observations.md 14:06–14:09` 记「0 个 opencode.exe 回合，办公室空闲到 14:58」。
- **邮件仍是 notebook-only：** `agents/ae/notebook/renewal-email-drafts-journ24.md 5126B 13:45:57` 已含 3 封草稿与 `[NEEDS CONFIRMATION]` 表，但 `wiki-tree.txt` 14:04 无 wiki 邮件页，`JOURN-24 updated_at 05:49:01Z` 后无更新，`@reviewer` QA 未发生。
- **尚未发生的验证：** 定时器 14:58（JOURN-24）与 15:16（JOURN-22）重投后是否自动将邮件 wiki 化、派 reviewer、翻 `Verified`；`board/dashboard` 计数对照；`audit log` 归因是否仍全 Human；`meter` 在重投后的仪表延迟；步骤 8 的 `never email renewals on Fridays` 即时策略是否可被聊天反馈即时铸造（本次未测）。
- **已知偏离（不计入 F6，按 `prereg.md §4` 登记）：** 12 条 stale request（`05:05–05:08Z` 播种）于 13:49 批量清理、`ai-revops-1/3/4 + task-skill-4` 4 条 starter lanes 于 13:20 取消、onboarding 走 `POST /onboarding/complete` API（`agents` 显式列 slug），均已在预注册登记，评分时未作扣分。
- **邮件 notebook-only 的备注：** 按 `observations.md 13:49:01` ae 自述「Ready for @reviewer QA before send」，邮件落 wiki 的缺口不影响 4 个已落库工件的 A 级可验性，但影响交易闭环完整性（见 Top 3-3）。

