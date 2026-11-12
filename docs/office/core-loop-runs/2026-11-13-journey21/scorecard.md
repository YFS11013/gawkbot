# Journey 21 判分卡 — 没有逃生口之后：正门找到了，但重提闸门把人锁在门外

判分依据：`observations.md` 的时间线、broker-state、wiki `git log`、opencode 会话库 `part` 表原始记录。

## 断言

| 编号 | 断言 | 判定 | 证据 |
|---|---|---|---|
| A1 | planning 回合自动启动，期间无人类 comment | **成立** | Issue 建于 16:56:15Z，同一秒 planning 回合入队（queue_wait 78 ms）；整轮 Issue 上没有人类 comment |
| A2 | 交付经 `/wiki/write-task`，提交指回 `cos`，0 条 `external edit` | **成立** | 5 条提交作者全是 `cos <cos@wuphf.local>`；内部页 `8858816`、对外页 `6708e77` + 两次改写；`external edit` 0 条 |
| A3 | 0 次成功直写；直写尝试必须被拒 | **成立（空集）** | 整轮 `write`/`edit`/`patch`/`bash` 调用 0 次——模型没去碰逃生口。D-2 的拒绝证据仍以 `d2p4` 探针为准 |
| A4 | 除卡之外 0 条「请点按钮/请授权」的消息 | **不成立** | 「JOURN-25 … 需要你在界面上批准/启动后即可标记 done」至少 3 条；最后一条直接请操作员允许它对内部页做「无内容改动的 touch」来骗过闸门 |
| V1 | 任务终态 `done` | **原轮次不成立，补跑成立** | 原轮次：`JOURN-2` 停在 `in_progress`，返工已落盘但 `submit_for_review` 被 400 挡在门外（D-3）。装上修复后的二进制、一条消息要求重提，17:56:33 `done` |
| V2 | 至少一个任务进过 `decision`，卡列出交付路径 | **成立（措辞有偏差）** | `request-19`（`decision:JOURN-2`）列了 lane 的 `team/reviews/j21-public.md`；它 17:00:02 升起时 `task.Artifact` 为空，所以还写着「ships no single file of its own」，而退回时 `task.Artifact` 已是内部页 |
| V3 | 内部页 hash ⊆ 给定 5 条；对外页 0 hash | **成立** | 内部页 5 行正是 `2bf9f92/29f3ec0/8400015/167b1f5/3d68468`；对外页 3 条无 hash |
| V6 | 有 tick 落进回合窗口时 0 条 `task_stalled` | **未测** | 0 条告警，但 `task_reminder` 的 5 分钟 override 在整轮里没有产生任何一次 tick（`scheduler_activity` 只有 `throttled`），无法构造重合 |
| V7 | 退回后重提：字节变化，或卡带「has changed」警告 | **成立（补跑后两侧都成立）** | 退回时刻对外页 `e88cd08c…`，改写后 `2169e9a3…`（commit `0453dac`），两条意见都对着改了；卡没有重升，因为重提本身被 D-3 挡住了；补跑里 `request-54` 没有那句警告，说明集合摘要已经被识别为「变了」 |

## 失败模式

| 编号 | 是否发生 |
|---|---|
| F-a 停在 planning/decision 不动 | **部分**：不是「不动」，是返工后被闸门锁死（D-3） |
| F-b 断言外的多余卡 / lane 自升卡 | **发生**：`JOURN-22`、`JOURN-25` 两个重复 lane + `request-29`/`request-36`/`request-43` 三张多余卡 |
| F-c 被拒之后转去求人 | **部分**：`human_request` 授权路径 0 次成功，但出现多条「请你在界面上批准/启动」 |
| F-d 退回后重提、字节没变且无警告 | 未发生（字节确实变了） |
| F-e 超预算仍不收敛 | 预算正好 4 次，第 4 次（驳回重复 lane）之后环路仍未闭合 |
| F-f 直写成功（D-2 回归） | 未发生 |

## 结论

**D-1 在真模型下成立**：bot 自建的 Issue 拿到 planning 回合、计划卡自己升起，全程没人碰 Issue——这是 J20 里靠人留言才换来的那一步。

**D-2 没有回归**，但这一轮**没有锻炼到它**：模型从头到尾没尝试直写。逃生口关掉之后，它面对 `human_request` 被拒的反应是继续在正门上试（先试错 13 次：6 次非人类消息、4 次没明确要求直写、3 次路径未声明），最后自己 `define` 出 `artifact_scope` 再写——这是想要的行为。

**新露出的是闸门之间的缝**：卡按**交付集合**说话，重提闸门按 **Issue 自己的单个 artifact** 判定；返工落在 lane 上时，`submit_for_review` 永远 400。办公室因此新建了两个重复 lane、升了三张多余卡，最后反过来请求操作员批准一次「无内容改动的 touch」。这条已修（D-3）。

**重复 lane 的代价**：`JOURN-25` 与 `JOURN-14` 声明同一个 wiki 路径，5 次覆写 `team/reviews/j21-public.md`（台账每次都记 `artifact_overwritten`，写入照常成功）。「同一路径能否被第二个任务写」是 `TODO-NEXT.md` 里未拍板的产品决策，本轮不修，只把它从「理论问题」变成「有次数、有 commit 的实况」。

**第二个新缺陷**：`team_task` 非 `create` 动作漏传 `id` 会回 `404 task not found`，模型一轮内重试了 65 次。已修（D-4）。

## 补跑后的结论

修复装上以后，同一个工作区用一条消息就闭合了：`submit_for_review` 不再 400，卡把 lane 文件与 Issue 自己的文件都列了出来，批准后任务 `done`、重复 lane 保持 canceled、环路自带的维护任务也跟着收尾。**「退回 → 返工 → 重提 → 批准 → done」第一次在真模型上跑通**。

## 下一步

1. 用带 D-3/D-4 修复的二进制重跑同一场景（Journey 22）：目标是把「退回 → 返工 → 重提 → 批准 → done」这一段第一次跑通，并让 V7 的卡侧（「has changed」警告）也被锻炼到。
2. `request-36`（`dedupe_key` 为空的聊天式采访卡）值得单独看一眼：它和 `request-43` 是同一件事的两张卡，属于重复面。
3. 容器卡正文与 `task.Artifact` 的措辞不一致（「ships no single file of its own」vs 实际有 artifact）：D-3 修的是判定，这句话本身也该跟着集合走。
