# Journey 9 — 观察记录（2026-11-09）

只记时间戳事实，不写结论。结论在 `scorecard.md`（独立判分）。

## 0. 环境准备与第一次作废

| 时刻(UTC) | 事实 |
|---|---|
| 13:39 | `workspace create --from-scratch journey9` → broker :7910 / web :7911；home `C:\Users\ireke\.wuphf-spaces\journey9\.wuphf`（与主办公室 `~/.wuphf` 不同目录，主办公室 7892 未被指向） |
| 13:39 | `/health` → `provider_model=opencode-go/crouton-alpha-free`、`memory_backend=markdown`、`wiki_ready=true`、members=1 |
| 13:40:33 | **第一次提交场景** `JOURN-2`（此时 onboarding 蓝图 starter lane 未取消） |
| 13:40:58–13:43:06 | cos 在 `ai-revops-4`（review 频道）上连开 5 个回合：「频道列表为空，改用默认频道重试」「把 #ai-revops-4 拆成三条子任务」「子任务创建遇到渠道404」… |
| 13:41:43 | `request-24`（approval，**blocking**）「Create the #hygiene channel?」— 蓝图为 starter lane 起的信道审批卡 |
| 13:43 | 判定为环境作废：场景任务与 starter lane 抢同一个 cos，且 prereg §2.1 要求在提交场景前清 starter。`workspace shred --yes journey9` |
| 13:44:5x | 重建 `journey9`（同端口），重新 onboarding（`ai-revops`，agents cos,revops,analyst,ae,reviewer） |
| 13:45:01–02 | 取消 `ai-revops-1..4`（成功；此前 cos 只在频道里发过一条自我介绍，starter lane 上无实质动作） |

作废轮次唯一的落盘工件是 `snapshots/2145-aborted/`（目录名带 `-aborted`，来自第一次建办公室时的 `/tasks` 与 `scheduler` 抓取，包含 `ai-revops-1..4` 与 `JOURN-2`，判分时不计入场景）。

## 1. 场景提交与计划

| 时刻(UTC) | 事实 |
|---|---|
| 13:45:26 | `POST /tasks` 建 `JOURN-6`，`channel=cos__human`、`owner=cos`、`created_by=human`、`task_type=issue`、无 `definition`；响应 `lifecycle_state=planning`、`due_at=14:45:26Z` |
| 13:45:43 | cos：「已把续约依据写入计划便签，正在为你整理可审阅的执行方案。」 |
| 13:45:48 | cos 在频道贴出计划全文（目标 / 步骤 1–4 / 拟拆分子任务「批准后才创建，本轮不创建」） |
| 13:45:49 | `request-10`（approval）「@cos wants to start work on Journey 9…」；`JOURN-6.lifecycle_state` 仍 `planning` |
| 13:47:04 | **唯一一次人工干预**：`POST /interview/answer {"id":"request-10","choice_id":"approve"}` → `{"ok":true}` |

## 2. 执行段（每 5 分钟快照到 `snapshots/HHMM/`）

### 13:47–13:52

| 时刻(UTC) | 事实 |
|---|---|
| 13:47:17 | `JOURN-14`（A 报价）、`JOURN-16`（B 邮件）、`JOURN-18`（C 终审）同时创建，`created_by=cos`、`parent=JOURN-6`，三条都带 `artifact_path` |
| 13:47:42 | `JOURN-32` 创建：`created_by=cos`、`parent=JOURN-6`、owner=reviewer、`team/reviews/acme-renewal-final-check.md`，标题「Acme 续约数字与事实终审」 |
| 13:47:43 | `JOURN-35` 创建：`created_by=system`、**无 parent**、owner=cos、`team/reviews/JOURN-14.md`，标题「审阅：子任务 A…」——审阅 lane 由 broker 自动派生 |
| 13:47:57 | `JOURN-6.lifecycle_state` → `running`；`request-10` status=`answered` |
| 13:48:03 | ae：「跟进邮件草稿已写入 `team/emails/acme-renewal-followup.md`，JOURN-16 已提交审核」 |
| 13:48:52 | system：new wiki article「JOURN-14 审阅结论…」(by @cos) |
| 13:49:20 | `JOURN-56` 创建（system，审阅 JOURN-32） |
| 13:51:39 | `JOURN-77` 创建（system，审阅 JOURN-18） |
| 13:51:53 | 任务面：14/18/32/35 `done`+`approved`；16 `decision`；6/56/77 `running` |
| 13:51:53 | scheduler：`task_follow_up` due 13:53:39、`plan_decomposition_check` due 13:52:04、`recheck` due 14:02:54、两条小时级 `task_follow_up` 14:49:20 / 14:51:39（对应 13:49/13:51 创建的审阅 lane）、`request_follow_up` 14:21:11 |

