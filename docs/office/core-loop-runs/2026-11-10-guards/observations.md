# Guard exercise observations — 2026-11-10，被测 `0464f91`

操作员用 broker HTTP API 直接扮演抄近路的 agent，不经过模型。工作区 `guards15`（broker :7910）。

## 0 先说两次试跑

**第一次（`GUARD-1`）没搭成场景，但结果有意义**：lane 建好后 `/wiki/write-task` 少了 `mode`（`unknown write mode ""` → 500），文件没落地，两条 lane 停在 `in_progress`、`artifact=null`。于是容器「没有已交付 lane」，三道门按设计**都不适用**：

- `artifact_path=team/accounts/guard-quote.md` → 409，但文案是**存在性**门（`does not exist in the wiki`），不是容器门；
- `definition.artifact_path=team/emails/guard-email.md` → **200**（见 §3 的旁路）；
- `complete` + `not_required` → **200，`done/approved`，`hollow_completion=true`**。

第三条与 `TestEmptyContainerStillClosesFromTheTaskEndpoint` 的约定一致：**空容器**从任务端点关掉是合法的，`hollow=true` 是诚实的（它确实什么都没交付）。所以这不是缺陷，而是「门只在有已交付 lane 时存在」的边界被实测确认。

**第二次（`GUARD-9`）搭成了**：`mode=create` 写盘成功（`bytes_written=33`，commit `8279a96` / `4ab8eb7`），两条 lane `complete` 带 `artifact_path` → `done/approved`、`verification_result.pass=true`。

## 1 三次撞门（原文）

```
V1  complete + artifact_path=team/accounts/guard-quote.md
    409 task GUARD-9 is a container: team/accounts/guard-quote.md is GUARD-12's deliverable,
    not this Issue's own. Leave artifact_path empty — its lanes' files are already on record
    and the decision card names every one of them — or pass a file GUARD-9 wrote itself.

V1b define + definition.artifact_path=team/emails/guard-email.md
    409 task GUARD-9 is a container: team/emails/guard-email.md is GUARD-14's deliverable,
    not a file this Issue writes itself. Leave artifact_path out of the definition — the
    decision card names every lane file.

V7  complete + verification=not_required（bot 身份 cos）
    409 task GUARD-9 is a container with 2 delivered lane(s): a bot may not close it from
    here — that would skip the human. Call submit_for_review on it with no artifact_path;
    the card names every lane file and the human's approve closes the Issue.
    verification=not_required does not describe an Issue whose …
```

三次之后容器仍是 `in_progress/running`、`artifact=null`、`artifact_not_required=null`（G4 成立：没有被任何一次写入污染）。

## 2 正路与卡

```
submit_for_review（不带 artifact_path）  200  -> review/review, artifact=null
request-21  dedupe_key=decision:GUARD-9  pending
approve_decision                          200  {"ok":true}
GUARD-9 -> done/approved, artifact=null, hollow_completion 缺省(=false)
```

卡片正文（原文）：

> @cos finished Guard exercise 2: two-lane pack (GUARD-9) and the review is in. Nothing else moves until you call it.
>
> This Issue's lanes delivered 2 file(s):
>
> - team/accounts/guard-quote.md
> - team/emails/guard-email.md
>
> The Issue ships no single file of its own; those are what you are accepting.
> Approve and the lane closes. Send it back and the owner picks up your note as the rework.

**G1/G2/G3/G4/G5/G6/G7 全部成立**（H-a…H-e 均未命中）。

## 3 判据外的新发现：容器的写范围守卫有个「时窗」

第一次试跑里 `definition.artifact_path` 指向一条**尚未交付**的 lane 的声明路径 → **200 接受**。原因：`childOwningArtifactLocked` 只匹配子任务的 `Artifact`（已记录产物），不看子任务的 `Definition.ArtifactPath`（声明的写范围）。所以在「lane 已声明路径、还没交付」的窗口里，容器可以把自己授权去写那条 lane 将来要写的文件——正是 §9.24 要堵的那类越权，只是早了一步。

修法：匹配时同时看子任务声明的 `Definition.ArtifactPath`，并跳过 `canceled/archived`（取消的 lane 不再拥有该路径）。

## 4 其它观察（与判据无关）

