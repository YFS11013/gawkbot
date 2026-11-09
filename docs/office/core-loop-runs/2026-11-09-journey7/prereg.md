# Journey 7 — 预登记（2026-11-09）

> 本轮仍然是回答 **J3-3 的端到端问题**：bot 会不会把 403 拒写文案读进去、据此改变做法。
> Journey 6 给了完整事实，却因为（a）onboarding 遗留任务与前一轮的采访卡把 cos 拽走、（b）批准后没有分解检查点，45 分钟内一次写尝试都没有，V1–V4 全是「未观察」。
> 本轮把这两个已知干扰项都处理掉，再问同一个问题。

## 1. 被测提交

- `336bbbc` fix(team): an approved plan gets a decomposition check, a question dies with its task
  - 批准后 5 分钟装一次性 `plan_decomposition_check`：无子任务则 `task_not_decomposed` 告警 + 唤醒 owner 一次
  - 任务进终态时取消它名下所有 active request（采访卡不再压住 bot 的唤醒）
  - `request_follow_up` 对非阻塞采访也发可见提醒
- 基线：`d52293e`（拒写文案 + `artifact_path_missing` 提示）、`61667e0`（`status_reason`）、`852dc3d`（`last_run` 语义）
- 全量测试：`go test ./internal/team/ -timeout 30m -count=1`，结果写在 observations §1（本轮起点）。

## 2. 环境

| 项 | 值 |
|---|---|
| 隔离办公室 | `journey7`（broker :7910，web :7911） |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/crouton-alpha-free` |
| 蓝图 | `ai-revops`，agents `cos, revops, analyst, ae, reviewer` |
| 主办公室 | 7891/7892，**不得被本轮任何命令指向** |

### 2.1 与 Journey 6 的两处隔离改动（写清楚，判分时按此核对）

1. **场景频道换成专用频道 `j7`**（不是 `cos__human`）。Journey 6 的污染源是：onboarding 的计划 lane 出错后办公室自动建了一条 `JOURN-3` 落在 `cos__human`，与场景共享同一会话上下文。本轮把场景放在 `j7`，并在 onboarding 之后、提交场景之前把 `ai-revops-1..4` 全部取消。
2. **onboarding 遗留物一律清理**：除取消 4 条 starter lane 外，提交场景后立即检查是否出现「不属于本场景」的任务/采访卡；若出现，逐条取消并记进 observations（判分时单独计数，不算场景干预）。

## 3. 场景

父 Issue，`channel=j7`、`owner=cos`、`created_by=human`、`task_type=issue`，**不带 `definition`**（这正是要触发 403 的条件：子任务创建时没人钉 `artifact_path`）：

```
title:   "Journey 7: Acme 续约报价与跟进"
details: "客户 Acme（40 席位，SaaS）。现行合同：年付 $18,000，2026-12-31 到期，付款 30 天。
          拟续约：年付 $21,600（+20%），条款不变，有效期 14 天。对方决策人 Dana（VP Ops）。
          两件交付物：① 一页报价说明（价格/范围/有效期/付款/决策点）② 48 小时内的跟进邮件草稿。
          只认以上数字，不要引入其它假设。"
```

预期路径：cos planning → 自动出批准卡（`dedupe_key=plan-approval:<父ID>`）→ 批准 → 拆 lane → lane 无 `definition` → lane 尝试 `POST /wiki/write-task` 交付 → 403 → 拿到点名 creator 的文案 + 频道里一条 `artifact_path_missing` → 改变做法。

## 4. 验证项

| 编号 | 判定 | 通过判据 |
|---|---|---|
| V1 | lane 真的尝试写 wiki | 隔离办公室 `broker.log` 出现至少 1 条 `wiki write-task refused: task=<id> slug=<bot>`，且 slug 属于本轮 lane 的 owner（不是操作员探针） |
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
- **F-E**（本轮新增，针对 `336bbbc`）：批准后 10 分钟内既无子任务、也无 `task_not_decomposed` 告警（分解检查判失效）；或检查发出告警后 owner 仍未在 15 分钟内产出任何子任务/交付。

## 6. 人工干预

**只允许 1 次**：点批准卡。其余人工步骤（取消 onboarding 遗留、清理非场景任务）必须逐条记进 observations，并在判分时单列，不算场景干预。

## 7. 停表

连续 15 分钟无新消息且无 active 回合，或 60 分钟上限（Journey 6 的 45 分钟没跑到最后一次安排唤醒，本轮放宽）。停表后**先复制 `broker.log` 相关行、lane notebook、wiki 提交列表进 `artifacts/`，再 `workspace shred`**。快照每 5 分钟一次到 `snapshots/HHMM/`。
