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
- In Linux containers whose PID 1 does not reap orphaned processes (such as Claude Code cloud sessions), `TestBashLimitsAndCancellation/background` reports `child … survived` because the killed child stays a zombie. macOS and CI are unaffected; run the checks under a process that sets `PR_SET_CHILD_SUBREAPER` and reaps children.

## Architecture overview

- `internal/app.go` is the composition root. `runCLI` routes `mino mcp ...` to `runMCP` (`internal/mcp_command.go`) and `version`/`update` to `gateway.CLI.Command`. Everything else goes to `runChat`, which runs these steps in order:
  1. Load or prompt the config, then load `~/.mino/SOUL.md`.
  2. Build Bash (`tools.NewBash(cwd, home)`), the file tools (`tools.NewFiles`, rooted at the Bash directory) and `skill_view`.
  3. Load `~/.mino/mcp.json` (`mcp.Load`), connect the MCP servers (`mcp.Connect`) and append their tools.
  4. Open `agent.OpenSessions(~/.mino, tools)`, wrap it in `agent.NewSessionManager`, then hand control to `gateway.CLI.Run`.

  The same tool list goes to both the Session (for replay validation) and the Agent.
- `internal/gateway/` depends on `agent` through the `Handler`/`Interaction`/`Event`/`Confirmation` contract in `internal/agent/agent.go`; the reverse import is forbidden.
- `internal/agent/`:
  - `agent.go`: `Handle` persists the user message, then `runLoop` streams Responses API calls and enforces request/tool limits. `runCall` runs each call in this order: `Prepare`, then `Policy.Decide`, then `Inspect` (live only), then confirmation, then `Execute`.
  - `session.go`: `Session` and the history record state machine (`historyState.apply`), which validates every record on load and on append; memory changes only after write and sync succeed. Replay calls `Prepare` on recorded arguments; an `mcp_` tool that is no longer configured is replaced with `mcp.Unavailable`, which never executes.
  - `history.go`: JSONL file opening, locking, private permissions, tail recovery, and directory sync. `OpenSession` opens the single conversation in `sessions/` the way Chapters 02–04 did; tests use it.
  - `sessions.go` / `session_commands.go`: multi-session store (`OpenSessions`, which adopts a lone pointer-less session) and slash commands.
  - `compact.go`: context budget estimate and summary compaction.
- `internal/tools/`: SDK-independent `Tool` interface (`Definition`, `Prepare`, `Execute`), which the Agent translates to and from Responses types.
  - Some `Call` fields are set only by trusted tool code, never decoded from model arguments: `ReadOnly` and `Target` feed the policy, and `Expect` is the inspected-content digest that `Execute` rechecks. This is how authorization reaches the Agent.
  - The optional `Inspector` (used by the file tools in `file.go`) is the only step allowed to touch the filesystem before approval, because `Prepare` must stay side-effect free for replay.
- `internal/policy/`: holds one session's permission mode (`manual`, `accept-edits`, `bypass`; `SessionManager` keeps one `Policy` per session ID and `/mode` in `session_commands.go` changes the current one) and the built-in protected paths, and returns the allow/ask/deny decision for each call.
- `internal/mcp/`: `config.go` handles `mcp.json` (shared with `mino mcp`); `client.go` connects over HTTP or stdio, discovers tools, and adapts them to `tools.Tool`.
- `internal/skills/`: project `.agents/skills` discovery and confined reads used by `skill_view`.
- Tests in `internal/agent/` drive the real loop against local `httptest` Responses servers, building Agents and isolated Sessions with the helpers in `fixture_test.go`; per-feature suites follow the chapter names (`files_test.go`, `mcp_test.go`, `guardrails_test.go`, `compact_*_test.go`, ...). To meet the HOME-isolation rule, use `isolateConfig(t)` (in `internal/agent/fixture_test.go` and `internal/config_setup_test.go`) or `t.Setenv("HOME", t.TempDir())`.

## Finishing a change in Claude Code

`AGENTS.md` spreads the handoff steps across several sections. In Claude Code, do them in this order:

1. Run `bash scripts/check.sh`. It fails on unformatted files, so run `gofmt -w` on them first.
2. Launch the `book-writer` subagent (`.claude/agents/book-writer.md`) with the diff, behavior changes, verification evidence, and version. It is the Claude Code counterpart of the Codex `book_writer` role. It reads `.codex/agents/book_writer.toml` at startup, so change writing rules in that file, not in the subagent. Every code change needs this review; a no-impact result must give its reason.
3. The writer edits only `docs/books/`. You update README files, `CHANGELOG.md`, and the default `SOUL.md` when the writer or the change requires it.
4. Review the writer's facts and translation, run `npm run book:build`, and check changed Mermaid diagrams in a browser.
5. Commit. Building the book does not authorize publishing, tagging, or releasing.
