# RUNTIME-API.md — broker 运行时接口契约

适用：巡检、关单、改 routine、读写数据空间、改 app 元数据。全部是**运行时**数据，不进 git；
权威接口是 broker HTTP，不是 `~/.wuphf` 下的 json/sqlite（内存持有，手改会被覆盖）。

鉴权：`token = GET 127.0.0.1:7891/api-token`，之后 `Authorization: Bearer <token>` 打 `7892`。
每次 broker 重启 token 会变，且 `%TEMP%\wuphf-broker-token-7892` 会被重写。

## 1. 任务

| 操作 | 请求 |
|---|---|
| 列表（含各 lane + 已关闭） | `GET /tasks?all_channels=true&include_done=true`（不带参数只回 general，会漏 DM-first 任务） |
| 建单（绕开 CoS 的唯一可靠链路） | `POST /tasks {"action":"create","title","details","owner","created_by":"human"}`，**不带 channel** |
| 关单 | `POST /tasks {"action":"complete","id":"ACMER-NNNN","created_by":"human"}` |
| 停单（终态，不会被再唤醒） | `POST /tasks/{id}/cancel {"reason":"..."}`，返回 `{"task":…,"changed":bool}`；已终态再调返回 `changed=false` |
| 数「空壳完成」 | `status=done && artifact 为空 && artifact_not_required 为空`（`hollow_completion=true` 是同一事实的显式标记）。2026-11-08 起新关的单**不可能**落到这个形状：done 必须带已校验的 `artifact_path`，或显式 `verification=not_required` + `verification_reason`（落在 `artifact_not_required`），见 §14.15 |

`complete` 走 `MutateTask`（`internal/team/broker_tasks_mutation_service.go`，done 转换处理在 `reachedDone` 分支）。

**`cancel` 与 `block` 的区别是终态**：`block` 是「可恢复、等上游」，`scheduleTaskLifecycleLocked` 会立刻给它排一个 `recheck` job，所以 blocked 任务会被再唤醒——它停不住一个跑飞的任务。`cancel` 落到终态分支，清空 `follow_up_at`/`recheck_at`/`reminder_at`/`due_at` 并把 scheduler job 置 `done`，因此不会被再唤醒。实测 2026-11-02：`task-skill-2923` 手工 block 后又跑了 2 轮（42 轮累计），改 canceled 后才真正停。

`cancel` 在 2026-11-02 之前不存在，停任务只能停 broker、手改 `broker-state.json` 的 `status`、再启动——而那套流程会因为 `Stop-Process` 后进程仍在优雅退出路径里 `saveLocked()` 而反复覆盖手工改动。

### 1.1 「Bot stuck on」求救单

`[@bot] Bot stuck on: <原任务>` 是 error-recovery 自动建的止血单（details 末尾带 `Original work: … · ACMER-NNNN`）。
原任务已解决就**直接 complete**，不要派活给它。

- 关掉一张**常会再生一张**（例：2710 → 2743）——关完必须再 `GET /tasks?all_channels=true&include_done=true` 复查。
- 同一个求救单反复冒头（例：`task-skill-1882` 之后又出 2730）= 根因没修，关单只是止血，要在同轮指出根因。

### 1.2 干净上下文审阅回合（`internal/team/task_review.go`）

任务**通过机器校验并转到 done** 后，broker 自动开一张审阅任务，交给**非作者**的 bot：

| 字段 | 值 |
|---|---|
| `task_type` / `pipeline_id` | `research` / `review` |
| `source_signal_id` | `review:<被审阅任务ID>`（幂等键，重复触发不会开第二张） |
| `depends_on` | `[<被审阅任务ID>]` |
| `owner` | `WUPHF_TASK_REVIEWER` → 办公室 lead → 按 slug 排序首个非作者成员 |
| `definition.artifact_path` | `team/reviews/<被审阅任务ID>.md` |
| `verification` | `wiki_artifact` + `required`（审阅自己也要过机检，不能空手关单） |

审阅者拿到的 `details` 只有：被审阅任务 ID/作者、原始 goal、原始成功标准、交付物路径。**不含**作者的聊天、计划、推理——这就是"干净上下文"。

跳过条件：任务没有 `Definition`、没有 `artifact`、`verification.pass=false`、父任务本身是审阅、非 done 状态、办公室没有第二个 bot、`WUPHF_TASK_REVIEWER=off|none|disabled`。审阅**不改**被审阅任务、不重开、不 veto —— 结论只给人/CoS 决策。

**写授权**：审阅者不需要人类逐次批准就能写那份文件。`definition.artifact_path` 就是它被授权的唯一路径，写入口是 `POST /wiki/write-task`
（`{slug, task_id, path, content, mode, commit_message}`，`internal/team/wiki_handlers.go`）：

| 情况 | 结果 |
|---|---|
| task 不存在 | 404 |
| 调用方不是该任务 owner | 403 |
| 任务已 done/failed/cancelled/archived | 409 |
| 路径 ≠ 任务声明的 `artifact_path` | 403（`declares artifact_path "…", not "…"`） |
| 通过 | 200 + `{path, commit_sha, bytes_written}`，并在 `broker.err.log` 记 `wiki write-task: task=… slug=… path=… bytes=…` |

MCP 侧：`team_wiki_write` 增加 `task_id` 参数；给了 `task_id` 就不再要求 `human_request`。
**路径必须在 `team/` 下**——`wiki_fs.go:164` / `wiki_git.go:1321` 只接受 `team/` 根，`reviews/…` 会被 wiki 层拒绝。

成本与收益怎么判断（依据 S3：干净上下文审阅者审同模型写的 PR，平均 2 bug/58% 严重）：见 `RESEARCH-2026-11-01-multi-agent.md` §6 的自测口径——
"阻断"级发现数 ÷ 被审阅交付数低于 0.5 就该关掉或只对特定 pipeline 开。

巡检证据（2026-11-01 实跑）：

```bash
grep "wiki write-task" ~/.wuphf/logs/broker.err.log | tail -3
ls -l ~/.wuphf/wiki/team/reviews/
curl -s -H "Authorization: Bearer $TOKEN" "127.0.0.1:7892/tasks?all_channels=true&include_done=true" \
  | python -c "import sys,json;print([t for t in json.load(sys.stdin)['tasks'] if t.get('source_signal_id')])"
```

## 2. 调度 / routine

| 操作 | 请求 |
|---|---|
| 全部 job | `GET /scheduler`（含 `kind / payload / enabled / target_id / last_run / last_execution_status`） |
| 改 job | `PATCH /scheduler/{slug}`，可写 `enabled / schedule_expr / interval_override / payload / label / target_type / target_id / channel / change_note`，每次生成 revision（`internal/team/broker_scheduler.go:823`） |
| 强制跑一次 | `POST /scheduler/{slug}/run` |
| bot 注册栈（dedup） | `POST /scheduler/routines`（按 purpose+schedule+channel+target 去重更新） |

`/routines` 是 bot service 内部端点，不是这里的东西，别混。

**`kind=agent_routine` 的 job 到点后建 task，不再发 routine 消息**（2026-11-01，`internal/team/broker_routine_tasks.go`）。
以前 routine 只 post 一条 `Kind=automation` 消息，工作做完没有 task 承接 artifact，core-loop 第 7 步（蒸馏）对每日简报永远空转。现在：

- 引擎用 `created_by=system` 建一张 task（`title`=job.label，`details`=job.payload，`owner`=job.target_id，`channel`=解析出的 DM/频道），`task_created` 的分发就是唤醒——**不再额外发 routine 消息**（发了就是双唤醒，第二趟没有 task）。
- 去重键是 `source_signal_id = routine:<slug>:<job.next_run>`：同一班次重放（重启补跑）不会叠第二张 task。
- 建 task 失败时回落到旧的消息路径，并在 Runs 里记一条 `failed`（`Message` 带原因）。所以 Routines 里出现 `failed` + 消息照常落地 = task 层坏了，不是 routine 坏了。
- `POST /scheduler/{slug}/run` 只把 `last_run` 标记为 triggered；**agent routine 的手动触发不会立刻执行**（它的 `next_run` 不动）。要立即跑，临时 `PATCH schedule_expr` 到两分钟后，跑完改回来。
- 非 `agent_routine`（`task_follow_up` / `request_follow_up` / `recheck` / workflow / operator）仍走原路径。

**routine task 自带 `verification_kind=wiki_artifact`（required）**，这是"打通蒸馏"的另一半。
蒸馏（`internal/team/task_distill.go`）只在 **done + 机器校验通过** 时才写 learning 记录；只建 task 不校验，简报照旧不进记忆。该 kind 的判定：

- 判定对象是**complete 时自报的 `artifact_path`**（wiki 相对路径，如 `artifacts/forex/2026-11-02.md`）。
- 必须落在 `WikiRootDir()` 内且是非空文件：没传 `artifact_path`、文件不存在、空文件、`../` 越界，一律 fail 并把原因写进 `verification_result.detail`。
- **文件必须是这张 task 存在之后写的**（mtime ≥ task `created_at` − 5 分钟容差）：拿一份旧页面冒充交付会被打回，detail 写 `was last written <时间>, before this task existed`。这条挡的是"先声明后交付"——bot 用预判的 id/路径 complete，而文件还没落盘或根本是别人的活（2026-11-01 两个演练各命中一次）。
- `complete` 没有 artifact 会被这条挡住（`verification_kind` required），bot 的修法是把交付物写进 wiki 再 complete——这正是我们想要的：简报必须落盘才算交付。
- 校验通过 → `distillCompletedTask` 记录的 learning 进 `team/learnings.jsonl` 并生成 `learning: <type>/<key>` 提交；实体抽取另有 `archivist:` 提交（无实体命中时不会有）。

**`workflow_provider` 两种解析路径**（决定一个 workflow job 是真跑还是每天失败）：

- `composio`：定义 JSON 存本地 `~/.wuphf/workflows/composio/<key>.json`
- `one`：同样读 `~/.wuphf/workflows/one/<key>.json`（`internal/action/workflow_store.go:15-25`，目录取 `filepath.Dir(config.ConfigPath()) + /workflows/<provider>/`）
- **文件不存在 = 每次运行都 `failed`**，且 skill record 里只有 `workflow_key`，没有 `workflow_definition` 可改。要修只能补定义文件，PATCH payload 无效。

## 3. 数据空间

| 操作 | 请求 |
|---|---|
| 空间 + schema | `GET /data/spaces/{space_id}`（返回 object_types / attributes / record_count） |
| 查记录 | `POST /data/spaces/{space_id}/records/query`，body `{"object_type":"<slug>","limit":20}`（`internal/team/broker_data_routes.go:258`） |
| 建记录 | `POST /data/spaces/{space_id}/records`，body `{"object_type","items":[…]}` |
| upsert | `PUT /data/spaces/{space_id}/records`，body 加 `"matching_attr"` |
| 改一条 | `PATCH /data/spaces/{space_id}/records/{rec_id}`，body `{"values":{…}}` |

**必须同时带 `X-WUPHF-Agent: <bot>`**（`internal/team/broker.go:49`）。只带 token 会 403：

```
{"error":"this request did not identify its caller; data spaces are reachable by a bot (X-WUPHF-Agent) or by the operator's own workspace UI"}
```

`GET /apps/{id}/data-space` 只回 `{"space_id"}`，不是数据。

## 4. App 元数据

| 操作 | 能改什么 |
|---|---|
| `GET /apps` | 列表（只有 `status=ready` 的 app；`app.json` 为 0 字节的损坏 app 不出现） |
| `GET /apps/{id}` | `{app, html}`；`?source=1` 额外给 source + capabilities |
| `PATCH /apps/{id}` | **只有 `{"name":...}`（rename）**，human 或 App Builder 才允许（`internal/team/broker_apps.go:439`） |
| summary / description | 无法直接 PATCH，只能由 App Builder 的 `register_app` 写入 |

因此「改 app 文案」= 起一次 App Builder（版本号 bump + 热重载）；纯文案差异不值得起构建。

**app 页顶部「Connect GitHub」横幅**的显示条件在 `web/src/appdetail/components/AppIntegrationBanner.tsx:39-69`：
`describedIntegrations(summary + description)` 命中关键词表（github/slack/hubspot…）**且**该集成未连接才渲染，
`refs.length === 0` 直接 `return null`。所以「去掉横幅」= summary/description 里不出现产品名，改完立即生效，无需重新构建。

## 5. 已知数据链路（谁在真写，别再考古）

| 产出 | 真实写入方 | 证据 |
|---|---|---|
| 模型快照（wiki + 数据空间 `models_snapshot`） | Windows 计划任务 `WUPHF-ModelSnapshot` → `~/.wuphf/data/model-snapshot.ps1`（08:00 UTC，按 `date` upsert，空值写 `''`） | `~/.wuphf/logs/model-snapshot.log`；记录 `created_by="model-snapshot"` 是脚本常量，无同名 bot |
| forex digest | `~/.wuphf/data/forex-bridge.py`（VBS 无窗口，每 6h） | `forex_bridge.json` mtime |

