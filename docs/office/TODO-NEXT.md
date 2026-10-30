# 待办清单（2026-10-29 19:40 北京）

> 与 `DESIGN-DECISIONS.md` 配套。每条都有验收标准，不用“看起来正常”结案。
> 时区：**UTC 为主，括号内为北京时间**。

## 已完成 — 工作流与规则重梳（2026-10-30 20:20）
- `AGENTS.md` 重写为唯一工作流入口：三种任务→三条路径（巡检/排障/变更）、五层判据、四条处置原则、六条硬规则、六条禁止项。
- `RULES.md` 修正过时基线（单轮 10–15 分钟 → 18–60 秒）并指向 `AGENTS.md`。
- `DESIGN-DECISIONS.md` §5 改为“机制清单”，不再重复流程。

## P0-0 — 唤醒/调度链路
- ✅ 2026-10-30 19:30：外部看门狗已上线（`WUPHF-BrokerWatchdog`，每 15 分钟，超期 >10 分钟重启，20 分钟防抖）。
- ✅ 调度器已加心跳日志：`grep scheduler ~/.wuphf/logs/broker.err.log`，正常应每 10 分钟一行 `heartbeat`；断流即停摆。
- 每日巡检动作：`tail -3 ~/.wuphf/logs/watchdog.log`（无文件=健康）+ 上面那条心跳 grep。
- 未验证：看门狗的 kill/restart 分支。首次真实触发时优先核对 `broker.log` 是否重新出现 `Loop started`。
- 判据：human 消息 2 分钟内日志无 `stage=started`，且消息已在 `broker-state.json` ⇒ 唤醒掉线。
- 处置：重启 broker（命令见附录），重读 token，重发。
- 复发记录：2026-10-30 05:11（空闲 16 小时后首次唤醒失败）。
- 待办：定位 notify/dispatch 循环退出的原因（上游级）。

## P0 — 明早必须盯的两次自动运行

### 1. forex-watch 08:30 UTC（16:30 北京）自动简报
- **要验证三件事同时成立**：
  1. per-agent 配置含 `mcp: ['wuphf-office']`（MCP 修复未回退）
  2. turn 正常结束，无 `exit status 0xffffffff`
  3. 聊天内容与 `wiki/artifacts/forex/<date>-headline.md` **逐字一致**
- **验收命令**：
  - `grep "bot=forex-watch" ~/.wuphf/logs/headless-codex-latency.log | tail -2`
  - `cat ~/.wuphf/wiki/artifacts/forex/<date>-headline.md`
  - 对比同一时刻 `forex-watch__human` 的最后一条消息
- **失败时**：抓当轮 opencode 原始 stdout（`~/.wuphf/logs/headless-codex-forex-watch.log` 的 `opencode_result` / `error` 行），不要先杀进程。

### 2. market-analyst 07:30 UTC（15:30 北京）是否被停用门禁拦住
- 该 routine 已 `enabled=false`，但 10-29 07:30 仍点火（`schedulerJobDue` 不看 Enabled）。
- **验收**：`GET /scheduler/market-analyst-daily-direction-rating-after-forex-brief/runs` 最新一条不是 `2026-10-30T07:30`。
- 若仍触发：查 `kind` 是否被规范化成别的值（门禁只对 `agent_routine` 生效）。

## P1 — 需要决定的两件事

### 3. ~~branch-tracer 工具缺文件~~ ✅ 已完成（2026-10-29 19:39）
- 定性：omp-evolve A/B harness 的实验脚手架（分支执行追踪器），opencode 自动加载 `tools/*.ts`，无配置引用 → 可删。
- 处置：`branch-tracer.ts` 与 `branch-tracer-stub.ts` 一并移到 `~/.config/opencode/tools.disabled-20261029/`（可回滚，未硬删）。
- 验证：19:41 cos turn 12 秒成功、`team_*` 工具齐全；`Cannot find module` 最后一条停留在 16:56:54（移动之前）。
- 同族清理：`tools/tsc-analyzer.ts` 已于 21:45 一并移入同一目录（子代理复核：omp-evolve 脚手架）。
- 保留：`tools/probe-bg.ps1`——`.ps1` 不被 opencode 当 tool 加载，属惰性文件。

### 3b. ~~CoS 假转交~~ ✅ 已完成（20:50）
- policy 1 已写入（scope: cos）：必须真派发 + 回执。验证：CoS 新建 `cos__forex-watch` 频道、发 msg-1794、forex 71 秒作答。
- policy 2 已写入（scope: forex-watch / market-analyst）：只许转述当日 artifact，缺数据写「digest 未收录」。
- 残留：forex 复述时把 Core PCE 的前值/预期漏成「未单独列出」（digest 里其实有 0.2% → 0.3%），属保守方向的错误，暂不处理。