- **撞门之后文件仍被改写，但不是容器干的**：`GUARD-9` 关掉后（21:25:40）`GUARD-4`（上一轮遗留的 lane）用 `write-task` 把 `team/accounts/guard-quote.md` 从 33 字节改成 1642 字节，`write-task` 200。`GUARD-4` 自己声明了同一路径，所以这是「两条 lane 声明同一文件」，属我搭场景时留下的同名声明，不是容器越权。但它暴露出一个**未设防的相邻问题**：一个任务可以写另一个任务已记录的 `artifact`（这里 GUARD-12 已交付该文件，GUARD-4 仍可覆盖）。是否该独占未定——`mode=append_section` 的共享写法是有意的；先记为观察，不动。
- `GET /requests`（HTTP）在这个构建上返回 `requests: []`，而同一刻 `broker-state.json` 里有 `pending` 卡。脚本改用状态文件取卡 id。**已查明并修复**（见 §6）：不是视图不同步，是裸 `GET /requests` 的 `channel` 被 `normalizeChannelSlug` 洗成了 `general`，于是按 `general` 过滤、一条都不剩；那条「channel is required」的 400 分支因为读的是洗过的值，从来没被执行过。
- 模型（cos）在 `GUARD-9` 期间自己起了一轮，挂出 `request-23`「两条 lane 应该是什么？我读不到定义」——这是它对一个**操作员搭的空壳场景**的正常反应，不计入判据。
- 上一轮 J14 的教训在这里复现了一次形态：场景里只要有一步没搭实（这次是 `mode` 缺失），门就无声地不适用。**门的断言必须同时断言前置条件成立**（本次 prereg 的 G4 只查了「没被污染」，应补「lane 确实 done」）。

## 5 回顾：G4 这条判据过宽（评分后修，prereg 不追改）

G4 写的是「三次撞门后容器仍 `in_progress`、`artifact=null`」，只断言**没被污染**，没断言**门有对象**。第一次试跑（`GUARD-1`）正好落在这个缺口里：`write-task` 少 `mode`，两条 lane 停在 `in_progress`，三道门按设计全部不适用——撞门要么被存在性门拦下（409 `does not exist in the wiki`）、要么直接 200——而 G4 依然成立。

本次因为第二次试跑把前置搭实了（两条 lane `done/approved`、`pass=true`、`checked_at=13:21:40Z` 磁盘可验）才没有误判，但那是运气不是判据。

修订（写进下一轮 prereg）：`G4' = G4 ∧ 探针前两条 lane 均 done/approved ∧ verification_result.pass=true ∧ artifact 精确等于各自声明路径`。缺前置时整体判**未观察**，不判成立。`scorecard.md` §4 有同一批评的更完整版本。

**证据纪律更正**：本文件的 prereg 虽然写在开跑前，但直到与结果同一提交（`db3cbea`）才进仓，且 G4 的补注是评分后追加的——违反 `core-loop-runs/README.md` §1「开跑前 commit、跑完不许改」。已在 `prereg.md` 顶部自曝，并把 G4 修订移到这里。下一轮剧本演练先单独 commit prereg 再开跑。

## 6 从 §4 的观察里查出来的缺陷：裸 `GET /requests` 静默答「没有请求」

§4 那条「`GET /requests` 返回 `requests: []` 而状态文件里有 pending 卡」不是看法问题，是一条真缺陷。

`handleGetRequests` 先 `normalizeChannelSlug(r.URL.Query().Get("channel"))`，而这个函数把 `""` 洗成 `"general"`；随后那条「channel is required」的 400 分支判断的是**洗过之后**的 `channel`，所以永远是假——分支是死代码。裸调用的结果：200、`channel: "general"`、`requests: []`，不管实际有多少张待决卡躺在别的房间里。我的驱动脚本正是这样被误导了 2 分钟（`NO CARD`）。

修复：判断改用**原始 query 值**（`rawChannel`），其余不变。裸 listing → 400 `channel is required: there is no default room to fall back to. Name a channel, pass an id, or pass scope=all …`；`?channel=<x>`、`?scope=all`、`?id=<rid>` 三种正常用法不受影响。测试 `TestGetRequestsWithoutAChannelSaysSo`（去掉修复即红：`get 200: {"channel":"general","pending":null,"requests":[],"scope":""}`）。

为什么危险：这是一个「自信的空答案」。任何外部脚本、巡检或未来的驱动脚本只要漏了 `channel`，都会读到「球不在人手里」。

