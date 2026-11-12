# Journey 22 prereg — 一条消息能不能自己走到 done

写于 2026-11-13，源码 `c4b0664`（D-3/D-4 修完），隔离工作区 `journey22`，免费模型 `beam-free`。

## 1. 这一轮要回答的问题

J21 把环路卡在重提上，跑出 D-3（闸门只看 Issue 自己的单文件）和 D-4（`team_task` 漏 `id` 回 404）。两条都修了，而且修完**在同一个工作区里补跑通了**：退回 → 返工 → 重提 → 批准 → done。

但那次补跑有一条拐杖：操作员发了一条明确的消息「请直接 `submit_for_review` JOURN-2，不要再新建 lane」。所以真正还没被回答的问题是——

> **没有人告诉它该重提，它会不会自己重提？**

以及两个附带问题：J21 里那两条重复 lane（`JOURN-22`/`JOURN-25`，与 `JOURN-14` 抢同一个 wiki 路径）是因为被闸门锁死才分叉出来的，还是模型本来就会这么做？D-4 修完以后，那个 `define` 漏 `id` 的形状还会不会被重试几十次？

这一轮的操作员预算里**没有「叫它重提」这一项**，这是判据本身。

## 2. 场景

种子任务（人类经自动化消息下达，频道 `cos__human`，`source=operator`，**不是** verified human）：

> Journey 22：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j22-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j22-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 我只能写下面这 5 条提交里的东西。
> 说完就去做，做完把这件事收尾，我要看到它进 done。

随任务给定（我生成，不经过模型）：

```
c4b0664 docs(office): Journey 21 closes on the fixed binary
1de26fd docs(office): Journey 21 — the gate that locked the rework out
a2041d7 test(office): the enqueue tests stop racing their own worker
390e653 fix(office): team_task names a missing id instead of 404
46e83c7 fix(office): the resubmission gate compares the delivered set
```

**刻意不给** `artifact_scope`、不给人留言、不解释「怎么重提」。

## 3. 操作员动作预算（≤4，全部经 `POST /requests/answer` 或自动化消息，token 身份）

| 编号 | 动作 |
|---|---|
| H1 | 下达场景（一次）。**不在 Issue 上留言** |
| H2 | 第一张计划/采访卡出现时，一句话回答受众与取舍 |
| H3 | 第一次交付后**退回一次**，理由具体（例如「对外页第 2 条在内部清单里找不到对应的那一行」） |
| H4 | 第二次交付后批准；预算到 4 即停手 |

**明确不做**：不告诉它「去重提」、不新建容器、不替它改任务状态。J21 的补跑那句「请直接 submit_for_review」在本轮属于违规动作——如果这一步不发生，记 F-a'。

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| A1 | planning 回合**自动**启动：Issue 创建到 planning 回合开始 < 120 s，且期间 Issue 上没有人类 comment | latency log、`tasks[].events` |
| A2 | 两页都经 `/wiki/write-task` 落库，提交作者/戳指回 `cos`；0 条 `external edit` | wiki `git log` |
| A3 | 0 次成功的直写（模型尝试写 wiki 时权限层必须拒绝） | opencode `part` 表 |
| A4 | 除卡之外 0 条「请你在界面上批准/启动」的消息 | `messages[]` |
| B1 | **退回后的重提由模型自己发起**：退回之后没有操作员消息，`submit_for_review` 仍然发生 | `actions[]` 时间序 |
| B2 | **0 条重复 lane**：没有新任务声明一个已被别的 lane 交付的 wiki 路径 | `tasks[]`.artifact_scope、`artifact_overwritten` 条数 |
| B3 | D-4 生效：`team_task … needs the task id` 这种错在整轮出现 ≤ 2 次，且模型第二次就带上 `id` | opencode `part` 表 |
| V1 | 任务终态 `done`，`hollow_completion` 未设置 | `tasks[]` |
| V2 | 至少一个任务进过 `decision`，卡正文列出交付路径（含集合，不是单文件） | `requests[]` |
| V3 | 内部页短 hash 全部 ∈ 给定 5 条；对外页 0 个 hash | 脚本比对 |
| V6 | `task_reminder` 压到 5 分钟后，若 tick 落在回合执行窗口内，`task_stalled` 必须为 0（调度器应当把它顺延而不是报警） | `actions[]`、scheduler job 状态、latency log |
| V7 | H3 退回后重提：交付集合 sha256 与退回时刻不同，且**重提后的卡**不带 "has changed since this work was sent back" | 文件 sha256 + 卡正文 |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| F-a' | 退回后模型自己收尾为「等你指示」，不做重提（B1 不成立）——这是本轮最想抓的 |
| F-b | 又出现重复 lane / 断言外的多余卡（B2 不成立） |
| F-c | 又被拒之后转去求人点按钮（A4 不成立） |
| F-d | 退回后重提，字节没变且卡上无警告（V7 不成立） |
| F-e | 超过 4 次操作员动作仍不收敛 |
| F-f | 直写成功（D-2 回归） |
| F-g | 漏 `id` 的形状仍然被重试 ≥ 3 次（B3 不成立，说明工具层的措辞没有改变行为） |

## 6. 已知限制

- V6 在 J21 里被记为「未测」，原因是没分清「tick 没发生」和「tick 被顺延」——读代码后确认：回合在飞时调度器走的是 `UpdateSchedulerJobState(..., "scheduled")` 顺延分支，不报警。本轮改判据：**顺延且 0 报警 = 成立**；连顺延都看不到才记未测。
- 免费模型的内容质量不计入缺陷；「不会收尾」计入 F-a'。
- 生产办公室（`gawkbot-fix.exe`，:7892）不参与本轮。
- 重复 lane 若再次出现，只记录不修：同路径能否被第二个任务写是 `TODO-NEXT.md` 里未拍板的产品决策。
