# 生产真任务：这一版发布说明（2026-11-20）

目的：J1–J30 里每一次「人类动作」都是 harness 代替人点的卡。机器侧证据已经够，但**没有人是否真的读交付物、是否愿意点卡**的证据。本轮在生产放一个真任务，操作员不代替人答任何卡。

## 已做（唯一一次写操作）

- 任务 `ACMER-3872`，频道 `cos__human`，owner `cos`，`created_by=human`，`task_type=issue`。
- 交付路径 `team/reviews/release-notes-2026-11-20.md`；`verification_kind=wiki_artifact`、`verification_spec` 同路径、`required=true`。
- 真值清单由创建脚本现场执行 `git log -20 --format='%h %s'` 得到，逐字写进 details；每个短 hash 必须能在清单里查到，不许编造。
- 任务正文明确要求走 `submit_for_review` 并附 `artifact_path`，不许自己宣布 done。

## 我方契约

**我不批准、不退回、不催、不发任何补充消息。** 所有卡留给真人。后续对话里若出现 request，只报告事实，不代答。

## 基线（创建前）

- 生产 `/health`：`provider_model=opencode-go/hy4-flash-free`，`wiki_ready=true`，`DRIFT: none`。
- 开放任务 2 条：`ACMER-3754`、`ACMER-3852`（都是 forex 日简报遗留，与本任务不同频道）。
- 待答 request 2 条：`request-3761`、`request-3803`（forex-watch__human，与本任务无关）。

## 后续要观察什么

| 问题 | 判据 | 读哪里 |
|---|---|---|
| 机器侧是否自己走到人面前 | planning 卡出现 → 页面经任务门落盘 → 决策卡出现，0 直写 | `/requests?scope=all`、`~/.wuphf/wiki/team/reviews/release-notes-2026-11-20.md`、wiki `git log` |
| 真人是否读了 | 人自己点卡（approve / send back）或给出具体反馈；若长期无人答，记录为「无人读」而不是判代码缺陷 | `actions[]` 的 `request_answered`、任务频道消息 |
| 内容是否可信 | 每个短 hash 都在 `git log -20` 清单内；无编造；两段结构与文末「下一版注意」齐全 | 盘上文件 + 真值清单 |
| 是否绕过人 | bot 自己 `complete` 直通 `done`/`not_required` | `tasks[ACMER-3872].lifecycle_state` 与 `guard_refused` 审计 |

复核命令（只读，别代答）：

```bash
curl -s -H "Authorization: Bearer $(curl -s http://127.0.0.1:7891/api-token | jq -r .token)" \
  'http://127.0.0.1:7892/tasks/ACMER-3872'
curl -s -H "Authorization: Bearer $(curl -s http://127.0.0.1:7891/api-token | jq -r .token)" \
  'http://127.0.0.1:7892/requests?scope=all'
git -C ~/.wuphf/wiki log --oneline -- team/reviews/release-notes-2026-11-20.md
```

生产 web UI：`http://127.0.0.1:7891/`，进 `cos__human` 的 Tasks / Inbox 看卡。

## 首次被动观察（2026-11-20 23:00 本地 / 15:00Z）

- 22:59:25 `cos stage=started`，23:00:31 `status=ok`（65.5s，正文 3359 字）；`ACMER-3872` `in_progress` / `planning`。
- 计划卡 `request-3875`（`dedupe_key=plan-approval:ACMER-3872`）**自行升起且 pending**。我方未答任何卡、未发任何补充消息。
- 其余两张 pending（`request-3761` / `request-3803`）属于 `ACMER-3754` 的 forex 遗留，与本任务无关，同样未动。
