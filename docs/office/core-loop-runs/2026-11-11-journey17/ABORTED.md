# Journey 17 — 本轮作废（上游限流），计划保留

写于 2026-11-11 02:35 UTC。**这不是一次有效的 journey**：模型一次都没有产出，没有任何判据可判。prereg（`5696d9c`）仍然有效，等配额恢复后按原样重跑。

## 发生了什么

| UTC | 事件 |
|---|---|
| 02:07:12 | 操作员建 Issue `JOURN-1`；`headless-codex-latency.log`：`bot=cos stage=started queue_wait_ms=67` |
| 02:07:22 / 02:07:37 | `opencode.log` 两条 `stream error … providerID=opencode-go modelID=longcat-2.5-free`（第二条 `small=true agent=title`） |
| 02:17:12 | 办公室判定该回合失败：`JOURN-1` → `blocked`，系统消息 `msg-3 Blocked — @cos on JOURN-1`，并自建止血单 `JOURN-4 [@cos] Bot stuck on: …`（`action-3`） |
| 02:20:29 | 操作员补发一条人类消息催办（`msg-5`，tagged cos） |
| 02:21:02 / 02:21:04 | 重试回合同样两条 `stream error` |
| 02:31:06 | 直连探测模型端点仍 `429 rate_limit_error` |

## 根因（直接证据，不是推断）

用 `~/.local/share/opencode/auth.json` 的 `opencode-go` key 直连 `https://opencode.ai/inference/go/v1/chat/completions`，模型 `longcat-2.5-free`：

```
02:29:xx  http=429  {"error":{"type":"rate_limit_error","message":"Rate limit exceeded. Please try again later."}}
02:31:06  http=429  （同样）
```

`opencode.log` 里 8 条 `stream error` 全部指向同一 provider/model。所以这不是 broker 的调度问题，也不是本轮要验的 D3（`human_interview` 轮询）——模型根本没跑到工具调用那一步。

## 本轮判什么、不判什么

- **不判**：prereg 的 V1–V7 全部无对象（没有 request、没有消息、没有 lane）。
- **可记的事实**：失败回合的自愈路径是完整的——失败 → `blocked` → 自动止血单 → 人类催办后会重新起回合。止血单 `JOURN-4` 属正常行为（见 runbook §3a）。
- **不记入跨轮对比**：J15/J16 的卡片数与守卫行为不受影响。

## 怎么重跑

1. 等配额恢复（探测命令：`curl -s -o /dev/null -w '%{http_code}' https://opencode.ai/inference/go/v1/chat/completions -H "Authorization: Bearer <key>" -H 'x-opencode-session: probe' -H 'Content-Type: application/json' -d '{"model":"longcat-2.5-free","messages":[{"role":"user","content":"ok"}],"max_tokens":5}'`，要 200）。
2. `workspace shred --yes journey17` 清掉这次残留（含 `JOURN-1`/`JOURN-4`）。
3. `workspace create --from-scratch journey17` → 按端口杀掉重起（完整 env 块，见 J16 D1 / prereg §7）。
4. 原样按 `prereg.md` 执行，包括「采访卡出现后等 4 分钟再答」。

## 环境

| 项 | 值 |
|---|---|
| 源码 / 二进制 | `a8272ba` / `376955ac…` |
| runtime_home | `C:\Users\ireke\.wuphf-spaces\journey17`（已 shred） |
| 主办公室 | `:7892` 正常（`/health` ok、`provider_model=opencode-go/longcat-2.5-free`），同一配额但当时无活跃回合 |
