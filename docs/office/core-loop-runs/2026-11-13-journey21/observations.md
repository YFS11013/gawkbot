# Journey 21 观测 — D-1/D-2 上线后的第一次真跑

隔离办公室 `journey21`（broker :7910，web :7911），二进制 = 仓库根 `gawkbot-fix.exe`（`2bf9f92`，D-2 修完），免费模型 `beam-free`。
操作员预算 4 次全用：下达场景、批准计划卡、退回一次、对重复 lane 驳回并给方向。

## 时间线（UTC）

| 时间 | 事实 | 来源 |
|---|---|---|
| 16:55:22 | `task_reminder` 的 `interval_override` 设为 5 分钟（V6 用） | `scheduler_activity` |
| 16:55:30 | 场景消息下发（`source=operator`，发送者是 `nex`） | `msg-1` |
| 16:56:15 | 模型建 Issue `JOURN-2`（`created_by=cos`，`task_type=issue`） | `tasks[]` |
| 16:56:15 | **planning 回合自动启动**（queue_wait 78 ms），期间 Issue 上没有任何人类 comment | latency log |
| 16:58:00 | 计划回合结束，`request-8`（`plan-approval:JOURN-2`）**自己升起** | `requests[]` |
| 16:58:21 | 批准后回合启动；先 `team_wiki_write` 带 `human_request: msg-1` → 被拒（"not a human-authored message"），改 `msg-11` → 被拒（"does not explicitly ask for a direct wiki write"），再带 `task_id` → 被拒（"artifact path not declared"） | opencode `part` 表 |
| 16:59:46 | 模型先 `team_task action=define` 给 `JOURN-2` 声明 `artifact_scope`，随后 `team_wiki_write` **走 `/wiki/write-task` 成功**：commit `8858816`，作者 `cos` | wiki `git log` |
| 16:59:49 | 另建 lane `JOURN-14`（`artifact_scope=team/reviews/j21-public.md`），写入成功：commit `6708e77`，作者 `cos` | `tasks[]`、`git log` |
| 17:00:08 / 17:00:20 | 又建两个**重复** lane `JOURN-22`（内部页）、`JOURN-25`（对外页），两者都只等在 planning | `tasks[]` |
| 17:00:46 | 对容器 `complete` → 409「container with 1 delivered lane … Call submit_for_review」 | 工具输出 |
| 17:01:52 | `submit_for_review` 成功，`JOURN-2` 进 `review`，容器卡 `request-19`（`decision:JOURN-2`）升起 | `requests[]` |
| 17:02:01 | **操作员退回一次**，理由：对外页第 2 条是内部流程、读者看不懂；第 3 条太笼统 | `actions[]` |
| 17:02:02 | 退回后**回自己起来了**（queue_wait 100 ms），回合开头原话：「The human gave feedback on JOURN-2 (changes requested)…」 | 会话库 reasoning |
| 17:02:58 | 对外页按意见改写：commit `0453dac`（正文见下） | wiki `git log` |
| 17:03:04 / 17:03:28 | `submit_for_review` 连续两次 **400**：`resubmission requires an artifact change — the artifact (team/reviews/j21-internal.md) is byte-identical …` | 工具输出 |
| 17:04:02 | 操作员对 `request-29` 驳回并给方向（两条是重复任务，回去改 `JOURN-2`） | `actions[]` |
| 17:06:58 | 模型自己诊断出根因并要求操作员二选一：「broker 的变更检测键在内部页上、因内部页未改动而阻塞…请指示：直接批准 done，或允许我对内部页做无内容改动的 touch」 | `msg` |

运行结束时：`JOURN-2` 停在 `in_progress`（rework 已落盘、卡在 resubmit 上），`JOURN-22`/`JOURN-25` 停在 `in_progress`，`request-36`/`request-43` 两张卡仍 `pending`。

## 断言相关的原始证据

