# Journey 12 prereg — 容器任务的 artifact 写入侧 + 聚合卡的第一次真机锻炼

写于 2026-11-10，被测 `4c53083`（`75371dc` 的容器守卫 + `d2348b7` 的卡片容器口径 + `54ad984` 的聚合谓词）。

## 1. 为什么要跑这一轮

Journey 11 把父 Issue 的卡验到了「出现 + 能点 + 闭环」，但**聚合谓词一次也没被真机锻炼**：cos 在 `submit_for_review` 时把 `JOURN-22` 的路径盖成了 `JOURN-2` 的 `artifact`，于是卡片命中「自己已有 artifact」那一支，聚合分支只在进程内测试里成立。`75371dc` 把那条写入路径堵上了（409 + 点名 lane）。所以这一轮的对象是**两个连在一起的判定**：

1. 模型带 lane 路径提交容器任务时，真的会拿到 409，并且**会重试**（而不是放弃或换成另一条 lane 的路径）；
2. 父 Issue 的 `artifact` 因此保持为空时，聚合谓词真的会发卡、卡正文真的会列出**两条** lane 的路径、点击真的能闭环。

## 2. 场景

与 J10/J11 逐字相同：Acme 续约（报价说明 + 跟进邮件），英文，office 模式，`opencode-go/longcat-2.5-free`。人工只做三件事：答语言采访、批准计划、批准父 Issue 的卡。**不加**任何额外提示，不预告 409。

## 3. 验证项

| 编号 | 断言 | 读哪里 |
|---|---|---|
| V1 | 容器写入被拒：`submit_for_review`（或任何带 `artifact_path` 的变更）对父 Issue 使用任一 lane 的路径时返回**错误**，文案含 `is a container` 与那条 lane 的 ID | 任务的 `ledger`/`messages`（bot 会把工具错误贴出来）、或我作为人类直接发一次同样的请求 |
| V2 | 模型重试后父 Issue 的 `artifact` 为空（不强求「不再提交」，只要求没写成 lane 路径） | `broker-state.json` 的 `JOURN-*` |
| V3 | 父 Issue 进 `decision` 后 ≤3 分钟出现 `dedupe_key=decision:<父ID>` 卡，`blocking`/`required` 为 null | `requests` |
| V4 | 卡正文列出**每一条**已交付 lane 的路径（本轮期望两条），且不出现 `The deliverable is <单一路径>` 的单文件口径 | 卡 `question` |
| V5 | 点 `approve_decision` 不带 `artifact_path` → 200，父 Issue `status=done` `lifecycle_state=approved`，`hollow_completion` 不为 true | `broker-state.json` + 屏幕 |
| V6 | 回归：计划卡唯一、`task_not_decomposed` 0 条、每条 lane 的声明路径与落盘路径逐字一致、wiki 里有 `team/decisions/<父ID>.md` | `requests`、`messages`、wiki `git log` |

## 4. 失败模式（硬失败）

| 编号 | 含义 |
|---|---|
| F-a | 409 出现但模型**没有**重试，父 Issue 停在一个既没有 artifact 也没提交的状态（=新死路） |
| F-b | 模型绕道：换一条 lane 的路径再试（说明守卫只堵了一条路径而不是一类） |
| F-c | 父 Issue 的 `artifact` 仍被写成某条 lane 的路径（说明还有第三条写路径） |
| F-d | 父 Issue `artifact` 为空时**卡没发**（聚合谓词在真机上不成立，J10 的 F-I 复发） |
| F-e | 卡发了但只点名一条路径（`d2348b7` 的容器分支没生效） |
| F-f | 点击后 409/500，或 `hollow_completion=true` |
| F-g | 计划卡/写范围/路径钉出现回归 |

## 5. 干预预算与停表

- 预算：**3 次**（语言采访、计划卡、父 Issue 卡）。任何额外干预（催提交、取消 lane、解释 409）都记进 `observations.md` 并算超预算。
- 停表：连续 15 分钟无新消息、无 active 回合、`requests` 无未答 `decision:` 卡；或累计 60 分钟。
- 停表后：归档 `wiki-log.txt`、`team-tree.txt`、`broker-state.json`、`messages.json`、broker.log 尾、`headless-codex-latency.log`、关键产物原文到 `artifacts/`，然后 `workspace shred --yes journey12`。

## 6. 判分

独立子代理按本文件写 `scorecard.md`（不采信 `observations.md` 的结论化措辞）。重点看 V1/V2 的工具错误原文与 V4 的卡正文，不要只看最终 `status`。

## 7. 已知的环境限制

- gbrain `put_page` 可能继续 `EOF`（不影响判定，只影响 source_capture）。
- 测试基建的 `TestObsidianWatcher` flake 与这一轮无关（`§9.23` 已记）。
