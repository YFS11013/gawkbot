# Journey 2 — 观察记录

时间线（本地 +08:00，UTC 在括号内）。判定见 `prereg.md`，评分见 `scorecard.md`。

## 环境建立

| 时间 | 事件 | 证据 |
|---|---|---|
| 16:44 | 清掉 registry 里 `acme-revops` 残留（`state=running`、`broker_port=0`、runtime_home 指向已删的 `%TEMP%` 目录）；备份 `registry.json.bak-journey2` | 改后 `workspace list` 输出 `No workspaces yet` |
| 16:46 | `workspace create --from-scratch journey2` → broker `:7910`，web `:7911` | CLI 输出 `Ready (broker :7910, web :7911)` |
| 16:47 | 首次 onboarding 用自定 slug 列表，实装在 `ai-revops` 之外的合成蓝图上（`cos/gtm-lead/founding-engineer/pm/designer`），与 Journey 1 不同构 → shred 重来 | `broker-state.json` members；`selectionLooksStaleForStarterBots` 回退到全量 |
| 16:49 | 重建 journey2；`config.json` 钉 `memory_backend=markdown`、`llm_provider=opencode`；`/onboarding/progress` 写 `company_name=Nex`；`/onboarding/complete` 带 `blueprint=ai-revops`、`skip_task=true` | onboarding state `company_name=Nex` |
| 16:49 | 团队落地 `cos / revops / analyst / ae / reviewer`，频道 `cos__human / human__revops / analyst__human / ae__human / human__reviewer` | `broker-state.json` |
| 16:49 | onboarding wiki 物化：`dirs=5 articles_created=5`，getting-started pages 提交 `f185539` | broker.log |

## 作废的两次 bring-up（详见 prereg「环境修正」）

| 时间 | 事件 |
|---|---|
| 16:47 | 首次 onboarding：`agents` 自定列表对合成蓝图的 starter 名册判 stale，装上 `cos/gtm-lead/founding-engineer/pm/designer` → shred |
| 16:49 | 第二次：`config.json` 写在 broker 启动之后，launcher 按默认值选 `claude-code`；NEX-2 的 6 个回合全部以 `Failed to authenticate. API Error: 401 Invalid API key` 收尾（cos 把报错当正文贴进 DM） |
| 17:00 | shred 重建；这次**先写 `config.json` 再重启**，`/health` 报 `provider=opencode, model=opencode-go/beam-free` |
| 17:01 | onboarding（`blueprint=ai-revops`）：名册 `cos/revops/analyst/ae/reviewer` 与 Journey 1 一致 |
| 17:03 | 安装期审批 #1/#2：cos 要建 `#hygiene`、`#pipeline` → Approve（prereg 豁免 ≤2 次） |
| 17:05 | 安装期审批 #3/#4：reviewer 要求把 `team/hygiene/scorecard.md` 提升进 wiki（来源 `agents/reviewer/notebook/…`）、cos 要建 `#review` → 均 Reject，理由 `Not part of this run.` |

## 场景启动

- 08:50Z 写 `prereg.md`（在任何 office 代码动作之前）。
- **09:06:14Z（17:06:14）`POST /tasks` 建父 Issue NEX-62**：`owner=cos`、`created_by=human`、`channel=cos__human`、`task_type=issue`，正文与 Journey 1 同一份三客户续约简报。这是本轮判定的起点。
- 场景启动瞬间 `/office/stats`：`active=3`、`done=1`、`blocking=2`——onboarding 自带的 orientation 任务还在占并发槽，父 Issue 的 cos 回合排在其后。

## 场景进行时

| 时间（本地） | 观察 |
|---|---|
| 17:06:14 | NEX-62 建好，状态 `planning`。同一秒 cos 的 lane 就 `queue-park`：`active=4` 全被 onboarding orientation 任务占着（`ai-revops-1/3/4`、`NEX-9`、`NEX-25/27`、`task-skill-32`）。 |
| 17:06–17:10 | cos 连续 3 次 `queue-park: at concurrency cap`（17:06:14 / 17:07:36 / 17:09:56），父 Issue 一直 `definition=null`。 |
| 17:06 | `ai-revops-2`（analyst）被完成门拦下：`external-action turn ... marked done/approved without recorded external execution evidence`，随后 `error-recovery: already produced durable progress`。同一类错误在 `NEX-23` 上复现。 |
| 17:07 | ae 在 `ai-revops-3` 里报 `channel not found`：蓝图要求 `#outreach`，安装后并没有这个频道，于是回合空转并把问题抛回人。 |
| 17:09 | 两条 orientation 审批挂着没人处理：`request-44`「Approve new channel #general?」、`request-46`「Resolve pending #review channel approval」。产品一边宣称频道已退役、一边让 cos 申请建 `#general`，前后矛盾。 |

判定要点：**父 Issue 的 `planning` 回合被安装期任务饿死**，不是 NEX-62 自己的问题。核心循环的「起步延迟」这一步在第二轮依旧存在，而且成因换成了 onboarding 自带的 orientation 任务占满全局并发（`clamp(cores,2,6)`，本机 4 槽）。

