# Journey 16 observations — 2026-11-11，被测 `897da73` / 二进制 `c0ff6f76…`

操作员记录：只写看到的事实与时间戳（UTC），不写结论；结论留给 `scorecard.md`。

## 0 环境

| 项 | 值 |
|---|---|
| prereg | `a4b3b2f`（开跑前单独提交） |
| 源码 | `897da73`（`git rev-parse --short HEAD`，工作树干净） |
| 二进制 | `c0ff6f76…`，`/health` = ok / `session_mode=office` / `provider_model=opencode-go/longcat-2.5-free` / `memory_backend=markdown`（active、ready）/ `wiki_ready=true` / `members=1` |
| runtime_home | `C:\Users\ireke\.wuphf-spaces\journey16` |
| broker / web | `:7910` / `:7911` |
| 主办公室 | `:7892` / `:7891` 同机运行（pid 60388，longcat） |
| 起跑时任务数 | 0 |
| 起跑时 `requests` / `actions` | 0 / 0 |

### 0.1 起跑前的两处环境动作（影响可比性，先记）

1. **watchdog 已修**：`~/.wuphf/data/broker-watchdog.ps1` 的 `Restart-Broker` 原先按 `Name='gawkbot-fix.exe'` 杀全部同名进程（README §6 的修法只是文档，脚本里没有）。本轮开跑前改为按监听端口 7892 定位主办公室、按 `ParentProcessId` 取后代；`opencode.exe` 同样只清主办公室树内的，隔离办公室子树列入 `keep`。验证：故意把 `$TEMP\wuphf-broker-token-7892` 写成坏 token 触发一次真实重启，主办公室 pid 40920 → 60388，隔离办公室 7910 仍为 47992 且 `/health` ok（两边同时 200）。备份 `broker-watchdog.ps1.bak-j16-20261111-090557`。
2. **隔离办公室改为手工启动**：`workspace create --from-scratch journey16` 起出来的 broker 用默认模型 `opencode-go/beam-free`（上游已退役）；`workspace pause --force journey16` 报 `context deadline exceeded` 且进程仍在，`resume` 报 `workspace is already running (port 7910)`。处理：`Stop-Process` 掉 pid 47992，用 `WUPHF_RUNTIME_HOME=…\journey16` + `WUPHF_OPENCODE_MODEL=opencode-go/longcat-2.5-free` + `WUPHF_HEADLESS_MODEL_FALLBACK=opencode-go/iceman-contributor-free` 手工 `Start-Process` 起 `--broker-port 7910 --web-port 7911 --no-open`，stdout/stderr 重定向到 `<home>\logs\broker.log|broker.err.log`。**判据外发现（D1）：workspace pause/resume 在这台机器上对该 workspace 不可用。**

## 1 时间线（UTC，取自 `artifacts/broker-state.json` 的 `tasks[].created_at` / `requests[]` / `actions[]`）

| UTC | 事件（原文/原值） |
|---|---|
| 01:17:54 | 操作员建 Issue `JOURN-1`（`cos__human`，`created_by=human`，owner `cos`，`task_type=issue`）→ `in_progress/planning`；`action-1 task_created` |
| 01:17:54 | `headless-codex-latency.log`：`bot=cos stage=started queue_wait_ms=120` |
| 01:22:36 | cos 投计划（`action-3`）并建事实采访 `request-3`（`interview`，无 dedupe_key）：`Acme 续约的条款（方案、价格、周期、续约调整）和 Dana 的联系方式不在办公室里，两份交付物都要用。你现在能提供吗？` |
| 01:22:36 / 01:23:49 / 01:25:06 | cos 三轮调用 `wuphf-office_human_interview`（opencode.log 三条 `evaluated permission=wuphf-office_human_interview`） |
| 01:26:17 | cos 放弃等待（`action-4`）：`采访卡片工具连续超时，我直接把问题发给你。`（`msg-6`），同时建计划卡 `request-7`（`plan-approval:JOURN-1`，`action-5`） |
| 01:27:06 | 操作员作答（干预 1/3）：`POST /requests/answer {"id":"request-3","custom_text":"…全部死数字…"}` → `{"ok":true}`；`action-6 request_answered`、`msg-9` |
| 01:28:43 | `msg-10`：“收到死数字，正在拆成两条子任务并推进 JOURN-1。” |
| 01:30:55 | 操作员批准计划（干预 2/3）：`choice_id=approve` → `{"ok":true}`；`action-13 task_updated [plan approved]`、`action-14` |
| 01:32:06 / 01:32:09 | 两条 lane 建立：`JOURN-18`（`team/accounts/acme-renewal-quote.md`）、`JOURN-20`（`team/emails/acme-renewal-followup.md`）；`action-15/16 task_created`；`action-17/21 verification_replaced none -> wiki_artifact <各自路径>` |
| 01:32:25 / 01:32:29 | **`definition-gap` 卡出现（新形状）**：`request-22`（`definition-gap:JOURN-18`）/ `request-25`（`definition-gap:JOURN-20`），正文均为 `success criterion: "公司名与发件人使用占位符 [COMPANY NAME] / [SENDER NAME]"` —— 命中确定性占位符检测器，不是 access 追问 |
| 01:32:25 / 01:32:29 | 两条 lane 的 `definition.access_needed` 为**空**（`artifacts/tasks.json`）——模型这一轮没有把 wiki 写入权限写进 access |
| 01:33:35 / 01:33:36 | 两条 lane `done/approved`，`artifact` 与 `definition.artifact_path` 逐字一致；同时 `action-25/27 request_canceled`：`task closed (done) Missing details for JOURN-18/JOURN-20`（`8026e40` 的关单撤卡级联生效） |
| 01:33:56 | `action-30 guard_refused`：`container artifact: team/accounts/acme-renewal-quote.md is JOURN-18's …` —— 模型把 lane 路径填进父 Issue 的 `artifact_path`，被 409 拦住 |
| 01:34:18 | `action-32 guard_refused`：`container close: cos tried to close it with 2 delivered lane(s)` —— 模型改从任务端点关容器，被 409 拦住 |
| 01:34:27 | 父 Issue 进 `review`（`action-33`） |
| 01:34:32 | 容器决策卡 `request-35`（`decision:JOURN-1`），正文：`This Issue's lanes delivered 2 file(s):` + 两条路径 + `The Issue ships no single file of its own; those are what you are accepting.` |
| 01:36:16 | 操作员批准（干预 3/3）：`choice_id=approve_decision` → `{"ok":true}`；`request-35 status=answered` |
| 01:36:03 | `JOURN-1` → `done/approved`，`artifact=null`，`sessionReport.metadata=null` |
| 01:36:16 | `wiki/team/decisions/JOURN-1.md` 由 `99ba193 decision: promote JOURN-1 on approval` 写入 |

