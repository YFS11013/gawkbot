# 多智能体能跑到哪：外部调研记录（2026-11-01）

> 姊妹文档：`USE-CASES-2026-11-01.md`（这套办公室具体该接哪些活）。
>
> 本文件是 `DESIGN-DECISIONS.md` §10 四条判据的**证据档**。判据（要做什么）留在设计决策里；
> 来源、原文数字、口径、局限只放这里。补来源只动本文件；改判据要同步两边。

调研目的是回答一个问题：**这套办公室适合干什么，不适合干什么**。时点在核心循环演练之后，
拿"一手工程文 + 一组行业数据"做对照，避免用直觉当证据。

## 1. 方法与局限（先读这段再引用数字）

- **一手工程文 3 篇**（S1–S3）：作者是在生产里跑多智能体的人。可信度较高，但都是**自述**——
  没有可复现的实验设计、没有基线配置、没有失败样本，也没有第三方复现。
- **行业数据 1 组**（S4）：IDC 的 88% 是**经二手汇总文转述**，原文未读。方向可信，精确值不可引。
- **产品观察 1 组**（S5）：无一手链接，属观察而非证据。
- 全部数字**未复现**。下文引用一律注明口径（谁的评测、拿什么比什么）。
- 时效性：模型换代或价格结构变化后，"该不该并行"的结论会变。**更新触发条件见 §6**。

## 2. 来源逐条

### S1 — How we built our multi-agent research system

| 项 | 内容 |
|---|---|
| 链接 | https://www.anthropic.com/research/multi-agent-research-system （原 `/engineering/built-multi-agent-research-system` 已 301） |
| 发布 | Anthropic Engineering，**2025-06-13**（Jeremy Hadfield 等 6 人） |
| 可信度 | 高（一手工程自述） |

**原文关键数字与口径**

- **+90.2%**：内部 research eval，"multi-agent system with **Claude Opus 4 as the lead agent and Claude Sonnet 4 subagents** outperformed single-agent Claude Opus 4 by 90.2%"。→ 口径是**特定模型组合 vs 单一大模型**，不是"多智能体普遍优于单 agent"。
- **4x / 15x token**：agents 约为普通 chat 的 4x，multi-agent 约为 chat 的 **15x**。原文同时给出经济性门槛："multi-agent systems require tasks where the value of the task is high enough to pay for the increased performance"。
- 性能归因：BrowseComp 上三个因素解释 95% 方差，其中 **token 用量单独解释 80%**——即多智能体的收益很大程度来自"花了更多 token"。

**适合 / 不适合**

- 适合：heavy parallelization、信息超出单上下文窗口、要对接大量复杂工具。
- 不适合："most coding tasks involve fewer truly parallelizable tasks than research"，且模型实时协调/委派能力仍不足。

**两条对我们直接用得上、但当时没写进判据的附录建议**

1. **Subagent output to a filesystem to minimize the "game of telephone"**：子代理把成果写进外部系统、只把轻量引用回传给协调者，既保真又省 token。——这条其实是判据 3（子代理写文件、只回传引用）的**一手出处**，此前误记作推论。
2. **End-state evaluation**：对会持续改状态的多轮 agent，按**最终状态**评估，而不是逐步校验流程（复杂流程拆成离散 checkpoint）。——对"完成即 done"的语义设计有参考价值，本仓库尚未据此改造。

### S2 — Don't Build Multi-Agents

| 项 | 内容 |
|---|---|
| 链接 | https://cognition.com/blog/dont-build-multi-agents |
| 发布 | Cognition（Walden Yan），**2025-06-12** |
| 可信度 | 高（一手，但无量化数据） |

**两条原则**（原文自述为"违反了就该默认排除这种架构"）

1. **Share context**：共享完整 agent trace，而不是单条消息——否则子任务会被曲解。
2. **Actions carry implicit decisions, and conflicting decisions carry bad results**：并行写手各自补边界/风格/模式，彼此的隐性决策冲突，合并结果碎片化。文中以"Flappy Bird 克隆"分工失败为例。

**对本仓库的意义**：这是判据 1（一个任务一个写手）的直接依据。它同时示范了一个正例——Claude Code 的子代理"只回答问题、不写代码"，因此不产生写冲突。

### S3 — Multi-Agents: What's Actually Working

| 项 | 内容 |
|---|---|
| 链接 | https://cognition.com/blog/multi-agents-working |
| 发布 | Cognition（Walden Yan），**2026-04-22** |
| 可信度 | 高（一手 + 唯一给出量化收益的一篇） |

**核心数字（口径必读）**

- **平均每 PR 2 个 bug，其中 ~58% 属严重**（逻辑错误、缺边界、安全问题）。
  口径：**Devin 写的 PR，由 Devin Review（同模型、独立 harness）来审**——不是"人类 PR 由 AI 审"，也不是"不同模型互审"。所以这条证据的意思是"同模型换一个干净上下文就已显著有效"。

**反直觉的机制**

- 效果最好时，**写手与审阅者事先不共享任何上下文**。理由是 context rot：写手跑了几小时后上下文很长，注意力被稀释（原文引 Chroma 的 Context Rot 研究：https://www.trychroma.com/research/context-rot）；审阅者只看 diff，被迫从实现反推规格，反而更容易发现"用户指令本身有问题"这类错误。
- 但**结论需要写手来过滤**：原文强调"沟通桥"——写手要用自己更宽的上下文判断哪些发现越界/无关，否则会循环、越界、违背用户意图。→ 这正是本仓库"审阅不 veto、结论交人/CoS 决策"的旁证。

**结论句**："multi-agent systems work best today when **writes stay single-threaded** and the additional agents contribute intelligence rather than actions."
无结构 swarm 被明确称为干扰项；实用形状是 **map-reduce-and-manage**。

