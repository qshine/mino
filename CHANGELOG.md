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

## [Unreleased]

## [0.9.0] - 2026-10-12

### Added

- Chapter 09 guardrails: every live tool call passes one policy decision (allow, ask,
  or deny) before inspection or confirmation. Denied calls return a `denied` result
  explaining the policy and never run; protected files are never opened, even to
  build a confirmation.
- Optional `~/.mino/policy.json`, owned by the user and not writable by others:
  `bash` (`ask` or `deny`), additional `protected_paths`, `allow_tools` for MCP tools
  that may run without confirmation, and `deny_tools`. It is read once at startup and
  never from a project. Startup prints a one-line policy summary.
- Default protected paths: `.env`, `.env.*`, `*.pem`, `*.key`, `*.p12`, SSH keys,
  `.netrc`, `.npmrc`, `.pypirc`, and the `.git/`, `.ssh/`, `.aws/`, and `.gnupg/`
  directories, matched on every path component.
- Denied tools, including Bash in controlled mode, are not offered to the model.
- Protected-path matching ignores case, matching the default macOS file system.
- Calls allowed by `allow_tools` still print a line naming the tool; policy names that
  match no available tool produce a startup warning.

### Changed

- File tools refuse symbolic links in directory components, so a project link cannot
  rename a protected directory.
- `allow_tools` accepts only MCP tool names: Bash and file writes always ask unless
  denied. There is no OS sandbox; Bash and MCP processes remain outside path rules.
- History records remain version 4; no new record kinds or statuses.
- Extend the default SOUL with policy guidance, preserving existing user SOUL files.

## [0.8.0] - 2026-10-11

### Added

- Chapter 08 MCP tools: connect remote Streamable HTTP servers such as Context7 and
  local stdio servers at chat startup, discover their tools with page, count, and size
  limits, and expose them as `mcp_<server>_<tool>` Responses functions. One failed
  server is reported and skipped.
- `mino mcp add`, `list`, and `remove` manage a private `~/.mino/mcp.json` (mode 0600).
  Header and environment values, including `--bearer` tokens, are entered without
  echo and never shown, logged, or sent to the model. OAuth and legacy SSE are not
  supported. Project-local MCP configuration is never read.
- Each MCP tool call requires separate default-deny approval showing the server,
  destination, tool, and arguments; annotations never grant approval. Remote URLs
  require HTTPS (plain HTTP only on localhost), redirects are not followed, and
  messages are limited to 1 MiB. stdio servers run without a shell in their own
  process group with a minimal environment and are reaped on exit.
- Results keep text in order, mark omitted non-text content, fall back to structured
  content, and are truncated at 64 KiB. Server-reported tool errors fail; timeouts,
  cancellations, and lost connections after sending become `unknown` and are never
  retried.

### Changed

- History records remain version 4. Recorded MCP calls validate by argument shape even
  when their server is offline or removed, so sessions stay readable; `chapter-07`
  cannot load sessions that started MCP calls.
- A tool result that becomes `unknown` during a live session is acknowledged before the
  next message is sent, instead of requiring a restart; declining keeps the message unsent.
- Add the official Go MCP SDK (v1.8.0) and complete third-party notices, including the
  previously omitted YAML parser.
- Extend the default SOUL with MCP guidance, preserving existing user SOUL files.

## [0.7.0] - 2026-10-11

### Added

- Chapter 07 project file tools: `read_file`, `write_file`, and `edit_file`, confined
  to the startup project directory with `os.Root`. Symbolic-link targets, special
  files, multiple hard links, non-UTF-8 text, and files over 64 KiB are refused.
- `read_file` is a confined read and needs no approval. `write_file` requires an
  explicit `create` or `overwrite` mode; `edit_file` replaces one exact fragment that
  occurs once. Each write or edit shows a bounded preview and requires separate
  default-deny approval bound to the inspected file content; a file changed after
  approval is not written. Writes publish atomically through a synced temporary file.
- File tools are unavailable when the project directory contains `~/.mino`, such as
  when Mino starts in the home directory, so unapproved reads cannot expose settings
  or session history.

### Changed

- Tools may implement a live-only inspection step before approval; history replay
  still validates recorded arguments without reading project files. History records
  remain version 4, but `chapter-06` cannot load sessions that started file tools.
- Extend the default SOUL with file tool guidance, preserving existing user SOUL files.

## [0.6.0] - 2026-10-11

### Added

- Chapter 06 project Skills: discover only direct `.agents/skills/<name>/SKILL.md`
  files in the captured startup project. Parse YAML name/description with a pinned
  YAML parser; do not scan home-directory skills or preload bodies into instructions.
