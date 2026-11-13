# Journey 24 scorecard — 卡念出整套，模型就不必去骗闸门

2026-11-13，源码 `dd6b14f`（D-6/D-7 第一版），工作区 `journey24`（:7910/:7911），模型 `opencode-go/beam-free`。
操作员动作 **4/4**：H1 下达、H2 批准计划（01:44）、H3 退回（01:51:07）、H4 批准。

## 断言

| 编号 | 判定 | 证据 |
|---|---|---|
| D1 卡列出两个路径 | **不成立** | `request-16`：「The deliverable is team/reviews/j24-public.md.」——单数。原因不是代码没生效，而是这一轮的 `define` **根本没有 `deliverables` 数组**，只有 `success_criteria` 散文 + `artifact_path`（见下 D-8） |
| D2 重提卡不带假警告 | **不成立** | `request-23`：「None of those files has changed since this work was sent back at 2026-11-13T01:51:07Z」——而内部页在 01:53 被改（`db0ffbf`） |
| D3 内部页无无意义提交 | **不成立** | `801fccf Journey 24: add trailing newline (no content change)` → `6700416 Journey 24: restore original content (remove trailing newline)`。**与 J23 同一手法，换了空白字符** |
| D4 漏 `id` ≤3 | **不成立** | **45 次**（J23 54、J22 7）。每次错误里都已点名「in cos__human that is JOURN-2」，模型仍然重发 |
| D5 ≤4 动作闭合、终态 done | **成立** | 4/4；`done` / `review_state=approved`；0 直写；D-5 守卫照常拦下一次 `complete`（01:46:16） |

## 失败模式

| 编号 | 是否发生 | 说明 |
|---|---|---|
| G-1 卡只念一个文件 | **是** | 但成因是定义没写 `deliverables`（D-8），不是卡片代码 |
| G-2 假警告 | **是** | 集合仍只看 `artifact`（对外页），返工落在内部页 |
| G-3 篡改交付物 | **是** | 尾随换行，随后自己还原 |
| G-4 漏 id 几十次 | **是** | 45 次 |
| G-5 第 5 个动作 | 否 | 4/4 |

## 两个新缺陷

### D-8 集合的来源错了：定义可以缺 `deliverables`，写入台账不会

J23 的定义里 `deliverables` 有两个路径，D-6 的修法因此生效；J24 的定义里**没有这个数组**，规划器把两个文件写进了 `success_criteria` 散文，于是修法读不到任何东西，卡又退回单数、指纹又只看一个文件。

定义是模型写的，形状不稳定；**它实际写了哪些路径**是事实。`/wiki/write-task` 每次成功都带着 `task_id` 和 `path`，这份台账不依赖定义怎么写。

修法：`teamTask.Artifacts []string` 记录本任务发布过的每个 wiki 路径（`/wiki/write-task` 成功后追加、`SameWikiPath` 去重、随状态持久化），`deliverablePathsLocked` 把它并入集合。测试 `TestDeliverableSetFollowsWhatTheTaskActuallyPublished`（红证：去掉台账循环 → 「both published pages must be in the set, got [team/reviews/j24-public.md]」）。
定义里声明的路径仍然保留在集合里——两个来源都对，取并集。

### D-9 错误文案说服不了一个正在重发的模型

J23 的修法是「错误里点名当前任务」，J24 在**每条**错误里都点了名（45 条），模型 45 次照旧不带 `id`。它不是没看到，是它在按同一个形状反复重发。

修法：当该 agent 在频道里**只有一个**未终结任务时，工具直接用它，并在结果末尾写明「No id was passed, so this acted on JOURN-2 — your only open task」。零个或两个以上仍然报错——在两个候选之间替模型挑一个，不是任何人的默认值。测试 `TestMissingTaskIDResolvesTheAgentsOwnTask`（断言 broker 请求体里带上了 `"id":"JOURN-2"`）、`TestMissingTaskIDErrorStaysGenericWithTwoLiveTasks`。

## 沿用 J23 的部分

- D-5 守卫第二次现场生效（`01:46:16 guard_refused | delivery close`）。
- `complete` 另被「declares the write scope …」拦了 2 次——写入范围与交付路径不一致，属旧守卫。
- 写第二个文件仍要走 `human_request`：4 次 403「artifact path not declared」、4 次「msg-1 is not a human-authored message」、2 次「does not explicitly ask for a direct wiki …」、2 次 500。窄缝没变。