**另外两个模式（本仓库尚未使用，与候选场景"按技能路由模型"直接相关）**

- **Smart friend**：小模型在遇到难题时调用一次大模型。作者自评：SWE-1.5 做主模型时失败（不会判断何时升级、该问什么），而**跨 frontier 模型**的版本已在生产跑出真实收益，且形态变成"能力路由"（哪个模型更擅长哪个子任务）而非"难度升级"。
- **Manager Devin**：管理器把大任务拆给子 Devin，经内部 MCP 协调。难点不在拆分，在上下文传递（子代误以为与管理器共享状态、子代发现无法回传给兄弟）。

### S4 — 企业 AI pilot 的存活率

| 项 | 内容 |
|---|---|
| 链接 | https://www.ayautomate.com/blog/ai-pilot-stuck-deployment-infrastructure （转述 IDC） |
| 发布 | 二手汇总文，无日期 |
| 可信度 | **中低**（数字二手，未读 IDC 原文） |

- **88%** 的 pilot 进不了生产。数字不可精确引用。
- 但它的**机制解释**可用：成功案例的共同点是**成功判据简单且可验证**；死因不是模型能力，而是没人敢放开不可验证的写操作。→ 判据 4 的依据。

### S5 — 金融研究产品形态

| 项 | 内容 |
|---|---|
| 链接 | 无一手链接（Brightwave / AlphaSense / Hebbia / Rogo 公开产品页） |
| 可信度 | **低**（观察） |

卖的是"成品研究报告 / 投资产品"，不是聊天框。与本办公室"研究简报"定位一致，属旁证。

## 3. S1 与 S2 看似矛盾，边界在"谁写"

S1 说多智能体在广度研究上收益大，S2 说别做多智能体。两者不矛盾：

- S1 研究的是**只读并行**（多个搜索/摘要 agent 把压缩后的信息交回 lead），写操作从不并行。
- S2 反对的是**并行写手**（多个 agent 各自产出最终交付物，隐性决策冲突）。
- S3 是两者的合流：**写单线程 + 其他 agent 只贡献智能**，并给出了量化收益。S3 脚注还记了一笔——S1 发布次日（2025-06-13）Anthropic 那篇就出来了，两文的"首个可用领域是只读 agent"结论一致。

**本仓库的判断（属推论，非原文）**：并行只放在写操作之外——只读子代理、审阅者；所有落盘交付物保持单写手。

## 4. 由此固化的四条判据（正文在 `DESIGN-DECISIONS.md` §10）

| 判据 | 直接证据 | 落地位置 / 状态 |
|---|---|---|
| 1. 一个任务一个写手；并行只发生在写操作之外 | S2 原则 2、S3 结论句 | `owner` 语义 + `artifact_path` 单写者；已执行 |
| 2. 干净上下文的审阅回合是 15x token 最划算的用法 | S3（2 bugs/58%，且**同模型**换干净上下文即有效）、S1（15x 成本） | `internal/team/task_review.go`；**2026-11-01 已落地**（`RUNTIME-API.md` §1.2） |
| 3. 子代理写文件、只回传引用 | **S1 附录（game of telephone）**、S2（避免传话失真） | `artifact_path` + `wiki_artifact`（含新鲜度校验）；已执行 |
| 4. 能写死的判据优先：`command` > `wiki_artifact` > 人眼 | S4（死因是不可验证的写操作） | 校验器分层（`task_verification.go`）；已执行 |

**反面清单**（有证据支持"别做"）：并行写同一份 wiki / 同一条任务（S2）；无结构 swarm 协商（S2、S3）；无人值守放开外发与 canonical 写入（S4）；用多 agent 硬啃长程编码（S1）。

## 5. 候选场景与状态（对应 `TODO-NEXT.md` 开放项 J）

| 候选 | 依据 | 状态 | 卡在哪 |
|---|---|---|---|
| 干净上下文审阅回合 | S3 | ✅ 已落地（`task_review.go`；E2E：ACMER-2847→2852、ACMER-2893→2899） | — |
| 广度研究拆只读子代理 | S1（+90.2%，但 token 15x） | 未做 | 当前 routine 的广度需求未超单上下文；要先有"一个任务塞不下"的实例，否则只是多花 token |
| 数值断言检查单 | S4；已知事故：10-30 简报三处编造数字 | 未做 | 需先定"哪些数字必须带来源+时间"的字段级契约，否则检查单退化成人工复核 |
| 按技能路由模型 | S3（smart friend / 能力路由）；S1（15x 成本压力） | 未做 | 需要分技能的成本/质量数据；当前只有一个 provider 在跑，无法归因 |

## 6. 什么时候该重读、怎么自测收益

满足任一条就重跑一轮调研并更新 §2：

1. 主力模型换代（当前 `opencode-go/beam-free`，兜底 `bear/deepseek-chat`），或 provider 价格结构变化；
2. 出现"一个任务的信息量明显超出单上下文"的实例（这是 S1 判据的触发条件）；
3. 审阅回合累计 ≥ 20 次，可对照 S3 自测收益。

**自测口径**：`team/reviews/` 下审阅文件的「阻断」级发现数 ÷ 被审阅交付数，与 S3 的"每 PR 2 个 bug、
58% 严重"对比。低于 0.5 说明本办公室的交付质量已高到不值得付这轮 token，应该
`WUPHF_TASK_REVIEWER=off`，或只对特定 pipeline 开。

当前样本量：2 次审阅（ACMER-2847 → 0/3 验收通过、2 条阻断；ACMER-2893 → 1/3 通过、2 条阻断）。
样本太小，不作为结论，只作链路验证。
