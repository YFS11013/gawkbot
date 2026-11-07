# 最小配置试验（2026-11-02 → 2026-11-09）

## 为什么做

7 天前的问题是「office 这套使用场景还有意义」。当时能给的证据只有修复记录，没有使用记录：不知道简报有没有人读，不知道论点台账会不会有第二条，只知道团队层产生了大量维护成本（一天内修了 4 个协作状态机缺陷，净产出没有变化）。

这个试验把系统砍到「只保留有明确产出、且产出由脚本或定时器驱动」的部分，7 天后如果没有任何缺失，就证明团队层没有产生价值。

判据不是「感觉少了什么」，而是：简报照常出、论点台账照常判、你没有需要它去做的事。

## 保留（8 个 scheduler job）

| job | 作用 |
|---|---|
| `5` | 每个交易日 08:30 UTC 中文简报（owner: forex-watch） |
| `one-relay-events` | 系统：消息中继 |
| `request_follow_up` | 系统：请求跟进 |
| `review-expiry` | 系统：评审过期 |
| `task_follow_up` | 系统：任务跟进 |
| `task_recheck` | 系统：任务重检 |
| `task_reminder` | 系统：任务提醒 |
| `wiki-archive-sweep` | 系统：wiki 归档清扫 |

## 保留（Windows 计划任务）

| 任务 | 作用 |
|---|---|
| `WUPHF-BrokerWatchdog` | broker 挂死重启 |
| `WUPHF-ThesisReview` | A1 论点周审（周五 13:00） |
| `ForexBridge` | 驱动 `forex-bridge.py`，简报的数据源 |

`ForexBridge` 是简报唯一的数据输入，**不能停**。停掉它等于停掉简报本身。

## 停用

**Broker routine（10 个，`PATCH /scheduler/{slug} {"enabled":false}`）**

```
weekly-saturday-fund-portfolio-ratio-analysis-report
forex-bridge-refresh-daily-data-pull
data-keeper-daily-bridge-health-check
market-analyst-daily-direction-rating-after-forex-brief
daily-0800-utc-opencode-model-snapshot-to-data-space-space-7c90a37850d8b0a1
one-workflow:cos__human:daily-opencode-model-snapshot
check-scheduled-task-status-for-model-snapshot-daily-archive-verification
daily-model-snapshot-verification-routine
```

其中 4 个在停用前就已经 `enabled=false`，另外 4 个的 `next_run` 停在 2026-10-29/10-30 —— 它们已经 3–4 天没跑，只是状态还挂着 `scheduled`。**这批不是「能跑但没价值」，是「已经不跑了但没人发现」**，这本身是当时没有哨兵的证据。

**Windows 计划任务（3 个，`Disable-ScheduledTask`）**

```
WUPHF-ModelSnapshot      model-snapshot.ps1              模型快照
WUPHF-RoutineScorecard   routine-scorecard.ps1           系统给自己打分（自指）
WUPHFModelSnapshot       data-keeper/snapshot-models-run.vbs  与上面重复
```

`WUPHFModelSnapshot` 在 `\Gawkbot\` 路径下，按名字禁用会报「找不到指定的文件」，必须用 `Get-ScheduledTask -TaskName ... | Disable-ScheduledTask -InputObject`。

**任务（2 个，`POST /tasks/{id}/cancel`）**

```
ACMER-1119  每日方向解读            owner 为空，没有到决策的路径
ACMER-1150  今日投资机会点（人类点名）owner 为空
```

## 恢复

全部可逆，不需要重建任何东西。

```bash
TOKEN=$(cat "$TEMP/wuphf-broker-token-7892")

# broker routine：逐个打开
curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"enabled":true,"change_note":"restore from minimal-config trial"}' \
  "http://127.0.0.1:7892/scheduler/<slug>"

# 任务：用 cancel 之前的写法重开
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"action":"reopen","id":"ACMER-1119","created_by":"human"}' \
  "http://127.0.0.1:7892/tasks"
