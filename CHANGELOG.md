# Changelog

Versions follow the tutorial chapters: `0.<chapter>.<patch>`. Git tags use
`chapter-NN` for each chapter's initial release and `chapter-NN.PATCH` for later
fixes (for example, `chapter-04.1` means `0.4.1`). The main history has one
complete commit per chapter.

At the owner's request, the chapter snapshots were rebuilt with consistent
Gateway/Agent directory boundaries and chapter-prefixed tags. Chapters 01 and 02
were reissued, and Chapters 03 and 04 published, on 2026-09-26. All four installers
understand chapter tags and numeric version aliases. This authorized replacement
is an exception; future fixes receive new patch tags.

## [0.3.0] - 2026-09-26

### Added

- Chapter 03 Bash tool and Agent loop (`chapter-03`, version `0.3.0`). `internal/tools/` holds tool
  definitions, strict arguments, and execution; calls and results retain their Responses `call_id` during replay.
- Per-call, default-deny interactive approval showing the escaped command, real
  working directory, minimal environment, and limits. Piped input cannot approve.
  Commands use a 30 second timeout, a combined 64 KiB output cap, and process-group
  cleanup; turns stop at 8 model requests or 16 calls. Bash is not an OS sandbox.
- Durable tool start/result records and restart recovery without re-execution.
  Unknown results require acknowledgement before chat, retained across restarts.
- Bilingual Chapter 03 lesson and tests for real Bash with a mock model, approval,
  invalid and duplicate calls, limits, cancellation, storage failures, and recovery.


### Changed

- Separate CLI Gateway from Agent processing with synchronous interaction contracts
  and constructor injection. `Agent.Handle` saves the user message before `runLoop`,
  which directly calls the Responses SDK and saves every completed response.
- Make `Session` the owner of in-memory history, committing changes only after
  JSONL writes and syncs succeed. Existing history files remain compatible.
- Register tools through `Tool.Definition`, `Prepare`, and `Execute`; a new tool
  needs no Bash-specific branch in the Agent loop. Non-process tools may omit an
  exit code. Bash keeps its existing execution limits and approval requirements.
- Use generic CLI approval fields and `Approve this operation? [y/N]` for the
  shared tool contract. The default remains denial.

- History writes format `v: 2` and still reads Chapter 02's `v: 1`. Older releases
  reject new records; retain a private pre-upgrade backup if rollback is needed.
  Paired tool turns remain in context after failure or interruption with an
  explicit notice; incomplete text-only turns are still excluded.
- Default SOUL now describes approved Bash use and untrusted tool results. Existing
  user SOUL files remain untouched; update obsolete no-tools capability statements
  manually when upgrading.

- Adapt the book writer's narrative style from `khazix-writer` while keeping
  factual evidence, engineering tradeoffs, and bilingual tutorial conventions
  explicit; document the writing process in both maintenance guides.
- Put the tutorial's audience, interaction-based learning approach, and completed
  chapters before installation details in the READMEs; make the same information
  prominent on both book homepages.

## [0.2.0] - 2026-09-26

### Added

- Chapter 02 JSONL history in `~/.mino/history.jsonl`: one conversation per file,
  ordered turns paired by `turn_id`, and recovery of completed turns on restart.
  Preserve Responses output items, assistant phase, and encrypted reasoning state
  for stateless replay while continuing to stream answers immediately.
- Private history and lock files, a single-writer lock, incomplete-tail backups,
  record-order validation, and bounded file sizes. Failed or interrupted turns
  stay recorded but do not enter the next request; storage failures stop chat.
- Bilingual Chapter 02 lessons and tests for restart continuity, streaming,
  cancellation, write failures, recovery, and file locking.
- Author pages and `book_review.sh` for local bilingual book review.

### Changed

- Reissue Chapter 02 with terminal interaction in `internal/gateway/` and model
  requests, Agent handling, and the history Session in `internal/agent/`.
  The existing `v: 1` JSONL format remains unchanged; no tools or multiple-session
  commands are introduced in this chapter.
- Rerun the current README installer with `chapter-02` to receive the replacement,
  even when already on `0.2.0`; older installed updaters do not understand the new
  tag names. Existing settings, SOUL, and history are preserved.
- Center the bilingual tutorial on observable interactions and engineering choices.

## [0.1.0] - 2026-09-26

### Added

- Chapter 01 independent terminal conversations through the OpenAI Responses API
  and official Go SDK. Stream answer and refusal fragments immediately; report
  interruption or failure without repeating the answer or resubmitting a request.
- Explicit model and API-key setup, private `~/.mino/config.json`, and an editable
  `~/.mino/SOUL.md` loaded once at startup. Preserve existing settings and identity.
- Verified Apple Silicon and Intel macOS packages, `mino version`, and
  `mino update`. Public downloads use `curl` without GitHub CLI or login.
- Matching English and Chinese lessons, deterministic mock-service tests,
  installation checks, and automated release packaging.

### Changed

- Reissue Chapter 01 with `cmd/mino` as the entry point, `internal/app.go` as
  composition, `internal/gateway/` for terminal interaction and commands, and
  `internal/agent/` for model requests. Configuration and SOUL stay in `internal/`.
- This chapter still sends only the current question and instructions: no
  conversation history, tools, or session management is included.
- Rerun the current README installer with `chapter-01`, even if the version
  already reads `0.1.0`. Older installed updaters do not understand chapter tags.
  Existing settings and SOUL are preserved.

### Fixed

- Prevent duplicate Mermaid rendering after the book loads.
