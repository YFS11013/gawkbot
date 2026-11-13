# Journey 26 prereg — 两扇门,一本台账

写于 2026-11-13，源码 `246ff08`（D-12 归属修复，md5 `9b5a9f37…`），隔离工作区 `journey26`（:7910/:7911），免费模型 `beam-free`。

## 1. 这一轮要回答的问题

J25 把「集合」这件事推进到最后一步：台账记 `/wiki/write-task`，但返工走了人工授权门 `/wiki/write`，`task_id=None`——任务发布了两页,台账只有一页,卡只念一页。修法是给那扇门一个**仅用于归属**的 `task_id`（工具侧在只有一个未终结任务时自动填）。

这一轮要在现场看到：**人工门写进去的那一页,进不进得了任务台账,卡会不会因此念出两页。**

## 2. 场景（同形，换路径）

> Journey 26：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j26-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j26-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 我只能写下面这 5 条提交里的东西。
> 说完就去做，做完把这件事收尾，我要看到它进 done。

随任务给定：

```
246ff08 docs(office): Journey 25 — the write happened, the task never learned it
903054a fix(office): a human-authorized write still belongs to a task
3aa1c41 docs(office): Journey 25 prereg — the ledger is the fact, the card reads the ledger
abd0e24 docs(office): Journey 24 — the card lied again, so the model doctored the file again
2be6e14 fix(office): the delivered set follows what the task published
```

## 3. 操作员动作预算（≤4）

| 编号 | 动作 |
|---|---|
| H1 | 下达场景 |
| H2 | 计划卡：一句话回答受众 |
| H3 | 第一次交付后就 **`task.artifact` 之外的那一页**退回（并在文案里明确授权那条 wiki 路径——这正是走人工门的条件） |
| H4 | 第二次交付后批准 |

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| F1 | 只出现一个顶层任务（无重复 lane） | `tasks[]` |
| F2 | 返工之后那张卡**列出两个路径**（人工门写的那页进了台账） | 卡正文 × `tasks[].artifacts` |
| F3 | 重提卡不带 "None of those files has changed" | 卡正文 |
| F4 | 退回→重提之间没有无意义提交（纯空白/换行） | wiki `git log` |
| F5 | `define` 漏 `id` 报错为 0（单任务时自动解析） | opencode `part` |
| F6 | ≤4 动作闭合，终态 `done`、`review_state=approved` | `tasks[]` |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| I-1 | 又出现重复 lane（第三次）——它同时会破坏 F5 的前提 |
| I-2 | 人工门写的页仍不进台账（D-12 未生效） |
| I-3 | 卡仍只念一个文件 |
| I-4 | 模型改动交付物的无意义字节 |
| I-5 | 第 5 个操作员动作 |

## 6. 已知限制

- 内容质量不计入缺陷；生产（:7892）不参与。
- 同一路径独占（重复 lane）仍是未拍板的产品决策，本轮只记录发生与否。
