# Journey 1 观察记录 — 2026-11-07

原始观察，按时间戳记，不写结论。时间为主机本地时间（UTC+8）；快照里的 `created_at`/`timestamp` 是 UTC，两者差 8 小时。

## 环境（开跑时）

| 项 | 值 |
|---|---|
| workspace | `journey1`，broker `:7910` / web `:7911`，broker pid 23420 |
| commit | `2e263f5`（二进制 13:23 构建；之后只有文档提交 `66a45af`、`9a6a3be`） |
| provider / model | `opencode` / `opencode-go/beam-free` |
| memory_backend | `markdown` |
| 团队 | `cos` `revops` `analyst` `ae` `reviewer`（/health `members=5`） |
| 既有任务 | `ai-revops-2`（analyst，done/approved，蓝图遗留）；`ai-revops-1/3/4`、`task-skill-4` 已于 13:20 取消 |
| 观测器 | `j1-sampler.ps1`（15s 轮询 `/tasks/JOURN-22` + `/activity`），日志 `.wuphf/logs/j1-sampler.log` |

## 时间线

### 13:40:54 — 任务创建（人给的入口）

`POST` 由 composer 发出，落到 **`cos__human` DM**（不是频道）。

| 字段 | 值 |
|---|---|
| id | `JOURN-22` |
| status / lifecycle | `in_progress` / `running` |
| owner | `cos` |
| created_by | `human` |
| execution_mode | `office` |
| provider | `opencode` |
| task_type / pipeline | `issue` / `issue`·`breakdown` |
| review_state | `pending_review` |
| thread_id | `msg-23` |
| created_at / updated_at | `2026-11-07T05:40:54Z` |
| due_at | `2026-11-07T06:40:54Z` |
| channel | `cos__human` |

任务载荷里**没有** `verification_kind` / `verification_spec` / `definition` 字段（预注册 §3 的排除项 1 由此而来）。`details` 与用户输入逐字一致，末尾无额外指令。

### 13:41:32 — 首次读取

- 任务已在 `/tasks/JOURN-22` 可见，owner 已是 `cos`，状态 `running`。
- 该 DM 只有 2 条消息：`cos` 的自我介绍（`05:16:00Z`，onboarding 遗留）与 system 的 `issue_created`（`05:40:54Z`）。
- 尚无采访、无计划、无 notebook 笔记。`wiki/agents/cos/notebook/` 只有旧 lane 的 3 个文件，没有 JOURN-22。

### 13:42:32 — 采样器基线

```
status=in_progress life=running owner=cos review=pending_review events=1
```

### 13:44 — 进程与日志

- broker 后代里有 **2 个 `opencode.exe`** 在跑（`74292`、`48092`），命令行均为 `opencode run --format json --pure --agent wuphf-headless --model opencode-go/beam-free`；各有一个 `cmd.exe` 包装。
- `mcp-team` 子进程 1 个（`49572`）。
- broker.log 无错误；最后一行 `13:36:09 scheduler: heartbeat (uptime=10m0s)`。
- wiki 中 `13:40` 之后没有新文件。

### 13:43:24 / 13:44:56 — cos 两次状态动作

`/tasks/JOURN-22/activity` 新增两条 `cos | action`，摘要均为任务原文 + `[in_progress]`。DM 里没有 cos 的文本消息（无采访、无计划说明）。

### 13:43:30 — 第一个子任务

`cos` 创建 `JOURN-24`，落同一个 `cos__human` DM：

| 字段 | 值 |
|---|---|
| title | Draft tailored renewal emails — Acme Corp, Brightline, Corti Labs |
| owner | `ae` |
| parent_issue_id | `JOURN-22` |
| status / lifecycle | `in_progress` / `running` |
| pipeline_stage | `implement` |
| details 摘要 | 为三个账号各写一封续约邮件；要求“按 playbook `team/playbooks/…` 调整语气” |

`details` 引用了 playbook 路径，但此刻 wiki 里还没有该 playbook（任务原文要求最后写它）。

### 13:45–13:46 — 执行中

- 子任务 `JOURN-24` 已派给 `ae`，状态 `running`。
- `JOURN-22` 仍 `running`，owner `cos`；activity 4 条。
- broker 后代有 2 个 `opencode.exe` 回合在跑（13:44 观测）。

### 13:44:55 / 13:45:57 — 两条 notebook 笔记（任务前置）

