# Journey 11 vs Journey 10（同一场景，同一模型）

被测代码：J10 = `52456d3`，J11 = `54ad984`（§9.22 的聚合谓词）。场景逐字相同（Acme 续约报价 + 跟进邮件），模型同为 `opencode-go/longcat-2.5-free`。J11 的判分见 `scorecard.md`，逐条证据在那里，这里只写**机制上的差异**。

## 1. 差异表

| 维度 | Journey 10 | Journey 11 |
|---|---|---|
| 父 Issue 进 `decision` | 02:54:20Z | 04:24:27Z |
| 父 Issue 的 `decision:` 卡 | **从不出现**（F-I）：`artifact=null`，旧谓词判「没东西可批」 | 04:24:51Z 出现（`request-83`），距进 decision 24 秒 |
| 卡正文 | —— | `The deliverable is team/accounts/acme-renewal-quote.md.`（**只一条**，见 §3） |
| 点击 | 人工探针补点，不带 `artifact_path` → 409 | 04:25:00 点 `approve_decision` → 200，04:25:06 `done/approved` |
| lane 进 `decision` | 0 条（`taskNeedsStructuredReview` 判假，交付即 `done/approved`） | 2 条（`JOURN-22`/`JOURN-24`），各一张 `decision:` 卡，各一次人工点击 |
| QA lane | 无（只拆 2 条 lane） | 有（`JOURN-26`，拆出时就带 reviewer）；报假阴性后被取消 |
| 干预次数 | 2 次（计划卡、探针补点各 1） | 6 次：语言采访、计划卡、两张 lane 卡、取消 QA、父 Issue 留言 |

## 2. 机制差异（为什么卡这次发了，而聚合分支没被锻炼）

J10 的父 Issue 进 `decision` 时 `artifact=null`，所以「有没有东西可批」只取决于 §9.22 新加的聚合谓词。J11 不同：cos 在 `submit_for_review` 时把 `team/accounts/acme-renewal-quote.md` 写成了 Issue 的 `artifact`，于是：

1. 旧谓词（只读自己的两个字段）同样会发卡——**这一轮的 V1 不能当作聚合谓词的证据**；
2. 卡先读 `task.Artifact`，退回 lane 列表的写法因此命中错误分支，正文只点名一条路径——V2 失败的真实成因；
3. `broker-state.json` 里 `JOURN-2.artifact` 全程是那条 lane 路径，`hollow_completion=null`。

聚合谓词的证据只有进程内一条：`TestDecisionCardRaisedForParentIssueWithDeliveredLanes`（`submit_for_review` → 无 reviewer → 收敛 → 卡 → approve，去掉谓词即 0 张卡、去掉门即 409）。

## 3. 本轮新出现、J10 没有的形状

- **lane 会停人**：两张 `decision:` 卡各需要一次点击。J10 判据里「lane 不停人」的假设在 office 模式 + 带 reviewer 的 lane 下不成立，点击预算要按 lane 数重算。
- **模型会声明做过检查并给出相反结论**：`JOURN-26` 的 `status=ok final_chars=945` 断言两份交付物「全库不存在」，同一时刻 `/wiki/read?reader=reviewer` 返 200；broker.log 里没有任何 `wiki read/search` 行（只有两条 `wiki write-task`），但 broker 本来就不记录读操作，因此这一条只作「模型行为」记录，不当代码缺陷。
- **人写不进 bot 私有频道**：`POST /tasks/JOURN-26/comment` → 500 `{"error":"failed to post message"}`；`/tasks/{id}/message` 不存在（404）。人只能改道父 Issue 的 `cos__human`。
- **cos 说「已转达」但没有任何回合启动**：`msg-75/76` 之后 `headless-codex-latency.log` 里没有 reviewer 的 `stage=started`，JOURN-26 一直 `in_progress`，直到被人工取消。

## 4. 判据本身的教训

J10 的教训是「把对象写成 lane 进 decision → 永远未观察」；J11 补上另一半：**判「交付物有没有被点名」时不能只读父任务字段**，模型会自己往里盖一条路径。判据要写明「列出 `parent_issue_id` 下所有已交付子任务的 `artifact`」，对应 `core-loop-runs/README.md` §6 第 6 条。
