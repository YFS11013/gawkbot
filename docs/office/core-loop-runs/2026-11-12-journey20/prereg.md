# Journey 20 prereg — 最小完整环路：这张 Issue 能不能真的进 done

写于 2026-11-12，源码 `b315a39`（O1–O4 全部修复后），二进制 = 仓库根 `gawkbot-fix.exe`（部署于 `a841336`，此后无生产代码改动）。

## 1. 这一轮要回答的问题

J18 的判分卡记的是 V7 **未达成**：父 Issue 一直停在 `decision`，从头到尾没有任何一次「办公室把一件事做完」的观测。之后四轮把 J18 暴露的 O1–O4 全修了，但**修完之后环路能不能闭合，没人看过**。这一轮就只回答这一件事。

场景沿用 J18 的形状（真值来自 `git log`），但缩到最小：5 条提交、两页、一张容器卡。目标是**跑完**，不是跑大。

## 2. 场景

种子 Issue（`created_by=cos`，人类经 `nex` 自动化消息下达，频道 `cos__human`）：

> Journey 20：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j20-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j20-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 我只能写下面这 5 条提交里的东西。
> 说完就去做，做完把这件事收尾，我要看到它进 done。

随任务给定（我生成，不经过模型）：`git log --oneline -5` 的输出。

## 3. 操作员动作预算（≤4，全部经 `POST /requests/answer`，token 身份）

| 编号 | 动作 |
|---|---|
| H1 | 下达场景（`POST /notifications/automation`），必要时补一句「按你自己的计划做」 |
| H2 | 第一次计划/采访卡出现时，用一句话回答受众与取舍 |
| H3 | 交付后**退回一次**，理由是具体的（例如「对外页第 2 条没有对应内部条目」） |
| H4 | 第二次交付后决定批准或退回；预算到 4 即停手，不再追加 |

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| V1 | 父 Issue 终态 `done`，`artifact=null`，无 `hollow_completion` | `tasks[]` |
| V2 | 恰好 1 张容器卡（`dedupe_key=decision:<父ID>`），问题里**列全两个路径**；lane 级决策卡 0 张 | `requests[]` |
| V3 | 全程 0 条要人类授权的消息（不含 `go-ahead` / `direct wiki` 措辞），0 次 `human_request` 授权路径 | `messages[]`、wiki git log |
| V4 | 两个文件都在 wiki，且各自提交信息带**自己 lane** 的 `(task ID)` | wiki `git log` |
| V5 | 内部页出现的短 hash 全部 ∈ 给定 5 条；对外页 0 个 hash | 脚本比对 |
| V6 | （O3 机会性观测）任何 `task_stalled` 告警的时间戳都不落在某个回合的执行窗口内 | `actions[]` × latency log |
| V7 | （O4/O2）H3 退回后：重提时要么文件字节与退回时刻不同，要么容器卡带「has changed since this work was sent back」警告；不得出现「一个字没改但被当成重做」 | 文件 sha256 + 卡正文 |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| F-a | 父 Issue 停在 `decision`/`review` 不动（V1 不成立） |
| F-b | 出现断言外的第二张容器卡，或 lane 自升卡 |
| F-c | 又要人类授权一句话（O1 修复在真模型下无效） |
| F-d | 退回后重提，字节没变且卡上无警告（O2/O4 无效） |
| F-e | 超过 4 次操作员动作仍不收敛（记为「环路不自主」，不算修 bug 目标） |

## 6. 已知限制

- 单测/探针已覆盖 O1 的 handler 与模型级行为；这一轮只关心**整条环路能否闭合**，不再重复验证单点。
- 免费模型（`beam-free`）质量有限，模型写不出内容属于场景问题，不计入缺陷；但「不会收尾」计入 F-a。
