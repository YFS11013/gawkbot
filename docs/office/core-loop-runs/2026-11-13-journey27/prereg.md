# Journey 27 prereg — 拒绝之后，模型知不知道该收尾

写于 2026-11-13，源码 `73c3d4b`（D-14 顶层独占修复，md5 `9de151ca…`），隔离工作区 `journey27`（:7910/:7911），免费模型 `beam-free`。

## 1. 这一轮要回答的问题

J22 / J25 / J26 连续三轮把「一句话的请求」拆成两个顶层任务。J26 的重复更糟：它不声明任何路径，标题就是「两页变更说明已写入 wiki」——模型为了**结案**另开了一个任务，然后把整轮 22/25 次工具错误都花在它身上。

D-14 在 create 入口拦下了它。但**拦下不等于解决**：J24 的教训是「只说不行」会产生 45 次重发。所以这一轮真正要看的不是「有没有第二个任务」（守卫生效就是 0），而是：

> 被拒绝之后，模型是**回到那个已经开着的任务上继续做事**，还是**换个说法再开一个**（标题改一改就绕过判据），还是**原地重试**？

## 2. 场景（与 J26 同形，这正是它拆过的那一版）

> Journey 27：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j27-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j27-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 我只能写下面这 5 条提交里的东西。
> 说完就去做，做完把这件事收尾，我要看到它进 done。

随任务给定：

```
73c3d4b docs(office): Journey 26 — the card told the truth, and the model stopped cheating
f0868ea fix(office): one request is one acceptance surface
b033a2c docs(office): Journey 26 prereg — both doors, one ledger
246ff08 docs(office): Journey 25 — the write happened, the task never learned it
903054a fix(office): a human-authorized write still belongs to a task
```

「做完把这件事收尾，我要看到它进 done」是 J26 触发结案任务的同一句话，本轮不改。

## 3. 操作员动作预算（≤4）

| 编号 | 动作 |
|---|---|
| H1 | 下达场景 |
| H2 | 计划卡：批准（一句话补充受众） |
| H3 | 第一次交付后退回一页（真实内容要求） |
| H4 | 第二次交付后批准 |

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| F1 | 顶层任务**只有一个**（D-14 生效） | `tasks[]` |
| F2 | 被拒绝的 create **≤1 次**，且之后没有换标题/换范围的第二次尝试 | opencode `part` 的 `task create` 调用 |
| F3 | 被拒绝之后模型回到原任务干活（该任务出现 plan / 交付 / comment） | 任务时间线 |
| F4 | 两页都进台账，卡念两个路径 | `tasks[].artifacts` × 卡正文 |
| F5 | `define` 漏 `id` 报错为 0 | opencode `part` |
| F6 | ≤4 动作闭合，终态 `done`、`review_state=approved`，0 次直写 wiki（除非人工门场景） | `tasks[]` / 写台账 |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| I-1 | 模型换个标题绕过守卫，仍然开出第二个顶层任务（守卫的判据不够） |
| I-2 | 模型原地重试同一个 create ≥2 次（拒绝文案没有给出路） |
| I-3 | 被拒之后模型不再做任何事（守卫把任务卡死） |
| I-4 | 台账只记一页 / 卡只念一页（D-8 / D-12 回归） |
| I-5 | 第 5 个操作员动作 |

## 6. 已知限制

- 内容质量不计入缺陷；生产（:7892）不参与。
- J26 遗留的 `JOURN-7` 属于旧工作区，不在本轮范围。