- `wiki/agents/cos/notebook/2026-11-07-renewal-briefs-playbook-draft-journ-22.md`（13:44:55，1680 B）
- `wiki/agents/ae/notebook/renewal-email-drafts-journ24.md`（13:45:57，5126 B）

cos 的笔记里逐账号写了 brief：Acme（Q3/$48k，champion 5 月离职，风险 High）、Brightline（Q3/$22k，用量 +40%，风险 Low）、Corti（Q4/$61k，两个未解决升级，风险 Medium-High），每条都列了 `Open:`（champion 姓名/替代人、用量基线、续约日期等）。

### 13:45:30 — `.reads/reads.jsonl` 更新（6394 B）

### 13:46:36 — 快照 1（`snapshots/134636`）

- 任务：`JOURN-22`（cos，running）、`JOURN-24`（ae，running，`parent_issue_id=JOURN-22`）、`ai-revops-2`（done）。
- wiki 新增两个提交：
  - `5fcc001 Renewal briefs + playbook draft (JOURN-22)` — 只改了 `agents/cos/notebook/2026-11-07-renewal-briefs-playbook-draft-journ-22.md`（31 行）
  - `8c19d89 Renewal Email Drafts — JOURN-24` — 只改了 `agents/ae/notebook/renewal-email-drafts-journ24.md`（112 行）
- `wiki/team/companies/` 仍只有 `crm-pipeline.md`、`no-crm.md`；**没有 Acme/Brightline/Corti 账号页**。
- `wiki/team/playbooks/` 仍只有 `owner-assignment.md` 与 `.gitkeep`；**没有 renewal-outreach playbook**。
- 全 wiki 中 "Acme" 只出现在 3 个 notebook 文件里。

### 13:48 — 状态

- `JOURN-22` `running`（updated 05:45:24Z）；`JOURN-24` `running`（updated 05:43:30Z）。
- 尚无采访消息、无 done-post、无 `Verified` 变化。

### 13:44:23 — wiki 写入被拒（关键）

broker.log：

```
wiki write-task refused: task=JOURN-22 slug=cos path=team/accounts/acme-corp.md:
  task JOURN-22 declares no artifact_path; pass human_request instead
```

即：bot 尝试把 Acme 的账号 brief 写进 wiki 页面，被 wiki worker 以「任务没有声明 `artifact_path`」拒绝。composer 创建的任务不带该字段 → 任务型 wiki 写入全部被拒，bot 只能退回写自己的 notebook。

### 13:45:24 — cos 回合结束（turn: ok）

`headless-codex-cos.log` 的回合结果：

> Wiki publish blocked on authorization (no human-authored chat message to cite; msg-23 is a system message) and the channel is blocked by a pending interview/approval that must be resolved before new messages. I've captured the drafted briefs + playbook to my notebook... Reply "save to wiki" once the pending request clears and I'll publish the briefs + playbook canonically.

两个阻塞点：① composer 发出的任务在消息流里记为 **system**（`issue_created`），不是人写的消息，不能作为发布授权的引用；② 频道被 pending request 阻塞。

### 13:49 — 12 条 pending 请求（开跑前遗留）

`GET /requests?channel=cos__human` 返回 12 条 `pending`，全部创建于 `05:05–05:08Z`（onboarding 播种期，早于任务）：

| id | 类型 | 内容 | 处理 |
|---|---|---|---|
| request-6/10/12/16 | approval | 创建 #outreach / #pipeline / #hygiene / #general | reject |
| request-8/26 | approval | 添加 @revops / @reviewer | reject |
| request-19/28 | approval | 晋升 wiki 文章（两篇） | reject |
| request-34/39 | freeform | CRM 平台问题 / ai-revops-1 进度 | hold |
| request-21/37 | interview | CRM 平台是什么 | answer_directly: “No CRM is connected yet. Use the wiki as the data source.” |

处理结果：pending 归零。**这是记录者做的开跑前清理**（人工干预，非任务内动作），原文已登记；副作用见下条。

### 13:49:27–13:49:32 — 请求处理产生的聊天消息（重复两次）

`cos__human` 出现 8 条消息，其中：

- `you | Put @cos's request on hold.`（同一内容出现 **2 次**）
- `you | Answered @cos's request with Answer directly: No CRM is connected yet...`（**2 次**）

即每条请求处理都在频道里写了两遍同文案的消息（重复写入）。

### 13:50 — 清理后

