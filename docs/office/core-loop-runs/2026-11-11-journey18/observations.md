# Journey 18 现场记录（进行中）

## 时间线（+08:00）

| 时刻 | 事件 | 证据 |
|---|---|---|
| 15:34:23 | `OFFICE-1` 建单（owner `cos`，`created_by=human`），清单 22 条提交写进 details | `tasks[]` |
| 15:34:23–15:35:04 | 回合 1：`status=ok total_ms=40249 first_event_ms=12240` | latency log |
| 15:54:00 | 回合 2 起（`request-4` 计划审批卡在 15:56 前后升起） | latency log |
| 15:56:52 | 回合 2：`status=ok total_ms=171290`；产出计划 + 「等你批准」 | `headless-codex-cos.log` |
| 15:5x | 计划卡 `request-4`（`dedupe_key=plan-approval:OFFICE-1`），选项 `approve / approve_with_note / needs_more_info / reject / reject_with_steer` | broker-state |
| 15:54–15:56 | 计划里两个交付路径写成 `docs/office/Journey18-*.md`（**仓库内**） | card question |
| 15:56 后 | 操作员用 `approve_with_note` 批注：写进 wiki 的 `team/reviews/journey18-{internal,public}.md`，别写进仓库 | `actions[] request_answered` |
| 16:01:53 | 办公室回：「内容已按清单拟好，但还需你一句明确回复才能写入」 | `msg-12` |
| 16:03:24 | 又回一次：「已请求你再发一条明确消息『请写入 wiki』」 | `msg-15` |
| 16:0x | 采访卡 `request-10` `pending`：对外说明强调什么、哪些不提 | broker-state |

## 已暴露的问题（待本轮结束后进 TODO）

### O1 根因（17:4x 查到）：写 wiki 的授权是一条模型自己够不到的契约

查 `internal/teammcp/server_wiki_tools.go` 才看清楚「还差一步授权」不是模型胆小，是**工具契约真的要一条人类消息**：

- `team_wiki_write` 要么给 `human_request`（一条**人类本人发的**、12 小时内、且文本里有「直接写 wiki」意图的 broker 消息 ID），要么给 `task_id`（**任务自己声明的 `artifact_path` 就是授权**，且必须是 owner 写）。
- 我用 `/notifications/automation` 发的消息 `From=nex`，`isVerifiedHumanDelegationSender` 只认 `you` / `human` / `human:` → 被判「不是人类发的」；我批的 `approve_with_note` 是 request answer，也不是消息。
- 所以办公室两次向人要「再发一条新消息」，是**照着工具描述做的**（描述里明写「never mention broker internals or message IDs to them」，它确实没提 ID），但人回的字不进它够得着的通道里。
- 而且**它从没试过 `task_id` 这条路**：拆出的 `OFFICE-22/24` 各自声明了 `artifact_path`，按契约本可直接交付，模型却把第一次在父任务（无 `artifact_path`）上被拒的错误读成了「必须人来授权」。

这一轮的形状因此和原计划不同：**不是「人被空卡卡住」，而是「授权契约把人和机器人卡在两条通道上」**。人能改消息内容，机器人只认 `From=you/human` 的消息，而人这一侧没有可写的口子（broker 没有人类发消息的 HTTP 端点，只能走 web UI `:7911`）。

### O1' 表象：计划批准之后还要第二句「请写入」才肯动手

计划卡已经被人 `approve_with_note` 批准，批注里**明确写了两个 wiki 路径**，模型仍然把「写入」当成需要第二次授权的动作，先后两次让操作员再回一句「请把这两份内容写入办公室 wiki」。这正好是本轮要抓的形状：**一张不携带任何新信息的卡**——人除了重复上次的批准，没有别的可答。

区别于 `8026e40`（关单收卡）和 `6a3f137`（lane 不自升卡）：这条不是卡的生命周期错，而是模型把「已批准的写入」重新当成待批。要修的不是收卡，而是**计划批准即授予写入**这件事在提示层有没有写清楚，以及 `POST /wiki/write-task` 被拒时（`artifact_path_missing`）有没有把拒绝原因直接告诉模型而不是让它转述给人。

### O2 计划里的交付路径默认落在仓库目录

模型把两个交付写成 `docs/office/Journey18-*.md`，而 core loop 的第 6 步要求交付进 Wiki。人的批注纠正了它，但**这个纠正本该由任务定义给出**：`created_by=human` 的 Issue 没写 `artifact_path` 时，模型就按 cwd（仓库）猜。观察到的后果是：如果人没盯，文件会写进 git 工作树。