broker 里同名 routine（`one-workflow:…daily-opencode-model-snapshot`、`daily-0800-utc-…`）都不是写入方；
前者因定义文件缺失每天 `failed`（2026-10-31 已 disable）。

## 6. Bot 记忆（`/notebook/*`）

**背景（2026-11-01 之前）**：MCP 的 `context_capture` / `context_lookup` / `context_promote`
调用 `internal/teammcp/context_tools.go` 里的四个端点，而 broker **一个都没注册** → 恒 404。
更糟的是客户端把 404 标成 `retryable`，bot 读到「可重试」就反复重发同一个笔记，有一条直接在
turn 里放弃并把产物标成 "pending on the storage side"。日志证据：`~/.wuphf/logs/headless-codex-*.log`
中 10-24 有 23 条 `/notebook/*` 404。

| 端点 | 语义 | 边界 |
|---|---|---|
| `GET /notebook/catalog` | 列出有书架的 bot：`{"agents":[{agent_slug,path,file_count}]}` | 跨 bot 只读（`context_lookup` 需要知道谁有笔记）；按 slug 升序 |
| `GET /notebook/search?slug=&q=` | 单 bot 书架内**字面量**检索（非正则），≤100 命中 | 只走 `agents/<slug>/notebook/`，绝不跨书架 |
| `POST /notebook/write` | `{slug,path,mode,content,commit_message}` → `{path,commit_sha,bytes_written}` | 只允许 `agents/<slug>/notebook/<file>.md`（单层、无子目录）；`create` 撞已存在回 400，要追加用 `append_section` |
| `POST /notebook/promote` | 起一张 **human 审批卡**（快照正文 + sha256），→ `{promotion_id,state:"pending",human_only:true}` | 从不直接写 `team/`；target 必须 `team/<section>/<slug>.md`；同内容重复请求复用同一张卡 |

失败语义（`internal/teammcp/context_tools.go` 的 `notebookErrorCode`）：

| 情况 | code | retryable |
|---|---|---|
| 404 / 405 | `notebook_endpoint_unavailable` | **否** |
| 400（路径越权、已存在、参数缺失） | `backend_error` | 是（但重试无用，属调用方错误） |
| deadline | `timeout` | 是 |

私有笔记**不进** `index/all.md`，也不发 `wiki:write` SSE 事件 —— 它不是团队目录内容。

自检（token 见 `%TEMP%\wuphf-broker-token-7892`）：

```bash
curl -s -H "Authorization: Bearer $TOKEN" 127.0.0.1:7892/notebook/catalog
curl -s -H "Authorization: Bearer $TOKEN" "127.0.0.1:7892/notebook/search?slug=cos&q=beet"
```

## 7. skill 归档（`POST /skills/{name}/archive`）

**归档是文件级事实，不是内存态**：handler 会把 `wiki/team/skills/{name}.md` 的
`metadata.wuphf.status` 重写成 `archived` 并提交 git；重启时 scanner 以该字段为准
（`internal/team/skill_scanner.go:512`，空值按 `active`）。

因此归档后必须验一次落盘：

```bash
grep -n "status:" ~/.wuphf/wiki/team/skills/<name>.md   # 必须是 archived
git -C ~/.wuphf/wiki log --oneline -1 -- team/skills/<name>.md
```

**反面教材（2026-11-01）**：10-31 的归档只改了内存/state，wiki 重写没落地
（文件 mtime 仍停在 10-29、frontmatter 仍是 `active`）。broker 重启后 scanner 按文件
重新导入 → skill 复活成 `active`，而 drift 检查只看 job 不看 skill 状态，所以没人发现。
重做一次归档后 frontmatter 变为 `archived`、提交 `0062abe`，才是可重启的。

## 8. `/health` 回显字段（巡检别猜字段名）

```bash
curl -s http://127.0.0.1:7892/health   # 不需要 token
```

| 想确认 | 字段 | 备注 |
|---|---|---|
| 模型是否进了进程 | `provider_model` | 回显 `WUPHF_OPENCODE_MODEL`，正常值 `opencode-go/beam-free`。**响应里没有 `model` 字段**，按 `model` 取只会得到 `null`，别据此判定"模型没传进去" |
| 进程号 | — | **响应里没有 `pid`**；查进程用 `Get-CimInstance Win32_Process -Filter 'ProcessId=<n>'` 看 `CommandLine` |
| 审阅者 | — | 环境变量 `WUPHF_TASK_REVIEWER`（bot slug，或 `off|none|disabled` 关闭审阅回合）；不设则走 lead → 首个非作者成员。**不回显在 `/health`** |
| 其余 | `status` / `provider` / `focus_mode` / `dispatch` / `memory_backend*` / `build` / `session_mode` / `one_on_one_agent` | `build.version=dev` 表示这是本地 `go build` 产物 |

日志重定向是否生效，不看 `/health`：看 `~/.wuphf/logs/broker.err.log` 里有没有**本次启动之后**的新行（`Start-Process` 的重定向是覆盖不是追加，重启后文件会缩到几行）。

## 9. 准备包（A4 会议 / N8 候选人）：建单 → playbook → 机械判据

一件活拆成三件可替换的件，一条链串起来。**两种用途共用同一条链与同一个判据脚本**，
只有目录、四节标题与文案不同——第二个 checker 必然与第一个漂移，所以不建。

| 件 | 位置 | 作用 |
|---|---|---|
| 建单脚本 | `scripts/office/new_prep_task.py` | `--who <对象> --date <YYYY-MM-DD> [--kind meeting\|candidate] [--owner bot] [--topic 主题] [--approve]`。路径只算一次，同时用于 `artifact_path`、成功标准文字、校验 spec——手写三次就会漂 |
| playbook 源 | wiki `team/playbooks/meeting-prep.md`、`team/playbooks/candidate-prep.md` | 四节写法与判据。标准 wiki 写入后自动编译成 `team/playbooks/.compiled/<name>/SKILL.md`，`/skills` 里出现同名 skill |
| 判据脚本 | `scripts/office/check_prep_pack.py` | `command` kind 的 spec；`--self-test` 内置 10 个正反样例（含候选人） |

`--kind` 的差异只有四处：

| | `meeting`（默认） | `candidate` |
|---|---|---|
| 产物目录 | `team/prep/` | `team/candidates/` |
| 四节 | 对方近况 / 我方既有结论 / 要问的问题 / 如果他说… | 候选人背景 / 与岗位的匹配 / 要问的问题 / 如果他说… |
| playbook | `meeting-prep` | `candidate-prep` |
| 额外的写法约束 | 无语义约束（gate 只管结构） | 简历自述不算出处（写进 playbook 与任务 details） |

**不要直接写 `team/skills/*.md`**：那是编译**输出**目录，`skill_scanner.go` 的 `scanSkippedDirs`
明确跳过它。裸写既不会编译也不会 register，bot 拿不到可调用的 skill。

判据（每条都是拒绝一种空壳，不是风格偏好）：

1. 四个小节齐全：`对方近况` / `我方既有结论` / `要问的问题` / `如果他说…`（各带别名，见脚本 `REQUIRED`）。
2. 「对方近况」每条 bullet 必须**同时**有日期（`YYYY-MM-DD` / `YYYY-MM` / `YYYY年M月`）与出处
   （链接、`出处：`、`来源：`、wiki 路径、`ACMER-` 任务号）。
3. 以「未收录 / 待确认 / 未核实 / 无记录」开头的 bullet **豁免**——诚实标注缺失不算违规；
   但整节都是占位则拒（"什么都没查到"不是准备包）。
4. 「我方既有结论」至少引用一个既有产物（`.md` 路径 / `team/` / `ACMER-` / `review` / `任务`）。
5. 问题 ≥3；分支 ≥1 且带 `→` / `=>` / `then`。

失败时 `verification_result.detail` 逐条列出缺项（缺日期、缺出处、问题不足），bot 按提示补写后重试一次。

### Windows 坑：`command` spec 里的引号（已修）

spec 含引号（Windows 路径必备）时，旧实现 `exec.Command("cmd", "/C", spec)` 把 spec 当一个 argv
重新转义，cmd 再把结果拼到工作目录后面——脚本根本没执行：

```
python: can't open file 'C:\Users\...\agent-scratch\data-keeper\"C:\...\check_prep_pack.py"': [Errno 22] Invalid argument
```

现实现（`task_verification_shell_windows.go`）用 `SysProcAttr.CmdLine = "cmd /C " + spec`，
把 spec 原样交给 cmd。回归测试 `TestVerificationShellKeepsQuotedSpec`——旧实现下 `echo "a b"`
输出 `\"a b\"`，测试 FAIL。**所有**用绝对路径的 `command` 校验都受这个 bug 影响，不止准备包。

### 9.1 实测（2026-11-01）

| 环节 | 证据 |
|---|---|
| playbook 编译 | `POST /wiki/write` 写 `team/playbooks/meeting-prep.md` → 10 秒内出现 `team/playbooks/.compiled/meeting-prep/SKILL.md`；日志 `compiled skill created name=meeting-prep slug=meeting-prep created_by=librarian owner_agents=4`；`GET /skills` 里可查到 `meeting-prep` |
| 建单 | `python scripts/office/new_prep_task.py --who forex-watch --date 2026-11-03 --owner data-keeper --approve` → `OFFICE-2943  team/prep/forex-watch-2026-11-03.md  owner=data-keeper  gate=check_prep_pack.py` |
| 交付 | `ACMER-2920` 产出 `team/prep/fund-invest-2026-11-02.md`（6224 B，四节齐全，「对方近况」6 条真实条目各带日期+出处、1 条「未收录」豁免、5 问、5 分支） |
| 判据（修复前） | `verification_result`: `exit: exit status 2 / python: can't open file '<cwd>\"C:\...\"'` |
| 判据（修复后） | 同一 spec 重跑：`prep ok: 4/4 sections; counterparty 6 dated+sourced (+1 stated-absent); questions 5; branches 5`，`pass: true` |
| bot 的诚实停下 | 交付前 data-keeper 停下来问口径（「判断是否被行情证伪」没有可比记录），把方向层证伪标「未收录」而不是补一个数字——playbook 的反幻觉规则真的生效了，这比产物通过更值钱 |
| 副作用（已修） | 30 分钟一次的 skill-compile cron 把两份准备包当可复用方法，注册成 `fund-invest-2026-11-02` / `forex-watch-2026-11-03` 两个 skill；`team/prep/` 已加进 `scanSkippedDirs`，两个 skill 已 archive（frontmatter `status: archived`，重启不复活），`/skills` 从 11 回到 9 |
| 自动审阅 | done 后生成 `OFFICE-2968`（owner `cos`，标题「审阅：准备包：与 fund-invest 的周度对齐」） |
| 判定为拒（第二份） | `OFFICE-2943`（forex-watch）第一次 complete 被拒：`prep check failed (1): counterparty bullet 6 has no source`——它引用了 `forex_bridge.json`，而判据当时只认 `.md`。同时暴露失败文本在 cp936 控制台被转成乱码，bot 读不到该改哪里 |
| 修正后 | 放宽 `.json` 也算可追溯出处 + stdout 强制 UTF-8；同一交付物重跑 `prep ok: 4/4 sections; counterparty 6 dated+sourced (+1 stated-absent); questions 6; branches 7`，`pass: true`，并生成审阅任务 `OFFICE-2991` |
| 审阅结果 | `team/reviews/ACMER-2920.md`（10086 B）：4 条验收全部通过，但判出 1 条**阻断**——准备包把 `team/drills/review-target-*.md` 的 smoke 审阅结果写成了 fund-invest 的业绩评价，会面时据此发问会跑偏 |
| 结论 | 机检只管结构与可追溯性，**语义正确性由审阅回合兜底**；两者都过才叫交付过关 |

`--self-test` 的 8 个样例覆盖：缺日期、缺出处、问题不足、分支无映射、诚实占位豁免、全占位被拒、空文件。其中「诚实占位豁免」是在真实交付上发现的——第一版把「未收录：持仓明细…」当成断言要求日期与出处，逼着写手编一个日期，与判据 4 的方向相反。

### 9.2 重写一版：三个干净上下文，不投票，只互相补漏

审阅判了阻断之后，重写前先做了三个**互不可见**的独立判断——每个只拿到文件路径，不给任何前序结论：

| 角色 | 问的事 | 实际产出 |
|---|---|---|
| 审阅判定 | 那条阻断对**当前**版本是否仍成立？审阅有没有过度或遗漏？ | 精确到行：4 处必改、4 处审阅没抓到的遗漏（含「`review-target-2` 的作者是 cos，被算成 fund-invest」这条更硬的错）、3 处审阅过度 |
| 事实核对 | 每条陈述与它标注的出处逐条比对 | 结论含糊、没有行号，无法据以修改——**弃用** |
| 重写草稿 | 直接产出改好的全文 | 抓到了上面那条 cos 的错；但它引用了不存在的 `team/index/all.md`，还写了与自身条目矛盾的「五天没有任何记录」 |

裁决规则：**只采纳能引用到行号的结论**。三份里两份被部分否定——多视角的价值不在投票，
在于各自抓到别人漏掉的行；但必须有人合并，否则草稿会把新错误带进交付。

