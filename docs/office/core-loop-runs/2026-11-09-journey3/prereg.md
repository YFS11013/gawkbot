# Journey 3 — 预注册

写于 **2026-11-09T00:57Z**（本地 UTC+8 08:57），在 journey3 工作区创建、任何 office 代码运行之前。本轮任务由记录者按「人给任务」的形状代拟（用户 2026-11-09 明确「Journey 3 的任务你替我处理」），文本一旦提交不再改写。

目的：验证 Journey 2 之后修掉的四类缺陷在**新隔离办公室**里是否真的成立，而不是只在新测试里成立。被测面：J2-1（done 产物门）、J2-2（审批卡不阻塞投递）、J2-3（取消撤销派活）、J2-4（recheck 拒绝时可见化 + 短重投）、J2-6（拒绝重做已交付），以及 K（wiki 与 memory backend 解耦）、I①（疑问句分流，纯前端，不在本轮回合里判）。

## 环境

| 项 | 值 |
|---|---|
| 工作区 | `journey3`（`workspace create --from-scratch`，shred 掉 paused 的 `journey2` 释放 7910/7911 后创建） |
| 端口 | broker `127.0.0.1:7910`，web `127.0.0.1:7911` |
| 二进制 | `gawkbot-fix.exe`，提交 `a3369a8`（含 J2-1/2/3/4/6、§9.8/§9.9/§9.10/§9.11、K 三层、I①） |
| provider / model | `opencode` / `opencode-go/crouton-alpha-free`（`WUPHF_OPENCODE_MODEL` 显式注入创建工作区的那个进程；`llm_provider_priority=["opencode"]` 由主办公室配置种入） |
| memory_backend | `markdown`（由 `seedWorkspaceConfig` 从主办公室继承；创建后先复核 `config.json` 再提交任务） |
| 蓝图 | `ai-revops`，团队 `cos / revops / analyst / ae / reviewer`（与 Journey 1/2 同构，便于对比） |
| 公司名 | `Nex` |
| 主办公室 | 不参与判定，`:7891/:7892` 照常运行 |

## 场景（逐字，提交后不得改写）

以 `created_by=human`、`owner=cos`、`task_type=issue`、频道 `cos__human` 建父 Issue，details 逐字如下：

> I need a decision-ready package on entering the SMB segment with our renewal-retention offer.
>
> Goal: decide by next Monday whether we pitch SMB (10-50 seats), and hand the team everything needed to execute if we do.
>
> Deliverables (all in the team wiki):
> 1. Research brief at team/research/smb-retention-2026q4.md - market size sanity check, 3 pricing bands, 2 named competitors and how each positions retention, and a section "What we don't know" listing every assumption with how to verify it.
> 2. Reusable playbook at team/playbooks/smb-renewal-outreach.md - ICP, a 5-step outreach motion, the three objections we should expect with answers, and a one-page checklist an AE can execute without asking questions.
> 3. One outreach email at team/emails/smb-renewal-intro.md, under 180 words, written for a 20-seat SaaS company whose renewal is 60 days out.
> 4. One HTML visual artifact that puts the go/no-go recommendation on a single screen.
>
> Success criteria:
> - All four deliverables exist at those paths and are readable in the team wiki.
> - Every number is either sourced (name the source) or marked unknown; no invented statistics.
> - The recommendation states a clear go or no-go, the single biggest risk, and what evidence would change the answer.
> - You tell me the links in this conversation when done.
>
> Constraints: do not send any email. Do not ask me to approve individual files - use the team. If something is genuinely undecidable, ask me once, in one message, with options.

任务设计意图（不参与判分，写在这里是为了让判据可审计）：四份产物分别落在 `team/research/`、`team/playbooks/`、`team/emails/`（均在 `validateTaskArtifactScope` 白名单内）+ 一个 HTML 视觉工件；「不要让我逐个批文件」直接压 J2-2/§9.8 的授权路径；「数字要么有来源要么标 unknown」给第 6 步一个可机检的硬条件。

## 排除项（跑之前定，跑完不改）

1. **步骤 2 判「是否写入结构化 Definition」**（§9.8 后的新口径）：只看 `definition.goal/success_criteria/artifact_scope` 是否真实写入、`defined_at` 是否非空；不要求 bot 在聊天里复述。
2. **步骤 4 不强求多子任务**：要求单一 owner 合理；子任务数量只作为加分观察。
3. **步骤 9 无第二个相似任务**：以「是否出现重复 lane / `context_used` 是否出现」判，不做跨运行比较。
4. **I①（疑问句入口分流）** 是纯前端 `/tasks/new` 行为，本轮从 API 建单，不判分；只在本轮里顺带记录 API 建单是否仍然允许问句型任务（应当允许）。

## 硬失败条件（任一触发即判「循环不成立」）