```

```powershell
# Windows 计划任务
foreach ($n in @('WUPHF-ModelSnapshot','WUPHF-RoutineScorecard','\Gawkbot\WUPHFModelSnapshot')) {
  Get-ScheduledTask -TaskName (Split-Path $n -Leaf) -TaskPath (Split-Path $n -Parent) |
    Enable-ScheduledTask
}
```

slug 里的 `:` 直接写，**不要**做 URL 编码（`%3A` 会让 curl 连不上）。

## 试验有效性的前提

**「没觉得少了什么」只有在系统确实在被调度的前提下才算证据。**

2026-11-02 的实测：`broker-state.json.pre-cos-close`（10-29 07:46 快照）里有 6 个 `enabled=true` 的 job，`next_run` 已经过期 61–103 小时，`last_run_status` 仍是 `ok`——那是停摆前最后一次运行留下的。调度器循环停了一整周，没有任何东西报警。

如果这 7 天里它再停一次，结论会是假的：不是「团队层没用」，是「它根本没在跑」。

所以每天（或至少在 11-09 判定的当天）必须确认调度器是活的：

```powershell
pwsh -NoProfile -File scripts/office/check-office-drift.ps1
```

- 首行 `DRIFT: none` 且 exit 0；
- 第二行给出比对范围（`比对 N 个 job、N 个计划任务、N 个运行中二进制`）——数字不对就是没查全；
- `next_run 已过期` 这条是 2026-11-02 新加的，专门抓「排班表上活着，实际没在跑」。

另有一条不在这个脚本覆盖范围内，但要写清楚因果：**那次一周停摆发生在 `WUPHF-BrokerWatchdog` 上线之前**。它最早的日志是 `2026-10-30T20:36:02`，一上线就抓到 5 个 overdue job 并执行了重启（`restart issued`）。所以「有 watchdog」和「watchdog 在起作用」是两件事，但这一件已经验证过了，不需要在试验期间再杀一遍进程去证明——那会断掉 bot 连接和正在跑的 turn。

**已知盲区（记录，本次不修）**：系统 cron（`wiki-archive-sweep` 等）的 `last_run` 是心跳时间而不是运行时间，`last_run_status` 常常为空。原因是 `updateSchedulerHeartbeat` 在真正执行前就调用了一次并写入 `LastRun`，而首次传入的 `runStatus` 是空串，所以状态字段留空；drift 的 `if ($job.last_run_status -and ...)` 遇到空值直接跳过。结果是**这几条 job 的失败对 drift 不可见**。它们都只做内部清扫，不影响简报，故列为已知而非待修——`updateSchedulerHeartbeat` 被所有系统 cron 共用，改动需要全量回归，与本周观察期的目标相悖。

## 7 天后怎么判

| 观察 | 结论 |
|---|---|
| 简报照常出，你没觉得少东西 | 团队层无价值，砍掉是对的 |
| 简报照常出，但你从没看 | 连简报都该砍，只留 `check_theses.py` |
| 你主动想让它做某件事，而它做不了 | 那就是唯一有价值的场景，做成单 agent 版本 |
| `TH-2026-11-02-001` 判定为 hit 或 miss | 论点台账机制验证通过，它值得保留 |

最后一条是这次试验里唯一有独特性的东西：`check_theses.py` 做的是人不会做的事——把判断写成可判死的条件、到期自动判、记账不可篡改。它不依赖 bot 团队，只依赖快照数据。

## 判定（2026-11-07 提前判定，不再等 11-09）

原定 11-09 结案，2026-11-07 拍板提前判定。**放弃的两项样本**（写在这里，不假装观察过）：

- job `5` 的 11-09 08:30 UTC 那一跑没等。已有 20 次简报记录（11-02..11-07），多一次不改变结论；
- `TH-2026-11-02-001` 到 11-09 才复核。但论点台账是 `check_theses.py` + 快照的脚本链，不依赖 bot 团队——它的成败不进入「团队层有没有价值」这个判断。

下面是窗口走完 5 天（11-02 → 11-07 11:00 UTC）的证据。

### 证据（2026-11-02 00:00 → 2026-11-07 11:00 UTC，wiki 411 次提交）

| 面 | 观测 |
|---|---|
| 业务产出 | **20 次** `artifact: forex brief`（job `5`，每交易日一次） |
| 人工需求 | **0 个**新建任务；`~/.wuphf/tasks` 自 11-02 起无写入 |
| 团队层自发活动 | `meeting-prep` 共 **44 次执行：1 次成功、2 次部分、41 次 aborted**（11-01 19 次、11-02 21 次、11-06 1 次）。wiki 只提交了 14 条 `playbook execution … aborted`，其余只落在 `meeting-prep.executions.jsonl` |
| 系统 cron | 7 条全部在跑；`wiki-archive-sweep` 每天 03:05 |
| wiki 噪声 | 303 次提交是 `team/alerts.md`（占比 74%），来源见下节；另有 8 次 `recover from crashed write` |
| 论点台账 | `TH-2026-11-02-001` 窗口 11-02..11-06 已关，复核 11-09，状态 pending |

### 试验的副作用（已修，记录在案）

停用采集器时没有同步告诉检查器「`enabled=false` 意味着不要查这条日志的新鲜度」，于是 drift 每 15 分钟报一次 `计划任务停更: WUPHF-ModelSnapshot 日志 99.8h 未更新`，文本里嵌着不断增长的小时数 → 内容哈希永不重复 → 去重失效：

- `alerts.md` 备份 1505 行里 **301 条**是这条告警（按 15 分钟节拍：11-03 56、11-04 96、11-05 96）；
- `watchdog.log` 里 `routine gate FAILED` **374 条**，原因同为 `snapshot is 118.7h old (limit 26h)`；
- 那份 33k 字符的 `alerts.md` 被 30 分钟一次的 skill-compile cron 反复判定，单篇 input 从 15k 涨到 28.6k，占当天 skillify 消耗的 99%。

修复在 `d94d36a`（`if ($t.enabled -eq $false) { continue }`）。**这条副作用本身就是试验的产出**：它证明「把东西关掉」需要配套改哨兵，否则噪声会把真正的告警淹掉——和 11-02 那天「4 条 routine 停了 3-4 天没人发现」是同一个病的两面。

模型目录 5 天里从 28 个涨到 83 个（`crouton-alpha-free` 等新模型出现）无人知晓，也是快照停摆的代价。**不因此恢复任务**：判据是「有读者」，而当时的读者并不存在。

### 结论

| 判据 | 结论 |
|---|---|
| 简报照常出 | 是，20 次 |
| 你没觉得少东西 | 成立。6 天里人类只发了 **1 条**消息，而且是回答 bot 的求助（让宿主机重跑 `forex-bridge.py` 重生 digest），不是提新需求；新建任务 0 个 |
| 团队层产生价值 | **没有**。8 个 enabled job 里 7 个是系统 cron，唯一业务产出来自一个定时脚本链；多 bot 自身产出的是 **10 条 `task_stalled` + 14 条 consult relay + 41 次中止的 playbook**，全部是维护自己的流量 |
| 论点台账值得保留 | 保留，但它与团队层无关：`check_theses.py` 只读快照数据 |

那 41 次中止的根因（`meeting-prep.executions.jsonl` 原文）：

> `task-skill-2923` 已被取消，执行上下文仍缺 `object_name` 与 `meeting_date` 两个必填输入，无法执行 playbook Step 1（wiki 检索）或确定输出路径。cos 无 `#data-keeper__human` 频道访问权限（403），无法回问发起方。停止重放，等待人类补输入后重建任务。