合并后的改动写成文件（`--details-file`），「不得再出现的字面串」变成判据（`--forbid`），
重写任务与首次交付共用同一个 gate。可复制的做法：任何「删掉某句错误表述」的返工，
都把要求锁进 spec，而不是写在 prompt 里靠自觉。

**第二轮（对再审的再审）改了三条做法**，因为第一轮的三人组各有固定失误：

| 现象 | 结论 |
|---|---|
| 「审阅判定」在第二轮**把「少一条行动项」定成阻断**，实际应为一般；把一句祈使语气定成一般，实际应为提示 | 除了问「成不成立」，还要**单独问「严重度定得对不对」**——定级偏差会让 bot 去修不值得修的，或者漏掉真正该修的 |
| 「事实核对」第一轮弃用，第二轮强制**输出一张表（编号 / 原文 / 声称出处 / 出处实文 / 一致?）**，从含糊变成最有产出的一份 | 不是这个角色不行，是**问法不行**：要求逐字引原文 + 一行一件事，它就没法含糊 |
| 「重写草稿」两轮都只按给它的意见改，**从不主动找审阅漏掉的错** | 草稿用来拿改法，事实错误必须由另外两个角色供 |

第二轮实际抓到的、两轮审阅都没发现的：`ACMER-998` 的记录日是 10-25 不是 10-24（把两天并成一天）；
「三次演练都栽在三件套上」过度归纳（`ACMER-2832.md` 全文无「基准」「时段」，三份的验收标准其实完全一样）；
点位证伪题没有基线快照却问「有几个被证伪」，当场无法判定。

### 9.3 spec 是 cmd 命令行，不是 argv：首字符是引号会塌

`verification_kind=command` 的 spec 交给 `cmd` 解析。cmd 有条规则：`/C` 后面那串首尾都是引号时，
两个都剥掉。于是这种 spec 在 `cmd /C <spec>` 下会丢开头的引号，剩下的半截被当路径：

```
"C:\...\python.exe" "C:\...\check_prep_pack.py" "team/prep/x.md" --forbid "..."
```

```
The filename, directory name, or volume label syntax is incorrect.   (exit 1)
```

解释器根本没启动，但 bot 收到的是"判据失败"，会去改交付物——改不出结果。
实现改为 `cmd /S /C "<spec>"`（`task_verification_shell_windows.go`）：`/S` 让 cmd 只剥最外层那一对，
spec 原样到达子进程。首字符不是引号的 spec（`python "..."`）两种写法等效，所以这是纯加固。

### 9.4 候选人简报（N8，2026-11-02 落地）

同一条链的第二个用途：一场面试的简报。命令只多一个 `--kind`：

```bash
python scripts/office/new_prep_task.py --kind candidate --who "张三" --date 2026-11-05 --owner data-keeper
```

产物是 `team/candidates/<slug>-<面试日期>.md`，四节为
`## 候选人背景` / `## 与岗位的匹配` / `## 要问的问题` / `## 如果他说…`。
判据仍是 `check_prep_pack.py`——`REQUIRED` 里给 counterparty 加了「候选人/应聘者」、
给 our_side 加了「岗位/匹配」两个别名，其余五条规则一字未改。

实测（2026-11-02）：建单返回 `OFFICE-3162  team/candidates/entity-ced07fb4-2026-11-05.md`，
`POST /tasks {"action":"cancel"}` 关掉（不 approve 就不会派活）。文件级判据：

```
prep ok: 4/4 sections; counterparty 2 dated+sourced (+0 stated-absent); questions 3; branches 1
```

去掉日期后同一份文件立刻变红（`counterparty bullet 1 has no date`），证明别名不是靠别的标题
碰巧命中。反向验证：把「候选人」从 `REQUIRED` 里删掉，同一份样例变成
`missing section: counterparty`——别名是承重的。

**新的陷阱与其修法**：`team/candidates/` 与 `team/prep/` 形态相同（一次一份、带日期），
不跳过就会被 30 分钟一次的 skill-compile cron 读成可复用方法。
`scanSkippedDirs` 已同时加入 `team/candidates/` 与 `team/style/`，
回归测试 `TestShouldSkipDirPrepDeliverables` 覆盖两者，并断言 `team/styles`（兄弟名）不被误伤。

## 10. 启动开关：`memory_backend` 必须是 markdown，否则 wiki 全挂

`/health` 里的 `memory_backend` 不只是记忆层选择，它还决定 **wiki worker 是否创建**：

- `ensureWikiWorker`（`broker_wiki_lifecycle.go`）第一行就是
  `if config.ResolveMemoryBackend("") != MemoryBackendMarkdown { return }`。
- 因此 `memory_backend=gbrain` 时 `b.wikiWorker` 永远为 nil，所有 `/wiki/write*` 返回
  `503 {"error":"wiki backend is not active"}`；bot 若仍要交付，只能绕开写门禁直接写文件，
  产物落在 `team/prep/...` 却不进 git（`git status` 里是 `??`，不 commit、不索引）。
- `ResolveMemoryBackend` 在**未显式设置**时：`WUPHF_MEMORY_BACKEND` → `config.json` →
  「gbrain 二进制在 PATH 且 embedding 可用就用 gbrain，否则 markdown」。所以同一个二进制、
  同一个命令，换个父进程环境（PATH 里多了 gbrain、或设了 embedding key）就会从 markdown
  变成 gbrain，表现为"重启后 wiki 突然不能写"。

**启动时显式钉住**：

```bash
WUPHF_MEMORY_BACKEND=markdown WUPHF_WIKI_BACKEND=memory \
WUPHF_OPENCODE_MODEL=<model> ./gawkbot-fix.exe --broker-port 7892 --provider opencode --no-open \
  >> ~/.wuphf/logs/broker.log 2>> ~/.wuphf/logs/broker.err.log
```

用一个 `>>` 而不是 `>`：`>` 会把上一次启动的诊断行覆盖掉，而这正是需要留下的东西。

**巡检判据**：`curl -s http://127.0.0.1:7892/health | grep -o '"memory_backend":"[^"]*"'`
必须是 `"markdown"`；`/wiki/list` 返回真实条目（不是 `_No articles yet._`）。

实测（2026-11-01）：未设该变量重启 → `/wiki/write` 503、`/wiki/read` 报
`gbrain call get_page ... initialize: EOF`、`/wiki/list` 空；加 `WUPHF_MEMORY_BACKEND=markdown`
重启后三个端点全部恢复，learnings 与 playbook 写入均返回 commit sha。

## 11. 风格知识库（N1）：草稿差异 → 规则 → 判据

人改 bot 的稿子这件事每天都在发生，但改动本身不留痕。这一节把它变成可累积的资产：
记录差异 → 归纳成规则 → 用判据阻止"记了不归纳"。

| 件 | 位置 | 作用 |
|---|---|---|
| 记录脚本 | `scripts/office/record_style_diff.py` | `--bot <b> --draft <文件> --final <文件> [--kind email\|prep\|brief\|review] [--publish]`。把 diff 追加到 `team/style/diffs/<bot>.jsonl` |
| 判据脚本 | `scripts/office/check_style_digest.py` | `--bot <b> [--since 7d] [--min-diffs 1]`。`command` kind 的 spec |
| playbook | wiki `team/playbooks/style-digest.md` | 归纳口径（什么该写成规则、什么不该）、格式契约、归并而非追加 |

### 11.1 记录（机械部分）

```bash
python scripts/office/record_style_diff.py --bot henji \
  --draft out/henji-draft.md --final out/henji-sent.md --kind email
# new dedf6213f4ed4 bot=henji changed=2 added=3 removed=3 hunks=2 -> team/style/diffs/henji.jsonl
```

一条记录的形状（`sort_keys=True`，一行一条）：

```json
{"id":"dedf6213f4ed4","ts":"2026-11-02T04:00:00Z","bot":"henji","kind":"email",
 "draft":"draft.md","final":"final.md","draft_sha256":"…","final_sha256":"…",
 "changed":2,"added":3,"removed":3,"truncated":false,
 "hunks":[{"op":"replace","at":1,"old":["感谢您的来信。"],"new":["这个问题我来处理。"]}]}
```

三个刻意的设计：

- **`id` 由 bot + 两份内容的 sha 决定**，重复跑同一对文件只会得到 `duplicate`，不会制造第二条"人又改了"的假信号。
- **两份完全相同也记一条 `changed: 0`**。丢掉它，周度视图就变成"从来没审过稿"；记下来，"这次一稿过"才是可见的证据。
- **hunk 有上限**（25 段 / 每段 12 行 / 每行 400 字符，超出标 `truncated`）。整段重写不是措辞规律，原样存进去只会把该学的东西埋掉。

`--publish` 用 `POST /wiki/write`（`mode: replace`）把整份 jsonl 提交进 git；不加就只写工作树，脚本会打印一行提示。

### 11.2 规则文件（人的判断部分）

`team/style/<bot>.md`：

```markdown
---
bot: henji
updated: 2026-11-02
---

# henji 的措辞库

## rule: 开头直接回答，不写「感谢您的来信」
evidence: dedf6213f4ed4

为什么：客户读起来像自动回执。
改法：第一句就回应对方问的那件事。
```

### 11.3 判据（五条，每条拒绝一种空转）

1. `## rule:` 条目至少一条；其他标题直接拒（不是规则就别放进来）。
2. 每条必须有 `evidence:` 行；token 全部要在 jsonl 的 `id` 集合里存在——**找不到就拒**，防止编造出处。
3. 规则按"去掉标点空格后"归一化比较，重复即拒：重复说明在追加而不是归并，文件几周就不可读。
4. frontmatter `bot` 与 `--bot` 不一致即拒（防止把 cos 的规则写进 henji 的库）。
5. **新鲜度**：窗口内（`--since`）有 `changed>0` 的记录时，`updated:` 必须 ≥ 最后一条的日期。记了差异却不归纳，等于没做。
   窗口内没有任何改动时不要求更新——空闲不是故障。

### 11.4 实测（2026-11-02）

| 环节 | 证据 |
|---|---|
| 记录 | `new dedf6213f4ed4 … changed=2 hunks=2`；同一对文件重跑 `duplicate dedf6213f4ed4`，jsonl 仍是 1 行 |
| 相同 → 记录 | `identical <id> changed=0`，jsonl 变 2 行 |
| 截断 | 40 处独立插入 → `changed=25 truncated=True`，`hunks` 不超过上限 |
| 判据通过 | `style ok: bot=henji entries=2 evidence=2 diffs(total=1 window=1) updated=2026-11-02` |
| 判据拒绝（stale） | `style file is stale: updated 2026-01-01 but a diff landed on 2026-11-02` |
| 判据拒绝（假 evidence） | `rule 1 cites unknown diff id(s): d00000000000` |
| 判据拒绝（重复规则） | `rule 3 duplicates rule 1: 开头直接回答，不写「感谢您的来信」` |
| 判据拒绝（非规则标题） | `heading 1 is not a 'rule:' entry: …` |
| 判据拒绝（串档） | `frontmatter bot is 'cos', expected 'henji'` |
| self-test | `record_style_diff.py` 10/10；`check_style_digest.py` 10/10 |

### 11.5 已知局限

- **"什么该写成规则"仍是判断**：脚本只提供证据（同一类改动出现几次），归纳由 bot/人做。playbook 给的口径是"同类 ≥2 次，或单次会导致事实错误"，其余不写。
- **`--publish` 是整份 replace**：两个人同时记录会丢一条。低频人工操作可接受；要并发就得改成 `append_section` 或加锁。
- **draft/final 由人提供**：脚本不猜"哪份是终稿"。接邮箱/IM 抽终稿是下一步，本轮没做。

## 12. Routine 记分卡（N9）：谁应该跑，谁真的跑了

「每天在跑」和「看起来在跑」的区别只能靠数字。这一节给出数字，并说明哪些数字**根本不存在**。

| 件 | 位置 | 作用 |
|---|---|---|
| 采集 | `scripts/office/routine_scorecard.py` | 读 broker `/scheduler` + 每个 slug 的 `/scheduler/{slug}/runs` + `/tasks`，再读 `schtasks /query /fo CSV /v`，写快照 |
| 判据 | `scripts/office/check_routine_scorecard.py` | 快照新鲜度、采集完整性、告警必须为空 |
| 页面 | wiki `team/routines/scorecard-<YYYY-MM-DD>.md` | 一页纸；`--publish` 走 `POST /wiki/write` 提交 |
| 数据 | `~/.wuphf/data/routines/scorecard-<date>.json`、`history.jsonl` | 机读快照与累计观测；wiki 只收 `.md`（`validateArticlePath`），所以这两个不进 git |

### 12.1 三个告警，不是两个

