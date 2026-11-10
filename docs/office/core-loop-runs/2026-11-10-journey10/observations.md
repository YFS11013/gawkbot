# Journey 10 — 观察记录（2026-11-10）

> 只写时间戳事实，不写结论。所有时间以 UTC 为主（本地为 +08:00）。
> 隔离办公室 `journey10`（broker :7910 / web :7911），home `C:\Users\ireke\.wuphf-spaces\journey10\.wuphf`。
> 被测提交 `52456d3`（+ `76e6171`、`fdb6fa8`），prereg `4f161fd` + 模型修订 `d6ca54f`。

## 0. 环境前置（提交场景之前）

| 时间（UTC） | 事件 |
|---|---|
| 02:27 | `workspace create --blueprint ai-revops journey10`，broker :7910/:7911 起，模型 `crouton-alpha-free` |
| 02:31 | 逐个 onboarding 阶段推进（greet → identity → website → blueprint → team → seed），蓝图 `ai-revops`，agents `revops, analyst, ae, reviewer` |
| 02:33 | 种子 5 个成员 + 4 条 onboarding lane（ai-revops-1..4），**每个 headless 回合 2 秒内 `exit status 1`**，频道落 24 条 `UnknownError: Unexpected server error`；broker 自建 `JOURN-3`（"Repeated errors blocked"） |
| 02:35 | 直连探测：`opencode run --model opencode-go/crouton-alpha-free` → `Model not found … Did you mean: beam-free, iceman-contributor-free, longcat-2.5-free?` |
| 02:37 | 三个候选用 `pong` 探针全过；带工具的同一题：`longcat-2.5-free` 16 s 一次读对 `internal/buildinfo/VERSION`，`beam-free` 22 s 读两次 |
| 02:38 | 主办公室（7892）重启到 `opencode-go/longcat-2.5-free`；`/health` 确认 `provider_model=opencode-go/longcat-2.5-free`、`wiki_ready=true` |
| 02:40 | `workspace shred journey10` + 重建（同蓝图、同模型），种子干净：5 成员、4 lane、2 条消息、无错误 |
| 02:41 | 取消 `ai-revops-1..4`（`POST /tasks/{id}/cancel` 均 200 `changed=true`） |

## 1. 场景

父 Issue `JOURN-14`（`cos__human`、`owner=cos`、`created_by=human`、`task_type=issue`、`pipeline_id=issue`、无 `definition`），提交于 02:42。

- 标题：`Journey 10: Acme 续约报价与跟进`
- 内容与 J7/J8/J9 逐字相同（40 席位 / $18,000 → $21,600 +20% / 14 天 / Dana VP Ops / 两件交付物）。

## 2. 时间线