- 12 条 pending 清空后，出现 1 个新的 `opencode.exe` 回合（13:50 左右）。
- `JOURN-22` 仍 `running`，activity 仍 5 条；wiki 无新增提交；`team/companies/`、`team/playbooks/` 仍无任务产物。

### 13:49:01 — ae 回合结束（JOURN-24）

`headless-codex-ae.log`：

> Drafts are complete and saved to my notebook at `agents/ae/notebook/renewal-email-drafts-journ24.md`. Three tailored renewal emails: … All [NEEDS CONFIRMATION] items flagged per briefs (champion name, usage metrics, renewal dates, contacts, sender identity, escalation details). Ready for @reviewer QA before send.

发帖被拒：`live-chat-relay-post-error: request pending in this channel`（此时 12 条请求尚未清理）。产出同样只落在 notebook，没有 wiki 页面。

### 13:49:32–13:49:33 — 人消息触发回合取消

broker 日志：

```
13:49:32 human-priority: cancelling active turn after 5.654s so the bot absorbs the human message
13:49:33 error: headless codex turn cancelled so newer queued work can run
```

即我处理请求所写下的 `you` 消息被当作人消息，触发「取消当前回合、重排新回合」。新回合 `opencode.exe` pid 65180 于 13:49:33 启动。

### 13:52 — 新回合仍在跑

- `JOURN-22` / `JOURN-24` 均 `running`，activity 未增长（`JOURN-22` 仍 5 条）。
- cos 的日志停在 13:49:33；ae 的在 13:49:01。
- wiki 无新提交；`team/companies/`、`team/playbooks/` 无任务产物。

### 13:52:26 / 13:52:34 — cos 发帖（阻塞解除后）

`cos__human` 新增两条 `cos` 消息：

> Briefs + playbook are drafted and ready in my notebook (3 per-account briefs + quarterly renewal-outreach playbook). To …

> Briefs + playbook are drafted in notebook and ready. Asked the human (msg-39) to reply "save to wiki" so I can publish a…

即：内容已备好，但发布需要人回一句 `save to wiki`（新的 pending request msg-39）。`JOURN-22` updated 05:52:30Z；`JOURN-24` 停在 05:49:01Z；此刻 0 个回合在跑。

### 13:53 — 结构性发现（代码定位）

`internal/team/wiki_handlers.go:160-165`：任务型 wiki 写入要求 `task.Definition.ArtifactPath` 非空，否则报 `declares no artifact_path; pass human_request instead`。

- composer 创建的任务不带 `Definition`（`TaskPostRequest` 无该字段，`TaskPostRequest` 只有 `Details` + `VerificationKind/Spec`）；
- `broker_intake.go:818` 走 intake 的任务会写 `Definition{Goal, SuccessCriteria}`，但也**不含 `ArtifactPath`**；
- 唯一设置 `ArtifactPath` 的调用点都在 `office_eval_*` 测试/评测代码里。

因此「人给的任务 → bot 写 wiki 产物」在正常路径上不可达；bot 只能退回写自己的 notebook，或用 `team_wiki_write` + `human_request`（需要引用一条**人写**的消息 id）。composer 的任务消息在流里是 `system`（`issue_created`），不能被引用 —— 这就是 cos 说 “no human-authored chat message to cite; msg-23 is a system message” 的由来。

补充核对（13:56）：

- `/wiki/write-task`（`wiki_handlers.go:66+`）是任务作用域写入，唯一授权是 `task.Definition.ArtifactPath`；授权失败时的提示语 “pass human_request instead” 指向另一条路。
- `/wiki/write`（`wiki_handlers.go:24-64`）**服务端没有任何授权检查**——任何持 token 的调用方都能写任意路径。也就是说 “必须引用人写消息” 是 **prompt/策略层**的要求（`prompt_builder.go:466`、`system_skills.go:66`），不是 broker 的硬门。
- 因此准确表述是：任务作用域写入被人给任务的缺字段挡住；直接写入在服务端是敞开的，但 bot 被策略要求先拿到一条人写的请求消息，而 composer 的任务消息是 `system`。cos 因此停在「等人回 save to wiki」。

如果人回一句，cos 应能通过直接写入把 4 个产物落到 wiki（届时记录实际路径与提交 sha）。

### 14:00:49 — 人回 "save to wiki"（预注册允许的那一次补参）

`cos__human` 出现 `you | save to wiki`（`06:00:49Z`）。

