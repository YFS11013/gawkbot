# Journey 8 — 预登记（2026-11-09）

> 本轮回答的是 Journey 7 留下的**产品决策**：lane 在被创建时到底有没有写范围。
> J3-3 的 403 教育链在 J6/J7 两轮 clean 运行里一次都没被锻炼，原因已定位到代码：`artifact_scope` 的工具描述提前告诉模型「没有它就不能用 `/wiki/write-task`」，模型**读了说明就不试**，于是交付全部落在 notebook 与 visual artifact，`team/**` markdown 为 0。
> 本轮按 §9.15 的结论（选 A）先改机制，再问：**规划者会不会在被拒后当场补写范围，owner 能不能真的把交付写进共享 wiki。**

## 1. 被测提交

- `5a026e7` fix(team): a lane is created with its write scope, or it is refused
  - `team_task action=create` 接受 `definition`（goal + `artifact_scope`），限 lead/human（与 `define` 同类），防止 bot 靠建任务自我授权
  - lead 建的子 Issue 若既无 `artifact_scope` 也无 `artifact_not_required_reason` → 拒绝（文案点名 creator、列允许根、声明原样重试仍会被拒）
  - 规划者系统提示补上「建 lane 时当场钉写范围」及其原因
  - 专家与内部 actor 豁免：专家没有钉共享路径的权限，对它强制只会变成死路
- 基线：`4be0069`（J7 记录 + 分解检查免顶替）、`336bbbc`（分解检查 + 终态取消采访卡）、`d52293e`（403 三要素文案）
- 全量测试结果写在 observations §1（本轮起点）。

## 2. 环境

| 项 | 值 |
|---|---|
| 隔离办公室 | `journey8`（broker :7910，web :7911） |
| 模型 | `WUPHF_OPENCODE_MODEL=opencode-go/crouton-alpha-free` |
| 蓝图 | `ai-revops`，agents `cos, revops, analyst, ae, reviewer` |
| 主办公室 | 7891/7892，**不得被本轮任何命令指向** |

### 2.1 隔离手段

命名频道不可用（`namedChannelsEnabled()` 硬编码 false），场景仍在 `cos__human`，靠清理隔离，与 J7 同法：

1. onboarding 后、提交场景前，立即取消 `ai-revops-1..4`；
2. 每次快照检查是否出现非场景任务/采访卡，出现即取消并逐条记进 observations，判分单列「环境清理」；
3. 场景本身不带 `definition`（这是让规划者自己决定 lane 写范围的必要条件）。

## 3. 场景

与 Journey 7 **逐字相同**（可比性），父 Issue：`channel=cos__human`、`owner=cos`、`created_by=human`、`task_type=issue`、无 `definition`：

```
title:   "Journey 8: Acme 续约报价与跟进"
details: "客户 Acme（40 席位，SaaS）。现行合同：年付 $18,000，2026-12-31 到期，付款 30 天。
          拟续约：年付 $21,600（+20%），条款不变，有效期 14 天。对方决策人 Dana（VP Ops）。
          两件交付物：① 一页报价说明（价格/范围/有效期/付款/决策点）② 48 小时内的跟进邮件草稿。
          只认以上数字，不要引入其它假设。"
```

预期路径：cos planning → 计划卡 → 批准 → 拆 lane。**本轮分叉**：lane 创建若不带写范围会被拒绝；期望规划者读取拒因，在同一或下一次 create 补 `artifact_scope`（或对该 lane 声明 `artifact_not_required_reason`），随后 owner 用 `/wiki/write-task` 把交付写进 `team/**`。

## 4. 验证项

| 编号 | 判定 | 通过判据 |
|---|---|---|
| V1 | 每条 lane 都带写范围 | `broker-state.json` 里每个子 Issue 都有 `definition.artifact_path` 或 `artifact_not_required`（后者带理由且合理） |
| V2 | 共享 wiki 真的有交付 | `team/**` 出现 ≥1 份 markdown，且 `wiki log` 里对应 commit 的作者是场景 lane 的 owner；`broker.log` 至少 1 条 `write-task` 成功 |
| V3 | 守卫是提示，不是墙 | 规划者撞写范围守卫 ≤ 每条 lane 1 次，且撞后**改做法**（补路径或声明不产出）；不得出现「原样重试同一 create」≥2 次 |
| V4 | 不回归 | 计划卡自动出现且只有一张；`task_not_decomposed` 0 条；`watchdog_alert` 0 条；拆 lane 后 `plan_decomposition_check` 正常收尾带 `status_reason` |
| V5 | 反向臂（操作员探针，停表后执行，不计场景干预） | 用错误路径打 `/wiki/write-task`：403 文案仍点名 creator、仍只投 1 条 `artifact_path_missing` |
| V6 | 不刷屏 | 频道里因写范围被拒的 create 总数 ≤ 5 |

## 5. 硬失败

- **F-A（回退触发）**：同一 lane 因写范围被拒 ≥3 次且 create 内容不变（守卫变成墙）。
- **F-B（回退触发）**：所有 lane 的 `artifact_path` 都是敷衍值（`team/tasks/x.md` 这类不含信息的路径），或声明 `artifact_not_required` 而实际有交付物。
- **F-C**：计划卡缺席（J3-1 回归）。
- **F-D**：拆 lane 成功但 60 分钟内 `team/**` 仍为 0 份（机制通了但没人用，等于失败）。
- **F-E**：批准后 10 分钟内既无子任务也无 `task_not_decomposed`（分解检查回归）。

**回退**：触发 F-A 或 F-B → 切 B（接受 notebook + visual artifact 为交付载体），把 §9.15 降级为「防路径不匹配」，转去修 archivist 从文档抽 fact。这是写死的判据，不在跑完后改。

## 6. 人工干预

**只允许 1 次**：点批准卡。取消 onboarding 遗留、清理非场景任务逐条记进 observations，判分单列为环境清理。停表后的反向臂探针同样单列，不计入场景干预。

## 7. 停表

连续 15 分钟无新消息且无 active 回合，或 60 分钟上限。停表后**先**把 `wiki-commits.txt`、`wiki-tree.txt`、lane notebook、`broker.log` 摘要复制进 `artifacts/`，**再** `workspace shred`。快照每 5 分钟一次到 `snapshots/HHMM/`。
