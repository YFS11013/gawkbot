# OFFICE-RULES.md — gawkbot office 工作流与规则

适用：`~/.wuphf` 下的产出、排障、运维、routine / digest / 桥脚本 / policy 改动。
配套文档：`DESIGN-DECISIONS.md`（决策与事故）、`TODO-NEXT.md`（待办与验收）。

## 0. 三种任务，三条路径

| 类型 | 触发 | 走 |
|---|---|---|
| 巡检 | 定时、或“看一下现在怎么样” | §1 |
| 故障 | bot 不回 / 任务不跑 / 产出异常 / 消息对不上 | §2 |
| 变更 | 改代码、policy、routine、digest、权限 | §3 |

## 1. 巡检流（目标 5 分钟，只读）

按顺序取六项证据，缺一项不算巡检完成：

1. `curl 127.0.0.1:7892/health` — PID、`provider_model`、`memory_backend_ready`
2. `GET /scheduler?due_only=true` — 应为 0；**有积压 = 调度停摆**
3. `headless-codex-latency.log` 今日的 `stage=started` / `status=error`
4. digest 生成时间 + `wiki/team/alerts.md` 是否存在
5. `git -C ~/.wuphf/wiki status --porcelain` — 应只剩 `.reads/reads.jsonl`
6. `~/.wuphf/logs/watchdog.log` — 无文件即健康

输出格式固定：**结论一行 → 健康表（项/状态/证据）→ 问题（每条带判据）→ 待办**。

## 2. 排障流（先取证，后动手）

**第一步，定位到层**：调度 / 唤醒 / 模型 / 配置 / 数据。五层的判据互斥，别混：

| 层 | 判据（满足才动手） |
|---|---|
| 唤醒 | 消息已在 `broker-state.json`，2 分钟内 latency 无 `stage=started` |
| 调度 | `due_only` 有积压，或 `broker.err.log` 的 `scheduler: heartbeat` 断流 |
| 模型 | `opencode run --model … "只回复OK"` 超时，或日志 `AI_APICallError` |
| 配置 | per-agent `opencode.<slug>.json` 的实际内容 ≠ 期望（看进程真正读的那份） |
| 数据 | digest 时间/条数异常，或 `alerts.md` 有内容 |

**第二步，处置按代价从低到高**：加日志 → 改配置/策略 → 重启 → 改代码。
**第三步，重启后必须验三件事**：人消息能唤醒、heartbeat 出现、token 重读（`%TEMP%\wuphf-broker-token-7892`）。
**第四步，记录**：`DESIGN-DECISIONS.md` §8 加事故条目，`TODO-NEXT.md` 更新状态。

## 3. 变更流

1. **先分类**：确定性的事（取数、时区、渲染、校验）给脚本；需要判断的事（预期差、取舍）才给模型。
2. **规则放哪，按强度排序**：`policy`（每次 turn 注入）＞ 权限 deny/allow ＞ 代码 ＞ prompt。prompt 写过不等于生效——2026-10-29 已证伪一次。
3. **改完必须**：重编译/重启 → 读回产物 → 贴出关键片段。退出码 0 不算验证。
4. **数字类产出**：正文由脚本渲染，模型只做「读文件 → 逐字复制」。不允许模型自由撰写百分比、票数、概率。

## 4. 硬规则

- **R1 记忆分层**：账本是 md（`docs/office/` 两份 + `~/.wuphf/wiki/**`，git 可 diff）；`recall`/`retain` 只放触发点与指针；一条事实只写一处；写完必须回读验证。
- **R2 证据制**：文件:行号、带日期的日志时间、进程命令行。禁用“看起来/应该/可能”。
- **R3 不先杀进程**：卡死先对基线（当前单轮 18–60 秒，首字节可到 2 分钟）；未超基线禁止 kill。
- **R4 同轮归档**：行为有变化 → 同轮更新 `DESIGN-DECISIONS.md` 与 `TODO-NEXT.md`。
- **R5 机制优先**：结论要落 policy / 权限 / 代码 / 脚本 / 计划任务；只写文档或记忆视为未完成。
- **R6 补丁清单**：本地二进制含 4 处补丁，换官方版本或重拉源码后必须重打（清单见 `DESIGN-DECISIONS.md` §7）。

## 5. 禁止清单

- 用“再加一个 bot”解决质量问题。
- 免费额度/推理模型跑生产简报（前者会硬停，后者把 `<think>` 推到频道）。
- 让模型自由撰写已公布事件的结果（会凭先验编 FOMC 票数与概率）。
- 把开发文档、决策、待办写进 `~/.wuphf/wiki`（那是 bot 的运行数据）。
- 直接编辑 `broker-state.json` 当接口（broker 内存持有，会被覆盖）。
- 跨 DM `@` 转交，或说“已转交”却没有 `task id` / `message id` 回执。
