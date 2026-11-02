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

## 7 天后怎么判

| 观察 | 结论 |
|---|---|
| 简报照常出，你没觉得少东西 | 团队层无价值，砍掉是对的 |
| 简报照常出，但你从没看 | 连简报都该砍，只留 `check_theses.py` |
| 你主动想让它做某件事，而它做不了 | 那就是唯一有价值的场景，做成单 agent 版本 |
| `TH-2026-11-02-001` 判定为 hit 或 miss | 论点台账机制验证通过，它值得保留 |

最后一条是这次试验里唯一有独特性的东西：`check_theses.py` 做的是人不会做的事——把判断写成可判死的条件、到期自动判、记账不可篡改。它不依赖 bot 团队，只依赖快照数据。
