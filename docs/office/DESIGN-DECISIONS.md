# 外汇 Office 设计决策（2026-10-29）

> 范围：`~/.wuphf` 这套 office（cos / forex-watch / market-analyst / data-keeper / fund-invest），
> 以及驱动它的 `forex-bridge.py` 与 gawkbot 0.240.1（源码 `C:\Users\ireke\code\gawkbot`）。
> 本文只记决策与依据，不记操作日志。

## 1. 问题陈述（有证据）

| 观察 | 数据 |
|---|---|
| bot 一天的计算量 | 95 个 turn，305 分钟，17 个失败 |
| 频道输出 | 91,678 字 |
| 真正可交付 | 1 份 3–5 千字简报（生成 108–213 秒） |
| 对照 | 同一问题，单次前台分析（含 5 个实时源交叉验证）产出更完整 |

结论：**产出占比约 2%，其余是噪声、循环和重复劳动。** 问题不在“bot 数量”，而在缺失闭环与错误的层。

## 2. 设计原则（自上而下）

1. **确定性优先**：能写成脚本的，不交给 LLM。数据装配、时区换算、日期推导、条数校验全部脚本化。
2. **LLM 只做判断**：一次调用，只写「预期差 / 已定价多少 / 什么会推翻」，5–8 行。
3. **产物落 artifact，聊天只放指针**：符合作者 core loop（`docs/specs/core-loop.md`：Task → artifact → deterministic learning hook）。
4. **单一 owner**：一个任务类型只有一个 bot；不设“读同一份数据的第二个分析 bot”。
5. **证据可回查**：任何数字必须能在 digest 原文里逐字找到，否则写「digest 未提供」。
6. **强制规则落机制**：写进 `RULES.md` / 权限配置 / 代码，不写进“记忆”就当完成。

## 3. 决策

### D1 模型层：用订阅内的 `opencode-go/beam-free`，禁 reasoning 模型

- 探活历史（2026-10-29 上午）：`opencode-go/beam-free` 报 `Free usage limit reached`；`gpt-5.4-nano` 报 `Insufficient account funds`。订阅恢复后两者不再是现状。
- 探活结果：`bear/qwen3.5-122b` 7s ✓；`bear/deepseek-chat` 7s ✓；`bear/deepseek-reasoner` 12s ✓ 但**把 `<think>` 过程当消息推到频道**；`bear/gpt-5.4`、`bear/kimi`、`bear/deepseek-v4-flash` 报 channel/token 错。
- **决策（2026-10-29 19:53 修订）**：`WUPHF_OPENCODE_MODEL=opencode-go/beam-free`（读取点 `internal/config/config.go:566`）。opencode-go 为订阅通道，配额恢复后实测 7 秒可用；订阅内使用不额外计费。
- 历史：当天早些时候该模型报 `Free usage limit reached`，才临时切到 `bear/deepseek-chat`；配额/订阅恢复后切回。
- **兜底**：若再次出现 `Free usage limit reached`，一条命令切到 `bear/deepseek-chat` 并重启（2026-10-29 实测 7s 可用、无 CoT 泄漏）。
- 仍禁用 `bear/deepseek-reasoner`：把 `<think>` 过程当消息推到频道。
- 代价：切换需重启 broker，token 随之轮换（外部脚本需重取）。

### D2 数据层：digest 是唯一数据源

`~/.wuphf/data/forex-bridge.py` 为唯一入口，每 6 小时由计划任务生成 `wiki/team/forex-bridge-digest.md`。已落地的字段约束：

| 字段 | 规则 | 原因 |
|---|---|---|
| 时间 | 每个事件输出 `MM-DD 周X HH:MM GMT / MM-DD 周X HH:MM 北京` | 模型自己换算必然出错（历史 bug：10-28 标成 Tue） |
| 星期 | 由解析日期生成，不由模型推导 | 同上 |
| 星级 | ★★★ 央行决议/声明/发布会；★★ 其他高影响/关联货币；★ 中影响 | 让“重要性”成为数据，而非模型判断 |
| 已过 | `（已过）` 标记 | 防止把昨天的 BoC/RBNZ 写成“未来事件” |
| 倒计时 | 顶部「下一个高影响事件 + 还有 N 小时」 | 简报第一句的定调直接引用 |
| 新闻 | 72h 内、按链接去重、附 ≤140 字摘要 | 无摘要时模型会从标题拼数字并混用单位 |
| 关联货币 | NZD/CAD 单列，不进交易对 | 今天最大冲击（RBNZ/BoC）曾被过滤掉 |

### D3 输出层：脚本渲染全部正文，聊天只做逐字复制

```
digest ──► render_artifacts()（脚本，0 LLM）
             ├─ wiki/artifacts/forex/<date>.md           全文（定调/点位/时间轴/新闻/关键位）
             └─ wiki/artifacts/forex/<date>-headline.md  5 行摘要
                                                     └─► forex-watch：read + 逐字复制到聊天
```

- 原设计（脚本 80% + 模型写判断段）**在 2026-10-29 被实测否决**：模型两次编造 FOMC 结果（“维持 3.75–4.00%、10–2 票、Cook/Hammack 主张加息、12 月定价 88%”），digest 里根本没有这些字段；删掉日历里的“已过”行（幻觉钩子）后仍然复现。
- 因此收敛为：**模型不写任何面向人的数字**，只做“读文件 → 原样输出”。
- 预期差段落降级为 artifact 内的占位（`## 四、预期差（需人工或单独补充）`），等有可信实际值来源再启用。
- artifact 路径受 `internal/team/artifact_commit.go` 正则约束：`wiki/artifacts/[a-z][a-z0-9_-]*/[A-Za-z0-9][\w.-]*\.md`。

### D4 组织层：按 core loop 切，不按“职能”切

| 成员 | 决策 | 依据 |
|---|---|---|
| cos | 保留，只承接**人主动派活**与 intake | `core-loop.md` R4 的结构化提问就是 CoS 的职责 |
| forex-watch | 保留，唯一的外汇 owner | 单一 owner |
| market-analyst | **合并进 forex-watch** | 同读一份 digest，且 07:30 跑在 08:30 简报之前（依赖顺序反了） |
| data-keeper | **routine 降级为脚本 + assert** | “deterministic hook, not LLM whim” |
| fund-invest | 保留（不同数据源、周频） | 与 forex 无重叠 |

不采用“多加一个 bot 来监控另一个 bot”的做法（历史上 data-keeper 就是这么来的）。

### D5 权限与安全

- `task: deny` 写入 headless agent 权限（`internal/team/headless_opencode_mcp.go`）：模型用 subagent 会让 turn 卡在 “subagent route unsupported”。
- 目标状态：headless 去掉 `bash` 与 `external_directory`，只留 `read`（限 wiki/data）。理由：今天 cos 在探 `127.0.0.1:7892` 鉴权、直读 `broker-state.json`。
- 禁止 bot→bot @：跨 DM 唤醒与 `team_task` 均已验证不通（`doom_loop` 拒绝、403），堆着只会制造循环。

### D6 验证与淘汰

固定 5 维打分：时间准确 / 事件完整 / 预期差 / 已定价 / 风险上下文。

| 指标 | 基线（10-28） | 目标 |
|---|---|---|
| 每日 turn 数 | 95 | ≤ 6 |
| 单份简报 LLM 调用 | 3–5 轮探索 | 1 次 |
| 生成耗时 | 108–213s（背景 305min 噪声） | ≤ 120s |
| 失败 turn | 17 | ≤ 1 |
| 5 维评分 | 3/5 | ≥ 4/5 |

