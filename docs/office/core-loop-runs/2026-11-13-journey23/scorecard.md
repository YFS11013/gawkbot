# Journey 23 scorecard — 交付了文件，就得有人签收

2026-11-13，源码 `b3423e8`（D-5 修复），工作区 `journey23`（:7910/:7911），模型 `opencode-go/beam-free`。
操作员动作 **4/4**：H1 下达 00:37:57、H2 批准计划 00:42:45、H3 退回 00:47:57、H4 批准 00:54:38。

## 断言

| 编号 | 判定 | 证据 |
|---|---|---|
| C1 | **成立** | 00:44:29 `guard_refused \| delivery close: cos tried to close team/reviews/j23-internal.md without asking`；00:44:35 同一回合 `[review]` 改走 `submit_for_review`；00:44:38 决策卡 `request-13` |
| C2 | **部分成立** | 卡确实出现了，但正文只念一个文件：「The deliverable is team/reviews/j23-internal.md」——而定义里 `deliverables` 有两个（D-6） |
| C3 | **成立** | H3（00:47:57）之后到 `request-23`（00:52:38）之间 `messages[]` 只有 `source=operator` 的 H3 本身；无任何提醒、无操作员消息，模型自己重提 |
| C4 | **不成立** | `request-23` 带警告「None of those files has changed since this work was sent back at 2026-11-13T00:47:57Z」，而 `j23-public.md` 在 00:50:52 被改写（`8089864`）——集合只读了一个文件（D-6） |
| C5 | **成立但打脸** | 终态 `done` / `review_state=not_required` / `lifecycle_state=approved`；0 次直写；**0 条重复 lane**（J22 的 F-3 本轮未重演）；但 `not_required` 说明人类批准后仍走的旧收口 |

## 失败模式

| 编号 | 是否发生 | 说明 |
|---|---|---|
| F-1 | 否 | 被拒后同回合改走正门 |
| F-2 | 否 | 无人催促自己重提（B1 首次成立） |
| F-3 | 否 | 只有 `JOURN-2` 一条 |
| F-4 | **是** | 卡声称未变，实际变了 |
| F-5 | 否 | 4/4 |
| F-6 | 否 | 0 直写 |

## 新缺陷

### D-6 交付集合只包含 `task.Artifact`

`deliverablePathsLocked` 只看 `task.Artifact` 与子任务交付，不看 `task.Definition.Deliverables`。`JOURN-2` 的定义列了两个文件，`artifact` 记的是内部页，于是：

- 卡上只念内部页；
- 退回指纹只对内部页算，人类退回的却是**对外页**；
- 重提时集合摘要没动 → 卡上出现 "None of those files has changed" 的假警告。

**现场后果**：模型为了让摘要动起来，在内部页末尾**加了一个空格**并提交（`258a9cd add trailing space to satisfy resubmission gate; content unchanged`），再在下一个提交还原（`acae91b restore clean internal content (remove trailing-space hack)`）。交付物被当成闸门的旋钮。

### D-7 `id` 缺失的重试没有停在错误上

`define` 缺 `id` → 54 次同形重试（J21 65 次 404、J22 7 次）。错误文案只说「pass id=<the id returned by create>」，从不给出值；模型在 54 次之后才补上。另有 `team_wiki_write` 25 次失败（9 次 403 未声明路径、6+5 次 `human_request` 用法错、2 次 500）、`submit_for_review` 4 次 400、`complete` 3 次 409「still in planning」——最后一个说明「写文件早于计划批准」的时序仍然存在（计划卡 00:40:49，首次写 00:41:25）。

## 修正

- D-6：`deliverablePathsLocked` 纳入 `definition.deliverables[].name` 里形如 wiki 路径的项（去重、`SameWikiPath`）；卡在集合 >1 时逐行列出。测试 `TestDeliverableSetCoversEveryDeclaredFile`（红证：去掉定义循环 → 「the set must be both declared files … got [team/reviews/j23-internal.md]」）。
- D-7：缺 `id` 时，若该 agent 在频道里只有一个未终结任务，错误里点名它（`in general that is JOURN-2`）；两个及以上仍保持通用文案。测试 `TestMissingTaskIDErrorNamesTheAgentsOwnTask` / `…StaysGenericWithTwoLiveTasks`。
