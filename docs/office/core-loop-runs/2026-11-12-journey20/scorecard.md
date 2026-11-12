# Journey 20 判分卡 — 环路闭合了，但不是它自己走的

判分依据：`observations.md` 的时间线与 broker-state / wiki git log / opencode 会话库原始记录。

## 断言

| 编号 | 断言 | 判定 | 证据 |
|---|---|---|---|
| V1 | 父 Issue 终态 `done`，`artifact=null`，无 `hollow_completion` | **部分成立** | `done` ✓、无 `hollow_completion` ✓；`artifact=team/reviews/j20-internal.md` ✗——本轮没有 lane，Issue 自己就是交付者，`artifact=null` 的前提不成立（J18 的容器形状没出现） |
| V2 | 恰好 1 张容器卡、列全两个路径；lane 级卡 0 张 | **不成立（形态不同）** | `requests[]` 只有 1 张 **plan-approval** 卡；模型没有拆 lane，直接自己做完，所以容器卡与 lane 卡都不存在 |
| V3 | 全程 0 条要人类授权的消息 | **不成立** | 模型在 `msg-6` 明确要人「在 Issues 板上启动」；这条要求指向一个不存在的控件（D-1） |
| V4 | 两个文件都在 wiki，提交带各自 lane 的任务戳 | **一半成立** | 内部页 commit `b0aa7ee`（`/wiki/write-task`，但无 `(task J-2)` 戳）；对外页 commit `195f013` 由 **external edit** 合成作者提交（D-2） |
| V5 | 内部页 hash ⊆ 给定 5 条；对外页 0 个 hash | **成立** | 抽出 `{70f5154,b315a39,3d6b115,9534002,a841336}`，无清单外 hash；对外页 0 个 hash |
| V6 | 任何 `task_stalled` 不落在回合执行窗口内 | **未测** | 0 次告警，但也没有 tick 落进回合（最长 7.7 min vs 60 min 间隔） |
| V7 | 退回后重提：字节变了 或 卡片警告 | **未测** | 本轮没有发生退回；D-1 让话题在 planning 卡住了 |

## 失败模式

| 编号 | 判定 | 说明 |
|---|---|---|
| F-a 父 Issue 停在 decision/review 不动 | **成立（14 分钟）** | 停在 `planning`，从 20:01:32 到 20:07:53 无任何推进；不是我介入就是 60 分钟 |
| F-b 出现第二张容器卡 / lane 自升卡 | 不成立 | 只有 1 张 plan-approval 卡 |
| F-c 又要人类授权一句话 | **成立** | 见 V3；且要求指向不存在的控件 |
| F-d 退回后字节未变且无警告 | 未触发 | — |
| F-e 超过 4 次动作仍不收敛 | 不成立 | 用了 3 次 |

## 结论

**这一轮回答的问题**：Issue 能进 `done`。用的是 3 次操作员动作、6 分钟的真实干预、外加一次身份冒充（我以 `human` 名义在 Issue 上留言）——最后这条正是 D-1 的解药，不该存在。

**暴露的两条新缺陷**：

- **D-1**：bot 给自己开的 Issue 不会得到它的 planning 回合（`owner == lead == actor` 时通知被整段跳过），因此 plan-approval 卡永不升起，任务停在 `planning`，而模型反过来要求人类点一个不存在的按钮。这是**环路不自主**的直接证据，不是文案问题。
- **D-2**：`/wiki/write-task` 的路径授权对有文件系统权限的 bot 只是建议。模型在 `human_request` 被拒后改用普通 `write` 工具落盘，wiki worker 以 "external edit" 接受，提交作者变成一串 id，`index/all.md` 还给它建了索引。授权被绕开且不留痕迹。

**没测到的**：O3 守卫的真机行为（没有 tick 跨越回合）、退回重提路径（本轮没走到）、容器卡（模型没拆 lane）。

## 修复后复核（同日，工作区 `j20b`）

D-1 修完并在生产部署后，用同一形状重跑，**操作员只发了那一句消息，没有在 Issue 上留言**：

| 时刻 | 事件 |
|---|---|
| 20:33:14 | 聊天回合开始（`msg-1`） |
| 20:33:37 | **planning 回合自动入队并开始**（修复前不存在这一条） |
| 20:39:06 | 聊天回合结束（351 s），Issue `JB-2` 已建 |
| 20:43:26 | planning 回合结束（589 s） |
| — | `request-9` `pending` `plan-approval:JB-2` **自己升起** |

对照：修复前同一形状是「没有第二个回合、没有卡、模型要人去点一个不存在的按钮」，等了 14 分钟最后由我以 `human` 名义留言才推进。这一条把 V3/F-a 的根因从「模型措辞不好」降级为「代码漏了唤醒」。

`hollow_completion` 这一项在本轮结束后回查过（从 trash 恢复 `j20` 读 `broker-state.json`）：`J-2` 的字段不存在 = false，判分卡的「✓」有据。
