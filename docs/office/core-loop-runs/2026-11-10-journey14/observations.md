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

| 11:30 | 重启窗口内两条 lane 自动闭环:`done/approved`、`wiki_artifact` 校验 `pass=true`、artifact 已记录(`request-37`/`40` 由 answered 收口)。 |
| 11:43:32 | 停表前的快照:父 `JOURN-1` 进 **`in_progress/decision`, `artifact=null`**,卡片 `request-92`(`decision:JOURN-1`)待决 —— **聚合谓词第一次在真机上被触发**。 |
| 11:43:32 | 卡片正文(原文):「This Issue's lanes delivered 2 file(s): - team/accounts/acme-renewal-quote.md - team/emails/acme-renewal-followup.md / The Issue ships no single file of its own; those are what you are accepting.」—— 容器分支成立。 |
| 11:43:46 | **人工点 `approve_decision` -> 200 `{"ok":true}`**;父 `done/approved`, `artifact=null`, `artifact_not_required=null`, `hollow_completion` 缺省(=false)。请求记录 `answered={"choice_id":"approve_decision","answered_at":"2026-11-10T11:43:46Z"}`。 |
| 11:43:46 | **V3/V4/V5 成立**:父 Issue 经人类决策卡关闭,聚合交付被认(不 hollow),artifacts 仍留在两条 lane 上。 |

| 11:50 | 停表:三条任务全 `done/approved`(父 `artifact=null`、两 lane artifact 已记录且校验 `pass=true`);`requests` 仅剩两张 `definition-gap` 询问(`request-48`/`55`)未答。证据归档至 `artifacts/`(11 份),`journey14` 已 shred。 |
| 注 | **本轮经过两次二进制切换**:10:0x 起跑于 `5113926` 之前的二进制;因 lane 卡不可关闭的缺陷被发现,11:28 与 11:43 两次 `pause --force` + `resume` 换成含三处修复的构建。**父 Issue 的卡与批准(11:43:32-11:43:46)发生在第三个构建上**;lane 的 `define` 修正是操作员干预。 |

## 待补

- 停机（被 DoD 缺陷卡死）、归档、评分卡。
