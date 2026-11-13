# Journey 27 scorecard — 拒绝之后，模型知不知道该收尾

2026-11-13，源码 `73c3d4b`（D-14 在包里），工作区 `journey27`（:7910/:7911），模型 `opencode-go/beam-free`。
操作员动作 **4/4**（H1 下达、H2 批 JOURN-2 计划、H3 批 JOURN-2 交付、H4 批 JOURN-7 计划）。第 5 个动作（批 JOURN-7 交付）预算不允许，故此轮**没有闭合**——这本身就是证据。

## 断言

| 编号 | 判定 | 证据 |
|---|---|---|
| F1 顶层任务只有一个 | **不成立** | `JOURN-2`「两页变更清单（内部 + 对外）」+ `JOURN-7`「对外说明页」 |
| F2 被拒的 create ≤1 次 | **不成立（0 次拒绝）** | 两个 create 都 `status=completed`——守卫根本没触发 |
| F3 被拒后回到原任务 | 不适用 | 没有发生拒绝 |
| F4 两页都进台账、卡念两个路径 | **不成立** | `JOURN-2.artifacts=[j27-internal.md]`，卡 `request-19`：「The deliverable is team/reviews/j27-internal.md」；`JOURN-7.artifacts=[j27-public.md]`，卡 `request-38` 同样只念一页 |
| F5 `define` 漏 `id` 报错为 0 | **成立** | 2 次 `define` 调用，1 次 400「definition goal is required」，0 次漏 id |
| F6 ≤4 动作闭合 | **不成立** | 4 个动作用尽时 `JOURN-7` 仍 `in_progress`，交付卡 `request-38` 无人应答 |

## 失败模式

| 编号 | 是否发生 | 说明 |
|---|---|---|
| I-1 换标题绕过守卫 | **是** | 第二个任务标题「对外说明页」，与旧标题只共享一个 bigram（对外） |
| I-2 原地重试 create | 否 | 模型没重试，直接就把任务建成了 |
| I-3 被拒后躺平 | 不适用 | 没有拒绝 |
| I-4 台账/卡只念一页 | **是** | 两个任务各持一页，两张卡各念一页 |
| I-5 第 5 个动作 | **即将发生** | 要闭合需要第 5 个动作，预算不允许 |

## 新缺陷

### D-15 守卫的判据漏了「路径」这一层（已修）

D-14 的判断依赖两条语言信号：声明同一 `artifact_path`，或标题复述。J27 两条都没命中：

- `JOURN-2` 在 create 时**没有**声明 `artifact_scope`（它的路径是后来在计划/define 里定的），所以「同范围」无从比较；
- 两个标题的 bigram 交集只有 `对外` 一个词，containment 0.25，低于 0.35 门槛。

而真正铁一样的证据就在手边：`JOURN-7` 声明 `team/reviews/j27-public.md`，**这个路径已经写在 `JOURN-2` 的 details 里**（人类那一句话同时点了两个文件）。路径是交付物的身份，不随措辞漂移。

修法：新增 `openTaskNamesPath` —— 新任务声明的写入路径若已经出现在某个未终结顶层任务的 title / details / goal / deliverables / artifact_path 里，就拒绝。三条信号（同范围 → 路径被点名 → 标题复述）并联。

### D-16 请求形状错误被当成服务器故障（已修）

`mode=create` 写一个已存在的页面，返回 **500 Internal Server Error**，正文却是「use replace or append_section」——文案在教模型怎么改，状态码在告诉模型「服务端坏了」。模型于是重试了一次。

修法：`ErrWikiArticleExists` / `ErrWikiInvalidRequest` 两个哨兵错误，`wikiWriteRequestStatus` 映射到 409 / 400，两扇写入门都走它。

### 观察（不修）

自动化消息（`POST /notifications/automation`）的发送者恒为 `nex`，因此 `human_request` 校验永远判它「不是人写的」——本轮 2 次。这是刻意的信任边界（automation 不等于人），但也意味着**人工授权门 D-12 至今没有在隔离工作区里被现场走过一次**：harness 发不出真正的人消息。五轮以来都是这样，记录在案。