| kind | 何时触发 | 为什么单独一类 |
|---|---|---|
| `overdue` | `next_run` 早于 now − 15 分钟 | **不需要周期**就能判：调度器没把下次时间往前推。cron 型 job（`schedule_expr`）也能判 |
| `stale` | 已知周期，且 `now − last_run > 2×周期` | 需要周期；broker 系统 cron 用 `interval_minutes`，Windows 任务用 `重复: 每` |
| `dormant` | `enabled=false` 且最后运行 ≥ 7 天前 | **不进告警**，单列一节。8 条"115 天前停用"混进告警，等于让两条真告警没人读 |
| `failed` | Windows `Last Result` ∉ {0, 267011, 267009} 且任务 `enabled` | 停用任务的失败码是历史，不是当前问题 |

`267011`（从未运行）与 `267009`（正在运行）不算失败；写成失败会让新装的计划任务第一天就红。

### 12.2 两种"不知道"

- **broker 的 run 历史是环形缓冲**（`schedulerRunHistoryLimit = 20`），重启即空。所以 `runs_seen` 不是窗口内次数，**不能拿来算成功率**；高频 job 会一直显示 20。
- **`last_run` / `next_run` 在很多 job 上是空的**（实测 23 个 broker job 里 7 个系统 cron 全空）。这类 job 在页面上标 `*no data*`，判据不判它坏——第一版把它当 `never ran`，一次刷出 6 条假告警。

### 12.3 实测（2026-11-02）

```
scorecard: 36 jobs, 0 alerts, pending=0 -> team/routines/scorecard-2026-11-02.md
published team/routines/scorecard-2026-11-02.md aa9ccaf
routine ok: 36 jobs, 0 alerts, snapshot 2026-11-02T03:47:09Z
```

36 = 23 个 broker job + 13 个 Windows 任务（`--match 'wuphf|gawkbot|forex'`，含 8 个 115 天前停用的 BlockCell 旧任务）。

判据负例全部验证过：`alerts` 非空 → 逐条列出并退出 1；`generated_at` 改成一个月前 → `snapshot is 771.7h old (limit 26h)`；采集报错 → `collector error`；空快照 → `no observations`。

### 12.4 踩到的坑：`schtasks` 表头是本地化的

`schtasks /query /fo CSV /v` 的**列名随系统语言翻译**（zh-CN 是 `主机名/任务名/下次运行时间/…`），列顺序固定。按列名取值在中文机上得到 0 行，而且**不报错**——第一次运行报"23 jobs, 0 alerts"，看起来一切正常，其实是 13 个 Windows 任务全被静默丢掉。

现在按位置读（`WIN_COL`），并顺手从第 24 列 `重复: 每` 取周期（`0 小时， 15 分钟` → 15），比 `Next Run − Last Run` 更接近真实计划。回归用 `parse_repeat_minutes` 的四个样例（英/中/天/N/A）守住。

**这个坑的通用形状**：一个数据源解析失败却返回空集，比报错更危险。凡是"过滤后为空"的采集，都要能区分"确实没有"和"没解析出来"。

同一类坑还有一处：**从未运行的任务把 `Last Run Time` 报成 `1999/11/30`**。当真日期读，会算出 1400 万分钟的周期，页面上写着"上次运行 1999 年"。判定以 `Last Result=267011` 为准，年份 ≤2000 作为兜底（结果码已丢时仍能识别）。

## 13. 论点台账（A1）：把"我说过什么"变成可判死的数据

简报的价值只在被证伪时兑现。这一节定义怎么把一句方向性判断变成一条能被脚本判分的记录，以及**判不了的时候会发生什么**。

| 件 | 位置 | 作用 |
|---|---|---|
| 录入 | wiki `team/theses/intake.md` | 每条判断一个 ```` ```thesis ```` 围栏块（JSON）。**这是唯一真值**，因为它进 git：有作者、有时间、改一次留一次提交 |
| 校验/判分 | `scripts/office/check_theses.py` | `--lint` 校验口径；`--score` 从价格序列算 verdict；`--report --emit` 发布周页 |
| 派生 | `~/.wuphf/data/theses/ledger.jsonl`、`verdicts.jsonl`、`rejects.jsonl` | 每次运行**全量重建**，不是状态 |
| 观测 | `~/.wuphf/data/forex/snapshots/<YYYY-MM-DD>.jsonl` | `forex-bridge.py` 每次运行追加；没有它就没有"上周收盘"这个事实 |
| 周页 | wiki `team/theses/week-<date>.md` | 给人看的一页 |
| 调度 | 计划任务 `WUPHF-ThesisReview`（周五 21:00 Asia/Shanghai） | `~/.wuphf/data/thesis-week.ps1` |

### 13.1 为什么 md 是源、JSONL 是派生物

本机每个 bot 的 permission 都是 `{bash: deny, edit: allow, external_directory: allow}`（`~/.config/opencode/opencode.*.json`）。把台账只放 `~/.wuphf/data/` 等于让任何 bot 把昨日 miss 改成 hit，而且**没有第二个见证者**——`~/.wuphf` 不是 git 仓库。md 在 wiki 里，改一行就多一次提交，作者与时间都在。

派生物每次全量重建，因此不存在"忘了同步副本"这种故障。N9 今天正好留了个反例：`team/routines/history.jsonl` 在 wiki 树里有一份 11:40 就停止更新的副本，没人删；谁按路径直觉去找就会读到一份看起来完全正常、其实已经死掉的旧数据。

### 13.2 `criterion` 只有一种 kind 能判

```json
{"kind":"level","series":"USDJPY","metric":"close",
 "window":{"start":"2026-11-02","end":"2026-11-06"},
 "agg":"any","op":"gt","value":157.5,
 "require":{"min_obs":3,"stale_days":2}}
```

| 字段 | 作用 | 省略会怎样 |
|---|---|---|
| `agg=any` | 窗口内任一天满足即命中（存在性主张） | — |
| `agg=all` | 每天都满足才命中；**一个反例立即判 miss** | 空子集上的全称命题恒真 → 假命中 |
| `agg=last` | 取窗口末端的观测；离 `window.end` 超过 `stale_days` 判"数据不足" | 拿上周收盘当本周收盘 |
| `require.min_obs` | 下结论前的最小观测数 | 1 天数据就能让 `all` 变成 hit |
| `require.stale_days` | 窗口末端多久没新数据算覆盖不全 | 假期/上游停更被当成"没涨" |

`relative`（A 跑赢 B）与 `event`（事件结果）**在 v1 被明确拒绝**，code 是 `kind_not_scored`，不是静默挂起。理由：一个永远 pending 的 kind 与一个能用的 kind 在页面上长得一样。

**已知局限**：`min_obs` 只保证"观测数够"，不保证"覆盖连续"。`agg=all` 配 `window` 5 天、`min_obs=3` 且窗口内恰好有 3 天数据时，中间缺的两天不会被发现——判据会说"每天都满足"，但那两天其实没被检查过。要修需要交易日历（本机没有），或者把 `min_obs` 强制等于窗口天数（会把周末/假期判成无数据）。第一版的取舍是：**报告里对 `agg=all` 的条目同时打印实际观测数与窗口天数**，让缺口可见而不是被 cosh 掉。

### 13.3 判定顺序（先失效条件，再数据，最后覆盖）

1. `today < review_date` → pending（不写盘，每次重算）
2. 窗口未闭 → pending
3. **先找决定性见证**：`any` 找到一真即 hit；`all` 找到一个假即 miss
4. 没有决定性见证时才看覆盖：观测数不足或数据太旧 → 宽限期内 pending，过期 `void(no_data)`
5. 仍无定论 → 按 `agg` 收口

`void` 与 `miss` 必须分开：把"判不出来"记成"我错了"会让人再也不敢写可证伪的判断，那是这个台账的反向目标。

### 13.4 三条红线（任一触发即 exit 1）

1. **空台账**：`intake.md` 里一条都没有，且**首次 git 提交距今超过 14 天**（刚建当天不算）。为什么不用 mtime：wiki watcher 遇到外部编辑会重写并重新提交，mtime 被刷成"现在"，于是三周没动的空页看起来像今天刚建。
2. **超期未判**：条目过了 `review_date + 2d` 仍是 pending。这条实际抓的是"判分脚本没跑"——`--score` 从没执行过时，verdicts 为空，全部显示 pending。
3. **采集静默失败**：`intake.md` 里有 ```` ```thesis ```` 标记但解析出 0 个块。这与 §12.4 是同一个形状：过滤后为空却返回空集，比报错更危险。

`--lint` 另有一条：任何被拒的录入都会 exit 1，且拒绝理由出现在周页的"被拒的录入"里。写不出 `series + op + value + window` 的判断不该进台账，但**它被拒这件事本身要留痕**。

### 13.5 观测从哪来

`forex-bridge.py` 每次运行把日线写进 `~/.wuphf/data/forex/snapshots/history.jsonl`，按 `(pair, as_of)` 去重后全量重写。

**桥能回补**：`GET_HISTORICAL_DATA:{pair},1440,{n}` 一次返回任意根数，所以第一次运行就写了 **300 行 = 5 个货币对 × 60 个交易日**（2026-08-11 起），不需要一天天积累。实测三类聚合都能判：

```
窗口 2026-10-26..2026-10-30
  agg=all  gt 155  → hit  （5 条收盘 157.487/157.943/157.831/157.703/157.561 全满足）
  agg=any  gt 160  → miss （无一日上破）
  agg=last gt 157  → hit  （末日 157.561）
```

**一个会让人误判的语义**：`pivots.<pair>.date` 取的是 `bars[-2]`，即上**一个完整**交易日的 HLC。周一早上它显示上周五的日期，这不是桥停了，是枢轴本来的含义。判断桥是否活着要看 `history.jsonl` 的 `captured_at`，别看 pivot 日期。

### 13.6 实测（2026-11-02）

```
python scripts/office/check_theses.py --self-test     # 23 checks, 0 failed
python scripts/office/check_theses.py --score         # theses: 0 entries ... (0 day(s) old, inside the 14-day grace)
python scripts/office/check_theses.py --report --emit # team/theses/week-2026-11-02.md 6c8f1bc
```

端到端（临时根 + 5 天收盘序列）：同窗口两条相反判断 → `hit=1 miss=1`。红例全部验证过：空台账且首次提交在 20 天前 → exit 1；条目超期未判 → exit 1；坏 JSON / 拼错键 / 字符串阈值 / `review_date` 早于窗口末端 / 重复口径 / `relative` → 各返回对应 code 且不入账。

`scanSkippedDirs` 已加 `team/theses/`：一张周页是某一周的记录，不是可复用方法；不加就会给每周生成一个以日期命名的 skill（`team/prep/`、`team/routines/` 各踩过一次）。

### 13.7 实际值只取一手来源（P2 #5 / B，2026-11-08）

FF 的周历 XML **根本没有 `<actual>` 节点**（2026-11-08 实测 0 个），所以过去每一处「实际值」都是 bot 从新闻标题里拼的——正是单位混用的来源。现在桥自己算：

| FF 事件（country + title） | 一手来源 | 序列 | 2026-09 核对 |
|---|---|---|---|
| USD `CPI m/m` | BLS CPI-U 全项（SA） | `CUSR0000SA0` | 0.6%（发布文本 "increased 0.6 percent"，`cpi.t01` Aug→Sept 列 0.6） |
| USD `Core CPI m/m` | BLS CPI-U 除食品能源（SA） | `CUSR0000SA0L1E` | 0.3%（"less food and energy rose 0.3 percent"，表 0.3） |
| USD `PPI m/m` | BLS 最终需求 PPI（SA） | `WPSFD4` | 1.3%（"final demand increased 1.3 percent"，`ppi.t01` FD 4 末列 1.3） |
| USD `Core PPI m/m` | BLS 最终需求除食品能源（SA） | `WPSFD49104` | 0.7%（`ppi.t01` FD 49104 末列 0.7） |

- **算出来的数，不是抄来的数**：BLS v1 公开 API（无需 key，25 序列/天/IP）给指数水平，桥取最近两个月算 `(cur/prev-1)*100` 并 `ROUND_HALF_UP` 到 0.1——与发布表一致。**序列 ID 会咬人**：`WPSFD4131` 不是「最终需求除食品能源」，它算出来 0.2%，而发布表是 0.7%；正确的是 `WPSFD49104`。这条差异就是必须拿发布表核的原因。
- **缓存 6h**（`~/.wuphf/data/bls_actuals.json`）：桥每 6h 跑一次，BLS 每天只给 25 次/序列；取不到就退回上一份缓存，再取不到就整段不写实际值。
- **`actual` / `actual_period` / `actual_source` / `actual_is_event`** 挂在 `forex_bridge.json` 的 `calendar.events[]` 上。`actual_is_event` 为真只有当事件已过**且**其覆盖月份正是已发布月份；否则 digest 写「上次实际」，绝不把上个月的数当今天的。
- **digest 与简报**都写：digest 顶部一行汇总 + 每个事件行内 `实际/上次实际 X（period，BLS）`；简报 `## 四、实际值（一手来源）与预期差` 在事件公布后自动算 `实际 - 预期`（`<x` 这类带比较符的值跳过，不猜）。
- **口径比对**：事件行会带 `与 FF 前值口径差 ±x pp`，只在非零时出现。2026-11-08 四个序列全部 0.0（FF 前值与一手来源一致），这也是选择 `WPSFD49104` 后的验证。
- 未接来源的事件（GBP GDP、USD Claims、央行讲话）**不写实际值**：digest 尾注要求写「实际值未收录」，禁止凭记忆补写。