三个独立缺陷串在一起：① playbook 的 `## Inputs` 只是散文（`必填`），调用侧无法校验，于是缺输入的调用照样起执行；② 执行方被 403 挡住，问不了发起方；③ 记录里写着「停止重放」，实际每 30 分钟继续重放，直到人类取消任务才停。

**判定：多 bot 协作层在这 5 天里是净成本。** 唯一的业务产出（外汇简报）由「定时脚本 + ForexBridge + 一个 routine」这条链产生，去掉别的 bot 也能跑；而多 bot 层的实际产出是让自己看起来有事在做。

**由此决定不修的**：`meeting-prep` 那 41 次中止涉及的三个缺陷（散文式 Inputs 无法校验、403 无法回问发起方、abort 后仍每 30 分钟重放）**不修**。协作层不再投入，它们修好也没有下游；将来重新启用时按那时的需求重做，而不是修一条可能被删的路径。

**没有因此改动的**：最小配置本身（8 个 job、3 个计划任务、ForexBridge）维持现状，`expected-scheduler.json` 不动。这一判定是「不再投入」，不是「再砍一刀」——要不要再砍（含停掉简报）是另一个决定，需要人拍板。

### 附带结论：判定链路的价值边界

本日把技能判定链路换成单一模型（`crouton-alpha-free`）并加了兜底（`RUNTIME-API.md` §14.9–14.12）。按上面的判定，这条链路现在的消费方是 bot 团队——而 bot 团队 5 天没有接到任何活。也就是说：**技能编译目前是自己给自己产出，没有使用者**。它不是错的（真实需求出现时它就在那儿），但它不构成「保留协作层」的理由。
