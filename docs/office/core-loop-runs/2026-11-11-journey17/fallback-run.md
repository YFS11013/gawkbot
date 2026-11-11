# Journey 17 — 换主模型跑通全环（机制验证，不计分）

2026-11-11 05:13–05:24 UTC，隔离办公室 `:7910`。**这不是一次计分的 journey**：prereg（`5696d9c`）§8 要求 `opencode-go/longcat-2.5-free`，而该模型从 02:07 起一直 429（05:04 再探仍 429），所以本轮按 prereg 换不了；人工角色仍然是我，跨轮可比性没有。它要回答的只有一个问题——**D3 的修法在真机上有没有效，以及办公室在换到有额度的模型后能不能把整个环走完**。

## 与 prereg 的偏离

| 项 | prereg | 本轮 |
|---|---|---|
| 主模型 | `opencode-go/longcat-2.5-free` | `opencode-go/iceman-contributor-free`（longcat 429，iceman 200） |
| fallback | `iceman-contributor-free` | 置空（两个模型不能是同一个） |
| 判分 | 独立子代理按 V1–V7 判 | 不判分，只记现象 |

种子文案、死数字、频道（`cos__human`）、操作节奏（采访卡出现后等 4 分钟再答）与 prereg 逐字一致。

## 结果

**D3 成立。** 采访卡 `request-3` 05:13:49 建；人 05:17:49 答（整 4 分 00 秒，故意跨过原来会失败的那个窗口）。模型**没有**再调一次 `human_interview`、也没有在频道里手工重问，05:17:49 起的新回合 37 秒内正常收下答案并往下走（`msg-10/11/12`，`total_ms=37038`）。对照 J16：同一节奏下模型连调三次工具、自述「采访卡片工具连续超时」、把同一个问题在频道里重问一遍。

**整环走通，三张卡、零 lane 卡。**

| 时刻 (UTC) | 事件 |
|---|---|
| 05:13:49 | 采访卡 `request-3` |
| 05:14:57 | 计划批准卡 `request-7`（同回合建出） |
| 05:17:49 | 人答采访（+4:00）→ 同回合继续 |
| 05:19:07 | 人批计划 → 拆出两条 lane `JOURN-16`（报价单）/ `JOURN-18`（跟进邮件） |
| 05:20:38 | 两条 lane `done/approved`，容器卡 `request-28` |
| 05:23:42 | 人批容器 → `JOURN-1` `done/approved` |

- 交付物按声明路径落盘且内容含全部死数字：`team/accounts/acme-renewal-quote.md`（819 B，40 席位 / $18,000 → $21,600 / 2026-12-31 / Dana / Alex）、`team/emails/acme-renewal-followup.md`（618 B，CTA「本周内确认并回签」）。
- `hollow_completion` 三条全 false；`artifact_overwritten` 0 条；`headless-codex-latency.log` 10 条、`status=error` 0 条。
- 守卫被动触发 2 次（`action-17`、`action-20`，都是「容器带着两条已交付 lane 想自己关单」），与 J15/J16 同一形状——模型乱撞、守卫拦下、随后正常发起关单卡。
- 人需作答的卡 **3 张**（采访、计划、容器），与 J16 相同；没有 lane 自带的决策卡（§9.26 决策 ① 继续保持）。

## 顺带得到的结论

**单模型限流是可存活的。** 同一份工程在两小时内经历了两种形态：主模型可用（本轮，全环 12 分钟走完）与主模型整天 429（04:04–04:14，静默 10 分钟后 fallback 接手）。§9.28 的两条修法各管一种：marker 管「快速失败的 429」，零输出超时管「慢速挂死的 429」。

## 处置

本轮不作 J17 计分；prereg 保持原样，longcat 配额恢复后仍按 prereg 重跑才算数。运行时已 `workspace shred --yes journey17`。
