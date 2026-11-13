# Journey 28 observations — 拒绝之后，以及谁在验收

工作区 `~/.wuphf-spaces/journey28`，broker :7910，视觉 :7911。
源码 `88a125f`（D-15/D-16 在包里）。判分见 `scorecard.md`。

## 时间线（本地）

| 时刻 | 事件 |
|---|---|
| 23:27 | 工作区创建；office pid 1020 |
| 23:28 | `msg-2` 真人会话（`from=human:ireke`）下达 H1 |
| 23:29 | 回合启动 → **23:34 被下一条消息取消**（`human-priority: cancelling active turn after 5m54s`） |
| 23:34 | `msg-3` 同一请求 + `@cos` → 回合启动 → **23:42 再次被取消**（7m18s） |
| 23:42 | `msg-4`（`from=you`，owner 身份）→ 回合启动 → **23:42:25 `AI_APICallError: Rate limit exceeded`**（`longcat-2.5-free`），title 调用 3 次重试全挂 |
| 23:57 | `error: headless codex turn timed out after 15m0s`，`timeout-recovery: no matching task found to block` —— **没有后续重试**（`WUPHF_HEADLESS_MODEL_FALLBACK` 未设） |
| 00:05 | 把 workspace 的 opencode 配置钉到 `beam-free`；硬杀 broker |
| 00:08–00:11 | registry 卡在 `already running (port 7910)`（进程已死）→ `pause` → `resume`，带 `WUPHF_OPENCODE_MODEL=opencode-go/beam-free`、`WUPHF_HEADLESS_MODEL_FALLBACK=opencode-go/iceman-free` |
| 00:11–00:17 | resume 包带着未答的人类消息唤醒 cos，这次跑在 **beam-free** 上 |
| 00:17 | `JOURN-5` 创建（唯一顶层任务，英文标题），计划卡 `request-9`；系统提示 `msg-7 artifact_path_missing` |
| 00:17–00:20 | 4×`403 artifact path not declared` → 1×**`human_request "msg-3" does not explicitly ask for a direct wiki write`** → 2×写入成功（两页） |
| H2 | 批准 `request-9` |
| 00:20:40 | `complete` → `wiki_artifact` 校验通过（475 bytes，文件是它自己刚写的）→ **`done` / `not_required`，没有决策卡** |
| 00:21 | system 建 `JOURN-18`（重复错误沉淀）；两页落库 `2a3cfd2` / `f00f5ca`；模型发 `msg-13/19/21/22/24/25` 收尾 |

## 三个观察

**1. 机器校验把人挤出去了。** 这一轮是四轮里第一次：交付物真实存在、台账真实完整（两页都在 `artifacts`），但**没有人被问过**。模型在 `create` 时顺手指了一条 `wiki_artifact` 校验指向自己刚写的文件——校验通过 → J22 的守卫让路 → 一步到 done。J22 的注释写着「人是否看到交付物，不能取决于 owner 选了哪个动词」，现在它取决于 owner 有没有**顺手声明一条校验**。修法是把「机器证明」收窄到不是 bot 自己写的那些检查。

**2. 授权门拒绝了说中文的人。** H1 里那句「我现在就授权你把这两页写进 wiki」被判 `does not explicitly ask for a direct wiki write`。这扇门的存在理由就是「人没有预先声明任务范围时的直接授权」，而它的动词表是英文的。它的错误文案还让 bot「用大白话去问人」——人用中文再回一次，还是被拒。这是每轮都记着的那个「English-only intent matcher」开放项，这次拿到了现场证据。

**3. 拒绝后的恢复，连续两轮没测到。** J27 是守卫判据漏了（D-15），J28 是模型压根没想开第二个任务——`create` 只有一次。要测到「被拒之后模型是否第一次就恢复」，场景必须逼出第二次 create，光靠「同形的两页请求」已经不够了。

## 环境与 harness（这轮的主要时间开销）

- **真人身份可行**：`POST /humans/invites`（broker 身份）取 token → `POST /humans/invites/accept` 拿 cookie → `POST /messages` 带 cookie，`From` 就是 `human:ireke`；用 broker token 时传 `from:"you"` 得到 owner 身份。两条都能唤醒 office。
- **每条新消息都会取消进行中的回合**（`human-priority: cancelling active turn`）——所以「发一次、然后等」是硬规则，重发只会把上一回合掐掉。
- **默认模型不是 beam-free**：新工作区解析到 `opencode-go/longcat-2.5-free`，当场 429，回合 15 分钟超时后**没有任何自动恢复**。`WUPHF_HEADLESS_MODEL_FALLBACK`（Journey 17 就有的旋钮，默认不设）就是为这个准备的；配上 `WUPHF_OPENCODE_MODEL` 一起注入，回合立刻活了。
- **硬杀 broker 会让 registry 卡在 `already running`**，`resume` 拒绝、端口却没人听；`pause`（graceful 会超时）→ `pause --force` 无效时直接 `Stop-Process` 再 `resume` 才能收回来。

## 工具面

9 次错误：4×`403 artifact path not declared`、1×**授权门中文被拒**、1×`define 400 goal is required`、1×`complete 409 still in planning`、1×`approve 403 parked`、1×`resume 409 already done`。写入 7 次：5 失败、2 成功（都是 `task_id=JOURN-5` 的任务门）。

**漏 `id` 报错 0**（F5 成立）——D-9 在这轮是干净的。

## 未测 / 已知限制

- 拒绝后的恢复（F1/F3）仍未测到，连续两轮。
- 人工授权门的**成功路径**仍未测到：D-17 已修（`cbc4bd1`，含红证），但工作区跑的是修复前的二进制，F4 的现场成功要等下一轮。
- 内容质量不计入缺陷：对外页三行是套话（「审批卡片现在如实反映操作结果」），但无 hash、无内部术语、≤3 行，格式达标。
- 生产（:7892）不参与本轮。
