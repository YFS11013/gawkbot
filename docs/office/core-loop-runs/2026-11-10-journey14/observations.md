# Journey 14 observations（原始观察，不写结论）

被测 `6caf6cf` + 自愈路径修复（见 `git log`）。隔离办公室 `journey14`（broker :7910 / web :7911），home `C:\Users\ireke\.wuphf-spaces\journey14`。prereg 见同目录 `prereg.md`。所有时间为 UTC，来源逐行注明。

| 时间 | 发生了什么 | 来源 |
|---|---|---|

| 10:03:03 | 操作员 `POST /tasks`（`action=create`）建 Issue `JOURN-1`，正文钉死形状：**两份独立文件 + 必须拆两条子任务**（J13 的教训，见 prereg §1） | 本机请求 200、`tasks` |

| 10:07 前后 | cos 挂出 `request-3`（事实采访，`provide_now/interview/placeholders`）与计划稿 msg-5；计划明确「两份各自独立交付物」 | `requests`、`messages` |
| 10:10:16 | **干预 1**：答 `request-3`（`provide_now` + 死数字，同 J13）；随后 `request-8`（`plan-approval:JOURN-1`）挂出 | `/requests/answer` 200 |
| 10:13:08 | **干预 2**：批准 `request-8` | `/requests/answer` 200 |
| 10:13–10:16 | cos 拆出**两条 lane**：`JOURN-12`（一页报价，msg-13）、`JOURN-14`（跟进邮件，msg-15），父 `JOURN-1` 保持 `running` | `messages`、`tasks` |
| 10:15:59 / 10:16:03 | 两份文件落 wiki：`8ab0cc2`（报价 36 行）、`135f9dc`（邮件 22 行），作者均为 cos | wiki `git log` |
| 10:16 前后 | 两条 lane 各自 `submit_for_review`：`artifact` 记上各自路径，并各自挂出 `decision:` 卡（`request-24`/`request-27`）——**lane 有 artifact 时卡片会出现**（J12 的 lane 因 artifact 为空而无卡） | `tasks`、`requests` |
| 10:19:14 | **干预 3**：人工点两张 lane 卡的 `approve_decision` → **两张都 409**：`task JOURN-12 definition-of-done check failed (artifact): no non-empty artifact matches "team/accounts/acme-renewal-quote.md"`（JOURN-14 同） | 本机请求 409 |
| 10:23:34 | lane 的 `verification_result` 记为 `pass=false`，`kind=artifact`、`spec=team/accounts/acme-renewal-quote.md` | `tasks` |

| 10:34 前后 | 409 之后 cos 重新提交两条 lane：新卡 `request-37`/`request-40`（仍是 `decision:JOURN-12/14`）；并挂出 `request-48`（`definition-gap:JOURN-12`，`access needed: wiki write access`） | `requests` |
| 10:34 前后 | cos 自己诊断出问题（msg-50）：**「校验卡在 artifact（工作区文件）上，而交付物是 wiki 文章。改用 define 把校验方式修正为 none。」** 但 `define` 的 `task.Verification != nil` 分支静默拒绝了替换，lane 的 `verification` 仍是 `artifact` | `messages`、`tasks` |
| 10:38:57 | 停机前状态：三条任务 `in_progress`（`JOURN-12`/`JOURN-14` 的 `verification_result.pass=false` 仍挂在 `artifact` 上），三张卡未答 | `broker-state.json` |

| 11:2x | **修复二未生效**(旧二进制):两 lane 仍 `in_progress/review`、`verification_result.pass=false`;卡片 `request-37`/`request-40` 待决。 |
| 11:28 | 用新二进制重启 journey14 broker(`pause --force` + `resume`),并 `POST /tasks` 两次 `define`(`verification_kind=wiki_artifact`)-> 均 200,两 lane 的门被替换。**修复二真机生效。** |
| 11:28:26 | 人工批准两张 lane 卡 -> 均 **409** `no artifact_path on the completion: pass the wiki-relative path of the deliverable` —— 卡本身不带 `artifact_path`,而 `wiki_artifact` 校验只读请求体。**新发现:lane 自己记录过的 artifact 不被校验采用**,lane 仍不可关闭。 |

## 待补

- 停机（被 DoD 缺陷卡死）、归档、评分卡。
