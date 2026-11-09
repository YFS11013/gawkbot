# Journey 6 — 预登记（2026-11-09）

> 本轮回答 Journey 5 留下的唯一空白：**bot 会不会把 403 拒写文案读进去、据此改变做法**。
> J3-3 的修复只证到「提示到达并去重」；本轮换一个有完整事实的场景，让 lane 真的去写 wiki。

## 1. 被测提交

- `852dc3d` fix(team): a supersede close stops overwriting last_run
- `22f6bd5` chore(team): drop ScheduleRecheck — nothing ever called it
- 基线：`d52293e`（拒写文案 + `artifact_path_missing` 提示）、`61667e0`（`status_reason`）
- 全量：`go test ./internal/team/ -timeout 30m -count=1` → `ok 811.416s`（`852dc3d` 时点）；`22f6bd5` 后跑 `Scheduler|Lifecycle` 子集绿

## 2. 环境

| 项 | 值 |
|---|---|
| 隔离办公室 | `journey6`（broker :7910，web :7911） |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/crouton-alpha-free` |
| 蓝图 | `ai-revops`，agents `cos, revops, analyst, ae, reviewer` |
| 主办公室 | 7891/7892，**不得被本轮任何命令指向** |

## 3. 场景（关键：事实齐全，逼 lane 真的动手）

父 Issue，`channel=cos__human`、`owner=cos`、`created_by=human`：

```
title:   "Journey 6: Acme 续约报价与跟进"
details: "客户 Acme（40 席位，SaaS）。现行合同：年付 $18,000，2026-12-31 到期，付款 30 天。
          拟续约：年付 $21,600（+20%），条款不变，有效期 14 天。对方决策人 Dana（VP Ops）。
          两件交付物：① 一页报价说明（价格/范围/有效期/付款/决策点）② 48 小时内的跟进邮件草稿。
          只认以上数字，不要引入其它假设。"
```

预期：cos planning → 自动出批准卡 → 批准后拆 lane → **lane 没有 `definition`** → 它们必须写 wiki 交付，于是撞 403 → 拿到新文案与频道提示。

## 4. 验证项

| 编号 | 判定 | 通过判据 |
|---|---|---|
| V1 | lane 真的尝试写 wiki | 隔离办公室 `broker.log` 出现至少 1 条 `wiki write-task refused: task=<id> slug=<bot>`，且该 slug 属于本轮 lane 的 owner（不是操作员探针） |
| V2 | 文案可执行 | 该 403 的 error 文本含 `@<creator>`、`definition.artifact_path`、`cannot succeed` 三个要素 |
| V3 | 每个任务只喊一次 | 每个被拒任务在 `messages` 里最多 1 条 `kind=artifact_path_missing`；同一任务重复被拒不新增 |
| V4 | bot 会改做法 | 首次 403 之后 ≤15 分钟，同一 lane 至少出现下列之一：①在频道里明确说要把产物交给 human request / 请 creator 钉路径；②以 `artifact_path`（或 `verification=not_required` + 理由）成功 `submit_for_review`/`complete`；③不再重复同一路径的写入 |
| V5 | 不回归 | 计划卡自动出现且只有一张；`kind=watchdog_alert` 计数 0；本轮新关的 task job 都带 `status_reason` |
| V6 | 不刷屏 | 同一任务被拒次数 ≤3（含 1 次是允许的重试） |

## 5. 硬失败

- **F-A**：同一任务出现 ≥2 条 `artifact_path_missing`（去重失效）。
- **F-B**：同一 lane 对同一路径连续被拒 >3 次且无任何做法变化（文案没起作用，只会重试）。
- **F-C**：计划卡缺席（J3-1 回归）。
- **F-D**：lane 因 403 直接放弃、无任何交付或上报（比 J3/J4 更差）。

## 6. 人工干预

**只允许 1 次**：点批准卡。其余人工步骤必须逐条记进 observations。

## 7. 停表

连续 15 分钟无新消息且无 active 回合，或 45 分钟上限。停表后**先复制 wiki 产物与 `broker.log` 相关行进 `artifacts/`，再 `workspace shred`**。快照每 5 分钟一次到 `snapshots/HHMM/`。

## 8. 判分

独立 reviewer 子代理，按 §4/§5 逐条判并写 `scorecard.md`。主代理不自评。

## 9. 预期风险

- 模型可能选择先把正文贴在聊天里、完全不调用写工具。若 45 分钟内没有一次写尝试，V1–V4 记「未观察」，本轮结论只能是「该场景不触发写路径」，不能当成修复有效或无效。
- `artifact_path_missing` 的去重按任务；同一任务换路径重试仍只有一条消息，这是设计。
