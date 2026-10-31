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
