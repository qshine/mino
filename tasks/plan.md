# Chapter 10 implementation plan

Chapter 10 adds evals: declared YAML cases run through the real Agent loop and are graded with deterministic checks. The user confirmed the scope on 2026-10-16: a complete chapter, two modes (replay by default, opt-in `--live`), and deterministic graders only. Target release: `chapter-10`, version 0.10.0; source builds remain `dev`. The design draft with the pi/hermes research lives in `docs/plans/10-evals.md`.

## Implementation decisions

- `internal/eval` owns the case schema (`case.go`), the scripted loopback Responses server (`fake.go`), the per-run runner (`run.go`), graders (`grade.go`), and the report (`report.go`); `internal/eval_command.go` parses `mino eval` and dispatches from `runCLI` like `mino mcp`.
- Replay mode drives `agent.Handle` against `FakeModel`, so the eval exercises the same loop, approval, policy, and history code as chat. The fake synthesizes token usage at four bytes per token and fails a run when the script runs out or a request lacks an expected substring.
- Every run uses fresh temporary `workspace/`, `home/.mino/`, and `session/` directories, Bash and file tools confined to the workspace, `skill_view` over the seeded `.agents/skills`, `policy.Default()`, and no MCP. The user's `~/.mino` is never read in replay mode; `--live` reads config and SOUL only.
- Graders read the run's session JSONL to rebuild requested calls, whether each started, and its result, so denied and unknown calls are visible without exposing Agent internals. The Agent's only change is a `response_completed` event carrying usage, which the gateway ignores.
- Rates use k = runs: pass@k = 1 − C(n−c,k)/C(n,k); pass^k = C(c,k)/C(n,k). Report prints a table and optional `report.json` plus per-run logs under `--out`.
- `scripts/check.sh` runs `bin/mino eval ./evals` after the build; the repository suite covers approved and denied Bash, a policy-protected `.env`, file creation, and an unknown tool.

## Build order and verification

1. Agent usage event and gateway case; agent and gateway tests.
2. Case schema and validation tests.
3. Fake server and item-shape tests.
4. Runner, graders, and replay tests (approval, denial, order, files, skills, timeout).
5. Report math and output tests.
6. Command, dispatch, help, banner, sample cases, command tests, and a smoke run of the built binary.
7. Design draft, AGENTS/CLAUDE/CHANGELOG/README/sidebar updates, then `book_writer` for Chapter 10 in both languages, `npm run book:build`, and diagram review.
8. `bash scripts/check.sh`.

Tests use Go's standard testing package, temporary homes, local scripted servers, and no real credentials.
