# Journey 12 observations（原始观察，不写结论）

被测 `4c53083`。隔离办公室 `journey12`（broker :7910 / web :7911），home `C:\Users\ireke\.wuphf-spaces\journey12`。prereg 见同目录 `prereg.md`。所有时间是 UTC，来源为 `broker-state.json` / `logs/headless-codex-latency.log` / 本机请求返回，逐行注明读的是哪个文件。

| 时间 | 发生了什么 | 来源 |
|---|---|---|
| 07:31:20 | 操作员建 Issue `JOURN-1`（`task_type=issue`，频道 `cos__human`，`status=intake`），正文只给两份交付物 + 英文 + 「形状你自己定」 | `messages` msg-2 |
| 07:31:20 | cos 回合启动（`queue_wait_ms=51`） | latency log |
| 07:33–07:35 | cos 发事实采访 `request-4`（「Acme 的续约事实目前不在办公室里，你希望我怎么拿到」），同回合贴出计划稿 msg-6 | `requests`、`messages` |
| 07:39:30 | **干预 1**：操作员答 `request-4`，直接给死数字（$18,000/年、2026-12-31 到期、续约 $21,600/+20%、14 天有效、Dana = VP Ops） | `/requests/answer` 200 |
| 07:40:13 | 计划卡 `request-8`（`dedupe_key=plan-approval:JOURN-1`）**干预 2**：批准 | `/requests/answer` 200 |
| 07:40:13 | cos 回合启动并拆出两条 lane：`JOURN-13`（一页报价，msg-14）、`JOURN-15`（跟进邮件，msg-16）。**本轮没有 QA lane** | `messages`、`tasks` |
| 07:42 前后 | 三条任务全部 `running`（`JOURN-1` 父、`JOURN-13`、`JOURN-15`），`artifact` 均为空 | `broker-state.json` |

## 待补

- lane 交付、父 Issue 提交时的 409（若发生）、卡正文、点击、停表、归档。

| 07:45–07:47 | 两条 lane 被 cos 标成 `review/ready_for_review`，随后进 `decision`；**没有任何 `decision:` 卡出现**，`requests` 里只有两张 `knowledge-promote:` 卡 | `broker-state.json` |
| 07:47–07:55 | 草稿落在 wiki（`4ec73f5` 报价草稿、`f415bb2` 邮件草稿），两条 lane 的 `artifact` 仍为空 | wiki `git log`、`tasks` |
| 07:55:28 | **干预 3、4**（预算外）：批准两张 `knowledge-promote` 卡（`request-21`/`request-23`），文件按声明路径上线：`d9a7c79 team/accounts/acme-renewal-quote.md`、`f044319 team/emails/acme-renewal-followup.md` | `/requests/answer` 200、`messages` msg-32/34 |
| 07:59:38 | 两条 lane 变 `done/approved`，`artifact` 记上了各自路径；日志 `task review: no reviewer available for JOURN-13/JOURN-15 (owner "cos")`——本轮 lane 没有 reviewer | `tasks`、`broker.log` |
| 08:00:39 | **父 Issue 被 cos 自己关掉**：`status=done lifecycle_state=approved`、`artifact=null`、`artifact_not_required="not_required: Container issue: its two lanes JOURN-13 (…) and JOURN-15 (…) already hold the deliverables. (cos, 2026-11-10T08:00:39Z)"`、**`hollow_completion=true`**。全过程**没有 `decision:JOURN-1` 卡、没有人工点击** | `broker-state.json` |
| 08:00:39 | cos 的原话复述了守卫的结论：「**父任务是容器，不能引用子任务的产物**——用 not_required 重新关闭。JOURN-1 已关闭。」（msg-42） | `messages` |

## 观察到的事实链（不写结论，供评分卡判）

1. 容器守卫**确实生效**：cos 第一次带 lane 路径提交父 Issue 时被拒，它随后自述「父任务是容器，不能引用子任务的产物」。
2. 被拒之后它没有走 `submit_for_review`（无 artifact），而是用 `complete` + `verification=not_required` 从任务端点把父 Issue 关闭；该路径不产生决策卡，人类没有被问。
3. 关闭记录里 `hollow_completion=true`，尽管两条 lane 都已交付并在 wiki 上。
4. 两条 lane 自身从 `review` 进 `decision` 时也**没有卡**（它们的 `artifact` 当时仍为空，谓词判「没什么可批」），直到知识提升把文件落到 wiki、cos 自己把 lane 标为通过。
5. 本轮干预 4 次（事实采访、计划卡、两张知识提升卡），父 Issue 的决策卡一次都没有出现。

## 待补

- 停表（父 Issue 已终态、无未答 request）、归档、评分卡、shred。