| 编号 | 可观察现象 | 观测位置 |
|---|---|---|
| F2 交付未成立 | 任务 `done` 但无 wiki artifact，或 artifact 不可读 / 链接不在 done-post | `GET /tasks/{id}`、`os.Stat(wiki/…)` |
| F3 假完成 | `verification.pass=true` 而机器校验没过，或没有校验却 `done`，或 done 且 `hollow_completion=true` | `verification` 与 `status` 矛盾 |
| F4 自耗循环 | 同一输入连续 ≥3 次 aborted，或 `stalled` 被取消且无可读原因 | 任务 journal + 日志 |
| F5 采访/权限不可达 | 提问没有同时落「任务聊天 + Inbox」，或无任务前置 notebook 笔记 | `GET /tasks/{id}`、`wiki/agents/<owner>/notebook/` |
| F6 人工干预超限 | 除安装期审批 ≤2 次外，人还触发/重放/挪过文件 | 运行日志 |

## 允许的人工干预

- 安装期：onboarding 本身的建频道审批 ≤2 次（Journey 2 实测会弹 `#hygiene`/`#pipeline`），记为安装动作。
- 场景开始后：**0 次**。所有 pending request 保持 pending，判定就是「不点会怎样」。唯一例外：产品自己弹阻塞卡导致整条链路停摆且无任何自动出路时，允许点 1 次并记入 observations（仍计 F6 边界内的「一次」）。

## 预测（跑完逐条判定）

| # | 预测 | 命中判据 | 证伪条件 |
|---|---|---|---|
| P1 | cos 把任务写成结构化 Definition 并声明产物范围 | `definition.defined_at` 非空，`artifact_scope`/`artifact_path` 覆盖 `team/research/`、`team/playbooks/`、`team/emails/` 中至少两个 | `definition` 为 null，或路径在白名单外被拒后不重试 |
| P2 | 子 Issue 落在 creator⇄owner 的 pair DM | 子 Issue `channel ∈ {analyst__cos, ae__cos, revops__cos, reviewer__cos}`，且其回合日志无 `channel access denied` / 403 | 子 Issue 落 `cos__human`，或 owner 写自己任务频道 403 |
| P3 | done 不再与产物脱钩（J2-1） | 每条 `done` 的子 Issue 要么 `artifact_path` 存在且文件可读，要么 `verification=not_required` + `verification_reason` 非空；出现拒绝时是 `409 artifact_required` 且点名任务 | 出现 `done + hollow_completion=true`，或 done 的 `artifact_path` 指向不存在的文件 |
| P4 | 不需要人说「save to wiki」（§9.8 + K） | 四份产物由 bot 写入 wiki 并产生 git commit，写入过程无人类回话；父 Issue 的完成消息里带路径/链接 | 需要人回一句授权话才写出，或产物只落在 `agents/*/notebook/` |
| P5 | 审批卡不阻塞投递（J2-2） | 存在 pending request 的频道里，bot 回合仍有非空正文投递（`final_chars>0` 且 `messages.json` 有对应条目），无 `request pending in this channel` 造成的丢正文 | 再次出现 `status=ok final_chars=0` 且日志有该 409/409-like 错误 |
| P6 | 空转/拒绝唤醒可见化 + 短重投（J2-4） | 出现 `kind=task_retry` 的 job 且 `next_run` 距空转 ≤3 分钟；或出现 `kind=watchdog_alert` 的人可见消息并随后 `task_retry` | 空转后下一次唤醒 ≥30 分钟，或拒绝唤醒只写 signals 不写消息 |
| P7 | 不再重做已交付的活（J2-6，仅观察） | 若办公室尝试重建已 done 的同题 lane，`create` 返回 `conflict` 并点名已有任务 id 或 `allow_duplicate` | 出现同题同 owner 的新 id 且未说明 |

## 观察点与命令

```bash
TOK=$(cat "$TEMP/wuphf-broker-token-7910")
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/health
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/tasks
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/tasks/<id>
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/scheduler
curl -s -H "Authorization: Bearer $TOK" http://127.0.0.1:7910/requests
git -C ~/.wuphf-spaces/journey3/.wuphf/wiki log --oneline -20
ls ~/.wuphf-spaces/journey3/.wuphf/logs
```

快照节奏：任务提交后 2/6/12/20/30 分钟各一次 `snapshots/<HHMM>/`（tasks/messages/scheduler/requests/events/wiki tree/日志尾部），停表条件：父 Issue 终态，或连续 15 分钟无新消息且无 active 回合。

## 判分安排

- 记录者：本实例，只写带时间戳的原始观察（`observations.md`）。
- 判分者：独立子代理（`reviewer`），输入 = 本文件 + `observations.md` + 任务 JSON + wiki git log；按九步表给 1–5 分并引用证据。记录者不参与打分，只在判分后核对事实。
- 证据分级：A 机器可验（`verification.pass=true` + git hash + artifact mtime ≥ task.created_at）/ B 文件在盘 / C 口述（不能单独支撑 ≥4 分）。

## 产出

`observations.md`、`snapshots/`、任务 JSON、wiki `git log` 与产物原文、`scorecard.md`、`delta-vs-journey2.md`。