- Persist a sorted catalog once per session, including empty catalogs. Restore it
  on restart and `/resume`, retain it through `/clear` and compaction, and scan anew
  for `/new`. Legacy sessions initialize their catalog on first use in this version.
  The skill instruction prefix stays stable; service-side cache hits are not guaranteed.
- Add `skill_view` with a strict name argument. Read the current complete file only
  for registered skills, without confirmation; save paired calls/results through the
  existing Agent loop. Bash remains separately approved with the existing limits.
- Confine skill reads to the captured project, reject unsafe/special files, and bound
  file, metadata, and catalog sizes. Missing directories are empty; malformed skills
  fail discovery. Skill text is task guidance and grants no tool authorization.
- Bilingual Chapter 06 and tests for discovery, lazy reads, invalid arguments, path
  boundaries, stable prompts, session isolation/recovery, durability, and compaction.

### Changed

- Write history version 4 and read versions 1–4. The new `session_skills` record is
  synced before use. Earlier releases cannot read new records; preserve private
  backups before upgrading if rollback is needed. Released as `chapter-06`.
- Extend the default SOUL with Skills guidance, preserving existing user SOUL files.
- Roadmap: add planned Chapter 07 structured project file tools; MCP and guardrails
  move to Chapters 08–09. Unify Chinese chapter title wording.

## [0.5.0] - 2026-09-26

### Added

- Chapter 05 combines manual `/compact` and automatic context compaction. Keep
  the latest two replayable turns and all active tool steps; summarize earlier
  complete turns with the previous summary and preserve original JSONL records.
- Optional `context_window` in `~/.mino/config.json`: default 128,000 tokens,
  overridden by a positive integer. Startup reports the value/source without
  querying model metadata. Omitted or zero uses the default; invalid values fail.
- Conservative serialized-byte context estimates, output reservation (one eighth
  of the window, capped at 8,192 tokens), and an automatic trigger above 80% of
  the remaining input budget. Check every request, including after tool results;
  one automatic summary per turn counts toward the existing eight-request limit.
- Persist summaries and covered whole-turn boundaries before using them; restore
  them on restart/resume and remove them with confirmed `/clear`. Tool-free
  summary requests retain source/uncertainty instructions and cannot authorize
  tools. Failed or non-shrinking summaries retain the old context; oversized
  requests fail without silent truncation or command retries.
- Bilingual Chapter 05 lesson and mock-service tests for configuration, summary
  failure/cancellation, durability, repeated compaction, tool pairing, automatic
  triggers, request limits, restart recovery and session isolation.

### Changed

- Write history record version 3 while reading versions 1–3. Older releases cannot
  read new records; retain private backups before upgrading if rollback is needed.
- Set an explicit output-token limit and disable server truncation on Responses
  requests. The local estimate is deliberately conservative, not exact tokenization;
  summary generation costs a request and may lose details. History that cannot fit
  in one summary request requires a new session or a correctly sized window.
- Merge former Chapters 05/06, renumber Skills/MCP/guardrails to Chapters 06–08,
  and update the default SOUL and terminal help. Existing SOUL files stay untouched.

## [0.4.0] - 2026-09-26

### Added

- Optional Umami Cloud analytics for the published book, with shared English and
  Chinese pageview/visitor statistics, domain-restricted collection, and setup
  instructions. Local previews and pull requests stay untracked; chapter
  navigation and browser back/forward are counted without counting hash anchors.
- Chapter 04 multiple sessions (`chapter-04`, version `0.4.0`): `/new`, `/sessions`,
  `/resume <id>`, `/clear`, and `/help`. Each private JSONL file carries its
  session ID in its filename; startup restores the last selection.
- Interactive clearing requires the complete current session ID. It keeps other
  sessions, configuration, SOUL, migration archives, and recovery backups; it
  does not undo commands or promise secure disk erasure.
- Recoverable import of legacy `history.jsonl`, retaining its archive and holding
  its legacy lock during migration. A durable marker prevents duplicate imports;
  conflicting targets are never overwritten.
- Bilingual Chapter 04 lesson and tests for request isolation, tool recovery,
  migration retries, corrupt selection, process locking, safe paths, and failed
  publication. Unknown tool results still require acknowledgement, never re-execution.

### Changed

- Store sessions under `~/.mino/sessions/`, track selection in
  `active-session.json`, and serialize the store with `sessions.lock`. Selection
  damage enters a command-only recovery state; storage write failures stop chat.
- Publish selection and clearing atomically after syncing temporary files, then
  sync their directory before advancing memory. Tool protocol and record versions
  remain unchanged. Existing SOUL files remain untouched.

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