## 17:06–18:00 的失败现场（按发生顺序）

| 时间 | 事件 | 证据 |
|---|---|---|
| 17:17 | 首次人工清场：`POST /tasks` 取消 NEX-9/23/25/27/57 成功，但 `ai-revops-*` 与 `task-skill-32` 报 `404 channel not found` —— 它们的频道 `hygiene/outreach/review/pipeline` 从未被创建 | `/tasks` 响应 |
| 17:19 | 只能改用 `/tasks/{id}/cancel` 才把 `ai-revops-1/3/4`、`task-skill-32` 关掉 | 四次 `{"changed":true}` |
| 17:21–17:23 | 给 NEX-62 补两条 operator 提醒后，第二条约 2 分钟时触发 `human-priority: cancelling active turn after 2m5s`，随后 `error: headless codex turn cancelled so newer queued work can run`。**更正**：我最初记为「没有新 worker 起来」，复查进程列表后错了——17:23:25 恰好有一个新的 `opencode.exe`（PID 28432），worker 正常重启，只是 opencode 本身卡住 | cos 日志 + `Get-CimInstance Win32_Process` |
| 17:26 | 重启 broker 后首次真实报错：`opencode_stderr: Error: Unexpected error / database is locked`，`error-recovery: blocked NEX-62 after failed turn` | cos 日志 |
| 17:30 | 我清掉残留的 7 个 `opencode.exe`，恰好杀掉了在跑的回合 → `exit status 0xffffffff` → 再次 `blocked NEX-62`（自伤，已记录） |
| 17:34 | 又一次 `resume` 被当成 human-priority 消息，取消运行中回合（3m48s） | cos 日志 |
| 17:45 | 停掉主办公室（7892 + watchdog）隔离后，发现根因：`~/.local/share/opencode/opencode.db` **6.6 GB**，多办公室共用同一 session 库 → `database is locked` | `ls -l ~/.local/share/opencode` |
| 17:45 | 把 `opencode.db*` 移到 `*.pre-journey2`，broker 重启后新建空库 | 目录列表 |
| 17:46–17:55 | 仍无回合完成：`latency.log` 只有 `stage=started`，无 `status=ok`；手动 `echo 'Say OK' | opencode run` 从重置前的 ~29 秒退化到 **>300 秒超时** | latency.log + 手工复现 |

## 结论：Journey 2 未完成，判据是「场景回合一次都没跑完」

- 从 17:06:14 建 NEX-62 到 18:00 停表，父 Issue 的 `definition` 始终为 `null`，`lifecycle_state` 在 `planning → running → blocked → running` 之间被错误恢复反复拉扯，**没有产生任何一次模型输出**。
- 因此 `prereg.md` 的 P1/P3/P4/P5/P6 全部无数据；只有 **P2 命中**：真实流量里 cos 为 revops 开的三张子任务（`NEX-23/25/27`）频道都是 `cos__revops`（creator⇄owner pair DM），说明 rehome 修复在真实路径上生效。

## 环境级发现（与产品代码分开记）

1. **boot 顺序**：`workspace create` 先起 broker 再让人写 `config.json`，launcher 在启动瞬间就把 provider 定成默认的 `claude-code`；workspace 里的 `llm_provider: opencode` 要等下一次重启才生效。README/CLI 没有任何提示。
2. **蓝图频道 vs 已退役频道**：`ai-revops` 蓝图仍声明 `hygiene/pipeline/outreach/review` 四个频道，安装时不创建，orientation 任务却按这些频道派活；结果是四个任务全部 `channel not found`、各空转 15 分钟（一次 `headless codex turn timed out after 15m0s`）。
3. **取消不杀子进程**：`/tasks/{id}/cancel` 后 `opencode.exe` 子进程仍在（17:30 前观察到 7 个，对应 4 个已取消任务）。
4. ~~**human-priority 取消后不重启 worker**~~ **推翻**：17:23:25 的进程列表里有一个同秒启动的 `opencode.exe`，worker 确实重启了。记录错误的原因是我在 17:24 数进程时把「7 个残留」当成了「全是旧的」。'
3. **取消任务不会停掉它的回合**（已修，见 §9.9）：`/tasks/{id}/cancel` 只改任务状态，正在跑的 provider 子进程继续烧 token 到自己的超时；17:24 还能看到 17:06/17:15/17:16 启动、对应任务早已被取消的 `opencode.exe`。
5. **办公室不是真的隔离**：所有 workspace 共用 `~/.local/share/opencode/opencode.db`（6.6 GB），并发即 `database is locked`；这与 `workspace` 宣称的「彼此不知道对方」冲突。

## 收尾

- 18:00 停 journey2（registry `state=paused`，数据保留），重新启用并拉起主办公室：`:7892/health` 200，`check-office-drift.ps1` 首行 `DRIFT: none`。
- `opencode.db` 旧库保留为 `~/.local/share/opencode/opencode.db.pre-journey2`（6.6 GB），确认不需要历史后可删。