### 14:01:55 / 14:02:00 — cos 发布完成

> All four wiki articles are live now: 3 per-account briefs (team/accounts/) + the quarterly renewal-outreach playbook (team/playbooks/renewal-outreach…)
> Done. JOURN-22 stays in_progress pending @ae's email drafts (JOURN-24) and @reviewer QA. All four wiki articles published.

### 14:01 — wiki 提交与产物（可读、有结构）

| 提交 | 内容 |
|---|---|
| `cf96d59` | Publish Acme Corp renewal brief per human 'save to wiki' authorization |
| `78c00e8` | Publish Brightline renewal brief |
| `328f6c3` | Publish Corti Labs renewal brief |
| `9a7f6f6` | Publish quarterly renewal-outreach playbook |
| `d808541` | archivist: compile playbook renewal-outreach |
| `0b2f18e` | librarian: compile skill renewal-outreach |

文件（大小 / 时间）：

- `team/accounts/acme-corp.md` 1681 B（14:01）
- `team/accounts/brightline.md` 1555 B（14:01）
- `team/accounts/corti-labs.md` 1838 B（14:01）
- `team/playbooks/renewal-outreach.md` 2604 B（14:01）
- `team/playbooks/.compiled/renewal-outreach/SKILL.md` 2158 B（14:01）

内容抽样：Acme brief 有 Account snapshot / Situation / Risk drivers / Recommended approach 分节；playbook 有 Purpose / Inputs required /（后续步骤）分节。均为人可读的 markdown，无 `[REDACTED]`。

### 14:01:43 — 技能编译

broker.log：`writeCompiledSkillLocked: compiled skill created name=renewal-outreach slug=renewal-outreach created_by=librarian sha=0b2f18e owner_agents=5`。

### 14:01:42 — 一条 gbrain 报错（与 markdown 后端不一致）

`source_capture: write note source "note-renewal-outreach-…": gbrain call put_page … EOF`。memory_backend 已钉 markdown，但 source_capture 仍走 gbrain 并失败。

### 14:03–14:04 — 状态

- `JOURN-22` `running`（updated 06:01:55Z）；`JOURN-24` `running`（updated 05:49:01Z，未更新）。
- 0 个回合在跑；两个任务都是「running 但无人执行」。
- `JOURN-24` 的三封邮件仍只在 `agents/ae/notebook/renewal-email-drafts-journ24.md`，wiki 无对应页面。
- 尚无 done / Verified / review 事件；`team/accounts/` 之外没有 Acme/Brightline/Corti 的实体页（步骤 7 的 entities 未生成）。

### 14:06–14:09 — 停滞（可观察到的 dead zone）

- 0 个 `opencode.exe` 回合；`JOURN-22` 停在 `06:01:55Z`，`JOURN-24` 停在 `05:49:01Z`。
- DM 最后两条消息都是 cos（14:01:55、14:02:00），此后无任何新消息。
- wiki 最后一条提交 `0b2f18e`（14:01）。
- 两个任务的定时器（UTC → 本地）：

| 任务 | follow_up_at | recheck_at | reminder_at |
|---|---|---|---|
| JOURN-24 | 06:43:30Z（14:43） | 06:58:30Z（**14:58**） | 07:13:30Z（15:13） |
| JOURN-22 | 07:01:55Z（15:01） | 07:16:55Z（**15:16**） | 07:31:55Z（15:31） |

即：除非定时器触发重投，办公室会空闲到 **14:58**（约 56 分钟）。ae 在 13:49 的回合里产出草稿后没有把任务收口，broker 也没有为「回合结束但任务未完成」重投。

### 14:14 — 更正：任务其实**有** Definition（预注册排除项 1 的前提不成立）

`snapshots/140420/task-JOURN-22.json` 的 `definition` 非空：

```
goal: "Land all three upcoming renewals (Acme Corp $48k Q3, Brightline $22k Q3, Corti Labs $61k Q4) by producing ..."
success_criteria[4]: 3 emails / 3 briefs to team/accounts/ / playbook to team/playbooks/renewal-outreach.md / QA by @reviewer
defined_at: 2026-11-07T05:43:24Z
```

即 cos 在 13:43:24 把 composer 的散文任务转成了结构化 Definition。`JOURN-24` 的 `definition` 为 `null`。

**这意味着：**

