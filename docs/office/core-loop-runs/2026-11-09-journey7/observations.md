# Journey 7 — 观察记录（2026-11-09）

> 只记时间戳事实。判分见 `scorecard.md`（独立 reviewer）。全部 UTC。
> 隔离办公室 `journey7`（broker :7910，`opencode-go/crouton-alpha-free`）。prereg：`f3b62d4`（含 2.1 节对「专用频道不可行」的更正）。停表：连续 15 分钟无新消息 → 08:11 前后。

## 1. 环境清理（这次做干净了）

| 时间 | 事实 |
|---|---|
| 07:42:20 | `workspace create --from-scratch journey7` → `Ready (broker :7910, web :7911)` |
| 07:42:43 | onboarding 完成（`ai-revops` 蓝图，cos/revops/analyst/ae/reviewer） |
| 07:42:43 | `ai-revops-1..4` 全部取消（`{"changed":true}` ×4） |
| 07:42:51 | 检查：`tasks` 只剩 4 条 canceled，`requests` 为空。**Journey 6 的污染源（`[@analyst] Repeated errors blocked` 落进 `cos__human`）这次没有出现** |

## 2. 时间线

| 时间 | 事实 |
|---|---|
| 07:42:55 | 提交父 Issue `JOURN-6`（`cos__human`、owner cos、`created_by=human`、`planning`，无 `definition`） |
| 07:44:21 | `request-9` 自动出现：`kind=approval`、`dedupe_key=plan-approval:JOURN-6`、`pending`，全办公室仅此一张 |
| 07:44:30 | **唯一一次人工动作**：`POST /interview/answer {"id":"request-9","choice_id":"approve"}` → `{"ok":true}`；同秒 `JOURN-6` → `running` |
| 07:44:30 | 同一次保存里装了 `plan-decomposition-check:cos--human:task:journ-6`（due 07:49:30） |
| 07:44:55–07:45:00 | 拆出 6 条 lane：`JOURN-13`(cos)、`JOURN-16`(ae)、`JOURN-18`(cos)、`JOURN-21`(ae)、`JOURN-23`(reviewer)、`JOURN-25`(reviewer)，`parent_issue_id=JOURN-6`，**全部无 `definition`** |
| 07:45:13 | `JOURN-25` 以 `artifact_not_required=not_required: 复核结论已在任务记录中，无文件交付` + `hollow_completion=true` 收 `approved`（done 门按设计放行） |
| 07:47:40 | `JOURN-21` 交审，交付物是 `artifact=ra_7dda77e1f5b39750`（visual artifact），`review_state=ready_for_review` |
| 07:49:30 之后 | `plan-decomposition-check` 在快照 `0747` 里已是 `done`，`status_reason=superseded by task_follow_up`——父任务拆出子 Issue 引起的生命周期重排把它顶掉了（本该静默收尾，只是理由写的是「被顶替」而不是「已分解」） |
| 07:48–07:50 | 六条 lane 全部进 `decision`（等人类评审）；`JOURN-6` 保持 `running` |
| 07:56:08 | 一条 `task_stalled`（「@cos has gone quiet on Journey 7 … for about 11 minutes」） |
| 08:11 | 停表（连续 15 分钟无新消息） |

## 3. 机制侧观察

1. **分解这次正常**：批准后 30 秒内 6 条 lane，`F-E` 未触发；`task_not_decomposed` 告警 0 条（因为不需要）。
2. **没有一次 wiki 写尝试**：`broker.log` 的 `write-task` 行 **0**，`403` **0**，`refused` **0**，`artifact_path_missing` **0**。因此 V1–V4 再次**未观察**——但这次不是被环境干扰，而是模型的选择。
3. **lane 走的是 notebook / visual-artifact 这条路**（先把「有没有写过 wiki」按提交核清）：
   - `artifacts/wiki-commits.txt` 共 33 条。onboarding 之后（15:44:55 +0800 起）有 5 条：`ae artifact: create visual artifact ra_7dda77e1f5b39750`(15:45:29)、`reviewer fact: people/cos …`(15:45:13)、`archivist: entity article people/cos`(15:45:15)、`ae artifact: create visual artifact ra_2fdd9ec06a768730`(15:46:53)、`cos artifact: create visual artifact ra_6dae47db21a3806a`(15:47:34)。
   - 也就是说：**bots 确实写了 wiki 仓库，但走的是 visual-artifact / fact / entity 通道，没有一条 `team/**` 的 markdown 交付，也没有一次 `write-task`**。
   - 落盘的 notebook：`wiki/agents/ae/notebook/journ-16-48.md`、`journ-21-48.md`、`wiki/agents/cos/notebook/acme-48.md`。
   - `JOURN-21` 的交付是 `ra_7dda77e1f5b39750`；`JOURN-25` 显式声明 `artifact_not_required`。
4. **与工具设计一致**：`team_wiki_write` 在不带 `task_id` 时回的是「需要人类直接授权……bot 侧知识请用 notebook_write → notebook_promote」（`internal/teammcp/server_wiki_tools.go:40-52`、`:100-110`）。也就是说 403 `artifact path not declared` 这条路径**只有在 bot 主动带 `task_id` 调 write-task 时才会命中**，而任务的 `definition.artifact_path` 根本没人钉。J3/J4 之所以撞上 403，是那几轮 lane 自己选择了「带任务 ID 写入」；J6/J7 的 lane 选择了 notebook。
5. **父任务没有人收口**：六条 lane 都在 `decision` 等人类评审，`JOURN-6` 保持 `running`；按 prereg 只允许点批准卡，所以本轮不再有推进。
6. **一个新的可观测点**：`JOURN-25` 是 `approved` + `hollow_completion=true` + 显式 `artifact_not_required`——逃逸舱口按设计工作，没有静默放行。

## 4. 本轮不直接判分的补充事实

- 计划的验收口径写得很实（金额/席位/有效期/付款/决策点五项），lane 的数字复核也确实逐项对照了父任务给定值（`JOURN-23` 07:48:13 那条写明 $18,000→$21,600 的计算）。
- 六条 lane 里有两条（`JOURN-13`、`JOURN-18`）的 owner 也是 cos：同一 bot 在同一时刻持有多条 lane，这是 plan 拆分的结果，不是 bug。
- `write-task` 0 次与「没有 `artifact_path_missing`」互相印证：写门没有被触发过，J3-3 的提示链路本轮同样未被锻炼。
