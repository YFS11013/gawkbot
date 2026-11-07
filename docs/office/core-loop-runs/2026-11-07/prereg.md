# Journey 1 预注册 — 2026-11-07

按 `docs/specs/core-loop-icp-eval.md` 的 Journey 1 原文与九步 rubric，在隔离办公室里跑一次**人给的任务**，逐段取证。判定标准见 `DESIGN-DECISIONS.md` §9.5：人可见交付 + 可复现命令；代码路径存在只算机器就绪。

## 1. 固定参数

| 项 | 值 |
|---|---|
| runtime_home | `%USERPROFILE%\.wuphf-spaces\journey1`（隔离 workspace，不是主办公室） |
| 端口 | broker `127.0.0.1:7910` / web `127.0.0.1:7911` |
| commit | `2e263f5`（被测 broker 由该 commit 构建） |
| provider / model | `opencode` / `opencode-go/beam-free` |
| memory_backend | `markdown`（gbrain 下 `/wiki/write*` 一律 503，故为必需而非保险） |
| 团队 | 蓝图 `ai-revops` 全量：`cos`(lead) · `revops` · `analyst` · `ae` · `reviewer` |
| 任务来源 | **人**在 `http://127.0.0.1:7911` 的 composer 发出，逐字文本见 §2 |

## 2. 任务原文（逐字，跑完不得改写）

> We have three renewals coming up: Acme Corp (Q3, $48k, champion left in May), Brightline (Q3, $22k, usage up 40%), and Corti Labs (Q4, $61k, two unresolved support escalations). Draft a tailored renewal email for each, capture a per-account brief on the wiki, and write a renewal-outreach playbook we can reuse every quarter.

## 3. 排除项（跑之前定，跑完不改）

1. **步骤 2「结构化 Definition」**：任务模型没有该字段（`TaskPostRequest` 只有 `Details` 散文 + `VerificationKind/Spec`），`definition.defined_at` 永远为空。改判「bot 是否在任务聊天里复述 Goal / Deliverables+格式 / Success criteria，并据此执行」。
2. **步骤 4 不排除**：本次团队 5 人，按 rubric 判 owner、子任务、可见依赖边。

既有残留：蓝图播种的 `ai-revops-2`（analyst，done/approved）留在办公室里作为历史，不作为评分对象。

## 4. 开跑前已发生的干预（逐条登记，不计入 F6）

| 时间 | 干预 | 原因 |
|---|---|---|
| 13:16 | onboarding 经 API 完成（`POST /onboarding/complete`，`agents` 显式列出 slug） | UI 点击需人；首轮传 `agents: []` 只种入 lead——`botSelectionFilter` 只在 `nil` 时返回「全量」，JSON 空数组是「只留 lead」，这个差异记录在产品侧 |
| 13:20 | 取消蓝图 starter lanes：`ai-revops-1/3/4`、`task-skill-4` | 避免与主任务争用同一批 bot 和免费额度，让「人给的任务」是唯一在跑的工作 |
| 13:20–13:24 | 修三处 Windows 运行缺陷并重建 broker：`1a9db10`（回合/校验开窗）、`32c1533`（脱离控制台的进程默认开窗）、`2e263f5`（DETACHED broker 被父 job 杀掉） | 运行中实测发现；非评分对象，但决定这次能不能稳定观察 |

## 5. 允许的人工干预（任务创建后）

- `approve` ×1、`补参` ×1（回答采访问题）。
- 其余任何触发、重放、挪文件 → 命中 F6。

## 6. 硬失败条件（触发任一即判「循环不成立」，停止调参）

| 编号 | 现象 |
|---|---|
| F2 交付未成立 | 任务 `done` 但无 wiki artifact，或 artifact 不可读 / `[REDACTED]` / 链接不在 done-post |
| F3 假完成 | `Verified` 通过而机器校验没过，或没有校验却 `done` |
| F4 自耗循环 | 同一输入连续 ≥3 次 aborted，或 `stalled=8` 被取消且无可读原因 |
| F5 采访/权限不可达 | 提问没有同时落「任务聊天 + Inbox」，或无任务前置 notebook 笔记 |
| F6 干预超限 | 超出 §5 的人工动作 |

## 7. 观察点与命令

```powershell
$TOK = Get-Content "$env:TEMP\wuphf-broker-token-7910"

# 任务列表与状态
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/tasks

# 单个任务（含 verification / 活动）
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/tasks/<id>

# wiki 产出与提交历史
git -C "$env:USERPROFILE\.wuphf-spaces\journey1\.wuphf\wiki" log --oneline -20
git -C "$env:USERPROFILE\.wuphf-spaces\journey1\.wuphf\wiki" show --stat HEAD

# bot 回合日志
Get-ChildItem "$env:USERPROFILE\.wuphf-spaces\journey1\.wuphf\logs" | Sort-Object LastWriteTime -Descending | Select-Object -First 5

# 健康与成员
curl -s http://127.0.0.1:7910/health
```

逐步要看的：

| 步骤 | 观察什么 |
|---|---|
| 1 创建任务 | composer 提交后任务立刻可见、有 owner 或明确的未分配态 |
| 2 定义清楚 | 是否有一次合并采访；bot 是否复述三要素（Goal / Deliverables+格式 / Success criteria） |
| 3 上下文采访 | 提问是否同时出现在任务聊天 + Inbox；回答是否被吸收 |
| 4 拉起团队 | owner 是否合理；有子任务时依赖边是否真实 |
| 5 专注执行 | 任务前置 notebook 笔记是否存在；回合活动是否显示 context 使用；跨回合是否失忆 |
| 6 交付物 | wiki artifact 是否生成、可读、链接出现在 done-post；`Verified` 与机器校验是否一致 |
| 7 学习钩子 | 实体文章（Acme / Brightline / Corti）是否有引用与互链；playbook 草稿是否出现 |
| 8 技能与策略 | playbook 是否编译成技能；能否从聊天反馈即时生成策略 |
| 9 复用去重 | （本次只跑 Journey 1，第 9 步以「是否更新而非重复」的初判记录） |

## 8. 判分安排

- 记录者：本实例。只写带时间戳的原始观察（`observations.md`），不写结论。
- 判分者：独立子代理，输入 = 本文件 + `observations.md` + 任务 JSON + wiki git log；按九步表 1–5 分并给证据引用。
- 证据分级：A 机器可验（`verification.pass=true` + git hash + artifact mtime ≥ task.created_at）/ B 文件在盘（路径 + 字节数）/ C 截图或口述（不能单独支撑 ≥4 分）。

## 9. 产出

`observations.md`、任务 JSON 快照、wiki `git log` 与 artifact 原文、`scorecard.md`、`delta-vs-v1.md`（对照 `.icp-eval/` 的 5/10 基线）。