- 预注册 §3 排除项 1 的前提（「任务模型没有该字段」）**不成立**——`TaskPostRequest` 确实不收 definition，但 office 可以通过 action 写；我 13:41 的那次读取（定义写入前）得出了错误结论。
- 缺的不是 Definition，而是 `artifact_path`：`write-task` 的授权门读的就是这个字段，而 Definition 的构造处（`broker_intake.go:818` 一带）只写 Goal/SuccessCriteria。
- 预注册不改（跑完不动判据）；这条更正记在观察里，替总时按「Definition 存在、但无 artifact_path」判步骤 2。

### 14:44–14:58 — recheck 触发，办公室自己醒了（推翻「终局」判断）

- **14:44:53 / 14:45:45** ae 回合：把三封草稿投递到 `#ae__human`（`msg-57`）并 @cos（`msg-58`）。日志末行：
  `live-chat-relay-post-error: channel access denied` + `@ae has no write access to #cos__human; @cos needs to relay submit_for_review or grant channel access`。
- **14:46:09** `JOURN-24 [review]`；**14:46:09–14:54:06** reviewer QA → **14:54:06** `JOURN-24 [done]`。
- **14:54:13** 新任务 `JOURN-75`（owner `ae`，title「Send approved renewal emails — Acme Corp, Brightline, Corti Labs」，`definition=null`）。
- **14:54:07** `JOURN-22` updated（`06:54:07Z`），仍 `running` / `pending_review`。
- **14:58:28 / 14:58:37** `JOURN-24 [review]` → `[done]`；**14:58:37** `status=done`、`review_state=approved`、`verification=not_required`。
- **14:58:54** cos：`@ae — JOURN-24 drafts approved (QA CLEAN), task closed. Next lane is JOURN-75: send the three approved emails…`

**这就是「回合结束未收口 → 无重投」的反证**：14:54 的 `follow_up_at`/`recheck_at` 确实把任务唤醒并收口了，但代价是 **从 13:49 到 14:54 的 65 分钟空转**。看门狗只重置 bot 状态、不重投；重投来自定时器。

### 14:58 — 实体页由 archivist 自动生成（步骤 7 补上一半）

wiki 新提交：

| 提交 | 内容 |
|---|---|
| `b07bd77` | archivist: entity article `people/cos`（1 fact） |
| `e31ec4a` | archivist: entity article `companies/acme-corp`（1 fact） |
| `a6f36d7` | archivist: entity article `companies/corti-labs`（1 fact） |
| `86997ca` | archivist: entity article `companies/needs-confirmation`（2 facts） |

`team/companies/acme-corp.md` 1291 B（14:58），front matter 有 `last_synthesized_sha` / `fact_count_at_synthesis`，正文是模板化摘要（`Kind: company`、`Tasks: JOURN-24`）——**有页面、有事实来源链，但没有 `[[wikilink]]`、没有 infobox、内容极薄**。Brightline 未生成实体页（只有 acme / corti / needs-confirmation）。

### 15:09 — JOURN-75 无法执行，且报不出去

`headless-codex-ae.log 14:57:53`（原文）：

> JOURN-75 无法执行发送。核对草稿与环境后发现三个阻塞：1) 收件人邮箱与发件人身份均未确认——三份草稿的 To/From 仍为 `[NEEDS CONFIRMATION]` 占位符；2) 本办公室未连接任何邮件集成（`team_action_connections` 返回空）；3) Corti Labs 按 playbook 标记为 HELD。… 另外，当前 @ae 对 #cos__human 无写入权限（403），以上阻塞未能写入 Issue 评论/Interview，仅在此汇报。

随后：

- `14:57:53` activity：`turn | ok (no durable trace: no message, no task mutation)`；
- `15:09:53` activity：`action | no visible activity on JOURN-75 for 11m0s — marked stalled`；
- 系统消息 `15:09:53`：`@ae has gone quiet … for about 11 minutes`。

即：**子任务 bot 没有向父频道/负责人写字的权限，阻塞报告只能落在自己的 headless 日志里**；对协作层来说这一回合等于没发生（no durable trace），11 分钟后被标 stalled。

## 待记录

- 采访是否出现、落在哪里（任务聊天 + Inbox）；
- 是否有 Definition 复述（三要素）；
- 子任务 / 依赖边；
- 回合活动的 context 行（`learning:X` / `wiki:Y`）；
- 交付物（wiki artifact）与 done-post 链接；
- `Verified` 与机器校验的一致性。
