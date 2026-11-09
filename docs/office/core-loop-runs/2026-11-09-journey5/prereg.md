# Journey 5 — 预登记（2026-11-09）

> 本轮只验 J3-3 修复与两处不回归。九步分不在范围内。
> **先 commit 本文件，再起隔离办公室**（README §流程第 2 条）。

## 1. 被测提交

- `d52293e` fix(team): a refused artifact write now reaches the person who can fix it
- `61667e0` fix(team): record why a scheduler job closed
- 上一步的基线：`3156185`（docs）／`f71413b`（Journey 4 记录）
- 全量 `go test ./internal/team/ -timeout 30m -count=1` → `ok 765.686s`；一轮里 `TestObsidianWatcher_StopDuringPendingDebounce` 因 Windows 临时目录句柄在并行下 cleanup 失败，单独跑通过（与本轮改动无关）

## 2. 环境

| 项 | 值 |
|---|---|
| 隔离办公室 | `journey5`（`workspace create --from-scratch journey5`，broker :7910，web :7911） |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/crouton-alpha-free`（创建命令与重启都必须显式带） |
| 蓝图 | `ai-revops`，agents `cos, revops, analyst, ae, reviewer` |
| 记忆后端 | markdown + wiki（默认） |
| onboarding | `POST /onboarding/complete`，`skip_task=true`；随后取消 4 条 starter lane |
| 主办公室 | **不得被本轮任何命令指向**（pid 常驻 7891/7892） |

## 3. 场景

父 Issue，一句话 + 两个明确交付物，故意**不给产物路径**（这正是 J3-3 的触发条件）：

```
channel:  cos__human
title:    "Journey 5: 报价单与跟进邮件"
details:  "给一家 40 人的 SaaS 公司做续约报价：一页报价说明 + 一封 48 小时内的跟进邮件。"
owner:    cos
created_by: human
task_type: issue
```

预期：cos 进入 planning 姿态 → 计划帖 → 自动出批准卡；批准后拆两条 lane（分析 + 文案），两条都**没有** `definition.artifact_path`。

## 4. 验证项

| 编号 | 判定 | 通过判据 |
|---|---|---|
| V1 | 计划卡自动出现 | 计划帖之后 ≤10 分钟、无人工动作，`requests` 里出现 `dedupe_key=plan-approval:<issue>` 且 `status=pending` |
| V2 | 拒写文案点名 creator | 任一 lane 的 `/wiki/write-task` 403 文本包含 `@<creator>`、`definition.artifact_path`、`cannot succeed` |
| V3 | 提示只喊一次 | 每个任务在任务频道里最多 1 条 `kind=artifact_path_missing` 消息；同一任务第 2、3 次被拒不再新增 |
| V4 | 提示落在对的地方 | 该消息出现在**任务自己的频道**（如 `analyst__cos` / `ae__cos`），且同时 @creator 与 @owner |
| V5 | 关掉的 job 有理由 | `/scheduler` 里本轮新关的 task job，`status=done` 且 `status_reason` 非空 |
| V6 | 不回归 | 全轮 `kind=watchdog_alert` 计数 0；`watchdogs[]` 为空 |

## 5. 硬失败

- **F-A**：同一任务出现 ≥2 条 `artifact_path_missing`（去重失效 → 刷屏）。
- **F-B**：任一 403 文案不含 creator 名（修复未生效）。
- **F-C**：计划卡没出现（J3-1 回归）。
- **F-D**：批准后 5 分钟内没有拆出任何 lane（J3-3 修法把创建路径弄坏了）。

## 6. 人工干预

**只允许 1 次**：点批准卡（`POST /interview/answer`）。其余任何人工步骤都要在 observations 里记一笔。

## 7. 停表

1. 连续 15 分钟无新消息且无 active 回合；或
2. 达到 45 分钟上限。

停表后：**先把 wiki 产物 `cp` 进 `docs/office/core-loop-runs/2026-11-09-journey5/artifacts/`，再 `workspace shred`**（Journey 4 的教训：shred 之后正文不可复核）。快照每 5 分钟一次，存 `snapshots/HHMM/`。

## 8. 判分

由独立 reviewer 子代理按 §4/§5 逐条判，写 `scorecard.md`。主代理不自评。

## 9. 预期风险

- 若两个 bot 在拿到 403 后**仍然**退到 notebook（不读提示），本轮只能证明「提示到达」，不能证明「bot 据此修正」。这一点必须写进结论，不许含糊。
- 提示是按任务去重的；若同一任务换了路径重试，仍只有一条——这是设计，不是漏。
