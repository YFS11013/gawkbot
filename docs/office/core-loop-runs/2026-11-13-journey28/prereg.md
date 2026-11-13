# Journey 28 prereg — 拒绝之后必须有人接住，以及真人身份的第一扇门

写于 2026-11-13，源码 `03e0f54`（D-15 路径判据 + D-16 4xx 已上线），隔离工作区 `journey28`（:7910/:7911），免费模型 `beam-free`。

## 1. 这一轮要回答的问题

三件事，缺一件都算没跑：

1. **拒绝之后的恢复**（J27 没测到：守卫根本没触发）。第一次 `409` 之后 ≤1 次重试，且模型回到那个已存在的任务上干活；**≥2 次重试就说明文案不够可行动**——J24 的 45 次重发就是这么来的。
2. **D-12 的第一次现场**：harness 一直用 `/notifications/automation`（发送者恒为 `nex`），`human_request` 必判非人，人工授权门五轮没有一次真正跑通过。这轮改走真人路径：邀请 → 接受 → 会话 cookie → `POST /messages`，`From=human:<slug>`。
3. **一张卡念两页**：守卫现在会把一个请求压回一个任务，而 `artifact_scope` 只装得下一条路径——第二页必然要走人工授权门。也就是说：**修复 D-14/D-15 之后，D-12 从「可选路径」变成「结构性必需」**，这一轮必须看到。

## 2. 场景（与 J27 同形，外加授权句）

> Journey 28：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j28-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j28-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 我只能写下面这 5 条提交里的东西。
> 我现在就授权你把这两页写进 wiki：`team/reviews/j28-internal.md`、`team/reviews/j28-public.md`。哪一页不在你的任务范围里、需要 `human_request`，就用我这条消息。
> 说完就去做，做完把这件事收尾，我要看到它进 done。

随任务给定：

```
03e0f54 docs(office): Journey 27 — make refusal recovery the next round's hard assertion
5d7a60d docs(office): Journey 27 — the guard never fired, and the split cost an extra card
c681b5a fix(office): a request-shaped wiki failure is a 4xx, not a 500
39a3bbb fix(office): the duplicate signal is the path, not the wording
73c3d4b docs(office): Journey 26 — the card told the truth, and the model stopped cheating
```

## 3. 操作员动作预算（≤4，每一条都算）

| 编号 | 动作 |
|---|---|
| H1 | 真人消息（`POST /messages`，带会话 cookie）——下达场景，同时是那条 wiki 授权 |
| H2 | 计划卡批准 |
| H3 | 第一次交付后退回一页（真实内容要求） |
| H4 | 第二次交付后批准 |

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| F1 | **守卫至少触发一次**，且每次拒绝后 create 重试 ≤1 | opencode `part` 的 `task create` + 错误码 |
| F2 | 最终顶层任务 = 1 | `tasks[]` |
| F3 | 拒绝之后原任务上有新动作（plan / 交付 / comment 任一） | 任务时间线 / `actions[]` |
| F4 | 第二页走人工门写成：`human_request=<H1 的消息 id>` 且 200；`artifacts` 齐两页；卡念两页 | `/wiki/write` 响应 × `tasks[].artifacts` × 卡正文 |
| F5 | `define` 漏 `id` 报错为 0 | opencode `part` |
| F6 | ≤4 动作闭合，终态 `done`、`review_state=approved` | `tasks[]` |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| I-1 | 守卫又没触发（这轮就是为它设计的，空转 = 断言作废） |
| I-2 | 拒绝后 ≥2 次重试，或换个说法再开第二个任务 |
| I-3 | 拒绝后模型不干活（守卫把任务卡死） |
| I-4 | 台账只有一页 / 卡只念一页 / 人工门仍报「not a human-authored message」 |
| I-5 | 第 5 个操作员动作 |

## 6. 已知限制

- 内容质量不计入缺陷；生产（:7892）不参与。
- `TestEnsureHiddenConsoleHidesTheWindow` 在本机挂死（J27 已在 stash 基线上复现，与本轮无关），套件继续 `-skip` 它跑。
- 邀请产生的会话是 harness 自己的成员身份，不是 owner；它发的消息 `From=human:<slug>`。
