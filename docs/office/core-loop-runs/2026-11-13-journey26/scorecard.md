# Journey 26 scorecard — 两扇门，一本台账

2026-11-13，源码 `246ff08`（D-12 归属修复），工作区 `journey26`（:7910/:7911），模型 `opencode-go/beam-free`。
操作员动作 **4/4**：H1 下达、H2 批准 `JOURN-3` 的计划、H3 退回、H4 批准。

## 断言

| 编号 | 判定 | 证据 |
|---|---|---|
| F1 只出现一个顶层任务 | **不成立** | `JOURN-7`（第三方重复）标题就是「Journey 26：两页变更说明已写入 wiki」——它是为了**结案**而建的，不是为做事 |
| F2 返工后卡列出两个路径 | **成立** | `request-24`（首次交付）与 `request-36`（返工后）都是「The deliverables are:」+ 两行路径；`JOURN-3.artifacts` = 两页 |
| F3 重提卡无假警告 | **成立** | `request-36` 正文无任何 "unchanged" |
| F4 无无意义提交 | **成立** | 返工只有一个提交 `6cbd516 fix: add parenthetical clarification to delivered-set line per human feedback` |
| F5 漏 `id` 报错为 0 | **不成立** | 4 次（2 次「definition goal is required」、2 次「task not found」）。两个未终结任务并存，唯一解析不成立 |
| F6 ≤4 动作闭合 | **成立** | 4/4；`JOURN-3` `done`/`approved`；0 直写 |

## 失败模式

| 编号 | 是否发生 | 说明 |
|---|---|---|
| I-1 重复 lane | **是（第三次）** | J22 覆写、J25 拆分、J26 建了个「已写入 wiki」的空任务 |
| I-2 人工门写的页不进台账 | 未触发 | 本轮两次写入都走 `/wiki/write-task`（`task_id=JOURN-3`），D-12 的归属支路没被锻炼 |
| I-3 卡只念一个文件 | 否 | 首次交付就念了两个 |
| I-4 篡改交付物 | 否 | 返工是真实内容修改 |
| I-5 第 5 个动作 | 否 | 4/4 |

## 新缺陷

### D-14 一个请求被拆成两个顶层任务（已修）

三轮里第三次。J26 的重复更露骨：`JOURN-7` 不声明任何交付路径，标题写着「两页变更说明已写入 wiki」——它是模型为了**把工作结案**而新建的任务。它没有路径可写，于是整轮的错误都堆在它身上：7 次 `409 … is still in planning`（`complete`）、4 次 `403 … parked`（`approve`）、3 次 `403 artifact path not declared`（写文件）、1 次 interview 工具超时（还升起一张「是否批准？」的面试卡）。同时它把 D-9 的唯一解析前提废掉（两个未终结任务 → 4 次 `define` 报错）。

修法：`applyOpenTopLevelExclusivityLocked` —— bot 在同一个频道里已经有未终结的顶层任务时，新建的顶层任务如果**声明了同一个写入范围**或者**标题在复述那件事**，一律 `409`，文案点名那个任务并指回 `submit_for_review` / `complete`，`allow_duplicate=true` 仍是逃生门，人类与内部 actor 不受影响。

标题判定用了专门的 `topLevelTitlesRestate`：lane 级的阈值（`laneTitlesSharePhrase`）对顶层文案不够——「把这批改动写成两页（…）」与「两页变更说明已写入 wiki」的包含度只有 0.375，低于 lane 的 0.5 门槛，但它们共享三个命名这件事的 bigram（两页/变更/说明）。顶层判定用「共享 ≥3 且 ≥0.35」。

测试 `TestBotCannotSplitOneRequestAcrossTwoTopLevelTasks`：同范围、复述标题都必须被拒，且文案里要有任务 id 与 `submit_for_review`；不相关的工作、`allow_duplicate`、人类创建都必须放行。红证：把标题规则短路 → 「restates the open task: a second top-level task for the same work must be refused」。
