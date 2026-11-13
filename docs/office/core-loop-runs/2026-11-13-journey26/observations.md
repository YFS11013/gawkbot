# Journey 26 observations — 两扇门，一本台账

工作区 `~/.wuphf-spaces/journey26`，broker :7910，视觉 :7911，`opencode-go/beam-free`。
源码 `246ff08`。判分见 `scorecard.md`。

## 时间线（UTC）

| 时刻 | 事件 |
|---|---|
| 05:0x | H1 下达（j26-*，5 条提交） |
| 0x | `JOURN-3` 创建（两页，`artifact_path=…public.md`） |
| 0x | **`JOURN-7` 自建**：无路径，标题「两页变更说明已写入 wiki」 |
| 0x | 两张计划卡（`request-14` / `request-18`）+ **一张 interview**（`request-9`：「是否批准？」） |
| 0x | H2 只批 `JOURN-3` 的计划 |
| 0x | 首次交付：两页都走 `/wiki/write-task`（`task_id=JOURN-3`）；`j26-internal.md` `e15b017`、`j26-public.md` `2402055` |
| 0x | 决策卡 `request-24`：**「The deliverables are:」两行路径** |
| 0x | H3 退回：只改内部清单，明确授权该 wiki 路径 |
| 0x | 返工 `6cbd516`（真实内容修改，只动内部页） |
| 0x | 决策卡 `request-36`：同样两行路径，**无 "unchanged" 警告** |
| 0x | H4 批准 → `JOURN-3` `done`/`approved`；`JOURN-7` 仍 `in_progress` |

## 三个观察

**1. 台账和卡终于对上了。** 两次验收卡都是「The deliverables are:」+ 两行路径，返工之后也没有任何「文件没变」的假警告，模型也没有去动不该动的字节。这是四轮里第一次**卡说的是真话，模型因此不必作弊**。

**2. 重复 lane 是这一轮唯一的大噪音，而且形态更糟。** `JOURN-7` 不声明任何路径，标题写着「两页变更说明已写入 wiki」——它不是另一份工作，是模型给自己造的**结案通道**。它把整轮的错误都揽了过去：7 次「still in planning」、4 次「parked」、3 次「artifact path not declared」（它没有路径可写）、1 次 interview 工具超时（还升起一张问「是否批准？」的卡）。同时它让 D-9 的前提失效（两个未终结任务），`define` 又报了 4 次错。

也就是说：**重复 lane 不只切碎验收面，它还是错误的主要来源**。J26 的 25 次工具错误里有 22 次出自那个空任务。

**3. 修法要落在 create 的入口。** 三轮的证据（J22 覆写已交付文件、J25 拆分两页、J26 造结案任务）指向同一件事：bot 能在同一个频道里为同一件事开第二个顶层任务。现在 `applyOpenTopLevelExclusivityLocked` 在 create 时就拦：同写入范围、或标题在复述那件事 → `409`，文案点名已有的任务并指回 `submit_for_review` / `complete`。

标题判定上 lane 级的阈值不够用（0.375 < 0.5 门槛），顶层改用「共享 ≥3 个 bigram 且包含度 ≥0.35」。这不是把阈值调松到全局，而是给这一类加一条自己的判据——lane 与顶层是两种文案形状。

## 工具面

`team_task` 41 次：7 次 `complete` 撞「still in planning」（全在 `JOURN-7`）、4 次 `approve` 撞「parked」（也在它）、4 次 `define` 报错；`team_wiki_write` 12 次调用里 5 次失败（3 次 403「artifact path not declared」、2 次 `human_request "msg-1" is not a human-authored message`）。

写入全部走 `/wiki/write-task`、`task_id=JOURN-3`，所以 D-12 的人工门归属支路本轮没有现场数据——它由 `TestHumanAuthorizedWriteIsAttributedToItsTask` 与 `TestHumanRequestedWikiWriteCarriesTheSoleTaskID` 覆盖。

## 未测 / 已知限制

- D-12 的「人工门 → 台账」现场未触发（本轮两次写入都是任务声明范围内）。
- 内容质量不计入缺陷（对外页仍是三行泛泛的话，但没出现内部术语）。
- 生产（:7892）不参与本轮。
- `JOURN-7` 仍 `in_progress`：它是在修复**之前**建的，不会被追溯清理。
