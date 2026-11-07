# Journey 2 — 预注册

写于 2026-11-07T08:50Z，**在任何 office 代码运行之前**。目的是让第二轮的判分不是事后叙事：下面每一条预测都写死了「什么算命中、什么算证伪」，跑完照单核对。

## 环境

| 项 | 值 |
|---|---|
| 工作区 | `journey2`（`workspace create --from-scratch`，`~/.wuphf-spaces/journey2`） |
| 端口 | broker `:7910`，web `:7911` |
| 二进制 | `gawkbot-fix.exe`，提交 `738c485`（含 §9.8 修复 + `task_retry` 独立 job 身份） |
| 记忆后端 | `markdown`（`config.json` 显式钉住） |
| 蓝图 | `ai-revops`，团队 `cos / revops / analyst / ae / reviewer`（与 Journey 1 同构） |
| 公司名 | `Nex`（同 Journey 1） |
| 主办公室 | 不参与判定，`:7891/:7892` 照常运行 |

## 场景

与 Journey 1 完全相同的业务简报，以便逐条对比：

> We have three renewals coming up: Acme Corp (Q3, $48k, champion left in May), Brightline (Q3, $22k, usage up 40%), and Corti Labs (Q4, $61k, two unresolved support escalations). Draft a tailored renewal email for each, capture a per-account brief on the wiki, and write a renewal-outreach playbook we can reuse every quarter.

提交方式同 Journey 1：`POST /tasks` 以 `created_by=human`、`owner=cos`、`task_type=issue`、频道 `cos__human` 建立父 Issue，然后不再给人写任何指令。

## 环境修正（场景开始前，均未产生判定用流量）

两次作废的 bring-up 都发生在**场景任务提交之前**，不计入任何一步的判定：

1. 16:47 第一次 onboarding 用自定 slug 列表，结果被合成蓝图的 starter 名册判为 stale，装上了 `cos/gtm-lead/founding-engineer/pm/designer`，与 Journey 1 不同构 → shred 重来，改带 `blueprint=ai-revops`。
2. 16:49 第二次装好后，`config.json` 是在 broker **启动之后**才写的，导致 launcher 按默认值选了 `claude-code` 通道，NEX-2 的回合连续以 `API Error: 401 Invalid API key` 收尾（cos 在 DM 里贴了 6 次同样的报错）。17:00 起改成「先写 `config.json` 再重启 broker」，`/health` 显示 `provider=opencode, model=opencode-go/beam-free` 之后才重新提交任务。

修正后的环境与上表一致；记在这里是为了让第二轮的时间线可审计，而不是让 401 回合混进核心循环的分数。

## 预测（跑完逐条判定）

| # | 预测 | 命中判据 | 证伪条件 |
|---|---|---|---|
| P1 | cos 在 `define` 时声明产物路径 | 父 Issue 的 `definition.artifact_path` 非空，且落在 `team/emails/`、`team/accounts/` 或 `team/playbooks/` 下 | `definition` 里没有路径，或路径在白名单外被拒后 cos 不重试 |
| P2 | 子任务落在 creator⇄owner 的 pair DM | ae 的子 Issue `channel == "cos__ae"` | 仍是 `cos__human` 或别的单人 DM |
| P3 | owner 在自己的任务频道里能写 | ae 在子 Issue 上留下 comment 或完成事件，headless 日志无 `channel access denied` / `403` | 出现 403，或 ae 只能用 `no durable trace` 收尾 |
| P4 | 人不需要补 `save to wiki` | cos/ae 直接 `/wiki/write-task` 成功，`team/playbooks/…md` 与 `team/emails/…md` 由 bot 写出 | 需要人回一句 `save to wiki` 才写出 |
| P5 | 空转回合被短重投接住 | 出现 `kind=task_retry` 的 scheduler job，`NextRun` 距回合结束 ≤ 3 分钟，且触发一次后转 `done` | 空转后仍要等 ≥ 30 分钟才有下次唤醒，或 retry job 反复重放 |
| P6 | 端到端有可核对的人可见交付 | wiki 里存在三封邮件草稿 + 每账户 brief + 一份 playbook，且父 Issue 走到 `done`/`approved` | 交付只存在于 `agents/*/notebook/`，或父 Issue 停在 `running` |

补充约束：

- **场景开始后人工干预预算 0**（Journey 1 用掉了 approve ×1 和补参 ×1）。只有在产品自己弹审批卡且不点就走不下去时，才允许点一次，并记入 observations。
- **豁免：安装期审批 ≤ 2 次**。onboarding 自带的 wiki orientation 在场景任务提交前弹了两个建频道审批（`request-2` #hygiene、`request-5` #pipeline）。这是安装动作，不是交付判定，允许点 Approve；两次都记入 observations。
- 判定标准沿用 `docs/specs/core-loop.md` 九步记分，基线为 Journey 1 的 6/10（步骤分 4/3/2/4/3/3/2/3/2）。
- 只读证据：broker-state.json 快照、headless 日志、`/tasks` 与 `/scheduler` 响应、wiki 目录树。

## 已知不预测的部分

- 邮件不会真的发出去（本轮没有接 SMTP）；判定 F2/F3 看「草稿写到 wiki 并被父 Issue 引用」。
- composer 卡片仍标 `from: system`（§9.8 明确不改）。
- `/wiki/write` 服务端仍不校验授权（同上）。