## 7 判据外第三条：三次 409 全都没有留痕（已修）

`artifacts/actions.json` 26 条全是 `task_created`/`task_updated`/`request_created`/`request_answered`/`automation`，`broker-log-tail.txt` 只有 3 条成功的 `wiki write-task`——**三次拒绝一点痕迹都没有**。评分卡因此只能把 G1/G2/G3 的证据来源标成「操作员记录，非磁盘原始」。

对一个「拦住越权」的门来说这是错的：越权尝试本身就是要被计数的事件（谁、什么时候、想写哪条 lane 的哪个文件）。现在三条守卫的拒绝各写一条 `guard_refused`：

- `container scope` —— `define` 的写范围指向 lane 的文件；
- `container artifact` —— `artifact_path` 指向 lane 的文件；
- `container close` —— bot 想从任务端点关掉有已交付 lane 的容器。

`related_id` 是任务 id，summary 里带被撞的路径与 lane id，`channel` 沿任务的频道。测试 `TestContainerGuardRefusalsLeaveALedgerRow`（把记录函数改成空实现即红：`got 0`）。

**实现陷阱**：必须记在 `rollbackTask()` **之后**。`brokerTaskMutationSnapshot.restore` 会把 `b.actions` 整体重置回快照，先记就等于没记——这一点由 `broker_tasks.go:29` 的 `restore` 读取 `snapshot.actions` 决定，不是猜测。

同时确认了一件与判据有关的事：`complete` + `verification=not_required` **不带** `verification_reason` 会在更早的门被拒（`verification=not_required needs verification_reason`），所以撞容器关闭门时必须像真机那样把 reason 也带上，否则测的是另一道门。

**没有一起改的同类点（留作已知缺口）**：决策批准路径（`broker_decision_packet.go`，`RecordTaskDecision*` 的 pre-phase）对同一个 `childOwningArtifactLocked` 拒绝**仍然不留痕**。那里在 `b.mu` 持有期间返回（`defer Unlock` 在 895 行），只能用 locked 变体追加、且不能顺便落盘；既然这一轮没有覆盖它的测试，就先不动，把它写成缺口而不是加一条没人测的代码。

**同一处缺陷的第二面（同轮补）**：`byID` 也读的是洗过的 `channel`，所以**裸 `?id=<rid>`** 同样恒假，落进 `reqChannel != "general"` 过滤 → `200 {"requests":[]}`。这条比列表那条更难发现，因为它有真实调用方：`web/src/api/botKnowledge.ts:151` 的 `getPendingPromotion` 只传 `{id, viewer_slug}`，不传 channel——而提升请求建在 agent 自己的频道（如 `cos__human`）里，所以待批准的知识提升快照在这个构建上**基本显示不出来**（只有恰好落在 `general` 的才碰巧能读到）。顺带修掉早期那道 `canAccessChannelLocked(viewerSlug, "general")` 的 403 闸：by-ID 读不该先要求 general 的可见性，可见性按每条请求单独判。测试补了三段：裸 `?id=<存在的 id>`（跨频道）必须命中一条、未知 id 必须 200 空列表、裸列表必须 400。去掉 `rawChannel` 那处改动即红：`a bare by-ID read must return the request it names (cross-channel), got []`。

## 8 结案：§7 留的缺口已修（2026-11-10 晚，`01bfcb8`）

§7 末尾「没有一起改的同类点」写的是决策批准路径**仍然不留痕**。已修：`recordTaskDecisionInternal` 的 pre-phase 在 `childOwningArtifactLocked` 拒绝处调用 `recordGuardRefusalLocked`（`broker_decision_packet.go:959`）。

当时担心的「在 `b.mu` 持有期间返回、且不能顺便落盘」不成立：这道门跑在任何 mutation **之前**，没有 `rollbackTask()` 之后的快照恢复问题；`recordGuardRefusalLocked` 自己 `saveLocked()`，行会持久化。

`TestDecisionApproveRefusesALanePathAsTheIssueArtifact` 现在断言这一行（去掉记录即红：`got 0`），并断言 summary 带路径与 lane id；`TestDecisionApproveWithoutALanePathLeavesNoGuardRow` 钉住「干净的 approve 不留行」，避免这个计数变成噪音。
