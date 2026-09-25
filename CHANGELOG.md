# Changelog

Versions follow the tutorial chapters: `0.<chapter>.<patch>`. Git tags use
`chapter-NN` for each chapter's initial release and `chapter-NN.PATCH` for later
fixes (for example, `chapter-05.1` means `0.5.1`). The main history has one
complete commit per chapter, and each chapter tag points at that commit.

On 2026-10-21, at the owner's request, the tutorial was restructured from nine
chapters into ten and every chapter tag and release was republished. The former
Chapter 03 was split into Chapter 03 (the Agent loop) and Chapter 04 (recovering
interrupted tool turns); later chapters moved up by one. Chapter 02 now stores its
single conversation in the session directory used by Chapter 05, so the former
`history.jsonl` migration no longer exists. Releases published before that date
used the old numbering and are superseded; sessions they wrote under
`~/.mino/sessions/` remain readable, but a legacy `~/.mino/history.jsonl` is no
longer imported. This authorized replacement is an exception; future fixes receive
new patch tags.

## [Unreleased]

## [0.5.0] - 2026-10-21

### Added

- Chapter 05 multiple sessions: `/new`, `/sessions`, `/resume <id>`, `/clear`, and
  `/help`. Each private JSONL file carries its session ID in its filename; startup
  restores the last selection from `active-session.json`.
- Interactive clearing requires the complete current session ID. It keeps other
  sessions, configuration, SOUL, and recovery backups; it does not undo commands or
  promise secure disk erasure.
- The single conversation left by Chapters 02–04 is selected in place on first
  start, because it already lives in `~/.mino/sessions/`; nothing is copied.

### Changed

- Serialize session commands and Agent turns with the store-wide `sessions.lock`.
  Selection damage, or a missing selection with several sessions, enters a
  command-only recovery state; storage write failures stop chat.
- Publish selection and clearing atomically after syncing temporary files, then
  sync their directory before advancing memory. Tool protocol and record versions
  remain unchanged. Existing SOUL files remain untouched.

## [0.4.0] - 2026-10-21

### Added

- Chapter 04 recovery of interrupted tool turns. Failed, cancelled, or interrupted
  turns that contain tool calls are replayed with their paired results and an
  explicit Mino runtime notice, so the model can see that commands may have had
  effects. Text-only unfinished turns are still excluded.
- Unknown results (a saved start without a saved result) require acknowledgement at
  startup before chat continues. The acknowledgement is saved as a `recovery_ack`
  record and survives restarts; piped input cannot acknowledge. Nothing is rerun.

### Changed

- Records remain version 2. `chapter-03` cannot read a session after a
  `recovery_ack` record has been written; `chapter-04` reads `chapter-03` sessions
  and asks for acknowledgement when they contain unknown results.
- The default SOUL tells the model never to retry an unknown operation automatically.

## [0.3.0] - 2026-10-21

### Added

- Chapter 03 Bash tool and Agent loop. `internal/tools/` holds tool definitions,
  strict arguments, and execution; calls and results retain their Responses
  `call_id` during replay. Tools implement `Definition`, `Prepare`, and `Execute`,
  so a new tool needs no Bash-specific branch in the Agent loop.
- Per-call, default-deny interactive approval showing the escaped command, real
  working directory, minimal environment, and limits. Piped input cannot approve.
  Commands use a 30 second timeout, a combined 64 KiB output cap, and process-group
  cleanup; turns stop at 8 model requests or 16 calls. Bash is not an OS sandbox.
- Durable tool start and result records: each step is synced before the next. On
  restart, unfinished tool turns are closed without re-execution (`not_executed` or
  `unknown`) and excluded from context with a notice.

### Changed

- Separate CLI Gateway from Agent processing with synchronous interaction contracts
  and constructor injection. `Agent.Handle` saves the user message before `runLoop`,
  which directly calls the Responses SDK and saves every completed response.
- Make `Session` the owner of in-memory history, committing changes only after
  JSONL writes and syncs succeed.
- History writes format `v: 2` and still reads Chapter 02's `v: 1`. Older releases
  reject new records; retain a private pre-upgrade backup if rollback is needed.
- Default SOUL now describes approved Bash use and untrusted tool results. Existing
  user SOUL files remain untouched.

## [0.2.0] - 2026-10-21

### Added

- Chapter 02 JSONL history: one conversation in `~/.mino/sessions/<session_id>.jsonl`,
  named by an ID generated on first start, with ordered turns paired by `turn_id` and
  recovery of completed turns on restart. Preserve Responses output items, assistant
  phase, and encrypted reasoning state for stateless replay while continuing to
  stream answers immediately.
- Private session directory, files, and `sessions.lock`; a single-writer lock,
  incomplete-tail backups, record-order validation, and bounded file sizes. Failed
  or interrupted turns stay recorded but do not enter the next request; storage
  failures stop chat.

### Changed

- Terminal interaction stays in `internal/gateway/`; model requests, Agent handling,
  and the history Session live in `internal/agent/`. No tools or multiple-session
  commands are introduced in this chapter.

## [0.1.0] - 2026-10-21

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
