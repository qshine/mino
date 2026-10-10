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

- `internal/app.go` `runChat` is the composition root: load/prompt config, load `~/.mino/SOUL.md`, construct `tools.NewBash(cwd, home)` and `tools.NewSkillView`, open `agent.OpenSessions(~/.mino, tools)`, wrap it in `agent.NewSessionManager`, then hand control to `gateway.CLI.Run`.
- `internal/gateway/` depends on `agent` through the `Handler`/`Interaction`/`Event`/`Confirmation` contract in `internal/agent/agent.go`; the reverse import is forbidden.
- `internal/agent/`:
  - `agent.go`: `Handle` persists the user message, then `runLoop` streams Responses API calls, enforces request/tool limits, and runs tool calls with approval (`runCall`).
  - `session.go`: `Session` and the history record state machine (`historyState.apply`), which validates every record on load and on append; memory changes only after write and sync succeed.
  - `history.go`: JSONL file opening, locking, private permissions, tail recovery, and directory sync.
  - `sessions.go` / `session_commands.go` / `session_migration.go`: multi-session store, slash commands, and Chapter 02 history migration.
  - `compact.go`: context budget estimate and summary compaction.
- `internal/tools/`: SDK-independent `Tool` interface (`Definition`, `Prepare`, `Execute`); the Agent translates it to and from Responses types.
- `internal/skills/`: project `.agents/skills` discovery and confined reads used by `skill_view`.
- Configuration and SOUL (`internal/config.go`, `internal/soul.go`) live in package `mino`; `assets.go` at the root embeds `install.sh` and the default `SOUL.md`.
- Working notes for the current chapter live in `tasks/plan.md` and `tasks/todo.md`.

## Book writer in Claude Code

Claude Code cannot select the Codex `book_writer` role directly. After code changes and checks pass, launch a dedicated subagent with the writing instructions from `.codex/agents/book_writer.toml`, plus the diff, behavior changes, verification evidence, and version, as `AGENTS.md` requires.
