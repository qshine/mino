# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

`AGENTS.md` (imported above) is the authoritative source for project rules, chapter scope, storage formats, safety controls, and the book workflow. Edit rules there, not here; this file only adds Claude Code–specific notes.

## Commands

Beyond the commands listed in `AGENTS.md`:

- `go test ./internal/agent -run TestName -v`: run a single test (add `-race` to match `scripts/check.sh`).
- `go test -race -timeout 90s ./...`: the test step that `scripts/check.sh` runs.
- `npm run book:dev` / `npm run book:build`: serve or build the VitePress book (Node >= 22; run `npm ci --ignore-scripts` first if `node_modules` is missing).
- `bash book_review.sh`: install book dependencies if needed, build, and open a local preview.

## Architecture overview

- `internal/app.go` is the composition root. `runCLI` routes `mino mcp ...` to `runMCP` (`internal/mcp_command.go`) and `version`/`update` to `gateway.CLI.Command`. Everything else goes to `runChat`, which runs these steps in order:
  1. Load or prompt the config, then load `~/.mino/SOUL.md`.
  2. Build Bash (`tools.NewBash(cwd, home)`), the file tools (`tools.NewFiles`, rooted at the Bash directory) and `skill_view`.
  3. Load `~/.mino/policy.json` (`policy.Load`) and `~/.mino/mcp.json` (`mcp.Load`), connect the MCP servers (`mcp.Connect`) and append their tools.
  4. Open `agent.OpenSessions(~/.mino, tools)`, wrap it in `agent.NewSessionManager`, then hand control to `gateway.CLI.Run`.

  The same tool list goes to both the Session (for replay validation) and the Agent.
- `internal/gateway/` depends on `agent` through the `Handler`/`Interaction`/`Event`/`Confirmation` contract in `internal/agent/agent.go`; the reverse import is forbidden.
- `internal/agent/`:
  - `agent.go`: `Handle` persists the user message, then `runLoop` streams Responses API calls and enforces request/tool limits. It offers only the tools that `Policy.Offered` allows. `runCall` runs each call in this order: `Prepare`, then `Policy.Decide`, then `Inspect` (live only), then confirmation, then `Execute`.
  - `session.go`: `Session` and the history record state machine (`historyState.apply`), which validates every record on load and on append; memory changes only after write and sync succeed. Replay calls `Prepare` on recorded arguments; an `mcp_` tool that is no longer configured is replaced with `mcp.Unavailable`, which never executes.
  - `history.go`: JSONL file opening, locking, private permissions, tail recovery, and directory sync.
  - `sessions.go` / `session_commands.go` / `session_migration.go`: multi-session store, slash commands, and Chapter 02 history migration.
  - `compact.go`: context budget estimate and summary compaction.
- `internal/tools/`: SDK-independent `Tool` interface (`Definition`, `Prepare`, `Execute`), which the Agent translates to and from Responses types.
  - Some `Call` fields are set only by trusted tool code, never decoded from model arguments: `ReadOnly` and `Target` feed the policy, and `Expect` is the inspected-content digest that `Execute` rechecks. This is how authorization reaches the Agent.
  - The optional `Inspector` (used by the file tools in `file.go`) is the only step allowed to touch the filesystem before approval, because `Prepare` must stay side-effect free for replay.
- `internal/policy/`: loads `policy.json` and returns the allow/ask/deny decision for each call.
- `internal/mcp/`: `config.go` handles `mcp.json` (shared with `mino mcp`); `client.go` connects over HTTP or stdio, discovers tools, and adapts them to `tools.Tool`.
- `internal/skills/`: project `.agents/skills` discovery and confined reads used by `skill_view`.
- Configuration and SOUL (`internal/config.go`, `internal/soul.go`) live in package `mino`; `assets.go` at the root embeds `install.sh` and the default `SOUL.md`.
- Tests in `internal/agent/` drive the real loop against local `httptest` Responses servers, building Agents and isolated Sessions with the helpers in `fixture_test.go`; per-feature suites follow the chapter names (`files_test.go`, `mcp_test.go`, `guardrails_test.go`, `compact_*_test.go`, ...).
- Working notes for a chapter go in `tasks/plan.md` and `tasks/todo.md`. They may still describe an older chapter, so check them against `AGENTS.md` before relying on them.

## Book writer in Claude Code

The project subagent `book-writer` (`.claude/agents/book-writer.md`) is the Claude Code counterpart of the Codex `book_writer` role. After code changes and checks pass, launch it with the diff, behavior changes, verification evidence, and version, as `AGENTS.md` requires. It loads `.codex/agents/book_writer.toml` at startup, so that file stays the only writing contract: change writing rules there, not in the subagent file. The writer only edits `docs/books/`; the parent still updates README files and the changelog, reviews the result, and commits.