| 时间（UTC） | 观察 |
|---|---|
| 02:42:0x | 提交父 Issue `JOURN-14`（`pipeline_id=issue`，`lifecycle_state` 随 planning 姿态推进） |
| 02:42:4x | 频道 `msg-13`（cos）："正在读取外联与 CRM 相关资料，然后把跟进序列拆成可执行的子任务。" |
| 02:43:0x | `msg-16`（cos）："Scoping the two deliverables under #JOURN-14 and handing the renewal quote page to @revops and the follow-up email to @ae" |
| 02:43:4x | 计划卡 `request-17`（approval，`cos__human`）出现，`msg-18` 通知："@cos wants to start work on Journey 10 … Approve and they begin."；`JOURN-14.lifecycle_state=planning` |
| 02:44:33 | 操作员批准（见 §3） |
| 02:44:5x | `JOURN-14.lifecycle_state=running`；频道 `msg-20`（you）："Approved @cos's request." |
| 02:45:0x | 第一条 lane `JOURN-21`（owner=revops，`cos__revops`，`definition.artifact_path=team/accounts/acme.md`，标题「Acme 续约一页报价说明（落 team/accounts/acme.md）」），`msg-22` 通知 |
| 02:45:1x | scheduler：`task-follow-up:cos--human:journ-14` due `03:44:33`；`plan-decomposition-check:…:journ-14` due `02:49:33` |
| 02:45:16 | 第二条 lane `JOURN-23`（owner=ae，`team/emails/acme-renewal-followup.md`）；cos 在 `msg-25/26` 说明「产出回收后我在 #JOURN-14 内核对一致性再交你过目」 |
| 02:45:31 | 两条 lane 齐（V7 观察点：`team/accounts/acme.md` vs `team/emails/acme-renewal-followup.md`，路径不同） |
| 02:46:02 | `msg-27`（general）：`New wiki article: Acme 续约报价说明 (by @revops)`；`msg-28` revops："先认领 #JOURN-21，然后写 team/accounts/acme.md" |
| 02:46:55 | `msg-29` `task_delivered`：`team/accounts/acme.md 存在且非空`；同一秒 `msg-31` `task_review_requested`（系统建 `JOURN-30`，owner=cos，直立审阅 lane，artifact `team/reviews/JOURN-21.md`）；`msg-32` revops："本地 worktree 里没有该路径，补一份到 worktree 以满足 artifact 校验" |
| 02:47:02 | `JOURN-21` 终态：`status=done`、`lifecycle_state=approved`、`review_state=not_required`、`execution_mode=office`、`verification.kind=artifact`、`artifact=team/accounts/acme.md`（**没有进 `decision`，没有找人类签字**） |
| 02:47:1x | wiki 复核：`team/accounts/acme.md` 617 B 在盘，commit `dce4efc`；`team/emails/acme-renewal-followup.md` 与 `team/reviews/JOURN-21.md` 均不存在；`git status` 仅 `.reads/` 未跟踪 |
| 02:47:3x | wiki commit `867b378`（learning）+ `a269e6f`（archivist playbook `acme-team-accounts-acme-md`）自动产生 |
| 02:49:18 | wiki commit `6d9cc8e`：`team/reviews/JOURN-21.md`（审阅结论，三项验收标准全过，附 2 条提示级措辞项）；`msg-35` general 通知 |
| 02:49:27 | `JOURN-30` 终态：`done` / `approved` / `review_state=not_required` / `artifact=team/reviews/JOURN-21.md`；`msg-36` `task_delivered`；`msg-37` cos："审阅完成：JOURN-21 三项验收标准全部通过" |
| 02:50:03 | `msg-38`（general）：`New wiki article: Acme 续约跟进邮件草稿 (by @ae)`；`msg-39` ae："正在起草 Acme 续约跟进邮件并落 wiki" |
| 02:50:28 | `msg-40`（cos__human，cos）："On it — 核对两份产出的落地情况，再决定是聚合送审还是继续等 @ae。" |
| 02:51:1x | wiki 复核：`team/emails/acme-renewal-followup.md` 910 B 在盘，commit `5c3c6b3`；**两条 lane 的声明路径与落盘路径逐字一致** |
| 02:52:0x | 状态复核：`JOURN-23` 仍 `in_progress` / `review_state=pending_review`；`requests` 里只有 `request-17`（已答），**没有任何 `decision:` 卡** |
| 02:49:02 | `actions[].kind=watchdog_alert` 两条：`action-32`（@cos "still needs to move Journey 10 …"，related JOURN-14）、`action-33`（@ae "still needs to move Acme 续约 48 小时跟进邮件草稿"，related JOURN-23）。**同一时刻两人都在跑回合**：cos 回合 02:49:03 起、02:49:38 结束（latency log `total_ms=161659`），ae 于 02:50:03 落稿 |
| 02:50:03 | ae 的回合结束方式：`headless-codex-ae.log` 记 `queue-drop: bot already handling same task`（02:49:03）后正常出稿；**`JOURN-23` 没有随之收尾**，仍是 `in_progress` |
| 02:53:1x | scheduler 现状：`task-follow-up:cos--human:journ-14` 被提早到 `03:19:02`（原 `03:44:33`，提前 25 分钟）；`task-follow-up:ae--cos:journ-23` 为 `03:19:03`；两条 lane 的 `task_follow_up` 都是半小时级，**03:19 之前没有任何指向 JOURN-23 的短周期 job** |
| 02:54:16 | `JOURN-23` 收尾：`msg-45` `task_delivered`（`team/emails/acme-renewal-followup.md 存在且非空`）；同一秒 `msg-47` 建审阅 lane `JOURN-46`（artifact `team/reviews/JOURN-23.md`） |
| 02:54:20 | **父 Issue `JOURN-14` 进入 `lifecycle_state=decision`**（`review_state=ready_for_review`，`pipeline_stage=review`，`artifact=null`，`artifact_not_required=null`）；`msg-48` cos："#JOURN-14 已进入待审，等你在 Inbox 过目。" |
| 02:54:52 / 02:55:12 | ae 播报落库与完成（`msg-49/50`） |
| 02:55:42 | `requests` 复核：**只有 `request-17`（已答）**，没有任何 `dedupe_key=decision:JOURN-14` 的卡。`JOURN-14` 的 `artifact` 与 `artifact_not_required` 都是空 → `hasApprovableDeliverable` 为假 → `raiseDecisionCardLocked` 直接返回 `""` |
| 02:56:1x | `JOURN-46`（审阅 email）落 `team/reviews/JOURN-23.md` 并 `done/approved`；全部四条 lane 与两条审阅 lane 均 `approved`，只剩父 Issue 停在 `decision` |
| 02:57:2x | 最后一条消息 `msg-57`（ae 给 cos 的跟进节奏建议）；此后无新消息、无 active 回合 |
| 02:59:41 | 最后一条消息 `msg-62`（ae："审阅结论已落库并汇报，任务板无待办。"）；此后 55 条消息不再增长 |
| **03:04:20** | **父 Issue 在 `decision` 已满 10 分钟，`requests` 仍只有 `request-17`（已答），没有 `decision:` 卡 → prereg 的 F-I 触发** |
| 03:04:31 | 复核：`JOURN-14` `status=in_progress`、`lifecycle_state=decision`、`updated_at=02:54:20`（进入 decision 后未再变化）；五条 lane/审阅任务全部 `approved`；`actions` 里 `watchdog_alert` 仍是 02:49:02 那两条 |
| 03:09:03 | scheduler 的 `recheck:cos--human:task:journ-14` 跑过一次（`last_run=03:09:03`，`status=done`），payload 里带 cos 写的「聚合核对（#JOURN-14）：① team/accounts/acme.md — JOURN-30 独立审阅通过全部验收标准…」 |
| 03:09:11–03:09:19 | ae 被再次唤醒，连发 `msg-63/64/65`：先说审阅结论已落库、再说「deliverable already exists … let me confirm the office state」、最后又播报一遍。**没有产生任何新交付或状态变化**（重复播报） |
| 03:11:44 | `requests` 仍只有 `request-17`；`JOURN-14` 仍是 `decision` / `updated_at=02:54:20` |
| 03:14:57 | 同前：58 条消息、无新卡；父 Issue 进入 `decision` 已 20.6 分钟 |