- **交付内容**（退回归档时刻的 sha256 前 16 位）：`j21-internal.md` `940e72d4802cf715`（从未变过）；`j21-public.md` 退回时 `e88cd08c8d734f7c`，改写后 `2169e9a396d60f27`。
- **对外页改写后正文**：第 2 条从「领导 Issue 获得独立的规划轮次」改为「负责人自己的任务现在也会先提出方案供你审阅，不再跳过规划直接执行」；第 3 条从「更新了评分卡与预注册文档」改为「记录了真实使用中出现的自唤醒现象；为已修复的问题追加了复检轮次；为新的测试场景预先登记了测试条件」——两条都对着退回意见改的。
- **内部页正文**：5 行，短 hash 全部来自给定 5 条；对外页 0 个 hash。
- **整轮 wiki 提交**：`8858816`(cos) → `6708e77`(cos) → `aef44ad`(cos) → `c3fd8ae`(cos) → `0453dac`(cos)。**0 条 `external edit`**，作者全是 `cos <cos@wuphf.local>`。
- **直写尝试**：整轮会话库里 `write` / `edit` / `patch` / `bash` 工具调用 **0 次**（`glob` 7 次、`read` 0 次、`team_*` 若干）。也就是说这一轮模型根本没去碰逃生口，D-2 的现场证据仍是 `d2p4` 探针。
- **工具错误分布**（整轮 66 条 `team_task` 错误、13 条 `wiki_write` 错误）：
  | 次数 | 错误 |
  |---|---|
  | 65 | `define` 不带 `id` → `404 task not found` |
  | 6 | `wiki_write` 的 `human_request` 不是人类消息 |
  | 4 | `wiki_write` 的 `human_request` 没明确要求直写 |
  | 3 | `define` 没给 `goal` → 400 |
  | 3 | `wiki_write` 的路径没被任务声明 |
  | 1 | 容器 `complete` → 409（行为正确，带修复指路） |

## 两个新缺陷（本轮当场复现，已修）

### D-3 容器 Issue 的重提闸门只看它自己的 artifact —— 改在 lane 里的返工永远过不去

`gateTaskResubmissionArtifactDelta` 用 `task.Artifact` 单文件哈希判定「有没有改」。容器 Issue 的 `artifact` 是**其中一个** lane 的文件（这里是内部页），而人类退的是**交付集合**；返工落在另一个 lane（对外页）时，单文件哈希不变，`submit_for_review` 永远 400。本轮两次 400 之后，模型的反应是新建两个重复 lane —— 这条闸门把「改完了」变成了「退不回去」。

修法：决策路径已经在退回时盖了**交付集合**摘要（`SentBackFingerprint`，和卡上那句「no file changed」用的是同一个摘要）；闸门在有这个摘要时按集合比对，没有时回落到单文件哈希。测试 `TestArtifactDelta_ContainerLaneChangeClearsTheGate`（红：退回后只改 lane 文件，闸门仍报 `byte-identical` 并点名内部页）。

### D-4 非 create 动作漏传 `id` 报「task not found」，模型据此重试 65 次

`team_task` 的 `id` 是 `omitempty`，`action=define` 不带 `id` 也能过 schema，broker 里匹配不到任何任务，回 `404 task not found`。一个回合内模型用这个形状重试了 65 次——错误信息指向「任务不存在」，而真实问题是「你没传参数」。

修法：`handleTeamTask` 在发请求前判定非 `create` 动作必须带 `id`，错误写成 `team_task action="define" needs the task id — pass id=<the id returned by create>`。测试 `TestHandleTeamTaskNamesAMissingIDInsteadOf404`（红：去掉守卫后 6 个动作全部回落到 404 文案）。

- 容器卡 `request-19` 是 17:00:02 第一次 `submit_for_review` 时升起的，那一刻 `task.Artifact` 还是空的，所以正文写「The Issue ships no single file of its own … lanes delivered 1 file(s): team/reviews/j21-public.md」。之后（17:01:52 那次 submit）Issue 的 `artifact` 变成 `team/reviews/j21-internal.md`，退回时闸门就只看这一个文件——**卡按集合说话、闸门按单文件判定**，这是 D-3 的直接来源。
- 重复 lane 不只是噪音：`JOURN-25`（与 `JOURN-14` 声明了同一个 wiki 路径）**5 次覆写了 `team/reviews/j21-public.md`**，台账每次都记了 `artifact_overwritten`（17:00:31 / 17:01:34 / 17:02:58 / 17:05:37 …），但写入照常成功——「一个任务能否覆盖另一个任务已交付的 artifact」是 `TODO-NEXT.md` 里早就挂着的产品决策，本轮把它的代价量到了：连返工那次改写（`0453dac`）都是记在 `JOURN-25` 名下的。

## 旁证

- 计划未批准时建子任务被 409 拦住；容器 `complete` 被 409 拦住并指名 `submit_for_review`——两条错误信息自带修复路径，模型都照着做了。
- `human_interview` 工具一次调用直接 error（无输出），模型改用 `team_task` 通道；随后 `request-36` 以聊天式采访卡升起（`dedupe_key` 为空）。
- 计划回合把「受众与取舍」自己定了（「对外说明：≤3 条中文人话」），没有为这个开采访卡。
- `task-skill-5`（Run skill: Wiki maintenance）从 16:57 起一直 `in_progress`，与主环路无关，未参与判分。