## 14. skill-compile 的判定成本：为什么告警日志比告警本身贵

30 分钟一次的 `skill-compile` cron（`internal/team/skill_compile.go`）把 `team/**/*.md` 逐篇送给 LLM 判「是不是可复用技能」。**每篇一次独立调用**，每篇输入 = `skill-creator.md` 系统提示 + 正文 + 现有技能清单（`buildSkillUserPrompt`，`skill_scanner.go:759`）。判否时输出只有 7–14 token，输入 13k–28k。

去重靠内容 SHA（`SkillScanner.mtimeCache`）：正文不变就跳过。所以「判定成本」完全由**正文是否在变**决定。

### 14.1 2026-11-06 实测：一篇日志吃掉当天 40% 的 input token

| 指标 | 数值（opencode.db，本地 5790 个 session） |
|---|---|
| 全部 skillify 判定 | 1473 次，input 19.9M / output 37k / reasoning 201k |
| `team/alerts.md` 单篇 | **152 次，input 3.13M**，每次判定结果 `{"is_skill": false}` |
| 当天（11-06）全部 input | 2.73M，其中 skillify 1.08M，其中 alerts.md **1.084M（99.96%）** |
| 单次 input 增长 | 13,052 → 28,631（+~110 token/次，正文 +180 字符/30 分钟） |

原因链：`check-office-drift.ps1` 的 scheduled_tasks 循环没读 `enabled:false` → 对一份**有意停更**的日志每 15 分钟报「停更」→ 告警文本里的年龄数字每轮都变（26.1h → 100.1h）→ 内容哈希永不命中 → `alerts.md` 每 15 分钟长 90 字符 → 下一篇判定必然 cache miss，且正文越来越长。301 条告警、36KB，全部是同一句话。

### 14.2 三处修复（2026-11-06）

1. **enabled 门控**：`scheduled_tasks` 循环开头 `if ($t.enabled -eq $false) { continue }`。停用的任务日志本来就不该新鲜；对它查新鲜度只会产生一条永远为真的告警。broker 那半边的循环一直有这道门（`if ($e.enabled -and ...)`），scheduled task 这半边漏了。
2. **去重键规范化**（`ConvertTo-DedupeKey`）：alerts 正文保留可读的全量信息（年龄、构建时间、几个提交），**哈希只取故障身份** —— 任务名/字段/类别。剔除：时长（`26.1h`）、超时小时数、提交列表（`bb3aedc, 1a2b3c4`）、时间戳、临时目录名（每次进程都不同）、`+N` 文件版本戳。排序用 `StringComparer::Ordinal`：文化排序会让哈希随系统区域设置变，区域一换所有存量故障重报一次。
3. **判定缓存落盘**（`skill_scan_cache_file`，`skill_scanner.go`）：`mtimeCache` 原本只在内存，broker 重启（watchdog、重建）就把整棵树重判一遍，约 1.4M token/次。现在写到 `<wiki>/.git/skills-scan-cache.json`（跟着 wiki 走、不会被下一轮扫描当成文章），重启用内容哈希命中即可跳过。带 `version` 字段：格式变更时丢弃而不是误读。作用域扫描（`scope_path`）只覆盖自己的子树，其余条目原样带过，不会把全树缓存抹掉。

### 14.3 验证

```
pwsh -File scripts/office/check-office-drift.ps1                  # DRIFT: none
# 反向：清单里把 WUPHF-ModelSnapshot 改回 enabled:true
  → DRIFT: 1  计划任务停更: WUPHF-ModelSnapshot 日志 100.4h 未更新（阈值 26h）
# 去重：同一故障连跑两次（年龄不同）
  第一次 ALERT: appended / 第二次 ALERT: unchanged since last run (deduped)
# 新故障：同一次运行里加一条日志缺失
  → ALERT: appended（键集变了，就该再报一次）
```

`go test ./internal/team -run TestSkillScanner_VerdictCache`：重启后 0 次 LLM 调用、改一篇文章只重判一篇。把 `loadSkillScanCache` 注释掉后该测试失败（`got 2`），证明它测的是缓存而不是「扫描本来就没跑」。

### 14.4 剩下的坑

- 任何**新**的长期故障如果文本里带单调数字，仍会绕过规范化（正则只覆盖已知形状）。新增检查项时，先问「这行文字下个小时会不会不一样」。
- `alerts.md` 已从 301 条合并为 1 条 + 说明（原文留在 `~/.wuphf/backups/alerts.md.pre-cleanup-20261106`）。合并是一次性清理，不是自动轮转 —— 没有上限，只是不再每 15 分钟增长。

### 14.5 判定结果解码（2026-11-06，同日第二个修复）

14.3 的验证通过之后，日志里仍然每篇一次 `parse llm json: invalid character '{' after top-level value`。查下去不是模型坏，是**读错了管子**：

```
opencode run --format json   # internal/provider/opencode.go:236 固定加这个 flag
  → {"type":"step_start",...}
  → {"type":"tool_use",...}          # 模型把「这是不是可复用技能」当成 agent 任务，去 glob/bash 翻仓库
  → {"type":"text","part":{"type":"text","text":"{\"is_skill\": false}"}}
  → {"type":"step_finish",...}
```

`runOpencodeCommand` 把这些行拼成一个字符串，`json.Unmarshal` 说「顶层值之后还有 `{`」。**自 `--format json` 引入起，一次判定都没解析成功过**：每个 pass 付 50 次调用，50 个结论全丢。

真正的危害在缓存上线之后：解析失败返回 `(false, nil)`，与「模型说不是」不可区分，于是 79 篇文章的 `is_skill:false` 被当作结论写进缓存。文章不改内容就永远不会重判——编译功能看似健康，实际已死。

修法（`internal/team/skill_scanner.go`）：

1. `parseSkillJSONFull` 三级解码：整串直接解 → `skillAnswerFromEventStream`（逐行解事件、拼 `part.text`）→ `skillVerdictCandidates`（括号配平扫出所有候选，取**最后一个**且**顶层真有 `is_skill` 键**的）。
2. `unmarshalSkillVerdict` 先解成 key map 再解结构体：缺 `is_skill` 键就报错。否则一个「内部引用了判定 JSON」的外层事件会被解成全零值，读成 `is_skill:false`，比解析失败更危险。
3. **没有结论就不许缓存**：解析失败与调用超时都返回 error，`Scan` 因此删掉该路径的缓存条目，下一个 pass 重试。WARN 里带 `response_head`（200 字符），provider 再换形状时一眼能认出来。

验证：

```
go test ./internal/team -run 'TestParseSkillJSON|TestLastSkillVerdictObject|TestSkillScanner'
# 真实 CLI 输出（含 tool_use 步骤）喂给解码器 → is_skill=false，无 error
# 只有 tool_use 没有 text 的那一类输出 → 报错，且缓存里没有该文章
```

剩下的坑（本次未修，属成本问题不属正确性）：`--pure` 只关插件，**不关工具**。判定请求跑在仓库 cwd 里，模型会自己 glob/bash 翻代码，单次从 7.7s/13k token 涨到 33s/25k，偶尔整轮没有任何 text 输出。opencode 没有 `--no-tools`；只能用自定义 agent（`agent.<name>.tools` 全 false + `--agent`），实测 0 次 tool_use、7.7s。要接的话得让运行时那边的 opencode 配置带上这个 agent，属于另一件事。

### 14.6 判定改走 HTTP（2026-11-06 夜）

14.5 之后判定能读懂了，但每次仍在起一个 **agent**：`opencode run --pure` 只关插件，不关工具，模型拿到 repo cwd 就会 `glob`/`bash`/`read` 自己找线索。实测同一条判定提示：

| 路径 | 延迟 | input tokens | 稳定性 |
|---|---|---|---|
| opencode CLI（工具开） | 8–33 s，最长一次 8 min | 11k–25k | 有时整轮只有 tool_use、没有 text → 无结论 |
| opencode CLI（自定义 agent 关掉全部工具） | 7.7 s | 13k | 稳定 |
| **DeepSeek `/chat/completions`** | **0.6–0.8 s** | **1.6k** | 稳定（temperature 0） |

根因在 `provider.RunConfiguredOneShotCtx`：它按**活动 provider** 派发，而只有 claude/codex/opencode 实现了 one-shot。活动 provider 是 HTTP 类（ollama/mlx-lm/exo/hermes/openclaw-http）时，一次「问一句话」会**静默退回 opencode CLI**，也就是起一个带工具的编码 agent。

改法（两层）：

1. `internal/provider/oneshot_http.go`：`RunOneShotHTTPCtx` —— 纯 `/chat/completions` POST，无 tools、无 stream、temperature 0；base URL 与完整 endpoint 都接受（不会拼出 `/chat/completions/chat/completions`）。`OneShotHTTPConfigFor(kind)` 复用既有的 env > config > 默认解析链。
2. `skill_scanner.go`：有判定端点就走 HTTP，没有才回落到 provider one-shot。配置：
   - `WUPHF_SKILL_JUDGE_PROVIDER=<kind>`（复用该 kind 的 base_url/model/api_key），或
   - `WUPHF_SKILL_JUDGE_BASE_URL` + `WUPHF_SKILL_JUDGE_MODEL` + `WUPHF_SKILL_JUDGE_API_KEY`。

本机落在 `~/.wuphf/data/broker-watchdog.ps1`（`WUPHF_SKILL_JUDGE_BASE_URL=https://api.deepseek.com/v1`、`MODEL=deepseek-chat`、`KEY=$env:DEEPSEEK_API_KEY`），备份 `broker-watchdog.ps1.bak-judge`。

验证（新二进制、broker PID 67412）：

```
POST /skills/compile {"dry_run":true,"scope_path":"team/people"}
→ {"scanned":5,...,"duration_ms":3403}      # 5 篇 3.4s, 每篇 ~0.68s
→ 同期无 opencode.exe 进程被拉起（最近 10 分钟 0 个）
→ 缓存 version 2, 条目 3 + 5 = 8，无任何 skill_scanner 告警
```

同一批文章走 CLI 是 13–33 s/篇（合计 66–165 s），且修复前每次都留一条解析失败告警。成本侧：每篇约 1.6k input + 7 output，50 篇的 pass 约 0.01 美元量级；相比 13k–25k 的 agent 输入，降低约一个数量级。

当时没做（2026-11-07 补上了，见 §14.12）：`RunConfiguredOneShotCtx` 仍保留「HTTP provider → 回落 opencode」的行为，只给判定加了显式端点。

### 14.7 判定端点换到 8848 的 qwen（2026-11-06 深夜）

14.6 先用公网 DeepSeek 验通了 HTTP 判定，随后按用户要求改到局域网 8848 网关（`bear` provider，`http://192.168.192.118:8848/v1`）。同一个真实判定提示的实测：

| 模型 | 延迟 | input/out tokens | 输出 |
|---|---|---|---|
| **`qwen3.5-122b`** | **1.2–1.6 s** | 1687 / 7 | `{"is_skill": false}`，精确 |
| `Qwen3-Instruct` | 1.24 s | 1687 / 7 | 同上 |
| `qwen3.6` | 2.9 s | 1723 / 187 | 正确，但带 reasoning tokens |
| `deepseek-flash` | 0.86 s | 20 / 8 | 同上（控制组，短提示） |

`deepseek-v4-flash`、`gemini-3.1-flash-lite` 在该 token 的 group 下返回 `503 model_not_found`（没有可用渠道），`gpt-5.4` 返回 401 —— 网关按 token group 授权，不是所有列出的模型都能用。

**踩到的坑**：`~/.local/share/opencode/auth.json` 里的 key 带尾随 CR（文件是 CRLF），直接塞进 `Authorization` 会被网关判 400 `Bad Request`（没有任何 error body，很容易误判成端点不通）。现在两处都处理：watchdog 里 `.Trim()`，`RunOneShotHTTPCtx` 里 `strings.TrimSpace`，并有测试锁住（`TestRunOneShotHTTPCtx_TrimsAPIKey`）。

配置落在 `~/.wuphf/data/broker-watchdog.ps1`：`BASE_URL=http://192.168.192.118:8848/v1`、`MODEL=qwen3.5-122b`、`KEY` 从 auth.json 现读（读失败回落 `DEEPSEEK_API_KEY`），备份 `broker-watchdog.ps1.bak-judge`。

验证（broker PID 12892）：

```
POST /skills/compile {"dry_run":true,"scope_path":"team/reviews"}
→ {"scanned":7,...,"duration_ms":11076}      # 7 篇 11.1s, 每篇 ~1.6s
→ 同期 opencode.exe 新进程 0 个
→ 缓存 version 2, 条目 8 → 15
→ skill_scanner 告警 0 条
```

代价：判定现在依赖 192.168.192.118:8848 可达。网关不通时判定报错、不写缓存，下个 pass 重试 —— 不会像旧的解析失败那样把结果记成「不是技能」。要双活的话得给 `skillJudgeHTTPConfig` 加一组备用端点，本次没做。

