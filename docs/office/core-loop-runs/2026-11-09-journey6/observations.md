# Journey 6 — 观察记录（2026-11-09）

> 只记时间戳事实。判分见 `scorecard.md`（独立 reviewer）。全部 UTC。
> 工作区 `journey6`（broker :7910，`opencode-go/crouton-alpha-free`）。停表：**45 分钟上限**（07:06 前后），未分解、无写尝试。

## 1. 时间线

| 时间 | 事实 |
|---|---|
| 06:20:59 | onboarding 完成；`ai-revops-1..4` 取消 |
| 06:20:59 | 父 Issue `JOURN-8` 创建（`cos__human`，owner cos，`created_by=human`），`lifecycle_state=planning` |
| 06:21:28 | cos 先说「诊断这是 analyst 卡死重试」——它在处理另一条任务 `JOURN-3`（标题 `[@analyst] Repeated errors blocked: Establish the pipeline-truth baseline`，`created_by=cos`，onboarding 之后自建） |
| 06:21:31 | cos 贴 `JOURN-8 计划`（目标/拆分/验收/风险，四段） |
| 06:21:32 | `request-12` 自动出现：`kind=approval`、`dedupe_key=plan-approval:JOURN-8`、`channel=cos__human`、`pending` |
| 06:22:00 | `request-16` 出现：`kind=interview`、无 dedupe、`cos__human`，问「管道真实数据在哪管理」——**属于 `JOURN-3`，不属于本轮场景** |
| 06:23:19 | 唯一一次人工动作：`POST /interview/answer {"id":"request-12","choice_id":"approve"}` → `{"ok":true}`；同秒 `JOURN-8` 被 update；`request-follow-up:request-12` 以 `target closed` 收尾 |
| 06:23:26 | cos 整轮正文谈的仍是 `JOURN-3` 的管道数据问题；**没有创建任何子任务** |
| 06:27:32 | 环境清理：取消 `JOURN-3`（`{"changed":true}`，`task_canceled` 落账）。这是**操作员动作**，不是场景的一部分 |
| 06:33:35 | `task_stalled`：「@cos has gone quiet on “Journey 6: Acme 续约报价与跟进” for about 10 minutes…」 |
| 06:52:15 | `request-follow-up:cos--human:request-16` 触发：写 `watchdogs[0]`（`request_waiting`）+ `actions` 一条 `watchdog_alert`，**没有发任何聊天消息**，随后 `next_run=07:22:15` |
| 07:01:35 | 仍只有 `JOURN-8`（`running`，无 `definition`），`tasks` 里没有任何子 Issue；`broker.log` 里 `write-task` 行数 **0** |
| 07:06 | 停表（45 分钟上限） |

## 2. 机制侧观察

1. **批准后的激活回合跑到了别的任务上**：06:23:19 的批准确实触发了 cos 的一次回合（06:23:26 有正文），但回合产出的是 `JOURN-3` 的管道问题回复。此后 42 分钟里 `JOURN-8` 没有任何新唤醒——它自己的 `task-follow-up:cos--human:journ-8` 下一次 `next_run=07:23:19`。**批准 → 首次分解**之间没有短周期重投，只有小时级 follow-up。
2. **没有触发 `task_retry`**：`ScheduleNoTraceRetry` 只在回合「什么都没留下」时排；06:23 那轮留了消息，所以不排。也就是说「回合有产出，但不是这本任务要的产出」这种状态，系统看不见。
3. **非阻塞采访的提醒是静默的**：`request-16` 的 follow-up 在 06:52:15 触发，只写 `watchdogs` 与 `actions`（`kind=watchdog_alert`），聊天里**一条消息都没有**（`processRequestJob` 只在 `req.Blocking && !existing` 时投 automation 消息；`request-16.blocking` 为空）。同型现象在 Journey 5 也有（`request-13` 的 `request_waiting`）。
4. **本轮完全没有写 wiki 的尝试**：`broker.log` 的 `write-task` 行数为 0，`artifact_path_missing` 消息为 0。因此 J3-3 的 403→提示→改做法这条链**本轮未被触发**（prereg §9 已预告这种结局）。
5. **场景被环境干扰**：onboarding 之后办公室自建了 `JOURN-3`（`[@analyst] Repeated errors blocked`），并把它和场景任务放进同一个 `cos__human`。cos 的选择是先处理旧任务。取消 `JOURN-3` 后也没有补上分解动作——说明这不是「一次分心」，而是**批准后没有分解检查点**。