### 1.1 卡片总量（`artifacts/requests.json`）

| id | kind | dedupe_key | 结果 |
|---|---|---|---|
| request-3 | interview | — | answered（人答 1 次） |
| request-7 | approval | `plan-approval:JOURN-1` | answered（人点 1 次） |
| request-22 | interview | `definition-gap:JOURN-18` | **canceled**（01:33:35 关单级联，人未看到/未答） |
| request-25 | interview | `definition-gap:JOURN-20` | **canceled**（01:33:36 关单级联，人未看到/未答） |
| request-35 | approval | `decision:JOURN-1` | answered（人点 1 次） |

`actions` 计数：`task_created 3`、`request_created 5`、`automation 13`、`request_answered 3`、`task_updated 7`、`verification_replaced 2`、`request_canceled 2`、`skill_update 1`、`guard_refused 2`。**`artifact_overwritten` 0 条。**

## 2 盘上产物

| 路径 | 字节 | 来源 |
|---|---|---|
| `wiki/team/accounts/acme-renewal-quote.md` | 848 | `JOURN-18`，`git log` 有提交 |
| `wiki/team/emails/acme-renewal-followup.md` | 811 | `JOURN-20`，`git log` 有提交 |
| `wiki/team/decisions/JOURN-1.md` | 131 | 批准后 promotion |

两份文件副本与 wiki `git log` 见 `artifacts/`。

## 3 判据外发现

- **D1** workspace pause/resume 不可用（见 §0.1 第 2 条）。
- **D2** `definition-gap` 卡的新触发面：`success_criteria` 里出现 `[COMPANY NAME]` 这类**故意**的占位符说明也会被确定性检测器判成缺口。两条 lane 都在建单后 66 秒被关单级联撤掉，人没有被打扰——`8026e40` 的撤卡在这里是承重的。
- **D3** 采访工具会阻塞并超时：01:22:36 建卡后，cos 三轮调用 `wuphf-office_human_interview`（01:22:36 / 01:23:49 / 01:25:06），01:26:17 自述「采访卡片工具连续超时」并在频道里手工重问（`msg-6`），随后 4 分钟里又把同一件事包成计划卡 `request-7`。人答的其实是 `request-3`（01:27:06），但同一问题在频道里出现两次。
- **D4** 计划卡正文引用了 `action-3` 的整段计划，且模型在 01:30:36（`action-11`）自述「子任务仍被计划的审批门槛挡住，我把计划正式提交给人类确认」——审批门槛按设计工作，但模型为此多花一轮。

## 4 证据强度

- 所有时间戳与字段取自 `artifacts/broker-state.json`（起跑目录原文件的逐字节副本）。
- 409 原文未落盘（broker.err.log 只有 warn 级），守卫拒绝以 `actions[]` 的 `summary` 为准，与 J15 同一口径。
- `artifacts/` 另有 `tasks.json`、`requests.json`、`actions.json`、`guard_refusals.json`、`wiki-git-log.txt`、`headless-codex-latency.log`、`broker.err.log`。