淘汰规则：一周后若脚本版在 ≥3 维不输，砍掉 bot 的写作职责；若 LLM 段每天被人手改，连这一段也不自动化。

### D7 Composio 在 Windows 上不做自动安装（可恢复）

决策：`composioAutoInstallSupported` 在 Windows 恒为 false，sign-in 流程**不进入 `installing`**，直接返回 `cli_missing` + 手动命令；
`/integrations/composio/signin/status` 携带 `auto_install`，前端在没有安装器的平台上不渲染「Set it up」，也不说「setup did not finish」。

原因（`OFFICE-3043` 期间实测）：Windows 的 `defaultComposioInstaller` 必然返回「不支持」（官方安装器是 `curl | bash`，无 Windows 等价），
但旧流程仍会进 `installing`，耗完预算落到 `install_failed`，而它的按钮是 **Try again** —— 再失败，形成不收敛的循环；
每次 broker 启动都会重放一次（`22:35` 之前那次在 `21:42:06`，boot 后 10 秒），并把一个阻塞对话框推给没有任何东西可装的人。

| 请求 | 修复前 | 修复后（2026-11-01 22:55 新二进制实测） |
|---|---|---|
| `POST /integrations/composio/signin/start {"auto":false}` | `installing` → 超时 → `cli_missing` + "did not finish" | 立即 `cli_missing` + "cannot run the setup by itself"，`auto_install:false` |
| 同上 `{"auto":true}` | 每次启动重放 | `idle`（latch 生效） |
| broker 启动后的日志 | `composio signin: install: ... not supported` | 无新增 |

**恢复路径**（以后真要接第三方 SaaS 时）：
1. 装上 CLI：`npm install -g @composio/cli`，并配 `WUPHF_COMPOSIO_API_KEY` —— **装好后流程自动恢复正常登录，不需要改代码**（`composioBinary()` 命中即跳过整段安装分支）。
2. 若 Windows 将来有了官方安装器：在 `broker_composio_signin_windows.go` 实现 `defaultComposioInstaller`，并把同文件的 `composioAutoInstallSupported` 改为 true。全仓只有这一处读它。
3. 不需要这个能力时：不装即可，代价仅为「不能连接第三方 SaaS app」，office 的 wiki / 任务 / 审阅 / 准备包链路完全不经过它。

判据：`go test ./internal/team -run TestComposioSignin`（Windows 上自动跳过两条安装路径用例）、`npx vitest run src/components/apps/integrations`。

## 4. 被否决的方案

| 方案 | 否决理由 |
|---|---|
| 靠加 bot 提升质量（分析、盯梢、数据、路由各一个） | 组织架构解决不了 prompt/数据问题；协作链路实测不通 |
| 全脚本化、不留 LLM | 与作者 core loop 的“判断层”冲突；预期差与定价需要模型 |
| 用免费模型跑生产 | 硬停；且作者明说 token 不再是设计约束 |
| 用 reasoning 模型出简报 | `<think>` 会作为消息推到频道，污染 surface |
| 把结论存进 bot 的 notebook 当“记忆” | 曾因 `/notebook/catalog` 恒 404 而失效（`internal/teammcp/context_tools.go:491` 调了 broker 未注册的路由）。**2026-11-01 已修**：四个 `/notebook/*` 路由已注册并实测，契约见 `RUNTIME-API.md` §6 |
| 直接改 `broker-state.json` 改 prompt | broker 内存持有该文件，会被覆盖；必须走 `PATCH /scheduler/{slug}` |

## 5. 强制机制（不靠记忆）

> 工作流（巡检 / 排障 / 变更）与禁止清单见 `OFFICE-RULES.md` §0–§4、§7，接口契约见 `RUNTIME-API.md`；本节只登记“机制清单”，不重复流程描述。

- `~/.omp/agent/RULES.md`（sticky，每次请求注入）：先 recall、证据制、卡死先对基线、改完必须读回产物。
- `headless` 权限：`task: deny`；下一步去 `bash`。
- routine prompt（892 字）：数字铁律、同刻合并、首行定调、禁 subagent。
- **policy 1（scoped: cos）**：跨 bot 派活必须真的调用 `team_task` / `team_broadcast` 并给出回执（task id / message id）；禁止无回执的“已转交”；私聊里 @ 非成员不会唤醒，必须开到对方所在频道。`policy-1793277982913860200`
- **policy 2（scoped: forex-watch, market-analyst）**：回答日历/前值/点位只能转述当日 artifact 与 headline，保留 GMT+北京双时区；文件没有的事件与数字写「digest 未收录」，禁止引用 RSS 标题或其他网站日历。`policy-1793278353959898300`
- 桥脚本：时区/星期/星级/已过/去重全部由脚本计算，模型无权换算。
- **运行时契约**：`docs/office/RUNTIME-API.md`（数据空间必须带 `X-WUPHF-Agent`、`PATCH /apps/{id}` 只 rename、`workflow_provider` 定义文件缺失即每次 failed、任务/routine 只走 HTTP 不手改 json/db）。

## 6. 已知风险与开放问题

1. ~~**上游 bug**：`context_lookup` 调 `/notebook/catalog` 404~~ —— **已修（2026-11-01，本 fork）**：上游 main 同样没有这四个路由（`git grep '"/notebook' <upstream>/internal/team` 无 handler），所以是本 fork 自己补的：`internal/team/broker_notebook_routes.go` + `notebook_git.go` + worker 的 `IsNotebookNote` 分支。同时把客户端「404 也算 retryable」改成 `notebook_endpoint_unavailable` 且不可重试 —— 否则 bot 会继续在永不痊愈的 404 上烧 turn。
2. **routine 无 artifact 输出**：`operator-agent-routines.md` 里 md artifact 标为 DONE，但本版本未见落盘；需在 prompt 里显式写路径，或补实现。
3. **模型可用性单点**：`bear/*` 有 channel 级报错（gpt-5.4/kimi/deepseek-v4-flash 不可用），切换前必须探活。
4. **未决**：是否订阅 OpenCode Go（保留原厂工具链）而非走 bear 代理。
5. **未决**：`market-analyst` 是合并进 forex，还是改成“静默计算、只写 artifact 不发言”。
6. ~~branch-tracer 工具缺文件~~ **已解决（19:39）**：该工具是 omp-evolve A/B harness 的脚手架，opencode 自动加载 `tools/*.ts`、无配置引用，已连同 stub 移至 `~/.config/opencode/tools.disabled-20261029/`。验证：19:41 cos turn 成功，无新增 `Cannot find module`。同族的 `tsc-analyzer.ts` 已一并移走（21:45）；`tools/probe-bg.ps1` 保留——`.ps1` 不被 opencode 当作 tool 加载，属惰性文件。
7. **`internal/team` 测试包编不过**：`headless_session_typing_test.go` 缺 `sessionAnswer` / `pollUntil`，新增的回归用例无法执行；`go build` 正常，仅测试侧破损。
8. **从未启动的任务不会被 `StalledTurns` 回收**（2026-11-02 发现，来自 `ACMER-3238`）。`StalledTurns` 的计数发生在 `AppendTaskLedgerEntry` 内，只有 bot 跑过至少一轮、且那一轮没有留下可验证产物时才 +1。任务创建后 bot 根本没开始（`pipeline_stage` 停在 `breakdown`），计数器就没有机会动，8 轮阈值永远到不了。这类任务只能靠 UI 的 `[task_stalled]` 通知发现，而通知需要人接——`ACMER-3238` 从 15:06 挂到 23:05，无人处理后才被手工取消。**判据缺口**：现有实现覆盖「跑了但没进展」，不覆盖「从未开始」。
9. **`PATCH /scheduler/{slug} {"enabled":true}` 对已过期的 job 会立即点火**（2026-11-02 实测）。为验证 overdue 检测而启用一个 `next_run` 停在 2026-10-29 的 routine，调度器在同一秒执行了它并创建任务 `ACMER-3238`；紧接着用 `enabled:false` 关掉 job，任务不会被回收。**后果**：一次 20 秒的测试留下了挂了 8 小时的活跃任务。禁用只阻止未来的触发，不撤销已经创建的任务。
10. **`WUPHF-BrokerWatchdog` 没有「同一 job 反复过期就停手」的闸**。`Restart-Broker` 只有 `restartFloorMinutes = 20` 的防抖，即两次重启至少间隔 20 分钟，但没有记录是哪个 job 导致的重启。如果某个 job 因前置条件不满足被跳过且 `next_run` 不推进，它会持续过期，watchdog 就会每 20 分钟重启一次 broker。2026-10-30 已出现两次连续重启（20:36 与 22:21），根因正是 `task-follow-up` 的 `next_run` 停在过去。当前最小配置下没有触发条件，但重启循环会破坏「7 天无异常」这个试验前提。

