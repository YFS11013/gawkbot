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

`complete` 走 `internal/team/broker_tasks_mutation_service.go:503`。

### 1.1 「Bot stuck on」求救单

`[@bot] Bot stuck on: <原任务>` 是 error-recovery 自动建的止血单（details 末尾带 `Original work: … · ACMER-NNNN`）。
原任务已解决就**直接 complete**，不要派活给它。

- 关掉一张**常会再生一张**（例：2710 → 2743）——关完必须再 `GET /tasks?all_channels=true&include_done=true` 复查。
- 同一个求救单反复冒头（例：`task-skill-1882` 之后又出 2730）= 根因没修，关单只是止血，要在同轮指出根因。

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
| 其余 | `status` / `provider` / `focus_mode` / `dispatch` / `memory_backend*` / `build` / `session_mode` / `one_on_one_agent` | `build.version=dev` 表示这是本地 `go build` 产物 |

日志重定向是否生效，不看 `/health`：看 `~/.wuphf/logs/broker.err.log` 里有没有**本次启动之后**的新行（`Start-Process` 的重定向是覆盖不是追加，重启后文件会缩到几行）。