### 14.8 判定模型换成 8848 的 `openrouter/free`（2026-11-07 00:30）

`bear` 网关（`192.168.192.118:8848/v1`）上 `openrouter/free` 可用，cost=0。同一个真实判定提示的实测：

| 模型 | 延迟 | in/out | 判定 |
|---|---|---|---|
| `qwen3.5-122b` | 1.2 s | 1687 / 7 | `{"is_skill": false}` |
| **`openrouter/free`** | **7–120 s**（同一提示方差极大） | 1479–2505 / 106–810（含 reasoning） | 同样精确，但会给出 `enhance` 建议 |

关键差别不在速度，在**结论**：拿 7 篇 `team/reviews/` 复跑（带真实的 EXISTING SKILLS 块）：

| 文章 | qwen3.5-122b | openrouter/free |
|---|---|---|
| ACMER-2847 | false | **true + enhance=`forex-watch-2026-11-03`** |
| OFFICE-3012 | false | **true + enhance=`fund-invest-2026-11-02`**（这次调用 120 s） |
| 其余 5 篇 | false | false |

也就是说 qwen 把「这篇 review 里有对现有技能的改进意见」全判成了 false；`openrouter/free` 认出来了，走 enhance 路径。broker 自己的 dry-run pass 复现同一结果：`{"scanned":7,"matched":2,"proposed":2,"duration_ms":148123}`（唯一一次非零 matched）。

配置（`~/.wuphf/data/broker-watchdog.ps1`）：`MODEL='openrouter/free'`，并把 `WUPHF_SKILL_LLM_TIMEOUT` 从默认 30 提到 **180** —— 观测到的长尾是 120 s，30 s 会瞬杀掉约三分之一的树，而超时等于「无结论」，会被每轮重试。

**缓存现状**：`skills-scan-cache.json` 里 72 条是 qwen 时代的结论、7 条是 `openrouter/free`（v2 版本号不区分模型）。要拿到单一模型的基线就得整体重判（budget 50/轮 → 两轮，约 20–30 分钟）；但 cron 的 pass 不是 dry-run，重判会**真的**把 `forex-watch-2026-11-03`、`fund-invest-2026-11-02` 两篇技能改写（追加 “Quality checklist / Enhancements from OFFICE-3012” 段）。所以没有自动做，等操作者决定。

另：`team/skills/` 下是**扁平的 `*.md`**，不是目录 —— 之前用 `ls -d */` 数出 0 是错的，实际有 14 个技能，`buildExistingSkillsSummary` 的 2 KB 上限基本用满（实测块 2,051 字节）。

### 14.9 判定前置决策模型（gate），2026-11-07

`openrouter/free` 能认出 enhance 信号，但延迟 7–120 s，靠 180 s 超时兜底，超时即「无结论」→ 每轮重试。判定的第一步其实是**一个 yes/no**，不需要通用文本模型。OpenRouter 的 Decisions API 正是这个形状（`POST /api/alpha/decisions`，`state` + 类型化 questions → `noul` 概率 / `choice` / `score`，**不产生自由文本**）。

**离线评测**（11 篇技能来源文章 `source_articles` 作正例 + 19 篇 review/公司/索引作负例，同一份 state 与问题）：

| 模型 | 延迟中位 / p90 | input tokens | 正例均 p | 负例均 p | AUC | $/call |
|---|---|---|---|---|---|---|
| **perplexity/pplx-decider-flash-v1.2** | **0.86 / 1.02 s** | 3089 | 0.92 | 0.09 | **1.00** | 0.000054 |
| cloudflare/clef-flash | 1.08 / 1.45 s | 1696 | 0.73 | 0.15 | 1.00 | 0.000069 |
| microsoft/microsoft-decision-1 | 1.22 / 1.50 s | 1553 | 0.77 | 0.03 | 1.00 | 0.000114 |
| liquid/d1.1 | 0.96 / 1.36 s | 1735 | 0.64 | 0.04 | 0.98 | 0.000056 |
| typesafe/jev-1.14 | 0.71 / 0.82 s | 2212 | 0.68 | 0.10 | 0.99 | 0.000148 |
| baseten/joey-1 | 0.79 / 0.95 s | 1599 | 0.41 | 0.31 | 0.70 | 0.000023 |

选 `pplx-decider-flash-v1.2`：11 篇正例全部 ≥0.61（9 篇 ≥0.90），19 篇负例里 18 篇 ≤0.22（唯一例外是 `opencode-models-2026-11-02.md` 0.71 —— 机器生成的模型快照，本该进跳过列表）。

**接入**（`skill_scanner.go`）：

| 概率 | 行为 |
|---|---|
| ≥ 0.50 | 交给文本模型写 `name`/`description`/`body` |
| 0.30 – 0.50 | **不判**：返回 error，不写缓存，下轮再问。概率 0.4 不是「否」 |
| < 0.30 | 判定为不是技能，缓存结论 |

gate 有自己的 30 s 超时（文本路径的 180 s 是给生成用的）。`RunDecisionsCtx`（`internal/provider/decisions.go`）按文档要求做类型检查：`answers.<id>.type` 不是 `noul`、或 `noul` 缺失，一律报错而不是取零值——否则「拿不准」会变成「否」。

配置（`~/.wuphf/data/broker-watchdog.ps1`）：`WUPHF_SKILL_GATE_MODEL='perplexity/pplx-decider-flash-v1.2'`，key 取用户级 `OPENROUTER_API_KEY`。不设 `WUPHF_SKILL_GATE_MODEL` 即完全关闭 gate，行为退回 14.8。

**实跑**（broker 重启后、缓存里 `team/companies` 已剔除）：

```
POST /skills/compile {"dry_run":true,"scope_path":"team/companies"}
→ {"scanned":20,...,"duration_ms":10239}      # 20 篇 10.2s, 每篇 0.51s，全部被 gate 拒
→ matched 0：没有任何一篇花费文本调用
```

同一批走 `openrouter/free` 是 20 × ~10 s ≈ 200 s，且历史上其中 2 篇会撞 30 s 超时。

**运维注意**：`mtimeCache` 是进程内状态，首次 Scan 时从磁盘加载。手工改 `skills-scan-cache.json` 后**必须重启 broker** 才会生效（第二次 Scan 起只用内存）。

**读数口径（2026-11-08 复核）**：`matched=0` 只表示"没有一篇 p≥0.50 被送去写正文"，**不等于"全部被拒"**。同一棵 75 篇的树在主办公室实测：

```
POST /skills/compile {"dry_run":true}
→ {"scanned":75,"matched":0,"proposed":0,"rejected_by_guard":0,"duration_ms":2406}
→ errors: 3 条 uncertain（p=0.40 / 0.41 / 0.34，全在 0.30–0.50 的"不判"带）
```

- **72 篇是缓存命中**（`wiki/.git/skills-scan-cache.json` 正好 72 条），根本没调用模型；真跑 gate 的只有 3 篇。热缓存一轮 ≈2.4 s，冷跑才是 75 × 0.50 s ≈ 37.8 s —— 两种耗时都正常，别用它反推"有多少篇真判过"。
- 这 72 条是 **14.8 时代 qwen 文本判定**的结论（cache `version: 2`，不记模型）。11 篇 `source_articles` 正例里 `candidate-prep`、`meeting-prep`、`style-digest`、`task`、`smoke-clean-context-review-lane` 全在缓存里被标成"非技能"——**新 gate 从未复判过它们**。所以热缓存下的 `matched=0` 不能当 gate 的质量证据；要基线就先备份缓存、删条目、重启 broker 再跑（75 次调用 ≈40 s，约 $0.004）。
- `rejected_by_guard` 是内容守卫的计数器，与 gate 的拒绝无关，不要混读。
- 3 篇 drills/core-loop 文章长期停在 0.30–0.50，每轮重试（≈1.5 s + $0.00016/轮），不会自行收敛成是/否。

### 14.10 生成物移出扫描范围（2026-11-07）

`team/opencode-models-YYYY-MM-DD.md` 是 `model-snapshot.ps1` 每天写的模型清单（同样的 460 个 id 加一张变动表）。它不是流程，而且内容每天变，所以**每天都会被判一次**——它也是 14.9 校准里唯一的假阳性（p=0.71，其余非技能都 ≤0.22）。

`shouldSkipPath` 原先是目录前缀匹配，抓不到带日期的文件名，于是新增 `scanSkippedFilePrefixes`（文件前缀 `team/opencode-models-`），语义与目录表一致、同样带注释说明原因。

验证（broker 60344）：

```
POST /skills/compile {"dry_run":true}
→ {"scanned":74,...}        # 之前 80；恰好少了 6 篇 opencode-models-*.md
→ cache 73 条，含 opencode-models-* 的条目 0 条
→ 唯一未判的是 team/playbooks/forex-watch-2026-11-03.md（gate p=0.43，按设计每轮重试）
```

`team/alerts.md` 与 `team/index.md` 同样是机器维护、每轮都变的页面，但**没有**加进去：它们各自还有被人手写内容覆盖的可能，而模型快照是纯生成物。要一起排除的话，往 `scanSkippedFilePrefixes` 追加两行即可。

### 14.11 判定模型换成单一模型 crouton-alpha-free（2026-11-07）

14.8 起文本判定用的是 8848 网关的 `openrouter/free`。**它不是模型，是免费池的路由**：同一句「Say OK」连打四次，回包里的 `model` 字段分别是 `liquid/lfm-2.5-2.6b:free`、`recursive/flint-v1.0:free`、`cohere/north-mini-code:free`、`cohere/north-mini-code:free`。30 篇文章的评测里一共命中了 **9 个不同的后端**。

对一个会把结论写进持久缓存的分类器，这是错的形状：同一篇文章在不同轮次被不同的模型判，延迟在 2–22 s 之间摆（p90 21.8 s），`is_skill` 的边界取决于这次抽到哪个 2.6B。实测里 `team/playbooks/task.md` 与 `team/playbooks/smoke-review-lane-with-wake.md` 正是被 `liquid/lfm-2.5-2.6b` 判成「是技能」的。

**候选（同一套 30 篇文章、同一份 skill-creator 提示词、temperature 0）：**

| 模型 | 通路 | 延迟中位 | 输出 tokens | 严格 JSON | 结论 |
|---|---|---|---|---|---|
| **opencode/crouton-alpha-free** | opencode 推理 API | **3.7 s**（p90 13.1） | 否=7、是≈750 | 是 | 采用 |
| nvidia/nemotron-3-ultra-550b-a55b:free | 8848 网关 | 10.8 s | 107–1758 | 是 | 慢，且把一个正例判否 |
| openrouter/free | 8848 网关 | 5.6 s（p90 21.8） | 61–3685 | 是 | **路由，不可复现** |
| beam-free | opencode 推理 API | 31 s | 3000–4300（推理） | 是 | 同一篇文章中英文输出随机切换 |
| iceman-contributor-free | opencode 推理 API | 26 s | ~1300 | 是 | 慢 |
| deepseek-v4-flash / minimax-m3 / qwen3.8-27b `:free` | 8848 网关 | — | — | — | 网关列出但无上游通道，404 |

`opencode models` 现在有 83 个 `opencode/*`（10 月快照是 28 个），其中 10 个带 `free`；opencode 推理 API 的目录里只有 3 个（`beam-free`、`iceman-contributor-free`、`crouton-alpha-free`）。

**判定质量**（11 篇技能来源文章作正例 + 19 篇非技能，正例里含 8 篇一次性产物：日期化的 watch/invest 包、drill 包、smoke 产物、数据摘要）：

| | TP | FN | TN | FP |
|---|---|---|---|---|
| crouton-alpha-free | **3** | 8 | 19 | 0 |
| openrouter/free | 5 | 6 | 19 | 0 |

正例里通过的三篇——`candidate-prep`、`meeting-prep`、`style-digest`——都有 `Purpose` / `When to use` 段落；被拒的 8 篇是日期化产物与 smoke 产物，正是 14.4 的跳过列表要挡的东西。也就是说 crouton 的边界与 `team/skills` 自己的边界一致，`openrouter/free` 多「猜对」的两篇靠的是抽到 2.6B 小模型，而不是判断更好。

**接入**：`opencode.ai/inference/go/v1` 要求 `x-opencode-session` 头（缺了报 `MissingSessionID`），并且对陌生 UA 返回 1010，所以 `OneShotHTTPConfig` 增加 `Headers` 与 `SessionHeader`（每次请求现取 UUID，避免两次无关判定共享服务端会话）。`skillJudgeHTTPConfig` 在 base URL 含 `opencode.ai` 时自动加上这两项。

**运行配置**（`broker-watchdog.ps1`）：

```powershell
$env:WUPHF_SKILL_JUDGE_BASE_URL = 'https://opencode.ai/inference/go/v1'
$env:WUPHF_SKILL_JUDGE_MODEL    = 'crouton-alpha-free'
$env:WUPHF_SKILL_JUDGE_API_KEY  = <auth.json 的 opencode-go.key>
```

读不到 `opencode-go` 就整体回落到 8848 + `openrouter/free` + `bear` key（宁可结论不稳，也不要判定链路直接死掉）。

