# Journey 11 — 观察记录（2026-11-10）

> 时区：**UTC 为主**（本机 `+0800`，日志里是 12:xx）。
> 本文件只记时间戳事实，不写结论；判分在 `scorecard.md`（独立子代理）。
> 被测 `54ad984`，隔离办公室 `journey11`（broker :7910 / web :7911），模型 `opencode-go/longcat-2.5-free`。

## 1. 环境

| 时间（UTC） | 事实 | 证据 |
|---|---|---|
| 04:02:48 | `workspace create --from-scratch journey11`（进程带 `WUPHF_OPENCODE_MODEL=opencode-go/longcat-2.5-free`）→ registry `journey11`、`broker_port=7910`、`web_port=7911`、`state=running` | `~/.wuphf-spaces/registry.json` |
| 04:03:06 | broker `scheduler: loop started (poll=20s)` | `logs/broker.log` |
| 04:03:25 | `POST /onboarding/complete {skip_task:true, blueprint:ai-revops, agents:[cos,revops,analyst,ae,reviewer]}` → `{"ok":true}`；`onboarding: wiki materialized blueprint=ai-revops dirs=5 articles_created=5`、`wiki bootstrap committed 9f80823`、`getting-started pages committed b3a950f` | broker.log、`/onboarding/complete` |
| 04:03:25 | `/health`：`status=ok`、`provider_model=opencode-go/longcat-2.5-free`、`wiki_ready=True`、`dispatch.members=1` | `/health` |
| 04:03:25 | **场景提交**：`POST /tasks` 建父 Issue `JOURN-2`（`channel=cos__human`、`owner=cos`、`created_by=human`、`task_type=issue`、无 `definition`），正文与 J10 §3 逐字相同（仅标题前缀 `Journey 11`） | `/tasks/JOURN-2` |
| 04:04:51 | 取消 onboarding starter lanes `ai-revops-1..4`（均在 `in_progress/running`）→ 4× 200 `changed:true` | `/tasks/{id}/cancel` |

## 2. 时间线

| 时间（UTC） | 事实 | 证据 |
|---|---|---|
| 04:03:19–04:03:20 | cos 同时起 3 个回合（`stage=started`，`queue_wait_ms=43/62/122`）；revops/analyst/ae 各 1 个 | `logs/headless-codex-latency.log` |
| 04:04:06 | cos 第一段输出落频道：`Plan mode — read-only this turn. Let me gather the context I need before writing the plan.`（`msg-6`） | `broker-state.json.messages` |
| 04:04:51 | cos 的 3 个回合被 `error: headless codex turn cancelled so newer queued work can run` 取消；同秒 `wake-lead: re-delivering specialist completion from @revops (msg msg-4)`；新回合 `stage=started queue_wait_ms=85897`（约 86 秒排队） | `logs/headless-codex-cos.log`、latency log |
| 04:05:10 | 新回合再次落同一句 `Plan mode — read-only this turn…`（`msg-12`，与 `msg-6` 逐字相同） | `messages` |
| 04:08:23 | cos 发出**范围采访卡** `request-13`（`kind=interview`、`channel=cos__human`、`blocking=null`、`required=null`）：`给 Dana（VP Ops）的续约跟进邮件，用哪种语言？`，选项 `english/chinese/bilingual/something_else` | `broker-state.json.requests` |
| 04:08:23 | `pending_interview` 全部为 null；cos 的回合随即停住（此后到 04:12 无新输出） | `broker-state.json.tasks`、`headless-codex-cos.log` |
| 04:09:13 | **操作员回答 `request-13` → `english`**（200 `{"ok":true}`）。**这一次不在 prereg §6 的 2 次点击预算内**：场景与 J7/J8/J9/J10 逐字相同，前四轮都没有触发范围采访；本轮是模型行为差异，记录在此，判分时单列 | `/interview/answer` |
| 04:09:27 | 计划卡 `request-18`（`dedupe_key=plan-approval:JOURN-2`，选项 `approve/approve_with_note/needs_more_info/reject/reject_with_steer`） | `broker-state.json.requests` |
| 04:09:36 | **操作员点 `approve`（计划卡）→ 200 `{"ok":true}`**；父 Issue 同秒 `planning → running` | `/interview/answer`、`JOURN-2.lifecycle_state` |
| 04:09:56 / 04:09:59 / 04:10:04 | cos 拆出 3 条 lane：`JOURN-22`（analyst，`team/accounts/acme-renewal-quote.md`）、`JOURN-24`（ae，`team/emails/acme-renewal-followup.md`）、`JOURN-26`（reviewer，无 `definition`，`artifact_not_required="not_required: QA verdict only — no file deliverable; findings land as comments and an approve/request_changes decision. (cos, 2026-11-10T04:10:04Z)"`） | `messages` msg-23/25/27、`tasks[].definition` |
| 04:10:42 | 系统建审阅 lane `JOURN-32`（`status=open`、`lifecycle_state=queued_behind_owner`） | `tasks` |
| 04:10:5x | `JOURN-22` 落盘：commit `0f9d1af`（`team/accounts/acme-renewal-quote.md`），任务转 `review/review` | `wiki` git log、`tasks` |
| 04:11:21 | **lane 级 `decision:` 卡 `request-43`**（`dedupe_key=decision:JOURN-22`、`issue_id=JOURN-22`、`blocking=null`、`required=null`）：正文 `@analyst finished Draft Acme renewal quote one-pager (JOURN-22) … The deliverable is team/accounts/acme-renewal-quote.md.` | `requests`、`snapshots/decision-card.json` |
| 04:11:26 | **操作员点 `approve_decision` → 200**；`JOURN-22` 转 `done/approved`，`artifact=team/accounts/acme-renewal-quote.md`、`hollow_completion` 未置真，卡撤销（`request-43` 状态 `answered`） | `/interview/answer`、`tasks` |
| 04:11:32 | 父 Issue 仍 `in_progress/running`，`artifact=null`（另两条 lane 未收） | `/tasks/JOURN-2` |