## 7. 实施状态（截至 2026-10-29 11:40 UTC）

| 项 | 状态 | 验证方式 |
|---|---|---|
| digest 双时区/星期/星级/已过/倒计时/72h/去重/摘要 | 已上线 | 重新生成后读回 digest 原文 |
| **日历只保留未发生事件**（移除幻觉钩子） | 已上线 | `未发生 9 条 / 本周共 24 条` |
| **脚本渲染 artifact + headline** | 已上线 | `wiki/artifacts/forex/2026-10-29.md`、`2026-10-29-headline.md` |
| **聊天改为逐字复制 headline** | 已生效 | 05:56 UTC 消息与 headline 文件逐字一致（101 秒，649 字） |
| QA assert 取代 data-keeper 质检 | 已上线 | 健康时无 alerts.md；异常追加 `wiki/team/alerts.md` |
| 关闭 market-analyst / data-keeper ×2 routine | 已生效 | PATCH 返回 `enabled=False` |
| 模型 `opencode-go/beam-free`（订阅） | 已生效并持久化 | `/health` 回显；bot 命令行 `--model opencode-go/beam-free`；cos 自检 21s 成功 |
| `task: deny` | 已编译 | 生成的 per-agent config 含该项 |
| `bash: deny` + `webfetch: deny` | 已编译 | 同上，`"bash": "deny", "webfetch": "deny"` |
| **MCP 写回修复**（`merged["mcp"] = mcp`） | 已上线 | cos/data-keeper/forex 的 per-agent 配置均含 `mcp: ['wuphf-office']`；forex `team_broadcast` 自检 msg-1740 成功 |
| **停用门禁修复**（`schedulerJobDue` 检查 Enabled） | 已编译，**未实测** | 等下一次到点或被 `Run now` 触发时观察 |
| **omp-evolve 脚手架停用** | 已做 | `branch-tracer.ts`、`branch-tracer-stub.ts`、`tsc-analyzer.ts` 移至 `tools.disabled-20261029/`；19:41 cos turn 12s 成功 |
| **artifact 进 git + 索引** | 已做（10-30 巡检） | bridge 内置 `git_commit`；新增 `team/forex-brief-<date>.md` 摘要并出现在 `index/all.md`；`2026-10-29` 两份已回填提交 |
| **倒计时加统计时刻** | 已做 | digest 顶部两行均带「统计于 HH:MM UTC」 |
| **调度器心跳 + panic 恢复** | 已上线（10-30 19:22） | `broker.err.log` 出现 `scheduler: loop started (poll=20s)` |
| **外部看门狗** | 已上线 | `WUPHF-BrokerWatchdog` 每 15 分钟；手动运行 rc=0 无动作 |
| **Medium 事件覆盖修复** | 已做 | `MEDIUM_KEEP` 扩充 cpi/gdp/ppi/employment/payroll/unemployment/retail sales/pmi；本周事件 24 → 30，德国 CPI/GDP、美国 ECI 恢复 |
| 清理 | 已做 | 删除 `data/brief-2026-10-29.html`、`wiki/artifacts/forex/brief-*.md`、临时构建产物 |

**未完成**：预期差的实际值来源（digest 无 actual）；`/notebook/catalog` 404；`market-analyst` 去留；`internal/team` 测试包破损。

> 可执行版本见同目录 `TODO-NEXT.md`（P0/P1/P2 分级，每条带验收标准）。

## 8. 事故与修复记录

### 8.1 MCP server 未写入 per-agent 配置（2026-10-29）

- **现象**：cos / data-keeper / market-analyst / forex-watch 的 turn 全部报“没有 team_broadcast / team_task”，只能做本地文件操作。
- **根因**：`internal/team/headless_opencode_mcp.go` 把 `wuphf-office` 写进局部 `mcp` map，但从未 `merged["mcp"] = mcp`。用户基础配置 `~/.config/opencode/opencode.json` 没有 `mcp` 段，于是该 map 被丢弃。
- **判据**：五份 per-agent 配置中仅 fund-invest 有 `mcp: ['wuphf-office']`——它的文件是回归前写的，其余四份为空。
- **修复**：补一行写回，重编译重启。
- **验证**：三份配置立即带 `mcp`；forex-watch 自检 `team_broadcast` 成功（msg-1740），`read` 读到 headline 并逐字贴出。
- **教训**：症状像“进程卡住”，实际是配置生成漏写。先比配置差异，再动进程。

### 8.2 停用 routine 仍然会点火（2026-10-29）

- **现象**：`market-analyst` 已于 05:30 UTC 前后 PATCH 为 `enabled=false`，07:30:17 仍被 scheduler 触发（`/scheduler/.../runs` 显示 `triggered_by: scheduler`）。
- **根因**：`schedulerJobDue()` 只判断 `Status`（done/canceled）与 `DueAt/NextRun`，不判断 `Enabled`；PATCH 只改了标志位和状态字符串。
- **修复**：对 `agent_routine` 增加 `!Enabled` 直接返回 false，并对 `Status == "disabled"` 兜底；task/request follow-up 不受影响（它们创建时 Enabled 为空）。
- **验证**：`go build` 通过；行为验证待下一次到点。新增用例 `broker_scheduler_enabled_gate_test.go` 因测试包既有破损暂不可跑。

### 8.3 模型编造已公布事件结果（2026-10-29）

- **现象**：forex-watch 两次写出「FOMC 维持 3.75–4.00%，10–2 票（Cook、Hammack 主张加息），12 月定价 88%」；market-analyst 另写出「RBA 今日升息至 4.85%」「Warsh 称 Q3 GDP 可能超过 3%」。
- **判据**：上述字段在 digest 中逐字搜索 0 命中；删除日历里的“已过”行（幻觉钩子）后仍复现。
- **处置**：正文改由脚本渲染，聊天改为逐字复制 headline；模型不再撰写任何面向人的数字。
- **遗留**：预期差段落因缺少可信 actual 来源，保持占位。

### 8.4 “已转交”是假回执，forex 从未被唤醒（2026-10-29 20:30）

