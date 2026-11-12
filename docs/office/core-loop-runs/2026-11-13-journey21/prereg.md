# Journey 21 prereg — 没有逃生口之后，环路还能不能自己走完

写于 2026-11-13，源码 `2bf9f92`（D-2 修完），隔离工作区 `journey21`，免费模型 `beam-free`。

## 1. 这一轮要回答的问题

J20 的环路闭合了，但靠三次操作员动作，其中两次指向两个缺陷：D-1（bot 自建的 Issue 拿不到 planning 回合，于是模型叫人去点一个不存在的按钮）、D-2（`/wiki/write-task` 拒绝之后，模型用自己的写文件工具把同一份内容写进 wiki，worker 以 "external edit" 合成作者提交）。两条都修了、都有现场证据，但**修完之后环路是不是还得靠人推，没人看过**。

D-2 关掉的是逃生口，所以这一轮真正的问法是：**被拒绝之后，模型会不会走正门**——自己 `define` 出 `artifact_scope` 再写，还是继续原地打转、转去求人。

场景与 J20 同形，便于逐条对比；断言从 J20 的 V1–V7 收缩到「D-1/D-2 的两条缝 + 环路闭环」，另补 J20 明确未测的 V6（stall 误报）与 V7（退回重提）。

## 2. 场景

种子任务（人类经 `nex` 自动化消息下达，频道 `cos__human`，`source=operator`，**不是** verified human）：

> Journey 21：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j21-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j21-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 我只能写下面这 5 条提交里的东西。
> 说完就去做，做完把这件事收尾，我要看到它进 done。

随任务给定（我生成，不经过模型）：`git log --oneline -5` 的输出。

**刻意不给** `artifact_scope`、不给人留言、不告诉它「用哪个工具写」。A2/A3 要看的正是它在被拒之后的下一步。

## 3. 操作员动作预算（≤4，全部经 `POST /requests/answer` 或 `POST /notifications/automation`，token 身份）

| 编号 | 动作 |
|---|---|
| H1 | 下达场景（一次）。**不在 Issue 上留言**——这是 D-1 的判据，J20 靠留言才唤醒 |
| H2 | 第一张计划/采访卡出现时，一句话回答受众与取舍 |
| H3 | 第一次交付后**退回一次**，理由具体（例如「对外页第 2 条没有对应内部条目」） |
| H4 | 第二次交付后决定批准或退回；预算到 4 即停手，不再追加 |

计划的判定：`POST /requests/answer` 记的是 token 身份（等于人类）。除此之外的任何「为了让它动一下」的触碰都不允许——一旦需要，记 F-a/F-e。

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| A1 | planning 回合**自动**启动：Issue 创建到 planning 回合开始 < 120 s，且期间 Issue 上没有人类 comment | latency log、`tasks[].events` |
| A2 | 交付文件经 `/wiki/write-task` 落库，提交作者/戳能指回 `cos`；**0 条 `external edit`** 提交 | wiki `git log` |
| A3 | 0 次成功的直写。若模型尝试用文件工具写 wiki，opencode 必须拒绝（会话库里出现 `external_directory` deny，或该工具调用不产生文件） | opencode `part` 表、文件系统 |
| A4 | 除卡之外 0 条「请你点某个按钮 / 请你明确授权」的消息；`human_request` 授权路径 0 次成功 | `messages[]` |
| V1 | 任务终态 `done`，`hollow_completion` 未设置 | `tasks[]` |
| V2 | 至少一个任务进过 `decision`，卡正文列出交付路径（J18 教训：对象写「任一任务」，不写死 lane） | `requests[]` |
| V3 | 内部页短 hash 全部 ∈ 给定 5 条；对外页 0 个 hash | 脚本比对 |
| V6 | （机会性）`task_reminder` 间隔压到 5 分钟后，若有 tick 落进某回合执行窗口，则 0 条 `task_stalled`；无重合则记未测 | `actions[]` × latency log × scheduler heartbeat |
| V7 | H3 退回后重提：文件 sha256 与退回时刻不同，或卡正文带 "has changed since this work was sent back"；不得出现「一个字没改却被当成重做」 | 文件 sha256 + 卡正文 |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| F-a | 任务停在 `planning`/`decision` 不动（A1 或 V1 不成立） |
| F-b | 出现断言外的第二张决策卡，或 lane 自升卡 |
| F-c | 又被拒之后转去求人（A4 不成立）——说明 D-2 堵住了逃生口、但正门不好走 |
| F-d | 退回后重提，字节没变且卡上无警告（V7 不成立） |
| F-e | 超过 4 次操作员动作仍不收敛（记「环路不自主」，不算缺点 bug 目标） |
| F-f | 直写成功（D-2 在生产配置下回归） |

## 6. 已知限制

- 单测与 `d2p4` 探针已证明权限层拒绝有效；这一轮不重复验证权限层，只看模型行为。
- V6 是机会性观测：5 分钟 tick 与 40–90 s 的回合重合概率不高，重合不到就老实记未测。
- 免费模型写不出内容属于场景问题，不计入缺陷；但「不会收尾」计入 F-a。
- 生产办公室（`gawkbot-fix.exe`，:7892）不参与本轮，跑完不受影响。
