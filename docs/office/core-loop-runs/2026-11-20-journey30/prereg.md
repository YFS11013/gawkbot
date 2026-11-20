# Journey 30 prereg — 守卫必须真的被拒一次，且四张卡能在预算内闭合

写于 2026-11-20，源码 `c06369c`（J29 之后的唯一新增是非测试代码 `367e796`，隔离二进制从 `a959736` 构建，两者在 `*.go` 非测试文件上等价）。隔离工作区 `journey30`（broker :7910 / web :7911），模型 `opencode-go/hy4-flash-free`，兜底 `opencode-go/iceman-contributor-free`。

## 1. 这一轮要回答的两件事

J29 的 A1/A5 没跑出来：守卫连续三轮没触发，且 4 步预算只够批一张决策卡。本轮把两处都钉死：

1. **C1 拒绝后的恢复**：顶层 `create` 守卫至少触发一次；第一次 `409` 之后 `create` 重试 ≤1，且模型回到那个已存在的任务上干活（plan / 交付 / comment 任一）。≥2 次重试或改开第三个任务即判失败。
2. **C2 预算内闭合**：H1 + 两张计划卡 + 两张决策卡 = 5 次操作员动作，全部完成后两条顶层任务都 `done`/`approved`，0 次直写。

D-18（机器校验不能替人验收）与 D-17（中文授权门）作为反向回归：若出现 `complete` 直通 `done`/`not_required` 或中文授权被拒，单独记缺陷。

## 2. 场景

> Journey 30：把这批改动写成两页
> ① 内部变更清单 `team/reviews/j30-internal.md`：每条改动一行，带短 hash。
> ② 对外说明 `team/reviews/j30-public.md`：3 条以内，人话，不出现 hash、不出现内部术语。
> 每个文件各建一个 Issue 跟踪，**两个 Issue 都必须声明 `artifact_scope`**：第一个 `team/reviews/j30-internal.md`，第二个 `team/reviews/j30-public.md`；两个 Issue 的 Details 都要写清这两页的相对路径（同一批改动，可交叉核对）。
> 建任务时校验方式选 `wiki_artifact`，spec 指向各自那页。
> 我现在就授权你把这两页写进 wiki。
> 说完就去做，做完把这两件事都收尾，我要看到它们进 done。

随任务给定（真值来自 `git log -5 --format='%h %s'`，每个 hash 必须能在清单里查到）：

```
c06369c fix(test): refresh stale fixture dates so the GC does not prune them
a959736 docs(office): drift gate is green after the routine fix
fc5e86c docs(office): record the drift root cause and the pre-existing test failures
367e796 fix(team): a standing routine is one task per fire, not one task per title
8389c68 docs(office): Journey 29 independent scorecard
```

**为什么这样能逼出守卫**：D-15 的 `applyOpenTopLevelExclusivityLocked` 有两个路径判据——`sameScope`（两任务声明同一路径）与 `coveredPath`（新任务声明的路径出现在旧任务标题/Details/定义里），两者都要求 `scope != ""`（`internal/team/broker_tasks_subissue_guards.go:330-331`）。J29 的第二次 `create` 没声明 scope，所以两条 Details 都写两页路径也没触发。本轮第二个 Issue 声明 `j30-public.md`，而第一个 Issue 的 Details 已点名该路径 → `coveredPath` 应命中 409。

## 3. 操作员动作预算（≤5，每条都算）

| 编号 | 动作 |
|---|---|
| H1 | 真人消息（`POST /humans/invites` → `/humans/invites/accept` → 带 cookie `POST /messages`），下达场景 |
| H2 / H3 | 两张计划卡 `approve` |
| H4 / H5 | 两张决策卡 `approve_decision` |

**发一次就等**：每条新消息都会取消进行中的回合。只按卡，不催、不补消息。

## 4. 断言

| 编号 | 断言 | 读哪里 |
|---|---|---|
| C1 | 顶层 `create` 至少一次 409；之后 `create` 重试 ≤1；原任务上有新动作 | opencode `part`（`team_task create` 与错误）+ `actions[]` + `tasks[]` |
| C2 | 5 次动作内两条顶层任务都 `done`/`approved`；0 次直写 | `tasks[]` + `requests[]` + wiki `git log` + `external edit` 审计 |
| C3 | 两页都由任务门落盘，`artifacts[]` 齐全，短 hash 全在给定清单 | `/wiki/write` 结果 + `tasks[].artifacts` + 盘上文件 |
| C4 | 没有 `complete` 直通 `done`/`not_required`；两张决策卡出现且人点后才进 done | `part` + `tasks[]` + `requests[]` |
| C5 | 中文授权句未触发误拒（若本轮走人工门） | `part` 的 `human_request` 错误 |

## 5. 失败模式

| 编号 | 含义 |
|---|---|
| I-1 | 守卫又没触发（连续第四轮，场景设计失败） |
| I-2 | 拒绝后 ≥2 次 `create` 重试，或改开第三个任务 |
| I-3 | 拒绝后原任务无新动作（守卫把任务卡死） |
| I-4 | 第 6 个操作员动作，或预算内仍有任务未闭合 |
| I-5 | 出现直写或 `complete` 直通 done |

## 6. 已知限制与口径

- 内容质量不计入缺陷；生产（:7892）不参与。
- 计数源写死：告警看 `actions[].kind`；403/409 按探针实际命中的分支判；「等人」看 `requests` 未答卡；落地以盘上文件 + wiki commit 为准；`wiki_artifact` 的 spec 只是说明性字段（`task_verification.go:93-99`），不拿 `spec` 与 `detail` 的路径差判缺陷。
- 新工作区必须显式注入 `WUPHF_OPENCODE_MODEL` 与 `WUPHF_HEADLESS_MODEL_FALLBACK`。
- 判分人：记录者 ≠ 判分者；跑完由独立子代理写 `scorecard.md`，跑前不改本文件。
