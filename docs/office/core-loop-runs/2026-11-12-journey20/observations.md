# Journey 20 现场记录 — 最小完整环路

工作区 `~/.wuphf-spaces/j20`（broker :7910 / web :7911，单 bot `cos`，免费模型 `beam-free`），源码 `b315a39`，prereg `70f5154`。全部时刻 +08:00。

## 时间线

| 时刻 | 事件 | 证据 |
|---|---|---|
| 19:53:42 | 工作区启动（`workspace create j20`） | registry |
| 19:53:50 | 回合 1 开始 | latency log |
| 19:53:5x | 我以 `nex` 身份投递场景（`POST /notifications/automation`，`msg-1`） | broker-state `messages[]` |
| 19:57:16 | 模型在**同一回合内**创建 Issue `J-2`（owner/actor 都是 `cos`） | `J-2.created_at` |
| 19:58:27 | `wiki write-task refused: task=J-2 … artifact path not declared` | broker.log |
| 19:59:19 | 模型给 J-2 `define` 了 `artifact_path` 后，同一路径写入成功 | broker.log |
| 19:59:54 | 对外页以 **"external edit"** 落库：commit `195f013`，作者 `83509926@wuphf.local` | wiki git log |
| 20:01:32 | 回合 1 结束（`total_ms=462279`），模型报「J-2 卡在 planning，只能由你在 Issues 板上启动」 | latency log、`msg-6` |
| 20:06:5x | 我以 `human` 身份在 J-2 上发 comment（操作员动作 2） | `actions[] task_updated` |
| 20:07:06 | 两个 task 回合启动（46.9 s / 69.4 s） | latency log |
| 20:07:53 | **plan-approval 卡 `request-11` 升起**（`dedupe_key=plan-approval:J-2`） | `requests[]` |
| 20:11 | 我 `approve`（操作员动作 3） | `requests[]` |
| 20:11:1x | `J-2 status=done artifact=team/reviews/j20-internal.md` | `tasks[]` |

## 结果

环路**闭合了**：Issue 从 `planning` 走到 `done`，交付物挂在 Issue 上。但要看清代价——它不是自己走完的。

### D-1（本轮核心缺陷）：bot 给自己开的 Issue 永远等不到它的 planning 回合

`notifier_targets.go:taskNotificationTargets`：

```go
if owner == lead {
    if lead != "" && lead != actor { addImmediate(lead) }
    return immediate, delayed
}
```

模型在**自己的回合里**创建 `J-2`，于是 `owner == lead == actor == cos` → 一个 target 都没有 → `deliverTaskNotification` 返回 false → planning 回合从未入队。而 plan-approval 卡只由 planning 回合结束时升起（`headless_opencode.go:386`），所以卡永远不会来。任务停在 `planning`：没有卡可答，没有回合会跑，唯一出路是 60 分钟后的 stall watchdog。

更要命的是模型据此告诉人类「只能由你在 Issues 板上启动」——**这个控件不存在**。代码里 `planning → running` 只有一条路：答那张 plan-approval 卡（`applyPlanApprovalAnswerLocked`），而卡没升起。

我用人身份在 Issue 上留了一句 comment 才把它唤醒（actor != owner → 立即派发）。这条口子说明：**只有当任务被别人碰一下，bot 自己开的工作才会开始。**

### D-2：授权闸门被一次普通文件写入绕过，而且绕得毫无痕迹

对外页的完整链（opencode 会话库 `part` 表原始记录）：

1. 模型调用 `team_wiki_write`，带 `human_request: "msg-1"` → **被拒**：`team_wiki_write human_request "msg-1" is not a human-authored message`（闸门是对的）；
2. 紧接着它用了 `write` 工具 → `Wrote file successfully.`，直接写进 `~/.wuphf-spaces/j20/.wuphf/wiki/team/reviews/j20-public.md`；
3. wiki worker 把这当成 "external edit"，以合成作者 `83509926@wuphf.local` 提交 `195f013`，还顺手把 `index/all.md` 加上了一条。

所以对**任何有文件系统权限的 bot**，`/wiki/write-task` 的路径授权是建议而非闸门：被拒之后换个工具就能落地，而且提交历史里看不出来是谁写的（作者是一串 id，不是 `cos`）。

### 旁的观测

- 模型在计划未批准时试图建子任务，被 409 拦住：`parent J-2 is still in planning — its plan has not been approved yet`。这道门是好的。
- 内部页走的是 `/wiki/write-task`（`wiki write-task: task=J-2 … bytes=461`），但没有像 J19 那样的 `(task J-2)` 戳——这条路径的提交信息只用了模型给的 message。commit `b0aa7ee cos Journey 20: internal changelog`。
- 计划卡的正文取自回合的最终消息（「Done — both review files verified…」），不是一份计划。人类看到的是"他要开始干活"，读不到要干什么。
- 全程 `watchdog_alert` 0 次；O3 的守卫没有在真机上被测试到（提醒间隔 60 分钟，回合最长 7.7 分钟，没有 tick 落进回合里）。

## 操作员动作

| # | 动作 | 是否必要 |
|---|---|---|
| 1 | `nex` 投递场景 | 是 |
| 2 | 以 `human` 身份在 J-2 上 comment「开始吧」 | **是，且不该是**——它只是 D-1 的临时解药 |
| 3 | 批准 plan-approval 卡 | 是（设计如此） |

预算 4，用了 3。