- **现象**：CoS 在 `cos__human` 回复「已转交 @forex-watch，等其回复」，forex 一直没动静。
- **判据**：`headless-codex-latency.log` 在 12:30 之后没有任何 `bot=forex-watch stage=started`；CoS 那轮 `final_chars=40`，无任何工具调用。
- **根因**：所有频道都是两两私聊，`cos__human` 只有 `['human','cos']`；在别人的 DM 里 @ 一个非成员 bot，broker 不会派发。
- **修复**：写入 policy 1（见 §5）。CoS 现在会真的开 `cos__forex-watch` 频道、发消息并回报 `msg-1794` 回执；forex 71 秒后在该频道作答。
- **附带发现**：派活修好后，forex 的*自由作答*仍在编造（把 Core PCE 预期写成 3.1%、加入德国 CPI / Tokyo CPI / KOF 等 digest 没有的事件，时间只写 UTC）。据此写入 policy 2，复测后回答收敛为 6 条 digest 事件 + 「digest 未收录」标注。

### 8.5 快照 routine 每天失败，真写入方是计划任务脚本（2026-10-31）

- **现象**：skill `daily-opencode-model-snapshot` 的 `last_execution_status` 持续 `failed`，但数据空间 `models_snapshot` 每天 08:00 都有新记录，来源不明。
- **根因**：`one-workflow:cos__human:daily-opencode-model-snapshot`（`workflow_provider: one`）每次执行都去读 `~/.wuphf/workflows/one/daily-opencode-model-snapshot.json`，该文件从未生成（`~/.wuphf/workflows` 与 `~/.wuphf/one` 不存在/为空）；实际写入是 Windows 计划任务 `WUPHF-ModelSnapshot` 跑 `~/.wuphf/data/model-snapshot.ps1`。
- **判据**：`created_at` 08:00:04 早于两条 routine 的 `last_run` 08:00:18；`created_by="model-snapshot"` 是脚本写库常量（无同名 bot）；`~/.wuphf/logs/model-snapshot.log` 有 `fetched N models` + `db: OK updated rec_…`；`grep -r daily-opencode-model-snapshot ~/.wuphf --include=*.json` 只命中 broker-state。
- **处置**：停用该 one_workflow（可逆）；`daily-0800-utc-…` agent_routine 的 payload 补「空值（new_models/removed_models 为空）写空字符串，不写"无"」，并把字段名从 schema 里不存在的 `published_at` 改为 `new_models`/`removed_models`；接口契约落 `RUNTIME-API.md`，硬规则落 `OFFICE-RULES.md` §5。
- **收尾（2026-10-31）**：skill `daily-opencode-model-snapshot` 已 archive（workflow 路径退役，真写入方仍是计划任务脚本）；scheduler job **没有 DELETE 端点**（`/scheduler/{slug}` 只支持 PATCH），所以只能 disabled，并由 `expected-scheduler.json` 的 `requires_file` 把关 —— 谁重新启用它，drift 检查立刻变红直到补上定义文件。
- **教训**：①「有产出」≠「那条 routine 在产出」，先对时间戳与 actor 再改 prompt；②`workflow_provider` 类 routine 失败是静默的，只在 skill 的 `last_execution_status` 留一行，巡检必看；③可编辑的 prompt ≠ 生效的 prompt，同一产出最多只有一条链路在真写。

### 8.8 同任务中途到达的唤醒被丢弃（2026-11-01 演练发现并修复）

- **现象**：ACMER-2795 演练。人工 `action=approve` 后任务停在 `running`，cos 却又在频道里出了一份计划（`msg-2799`）而非开工；直到人工在 `#cos__human` 催一次才继续执行。
- **日志证据**：`17:19:08 queue-drop: lead already handling same task`，此后 8 分钟没有任何 `stage=started`，而该任务的 `task_follow_up` 下一次在 60 分钟后（`WUPHF_TASK_FOLLOWUP_MINUTES` 默认 60）。
- **机制**：`enqueueHeadlessCodexTurnRecord` 的同任务去重分支在"该任务已有活动 turn"时直接 `return`，丢掉的正是 approve 产生的那次唤醒；活动 turn 的 prompt 是状态变化**之前**渲染的（它还看到 `lifecycle=planning`），所以它照旧出计划。等它结束，没有任何东西把新状态重新投递。
- **决策**：不并发跑同一任务这条保留；把被吞的唤醒挂到活动 turn 的 `Rerun` 槽（`headlessCodexActiveTurn.Rerun`），`finishHeadlessTurn` 结束时重放一次。
  - 只保留最新一条（后到的覆盖先到的），因为唤醒的 prompt 是在入队时渲染的，最新的那条就是最新状态。
  - 只重放一层：重放本身若再被吞就丢弃（`headlessCodexTurn.FromRerun`），否则 busy lane 会自我唤醒成环。
  - 重放前任务已终态（`done`/`canceled`/`archived`/`rejected`）则不重放，免得多跑一轮打已完成的工作。
- **验证**：
  - 单测 `TestEnqueueHeadlessCodexTurnRecordReplaysDroppedWakeAfterTurnFinishes`、`TestReplayedWakeIsNotHeldForAnotherReplay`（`internal/team/headless_codex_test.go`）。
  - 线上 ACMER-2809：17:55:26 连续两条 `queue-drop`，17:56:46 在**没有人工消息**的情况下自启回合，17:58:45 `status=done`、`verification pass=true`（`ra_79401b5e356910e9`，8157 字节），`hollow_completion` 未置位。
- **附带观察**：该回合第一次 `complete` 带的是**预判**的 artifact id，文件尚不存在，被 `wiki_artifact` 校验打回（`pass=false`），下一回合真正创建后才通过——门禁按设计挡住了"先声明后交付"。
- **加固（同日 18:12）**：只查"文件存在且非空"仍然可以拿旧页面冒充交付。`wiki_artifact` 现在追加一条**新鲜度**判定：文件 mtime 必须 ≥ 该 task 的 `created_at`（5 分钟容差，吸收时钟偏差与粗粒度时间戳），否则 detail 写 `was last written …, before this task existed` 并 fail。env 用 `taskVerificationEnv.Since` 传入，空值关闭（单测与无 task 的调用方不受影响）。
  - 单元测试 `TestRunTaskVerification_WikiArtifactFreshness`（旧文件 fail / 新文件 pass / 1 分钟偏差 pass）。
  - 线上冒烟 ACMER-2816：旧页面 `team/drills/core-loop-2026-11-01.md`（09:33 写、task 10:12 建）→ HTTP 409 + 上述文案；随即写入 `team/drills/freshness-smoke.md` 再 complete → 200、`pass=true`（108 字节）。

### 8.9 停滞任务自动退出轮转（2026-11-02）