### 4. market-analyst 的去留
- 现状：routine 已停用；今天 07:30 那份含三处编造（Cook 反对票 / RBA 今日升息 4.85% / Warsh GDP 表态）。
- 三个选项：
  - A：删成员（最干净，forex 已覆盖事件层）
  - B：保留成员、彻底不产出（人工点名才跑）
  - C：改成“只读 headline + 只写技术位，不做方向判断”（改动最大，收益最小）
- 建议 A 或 B。

## P2 — 数据缺口（决定“预期差”能不能恢复）

### 5. 已公布事件的实际值来源
- 现状：digest 只有前值/预期，没有 actual；预期差段在 artifact 里是占位。
- 候选来源（按可靠性排序）：
  1. 央行/统计局文本（一手，但每个国家一条抓取逻辑）
  2. FF 发布后页面的 actual 字段（需要新的解析路径，XML 不提供）
  3. 新闻标题里的数字（当前模型就用这个拼，正是单位混用的来源）
- **验收**：digest 里出现 `actual` 字段，且与一手来源逐条对齐 ≥ 3 个事件。

### 5b. 简报挂到 team_task（打通蒸馏）
- 现状：routine 直接产出 artifact，无 task → core loop 第 7 步（蒸馏→skill）对简报空转。
- 做法：routine 只负责触发，由 CoS 或引擎建 task（owner=forex-watch），artifact 作为交付物挂回 task。
- **验收**：一次运行后 `git -C ~/.wuphf/wiki log` 出现该 task 的 learning/archivist 提交。

### 5c. ~~artifact 进索引~~ ✅ 已完成（10-30 05:10）
- 做法：bridge 增加 `git_commit()`；额外输出 `team/forex-brief-<date>.md` 摘要（索引只扫 `team/`）。
- 验收：`git log` 有 `artifact: forex brief 2026-10-30`；`grep forex-brief ~/.wuphf/wiki/index/all.md` 命中。
- 备注：完整表格仍在 `artifacts/`，索引指向摘要文章，双份不同用途。

### 6. `/notebook/catalog` 404（上游 bug）
- `internal/teammcp/context_tools.go:491` 调了 broker 未注册的路由，导致 bot 的 `context_capture` 写进去也读不回来——“记忆”是黑洞。
- 处置：补路由（上游 PR）或先关闭 `context_capture`，避免制造“记住了”的假象。

## P3 — 工程债

### 7. `internal/team` 测试包编不过
- `headless_session_typing_test.go` 缺 `sessionAnswer` / `pollUntil`；新增用例 `broker_scheduler_enabled_gate_test.go` 因此跑不了。
- **验收**：`go test ./internal/team -run 'Scheduler|Opencode' -count=1` 能编译并绿。

### 7b. 三条僵尸任务（更新 10-30）
- ✅ ACMER-1742：plan 已批准（`lifecycle: running`），data-keeper 可收尾。
- 仍存：task-skill-1852 / 1857（in_progress，零消息）。

- ACMER-1742（planning, data-keeper）、task-skill-1852 / 1857（in_progress, cos，零消息）。
- 无操作员取消端点；交给 CoS 关闭可能再触发循环，暂记录。
- 若要清：让 CoS「取消 task-skill-1852/1857，无产出；1742 保留」并观察一轮。

### 8. 清理
- `YFS11013/gawkbot-office`（空仓库，创建后被 fork 方案取代）：删除需 `delete_repo` scope → `gh auth refresh -h github.com -s delete_repo && gh repo delete YFS11013/gawkbot-office --yes`，或网页删。
- `C:\Users\ireke\code\gawkbot\gawkbot-fix.exe.bak-20261029`（回滚点，确认稳定后再删）
- `C:\Users\ireke\code\gawkbot\gawkbot-fix.exe~`（更早的备份）

## 附：当前运行参数（避免下次再猜）

| 项 | 值 |
|---|---|
| 启动命令 | `gawkbot-fix.exe --broker-port 7892 --provider opencode --no-open` |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/beam-free`（用户级环境变量，已 `setx` 持久化）；兜底 `bear/deepseek-chat` |
| broker token | `%TEMP%\wuphf-broker-token-7892`，**每次重启都变** |
| 桥 | Windows 计划任务跑 `forex-bridge-run.vbs`，每 6 小时；健康时无 `wiki/team/alerts.md` |
| 简报产物 | `~/.wuphf/wiki/artifacts/forex/<date>.md` + `<date>-headline.md` |
| 常规检查 | `curl -s 127.0.0.1:7892/health`（含 model 与 members） |
