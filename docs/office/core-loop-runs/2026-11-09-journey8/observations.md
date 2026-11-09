# Journey 8 — 观测记录（2026-11-09）

> 只记时间戳事实，不解释、不打分。判分见 `scorecard.md`（独立判分者写）。
> prereg：`a4ff236`；被测代码：`5a026e7`。

## §1 起点

- 全量测试：`go test ./internal/team/ -count=1 -timeout 30m` → `ok 626.565s`（2026-11-09T16:5x 本地，代码 `5a026e7` 工作区）。
- `internal/teammcp/`：`ok 18.053s`。
- 主办公室 7892 `/health` 200（`provider_model=opencode-go/crouton-alpha-free`、`wiki_ready=true`），本轮未指向它。7910/7911 起前为空闲。

## §2 环境与时间线（UTC）

| 时刻 | 事件 |
|---|---|
| 09:29:44 | `workspace create --from-scratch journey8`；broker :7910 / web :7911，`members=1` |
| 09:29:5x | `onboarding/complete`（ai-revops，cos/revops/analyst/ae/reviewer）→ `members=5` |
| 09:30:0x | 取消 `ai-revops-1..4`（install cleanup） |
| 09:30:3x | 提交场景 `JOURN-6`（`cos__human`、owner=cos、created_by=human、无 definition）→ `status=in_progress`、`pipeline_stage=plan` |
| 09:30:49 | 计划卡 `request-9`（kind=approval，"Start work on Journey 8…"） |
| 09:31:1x | 点批准（**本轮唯一场景干预**） |
| 09:31:2x | 拆出 6 条 lane，**全部带 `definition.artifact_path`**：`JOURN-15` team/reviews/acme-renewal-quote.md、`JOURN-17` team/research/acme-renewal-quote.md、`JOURN-19` team/emails/acme-followup-48h.md、`JOURN-21` team/emails/acme-renewal-followup.md、`JOURN-23` 与 `JOURN-25` team/reviews/acme-renewal-qa.md |
| 09:31:35–38 | 3 次 `write-task` 被拒（**路径不匹配，不是未声明**）：JOURN-15 `team/accounts/acme-renewal-quote-draft.md`；JOURN-21/JOURN-19 `team/emails/acme-48h-followup.md` |
| 09:31:40–44 | 同一批 bot **改成声明路径重试**：JOURN-15 → team/reviews/acme-renewal-quote.md（570B）、JOURN-21 → team/emails/acme-renewal-followup.md（694B）、JOURN-19 → team/emails/acme-followup-48h.md（1957B） |
| 09:32:09 | JOURN-23（reviewer）被拒一次：`team/reviews/acme-deliverables-qa.md` ≠ 声明值 |
| 09:32:10–17 | 重试到声明路径成功（2509B）；cos 的审阅任务 `JOURN-41` 写 team/reviews/JOURN-15.md（5102B）、`JOURN-47` 写 team/reviews/JOURN-21.md（5149B） |
| 09:33:14 | reviewer 交付 QA：`team/reviews/acme-renewal-qa.md` |
| 09:34 | 快照 1734：16 任务 / 101 消息；`write-task` 成功 12、被拒 4；wiki `team/` 文件 47 |

## §3 中途事实

- **写范围守卫触发次数：0**（`broker.log` 里 `no write scope` 计数 0）。规划者在建 lane 时就写了路径，没有撞守卫。
- **未声明路径的 403（`artifact_path_missing`）计数：0**；本轮的 4 次 403 全是「声明了但路径不匹配」，因此不走 `nudgeUndeclaredArtifactPath`（设计如此）。
- 全量任务扫描（09:34）：所有非 canceled 任务都有 `artifact_path`，且没有 `artifact_not_required` 声明。
- `JOURN-17`（analyst，声明 team/research/acme-renewal-quote.md）的 `artifact` 记录为 `ra_4904a7a50a981735`（visual artifact），wiki 里 **没有** team/research/ 文件——声明了路径但没往那里写。
- cos 把审阅做成了独立顶层任务 `JOURN-41/47/88/100/105`（各带自己的 `team/reviews/JOURN-*.md` 路径）；其中 100/105 与已完成的 41/47 主题重复（"JOURN-6-B…" vs "子任务 B…"）。
- lane 也有同主题两份：A→15/17、B→19/21、C→23/25（标题措辞不同，兄弟去重按标题相似度，没拦住）。

## §4 停止与反向臂

| 时刻 | 事件 |
|---|---|
| 10:02:07 | 停表：连续 15 分钟无新消息（最后一条 09:46:44，是 watchdog 的「gone quiet」） |
| 10:02:40 | 反向臂：`created_by=human` 建探针任务 `JOURN-127`（owner=ae，无 definition；human 豁免新守卫） |
| 10:02:48 | 同一路径连续 3 次 `POST /wiki/write-task` 均 403：`artifact path not declared … @human must set definition.artifact_path … retrying this call cannot succeed` |
| 10:02:48 | 频道里出现 **1 条** `artifact_path_missing` 提示（@human + @ae 都被点名），此后两次重试**没有**新增 |
| 10:02:5x | 探针任务被取消（`reverse-arm probe complete`） |

## §5 停表时的运行态（10:02Z）

- 16 任务 / 111 消息 / 2 请求；`JOURN-6` 仍 `in_progress running`，`JOURN-25` 仍 `in_progress running`（09:32:00 起无更新）。
- `request-117`（kind=approval，"Promote to the team wiki: JOURN-100 审阅 JOURN-19 结论（补记）"）自 09:34:53 起 **pending**，未答。
- cos 最后一条自己的消息 09:35:57；此后只有两条 watchdog「gone quiet」（09:44:44 对 reviewer、09:46:44 对 cos）。
- `write-task`：成功 9、被拒 4（全部是「声明了但路径不匹配」，随后同任务改路径成功）；`no write scope` 守卫拒绝 **0** 次。
- wiki：79 commits、`team/**` 52 个文件（含 `team/emails/` 2 份、`team/reviews/` 5 份由 write-task 写入，其余为 onboarding/实体/fact 通道）。
- scheduler：两条 `scheduled` job 在 10:31（小时级 follow-up），即停表后仍无人处理 `JOURN-25` 与 `JOURN-6`。
- 抓取限制：`snapshots/*/scheduler.json` 是 405 响应体（`GET /scheduler/jobs` 不是 GET 路由），scheduler 事实一律取自 `broker-state.json` 的 `scheduler` 键。
- 快照只有 4 个（1730/1731/1734/1802）：本轮活动集中在 17:30–17:36，5 分钟节拍没有更多采样点。