- **现象**：`task-skill-2923`（meeting-prep）累计执行 **42 次，42 次 aborted**，输入逐字相同，全部卡在缺 `object_name`/`meeting_date`。cos 在第 25 轮的 notebook 里已经写了「需要的决策（人类，二选一）」，之后又烧了 17 轮——它写在自己的 notebook 里，而人不看 notebook。
- **为什么之前没有闸门**：唤醒类 job（`task_follow_up` / `recheck`）每次触发都只做"再投递一次"，从不问"这件事已经试过几次了"。`task.Ledger` 本来能答，但它 `taskLedgerMaxEntries = 20` 封顶并轮转，第 42 轮看到的窗口和第 21 轮完全一样。
- **判据不选"次数"，选"连续无产出"**：次数会误杀正常的长任务（一个持续产出改动的任务可以跑很多轮）。真正的信号是 `recordTaskLedgerEntry` 早就写下的那句 `ok (no durable trace: no message, no task mutation)`——它由 broker 可观测事实（Said/actions 皆空、无 error）拼出，不是模型自述。
- **机制**：`teamTask.StalledTurns` 不封顶、不随 journal 轮转。每轮结束：`noDurableTraceOutcome` 则 +1，任何有产出的轮次归零；达到 `maxStalledTurns = 8` 时由 `recordTaskLedgerEntry` 调 `CancelTask`。
  - **为什么是 8**：半小时一次的 follow-up 下等于两个工作日，小时级等于一天。门槛是「不会再有人补这个输入」，不是「bot 这轮发挥不好」。42 轮已经充分，8 已经宽松。
  - **为什么用 canceled 而不是 blocked**：blocked 是可恢复态，`scheduleTaskLifecycleLocked` 会立刻给它排 `recheck`，等于换个名字继续跑（这正是 §7j 实测过的坑）。canceled 是终态，清空四个调度字段并把 scheduler job 置 `done`。
  - 取消理由写进 `Details`，每个 packet 都会渲染，所以下一个读者不必翻 notebook 就知道为什么停。
- **验证**：
  - `TestEmptyTurnsAccumulateAndTripAfterThreshold`（1..7 不触发，第 8 次触发）、`TestProductiveTurnResetsTheStreak`（有产出归零，之后从 1 重新计）、`TestStallCounterSurvivesLedgerRotation`（跑 2× 上限，journal 轮转到 20 但计数保持 40）、`TestLauncherAutoCancelsAStalledTask`（走 launcher 真实路径，任务变 `canceled`、调度清空、原因可读）。
  - 反向验证：把 escalation 短路后 `TestLauncherAutoCancelsAStalledTask` 变红，报 `status="in_progress" stalled=8`。

## 9. 与作者 core loop 的对齐度（2026-10-29 21:00 评估）

依据：`docs/specs/core-loop.md` 九步循环、`ARCHITECTURE.md` 三条承重设计、`internal/bot/packs.go` 团队组合、`docs/specs/operator-agent-routines.md`。

| 作者的设计 | 当前实现 | 判定 |
|---|---|---|
| 1 建任务 | `agent_routine` 到点由引擎建 task（`created_by=system`），task 分发即唤醒；人类问答仍不建 | ✅（routine 不再绕开 task） |
| 2 结构化定义（goal/deliverables/success criteria） | routine prompt 是固定模板，无逐次定义 | 未实现 |
| 3 取上下文 + 工具 | MCP 修复后 cos 可见 42 个 team_* 工具 | ✅ |
| 4 单一 owner | analyst 停用，forex 唯一外汇 owner | ✅ |
| 5 执行 + 私有 KG / notebook | 四个 `/notebook/*` 路由已注册（2026-11-01）：`context_capture` 写 `agents/<slug>/notebook/*.md` 并 git 提交、`context_lookup` 可跨书架检索、`context_promote` 起 human 审批卡 | ✅（晋升仍需人工点头，符合原设计） |
| 6 交付 wiki artifact | `wiki/artifacts/forex/<date>.md` 已生成、已进 git，并新增 `team/forex-brief-<date>.md` 摘要进入索引 | ✅ |
| 7 确定性 learning hook（蒸馏） | 对 task 生效（wiki 有 `learning: pitfall/...`、`archivist: entity article` 提交）；routine 简报经 `wiki_artifact` 机器校验后同样进蒸馏 | ✅（2026-11-01） |
| 8 skills / policies 激活 | 两条 policy 已注入并实测（CoS 贴出原文、派活带 `msg-1794` 回执）；存在 `team/skills/daily-opencode-model-snapshot.md` | ✅ |
| 9 检索 + 去重 | 摘要文章 `team/forex-brief-<date>.md` 已进 `index/all.md`（`grep` 命中）；完整表格仍在 `artifacts/`，需按链接打开 | 部分 → 可用 |

### 9.1 刻意偏离（保留）

**去掉 bot 的 `bash` / `webfetch`**。作者给角色的 `AllowedTools` 是按能力白名单（如 `Bash(go*,git*)`），我为了止血一刀切禁用。方向安全（阻止 bot 探 broker 鉴权、编造网页数字），但比设计更严。稳定后应按角色恢复白名单，而不是永久禁用。

### 9.2 过渡妥协（待替换）

**用私聊派活**。policy 1 让 CoS 新建 `cos__forex-watch` 发请求，因为当前构建里这是唯一能唤醒对方的路径；但 `core-loop.md` R6 明确要删除“task channel 之外的 DM”。正解是改走 `team_task`，让 7→8 步（蒸馏→skill）真正跑起来。

### 9.3 结论

执行层与规则层已对齐作者（artifact、确定性渲染、policy 注入、单一 owner）；**知识层 2026-11-01 起补上了第一块**——私有 notebook 路由已通（写入/检索/晋升卡），artifact 也已进索引；routine 现已产生 task 并带机器校验，“蒸馏→skill”对每日简报不再空转（证据见 `TODO-NEXT.md` P2 #5b）。

### 9.4 收敛路径（按性价比）

1. ~~修 `/notebook/catalog`（或先关 `context_capture`，别让“记住了”是假的）——打通第 5 步。~~ **已做（2026-11-01）**：见 §5 第 5 步与 `RUNTIME-API.md` §6。
2. ~~把简报挂到 `team_task` 上（routine 只负责触发），让第 7 步有输入。~~ **已做（2026-11-01）**：见 §5 第 1、7 步与 `RUNTIME-API.md` §2。
3. artifact 写完后触发 wiki index 重建，让第 9 步可检索。
4. 按角色恢复 `bash` 白名单（第 3 步的最小权限版）。

### 9.5 复核（2026-11-07）：九步表当年是按"机器就绪"打的分

触发：最小配置试验（`MINIMAL-CONFIG.md`）跑完 5 天后，回头对账时发现 §9 那一列 ✅ 有一半是"代码路径存在"，不是"人看见交付"。按 §2 原则 6（规则落机制、不落记忆），这里把判定标准钉死，并留下一条机检门——否则下一次仍会拿"机制在"当"循环在跑"。

**新证据（2026-11-02 → 11-07）**

| 观察 | 数据 |
|---|---|
| 人从 composer 起的有定义任务 | **0** |
| 任务目录总数 / 其中带 decision packet | 53 / 44（最新 `ACMER-3344`，2026-11-06 19:43） |
| 最新任务是谁造的 | 调度器（job `5` 外汇简报），不是人——卡在"机器校验要求本任务期间生成新产物，而 bot 无 shell"，只能请宿主机跑 `forex-bridge.py` |
| 人为一条任务造成的中止 | `task-skill-2923`（meeting-prep）41 次 aborted，输入逐字相同 |
| 技能链路的输入 | wiki 散文（74 条缓存命中），不是任务交付物 |

**九步按新标准重判（只列有变化的）**