**干预计数（超出 prereg §6 的预算，逐条列出）**：① 04:09:13 回答范围采访 `request-13`（语言）；② 04:09:36 计划卡 `approve`；③ 04:11:26 lane 卡 `request-43` 的 `approve_decision`。prereg 只给了 2 次（计划卡 + 父 Issue 卡）；两次额外干预都由**本轮模型行为**引出（先问语言、再让一条 lane 先进 review→decision），不是判据对象。父 Issue 卡的点击若发生，将计为第 4 次并单独标注。

**lane 级卡与 J10 的差别（事实，不是结论）**：J10 没有任何 lane 进入 `decision`（交付后直接 `done/approved`）；本轮 `JOURN-22` 走了 `review → decision → 点击 → approved`，说明「有 reviewer 的 lane」这条路会真的停在人手里。
| 04:12:16 | `JOURN-24` 落盘：commit `8e8f10c`（`team/emails/acme-renewal-followup.md`，`bytes=955`），任务转 `review/review` | broker.log、wiki git log |
| 04:12:5x | lane 级卡 `request-56`（`dedupe_key=decision:JOURN-24`）pending | `requests` |
| 04:13:15 | **操作员点 `approve_decision`（`request-56`）→ 200**；`JOURN-24` → `done/approved`，`artifact=team/emails/acme-renewal-followup.md`。**第 4 次干预**，同样不是为了判据对象，而是父 Issue 要等所有 lane 收口 | `/interview/answer`、`tasks` |
| 04:13:41 | 审阅 lane `JOURN-32` 被 reviewer 自己取消（`canceled/approved`） | `tasks` |
| 04:13:59 | **`JOURN-26`（QA）回合 `status=ok`、`total_ms=234083`、`final_chars=945`，结论是「两份交付物在任何地方都不存在」**：msg-69 `Confirmed: the deliverables don't exist anywhere, and I can't read the producing channels.`、msg-71 `…在 wiki、notebook、artifact 和全库检索中均不存在…` | `logs/headless-codex-latency.log`、`logs/headless-codex-reviewer.log`、`messages` msg-69/71 |
| 04:16 | **反证**：同一时刻在同一个办公室上 `GET /wiki/read?path=team/accounts/acme-renewal-quote.md&reader=reviewer` → 200 且返回正文；`/wiki/search?pattern=Acme&reader=reviewer` → 200 且命中该路径；`/wiki/list` 的 `team/accounts` 段列出该页。即**文件在盘、索引里有、reader 身份不挡读**，QA 的「不存在」是假阴性；根因未定（候选：模型按主办公室的 `~/.wuphf/wiki` 路径去读、或在 04:11:03 之前就开始读了且没复查） | `/wiki/read`、`/wiki/search`、`/wiki/list`（停表前探针，不改动任何状态） |
| 04:18:50 | `POST /tasks/JOURN-26/message` → **404**（无此子路径） | 探针 |
| 04:19:0x | `POST /tasks/JOURN-26/comment {body:…}` → **500 `{"error":"failed to post message"}`**：JOURN-26 的频道是 `cos__reviewer`（cos×reviewer 的 DM），人类写不进去；错误只回 `failed to post message`，不说为什么 | `/tasks/JOURN-26/comment` |
| 04:20:05 | 改到父 Issue：`POST /tasks/JOURN-2/comment {body:"@cos 更正 QA 的前提…两份交付物都在 wiki 里…"}` → 200 `author=human:team-member`；cos 同秒起回合（`queue_wait_ms=126`）。**第 5 次干预** | `/tasks/JOURN-2/comment`、latency log |
| 04:21:56 | cos 回合 `status=ok total_ms=110790 final_chars=113`：回 `@reviewer 转达人工更正…`（msg-75）与 `已转达 @reviewer 按路径复核并收尾 JOURN-26；等它完成后我再将父 Issue JOURN-2 提交审核`（msg-76）。**此后没有任何 reviewer 回合启动**，JOURN-26 停在 `in_progress` | `messages`、`logs/headless-codex-latency.log` |
| 04:23:26 | **操作员取消 `JOURN-26`**（200 `changed:true`），理由写进请求：QA 前提是假阴性、lane 无法自纠、取消以便父 Issue 提交审阅。**第 6 次干预** | `/tasks/JOURN-26/cancel` |
| 04:23:58 | 操作员在父 Issue 留话请 cos `submit_for_review`（200 `author=human:team-member`）。**第 7 次干预** | `/tasks/JOURN-2/comment` |
| 04:24:27 | 父 Issue 转 `status=review lifecycle_state=review`（`updated_at=04:24:27Z`）：cos 按人话提交了终审，msg-79/80 | `broker-state.json` JOURN-2 |
| 04:24:51 | **父 Issue 决策卡出现** `request-83`（`dedupe_key=decision:JOURN-2`，`issue_id=JOURN-2`，`blocking`/`required` 均 null）。卡正文：`The deliverable is team/accounts/acme-renewal-quote.md.` —— **只点了一个文件** | `snapshots/parent-decision-card.json` |
| 04:24:51 | 点击前父 Issue 的字段：`artifact=team/accounts/acme-renewal-quote.md`、`hollow_completion=null`。**这个 artifact 不是代码写的**：cos 在 `submit_for_review` 时把其中一条 lane 的路径当成 Issue 的交付物报上去了 | `$TEMP/j11-parent.log` 的 `PARENT BEFORE` 行 |
| 04:25:00 | watcher 点 `approve_decision` → **200 `{"ok":true}`** | `$TEMP/j11-parent.log` |
| 04:25:06 | `JOURN-2` = `status=done lifecycle_state=approved review_state=approved`，`artifact` 仍是那一条路径，`hollow_completion=null` | `snapshots/parent-decision-card.json` |
| 04:25:xx | wiki git log 出现 `002deee decision: promote JOURN-2 on approval`，随后 archivist 把 `companies/vp-ops`、`companies/dana-vp-ops` 落成实体页 | `artifacts/wiki-log.txt` |

**父 Issue 卡的判读（不要过度声称）**：卡出现了、点了、闭环了，这一条是真的。但**聚合分支没有被真机锻炼**——父 Issue 进 `decision` 时自己已经有 `artifact`，旧谓词（只读自己的两个字段）同样会发卡。真正的差别只体现在正文：旧写法说「The deliverable is <一条 lane 的路径>」，新写法应当逐条列出两条。本轮正文仍是旧口径的单文件，因为它命中的是 `own != ""` 分支。已在被测代码里修掉（lane 列表优先，Issue 自己的路径只在没有 lane 认领时提一句），回归见 `TestDecisionCardNamesEveryLaneEvenWhenIssueCarriesOneArtifact`。

**干预合计 7 次**（prereg 预算 2 次）：语言采访 1、计划卡 1、`JOURN-22` 卡 1、`JOURN-24` 卡 1、父 Issue 更正 QA 前提 1、取消 `JOURN-26` 1、催 `submit_for_review` 1。评分卡 §3 记的是 6 次（把两次父 Issue 留言并成一次），以此表为准。