## 3. 本轮不判分的补充事实

- `JOURN-8` 计划的验收口径写得很实（金额/席位/有效期/付款与任务单一致、无额外假设），与任务单给定事实吻合；问题不在计划质量，而在计划批准之后没人再推动它落地。
- `request-12` 的 `request-follow-up` 在答卡后立刻以 `target closed` 收尾——与 Journey 5 一致，关闭理由字段按预期工作。
- `task_stalled` 消息在 06:33:35 正常发出（可见），但没有任何后续唤醒。

## 4. 修复验证（同日，隔离办公室 `j6smoke`，已 shred）

针对上面 §2 第 1/3 条（批准后无分解检查点、非阻塞采访提醒静默）落了两处代码，并做了真机验证。记录在这里是为了让「修了」不是只靠单测。

### 4.1 批准会装一个 5 分钟的分解检查 job（真机）

`POST /interview/answer {"id":"request-28","choice_id":"approve"}`（`plan-approval:JSMOK-2`）在 07:22:44Z 生效，同一次保存里出现：

```
task-follow-up:cos--human:jsmok-2              task_follow_up           scheduled  due 08:22:44Z
plan-decomposition-check:cos--human:task:jsmok-2 plan_decomposition_check  scheduled  due 07:27:44Z
```

### 4.2 任务在检查点之前就交付 → job 静默收尾，无噪音

`JSMOK-2` 07:23:09Z 直接 `approved`（cos 25 秒内交了一页报价，没有拆子任务——单回合就能交付的计划本来就该这样）。两个 job 同时关闭：

```
plan-decomposition-check:…:jsmok-2  done  last 07:23:09Z  status_reason=target closed
```

即「任务离开 running → 收尾」这条分支在真机上走通了，且没有产生 `task_not_decomposed` 告警。

### 4.3 「无子任务 → 唤醒」分支只在单测里走通（真机未能制造）

试着在停掉的 broker 上注入一个 `running` 且无子任务的 `SMOKED-1` + 到期的 `plan_decomposition_check` job，再 resume。结果 broker 重启时就对这条 running 任务补发了一次唤醒，cos 12 秒内把它拆成 `JSMOK-51`/`JSMOK-53`，检查 job 则被随后的生命周期重排以 `superseded by task_follow_up` 收尾——**没到「无子任务」那一刻**。

结论按证据分级：

- **A 级（真机）**：批准装 job（4.1）；任务完成时 job 收尾、零告警（4.2）。
- **B 级（单测，真实调度代码路径）**：`TestPlanDecompositionCheckWakesOwnerWhenNoLanes`（无子任务 → `task_not_decomposed` + 唤醒一次 + one-shot 关闭）、`TestPlanDecompositionCheckClosesWhenLanesExist`（有子任务 → 不唤醒、不告警、写明条数）、`TestApprovedPlanArmsDecompositionCheck`（去掉装 job 即红）。
- **未验证**：真机上 owner 因这个检查被唤醒后的回合会怎么走（那是模型行为，不是接线）。

另外记一条真机观察：检查 job 会被任何后续生命周期重排顶掉（`superseded by task_follow_up`）。这在语义上是对的（状态变了说明有进展），但意味着它只在「批准后什么都没发生」时才可能触发——正是设计目标。

## 5. 产物与代码证据

- `artifacts/wiki-commits.txt`：wiki 仓库全部 28 条提交，最新一条 `2026-11-09 14:20:54 +0800`（= 06:20:54Z，onboarding 的 bootstrap）。**任务 06:20:59 提交之后没有任何一次 wiki 提交**，作者全是 `wuphf-bootstrap` / 各 seed，没有 `cos`。这比「broker.log 里 0 行 write-task」更强：写路径确实一次都没走通。
- `artifacts/wiki-tree.txt`：提交时刻的 wiki 文件清单（48 个文件，全为模板/skill）。
- `artifacts/broker-log.txt`、`artifacts/headless-codex-cos.log`：该轮 broker 与 cos 运行日志原文（体量小，因为 45 分钟内只跑了 3 个回合）。