**实跑**（broker 32448，缓存里三篇正例已剔除）：

```
POST /skills/compile {"dry_run":true,"scope_path":"team/playbooks"}
→ {"scanned":11,"matched":3,"proposed":3,...,"duration_ms":37749}
```

11 篇 37.7 s（之前同一批走 `openrouter/free` 是 220 s，且有 1 篇 gate 不确定）。三篇通过的都是真正的通用方法。

**风险**：`crouton-alpha-free` 是 stealth 模型的 alpha 期免费通道，随时可能下线。下线时的症状是整轮 pass 全进 `errors`、缓存不增（错误不缓存）。修法是把 `WUPHF_SKILL_JUDGE_MODEL` 指向列表里其他可用模型，或临时退回 8848 的 `openrouter/free`。

**兜底（同日补）**：`crouton-alpha-free` 是 stealth 模型的 alpha 通道，下线时症状和 14.7 那类 bug 一样——整轮 pass 全进 `errors`、缓存不增、技能编译静默死亡。所以判定加了一次重试：

- `WUPHF_SKILL_JUDGE_FALLBACK_BASE_URL` / `_MODEL` / `_API_KEY`，当前指向 8848 + `nvidia/nemotron-3-ultra-550b-a55b:free`。选单一模型而不是 `openrouter/free`，理由同 14.8。
- **只在主判定明确失败时重试**（404/410/401/5xx）。超时不重试：一个卡住的服务不该让每篇文章付两个 180 s。
- 两次尝试各自拿一个 `WUPHF_SKILL_LLM_TIMEOUT` 预算，用的是**父 ctx**，不是第一次尝试那个已经耗尽的 ctx。
- 日志每次都会说清谁答的：`judge failed, asking the fallback` → `answered on the fallback model`。

**演练**（把主模型故意改成 `crouton-alpha-gone` 后重启）：

```
INFO  judge endpoint host=opencode.ai model=crouton-alpha-gone ... fallback_model=nvidia/nemotron-3-ultra-550b-a55b:free
WARN  judge failed, asking the fallback ... err="HTTP 400 ... Model is unavailable."
INFO  judge answered on the fallback model model=nvidia/nemotron-3-ultra-550b-a55b:free
```

pass 正常完成（7 篇 10.3 s，缓存补齐，零 errors），随后主模型改回 `crouton-alpha-free`。单测 `TestSkillJudge_FallsBackWhenPrimaryFails` / `TestSkillJudge_NoFallbackConfiguredPropagates` 覆盖「有兜底」和「没兜底时必须报错而不是缓存结论」。

### 14.12 one-shot 不再静默起编码 agent（2026-11-07）

14.6 留下的尾巴：`RunConfiguredOneShotCtx` 在活动 provider 没有 one-shot 时直接 `RunOpencodeOneShotCtx`，也就是为一个分类问题起一个带工具的 agent。当时只有判定链路绕开了它，其余 15 个调用点（lint、onboarding、entity 合成、company seed、operator、workflow 起草、openai-compat 桥…）仍然共用这条路径。

现在派发顺序是：

1. 活动 provider 自己实现了 one-shot（claude/codex/opencode）→ 用它，行为不变；
2. 否则该 kind 配了 HTTP 端点（`WUPHF_<KIND>_BASE_URL`/`_MODEL` 或 `provider_endpoints`）→ 走 `RunOneShotHTTPCtx`；
3. 都没有 → 回落到 opencode CLI，但**每个进程只警告一次**：

```
WARN provider: no one-shot for this kind, using the opencode CLI
     kind=ollama hint="set WUPHF_OLLAMA_BASE_URL and WUPHF_OLLAMA_MODEL to answer over HTTP instead"
```

为什么第 2 步是安全的：一个 HTTP provider 从来没有仓库工具，所以「不起 agent」不会丢掉调用方以为有、实际没有的上下文；反过来，今天的行为是让调用方**拿到一个它没选的模型**。

默认端点：`OneShotHTTPConfigFor` 现在读 `NewOpenAICompatStreamFn` 注册的那张表（`openAICompatDefaultsFor`），所以**本机装了 ollama/mlx-lm/exo、什么都没配**也会走 HTTP，`WUPHF_<KIND>_BASE_URL`/`_MODEL` 仍然优先。没注册 StreamFn 的 kind 依旧是 invalid，不会凭空猜一个地址。

代价要说清：provider 选的是本地模型但守护进程没起来时，以前会静默换成 opencode agent「假装答了」，现在直接报连接错误。这是有意的——调用方选的是那个模型，起 agent 只会给出一个它没选的答案。

测试：`TestRunConfiguredOneShotCtx_UsesHTTPWhenConfigured`（显式端点 → 命中一次 `/v1/chat/completions`，答案来自 HTTP）、`TestOneShotHTTPConfigFor_DefaultsToTheProviderEndpoint`（无 env 时落到默认端点、env 优先、未知 kind 保持 invalid）。当前运行态 provider 是 `opencode`（`SupportsOneShot=true`），所以这条改动在现网不改变任何行为。

### 14.13 watchdog 的两个 gate 不再对「按判定停用的采集器」报警（2026-11-07）

判定落地后，`WUPHF-RoutineScorecard` 是**有意停用**的（`MINIMAL-CONFIG.md`），但 `~/.wuphf/data/broker-watchdog.ps1` 的 routine gate 每 15 分钟跑一次 `check_routine_scorecard.py`，而那个脚本的第一条检查就是「快照不超过 26h」。没有采集器就没有新快照，于是它永远失败：

```
routine gate FAILED:
  - snapshot is 118.7h old (limit 26h)
```

到 11-07 11:36 已经 379 条。这不是告警，是一个不可能通过的门；它的真正代价是把真告警埋掉——这正是 §14 开头那条教训（告警文本比告警本身贵）。

三处改动，都是运行时文件（不入库，`broker-watchdog.ps1.bak-gateskip` 是改前备份）：

1. **gate 与被测对象对齐**：两个 gate 先读 `docs/office/expected-scheduler.json` 里对应任务的 `enabled`，为 `false` 就跳过并记录一次。routine gate 看 `WUPHF-RoutineScorecard`，thesis gate 看 `WUPHF-ThesisReview`。缺字段按启用处理（与 `check-office-drift.ps1` 的 `$t.enabled -eq $false` 同一条规则），所以字段写漏不会悄悄让门失效。
2. **去重键把数字归一**：原来对原始输出做 SHA256，而消息里嵌着会变大的数字（`118.7h` → `119.0h`），哈希每分钟都不同，去重永不命中。现在先 `\d+(\.\d+)?` → `#` 再哈希，同一类故障只记一行；故障种类变了仍会新记一行。
3. **`SKIP` 哨兵**：跳过只在状态文件从「无/哈希」变成 `SKIP` 的那一次写日志。实测连跑三次，`routine gate` 行数停在 379 不动。

两个坑写在这里，因为都会静默失败：

- **manifest 必须显式按 UTF-8 读**。watchdog 由任务计划器用 Windows PowerShell 5.1 启动，`Get-Content -Raw` 读无 BOM 的 UTF-8 文件会按 ANSI 解码，中文注释直接让 `ConvertFrom-Json` 抛异常——函数走 `catch` 返回 `$null`，于是「没有声明」被误读成「门正常运行」，跳过逻辑形同不存在。第一版就是这样：改了没生效，日志照旧 FAILED。改用 `[System.IO.File]::ReadAllText($path, [Text.Encoding]::UTF8)`。
- **缺 `enabled` 字段不等于停用**。第一版用 `[bool]$t.enabled`，而 `WUPHF-ThesisReview` 在 manifest 里根本没有 `enabled` 字段——结果一个**正常工作**的 thesis gate 被静默停掉。实测发现（日志出现 `thesis gate skipped`），改为 `PSObject.Properties['enabled']` 存在性判断。

验证：手工跑 `powershell -NoProfile -File ~/.wuphf/data/broker-watchdog.ps1` 两次，日志出现一次 `routine gate skipped: WUPHF-RoutineScorecard disabled by expected state`，`logs/watchdog-routine.state` 内容为 `SKIP`，第二次运行不再新增；thesis gate 不再出现 `skipped`，且不写状态文件（= `--score --dry-run` 通过）。

### 14.14 回合换模型重试（fallback），2026-11-08

14.11 之后办公室每个回合固定用 `WUPHF_OPENCODE_MODEL` 一个模型（当时是 `opencode-go/crouton-alpha-free`）。模型上游挂掉、限流，或 **id 本身写错**时，回合死掉后办公室就安静了，等人发现。队列里原有的两样东西都救不了这个形状：

- `transient-retry` 只在**同一个模型**上重连，最多 2 次（`headless_codex_queue.go`）；
- `recoverFailedHeadlessTurn` 走的是任务恢复阶梯（重试提示词 → BlockTask），也不换模型。

机制**是 wuphf 自己的**，不是 opencode 的能力：opencode CLI 只接受单个 `--model`，失败即非零退出，没有回退链；`internal/provider/opencode.go` 的单发路径同样不重试（那里唯一的 "fallback" 字样是条注释）。

**接入**（`0ebd90e`）：

| 位置 | 作用 |
|---|---|
| `internal/team/headless_model_fallback.go` | 读 `WUPHF_HEADLESS_MODEL_FALLBACK`；`withHeadlessModelOverride` / `headlessModelOverride`（**ctx 值**，不是环境变量——一个进程里多个 bot 并发跑回合，改 env 会把别人的回合一起改道） |
| `headless_codex_queue.go` | 同模型重试用尽且仍是 transient 类错误时，把回合标 `ModelOverride` 重新入队；同时把 `TransientRetries` 置满，**回退只试一次、不链第三个模型** |
| `headless_opencode.go` / `headless_codex_runner.go` | 取模型时 override 优先（`buildHeadlessOpencodeArgs` / `codexModelForBot`） |

失败形态判据沿用既有 `isTransientProviderError`：连接断开、5xx、以及 `bareExitStatusPattern` 里的裸 `exit status 1`——未知 model id 正是这个形状（opencode 把错误流成正文后以 1 退出）。

**实机验证**（journey2，主模型故意设 `bogus/nonexistent-model`，fallback `bear/deepseek-chat`）：

```
20:26:37 error: exit status 1 → transient-retry 1/2
20:26:40 error: exit status 1 → transient-retry 2/2
20:26:43 error: exit status 1 → model-fallback: switching this turn to bear/deepseek-chat
20:28:24 status=ok provider=opencode total_ms=100947 final_chars=510   # 正文正常投递
```

**运行配置**（`~/.wuphf/data/broker-watchdog.ps1`，`Restart-Broker` 内、紧接 `WUPHF_OPENCODE_MODEL` 之后）：

```powershell
$env:WUPHF_HEADLESS_MODEL_FALLBACK = 'opencode-go/iceman-contributor-free'
```

备选**也走 opencode-go 的 free 档**：回退只在主模型已经出事时才跑，不该引入第二个账单主体。三个 free 候选里 crouton 是主模型，`beam-free` 长输出会在中英文之间乱跳（14.11 实测），所以选 `iceman-contributor-free`（单模型、temperature 0、探活 9.2 s）。代价要说清：它也是 skill judge 的同一个模型，且通过的技能正例只有 3/11（14.11 表）——**能出话，但不保证质量**；回退回合仍会走正常的 manifest / 投递路径，所以质量差的输出照样可见、可追。

不配该变量 = 完全回到旧行为（同模型重试两次后失败）。`codex` 运行时同样读这个覆盖；claude / cli-agent / openai-compat 未接（回退 model id 与它们的 provider 不通用）。

### 14.15 done 必须有产物、cancel 必须真的停（2026-11-08）

Journey 2 暴露的两个缺陷（`core-loop-runs/2026-11-08-journey2/scorecard.md` §4 的 J2-1 / J2-3），同批修。

**J2-1：done 与产物脱钩。** 旧 gate 只绑「有 Definition（或父有 Definition）」的任务，于是 `NEX-100/102/148/150` 全部 `done + hollow_completion=true`，父 Issue 等着一件不存在的东西；`NEX-104` 还指向一个从不存在的 `team/playbooks/…md`——存在性检查在 `peekTaskDoneArtifact` 里对无 Definition 任务直接早退。

修复后的判据（`reachedDone` 分支，`broker_tasks_mutation_service.go`）：

| 场景 | 结果 |
|---|---|
| 带 `artifact_path`，文件存在 | 放行；`hollow_completion=false` |
| 带 `artifact_path`，文件不存在（wiki 或任务 worktree 都没有） | 拒（`artifact_required`，"does not exist in the wiki"）——**现在对无 Definition 的任务同样生效** |
| 无产物、无声明 | 拒（`artifact_required`），错误里同时给两条出路 |
| 无产物、`verification=not_required` + `verification_reason` | 放行；理由与 actor、时间戳落到 `artifact_not_required`（`not_required: <理由> (<actor>, <time>)`），随任务 wire 一起暴露 |
| `verification=not_required` 但没理由 | 拒 |
| 父 Issue 结算时仍有 `done && artifact 空 && artifact_not_required 空` 的子任务 | 拒（`conflict`），错误点名子任务 ID |