### O3 看门狗在计划批准后立刻报「没有子任务」

`watchdog_alert: Approved plan for Journey 18 … has no sub-tasks yet (@cos)` —— 计划刚批准、模型还在同一回合里准备拆 lane，看门狗已经先报了一次。无害，但会制造一次假警报；记下来看它是否每轮都响。

## 判据结果

| 编号 | 结果 | 证据 |
|---|---|---|
| V5 全部 hash 可追溯 | **通过** | 内部清单 23 行，与提交清单 23 条逐条对上；两页抽到的全部短 hash 都在清单内，无清单外 hash |
| V6 对外页不出现内部词、每条映射回内部页 | **部分通过** | 对外页 5 条，`lane/broker/headless/dedupe/queue/fallback/guard` 零命中；但第 5 条「开箱即用，零配置」在内部页 23 行里**找不到对应提交**——清单里没有任何一条改过默认配置或安装路径。**模型补了一条清单外的承诺**，正是 F-d 的软版本 |
| V1 无 lane 级决策卡 | **通过** | `requests[]` 只有 `plan-approval` 与 `interview` 两张，全程没有 `decision:*` |
| V3 关 lane 收卡（`8026e40`） | **未测** | 两条 lane 在操作员来得及关之前就自己走到了 `done`。这一项本轮没有证据，既不能算通过也不能算失败 |
| V4 Issue 卡列全两文件 | **通过** | `request-60`（`dedupe_key=decision:OFFICE-1`）问题里逐条列出 `team/reviews/journey18-internal.md`、`team/reviews/journey18-public.md`，并写明「Issue 自己不发文件，你接受的就是这两份」 |
| 守卫回归（J14） | **通过** | cos 试写 lane 的文件时被拒：`guard_refused: container artifact: team/reviews/journey18-internal.md is OFFICE-22's deliverable` |
| lane 自批准（J15 形状） | **形式变了，不再是空操作** | 两条 lane 走到 `done/approved`，但**没有自己的卡**（V1），人手上只剩容器卡一张可点——J15 那种「点了却落在已完成 lane 上」的空点击没有出现 |

## 尚未发生的（本轮要验的）

- V3 `8026e40`：本轮**没测到**——两条 lane 在操作员来得及关之前就自己跑完了，本轮形状下也**构造不出**这个窗口：`6a3f137` 的改动让 lane 在开着的 Issue 下根本不升卡，没有卡可收。改为引用既有 handler 级测试作为证据：`TestCompletingALaneClosesItsOrphanedDecisionCard`（`broker_request_task_close_test.go:353`，2026-11-12 复跑 `ok 0.228s`）钉住三件事——关掉的那条 lane 自己的卡变 `canceled` 且留 `request_canceled` 行、summary 含 `task closed`；兄弟 lane 的卡不受影响。'''

- V4：等 cos 给 OFFICE-1 升容器卡，看它是否把 `team/reviews/journey18-internal.md` 与 `team/reviews/journey18-public.md` 两条路径都写进问题里。

### O4 退回一次，模型报告「已修改」，文件一个字节没动

`request-60` 退回，理由是「对外页第 5 条没有来源」。办公室回：「已按反馈修改：对外说明删除第 5 条「开箱即用，零配置」，保留前 4 条；内部变更清单 23 行未动。」并重新升起 `request-72`。

实际状态：

| 检查项 | 结果 |
|---|---|
| `journey18-public.md` 字节数 | **724**，与退回前完全一致 |
| 第 5 条是否还在 | **还在**（`grep -c 零配置` = 1） |
| 文件 mtime | 18:20:53，退回发生在 00:10 UTC 之后 |
| wiki 提交 | `85e786a docs(reviews): add Journey 18 public notes`（18:20:53），此后再无该路径的提交 |

也就是说：**验收面（卡）在退回后重新升起，内容一字未变，而模型的自然语言报告说改完了**。操作员如果只读那句回报就点批准，批准的就是退回前那份东西。这比「空卡」更糟：卡至少不撒谎，这句回报会。

修法方向（待写进 TODO）：验收卡重升前必须比对交付物的实际字节/提交号与退回时是否不同；没变就不该重升，而是回到 owner 并说明「你没有写入」。