| 步骤 | §9 原判 | 现在能说什么 |
|---|---|---|
| 1 建任务 | ✅「routine 不再绕开 task」 | 机械上成立；**人给的入口 0 次**。ICP rubric 的标准是"composer → 任务存在且立刻看清"，这一步没有观测样本 |
| 2 结构化定义 | 未实现 | 仍未实现；routine 模板的定义不算，因为没人审过它是否足够。**2026-11-07 实测更正（见 §9.7）**：办公室会把 composer 任务转成定义（goal + success criteria，`defined_at` 有值），缺的是 `artifact_path`，不是定义本身 |
| 3 采访 / 权限 | ✅「cos 可见 42 个 team_* 工具」 | 工具可见 ≠ 采访可达。唯一真实采访是 bot 发起的，落在散文输入的 403 死路上（41 次 abort） |
| 6 交付 artifact | ✅ | 对调度任务成立（已拦下一次过期产物）；对人起的任务无样本 |
| 7 学习钩子 | ✅ | 钩子在跑，但输入是散文。按作者定义（任务交付物 → 知识 → 技能），这条链的**入口没有开过** |
| 5 notebook | ✅ | `wiki/agents/{cos,data-keeper,fund-invest}/notebook` 存在；最近没有由人工任务驱动的写入 |

**由此固化的两条规则**

1. **判"做到了"必须有一个人类可见的交付 + 一条可复现命令。** 代码里存在路径、路由已注册、测试通过，只算「机器就绪」，不能记 ✅。判定语言沿用 `docs/specs/core-loop-icp-eval.md` 的表头——「What "delivered" looks like to the human」。
2. **新 job / routine 必须在 `expected-scheduler.json` 里声明 `loop_step`**（服务核心循环 1–9 步，或 `support`）。没有声明 = drift 检查器报红。作者的 doctrine 是"不支持核心循环的都是浪费"，这条门把它变成每次加 job 时的必答题，而不是事后讨论。

**下一步（唯一在排期的验证）**

按作者 Journey 1 跑一条人给的、带 Goal / Deliverables+格式 / Success criteria 的任务，逐段按 rubric 记分，结果落 `docs/office/core-loop-runs/<date>.md`。跑通之前不再加场景——包括 `USE-CASES-2026-11-01.md` 里的 A2/A3。

### 9.6 专家评审（2026-11-07，五个视角）

目的是回答「两周内能不能拿出可复现证据证明这套办公室能干什么」。五份意见独立给出，下面是压缩版；原始长答不作为文档维护对象。

| 视角 | 结论 | 最硬的一条依据 |
|---|---|---|
| **产品推广** | 只能对「要本地可审计定时管线」的技术型创始人讲；撤回「多 bot 争论并交付」 | 今天可复现的三条：`forex-bridge.py` → wiki 产物、`check-office-drift.ps1` → `DRIFT: none`、`WUPHF_RUNTIME_HOME` 隔离复现 |
| **多智能体系统** | 只修一处：技能 `Inputs` 结构化 + 创建时校验 | `createSkillRunTaskLocked` 之前没有输入校验 → 散文 Inputs 照进队 → 403 问不到 → 30 分钟重放 = 41 次 abort，一处可截断两处 |
| **红队** | 两周内**不该推广** | 20 次是文件数不是简报数；分析段全 `待补`；唯一审阅样本是 87 字节演练件；`ACMER-3344` 的验收门结构性不可满足（产物 mtime 必须晚于任务创建，而 bot 无 shell） |
| **评估方法** | 现行 rubric 有 7 处可事后解释，必须预注册 + 硬失败条件 | 5 级行为锚、依赖阻断、A/B/C 证据分级、总分 = median(9 步)；触发 F1–F6 任一即判「循环不成立」 |
| **运维** | 用 `workspace` 隔离，别手设 `WUPHF_RUNTIME_HOME` | 端口池 791x 起、按端口取 token、Demo 必须钉 markdown；成本上限 12 turn / 8 分钟 / $0.05 |

**复核：专家说法里被实测推翻或补强的部分**

- `workspace create --from-scratch NAME` 实测可用（:7910 / :7911），但 **flag 必须写在子命令后、名字前**；写成 `create NAME --from-scratch` 会报 `unexpected extra argument`。`shred` 同理（`shred --yes NAME`）。
- 新 workspace 的 `/health` 实测是 `provider=claude-code`、`provider_model=""`、`memory_backend=gbrain`、`members=1`。所以「钉 markdown」不是保险而是**必需**（gbrain 下 `/wiki/write*` 一律 503，见 TODO K），provider 也必须显式指定；新建办公室还需要走一次 onboarding 才有团队。
- `/api-token` 返回的是 JSON `{token, broker_url, analytics}`，不是裸 token 字符串。
- 红队的两条硬纠正确认：`artifacts/forex/` 的 20 是**文件数**（10 天 × 2），窗口内只有 5 个交易日 / 5 份 `team/forex-brief-*.md`；每天的分析段 `待补` 各命中 1 次。审阅样本 `ACMER-2847` 审的确实是 `team/drills/review-target-3.md`（87 字节）。
- 读侧信号：窗口内 69 条 `reader=web, is_agent=false` 无法区分真人浏览与宿主工具，**消费仍未测量**；要测就得给读请求加调用方身份。
- `~/.wuphf-spaces/registry.json` 里有一条测试残留：`acme-revops`，`state=running` 但 `broker_port=0`、`runtime_home` 指向 `%TEMP%` 下已不存在的目录。`workspace list` 会把它当运行中。跑 Demo 前先 `workspace doctor` 清掉，否则脚本会误判基线。

**由此定下的四件事**

1. **预注册 + 硬失败条件**（评估方法那份的 F1–F6）作为 Journey 1 的前置：先 commit `core-loop-runs/<date>-prereg.md`，再建任务。
2. **证据分级**：A 机器可验（`verification.pass` + git hash + mtime 关系）> B 文件在盘 > C 截图口述；C 不能单独支撑 ≥4 分。
3. **对外口径收窄**：不宣称「AI 分析」——现有产物是脚本骨架；可宣称的是「定时、可审计、会被机检拒绝的交付管线」。
4. **`Inputs` 结构化排在 Journey 1 之后**：它修的是技能调用链，不是循环入口；但它是「协作层还救不救」的前置。

**未决（要人拍板）**：Journey 1 是否本周排（约 1–2 小时人工 + 新建 workspace 的 onboarding）；以及红队主张的「读侧信号」要不要先补（给 `/wiki/read*` 带调用方身份）。

### 9.7 Journey 1 实测结论（2026-11-07 14:08，隔离办公室 `journey1`）

预注册 `core-loop-runs/2026-11-07/prereg.md`，原始观察同目录 `observations.md`，评分卡同目录 `scorecard.md`（独立判分代理按九步表打分）。隔离办公室：broker `:7910` / web `:7911`，5 个 bot（cos / revops / analyst / ae / reviewer），`memory_backend=markdown`。人从 composer 建任务 `JOURN-22`，cos 拆 `JOURN-24` 给 ae。

**总分 6 / 10**（v1 基线 5/10，升 1 分）；F2–F6 全部未命中（任务未 `done`，`Verified` 未翻绿，没有 fake-done）。

| 步骤 | 分 | 判 | 依据 |
|---|---|---|---|
| 1 创建任务 | 4 | delivered | `created_at 05:40:54Z`，首读差 38 s；扣分：DM 而非共享频道 |
| 2 定义清楚 | 3 | partially | `definition.goal + 4 success_criteria`（`defined_at 05:43:24Z`），但无 `artifact_path`、bot 未复述三要素 |
| 3 上下文与采访 | 2 | broken | 无合并采访（`Dana Whitfield` 等问答未吸收），前置笔记有 |
| 4 拉起团队 | 4 | delivered | `JOURN-24 parent_issue_id=JOURN-22`、`owner=ae`、`members=5`；只拆 1 个子任务 |
| 5 专注执行 | 3 | partially | `context_used` 有 `wiki:team/getting-started/*` + 1 条 learning；缺业务页检索 |
| 6 交付物 | 3 | partially | 4 篇 wiki 落库可读（`cf96d59/78c00e8/328f6c3/9a7f6f6`）；3 封邮件仍 notebook-only；done-post 无链接 |
| 7 学习钩子 | 2 | broken | playbook 编译（`d808541`）；**无实体页、无 `[[wikilink]]`、无 infobox**；`source_capture` 仍调 gbrain 报 EOF |
| 8 技能与策略 | 3 | partially | 技能 `renewal-outreach` 编译并派给 5 个 bot（`0b2f18e`）；无 `policies` 落盘 |
| 9 复用去重 | 2 | broken/unproven | 无第二个相似任务，更新路径未验证 |