**13:51 实测的 CJK 标题包含度**（同一脚本、同一函数重算，样本=当时全部 JOURN 任务）：

| 对 | inter / min | containment | 是否撞守卫 |
|---|---|---|---|
| `JOURN-18` vs `JOURN-32` | 7 / 8 | **0.875** | 否（< 0.9）——两条都是 reviewer 终审，只是文件名从 `acme-renewal-check.md` 变成 `acme-renewal-final-check.md`，绕过了 `artifact_path` 唯一性 |
| `JOURN-14` vs `JOURN-35` | 8 / 8 | 1.000 | 否（35 由 system 创建，不走规划者守卫） |
| `JOURN-32` vs `JOURN-56` | 8 / 8 | 1.000 | 否（同上） |
| 其余各对 | — | ≤ 0.778 | 否 |

### 13:53–13:57

| 时刻(UTC) | 事实 |
|---|---|
| 13:53:01 | system：`审阅：子任务 C… delivered: 文件存在且非空 — artifact: team/reviews/JOURN-18.md` |
| 13:53:01–05 | cos：「已补齐记忆检索，正在重新完成任务」「JOURN-77 审阅完成…JOURN-18 通过」 |
| 13:53:07 | `request-96`（approval，非 blocking）「promote JOURN-77 审阅… into the team wiki」 |
| 13:53:52 | cos 在 `cos__human` 汇总：「JOURN-6 进展：A 报价说明草稿已通过独立审阅，C 数字与事实终审已通过，B 48 小时跟进邮件可就绪」 |
| 13:55 | 请求面：`request-10` answered；`request-71/91/96` 均 pending、`blocking` 字段为空（推广卡，不 gate） |
| 13:55 | actions 里出现 `watchdog_alert`：`@cos still needs to move Journey 9: Acme 续约报价与跟进 in #cos__human` |
| 13:55 | wiki `team/**` 已落：`accounts/acme-renewal-quote.md`、`emails/acme-renewal-followup.md`、`reviews/`（≥5 份） |

**scheduler 三次快照的父 Issue follow-up（D4 的直接观测）**

| 快照时刻 | `task-follow-up:cos--human:journ-6` due_at | 说明 |
|---|---|---|
| 13:48（snapshots/1348） | `14:45:26Z` | 批准后的小时级武装（13:45:26 + 60 min） |
| 13:52（snapshots/1352） | `13:53:39Z` | lane 落地后被提前到「落地时刻 + 2 分钟」 |
| 13:54（实时） | `14:23:42Z` | 父 Issue 在 13:53:42 被唤醒并推进后，回到常规 30 分钟节拍 |

父 Issue 的实际动作：13:53:39 唤醒 → 13:53:52 在频道发布进展 → 13:53:57「进展已发布」。中途无 10 分钟以上的空档。

**D6 在真机上的第一次复现**（`broker.log`）：

```
21:47:39 wiki write-task: task=JOURN-14 slug=analyst path=team/accounts/acme-renewal-quote.md bytes=477
21:47:40 wiki write-task refused: task=JOURN-16 slug=ae path=team/emails/acme-renewal-follow-up.md:
         task JOURN-16 declares artifact_path "team/emails/acme-renewal-followup.md", not "team/emails/acme-renewal-follow-up.md"
21:47:47 wiki write-task: task=JOURN-16 slug=ae path=team/emails/acme-renewal-followup.md bytes=499
```

