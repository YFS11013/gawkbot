# Journey 10 — 预登记（2026-11-10）

> 本轮回答的是 Journey 9 的两个修复**在真机上是否成立**：D2（球在人手里时必须有一张点得动的卡，§9.19，`76e6171`）与 D6（任务声明的产物必须落在声明的路径上，§9.20，`52456d3`）。
> 场景与 J7/J8/J9 **逐字相同**，差异只能来自机制，不能来自题目。
> 判据口径按 `core-loop-runs/README.md` §6 四条写死：计数源、文案按实际分支、`decision:` 卡 = 「球在人手里」、产物只认盘上文件。

## 1. 被测提交

- `52456d3` fix(wiki): keep a task's declared artifact on the path the task declared
  - `EnqueueTaskArtifact` + `wikiWriteRequest.PinnedPath`：`/wiki/write-task` 不再被 update-first 改道
- `76e6171` feat(team): ask the human for the decision a lane is waiting on
  - 三个收敛出口各发一张非阻塞卡（`dedupe_key=decision:<taskID>`，随 lane 关闭自动撤）
  - 应答走 `RecordTaskDecisionWithComment`；approve → `approved`，send back → `changes_requested` + 原话
- `fdb6fa8` fix(team): 写范围门（foreign artifact / 视觉替代 / 落地后改 artifact 一律拒绝）
- 基线：`690b601`（J9 被测）、`4863c50`（J9 prereg）

### 1.1 模型在上游被撤（2026-11-10 10:36 本地，提交场景之前）

J7/J8/J9 全用的 `opencode-go/crouton-alpha-free` 已从 opencode-go 的模型表里消失。硬证据：

```
$ opencode run --model opencode-go/crouton-alpha-free "reply with the single word: pong"
Error: Model not found: opencode-go/crouton-alpha-free. Did you mean: beam-free, iceman-contributor-free, longcat-2.5-free?
```

同一个标识在办公室里的表现是另一副样子：每个 headless 回合 2 秒内 `exit status 1`，频道里落一堆 `UnknownError: Unexpected server error`（`journey10/logs/headless-codex-*.log`、`broker-state.json` 的 `msg-5…28`），没有一条可读的「模型不存在」。

替换模型的三个候选都实测可用（同一句 `pong` 探针全过），再按「同一道带工具的真活儿」比较：

| 候选 | 结果 | 耗时 |
|---|---|---|
| `opencode-go/beam-free` | 读两次才落到 `internal/buildinfo/VERSION` | 22 s |
| `opencode-go/longcat-2.5-free` | 一次读对 | 16 s |
| `opencode-go/iceman-contributor-free` | 仅拼 `pong`，未测工具 | — |

**本轮选 `longcat-2.5-free`**：两个可用候选里更快且一次读对。代价写在这里：**与 J7/J8/J9 不再同模型**，因此跨轮差异里必须扣掉「模型换了」这一项——凡是只在 J10 出现的失败，先排除模型能力差异再归因到机制。主办公室同步换成同一个模型（同一晚 crouton 撤掉后它的每个回合也在失败）。

## 2. 环境

| 项 | 值 |
|---|---|
| 隔离办公室 | `journey10`（broker :7910，web :7911） |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/longcat-2.5-free`（**赛前改动，见 §1.1**） |
| 蓝图 | `ai-revops`，agents `cos, revops, analyst, ae, reviewer` |
| 主办公室 | 7891/7892，**不得被本轮任何命令指向** |

### 2.1 隔离手段（与 J7/J8/J9 同法）

1. onboarding 后、提交场景前立即取消 `ai-revops-1..4`；
2. 每次快照检查是否出现非场景任务/采访卡，出现即取消并逐条记进 observations，判分单列「环境清理」；
3. 场景本身不带 `definition`（让规划者自己决定 lane 写范围）。

## 3. 场景

父 Issue：`channel=cos__human`、`owner=cos`、`created_by=human`、`task_type=issue`、无 `definition`：

```
title:   "Journey 10: Acme 续约报价与跟进"
details: "客户 Acme（40 席位，SaaS）。现行合同：年付 $18,000，2026-12-31 到期，付款 30 天。
          拟续约：年付 $21,600（+20%），条款不变，有效期 14 天。对方决策人 Dana（VP Ops）。
          两件交付物：① 一页报价说明（价格/范围/有效期/付款/决策点）② 48 小时内的跟进邮件草稿。
          只认以上数字，不要引入其它假设。"
