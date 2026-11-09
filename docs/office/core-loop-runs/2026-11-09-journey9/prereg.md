# Journey 9 — 预登记（2026-11-09）

> 本轮回答的是 Journey 8 评分卡留下的三处修复**在真机上是否成立**（D2/D3/D4/D5）。J8 的新几条结论都是在单测与集成测试里证明的，而这三条碰巧全都落在「两家模型 + 两个 bot 真在一起跑」才暴露的路径上：规划者会不会再造重复 lane、lane 会不会拿视觉 artifact 顶替声明路径、一条 lane 落地后同一 Issue 的其他人会不会在两分钟内被叫醒。
> 场景与 J7/J8 **逐字相同**，这样差异只能来自机制，不能来自题目。

## 1. 被测提交

- `690b601` test(team): pin the path-mismatch refusal as self-service, on the call
  - 运行时行为与 `000c4b7` **完全相同**（本条只加测试与文档）。
- `000c4b7` fix(team): one file is one deliverable, and a landing lane wakes the rest
  - D2 兄弟去重加 CJK shingle 包含度 + 同父 `artifact_path` 唯一性；`allow_duplicate=true` 可越过
  - D3 完成门拒绝「声明了写范围 + `ra_` 收 done」，`verification=not_required` + 理由仍是出口
  - D4 一条 lane 落地 → 父 Issue 与仍 open 兄弟的**既有** follow-up 提前到 ~2 分钟后
  - D5 `plan_decomposition_check` 的 `status_reason` 报「lane 总数 + 仍 open 数」
- 基线：`a4ff236`（J8 prereg）、`c111b9e`（J8 记录）、`5a026e7`（A 方案）

## 2. 环境

| 项 | 值 |
|---|---|
| 隔离办公室 | `journey9`（broker :7910，web :7911） |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/crouton-alpha-free` |
| 蓝图 | `ai-revops`，agents `cos, revops, analyst, ae, reviewer` |
| 主办公室 | 7891/7892，**不得被本轮任何命令指向** |

### 2.1 隔离手段

命名频道不可用（`internal/channel/general.go` 的 `namedChannelsEnabled()` 硬编码 false），场景仍在 `cos__human`，靠清理隔离，与 J7/J8 同法：

1. onboarding 后、提交场景前立即取消 `ai-revops-1..4`；
2. 每次快照检查是否出现非场景任务/采访卡，出现即取消并逐条记进 observations，判分单列「环境清理」；
3. 场景本身不带 `definition`（让规划者自己决定 lane 写范围）。

## 3. 场景

父 Issue：`channel=cos__human`、`owner=cos`、`created_by=human`、`task_type=issue`、无 `definition`：

```
title:   "Journey 9: Acme 续约报价与跟进"
details: "客户 Acme（40 席位，SaaS）。现行合同：年付 $18,000，2026-12-31 到期，付款 30 天。
          拟续约：年付 $21,600（+20%），条款不变，有效期 14 天。对方决策人 Dana（VP Ops）。
          两件交付物：① 一页报价说明（价格/范围/有效期/付款/决策点）② 48 小时内的跟进邮件草稿。
          只认以上数字，不要引入其它假设。"
