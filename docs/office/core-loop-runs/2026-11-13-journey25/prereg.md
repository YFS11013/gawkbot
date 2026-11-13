# Journey 25 prereg — 台账是事实，卡要念出事实

写于 2026-11-13，源码 `abd0e24`（D-8 台账 + D-9 单任务解析），隔离工作区 `journey25`（:7910/:7911），免费模型 `beam-free`。

## 1. 这一轮要回答的问题

J24 证明了两件事：集合不能读「定义里声明的文件」（规划器可以整个不写 `deliverables`），而「在错误里点名任务」说服不了一个正在重发的模型（45 次）。两个修法都落在代码里、都有红绿测试，但**没有在现场跑过**：

1. **集合读写入台账**：只要任务通过 `/wiki/write-task` 发布过两个文件，卡就该念出两个，与定义写成什么样无关。
2. **缺 `id` 直接解析**：`define` 不带 `id` 时，工具用该 agent 唯一的未终结任务，结果里说明，且**不再产生错误**。

## 2. 场景（同形，换路径）

> Journey 25：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j25-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j25-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 我只能写下面这 5 条提交里的东西。
> 说完就去做，做完把这件事收尾，我要看到它进 done。

随任务给定：

```
abd0e24 docs(office): Journey 24 — the card lied again, so the model doctored the file again
2be6e14 fix(office): the delivered set follows what the task published
a6e64ce docs(office): Journey 24 prereg — the card names the set, so nobody has to fool the gate
dd6b14f docs(office): Journey 23 — the card lied, so the model doctored the file
3b2d2b8 fix(office): the delivered set is every file the definition names
```

## 3. 操作员动作预算（≤4）

| 编号 | 动作 |
|---|---|
| H1 | 下达场景 |
| H2 | 计划卡：一句话回答受众 |
| H3 | 第一次交付后**就 `task.artifact` 之外的那一页**退回（哪一页由交付时的 `artifact` 决定） |
| H4 | 第二次交付后批准 |

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| E1 | 首次决策卡**列出两个路径**（不论定义里有没有 `deliverables`） | 卡正文 |
| E2 | 返工落在非 artifact 页时，重提卡**不带** "None of those files has changed" | 卡正文 + wiki sha |
| E3 | 退回→重提之间**没有无意义提交**（纯空白/换行） | wiki `git log` |
| E4 | `define` 漏 `id` **不再报错**，且结果里说明解析到了哪个任务 | opencode `part` |
| E5 | ≤4 动作闭合，终态 `done`、`review_state=approved`，0 直写 | `tasks[]`、`part` |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| H-1 | 卡仍只念一个文件（台账未生效） |
| H-2 | 假警告仍在（集合与退回页不一致） |
| H-3 | 模型仍篡改交付物 |
| H-4 | 漏 `id` 仍报错或仍重试 |
| H-5 | 第 5 个操作员动作 |

## 6. 已知限制

- 内容质量不计入缺陷；生产（:7892）不参与。
- 第二个交付文件的 `human_request` 窄缝仍未修，若再撞 403 只记次数。
