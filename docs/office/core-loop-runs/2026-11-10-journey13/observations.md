# Journey 13 observations（原始观察，不写结论）

被测 `6caf6cf`。隔离办公室 `journey13`（broker :7910 / web :7911），home `C:\Users\ireke\.wuphf-spaces\journey13`。prereg 见同目录 `prereg.md`。所有时间为 UTC，来源逐行注明。

| 时间 | 发生了什么 | 来源 |
|---|---|---|
| 08:49:57 | 操作员 `POST /tasks`（`action=create`）建 Issue `JOURN-1`，正文为重建的 Acme 场景（原始 J12 prompt 未落盘，见 prereg §2） | 本机请求 200、`tasks` |
| 08:52:20 | 重发种子（第一次漏了 `action=create`，返回 404 `task not found`）→ 200，`status=in_progress` `lifecycle_state=planning` | 本机请求 200 |
| 08:58 前后 | cos 回合启动；先抱怨采访卡「超时」两次（msg-7/msg-8），随后同时挂出 `request-4`（事实采访，`paste/placeholders/source`）与 `request-9`（`plan-approval:JOURN-1`） | `messages`、`requests` |
| 09:01:09 | **干预 1、2**：答 `request-4`（`paste` + 死数字：$18,000/年、2026-12-31 到期、$21,600/+20%、14 天有效、Dana=dana@acme.example、发件人 Alex）；批准 `request-9` | `/requests/answer` 200 ×2 |
| 09:01 之后 | `JOURN-1` → `running`，cos 回合在跑，尚未拆 lane | `tasks` |

| 09:05:26 | cos 挂出 `request-14`（`definition-gap:JOURN-1`），「缺失的细节」写的是 `access needed: 用户已提供的合同事实（方案/价格/周期/续约调整/联系人）`——**这些事实在 request-4 里已经给过** | `requests` |
| 09:05:36 | cos 把两份交付物写进**同一个文件** `team/accounts/acme-renewal.md`（wiki `367d955`，64 行），**没有拆任何子任务**（`tasks` 里只有 `JOURN-1`） | wiki `git log`、`tasks` |
| 09:06:17 | `JOURN-1` 自行 `submit_for_review`：`review/review`、`artifact=team/accounts/acme-renewal.md`；同时挂出 `request-22`（`decision:JOURN-1`），正文为单文件口径 `The deliverable is team/accounts/acme-renewal.md.` | `tasks`、`requests` |
| 09:12:33 | **干预 3**：答 `request-14`（`answer_directly`：「按已提供的事实继续即可」）；**干预 4**：点 `request-22` `approve_decision` | `/requests/answer` 200 ×2 |
| 09:12:33 | `JOURN-1` → `done/approved`，`artifact=team/accounts/acme-renewal.md`，`hollow_completion` 未置；wiki 落 `team/decisions/JOURN-1.md`（`29c8463`）与实体事实（`0ebe13d`、`f499f45`） | `tasks`、wiki `git log` |

## 待补

- 15 分钟静默后的正式停表、归档、评分卡、shred。
