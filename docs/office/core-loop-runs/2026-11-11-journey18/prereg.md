# Journey 18 prereg — 让办公室写一次真的发布说明（真值来自提交列表，人只定受众）

写于 2026-11-11，源码 `2770b3c`，二进制 `da98b245eb62d739`。**本文件先单独 commit 再开跑。**

## 1. 这一轮为什么不是虚构客户

J10–J17 用的「Acme 续约」是我编的客户：席位、报价、到期日都写在提示词里，办公室没有可查的东西，模型只能把答案复述一遍。那几轮证明的是机制（采访、lane、容器卡、守卫），证明不了产品价值——没有真值，就没有「做对」这回事。

这一轮换成**这个仓库自己的发布说明**。真值不是我的记忆，是 `git log`：我把它导出成一份固定清单塞进任务，模型写的每一条都要能在清单里找到对应提交。人只回答两个办公室确实不知道的问题——**这版说给谁听、哪些不提**。

## 2. 场景

种子 Issue（`created_by=human`，频道 `cos__human`，owner `cos`）：

> Journey 18: 这一版的发布说明
> 把这批改动写成两份东西，**分成两个文件**：
> ① 内部变更清单：每条改动一行，带短 hash，说清楚改了什么、影响谁。
> ② 对外的说明：给自托管用户看的，3–5 条，人话，不出现内部术语。
> 我附了一份提交清单，**只能写清单里有的东西**；清单里没有的功能一个字都不要提。
> 我不知道该强调什么、也不知道哪些不想对外说——这个问我。写完把整件事收尾，我要看到它进 done。

随任务的输入（我生成，不经过模型）：

```
$ git log --oneline --no-decorate 6a3f137~1..HEAD
2770b3c docs(office): production stays on free models, by decision
cd1064c docs(office): live proof of the silent-turn guard on the real office
b8cffdd fix(team): cancel a turn that never emits anything and retry on the fallback
…（共 20 行）
```

操作员动作（**当场判断，不预填**）：

1. **H1** 提 Issue（可以按自己的话改）。
2. **H2** 采访卡出现后，用自己的话答「说给谁听 + 哪些不提」。
3. **H3** 其中一条 lane 还在跑、尚无交付时，**亲手关掉它**（`complete` 或 `cancel`）。这是 `8026e40` 的唯一触发条件。
4. **H4** 剩下的那条 lane 交付后，按内容点 Issue 卡（`approve_decision` / `request_changes_decision`）。

## 3. 要验的机制

| 编号 | 断言 | 读哪里 |
|---|---|---|
| V1 | 全程**没有** lane 级决策卡（`dedupe_key` 形如 `decision:JOURN-*`）；Issue 卡恰好 1 张 | `requests[]` |
| V2 | 两条 lane 只在自己 `definition.artifact_path` 内写入，全程**没有** definition-gap 采访（没有人被问「我需要写权限」） | `requests[]`、`actions[]` |
| V3 | H3 关掉的 lane 若留有 pending 卡，状态变 `canceled` 且 reason 含 `task closed`（`8026e40`） | `requests[]` |
| V4 | Issue 卡的问题里**列全两个文件路径**，不是只点名一个 | `requests[dedupe_key=decision:<父ID>].question` |
| V5 | 两份交付里出现的每个短 hash **都存在于我给的清单**；不得出现清单外的 hash | 脚本：正则抽 `[0-9a-f]{7,40}`，逐个在 `git log --oneline 6a3f137~1..HEAD` 里查 |
| V6 | 对外页每一条都能映射到内部页某一行（不得凭空加承诺）；对外页不出现 `lane / broker / headless / dedupe` 等内部词 | 两页逐行比对 |
| V7 | 结局：父 Issue `done`、`artifact=null`、`hollow_completion` 全 false；存活 lane 的 `artifact` 与 `definition.artifact_path` 逐字一致 | `tasks[]` |

V5/V6 是这一轮的核心：**它们无法靠「模型写得好看」通过**，只能靠真的读了清单。

## 4. 失败模式

| 编号 | 含义 |
|---|---|
| F-a | lane 自升决策卡（`6a3f137` ① 无效或模型绕过） |
| F-b | 仍在问已授权的写入权限（`6a3f137` ② 无效） |
| F-c | 关掉的 lane 卡仍 `pending`（`8026e40` 无效） |
| F-d | 出现清单外的 hash 或清单外的功能（模型在拼记忆，不在读输入） |
| F-e | 对外页复述内部术语（受众约束被忽略，等于没回答采访） |
| F-f | 模型把 H3 的关单当成失败，另起一条 lane 重做 |

## 5. 前置机械探针：已由单测覆盖，不再单跑

原本计划在隔离办公室用 API 连写两次同一页。核对后取消：`internal/team/wiki_artifact_overwrite_test.go` 走的就是真 handler（`handleWikiWriteTask` + `recordForeignArtifactOverwrite`），两条断言都钉住了——

- `TestOverwritingAnotherTasksDeliverableLeavesARow`：另一个任务已交付的页被 `mode=replace` 覆盖 → 恰好一行 `artifact_overwritten`，summary 里同时出现路径与原作者任务号，且写入本身返回 200（不阻断）；
- `TestRewritingAnUnownedPageLeavesNoRow`：写自己的页 / 写没人交付过的页 → 不留行（否则每存一次都记一笔，这行就没意义了）。

2026-11-11 复跑：`ok internal/team 2.954s`。再起一个办公室手搓一遍只是把同一段代码跑第二次，`6a3f137` ③ 的证据强度不会增加，所以本轮不重复做。

## 6. 干预预算与停表

- 操作员干预：H1–H4 四处。
- 模型：主 `opencode-go/longcat-2.5-free`、兜底 `iceman-contributor-free`（用户 2026-11-11 拍板继续用免费）。**开跑前先探配额**；若 `longcat` 仍 429，先把主模型临时换成 `iceman-contributor-free`（探针 200），否则每回合先付 4 分钟守卫，白白拉长这一轮。
- 停表：出现 `blocked`、或第二次 `silent-stall` 即记录并停，不重试。
- 本轮**不计分**：受众取舍由人临时决定，跨轮不可比。回答的问题是「交付物能不能被真值约束住、卡还出现在错的地方吗」。