声明可以来自请求，也可以已经落在任务上：`skill_run` 这类内部记账任务在创建时就带 `not_required: skill run — the invocation transcript is the record`（技能回合的产物是对话记录，不是文件），reopen 后的任务保留它原有的声明；一旦真的交付了产物，声明被清掉。MCP 侧 `team_task` 新增两个入参 `verification` / `verification_reason`（`internal/teammcp/server_types.go`、`server_tasks.go` 透传）。

**J2-3：取消不撤销已排的派活。** `NEX-95` `POST /tasks/{id}/cancel` 返回 `{"changed":true}` 后，同一秒状态回到 `in_progress`，6 秒后起了一个 11 次工具调用的回合。两个洞：

1. 终态收尾按**任务当前频道**关 job（`completeSchedulerJobsLocked("task", id, taskChannel)`）；任务换过 home 之后，留在旧频道里的 `task_follow_up` / `recheck` 不会被关。现在终态一律传空频道，关掉所有指向该任务的 job。
2. 状态可被拉回：`claim`/`assign` 无条件写 `in_progress`，`reassign` 只跳过 done/review，`resume`/`block`/`review` 也都能把终态任务拉回执行；计划刷新（`refreshPlannedTaskBlockStateLocked`）同样不看终态。现在 `rejectTerminalTaskRegressionLocked` 在 `MutateTask` 的既有任务分支统一拦：`canceled/cancelled/archived` 只允许 `comment`/`cancel`/`reopen`；`done/completed` 拒绝 `claim/assign/reassign/resume/block/review`。`reopen` 是唯一合法的复活路径。

**触发链（可解释实测）**：scheduler 取到已到期的 follow-up job 之后、在锁外派回合；同一时刻 cancel 落锁关 job。被派出的回合仍按旧快照跑，调 `claim` 把任务写回 `in_progress`，`scheduleTaskLifecycleLocked` 又装了一个新的 `task_follow_up`——`scheduler.json` 里那唯一一条 `task_follow_up` 正是重装出来的那条，而不是漏关的旧 job。所以两侧都要修：关 job 关全部频道，状态回退一律拒绝。

**夹具影响**：`office-eval` 里那些不以交付为考察对象的 job（lifecycle-basic、dependency-handoff、grounding 的停单/便条、human-sovereignty、live-paths 的 reopen 段、task-integrity 的双重重放段）现在显式声明 `not_required`；`task-integrity` 的自愈交付则改成先真正写进 wiki 再 complete（引用必须真实）。同一规则也让「done 任务被 reassign 触发 memory-gate 重跑」这条旧断言失效——现在 reassign 在终态直接被拒。
测试：`TestCompleteWithoutArtifactIsRefused`、`TestArtifactGate_BindsUndefinedTask`、`TestArtifactGate_RejectsPhantomArtifactWithoutDefinition`、`TestParentSettlementBlockedByHollowChild`、`TestCancelClosesJobsAcrossChannels`（去掉频道修复即失败）、`TestCanceledTaskRefusesRevivalActions`。

**J2-5：决策路径上的同一条门槛，原本是死路（同日补）。** J2-1 只接了 `POST /tasks` 与 MCP，两个入口都有 `verification` / `verification_reason`；人点的「Approve」走的是 `POST /tasks/{id}/decision`，那个请求体只有 `action` / `comment` / `created_by`，服务端 `recordTaskDecisionInternal(taskID, rawAction, actor, comment)` 也没有这两个参数。于是人批准一件真的没有文件的工作时会拿到 409，错误里写着的两条出路在这个端点上根本无处可放——UI 里 `artifact_path` 出现次数为 0，`updateTaskStatus` / `postDecision` 都不带产物字段。

现在：

| 层 | 变化 |
|---|---|
| 决策端点 | 请求体新增 `artifact_path` / `verification` / `verification_reason`（`broker_inbox_handler.go`）；存在性检查对**随决策提交的**路径生效 |
| 决策内核 | `recordTaskDecisionInternal(…, delivery DecisionDelivery)`；声明与产物在 gate **之前**解析成 effective 值，拒绝时任务上不留半截声明；放行后才落 `Artifact` / `ArtifactNotRequired` / `HollowCompletion` |
| 前端 API | `updateTaskStatus(…, { artifactPath, verification, verificationReason })`、`postDecision(taskId, action, comment?, delivery?)` |
| 前端 UI | `DeliveryGateDialog`：409 命中产物门槛时弹「贴产物路径」/「这次没有文件 + 理由」两选一；接在 `TaskActionToolbar`（Mark done / Approve）与 `DecisionPacketRoute`（决策卡 Approve） |

判据：无产物任务在决策卡上批准 → 409 且任务不动；补一句理由再点 → 200、`artifact_not_required` 落盘、`hollow_completion=true`；写不存在的路径 → 409 "does not exist"。测试：`TestDecisionApproveCarriesDeliveryFromHTTP`（真 HTTP handler）、`TestDecisionApproveRefusesPhantomSuppliedArtifact`、`TaskActionToolbar.test.tsx`（三例：弹窗、缺理由拦截、非本门禁错误仍走行内横幅）、`lifecycle.test.ts` 的 `postDecision delivery` 三例。

**顺带修掉全量测试的间歇性 `TempDir RemoveAll` 失败。** `queueTaskDistillation` / `queueTaskReview` / `queueWorkflowAppDetection` / `queueAppAcceptanceEval` 四条完成钩子用的是裸 `go func()`，不受 `bgWG` 管理：`Stop` 不等它们，于是钩子在测试目录被删之后仍往 runtime home 写，表现成 `TempDir RemoveAll cleanup: The directory is not empty`（隔离跑必过、全量约每三次撞一次）。四条统一改为 `b.trackBackground`——这本来是给「Stop 之后不能再落盘」准备的机制。测试侧 `newTestBroker` / `newCompletionHookBroker` 注册 `t.Cleanup(b.Stop)` 把钩子排干。

### 14.16 不重做已交付的活（J2-6，2026-11-08）

scorecard §4.5 的那半句缺陷已经说得很准：**「重投路径不查同题任务是否已完成，去重只覆盖同一 active 回合」**。`NEX-100`（续约邮件，ae）/ `NEX-102`（账户简报，analyst）在 `10:56:04Z` 建、随后 `done`；`11:21:58Z` 办公室又在同一个父 Issue `NEX-95` 下建了 `NEX-148` / `NEX-150`，把同一件事重做了一遍（两条新任务同样 `hollow_completion=true`）。四个空壳完成，换来两个已有的交付。

根因是三处「终态即忽略」：

| 位置 | 旧行为 | 后果 |
|---|---|---|
| `findReusableTaskLocked`（`broker_tasks_lifecycle.go`） | `isTerminalTeamTaskStatus` 直接 `continue` | `POST /tasks` create 与 `EnsureTask` 对已 done 的同题任务视而不见，直接新建 |
| 子任务兄弟去重（`applySubIssueCreateRulesLocked`） | 终态兄弟 `continue` | 父 Issue 下已交付的 lane 不能被 `NEX-148/150` 这类重述挡住 |
| 父 Issue 自身的已交付面 | 无此概念 | 父 Issue 的每条 lane 都交付完了，仍可再拆一轮 |

**为什么不能直接把终态纳入复用**：把 create 合并进一条 `done` 任务，等于把新请求**静默吞掉**——没人会去干活，任务看起来还是 done。所以新路径是**拒绝 + 点名已有的那条**，让人/办公室自己选：reopen 它，或者用 opt-out 明确承认这是新活。

修复（`findDeliveredTaskLocked`、`applySubIssueCreateRulesLocked`、`refuseNewLaneUnderDeliveredParentLocked`）：

| 场景 | 结果 |
|---|---|
| 顶层 create，同题 + 同 owner + 同频道已有 `done` 任务 | 拒（`conflict`）：`task "…" already shipped as NEX-100 (done, owner ae) — reopen NEX-100 instead of re-creating it, or pass allow_duplicate=true…` |
| 子任务 create，父下同题兄弟已 `done` | 拒，点名兄弟 ID；`allow_duplicate=true` 放行 |
| 子任务 create，父下**每一条**已有 lane 都已 `done` | 拒：`parent NEX-95 has already delivered every lane it had (…)`；`allow_duplicate=true` 放行 |
| 兄弟/父 lane 是 `canceled` / `archived` | 不算交付，仍可重建（J2-3：取消之后重做必须合法） |
| 同题但不同 owner | 不拦（换人=换活） |

范围：门禁在 `POST /tasks` 的 create 分支与子任务守卫上，也就是 MCP `team_task action=create` 走的那条路。`POST /task-plan`（人的新建表单与批量计划）**不在**门禁内——那是人显式列出的计划，是否重做由人决定。

`allow_duplicate` 是唯一的 opt-out，wire key 与 MCP 入参同名（`internal/teammcp/server_types.go`、`server_tasks.go`）。**默认 create 不带它**，所以门槛默认生效。

**歧义必须说清**：标题匹配用的是 `titlesAreSimilar`（归一化精确 key 或 Jaccard ≥ 0.85），而 J2 的真实两条重述是**换词 + 换 owner** 的中文标题——`为 Acme… 起草定制续约邮件` vs `起草三封定制续约邮件：Acme / Brightline / Corti Labs`，token 相似度 0.57；`整理三客户续约账户简报` vs `产出三份账户简报：…`，约 0.1（中文没有空格，整段汉字成一个 token；见 `task_title_similarity.go`）。所以**真正挡住 J2 那个形状的是父级「每条 lane 都交付完了」这条判据**，不是标题相似度。标题相似度只能挡字节级/词序级的重述。

代价（有意保留、已用测试钉住）：父 Issue 的 lane 若一条条地建、且前一条在创建下一条之前就已 done，下一条需要显式 `allow_duplicate=true`。正常情况下 lane 在计划批准后一次性建出来，这个顺序场景少见，而错误消息把该传什么写全了。

实机（journey2，:7910，`gawkbot-j26.exe`）：顶层 `NEX-375` 建 → approve → complete（`artifact_path=team/emails/renewals-q3-q4.md`）→ 同题再 create 得到 409 `task "…" already shipped as NEX-375 (done, owner ae) — reopen NEX-375 instead of re-creating it, or pass allow_duplicate=true…`，带 `allow_duplicate` 得 `NEX-379`。子任务：父 `NEX-381` 的下游 lane `NEX-384` done 后再建 lane 得 409 `parent NEX-381 has already delivered every lane it had (NEX-384)…`，带 `allow_duplicate` 得 `NEX-387`。探针任务已全部 cancel，journey2 已 `paused`。

测试：`TestCreateRefusesRedoOfDeliveredTask`、`TestCreateAllowsRedoAfterCancelAndForOtherOwner`、`TestSubIssueRefusesRedoOfDeliveredSibling`、`TestSubIssueRefusesNewLaneUnderDeliveredParent`、`TestNewLaneAfterDeliveryRequiresOptIn`、`TestTeamTaskCreateForwardsAllowDuplicateOnlyWhenSet`（MCP 只在显式设置时下发 `allow_duplicate`）、`TestCreateAllowsRepeatFromSystemAndHumanActors`（routine 的 `system` actor 与人新建表单不被挡）。

### 14.17 wiki 与 memory backend 解耦（K，2026-11-08）

`ensureWikiWorker` 首行原本是 `if ResolveMemoryBackend("") != markdown { return }`，于是 `memory_backend=gbrain` 的办公室**永远没有 wiki worker**：`/wiki/write*`、`/notebook/*`、`/review/*` 一律 503 `worker stopped`，bot 的绕法只有直接写盘——产物不进 git、`git status` 里是 `??`、provenance 页脚与提取循环全部缺席。

这是把两个独立层错绑在一起：wiki 是 `<runtime>/wiki` 里的 **文件系统 git repo**（每次写入按 bot 身份 commit），memory backend 是**检索层**（markdown / gbrain / 无），二者只在「检索索引从 wiki 里读」这一点上相邻，而索引后端本来就有自己的开关（`WUPHF_WIKI_BACKEND`、`newWikiIndexForBackend`）。

现在：

| 项 | 之前 | 之后 |
|---|---|---|
| `ensureWikiWorker` | 只在 `memory_backend=markdown` 时启动 | 任何 backend 都启动（失败仍只记 `wikiInitErr`，不致命） |
| `/health.memory_backend_*` | markdown 下 `memory_backend_ready` 兼作 wiki 探针 | 保持原义；markdown 下仍与 worker 绑定（既有语义不破） |
| `/health.wiki_ready` | 无 | 新增：worker 是否就绪，与 backend 无关——gbrain 办公室终于能看见 wiki 挂了 |

判据：`WUPHF_MEMORY_BACKEND=gbrain` 启动后 `WikiWorker() != nil`、`/health.wiki_ready=true`、`/wiki/write` 200 且产物进 git。测试：`TestWikiWorkerStartsUnderGBrainBackend`（同时断言 `wiki_ready`），既有 `TestHealthMemoryBackendReadyReflectsWikiWorker` 保持绿。
