# Core-loop run records

按作者的核心循环（`docs/specs/core-loop.md`，九步）逐段取证的运行记录。目录名 `<YYYY-MM-DD>/`，每次运行一个。

判定标准见 `docs/office/DESIGN-DECISIONS.md` §9.5：**人可见交付 + 可复现命令**；代码路径存在只算「机器就绪」。评分口径抄 `docs/specs/core-loop-icp-eval.md` 的逐步表。

**当前状态：0 次运行。**

## 1. 预注册（开跑前必须 commit）

跑之前先写 `prereg.md`，把下面这些钉死，跑完不许改：

```markdown
# Journey 1 预注册 <YYYY-MM-DD>
- runtime_home: <隔离 workspace 名>
- commit: <git rev-parse HEAD>
- provider / model: <…>
- 排除项: 见 §2（本次运行保留/新增了哪些）
- 允许的人工干预: approve ×1、补参 ×1，其余 >2 次即作废
- 判分人: 记录者 ≠ 判分者（各自独立写，跑完再对）
```

## 2. v1 声明的排除项（2026-11-07 定，跑之前写进 prereg）

两项在本次运行里**不按 rubric 判**，因为它们不是失效、是形状差异；不预先排除，跑完就成了事后解释：

| 步骤 | rubric 要的 | 为什么 v1 排除 | v1 的替代判据 |
|---|---|---|---|
| 2 结构化定义 | Goal · Deliverables+格式 · Success criteria 作为结构化 Definition 出现 | 任务模型没有该字段（`TaskPostRequest` 只有 `Details` 散文 + `VerificationKind/Spec`），`definition.defined_at` 永远为空 | bot 是否在任务聊天里**复述**三要素，并据此执行 |
| 4 拉起团队 | owner + 子任务 + 可见依赖边 | 全新办公室只有 1 个成员；且 S3 的结论本来就是写单线程 | 只要求单一 owner，不要求子任务 |

其余 F2–F6 一律按硬失败处理。

## 3. 硬失败条件（触发任一即判「循环不成立」，停止调参）

| 编号 | 可观察现象 | 观测位置 |
|---|---|---|
| F2 交付未成立 | 任务 `done` 但无 wiki artifact，或 artifact 不可读 / `[REDACTED]` / 链接不在 done-post | `GET /tasks/{id}`、`os.Stat(wiki/…)` |
| F3 假完成 | `Verified` 通过而机器校验没过，或没有校验却 `done` | `verification.pass` 与 `status` 矛盾 |
| F4 自耗循环 | 同一输入连续 ≥3 次 aborted，或 `stalled=8` 被取消且无可读原因 | 任务 journal + 日志 |
| F5 采访/权限不可达 | 提问没有同时落「任务聊天 + Inbox」，或无任务前置 notebook 笔记 | `GET /tasks/{id}`、`wiki/agents/<owner>/notebook/` |
| F6 人工干预超限 | 除各一次 approve / 补参外，人还触发/重放/挪过文件 | 运行日志 |

## 4. 证据分级

- **A（机器可验）**：`verification.pass=true` + git hash + artifact `mtime ≥ task.created_at`；
- **B（文件在盘）**：路径 + 字节数 + `hollow_completion=false`；
- **C（截图 / 口述）**：不能单独支撑 ≥4 分。

## 5. Runbook（2026-11-07 实测，改动前先重验）

```powershell
# 1) 起隔离办公室（端口池从 7910 起，自动分配）
./gawkbot-fix.exe workspace create --from-scratch <name>
#    输出：Ready (broker :7910, web :7911)
#    注意 flag 必须在名字前：`create <name> --from-scratch` 会报 unexpected extra argument

# 2) 取 token（/api-token 返回 JSON，不是裸字符串）
curl -s http://127.0.0.1:7911/api-token   # {analytics, broker_url, token}
Get-Content "$env:TEMP\wuphf-broker-token-7910"   # 同一个 token

# 3) 钉配置：新建办公室默认 provider=claude-code / model="" / memory_backend=gbrain / members=1
#    gbrain 下 /wiki/write* 一律 503（TODO K），所以 markdown 是必需不是保险
#    写 <space>/.wuphf/config.json: {"llm_provider":"opencode","memory_backend":"markdown"}
#    然后重启（见 4）

# 4) 重启的坑：`workspace pause` 的客户端 deadline 是 60s，优雅停机要 90s → 必然超时
#    回退：读 <space>/.wuphf/broker.pid，Stop-Process -Id <pid> -Force，再 workspace resume
#    清理：workspace shred --yes <name>   （`shred <name> --yes` 同样会被当成多余参数）

# 5) onboarding 还没走：/onboarding/state|prereqs|blueprints|complete 已注册，
#    但新办公室只有 1 个成员，团队要 onboarding 之后才有 —— 这一步需要人。
```

# 6) 坑（2026-11-07 新增）：watchdog 原本按**可执行文件名**杀进程
#    ~/.wuphf/data/broker-watchdog.ps1 的 Restart-Broker 会杀掉所有
#    gawkbot-fix.exe / opencode.exe ——包括隔离办公室。表现为 workspace resume
#    报成功、十几秒后 HTTP 000、broker.log 无任何错误。
#    修法：按主办公室端口（--broker-port 7892）取根进程，再按父子关系遍历后代一并杀。
#    验证：起隔离办公室 → 触发一次 watchdog 重启 → 两边 /health 都应 200。

**绝对不要用 `--workspace` 旗**：

```
$ ./gawkbot-fix.exe --workspace j1 --broker-port 7910 --web-port 7911 --no-open
warning: --workspace="j1" ignored: workspace orchestrator not wired (Lane B integration pending)
  Office already running at http://127.0.0.1:7891
```

这个构建里它是空操作，且会静默回落到**真办公室**。隔离只能靠 `workspace create` 起的那个进程 + 按端口说话。

## 6. 每次运行要留的证据

`prereg.md`、`observations.md`（带时间戳的原始观察，不写结论）、任务 JSON（含 `verification`）、wiki 的 `git log` 与 artifact 原文、日志片段、`scorecard.md`、`delta-vs-<上一次>.md`。

不留：broker 进程、workspace 目录（`shred` 会备份 30 天）。