## 3. 操作员动作

| 时间（UTC） | 动作 | 结果 |
|---|---|---|
| 02:44:33 | `POST /interview/answer {id:request-17, choice_id:approve}` | 200 `{"ok":true}` |

## 4. 停表判定

- 最后一条场景消息 `msg-62` 于 02:59:41；03:09:11–03:09:19 有一次**重复播报**（ae 被 recheck 唤醒，`msg-63/64/65`，无状态变化）。
- 之后到 03:25:25（16 分钟）无新消息、无 active 回合、`requests` 里没有未答的 `decision:` 卡 → **停表条件成立**（15 分钟静默 + 无回合 + 无未答卡）。
- 父 Issue `JOURN-14` 从 02:54:20 起停在 `decision`，末态由操作员探针 P1b 关闭。

## 5. 停表后探针（不计场景干预）

| 探针 | 动作 | 结果 |
|---|---|---|
| **P1a** | `POST /tasks/JOURN-14/decision {action:approve, created_by:human}`，父 Issue 当时 `artifact=null` | **409** `task JOURN-14 cannot be approved into done without a delivered artifact. Have the owner publish the deliverable…` — 文案把下一步交给 owner，而点按钮的是人 |
| **P1b** | 同上 + `artifact_path=team/accounts/acme.md` | **200** `{"action":"approve","status":"recorded"}`；父 Issue 转 `done/approved`，`artifact=team/accounts/acme.md`；`requests` 里 `request-17` 随之被撤（列表空） |
| **P2** | 用 `task_id=JOURN-23` + `slug=cos` 打 `write-task` | 403 `task JOURN-23 is owned by ae, not cos`（探针写错作者，非目标分支） |
| **P3** | 新任务 `JOURN-70` 声明 `team/reviews/pin-probe.md` → 写入 | 200，`commit 355e3b7`，文件在盘 |
| **P3'** | 新任务 `JOURN-73` 声明 `team/reviews/pin-probe-v2.md`（与上一条同目录相似名）→ 写入 | 200，`commit 0a4c053`，**落在 v2 自己的路径**；`pin-probe.md` 内容未被追加（对比 `decsmoke` 上同一形状的旧行为：v2 的内容会被追加进 v1 且 v2 不存在） |
| **P4** | 同一任务 `JOURN-73` 打未声明路径 `team/reviews/not-declared.md` | 403 `task JOURN-73 declares artifact_path "team/reviews/pin-probe-v2.md", not "team/reviews/not-declared.md"`；`not-declared.md` 未落盘 |
| — | 探针任务 `JOURN-70/73` 用完后 `POST /tasks/{id}/cancel` 收尾 | 200 |
