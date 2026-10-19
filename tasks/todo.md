# Chapter 10 tasks

- [x] Agent `response_completed` event with usage; gateway ignores it. `go test -race ./internal/agent ./internal/gateway` passed.
- [x] `internal/eval/case.go`: YAML schema, strict keys, bounds, grader parsing; `Load` over files and directories. Tests passed.
- [x] `internal/eval/fake.go`: scripted loopback Responses server with synthetic usage, exhaustion and `expect_input_contains` reporting. Tests passed.
- [x] `internal/eval/run.go` + `grade.go`: fresh directories per run, confined tools, approval from the case, calls rebuilt from the session JSONL, ten graders, `--out` logs. Replay tests passed, including denial, order, file, skill, and timeout cases.
- [x] `internal/eval/report.go`: pass@k, pass^k, table, `report.json`. Tests passed.
- [x] `mino eval` command, `runCLI` dispatch, help text, banner, `internal/eval/testdata`, command tests (replay, filter, out, invalid flags, live without config, live against httptest). Built binary smoke run in a temporary HOME: mixed suite exits 1, `evals/` suite passes, no `.mino` created.
- [x] `evals/` repository suite and `scripts/check.sh` hook.
- [x] Docs: `docs/plans/10-evals.md`, plans index, AGENTS.md, CLAUDE.md, CHANGELOG, READMEs, sidebar, tasks. `book_writer` wrote Chapter 10 en/zh and updated the roadmap, index, getting-started, and the Chapter 09 ending; reviewed facts and translation (one clarification: `--live` creates the bundled SOUL when absent). `npm run book:build` passed with the pre-existing large-bundle warning; Figure 10-1 rendered in Chromium in both languages, light and dark, and at a 390 px viewport without page-level horizontal overflow or console errors. SOUL.md unchanged: evals add no model-facing behavior.
- [x] Check steps run individually (gofmt, bash -n, vet, race tests, build, `bin/mino eval ./evals`): all pass except the pre-existing Linux-only Bash process-group test noted below. Committed and pushed to `claude/epic-brown-gfpjwm`.

- [ ] Release: flip docs to released (2026-10-19), squash to one chapter commit, open the PR, and after merge tag `chapter-10` on `main` (authorized by the user) and verify the Release workflow and assets.

Known environment note: this Linux container fails the pre-existing `TestBashLimitsAndCancellation/background` process-group test in `internal/tools` with or without Chapter 10 changes; project CI runs on macOS.
