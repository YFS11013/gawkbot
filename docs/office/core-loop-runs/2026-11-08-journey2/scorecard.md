# Journey 2 评分卡 — 2026-11-07（core-loop 九步）

> 评分时点：**19:16**（本地 UTC+8）。场景起点 **18:55:55**（`NEX-98` 创建），停表 **19:16**（办公室自 18:59 起无任何回合，19:11:54 的 recheck 只记了一条 alert，没有派活）。快照 `snapshots/1912`、`snapshots/1925`。

---

## 1. 判定摘要（≤10 行）

- **整体：5 / 10**（步骤合计 **22 / 36**；预注册命中 **2 / 6**：P2、P3 命中）。
- **一句话根因：** 拆活、派活、配对 DM 全部正确（9 秒内拆出三条 owner 明确的子 Issue，`ae__cos`/`analyst__cos` 写入无 403），但**产物没有落进团队 wiki**：两封 `hollow_completion=true` 的 done、playbook 只在 cos 的 notebook 而 `artifact` 指向不存在的 `team/playbooks/…md`，最后 cos 停在「等你点 Approve」的 promotion 卡上，父 Issue 永远 `in_progress`。
- **反证环境因素：** 本轮模型换成 `opencode-go/gpt-5.6-luna`（beam-free 被限流），回合 15–40 秒稳定返回，`final_chars` 有正文——不是模型没输出。
- **P1 未命中：** 父 Issue 与三条子 Issue 的 `definition` 全为 `null`，没有任何 `artifact_path` 声明。
- **P2 命中（命名不同）：** 子 Issue 频道是 `ae__cos` / `analyst__cos`，即 creator⇄owner 的 pair DM，不是 `cos__human`。
- **P4 未命中（机制可用，缺一次点击）：** 办公室**自己**弹了 `request-117 @cos wants to promote "Quarterly Renewal Outreach Playbook" into the team wiki. Approve?`——它知道该落团队 wiki，但要把人叫来点一下；这一张卡同时是 `cos__human` 的投递闸门。
- **P5 未命中：** cos 的 18:59:00 回合 `final_chars=0`（空转），没有任何 `task_retry` job，下一次唤醒是 19:55:55 的 `task-follow-up`（57 分钟后）。
- **P6 未命中：** 判定时 `team/emails/`、`team/accounts/`、`team/playbooks/quarterly-renewal-outreach.md` 都不存在，父 Issue `in_progress`。停表后经那一次点击才出现 playbook（计入补充实验，不改判定）。
- **新缺陷：** ①取消不撤销已排的派活 job（NEX-95 取消 2 分钟后被重新拉起）；②`scheduler` 的 recheck 只写 alert 不派回合，且 alert 没有变成人可见消息。
- **停表后的补充实验（不计分）：** 用 prereg 允许的那一次点击回答 `request-117` → `team/playbooks/quarterly-renewal-outreach.md` 立刻出现（带 provenance 页脚），`cos__human` 恢复投递；但办公室随即重做已完成的两件事，仍以 hollow 收尾。机制通，缺的是「不用人点」和「别重做」。

---

## 2. 九步评分表

