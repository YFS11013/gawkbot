# Journey 5 — 观察记录（2026-11-09）

> 只记时间戳事实，不判分。判分由独立 reviewer 按 `prereg.md` §4/§5 做。
> 全部时间为 UTC。工作区 `journey5`（broker :7910，web :7911，`opencode-go/crouton-alpha-free`）。

## 1. 时间线

| 时间 | 事实 |
|---|---|
| 04:51:54 | `POST /onboarding/complete` 完成；4 条 starter lane（`ai-revops-1..4`）随后 `cancel` |
| 04:52:02 | 父 Issue `JOURN-6` 创建（`cos__human`，owner `cos`，`created_by=human`），`lifecycle_state=planning`、`pipeline_stage=plan`，无 `definition` |
| 04:52:23 | cos 计划帖：四段拆分（`JOURN-6.1` 数字核对 / `6.2` 一页报价 / `6.3` 48 小时跟进邮件 / `6.4` QA 核查），末尾「审批后创建，带 parent JOURN-6」 |
| 04:52:24 | `request-9` 自行出现：`kind=approval`、`status=pending`、`channel=cos__human`、`dedupe_key=plan-approval:JOURN-6`、`issue_id=JOURN-6`，`question` 里带完整计划正文（= **唯一一次人工干预点**） |
| 04:53:01 | `POST /interview/answer {"id":"request-9","choice_id":"approve"}` → `{"ok":true}` |
| 04:53:0x | `JOURN-6` → `lifecycle_state=running` |
| 04:53–04:54 | 拆出四条 lane：`JOURN-15`（analyst，数字核对）、`JOURN-17`（ae，跟进邮件）、`JOURN-19`（reviewer，QA）、`JOURN-21`（analyst，报价说明）；**四条都没有 `definition`** |
| 04:54:0x | `request-13` 出现：`kind=interview`、`pending`、`cos__human`，无 `dedupe_key`——内容是问人类要「公司名/决策人、现行合同、拟续约价与涨幅、付款方式、发件人身份」，或允许「用一套示例口径先草拟（标示例）」 |
| 04:57:39 | 三条 lane 都给出「等源数字」的说明：`JOURN-15` 保持 `in_progress`、`JOURN-17` 与 `JOURN-21` 各自 `submit_for_review` 进 `decision`、`JOURN-19` 挂起等上游 |
| 05:08:08 | 本轮最后一条正文（cos/reviewer 确认依赖顺序） |
| 05:2x | 停表（自 05:08:08 起 15 分钟无新消息；期间无 active 回合） |

## 2. 机制侧观察

1. **计划卡幂等**：`requests` 里只有 `request-9` 一张 `plan-approval:JOURN-6`，批准后 `status=answered` 且不再出现第二张。
2. **swarm 拆分**：批准后 1 分钟内四条 lane 全部落地，全部指向 `parent_issue_id=JOURN-6`，全部无 `definition`——J3-3 的触发条件原样重现。
3. **无人尝试写 wiki**：整轮 `broker.log` 里没有一条 `wiki write-task` 记录，因此 `artifact_path_missing` 提示在**循环内未被触发**（`artifact_nudges=0`）；两个 bot 选择先把草稿留在频道/notebook，并明确说「等源文件」。这是「不编数字」的正确行为，但也意味着本轮**无法**从循环行为上证明 J3-3 的提示会被读。
4. **人工请求分两类**：`request-9` 是计划批准（`dedupe_key` 幂等），`request-13` 是缺事实的信息请求（无 `dedupe_key`）。后者在整轮里只有一张，没有重复刷。
5. **调度器关闭理由（本轮新字段）**：`/scheduler` 里能看到两种理由在**互相顶替**——
   - `task-follow-up:ae--cos:journ-17` → `done`、`status_reason="superseded by recheck"`
   - `task-follow-up:analyst--cos:journ-21` → `done`、`status_reason="superseded by recheck"`
   - 随后 `recheck:ae--cos:task:journ-17`、`recheck:analyst--cos:task:journ-21` → `done`、`status_reason="superseded by task_follow_up"`
   这直接解释了 Journey 4 的疑问（`recheck` 为什么存在又为什么 `done`）：它不是「审阅门关掉」，而是状态变更把 job 顶掉了。`JOURN-15`、`JOURN-19` 从头到尾没有 `recheck`——它们的状态没离开过 `in_progress`，`scheduleTaskLifecycleLocked` 的 `in_progress` 分支只装 `task_follow_up`。
