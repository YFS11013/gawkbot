# Agent Instructions

> **先看这里**：涉及 `~/.wuphf` 的运行时（broker、routine、任务状态、数据空间、app 元数据、巡检/排障/变更）
> 时，走 `docs/office/OFFICE-RULES.md`（工作流与硬规则）+ `docs/office/RUNTIME-API.md`（接口契约）。
> 那两处的规则优先于本文件里面向产品代码的流程约定。

## 先读哪份（按任务类型）

本文件只放**进门就要遵守的硬规则**。下面按需展开，不要凭印象：

| 你要做的事 | 读这个 |
|---|---|
| 动 `~/.wuphf` 运行时（broker、routine、任务、数据空间、app 元数据、巡检/排障） | `docs/office/OFFICE-RULES.md`（流程与硬规则）+ `docs/office/RUNTIME-API.md`（接口契约） |
| 查某条链路**到底谁在写**、某个 routine 为何静默失败 | `docs/office/RUNTIME-API.md` §5、`docs/office/DESIGN-DECISIONS.md` §8 |
| 当前待办 / 已定案 | `docs/office/TODO-NEXT.md` |
| 评估"要不要上多 agent / 并行 / 审阅环节"（外部证据：Anthropic、Cognition、IDC） | `docs/office/RESEARCH-2026-11-01-multi-agent.md`（来源与局限）+ `docs/office/DESIGN-DECISIONS.md` §10（四条判据） |
| 找"还能把什么活交给 bot"、给场景排期 | `docs/office/USE-CASES-2026-11-01.md`（按可落地性分层 + 每条的卡点与证据） |
| 多 agent 同树并行（探针、文件归属、临时状态、批量编辑） | `docs/agents/shared-tree-lessons.md` §C 共享工作树与工具可信度 |
| 派 sub-agent | `shared-tree-lessons.md` §Sub-agent dispatch contract（7 条必含项 + 模板） |
| 改 `web/` 提 PR（截图）、写 Storybook | `shared-tree-lessons.md` §Screenshots、§Storybook |
| 高风险设计（wire shape / 安全边界 / 新公共 API） | `shared-tree-lessons.md` §Triangulation、§Verification agents、§When to use which |
| 协议级包改动（demo + 跨语言 oracle） | `shared-tree-lessons.md` §Demo + cross-language oracle |
| 本仓库为何不照搬多轮评审仪式 | `shared-tree-lessons.md` §Multi-round review rhythm |
| Nex 上下文/记忆工具怎么用 | `shared-tree-lessons.md` §Nex Context |

## Base Agent Instructions

These instructions apply to AI coding agents working in Nex repositories. Keep
tool-specific root files (`CLAUDE.md`, `AGENTS.md`, etc.) pointed at the same
canonical repo file so Claude, Codex, and other agents receive equivalent
guidance.

No additional setup is required beyond a normal clone of the repository. The
`.github` repository provides rollout tooling and templates only; committed
repo-local instruction files are the runtime source of truth for contributors.

### Working Rules

- Read the relevant code before editing. Do not reason from assumptions when
  the repository can answer the question.
- Prefer narrow, surgical changes that follow existing repo patterns.
- Do not revert changes you did not make unless the user explicitly asks.
- Own bugs surfaced during the work. Do not dismiss them as unrelated when they
  block the requested outcome.
- When facing a rendering, architecture, or library-choice problem, research mature practice first: spawn sub-agents to collect source-level evidence (file paths + line numbers + code snippets) from 2-3 well-known open-source projects, compare in a table, then decide. No guessing from docs, no gut calls.
- Ask before destructive or hard-to-reverse actions: deleting state, clearing
  Docker volumes, applying migrations outside local dev, or changing production
  infrastructure.

### Git And PRs

- Never push directly to `main` (repos with other contributors; see each repo's
  own profile below for exceptions).