| 步骤 | 分数 | 判定 | 证据（A=机器可验，B=文件在盘，C=口述） |
|---|---|---|---|
| **1 创建任务** | **4** | delivered | **A：** `snapshots/1925/task-NEX-98.json` `created_at=2026-11-07T10:55:55Z`、`owner=cos`、`channel=cos__human`、`created_by=human`；`actions` `action-68 task_created actor=human 10:55:55Z`。首条子 Issue `NEX-100 10:56:04Z`，**9 秒**后出现。同 Journey 1：频道是 DM 而非共享频道，扣 1。 |
| **2 定义清楚** | **1** | broken | **A：** 全部任务 `definition=null`（`tasks.json`：NEX-95/98/100/102/104 与 `task-skill-107` 无一例外），`actions` 里只有 `task_created` / `task_updated [in_progress|review|done]`，没有任何 definition 写入。父 Issue 的成功标准、产物路径都只存在于自然语言正文里。 |
| **3 上下文与授权采访** | **2** | partially | **B：** 确实发起了采访：`request-110`（analyst→human，问三家客户事实）、`request-113`（ae→human，问联系人与 usage 基准），均写明「未知就写未知」——比 Journey 1 的 0 次采访好。**缺口：** 两条都只落在 `analyst__human` / `ae__human` DM，任务频道与 Inbox 无副本；`team/inbox/raw/` 自 17:05 起无新文件。 |
| **4 拉起团队** | **4** | delivered | **A：** `tasks.json` 三条子 Issue `NEX-100 ae__cos owner=ae`、`NEX-102 analyst__cos owner=analyst`、`NEX-104 cos__human owner=cos`，均由 `actor=cos` 在 `10:56:04Z` 同一秒创建（`action-72` 等）；`health.json dispatch.members` 五人在岗。**A：** ae/analyst 的回合各自跑完（latency `bot=ae total_ms=40087 final_chars=295`、`bot=analyst total_ms=35993 final_chars=288`）。 |
| **5 专注执行** | **3** | partially（执行成立、投递被闸门吞掉） | **A（执行确实发生）：** ae 回合 `total_ms=40087 final_chars=295`、analyst `35993ms/288`，`NEX-100/102` 都产出了正文并提交审核；cos 的回合也发起了 11 次工具调用（`events.jsonl` `turn_id=f45802a97db6747a` 含 `context_capture/context_promote/human_message/...`）。**A（投递失败）：** cos 三次回合 `final_chars=0`（`18:58:29`、`18:59:00`），伴随 `live-chat-relay-post-error: request pending in this channel; answer required before chat resumes here`（`cos__human` 有 3 条 pending request）——正文被投递闸门吞掉。**扣分理由：** 不是模型没执行，而是「生成成功但送不出去」被记成 `status=ok final_chars=0`，与闸门缺陷本身的记录相互矛盾。 |
| **6 交付物** | **2** | broken | **A：** `NEX-100 done hollow_completion=true`、`NEX-102 done hollow_completion=true`（`tasks.json`）；`wiki-tree.txt 1925` 无 `team/emails/`、无 `team/accounts/`；`NEX-104 artifact=team/playbooks/quarterly-renewal-outreach.md` 但判定时点该文件不存在（playbook 实际在 `agents/cos/notebook/2026-11-07-quarterly-renewal-outreach-playbook.md`）。**B：** analyst 落了 `team/companies/acme-corp.md`、`corti-labs.md`、`needs-confirmation.md`（18:58:00–18:58:05），Brightline 无页。 |
| **7 确定性学习钩子** | **3** | partially | **A：** archivist 于 18:58:00–18:58:05 重生成实体页并写 `team/entities/companies-{acme-corp,corti-labs,needs-confirmation}.facts.jsonl` + `.graph.jsonl`。**B：** 页面是模板薄页（`# Acme Corp` / `**Acme Corp** is a company.` 后接空 `Tasks`），Brightline 缺失。playbook 未提升进 `team/playbooks/`。 |
| **8 技能与策略已激活** | **2** | broken | **A：** `task-skill-107 Run skill: Wiki maintenance` 自 `10:58:29Z` 起 `in_progress`，`updated_at` 停在创建时刻，没有任何回合（`0` 个 `opencode.exe`）；本次运行产出的 playbook/邮件都没有编译成 `team/skills/*`（`wiki-tree.txt` 里的 skills 仍是 onboarding 的 `app-building/data-modeling/wiki-maintenance`）。 |
| **9 复用与去重** | **1** | broken | **A：** 重复 lane 与主线并行：`NEX-95`（同一份简报）在 `10:57:04Z` 被重新拉起并跑了一整个回合（`events.jsonl` `turn_id=f45802a97db6747a started_at 10:57:10Z`），与 `NEX-98` 同题同 owner；`task_follow-up` 两条并存。无任何复用证据（`context_used` 在整个 `events.jsonl` 里出现 **0** 次）。 |

合计 **4+1+2+4+3+2+3+2+1 = 22 / 36**。

---

## 3. 预注册判据逐条

| # | 判定 | 证据与理由 |
|---|---|---|
| **P1 cos 声明产物路径** | **未命中** | `task-NEX-98.json` `definition=null`；`task-NEX-100/102/104.json` 同。`actions` 无 definition 事件。产物路径只出现在 `NEX-104.artifact`（普通字段，事后填写），不是 `definition.artifact_path`。 |
| **P2 子任务落在 pair DM** | **命中（命名 `ae__cos` 而非 `cos__ae`）** | `tasks.json` `NEX-100 channel=ae__cos`、`NEX-102 channel=analyst__cos`。与判据写的字面顺序不同，但确是 creator⇄owner 的 pair DM，不是 `cos__human`；`cos__planner` 式的第三者在场问题消失。 |
| **P3 owner 能写自己的任务频道** | **命中** | `headless-codex-cos.log` `18:56:49 posted streamed output to #analyst__cos as msg-120`；`messages.json` `msg-141/143`（`ae__cos`）由 `ae` 与 `cos` 交替写入；全轮无 `channel access denied` / `403`。 |
| **P4 人不需要补 save to wiki** | **未命中（机制本身可用）** | 三份要求产物都不在团队 wiki；playbook 的 promotion 被做成一张人审批卡：`request-117 "@cos wants to promote \"Quarterly Renewal Outreach Playbook\" into the team wiki. Approve?"`（判定时 `status=pending`）。**停表后的补充实验（不计分）：** 用 prereg 允许的那一次点击回答 `request-117` → `{"ok":true}`，`wiki/team/playbooks/quarterly-renewal-outreach.md` **同一秒出现**（内容与 notebook 草稿一致）。所以发现问题不在提升机制，而在「必须有人点一下」。 |
| **P5 空转回合被短重投接住** | **未命中** | 18:59:00 的 cos 回合 `final_chars=0`；`/scheduler` 中不存在 `kind=task_retry` 的 job（只有 NEX-62/NEX-84 两条已 `done` 的历史），下次唤醒 `task-follow-up:cos--human:nex-98 next_run=2026-11-07T11:55:55Z`，距空转 **57 分钟**。 |
| **P6 端到端人可见交付** | **未命中** | 无 `team/emails/*`、无 `team/accounts/*`、无 `team/playbooks/quarterly-renewal-outreach.md`；父 Issue `NEX-98 status=in_progress updated_at=10:56:18Z`（自创建后未再变动）。 |

