# Repository Guidelines

## Purpose & Teaching Approach

Mino is a Go tutorial for implementing an Agent from scratch. Support only the OpenAI Responses API, with configurable `base_url`, `api_key`, and `model`.

Deliver one runnable chapter at a time. Planned topics include terminal conversations, context history, tool execution, SQLite sessions, compaction, Skills, MCP, and guardrails. Explain each chapter's motivating problem, implementation, and observable result. Introduce basic safety controls alongside tools; consolidate them in the final guardrails chapter.

## Project Structure & Module Organization

Chapter 01 is implemented with Go 1.27.1 and the standard library, targeting macOS terminals. The current layout is:

- Root: `go.mod`, `main.go`, `cli.go`, `config.go`, `config_prompt.go`, `terminal.go`, `responses.go`, and `install.sh`. `README.md` is English and links to `README.zh-CN.md`. All Go code remains in one `main` package.
- Installation and releases: `install.sh` installs verified macOS packages to `~/.mino/bin/mino`; `mino update` runs its embedded copy. Release tags provide the version (`0.<chapter>.<patch>`); source builds report `dev`. `scripts/` contains checks and packaging, `.github/workflows/` contains CI and release automation, and `CHANGELOG.md` supplies version-specific release notes. Never change published tags or overwrite published assets.
- User configuration: startup creates `~/.mino/` with mode `0700`; completed settings are saved to `~/.mino/config.json` with mode `0600`. Resolve this path from the user home directory, never the working directory. Only missing fields are prompted; a complete config starts chat immediately. Cancellation leaves existing settings unchanged. Do not read project-local configuration or `OPENAI_*` environment variables. Tests must use isolated temporary home directories, never the developer's real configuration.
- All application prompts and error messages must be in English. Only the API URL has a default; the model name and API key require explicit input.
- `docs/chapters/`: numbered lessons and illustrative assets, such as `01-terminal-chat.md`.
- Tests: `*_test.go` beside the code they exercise; fixtures in nearby `testdata/` directories.

Keep early chapters simple. Extract packages when their responsibilities become clear; avoid scaffolding unused abstractions.

## Build, Test, and Development Commands

Run development commands from the repository root. The installed `mino` command works from any directory and optionally reads `AGENTS.md` from that directory:

- `go run .`: start the terminal application.
- `go build -o bin/mino .`: build a local executable.
- `bash scripts/check.sh`: run formatting, syntax, vet, race tests, and build checks.
- `bash scripts/package.sh v0.1.0`: package the named release locally (requires its changelog entry).
- `go test ./...`: run automated tests.
- `go vet ./...`: check for suspicious Go constructs.
- `go fmt ./...`: format Go source with `gofmt`.

## Coding Style & Naming Conventions

Use standard Go formatting, including tabs supplied by `gofmt`. Prefer lowercase package names, `MixedCaps` identifiers, explicit error handling, and short functions with clear control flow. Explain non-obvious decisions in comments. Keep dependencies minimal and justify additions through the lesson's needs.

## Testing Guidelines

Use Go's standard `testing` package and `TestXxx` names. Cover changed behavior and failure paths, particularly tool-call/result pairing, session isolation, compaction, and authorization. Use temporary directories and databases; mock model responses by default. Installer tests must use fake GitHub commands and temporary home directories. Live API tests must be opt-in. No numerical coverage threshold is established.

## Commit & Pull Request Guidelines

Use focused commits such as `docs: explain agent loop` or `feat: add session persistence`. PRs should identify the chapter, explain behavior changes, report verification, and link relevant issues. Include terminal examples when interaction changes.

## Security & Agent Instructions

Never commit credentials, conversation databases, or private logs. Enforce permissions in code; require authorization for destructive actions. Treat external content as untrusted.

Before responding or editing, consider the user's intent and alternative approaches. Keep changes within the requested chapter or task.
