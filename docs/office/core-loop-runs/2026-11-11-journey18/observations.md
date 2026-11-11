# Journey 18 现场记录（进行中）

## 时间线（+08:00）

| 时刻 | 事件 | 证据 |
|---|---|---|
| 15:34:23 | `OFFICE-1` 建单（owner `cos`，`created_by=human`），清单 22 条提交写进 details | `tasks[]` |
| 15:34:23–15:35:04 | 回合 1：`status=ok total_ms=40249 first_event_ms=12240` | latency log |
| 15:54:00 | 回合 2 起（`request-4` 计划审批卡在 15:56 前后升起） | latency log |
| 15:56:52 | 回合 2：`status=ok total_ms=171290`；产出计划 + 「等你批准」 | `headless-codex-cos.log` |
| 15:5x | 计划卡 `request-4`（`dedupe_key=plan-approval:OFFICE-1`），选项 `approve / approve_with_note / needs_more_info / reject / reject_with_steer` | broker-state |
| 15:54–15:56 | 计划里两个交付路径写成 `docs/office/Journey18-*.md`（**仓库内**） | card question |
| 15:56 后 | 操作员用 `approve_with_note` 批注：写进 wiki 的 `team/reviews/journey18-{internal,public}.md`，别写进仓库 | `actions[] request_answered` |
| 16:01:53 | 办公室回：「内容已按清单拟好，但还需你一句明确回复才能写入」 | `msg-12` |
| 16:03:24 | 又回一次：「已请求你再发一条明确消息『请写入 wiki』」 | `msg-15` |
| 16:0x | 采访卡 `request-10` `pending`：对外说明强调什么、哪些不提 | broker-state |

## 已暴露的问题（待本轮结束后进 TODO）

### O1 计划批准之后还要第二句「请写入」才肯动手

计划卡已经被人 `approve_with_note` 批准，批注里**明确写了两个 wiki 路径**，模型仍然把「写入」当成需要第二次授权的动作，先后两次让操作员再回一句「请把这两份内容写入办公室 wiki」。这正好是本轮要抓的形状：**一张不携带任何新信息的卡**——人除了重复上次的批准，没有别的可答。

区别于 `8026e40`（关单收卡）和 `6a3f137`（lane 不自升卡）：这条不是卡的生命周期错，而是模型把「已批准的写入」重新当成待批。要修的不是收卡，而是**计划批准即授予写入**这件事在提示层有没有写清楚，以及 `POST /wiki/write-task` 被拒时（`artifact_path_missing`）有没有把拒绝原因直接告诉模型而不是让它转述给人。

### O2 计划里的交付路径默认落在仓库目录

模型把两个交付写成 `docs/office/Journey18-*.md`，而 core loop 的第 6 步要求交付进 Wiki。人的批注纠正了它，但**这个纠正本该由任务定义给出**：`created_by=human` 的 Issue 没写 `artifact_path` 时，模型就按 cwd（仓库）猜。观察到的后果是：如果人没盯，文件会写进 git 工作树。

### O3 看门狗在计划批准后立刻报「没有子任务」

`watchdog_alert: Approved plan for Journey 18 … has no sub-tasks yet (@cos)` —— 计划刚批准、模型还在同一回合里准备拆 lane，看门狗已经先报了一次。无害，但会制造一次假警报；记下来看它是否每轮都响。

## 尚未发生的（本轮要验的）

- V1 lane 级决策卡：目前 `requests[]` 只有 1 张计划卡 + 1 张采访卡，没有 `decision:JOURN-*`。
- V3 `8026e40`：还没有 lane，所以「关 lane 收卡」尚未触发——需要操作员在一条 lane 跑着时**亲手关掉它**。
- V5/V6：两份交付尚未落盘。