`-follow-up` 对 `followup`，7 秒后用声明路径重试成功；频道里没有为这次不匹配投任何提示。

### 13:57–14:02

| 时刻(UTC) | 事实 |
|---|---|
| 13:57 | 任务面：14/18/32/35/56/77 全 `done`+`approved`；`JOURN-16` 仍 `in_progress`/`decision`（13:48:03 起）；父 `JOURN-6` `running` |
| 13:57 | 全部 JOURN 任务的 `artifact`：14→`team/accounts/acme-renewal-quote.md`；16→`team/emails/acme-renewal-followup.md`；**18→`team/reviews/acme-renewal-final-check.md`**；32→同左；35→`team/reviews/JOURN-14.md`；56→`team/reviews/JOURN-32.md`；77→`team/reviews/JOURN-18.md` |
| 13:57 | `ls wiki/team/reviews/` → `JOURN-14.md`(4675)、`JOURN-18.md`(6776)、`JOURN-32.md`(6277)、`acme-renewal-final-check.md`(1931)。**`acme-renewal-check.md` 不存在**——`JOURN-18` 声明的路径从未被写入，它 `done` 时指向的是 `JOURN-32` 的文件 |
| 13:57 | 挂起请求 3 张，全部 `kind=approval`、`blocking` 空：`request-71`(13:51:11) / `request-91`(13:52:41) / `request-96`(13:53:07)，均为 notebook→wiki 推广 |

### 14:00–14:06（等待人工决策）

| 时刻(UTC) | 事实 |
|---|---|
| 14:02:54 | `recheck:ae--cos:task:journ-16` 到点派发（`scheduler: dispatched 1 due job(s)`） |
| 14:03:0x | 该 job 收尾：`status=done`、`status_reason=awaiting review (decision) — the ball is not with the owner`；`JOURN-16.updated_at` 仍 13:48:03，无新回合 |
| 14:04:43 | `cos__human` 出现 watchdog 消息：「@cos has gone quiet … for about 11 minutes」 |
| 14:05 | **`plan-decomposition-check:cos--human:task:journ-6` → `done`，`status_reason="decomposed into 4 sub-issue(s); 1 still open"`**。同父子 Issue 实为 4 条（14/16/18/32），其中 1 条 open（16）；35/56/77 由 system 创建、无 parent，不计 |
| 14:05 | 关掉的 job：`task-follow-up:ae--cos:journ-16` → `superseded by recheck`；14/18/32/35/56/77 的 `task_follow_up` → `target closed`；`request-follow-up:...:request-10` → `target closed` |
| 14:05 | 仍在跑的 job：父 `task-follow-up:cos--human:journ-6` due 14:23:42；三张推广卡的 `request_follow_up` 14:21:11 / 14:22:41 / 14:23:07 |
| 14:05 | `broker.log` 另有 `gbrain call put_page … EOF` 连续失败（memory 写入），与核心循环无关，单列 |
| 14:12 | `JOURN-16`：`status=in_progress`、`lifecycle_state=decision`、`review_state=ready_for_review`、`pending_interview=None`、`due_at=14:02:54Z`（已过）。**没有任何 interview/approval 卡指向它**——全库请求只有 `request-10`(answered) 与三张推广卡。人类能看到的只有 13:53:52 那条聊天：「待你确认发送」 |

## 3. 停表与操作员探针

