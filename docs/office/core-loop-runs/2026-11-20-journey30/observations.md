# Journey 30 observations — 守卫必须真的被拒一次，且四张卡能在预算内闭合

工作区 `~/.wuphf-spaces/journey30`，broker `:7910` / web `:7911`。源码 `c06369c`（隔离二进制从 `a959736` 构建，两者在非测试 Go 文件上等价），模型 `opencode-go/hy4-flash-free`，兜底 `opencode-go/iceman-contributor-free`（`/health` 回显一致，`members=1`）。时刻 UTC（本地 = UTC+8）。

## 时间线

| 时刻 UTC | 事件 |
|---|---|
| 12:15:07 | 真人会话建立（invites → accept，`human_slug=ireke`）；`msg-2` 下达场景一次；cos `stage=started` |
| 12:16:02 | `team_task create` → `JOURN-3`（internal），`artifact_scope=team/reviews/j30-internal.md`，completed |
| 12:16:12 | `team_task create` → public 页，声明 `artifact_scope=team/reviews/j30-public.md` → **409**：`open task JOURN-3 (…) already holds this work in #cos__human — one request is one acceptance…` |
| 12:16:28 | 同形状 `create` 第 1 次重试，`allow_duplicate=true` → completed，`JOURN-6` 建立 |
| 12:16:47 / 12:17:42 | `request-11`（plan:JOURN-3）/ `request-17`（plan:JOURN-6）出现 |
| 12:17:11 / 12:17:29 | 两页首次落盘：`write-task task_id=JOURN-3` 434 B commit `d26979b`；`write-task task_id=JOURN-6` 417 B commit `cd01ddc` |
| 12:17:41 / 12:18:42 | 两次 `complete` 被 409 `still in planning`；12:17:55 `approve` 被 403 `parked` |
| 12:18:17 / 12:18:30 | 两次 `resume` 成功 |
| 12:20:07 | 操作员批准 `request-11`、`request-17`（H2/H3，同一秒两次点击） |
| 12:20:46 / 12:20:58 | 模型回读发现内部清单只有 4 行、漏 `8389c68` 且第 4 行截断；重写两页，509 B commit `c35fc15`、412 B commit `d130d6b` |
| 12:21:09–12:21:33 | 5 次 `complete` 被 409 `a bot may not close it from here — that would skip the human`；`guard_refused` 4 条（JOURN-3 ×2、JOURN-6 ×2） |
| 12:21:19–12:21:44 | 4 次 `submit_for_review` 成功；`request-36`（decision:JOURN-3）/ `request-39`（decision:JOURN-6）出现 |
| 12:22:20 | 操作员批准 `request-36`、`request-39`（H4/H5，同一秒两次点击） |
| 12:22:51 | 两条 `task_updated … [done]`；两任务终态 `done`/`approved`，`verification_result.pass=true`；`/requests` 归零 |

## 工具面原始记录（opencode.db `part`）

- `wuphf-office_team_task create` ×3：
  - 12:16:02 `artifact_scope=j30-internal.md` → completed
  - 12:16:12 `artifact_scope=j30-public.md` → **error 409**（守卫命中）
  - 12:16:28 `artifact_scope=j30-public.md`、`allow_duplicate=true` → completed（`JOURN-6`）
- `wuphf-office_team_wiki_write` ×4，全部 completed：`d26979b`(434 B)、`cd01ddc`(417 B)、`c35fc15`(509 B)、`d130d6b`(412 B)；前两次带 `task_id` 与 `human_request=msg-2`，后两次只带 `task_id`。无直写、无 403/409 写入失败。
- `complete`/`approve` 拒绝：409 `still in planning` ×2、403 `parked` ×1、409 `a bot may not close it from here` ×5；`resume` ×2、`submit_for_review` ×4 均成功。
- 12:20:46 模型自述回读发现问题：「内部清单只有 4 行，漏了第 5 条 `8389c68`，且第 4 行被截断」，并在下一回合重写两页。

## 落盘产物

- wiki `git log --oneline`：`d26979b` / `cd01ddc` 首版，`c35fc15` / `d130d6b` 修正版，随后 `dbb3b5e` / `737fbd8` 决策提升与 `7a3c0c8` / `f1c2744` learning。
- `team/reviews/j30-internal.md`（509 B）：5 行，短 hash `c06369c / a959736 / fc5e86c / 367e796 / 8389c68` 全在给定清单内；交叉引用 public 页。
- `team/reviews/j30-public.md`（412 B）：3 条，无短 hash、无模型代号 / Journey / bot / wiki 术语。

## 终态（12:23 读取）

- `JOURN-3` / `JOURN-6` 均 `status=done`、`lifecycle_state=approved`，`artifact` 与 `artifacts[]` 各为对应页，`verification_result.pass=true`。
- `/requests?scope=all` 为 0；`actions` 中 4 条 `guard_refused` 均为 `delivery close`（非 create 守卫；create 拒绝落在 opencode part 的错误里）。

## 操作员动作

H1 12:15:07Z `msg-2`（下达 + 授权）；H2/H3 12:20:07Z 两张计划卡；H4/H5 12:22:20Z 两张决策卡。共 5 次，未超 prereg 预算。

## 关键观察

1. **守卫真的触发了**：第二次 `create` 因 `coveredPath`（声明 public 路径，而首条任务的 Details 已点名它）被 409 拒绝——J27/J28/J29 连续三轮未测到的路径终于被锻炼。
2. **拒绝后的动作形状**：唯一一次重试带 `allow_duplicate=true`，建出 `JOURN-6`（12:16:28Z）；同一时间窗口里 `JOURN-3` 在 409 后 26 秒执行 `define`，随后继续 `wiki_write` / `comment` / `submit_for_review`。第二页的 `artifact_scope` 与第一页正交，人在 H1 里明确要求「每个文件各建一个 Issue」。独立评分据此判 `allow_duplicate` 为合规豁免、C1 成立。
3. **预算 5 次正好闭合**：2 计划卡 + 2 决策卡 + H1，两条任务都进 `done`/`approved`，`/requests` 归零。
4. **D-18 回归**：bot 的 `complete` 被拦 5 次，改走 `submit_for_review` → 决策卡 → 人点后 done；没有直通 `done`/`not_required`。
5. **模型自纠正内容**：首版内部清单漏第 5 条且第 4 行截断，模型回读发现后重写两页——这是本轮唯一一次内容层面的自我修复。

## 未做 / 限制

- 工作区保留供评分复核；按 README 约定不提交运行产物。生产（:7892）不参与。
- 内容质量只在「hash 是否在清单内、public ≤3 条且无内部术语」上机检，不做主观文风判定。
