# Journey 4 预注册 — 验证 J3-1 / J3-2 两条修复（2026-11-09）

> 跑之前写死。`prereg` 先 commit，之后只补 observations / scorecard。
> 目的不是再打一次分，而是把 Journey 3 停表后修的两条**端到端**验一遍——它们当时只有单测（含 failing-before）。

## 1. 要验证的两条修复（代码与单测已在 `0a91081` / `d01b103`）

| 编号 | 修复 | 单测 | 本轮要看的人可见证据 |
|---|---|---|---|
| J3-1 | opencode 回合在 planning 姿态下出 plan 卡并 raise 审批 | `TestOpencodePlanningTurnRaisesPlanApproval`、`TestOpencodeRunningTurnRaisesNoPlanApproval` | 计划帖出现后，`/requests` **自己**出现一张 plan-approval 卡（不看代码、不听 bot 自述），且**全办公室只此一张** |
| J3-2 | watchdog 不再对 `review`/`decision`/`changes_requested`/`rejected` 的任务发「No turn could be dispatched」 | `TestSchedulerQuietForTasksAwaitingReview`（四态）、`TestSchedulerStillAlarmsParkedTask`、`TestSchedulerStillWatchesPlanningTasks` | 子任务进 `decision` 后，其频道与 `messages.json` **不再新增** `watchdog_alert`；同时 parked/planning 仍会报警（对照） |

## 2. 环境（钉死）

| 项 | 值 |
|---|---|
| 二进制 / commit | `gawkbot-fix.exe`（2026-11-09 10:58 构建，HEAD `364511c`，含 `0a91081`+`d01b103`） |
| 隔离工作区 | `journey4`，broker `127.0.0.1:7910`、web `7911` |
| runtime home | `~/.wuphf-spaces/journey4` |
| provider / model | `opencode` / `opencode-go/crouton-alpha-free`（`WUPHF_OPENCODE_MODEL` 从 CLI 进程继承） |
| memory backend | `markdown`（本机无 gbrain；K 的 wiki/gbrain 解耦不在本轮范围） |
| 蓝图 / 成员 | `ai-revops`：cos / revops / analyst / ae / reviewer |
| 公司名 | `Nex` |
| 主办公室 | pid 3044（7891/7892）**不动**；本轮任何命令不得指向 7891/7892 |

## 3. 场景（人给的任务，逐字提交）

创建方式与 J1–J3 相同：`POST /tasks`，`action=create`、`channel=cos__human`、`owner=cos`、`created_by=human`、`task_type=issue`。

标题：`ICP 假设与首触点邮件：一页式验证包`

details（逐字）：

> Goal: 在下周一之前，给出一页「我们该把 SMB 续费触达讲给谁听」的可验证假设，外加一封可直接审稿的首触点邮件样例；不需要发信、不需要接触真实客户。
>
> Deliverables:
> 1. `team/research/icp-hypothesis-2026q4.md`：3 条 ICP 假设，每条写清「凭什么这么假设」「怎么在两周内证伪」，允许标 unknown，但必须写清缺什么。
> 2. `team/emails/first-touch-sample.md`：一封 ≤180 词的首触点邮件样例，收件人是 20 席 SaaS 公司的续费决策人；不得编造客户事实，缺字段用占位符。
>
> Success criteria:
> - 两个文件都在给定路径、wiki 可读；
> - 每个数字有来源或明确 unknown，没有编造；
> - 邮件与假设口径一致（假设里说「价格敏感存疑」，邮件里就不能拿折扣当钩子）；
> - 完成后在对话里给两个链接。
>
> Constraints: 不要发任何邮件；不要建客户表；不要申请外部账号。证据不足时先写假设+验证动作，不要停下来问人。

**为什么选这个形状**：J2/J3 的经验是「Issue + 多交付物 + Goal/Deliverables/Success criteria」会触发 plan-first；本轮只需要它触发一次 planning，两个 lane 足够把 `decision` 态跑出来（J3-2 的验证点）。

## 4. 预测（跑之前钉死）

| 编号 | 预测 | 判据（可机器验） |
|---|---|---|
| V1 | 计划帖出现后 **10 分钟内** `/requests` 出现 plan-approval 卡 | `GET /requests`（`scope=office` 与该 bot 的 DM 都要查）里 `requestIsPlanApproval` 且 `issue_id=JOURN-*` 的条目 ≥1 |
| V2 | 全办公室**只有一张** plan-approval 卡（幂等） | 同上过滤后 `len==1`；重复 planning 回合不加卡 |
| V3 | 人可以**只点这张卡**（不碰 `POST /tasks approve`）就把任务推进到 `running` 并拆活 | 答卡后 `GET /tasks/{id}` → `lifecycle_state=running`，且 2 分钟内出现子 Issue |
| V4 | 子任务进 `decision` 后 **不再**出现 `watchdog_alert` | 该频道 `messages.json` 中 `kind=watchdog_alert` 且 `content` 含 `No turn could be dispatched` 的条目在该任务进 `decision` 之后 **增量 0** |
| V5 | 对照组：若任务停在 `planning` 且没人答卡，watchdog 仍会报警（J3-2 没有把所有报警关掉） | 观测窗口内出现指向 planning 任务的 `watchdog_alert` **或** 明确记录「窗口内未到期，未观察」——不许默认通过 |

## 5. 硬失败

| 编号 | 定义 |
|---|---|
| F-A | 计划帖出现后 10 分钟仍无任何 plan-approval 卡（J3-1 没生效） |
| F-B | 同一 planning 任务出现 ≥2 张 plan-approval 卡（幂等破） |
| F-C | 任务停在 `decision` 之后仍出现「No turn could be dispatched to wake …」告警（J3-2 没生效） |
| F-D | 答卡后任务没有进 `running` / 没有拆活（卡片路径本身坏了） |

## 6. 人工干预预算

- **1 次**：答 plan-approval 卡（这就是 V3 的被测动作，必须在**卡片**上点，不许走 `POST /tasks approve`）。
- 安装期：starter lane 清理的 `cancel` 不计入。
- 其它一律 0。若出现 F-A，**不许**用 `POST /tasks approve` 代替卡片来「救场」——那就是本轮结论。

## 7. 停表

满足任一即停并快照：
1. 父 Issue 终态（done/approved）；或
2. 连续 15 分钟无新消息、无 active 回合；或
3. 从提交任务起 60 分钟。

## 8. 记录

- `observations.md`：时间戳事实（谁在何时说了什么、`/requests`/`/scheduler`/`/tasks` 的变化），机制侧（代码/日志行）单列。
- `scorecard.md`：本轮不打九步，只做 **V1–V5 + F-A..F-D 的判定表**，每条给 A/B/C 证据级。
- 快照：`snapshots/<HHMM>/`，含 health / tasks / 每任务 JSON / requests / messages / scheduler / broker-state / 日志尾部。
- 跑完 `workspace shred --yes journey4`，释放 7910/7911。
