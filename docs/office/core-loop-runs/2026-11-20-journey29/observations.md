# Journey 29 observations — 被拒之后第一次就回正路、中文授权、以及谁验收

工作区 `~/.wuphf-spaces/journey29`，broker `:7910` / web `:7911`。源码 `3087138`，二进制 `gawkbot-j29.exe` 从该提交构建。模型 `opencode-go/hy4-flash-free`，兜底 `opencode-go/iceman-contributor-free`（`/health` 回显一致，`members=1`）。以下时刻均为 UTC（本地 = UTC+8）。

## 时间线

| 时刻 UTC | 事件 |
|---|---|
| 08:07:26 | `workspace create --from-scratch journey29` → Ready；config 已是 `llm_provider=opencode` + `memory_backend=markdown`，无需改配重启 |
| 08:08:40 | 真人会话建立：`POST /humans/invites` → `/humans/invites/accept`（session `session-3619ca2f8b1ab1c1`，`human_slug=ireke`）→ `POST /messages` 下达场景，`msg-2`，`from=human:ireke` |
| 08:08:40 | cos 回合 `stage=started`（`headless-codex-latency.log`） |
| 08:09:50 | `team_task create` → `JOURN-3`（内部清单），`artifact_scope=team/reviews/j29-internal.md`，`verification_kind=wiki_artifact`，details 里写了两页路径；状态 completed |
| 08:09:51 | `team_task create` → `JOURN-5`（对外说明），无 `artifact_scope`，`artifact_not_required_reason=对外页不在任何任务写范围内…`，details 里写了两页路径；状态 completed |
| 08:10:29 | cos 回合 `status=ok`，14.1s 首字、37.3s 完成，正文贴出 JOURN-5 的计划；`request-13`（`plan-approval:JOURN-5`）出现 |
| 08:11:04 | 操作员批准 `request-13`（`action-12` / `action-11`），JOURN-5 plan approved |
| 08:11:08 | `request-19`（`plan-approval:JOURN-3`）出现 |
| 08:11:10 | `team_wiki_write(task_id=JOURN-3, mode=create, j29-internal.md)` → 200，`bytes_written=587`，commit `60aa22c` |
| 08:11:11 | `team_wiki_write(human_request=msg-2, mode=create, j29-public.md)` → 200，`bytes_written=273`，commit `6f95267` |
| 08:11:55 | `team_task define`（JOURN-3）；`team_task submit_for_review`（JOURN-5，artifact=public 页） |
| 08:11:56–08:13:13 | 多次 `complete`/`approve` 被 409：`JOURN-3 is still in planning`（4 次）、`a bot may not close it from here — that would skip the human`（JOURN-3/JOURN-5 共 8 次）；一次 `human_request=msg-17` 被拒：`does not explicitly ask for a direct wiki write` |
| 08:12:10 | 模型 `team_poll` 读频道，取回 `msg-2` 的原文（含中文授权句） |
| 08:12:22 | 操作员批准 `request-19`（`action-28`），JOURN-3 plan approved |
| 08:12:30 | `team_task resume`（JOURN-3）→ in_progress |
| 08:12:58–08:13:05 | `submit_for_review`（JOURN-3 ×3、JOURN-5 ×2） |
| 08:11:59 / 08:12:59 | `request-28`（`decision:JOURN-5`）/ `request-42`（`decision:JOURN-3`）出现，选项 `approve_decision` / `request_changes_decision` |
| 08:13:14–08:13:15 | 两个回合 `status=ok`（52.0s / 46.8s） |
| 08:14:08 | 操作员批准 `request-42`（`action-53`，`choice_id=approve_decision`） |
| 08:15:08 | `task_updated … [done]`（`action-55`），JOURN-3 `done` / `approved` / `verification.wiki_artifact` |

## 工具面原始记录（opencode.db `part`）

- `wuphf-office_team_task create` ×2，均 `"status":"completed"`；第二次 input 无 `artifact_scope`，只有 `artifact_not_required_reason`。全库未见 create 的 409/冲突。
- `wuphf-office_team_wiki_write` ×4：
  - 08:11:10 `task_id=JOURN-3` / internal → `{"bytes_written":587,"commit_sha":"60aa22c"}`
  - 08:11:11 `human_request=msg-2` / public → `{"bytes_written":273,"commit_sha":"6f95267"}`
  - 08:11:56 `task_id=JOURN-3` / internal / `mode=create` → 409 `article already exists … use replace or append_section`
  - 08:11:57 `human_request=msg-17` / public → `team_wiki_write human_request "msg-17" does not explicitly ask for a direct wiki write`
- `wuphf-office_team_task` 的 `complete`/`approve` 拒绝原文（节选）：
  - `409 Conflict task JOURN-3 is still in planning — its plan must be approved before work can be completed.`
  - `409 Conflict task JOURN-3 recorded the deliverable team/reviews/j29-internal.md: a bot may not close it from here — that would skip the human, and office work is exactly where the bytes are the whole point. Call submit_for_review …`
  - `409 Conflict task JOURN-5 recorded the deliverable team/reviews/j29-public.md: …`
- `wuphf-office_team_broadcast` 08:13:24：模型自述「两页已写好并回读核对……已用你 msg-2 的授权走 human_request 写入。收尾卡在最后一步」。

## 落盘产物

- wiki `git log`：`6f95267 JOURN-5: 按人类授权写入对外说明页，3 条以内、无 hash、无内部术语`；`60aa22c JOURN-3: 记录本批 5 条内部改动，逐条带短 hash，交叉引用对外说明页`。
- `team/reviews/j29-internal.md`（587 B）：5 行，逐条短 hash `3087138 / fd154a0 / c2849fc / e20c0f6 / cb7244d`，交叉引用 public 页。
- `team/reviews/j29-public.md`（273 B）：3 条，无 hash、无模型代号、无 bot/wiki 术语。

## 终态（16:14 本地读取）

- `JOURN-3`：`done` / `approved`，`artifact` 与 `artifacts[]` 均为 `team/reviews/j29-internal.md`，`verification={kind:wiki_artifact, spec:team/reviews/j29-internal.md, required:true}`。
- `JOURN-5`：`in_progress` / `decision`，`artifact=team/reviews/j29-public.md`，`review_state=ready_for_review`，`artifacts=null`，`artifact_not_required=…human_request…`；`request-28` 仍 pending。
- `task-skill-8`（`Run skill: Wiki maintenance`）：`in_progress`，`review_state=not_required`。
- `/requests?scope=all`：仅 `request-28` 一条 pending。

## 操作员动作

H1 08:08:40Z `msg-2`（下达 + 中文授权）；H2 08:11:04Z `request-13`；H3 08:12:22Z `request-19`；H4 08:14:08Z `request-42`。共 4 次，未超过 prereg 预算。

## 未做 / 限制

- `request-28`（对外页决策卡）未批准：预算 4 次已用尽；JOURN-5 未进入 done。
- 没有第二次 `create`，守卫未被触发（本轮场景的两条 Details 虽都含两页路径，但第二次 create 没有 `artifact_scope` 可比对）。
- 生产办公室（:7892）未参与本轮。
- 工作区保留供评分复核；按 README 约定不提交运行产物。