```

除标题前缀 `Journey 10` 外与 J7/J8/J9 逐字相同。

## 4. 验证项

V1–V10 沿用 J9（可比）。V11–V15 是本轮的对象。

| 编号 | 判定 | 通过判据 |
|---|---|---|
| V1 | 每条 lane 都带写范围 | `broker-state.json` 里每个子 Issue 都有 `definition.artifact_path` 或 `artifact_not_required`（后者带理由且合理） |
| V2 | 共享 wiki 真的有交付 | `team/**` 出现 ≥1 份 markdown，`git log` 里**该路径有提交**且文件在盘；`broker.log` 至少 1 条 `write-task` 成功（**`bytes=` 本身不算证据**，§6 第 4 条） |
| V3 | 守卫是提示，不是墙 | 规划者撞写范围守卫 ≤ 每条 lane 1 次，且撞后改做法；不得「原样重试同一 create」≥2 次 |
| V4 | 不回归 | 计划卡自动出现且只有一张；`task_not_decomposed` 0 条；`actions[].kind=watchdog_alert` 0 条（唯一计数源）；拆 lane 后 `plan_decomposition_check` 带 `status_reason` 正常收尾 |
| V5 | 反向臂（停表后探针，不计场景干预） | 用错误路径打 `/wiki/write-task`：403 文案点名声明路径与尝试路径；`artifact_path_missing` 无新增 |
| V6 | 不刷屏 | 频道里因写范围被拒的 create 总数 ≤ 5 |
| V7 | 同父 lane 不重复 | 无两条同时 open 的 lane 声明同一 `artifact_path`；无 CJK shingle 判据可识别的重复标题 |
| V8 | 视觉 artifact 不能冒充交付 | 声明了 `artifact_path` 的 lane 以 `ra_` 收 done 必须 409；不得出现「声明了路径、`team/**` 无该文件、任务 done」 |
| V9 | 落地唤醒 | 任一 lane 落地后 ≤ 3 分钟内父 Issue 或仍 open 兄弟的 `task_follow_up.due_at` 被提前；同一 Issue 在仍有 open lane 时的最长静默 ≤ 10 分钟 |
| V10 | 分解计数 | `plan-decomposition-check` 的 `status_reason` 中 lane 总数与实际子 Issue 数量一致 |
| **V11** | **decision 态有一张卡（D2-1）** | 任一 lane 进 `lifecycle_state=decision` 后 ≤ 3 分钟内，`requests` 出现 `dedupe_key=decision:<taskID>`、`status=pending` 的卡；卡关联 `issue_id`；任务频道有对应通知；`pending_interview` **仍为 null**（非阻塞） |
| **V12** | **卡不堆积（D2-2）** | 卡出现后跨 ≥2 个 sweep 间隔（≥60 秒），同一 `dedupe_key` 的卡数恒为 1；`broker-state.json` 的 `requests` 总数不因重复 sweep 增长 |
| **V13** | **点击即落地（D2-3）** | 点 `approve_decision`：≤10 秒内 `status=done` 且 `lifecycle_state=approved`，卡从 `requests` 消失；点 `request_changes_decision` + 文字：≤10 秒内 `lifecycle_state=changes_requested`，原话出现在 `changes_requested.body`；两次点击路径都不得返回 200 而状态不变 |
| **V14** | **返工同轮发生（D2-4）** | send back 之后 ≤15 分钟内 owner 出现 active 回合或新消息，并推进（改写文件 / 重新 `submit_for_review`）；不得停在 `changes_requested` 无动作 |
| **V15** | **声明路径即落地路径（D6）** | 每条 `write-task` 200 的路径在 `git log` 有提交且文件在盘；不得出现「200 + `bytes=` 有日志、声明路径不存在」；相似名邻居文件不得被追加本任务内容 |

## 5. 硬失败

- **F-A（回退触发）**：同一 lane 因写范围被拒 ≥3 次且 create 内容不变。
- **F-B（回退触发）**：所有 lane 的 `artifact_path` 都是敷衍值（`team/tasks/x.md` 这类），或声明 `artifact_not_required` 而实际有交付物。
- **F-C**：计划卡缺席（J3-1 回归）。
- **F-D**：拆 lane 成功但 60 分钟内 `team/**` 仍为 0 份。
- **F-E**：批准后 10 分钟内既无子任务也无 `task_not_decomposed`。
- **F-F**：同一父 Issue 下出现两条同时 open、声明同一 `artifact_path` 的 lane。
- **F-G**：声明了 `artifact_path` 的 lane 用 `ra_` 收 done 且被放行，而声明文件不在盘。
- **F-H**：一条 lane 落地后，同一 Issue 仍静默 ≥ 15 分钟且无 active 回合**且没有未答的 `decision:` 卡**（有卡 = 球在人手里，不算停表；§6 第 3 条）。
- **F-I（D2-1 失败）**：lane 进 `decision` 后 ≥10 分钟仍无 `decision:` 卡，且 `pending_interview` 为空 —— 人类没有任何可点的东西。
- **F-J（D2-3 失败）**：点击返回 200 但 2 分钟内 `lifecycle_state` 未到目标态（状态分裂），或点击后卡未撤。
- **F-K（D2-4 失败）**：send back 后 15 分钟内 owner 无回合、无消息、无 follow-up 提前。
- **F-L（非阻塞被误用）**：卡出现后 15 分钟内**其余**仍 open 的 lane 无任何推进且无 active 回合（卡被当成阻塞闸门）。
- **F-M（D6 失败）**：出现「`write-task` 200 但声明路径无文件」或任务产物落进相似名邻居。

**回退**：F-A/F-B → 切 B（接受 notebook + visual artifact 为交付载体）。F-F/F-G → 对应判据判不成立。F-H → 重新评估唤醒（不是卡的问题）。F-I/F-J/F-K → D2 判「真机不成立」，回到 §9.19 重新论并考虑把应答改为阻塞闸门（那会让每轮被人工闸门挂起，故仅限于「lane 已交付、只差签字」的终态）。F-M → D6 判不成立，路径钉子回滚为诊断日志。

## 6. 人工干预

**只允许 3 次点击**，全部记时间戳：

1. 计划卡 approve（1 次）；
2. 第一张 `decision:` 卡 approve（验证 V13 正向）；
3. 若出现第二张 `decision:` 卡，点 send back + 一句返工要求（验证 V14）。

取消 onboarding 遗留、清理非场景任务逐条记进 observations，判分单列「环境清理」。停表后的反向臂探针（V5）与 D6 探针单列，不计入场景干预。

## 7. 停表

连续 15 分钟无新消息、**且**无 active 回合、**且** `requests` 里没有未答的 `decision:` 卡；或 60 分钟上限。停表后**先**把 `wiki-log.txt`、`team-tree.txt`、lane notebook、`broker.log` 摘要、`broker-state.json` 的 `tasks`/`requests`/`scheduler`/`actions` 段复制进 `artifacts/`，**再** `workspace shred`。快照每 5 分钟一次到 `snapshots/HHMM/`。

## 8. 判分

记录者（主代理）只写 `observations.md`（时间戳事实，不写结论）；`scorecard.md` 由**独立**子代理按本文件的 V1–V15 与 F-A…F-M 写，跑完再对。判分者必须回到 `artifacts/` 与 `snapshots/` 原始证据，不得采信 observations 的结论化措辞。
