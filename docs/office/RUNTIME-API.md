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
| 数「空壳完成」 | 在 `GET /tasks?...` 的结果里筛 `status=done && artifact 为空`；2026-11-01 起新关的单还会带 `hollow_completion=true`（同一事实的显式标记，历史任务没有该字段） |

`complete` 走 `MutateTask`（`internal/team/broker_tasks_mutation_service.go`，done 转换处理在 `reachedDone` 分支）。

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

## 9. 会议准备包（A4）：建单 → playbook → 机械判据

一件活拆成三件可替换的件，一条链串起来：

| 件 | 位置 | 作用 |
|---|---|---|
| 建单脚本 | `scripts/office/new_prep_task.py` | `--who <对象> --date <YYYY-MM-DD> [--owner bot] [--topic 主题] [--approve]`。路径只算一次，同时用于 `artifact_path`、成功标准文字、校验 spec——手写三次就会漂 |
| playbook 源 | wiki `team/playbooks/meeting-prep.md` | 四节写法与判据。标准 wiki 写入后自动编译成 `team/playbooks/.compiled/meeting-prep/SKILL.md`，`/skills` 里出现 `meeting-prep` |
| 判据脚本 | `scripts/office/check_prep_pack.py` | `command` kind 的 spec；`--self-test` 内置 8 个正反样例 |

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