**两个结构性问题（本轮最值钱的发现）**

1. **人给任务 → wiki 的授权往返。** `Definition` 有 goal/criteria，但 `artifact_path` 为空，`/wiki/write-task` 在 `13:44:23` 拒绝（`wiki_handlers.go:160`）；服务端 `/wiki/write` 无授权检查，而「必须引用人写消息」只在 prompt 层（`prompt_builder.go:466`）。composer 的任务消息是 `from: system`，不能当授权锚点，于是 bot 只能索要一句人话。人回 `save to wiki` 后 66 秒内 4 篇全部落库。**是设计摩擦，不是模型不作为。**
2. **回合结束但任务未收口时无重投。** ae 在 13:49 的回合产出草稿后未收口，broker 不重派，两个任务以 `running` 空转到 14:58 的 `recheck_at`（56 分钟）。看门狗只把卡住的 bot 状态重置成 idle，不重投任务。

**对 §9.5 判据的实证支持**：同一套系统里，「文件在盘 + git sha + mtime」让 4 篇工件可当场验证；而「分析质量」「采访是否发生」这类没有落盘信号的步骤只能给 2–3 分。九步表按「人可见交付」打分是对的。

**预先声明的排除项有误**：§3 排除项 1 说「任务模型没有 Definition 字段」，实测办公室会把 composer 散文转成结构化 Definition（`defined_at 05:43:24Z`）。预注册不改；更正记在 `observations.md` 14:14 段。真正缺的是 `artifact_path`。**同样更正 §9.5 表格第 2 行**（「结构化定义 未实现」）：定义会被写入，缺的是产物范围声明。

**下一步（按影响排序）**

1. 打通授权路径（P0）：`broker_intake.go:818` 填 `Definition.ArtifactPath`，或对 `created_by=human` 的任务在 `wiki_handlers.go:160` 豁免；另一条是把 composer 的 `issue_created` 记为 `human` 或补写授权锚点。
2. 补「回合结束未收口」的重投（P1）：`scheduler.go` 在 `recheck_at` 之外加 2 个 poll 周期的兜底。
3. 邮件 wiki 化 + 实体页/`[[wikilink]]` + done-post 带链接（P2）：`entity_index.go`、`MessageRenderer.tsx`。

### 8.5 巡检发现：覆盖、索引、倒计时（2026-10-30 05:10 UTC）

- **Medium 事件被静默丢弃**：`MEDIUM_KEEP` 只匹配央行类关键词，导致德国 10 月初值 CPI（07:29 UTC）、德国 GDP（09:00）、美国 ECI（12:30）、加拿大 GDP 全部从日历消失。修复后本周事件 24 → 30 条，当日未来事件从 1 条恢复为 4 条。
- **artifact 游离在 git 之外**：bridge 只写文件，`git status` 里 `artifacts/forex/*` 长期是 `??`/`M`，索引只收录 `team/`，所以简报永远检索不到。修复：bridge 内置 `git_commit()` 提交三份产物，并额外写 `team/forex-brief-<date>.md` 摘要（已出现在 `index/all.md`）。验证：`git log` 出现 `artifact: forex brief 2026-10-30`，工作区仅余 `.reads/reads.jsonl`。
- **倒计时会过期**：顶部“还有 N 小时”是生成时刻的快照；已加「统计于 HH:MM UTC」，读旧文件不会再误判。
- **遗留**：ACMER-1742 已于 10-30 19:20 批准 plan（`lifecycle: running`）；task-skill-1852/1857 仍是零消息的 in_progress，无操作员取消端点，暂记录。

### 8.6 调度器静默停摆 → 外部看门狗 + 心跳日志（2026-10-30）

- **现象**：05:14 UTC 重启后，scheduler 循环未再派发任何任务；`/scheduler?due_only=true` 积压 **12 个 overdue**，07:30 analyst、08:00/08:30/09:00 三条 routine 全部错过。19:15 重启后积压一次性补发，forex 用 08:19 UTC 的旧 headline 出了份“已过期”的简报。
- **根因**：未定位。启动路径确认走 `LaunchWeb`（日志 `Launching WUPHF Office web view...`），该路径 `go l.watchdogSchedulerLoop()` **没有 panic 恢复**；`processOnce` 与 `run` 也无任何日志，停摆时不留痕迹。
- **加固（代码）**：
  1. `scheduler.run()` 增加 `loop started` / `dispatched N due job(s)` / 每 10 分钟 `heartbeat` / `loop exited` 四类日志 —— 下次停摆日志会直接断在某一刻。
  2. `processOnce()` 返回派发条数，循环据此打点。
  3. `launcher_web.go` 的调度 goroutine 补 `recoverPanicTo("watchdogSchedulerLoop")`，与 boot 路径一致。
- **加固（外部）**：`~/.wuphf/data/broker-watchdog.ps1` + 计划任务 `WUPHF-BrokerWatchdog`（每 15 分钟）：`due_only` 里有 job 超期 >10 分钟就重启 broker；20 分钟内已重启过则跳过（防循环）。日志 `~/.wuphf/logs/watchdog.log`，健康时不写。
- **验证**：`scheduler: loop started (poll=20s)` 于 19:22:53 出现；看门狗手动跑一次 rc=0、无日志（健康）；任务已注册，下次 19:36。
- **未验证**：看门狗的 kill/restart 分支（没有安全的制造超期方法）；重启命令本身当天已重复使用 6 次以上。

#### 8.6.1 代码审计（子代理，2026-10-30）

排查结论把“静默停摆”的候选原因收窄到一种：

| 假设 | 判定 | 依据 |
|---|---|---|
| 进程崩溃 | 排除 | PID 未变、`panics.log` 无新条目 |
| `Stop()` 被调用 | 排除 | 生产仅两处调用：`launcher_drain.go:54`（admin pause，随后 `os.Exit`）与 `launcher_session.go:61`（Kill，随后 `broker.Stop`）——两者都会让 broker 一起消失 |
| panic 被 recover 吃掉 | 部分成立 | web 路径的 notify 三条（`launcher_web.go:151-153`）与 headless 路径四条（`:200-204`）**没有**外层 recover，panic 会整进程崩；`watchdogSchedulerLoop` 自本次起才有 recover |
| **processOnce 内某次 dispatch 永久阻塞** | **最可能** | 现象吻合：整形 broker 正常（消息能唤醒、CoS 能建 interview），只有 tick 不再发生；重启后 12 个 overdue 一次性补发 |

判读口径（写进巡检）：heartbeat 断流 **且** 没有 `scheduler: dispatched N due job(s)` 行 ⇒ tick 内阻塞；两者都没有且进程已换 PID ⇒ 曾崩溃。

附带确认：`prompt_builder.go:256` 的“跨 bot 用 team_task、DM 内 @ 不唤醒”只写在 focus-mode 的 lead 块，specialist 块没有；即便有，10-29 的实测也证明 prompt 约束不如 policy。

