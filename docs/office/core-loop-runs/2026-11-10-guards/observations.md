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
- `GET /requests`（HTTP）在这个构建上返回 `requests: []`，而同一刻 `broker-state.json` 里有 `pending` 卡。脚本改用状态文件取卡 id。**未深挖**，记在这里。
- 模型（cos）在 `GUARD-9` 期间自己起了一轮，挂出 `request-23`「两条 lane 应该是什么？我读不到定义」——这是它对一个**操作员搭的空壳场景**的正常反应，不计入判据。
- 上一轮 J14 的教训在这里复现了一次形态：场景里只要有一步没搭实（这次是 `mode` 缺失），门就无声地不适用。**门的断言必须同时断言前置条件成立**（本次 prereg 的 G4 只查了「没被污染」，应补「lane 确实 done」）。