**人工干预：0 次**（prereg 允许的 1 次未使用）。所有 pending request 都保持 `pending`，判定就是「不点会怎样」。

---

## 4. 本轮新发现（按严重度）

1. **done 与产物彻底脱钩**：`hollow_completion=true` 是系统自己打的标记，说明「无产物即完成」已被识别，却仍然放行并让父 Issue 等它。`internal/team/task_completion_hook.go` 的产物门只在 `artifact_path` 存在时校验，`null` 直接通过。
2. **投递闸门会吞掉自己的产出**：只要频道里有 pending request，`live-chat-relay` 就拒发（`request pending in this channel; answer required before chat resumes here`）。cos 的回合因此 `final_chars=0`，而卡住它的正是它自己弹的三张卡（`request-44 #general`、`request-46 #review`、`request-117 promote`）。任务频道同时是交付通道与审批通道，二者互锁。
3. **取消之后仍出现该任务的回合**：`NEX-95` 在 `10:55` 前后取消（响应 `{"changed":true}`），`10:57:04Z` 状态回到 `in_progress`，`10:57:10Z` 又起了回合（`events.jsonl` `turn_id=f45802a97db6747a`，11 次工具调用）。**触发路径尚未定位**：`scheduler.json` 里 NEX-95 只剩一条 `task_follow_up next 11:57:04Z`，没有 `task_retry`，所以不能断言是 recheck 重新派活，也可能是 cancel 与已在途的重投竞争。可确定的是 §9.9 的 `CancelHeadlessTurnsForTask` 只清 cancel 那一刻 active/queued 的回合，覆盖不到之后出现的派活。**待办**：复现并确认触发者（scheduler 重投 / boot reconcile / turn 内部重试），再决定 `broker_scheduler.go` 的 `cancelSupersededTaskJobsLocked` 是否要在终态时关掉全部指向该 task 的 job。
4. **recheck 只报警不派活**：`internal/team/scheduler.go:278-345` 的 `processTaskJob` 对非 blocked 任务只做 `CreateWatchdogAlert` + `RecordAction` + `deliverTask`，不派回合。19:11:54 的 `dispatched 1 due job(s)` 对应 `signals` 里一条 `task_stalled 11:11:54Z`，而 `messages.json` 最后一条停在 `11:08:39Z` —— 人什么也看不到，任务继续挂着。
5. **重做已完成的活、hollow 依旧**：审批放行后 `11:21:58Z` 又建 `NEX-148`（邮件）与 `NEX-150`（简报），而 `NEX-100`/`NEX-102` 早已 `done`；两条新任务同样 `hollow_completion=true`（`snapshots/1930/tasks.json` 共 4 条 hollow done）。重投路径不查「同题任务是否已完成」，去重只覆盖同一 active 回合。

---

## 5. 与 Journey 1 对比

| 维度 | Journey 1（6/10） | Journey 2（5/10） |
|---|---|---|
| 拆活与频道归属 | 子 Issue 落 `cos__human`，跨频道 403 | **修好**：`ae__cos`/`analyst__cos`，无 403（P2/P3 命中） |
| definition | 父 Issue 有 `goal/success_criteria`（缺 `artifact_path`） | **退步**：全 null |
| 团队 wiki 产物 | 3 份 accounts + 1 playbook 落库 | **退步**：0 份；两份 done 是 hollow |
| 自唤醒 | `recheck_at` 真的把办公室叫醒并跑完 reviewer QA | **退步**：recheck 到点只写 alert，不派回合 |
| 人工介入 | approve×1 + 补参×1 | 0 次（但也因此永远停在审批卡上） |
| 环境 | beam-free 可用 | beam-free 限流 → 换 `gpt-5.6-luna`；会话库已按工作区隔离（`~/.wuphf-spaces/journey2/.local/share/opencode`，auth 硬链接） |

---

## 6. 证据索引

- `snapshots/1925/tasks.json`、`task-NEX-{95,98,100,102,104}.json` — 状态、`definition`、`hollow_completion`、`artifact`。
- `snapshots/1925/messages.json`（106 条）— `msg-141/143` 的 ae/cos 交替；最后一条 `11:08:39Z`。
- `snapshots/1925/wiki-tree.txt`（348 个文件）— 无 `team/emails`、无 `team/accounts`、无 `team/playbooks/quarterly-renewal-outreach.md`。
- `snapshots/1925/scheduler.json` — `task-follow-up:…:nex-98 next 11:55:55Z`；无 `task_retry`。
- `snapshots/1925/headless-headless-codex-cos.log` — `live-chat-relay-post-error` 连发。
- `snapshots/1925/events.jsonl` — NEX-95 的 `10:57:10Z` 回合、`context_used` 出现 0 次。