| 时刻(UTC) | 事实 |
|---|---|
| 14:04:43 | 最后一条消息（system watchdog「gone quiet 11 minutes」）；此后无任何消息、无 active 回合 |
| 14:20:02 | 静默 15.28 分钟 → **停表**。停表时 `JOURN-6` 仍 `in_progress`/`running`，`JOURN-16` 卡在 `decision`，其余 6 条 JOURN 任务 `done`+`approved` |
| 14:20:04 | 探针 P1（V5/D6）：`POST /wiki/write-task`，`task=JOURN-16`、`path=team/emails/wrong-acme.md` → **403**，body 同时给出声明路径与尝试路径；`artifact_path_missing` 计数 0 → 0（未新增频道提示） |
| 14:20:05 | 探针 P3（D2）：以 `created_by=cos` 在 `JOURN-6` 下重建标题「Acme 续约数字与事实终审」→ **409** `duplicates sibling JOURN-32 … which already shipped under parent JOURN-6 — reopen JOURN-32 instead … or pass allow_duplicate=true`。另一次用「子任务 C：数字与事实终审（Acme 续约）」→ **409** 指向 JOURN-18。两次都是**完全相同**标题，命中的是既有 `titlesAreSimilar`，不是 CJK shingle 判据 |
| 14:20:5x | 探针 P2-a（首次，**用法错误**）：`{"action":"complete","id":"JOURN-16","artifact":"ra_…"}` → 200。`TaskPostRequest` 的字段是 `artifact_path`，`artifact` 被忽略，`JOURN-16` 以原有路径 `team/emails/acme-renewal-followup.md` 正常收 done，非门限行为。此条按「探针未命中」记录 |
| 14:21:0x | 探针 P2-b（D3）：新建探针 lane `JOURN-111`（`created_by=cos`、`parent=JOURN-6`、`definition.artifact_path=team/accounts/probe-d3.md`），用 `artifact_path=ra_0123456789abcdef` 收 done → **409**（「declared the write scope … a visual artifact … does not close it」）。D3 门在真机成立 |
| 14:21:0x | 探针 P2-c（D7）：同一 `JOURN-111` 改用**别的 lane 已写的文件** `team/accounts/acme-renewal-quote.md` 收 done → **200 done**。声明的 `team/accounts/probe-d3.md` 从未存在，门限没有拦。JOURN-18 的同型行为可现场复现 |
| 14:21:04 | 探针副作用：`JOURN-16` 收 done 后 broker 派生审阅 lane `JOURN-105`（system，无 parent），cos 随即写出 `team/reviews/JOURN-16.md`（5406 字节）。属探针引入，判分时单列 |
| 14:21:12 | 探针副作用：`JOURN-111` 收 done 同样派生审阅 lane `JOURN-114`（system） |
| 14:21:46 | **探针后遗症（真机复现的第二个洞）**：analyst 被完成事件唤醒，对**已经 done** 的 `JOURN-111` 再次发 `complete`，`artifact_path=ra_4e8fa813a190faa3` → **200**，任务 `artifact` 被改写成视觉 artifact。完成门只在「转为 done」时判定，`reachedDone=false` 的重复 mutation 不再受门限约束。终态：`JOURN-111.declaration.artifact_path=team/accounts/probe-d3.md`（文件不存在）、`artifact=ra_4e8fa813a190faa3`、`status=done/approved` |
| 14:22:10–29 | cos 把探针 lane 当成真实任务收束：「JOURN-111 已完成并已标记 done（artifact_path=ra_…）。结论是 D3 无法通过：本轮 wiki 检索…0 命中」——说明运营者/机器人无法区分探针任务与业务任务，探针污染了父 Issue 的汇总 |


### 交付物（停表时盘上实存）

| 路径 | 字节 | 作者 | 备注 |
|---|---|---|---|
| `team/accounts/acme-renewal-quote.md` | 477 | analyst | 报价说明；只用任务给的事实，未编造生效日 |
| `team/emails/acme-renewal-followup.md` | 499 | ae | 跟进邮件；`[Sender]` 占位符保留，未发送 |
| `team/reviews/acme-renewal-final-check.md` | 1024 | reviewer | JOURN-32 的终审（**JOURN-18 曾把这条当成自己的 artifact**） |
| `team/reviews/JOURN-14.md` | 4675 | cos | 审阅 JOURN-14 |
| `team/reviews/JOURN-32.md` | 6277 | cos | 审阅 JOURN-32 |
| `team/reviews/JOURN-18.md` | 6776 | cos | 审阅 JOURN-18 |
| `team/reviews/JOURN-16.md` | 5406 | cos | 审阅 JOURN-16（探针触发） |