6. **告警**：整轮 `kind=watchdog_alert` 消息 0 条；`watchdogs[]` 在 04:52–05:18 为空，05:23:25Z 出现 1 条 `request_waiting`（目标 `request-13`，人类没答采访卡）；`task_stalled` 3 条（05:05:08 `JOURN-6`、05:08:08 `JOURN-15`/`JOURN-19`）。
7. **缺事实时循环停在哪里**：`request-13` 未被回答（本轮只允许一次人工干预且已用于批准），四条 lane 于是停在「等源数字」。人类不答，循环就不推进——这正是期望的行为，也说明「缺资料」这个问题在此路径上有可见出口，不是静默卡死。

## 3. 操作员探针（不是循环行为）

停表后，为验证部署在同一份代码上的拒写通道，操作员直接以 `analyst` 身份对其任务 `JOURN-15`（无 `definition`）调用 `/wiki/write-task`：

| 时间 | 调用 | 结果 |
|---|---|---|
| 见 §4 | `POST /wiki/write-task {slug:"analyst", task_id:"JOURN-15", path:"team/research/probe.md"}` ×2 | 两次均 403，文案含 `@cos`、`definition.artifact_path`、`cannot succeed`；`analyst__cos` 里只出现 **1 条** `kind=artifact_path_missing` 消息 |

该探针由人工脚本发起，日志里与 bot 的调用可区分（无对应 headless 回合）。

## 4. 探针结果

| 项 | 结果 |
|---|---|
| 探针时间 | 05:23:08Z（同一脚本内两次连续调用） |
| HTTP | 两次均 `403`，`{"error":"artifact path not declared: JOURN-15 declares no artifact_path, so no wiki path is authorized for it. @cos must set definition.artifact_path on the task (or route this deliverable through a human request); retrying this call cannot succeed"}` |
| 频道消息 | `analyst__cos` 只出现 **1 条** `kind=artifact_path_missing`（05:23:08Z）：「@cos: JOURN-15 has no definition.artifact_path, so @analyst's write to team/research/probe.md was refused. Pin the path on the task (or have @analyst deliver through a human request) — until then the lane can only leave notes in a notebook. Owner: @analyst.」 |
| 审计行 | `actions` 末尾出现 `kind=artifact_path_missing, related_id=JOURN-15` |
| 去重 | 2 次调用 → 1 条消息，成立 |

## 5. 停表时刻的另一批事实（05:23:25Z）

- `watchdogs` 出现 1 条：`kind=request_waiting`、`target_id=request-13`、`status=active`——办公室发现人类没答那张「缺资料」采访卡，把它挂成待跟进。**不是** J3-2 那种「唤醒被拒」。
- `requests` 增至 3 条：`request-9`（answered，计划批准）、`request-13`（pending，cos 问人类要合同事实）、`request-43`（pending，`analyst__human`，analyst 也直接问人类要源数据）。
- `task_stalled` 消息 3 条（05:05:08 `JOURN-6` 11 分钟、05:08:08 `JOURN-15` 与 `JOURN-19` 各 10 分钟）。
- 终态：`JOURN-6` running、`JOURN-15` running、`JOURN-17` decision、`JOURN-19` running、`JOURN-21` decision；五条**全部**无 `definition`。
- wiki 侧：本轮没有任何 `team/*` 业务产物落盘；两个 bot 的草稿落在 `agents/analyst/notebook/`（已复制到 `artifacts/notebooks/`）。
