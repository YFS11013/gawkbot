# Journey 22 观测记录

工作区 `journey22`（broker :7910 / web :7911），免费模型 `beam-free`，源码 `c4b0664`。
prereg：`prereg.md`（commit `1c209eb`）。

broker 记录的时间是 UTC 但日期落在 11-12（工作区时钟比本机日期慢一天），下文一律照抄记录值；本机北京时间为 +0800，两者相差 8 小时。

## 时间线

| 记录时刻 | 本机时刻 | 事件 |
|---|---|---|
| 23:51:08 | 07:51:08 | 我把 `task_reminder` 压到 5 分钟（`scheduler_activity` 只留一行 `throttled`） |
| 23:51:58 | 07:51:58 | **H1** 下达场景（`/notifications/automation`，`msg-1`） |
| 23:51:58 | 07:51:58 | Issue `JOURN-2` 创建，planning 回合同一秒入队 |
| 23:52:45 | 07:52:45 | 内部页落盘（`wiki` commit `6232e21`，作者 `cos`）——**在计划被批准之前** |
| 23:53:01 | 07:53:01 | 对外页落盘（`11061f4`） |
| 23:53:30 | 07:53:30 | 计划卡 `request-7` 升起（`plan-approval:JOURN-2`） |
| 23:53:32 | 07:53:32 | **`JOURN-9` 被创建**：同一件事的第二个任务，同样的两个路径 |
| 23:54:13 | 07:54:13 | 对外页被第二次写入（`57485e1`，去掉「404」措辞）；模型回「JOURN-9 停在 planning/parked，需你启动/批准」 |
| 23:56:51 | 07:56:51 | **H2** 批准 `request-7`（同时给了受众与措辞要求） |
| 23:57:18 | 07:57:18 | **`JOURN-2 [done]`**——模型用 `team_task action=complete` 直接关掉，**没有 review、没有决策卡** |
| 23:58:11 | 07:58:11 | `JOURN-9` **覆写** `team/reviews/j22-public.md`（已被 `JOURN-2` 交付的文件，台账记 `artifact_overwritten`） |
| 23:58:51 | 07:58:51 | `JOURN-9` 自己把重复任务 cancel 掉（`request-13` 随之 cancel） |

## 断言与证据

**A1 成立**：Issue 创建与 planning 回合入队同一秒；整轮 Issue 上没有人类 comment（我只发了 `msg-1` 那条渠道消息）。

**A2 成立**：两页共 4 条提交，作者全是 `cos <cos@wuphf.local>`；0 条 `external edit`。

```
6232e21 Journey 22 internal change list
11061f4 Journey 22 public note
57485e1 Fix j22-public.md: remove internal term '404' from user-facing note
0488bcd Fix public note: remove internal term 404 per acceptance criteria
```

**A3 成立（空集）**：整轮工具调用 `team_task` 28 次、`team_wiki_write` 8 次、`team_wiki_read` 7 次、`glob` 4 次、`read` 3 次、`team_broadcast` 1 次、`team_tasks` 1 次、`human_message` 1 次；**`write` / `edit` / `patch` / `bash` 0 次**。

**A4 不成立**：07:54:13 一条「JOURN-9 停在 planning/parked，**需你启动/批准**」。

**B1 未测**：整轮没有发生过退回——`JOURN-2` 在计划被批准后 27 秒里直接从 `in_progress` 走到 `done`，没有决策卡，没有可退回的东西。

**B2 不成立**：`JOURN-9` 与 `JOURN-2` 声明同样的两个路径，创建于计划卡之后 2 秒；它不仅抢写，还在 `JOURN-2` 已经 `done` 之后把交付文件改了一次（23:58:11）。

**B3 不成立**：`define` 共 11 次，其中 **7 次没带 `id`**。D-4 的新措辞确实每次都立刻回 `team_task action="define" needs the task id — pass id=<the id returned by create>`，但模型照原样重试了 7 次（J21 是 65 次 404）。工具层的提示**降低了成本、没改变行为**。

**V1 成立但空**：`status=done`、`review_state=not_required`、`lifecycle_state=approved`、`hollow_completion` 未置位。任务确实 done 了——只是没人看过交付物。

**V2 不成立**：整轮只有两张 `plan-approval` 卡，**没有任何任务进过 `decision`**，`requests[]` 里不存在 `decision:*`。

**V3 成立**：内部页 5 行正是给定 5 条（`c4b0664`/`1de26fd`/`a2041d7`/`390e653`/`46e83c7`）；对外页 2 行、0 个 hash。

**V6 未测**：`task_reminder` 的 `interval_override=5` 生效（job 上能看到），`scheduler_activity` 只有一行 `throttled`；整轮 0 条 `task_stalled`。但系统 job 不落 tick 记录，**分不清「tick 没发生」和「tick 被顺延」**——这是观测缺口，不是缺陷。

**V7 未测**：没有退回，就没有重提。

## 根因：为什么 `done` 没有卡

`team_task action=complete` 在 `broker_tasks_mutation_service.go` 里：

```go
case "complete":
    ...
    } else if taskNeedsStructuredReview(task) {
        task.status = "review"; task.reviewState = "ready_for_review"
    } else {
        markTaskDone(task, now)      // ← 到这里就结束了
    }
```

而 `taskNeedsStructuredReview` → `taskWorkRequiresLocalExecution(owner, title, details)`，是**对英文关键词做子串匹配**（`code`/`repo`/`test`/`implement`…）。`JOURN-2` 的标题和描述是中文，写的是 wiki 页，一个关键词都不中 → `false` → `complete` 直接 done。

同一轮里 `JOURN-9` 先用 `submit_for_review`（于是 `status=review`，卡会升起）——**人类能不能看到交付物，取决于模型挑哪个动词**。这正是 J12 已经认定过的「人类主权的旁路」，只是 J12 挡住的是**容器**（有已交付子任务的 Issue），**叶子**这一侧一直敞着。

## 附带观察

- 模型在**计划被批准之前**就把两页写完了（07:52:45 / 07:53:01，卡是 07:53:30 才升的）。计划卡因此变成「事后通知」，`plan-approval` 对内容没有任何约束力。
- `JOURN-9` 在 `JOURN-2` 已 `done` 之后覆写它的交付文件，台账记了 `artifact_overwritten` 但写入照常成功——「同一路径能否被第二个任务写」仍是 `TODO-NEXT.md` 里未拍板的产品决策。