```

除标题前缀 `Journey 9` 外与 J7/J8 逐字相同。

预期路径：cos planning → 计划卡 → 批准 → 拆 lane（每条带写范围）→ owner 用 `/wiki/write-task` 交付 → 一条落地后其余在两分钟内被唤醒。

## 4. 验证项

沿用 J8 的 V1–V6（可比），新增 V7–V10 对应本轮四处修复。

| 编号 | 判定 | 通过判据 |
|---|---|---|
| V1 | 每条 lane 都带写范围 | `broker-state.json` 里每个子 Issue 都有 `definition.artifact_path` 或 `artifact_not_required`（后者带理由且合理） |
| V2 | 共享 wiki 真的有交付 | `team/**` 出现 ≥1 份 markdown，`wiki log` 对应 commit 作者是场景 lane 的 owner；`broker.log` 至少 1 条 `write-task` 成功 |
| V3 | 守卫是提示，不是墙 | 规划者撞写范围守卫 ≤ 每条 lane 1 次，且撞后改做法（补路径或声明不产出）；不得「原样重试同一 create」≥2 次 |
| V4 | 不回归 | 计划卡自动出现且只有一张；`task_not_decomposed` 0 条；`watchdog_alert` 0 条；拆 lane 后 `plan_decomposition_check` 带 `status_reason` 正常收尾 |
| V5 | 反向臂（操作员探针，停表后执行，不计场景干预） | 用错误路径打 `/wiki/write-task`：403 文案点名 creator，且**不**新增 `artifact_path_missing`（D6 判定） |
| V6 | 不刷屏 | 频道里因写范围被拒的 create 总数 ≤ 5 |
| **V7** | **同父 lane 不再重复（D2）** | 同一父 Issue 下不存在两条**同时 open** 的 lane 声明同一 `artifact_path`；不存在被 CJK shingle 判据识别的重复标题成对出现于 open 集合；守卫拒绝次数与「改做法」都记进 observations |
| **V8** | **视觉 artifact 不能冒充交付（D3）** | 声明了 `artifact_path` 的 lane 若以 `ra_` 收 done，必须收到 `artifact_required`（409）并随后改用 `/wiki/write-task` 或 `verification=not_required`；不得出现「声明了路径、`team/**` 为空、任务 done」 |
| **V9** | **落地唤醒（D4）** | 任一 lane 落地后 ≤ 3 分钟内，父 Issue 或仍 open 兄弟的 `task_follow_up.due_at` 被提前到「落地时刻 + 2 分钟」附近；同一 Issue 在仍有 open lane 时的最长静默 ≤ 10 分钟（J8 为 26 分钟） |
| **V10** | **分解计数（D5）** | `plan-decomposition-check` 的 `status_reason` 中 lane 总数与实际子 Issue 数量一致（J8 报 1、实际 6） |

## 5. 硬失败

- **F-A（回退触发）**：同一 lane 因写范围被拒 ≥3 次且 create 内容不变（守卫变成墙）。
- **F-B（回退触发）**：所有 lane 的 `artifact_path` 都是敷衍值（`team/tasks/x.md` 这类不含信息的路径），或声明 `artifact_not_required` 而实际有交付物。
- **F-C**：计划卡缺席（J3-1 回归）。
- **F-D**：拆 lane 成功但 60 分钟内 `team/**` 仍为 0 份（机制通了但没人用）。
- **F-E**：批准后 10 分钟内既无子任务也无 `task_not_decomposed`（分解检查回归）。
- **F-F（D2 失败）**：同一父 Issue 下出现两条同时 open、声明同一 `artifact_path` 的 lane。
- **F-G（D3 失败）**：声明了 `artifact_path` 的 lane 用 `ra_` 收 done 且被放行，而 `team/**` 里没有声明的那份文件。
- **F-H（D4 失败）**：一条 lane 落地后，同一 Issue 仍静默 ≥ 15 分钟且无 active 回合。

**回退**：触发 F-A 或 F-B → 切 B（接受 notebook + visual artifact 为交付载体），把 §9.15 降级为「防路径不匹配」。触发 F-F 或 F-G → 对应判据（CJK 包含度 / `ra_` 门）判为不成立，回到 DESIGN-DECISIONS 重新论。触发 F-H → D4 的改动不足以覆盖真实唤醒需求，按产品决策重新评估（加 sibling-completion wake 还是让 lead 自行为等待 lane 装 follow-up）。这是写死的判据，不在跑完后改。

## 6. 人工干预

**只允许 1 次**：点批准卡。取消 onboarding 遗留、清理非场景任务逐条记进 observations，判分单列为环境清理。停表后的反向臂探针（V5）与 D2/D3 的操作员探针同样单列，不计入场景干预。

## 7. 停表

连续 15 分钟无新消息且无 active 回合，或 60 分钟上限。停表后**先**把 `wiki-commits.txt`、`wiki-tree.txt`、lane notebook、`broker.log` 摘要、`broker-state.json` 的 `scheduler` 段复制进 `artifacts/`，**再** `workspace shred`。快照每 5 分钟一次到 `snapshots/HHMM/`。

## 8. 判分

记录者（主代理）只写 `observations.md`（时间戳事实，不写结论），`scorecard.md` 由**独立**子代理按本文件的 V1–V10 与 F-A…F-H 写，跑完再对。
