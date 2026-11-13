# Journey 27 observations — 拒绝之后，模型知不知道该收尾

工作区 `~/.wuphf-spaces/journey27`，broker :7910，视觉 :7911，`opencode-go/beam-free`。
源码 `73c3d4b`（含 D-14）。判分见 `scorecard.md`。

## 时间线

| 时刻 | 事件 |
|---|---|
| H1 | 下达（与 J26 逐字同形的两页场景） |
| | `JOURN-2`「两页变更清单（内部 + 对外）」创建，`task_type=follow_up`，**未声明 artifact_scope** |
| | `JOURN-7`「对外说明页」创建，`task_type=follow_up`，声明 `team/reviews/j27-public.md` |
| | 两张计划卡：`request-5`（JOURN-2）、`request-12`（JOURN-7） |
| H2 | 批 `JOURN-2` 计划 |
| | `JOURN-2` 只发内部页 → `artifacts=[j27-internal.md]`，卡 `request-19` 念一页 |
| H3 | 批 `JOURN-2` 交付 → `done` |
| H4 | 批 `JOURN-7` 计划 → 其发对外页 → `artifacts=[j27-public.md]`，卡 `request-38` 念一页 |
| — | 预算用尽；`JOURN-7` 停在 `in_progress` |

## 三个观察

**1. 守卫一次都没触发。** 不是「被拒后恢复得不好」，而是**根本没被拒**。D-14 的两条语言判据在这个形状下都落空：旧任务 create 时没有声明路径（路径是后来在计划里定的），两个标题的 bigram 交集只有 `对外`。

**2. 真正的判据是路径，不是措辞。** `JOURN-7` 要写的 `team/reviews/j27-public.md`，四十秒前就已经写在 `JOURN-2` 的 details 里——人类那一句话同时点了两个文件。措辞会漂（「两页变更清单」/「对外说明页」），路径不会。所以 `openTaskNamesPath` 拿新任务声明的 scope 去旧任务的 title/details/goal/deliverables 里找，是最省事也最硬的一条。

**3. 拆分的代价这一轮量出来了。** 同形场景在 J26 里是 4 个动作闭合；这一轮**4 个动作用尽仍未闭合**——两张卡各念一页，人类要先批准「一半」，再回来批准「另一半」，而且第二半的卡在预算之外才出现。这不是「多一张卡」的观感问题，是把一次验收付费两次。

## 工具面

20 次工具错误：4×`complete` 撞「JOURN-7 is still in planning」、3×`approve` 撞「parked」、2×`complete`/`approve` 分别在两个任务上撞「recorded the deliverable」（重复提交）、1×`define` 400「goal is required」、2×`/wiki/write` 403「human_request msg-1 is not a human-authored message」、1×`/wiki/write-task` **500**「article already exists … use replace or append_section」。

最后那个 500 是 D-16：文案在教模型怎么改，状态码却在说服务端坏了。模型信了状态码，重试了一次。

## 未测 / 已知限制

- 人工授权门（D-12）仍然没有现场数据：隔离工作区的自动化消息发送者恒为 `nex`，`human_request` 校验必然判它非人。要现场验证，harness 必须能以人类身份发消息（`/messages` + human actor），尚未接。
- 内容质量不计入缺陷：对外页三行是「审批卡片现在如实反映操作结果」这类泛泛的话，但**没有内部术语、没有 hash**，格式要求达成。
- 生产（:7892）不参与本轮。

## 附：本轮的环境问题

`TestEnsureHiddenConsoleHidesTheWindow` 在本机**挂住**（detached 子进程的 `CombinedOutput` 等不到 EOF），导致 `go test ./internal/team/` 30 分钟超时。已在 **stash 掉本轮全部代码改动**的基线上复现同样挂住，因此与本轮 diff 无关；单独跑该测试的内层模式（`WUPHF_TEST_CONSOLE_MODE=none`）正常返回 `VERDICT has_console=true visible=false`——内层正常、外层 spawn 挂住。

本轮的完整套件因此排除该测试跑：`-skip 'TestObsidianWatcher|TestEnsureHiddenConsoleHidesTheWindow'`。上午 J26 的套件还能跑完（768s），所以这是**本机环境**在这一轮之间变了（可能与并发跑了两份套件、杀进程留下的 detached 子进程有关），不是回归。记录在案，避免下一轮把它当成新缺陷。