### 8.7 消息入库但不唤醒（2026-10-30 05:11）

- **现象**：向 `forex-watch__human` POST 一条 @ 消息，接口返回 `msg-1862`，`broker-state.json` 里也能查到，但 `headless-codex-latency.log` 没有对应的 `stage=started`——bot 从未启动。
- **对比**：同一 broker 进程在 12:44 UTC（前一日）还能正常唤醒；此后 office 空闲约 16 小时。
- **处置**：重启 broker（PID 44800 → 44636）后重发，`stage=started` 立即出现，18.5 秒完成并逐字贴出 headline。
- **判断口径**：human 消息发出 2 分钟内，日志无 `stage=started` ⇒ 视为唤醒链路掉线，重启 broker 恢复；重启后 token 轮换，需重读 `%TEMP%\wuphf-broker-token-7892`。
- **根因**：未定位（疑似 notify/dispatch 循环在长时间空闲后退出）。这是已知间歇故障，重启是恢复手段，不是修复。

## 10. 外部调研：多智能体能跑到哪（2026-11-01）

调研目的是回答"这套办公室适合干什么"，来源为三篇一手工程文与一组行业数据。结论与我们当天演练的观察一致，因此不入待办，只作设计判据。

**来源、关键数字、局限与推导链在 `docs/office/RESEARCH-2026-11-01-multi-agent.md`**（本文件只留判据；数字改动只改那份）。一句话摘要（口径都在那份里，别转引成普适结论）：广度研究上多智能体比单 agent 高 90.2%（Opus 4 lead + Sonnet 4 subagents vs 单 Opus 4），但 token 是 15x；并行写手会因隐性决策碎片化；**同模型**换干净上下文审自己写的 PR，平均每 PR 抓 2 个 bug、58% 属严重。

由上述证据固化四条设计判据（都是本办公室已经在做的事，记录下来免得被"看起来更酷的架构"带偏）：

1. **一个任务一个写手**。任何并行都必须发生在写操作之外（只读子代理、审阅者）。`owner` 语义不是官僚主义，是防碎片化。
2. **干净上下文的审阅回合是 15x token 最划算的用法**。写手自审几乎无效（context rot）；换一个只拿到 artifact + 验收标准的 bot 去挑问题，才是外部验证过的收益点。
   **已落地（2026-11-01，`internal/team/task_review.go`）**：任务通过机器校验转 done 后自动开审阅任务，owner 取非作者（`WUPHF_TASK_REVIEWER` → lead → 首个非作者），交付 `team/reviews/<任务ID>.md`，审阅自己也要过 `wiki_artifact`。
   判据落地方式见 `RUNTIME-API.md` §1.2；写授权走 `POST /wiki/write-task`（任务声明的 `artifact_path` 即授权范围），不引入人类逐次批准。
   实测：`ACMER-2847`（fund-invest 交付 3 行速览）→ 审阅任务 `ACMER-2852`（cos）逐条判 0/3 通过、列 2 条阻断；`ACMER-2893` → `ACMER-2899` 通过新写入口落 `team/reviews/ACMER-2893.md`（5551 字节，`verification.pass=true`）。
3. **子代理把产出写文件、只回传引用**，避免传话失真。`artifact_path` + `wiki_artifact`（含新鲜度）就是这条的实现；不要为了"汇总方便"让子代理把全文塞回主 agent。
4. **能写死的判据优先**：`command` > `wiki_artifact` > 人眼。判据写不死的任务，进不了无人值守，只该有人在场跑。

反面清单（有证据支持"别做"）：并行写同一份 wiki/同一条任务；无结构 swarm 协商；无人值守放开外发与 canonical 写入；用多 agent 硬啃长程编码。

## 11. 外部调研：产物形态决定能不能自证（2026-11-02）

调研对象是上游产品：gawkbot 是 Grok Bot 的开源实现，官方有 56 个用例，社区有一个 32 帖、1588 次浏览的 bug 线程。结论直接改了我们选场景的判据。

**来源**：[x.ai/bot/use-cases](https://x.ai/bot/use-cases)（56 个用例）、[x.ai/bot/guides/grok-bot-101](https://x.ai/bot/guides/grok-bot-101)（xAI DevRel 的 Matt Palmer）、[forum.cursor.com/t/.../170358](https://forum.cursor.com/t/grok-bot-routines-dont-auto-run-on-schedule/170358)（32 帖，含 xAI 工程师的根因说明）。

### 11.1 一条外部证据，抵得上十次内部推理

xAI 工程师在原帖的解释，把「跑没跑」这个问题拆成了两种完全不同的观感：

> Your 15:00 routine looks healthy by comparison because **it writes to a file instead of messaging you**, so there is nothing missing to notice there.

同一套调度器、同样的延迟，**写文件的看起来健康，发消息的看起来失败**。不是执行差异，是「没有消息」这个信号无法区分「没跑」「跑了没事」「发了没看」。

### 11.2 由此固化的判据

**产物必须能在引擎静默时被独立验证。**

| 产物形态 | 引擎静默时能否判断 | 处置 |
|---|---|---|
| 写文件（含「无事」也落一份空文件） | 能：文件在/不在、行数、日期连续性 | 允许无人值守 |
| 内部状态（可通过 API 查询） | 能：`GET` 一次即知 | 允许无人值守 |
| 发消息 | 不能：三种情况观测相同 | 只能作附产物，不能作唯一产物 |

推论：一个任务如果**只有消息**没有文件，就不该进调度。

### 11.3 社区实测出来的参数，与我们的取值一致

同一线程里 Parag Chhibber 用探针实测后给的口径：

> We now treat daily jobs as "started on time" if they begin within ~1 hour of the slot, and only worry after ~2 hours. **Tight @every Nm intervals amplify the jitter and are a poor health check.**

我们在 2026-11-02 给 drift 定的过期阈值同样是 2 小时（`scripts/office/check-office-drift.ps1` 第 9 项）。两条独立路径得出同一个数，说明这个数量级不是拍脑袋。

同一帖里还有一条与我们「md 是源，JSONL 是派生物」同构的判断：

> When the UI history is empty we treat `automations/*/runs.json` as source of truth.

### 11.4 官方用例的两条设计约束

1. **"own a repeatable outcome, not a loose category of questions"** —— 判死了「每天看一眼」型任务。
2. 每个 starter prompt 都以**禁止外发**结尾（`Return a review list; don't send.` / `Don't change budgets.` / `Don't contact anyone.`）——产出必须是**待人处置的东西**，不是供人阅读的汇报。

56 个用例里唯一同型于我们简报的是 `Daily Briefing Writer`，被放在 General 类边缘位置，没有 starter prompt 细节。信息型产物在这套系统里是边角，不是主线。

### 11.5 对现有系统的审查结论（2026-11-02）

最小配置下 8 个 enabled job：7 个系统 job 的产物是内部状态（API 可查），1 个业务 routine（`5`）的产物是 `wiki/team/forex-brief-YYYY-MM-DD.md` **加**一条聊天消息。

**文件那一半让它可以被验证**——这是它比「纯消息型」安全的地方。但「有没有人读」是另一个问题，判据管不了，只能靠 2026-11-09 的试验结论。

### 11.6 与 §10 的关系

§10 回答「多智能体能跑到哪」（架构层面），本节回答「什么样的产物值得交给它」（选型层面）。两者不冲突：§10 说审阅回合要换干净上下文，本节说审阅的**产物**必须落盘。