- Use a branch and open a draft PR for code changes.
- Use Conventional Commits for commit messages.
- Run the repo's documented checks before opening or marking a PR ready.
- Do not use `--no-verify` to bypass hooks.

### Quality Bar

- Do not suppress lint or type errors with ignore comments. Fix the code.
- Do not introduce explicit `any` in TypeScript. Use specific types, `unknown`
  with narrowing, or preserve existing untyped boundaries.
- Do not commit secrets, tokens, credentials, or inline API keys.
- Source required secrets from environment files, secret managers, or the
  repo-documented local setup.
- Treat E2E failures as product signals. Do not hand-wave them away.

## Wuphf Agent Instructions

Use this profile for the Wuphf public repo and its worktrees.

### Commands

```bash
go build -o wuphf ./cmd/wuphf
bash scripts/test-go.sh
bash scripts/test-go.sh ./internal/team
bash scripts/test-web.sh
bash scripts/test-web.sh web/src/path/to/file.test.ts
```

Web UI commands run from `web/`:

```bash
bun install
bun run dev
bun run build
bunx tsc --noEmit
```

Always use `bun` / `bunx` for JavaScript tooling in this repo. Web unit and
component tests run through Vitest. Use `bash scripts/test-web.sh` for the full
Web suite and `bash scripts/test-web.sh web/src/path/to/file.test.ts` for
focused Web tests; do not use `bun test` inside `web/`, because that invokes
Bun's native test runner instead of the repo's Vitest setup.

### Landing Changes (this fork — overrides "Git And PRs" above)

- **`origin` = 上游作者仓库（无写权限，推送 403）；`mine` = 本机 fork。只允许推 `mine`，`origin` 永不推送。**
- 双保险已配置：`git config push.default nothing`（裸 `git push` 直接拒绝）+ `.git/hooks/pre-push`（显式 refspec 也拦）。确需推送时：`WUPHF_ALLOW_PUSH=1 git push mine <branch>`。
- 工作分支 `office-fixes`，跟踪 `mine/office-fixes`。
- 上游那条“commit to main and push directly / 不需要 PR 仪式”是**原作者仓库**的规则，对本 fork 不适用：改完先跑本地检查，再 commit（Conventional Commits，commitlint 会跑）。
- **pre-push 钩子与本地检查不是可选项**：永不 `--no-verify`。lefthook pre-push 会跑 Go 套件、web typecheck + 测试、build、vet、vhs。
- 新 worktree 第一次工作前：`./scripts/bootstrap.sh`，并在 `web/`、`agent/` 各跑一次 `bun install`（缺 `node_modules` 会让 pre-push 的 web 钩子失败）。
- Gotcha：别的 worktree 已占 `main` 时 `git checkout -B main` 会失败；改在 detached HEAD 上工作。

### Lint And Security

- Go: `gofmt`, `go vet ./...`, and `golangci-lint run ./...`.
- Web: `bun run lint:fix` from `web/`. It covers `src/` **and**
  `public/themes/` — the theme files are stylesheets like any other, and
  scoping the command to `src/` alone let a formatting error sit in
  `nex-shell.css` unnoticed.
- Secrets: `bunx secretlint`.
- Do not suppress lint warnings with ignore comments.

## 已外置的章节（改它们请同时更新上面路由表）

`docs/agents/shared-tree-lessons.md`：Nex Context、Triangulation through orthogonal sub-agents、Verification agents as sounding boards、When to use which、Screenshots、Storybook (web/)、Multi-round review rhythm、Sub-agent dispatch contract、Diagnostic probes in a shared tree、A verified fact about a shared tree has a shelf life、File ownership needs an escape hatch、Never create a state you intend to undo, in a shared tree、After a bulk edit, check what was REMOVED、A tool reporting success is not evidence that it did the thing、A guard that can go quiet is worse than no guard、A NUL byte makes a source file invisible to every text tool、A caveat that names the wrong item is worse than a vague one、Never drive the shared browser、Worktree-based parallelism、Demo + cross-language oracle for protocol-grade packages
