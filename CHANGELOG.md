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

On 2026-10-26, again at the owner's request, Chapter 10 was rewritten in place and
`chapter-10` (`0.10.0`) was republished: the optional `~/.mino/policy.json` was
replaced by three permission modes selected with `/mode`. Earlier chapters and their
releases are unchanged. A `~/.mino/policy.json` left by the first 0.10.0 release is
now ignored.

## [0.15.0] - 2026-10-29

### Added

- Chapter 15 custom subagents: Markdown definitions in the project's
  `.agents/agents/<name>.md` with YAML `name` (equal to the file name), `description`
  (at most 1,024 characters), and an optional `tools` list of 1 to 64 tool names that
  narrows the subagent's tools and never adds one (`spawn_agent` and `memory_save`
  cannot be listed); the body is the role text. Files are at most 64 KiB, front matter
  16 KiB, and the catalog 32 KiB; unusable definitions are skipped and named in one
  notice. Other front-matter keys, such as a model or permission mode, are ignored.
- The definitions' metadata is saved once per session in a `session_agents` record,
  kept by restart, `/resume`, `/clear`, and compaction, and scanned again by `/new`;
  older sessions get one on first use. The catalog is added to the parent's
  instructions; the role text never is.
- `spawn_agent` tasks are now `{agent, task}` objects: `agent` names a definition, or is
  `null` for a general subagent. An unknown name is rejected before confirmation. A
  role's subagent receives the definition's current body with its task as labelled
  project guidance and only the listed tools that are available; missing ones are
  reported in its result, and a definition removed since the session started fails
  only its own task. Recorded 0.14.0 calls with string tasks still replay.
- Built-in skills: Mino now embeds a `create-subagent` skill that guides the model in
  writing a definition with the user. It appears in every skill catalog marked
  `builtin`; a project skill of the same name replaces it, with a notice.
- Eval replay case `subagent-role`.

### Changed

- Session history now writes version 5 records and reads versions 1 to 5. Releases
  up to 0.14.0 cannot read sessions written by this version, and an existing session
  gets a version 5 `session_agents` record as soon as this version opens or resumes
  it. A session whose skill catalog was saved by an earlier release has no
  `create-subagent` skill until `/new`.
- `skill_view` results start with a header: for a project skill, its directory, the
  Bash working directory, and the rule that a skill's relative paths are relative to
  its directory; for a built-in skill, a note that it has no directory. A file too
  large to return with this header fails instead.
- The default `SOUL.md` describes built-in skills, the skill directory rule, and
  project subagents; existing `~/.mino/SOUL.md` files are not changed.

## [0.14.0] - 2026-10-28

### Added

- Chapter 14 subagent delegation: the `spawn_agent` tool takes 1 to 10 tasks and runs
  each as a child Agent with its own context, its own session log under
  `~/.mino/sessions/<session_id>.agents/`, its own 8-request and 16-call limits, and
  its own compaction. Children share the session's permission mode, tools (except
  `spawn_agent` and `memory_save`), skill catalog, and the turn's memory index, and
  run at most 3 at a time, or `max_concurrent_subagents` (1 to 10) from
  `~/.mino/config.json`.
- The parent receives one tool result with each task's status and final text, in
  task order, within a shared 48 KiB budget; the subagents' own tool output never
  enters its context. `spawn_agent` asks in `manual` and `accept-edits` modes, each
  subagent's calls are checked and confirmed separately, one question at a time and
  labelled with the subagent, and a turn may start at most 16 subagents. Parent
  history stays version 4.
- Eval replay cases `subagent-batch` and `subagent-approval`; replay runs subagents
  one at a time, and token totals include subagent requests.
- The default `SOUL.md` explains `spawn_agent`; existing `~/.mino/SOUL.md` files are
  not changed.

### Changed

- The OS sandbox now blocks reading and writing `~/.mino`, which holds the API key,
  conversations, and memories, for Bash and sandboxed stdio MCP servers. The
  confirmation's sandbox line says so.
- `/clear` also deletes the cleared session's subagent logs.

## [0.13.0] - 2026-10-27

### Added

- Chapter 13 user memory: `internal/memory` stores one Markdown file per memory in
  `~/.mino/memory/` (front matter `name`, `type`, `summary`, then optional details).
  At the start of every turn Mino renders an index of every memory's one-line summary
  and sends it as the first input item, an assistant message labelled as data rather
  than instructions, merged with any context summary. The index is never written to
  history and is empty when nothing is remembered.
- `memory_view` reads one memory's full file without approval. `memory_save` creates
  or overwrites one memory after the user approves the shown content; it asks in
  `manual` and `accept-edits` modes and is unasked only in `bypass`. Saves are atomic
  and fail if the file changed after approval or the index is full (200 entries or
  16 KiB).
- Unusable memory files are skipped and reported once as a notice instead of stopping
  chat.
- Eval cases accept a `memory` list seeded into each run's home; new replay cases
  `memory-recall`, `memory-save`, and `memory-injection-denied`.
- The default `SOUL.md` explains memory; existing `~/.mino/SOUL.md` files are not
  changed, so the tool descriptions carry the same rules.

### Changed

- A bad project skill no longer stops chat. Discovery skips each unusable skill
  (malformed metadata, unsafe file, symbolic-link skill directory, or one past the
  32 KiB catalog limit) and names it with its reason in one notice when the catalog
  is scanned; only an unusable project, `.agents`, or `.agents/skills` directory is
  still an error. Earlier releases stopped chat, and an over-long description failed
  with a message that named no skill.
- Confirmation screens and the lines shown for calls a mode allows now print
  printable non-ASCII text, such as a Chinese memory summary, as written. Control,
  format (bidirectional and zero-width), and non-ASCII space characters are still
  escaped; earlier releases escaped every non-ASCII character as `\u` sequences.
- Chat input is edited by Mino instead of the terminal driver. Delete, Ctrl+U, and
  Ctrl+W now erase Chinese and other wide characters by their full two columns,
  including across wrapped rows, so the screen keeps matching the input; text typed
  while an answer streams is shown again after the next prompt, and arrow-key escape
  sequences are discarded instead of entering the line. Ctrl+C, Ctrl+D, and Enter
  behave as before, and the terminal settings are restored on exit.
- Skill descriptions are limited to 1,024 characters, as in the Agent Skills format,
  instead of 1,024 bytes, so non-ASCII descriptions are no longer held to about a
  third of the limit.

### Compatibility

- History stays v4, but `chapter-12` and earlier cannot open a session in which a
  `memory_view` or `memory_save` call started; they report a started call for an
  unavailable tool.
- `chapter-12` and earlier also cannot open a session whose skill catalog holds a
  description longer than 1,024 bytes.

## [0.12.0] - 2026-10-26

### Added

- Chapter 12 OS sandbox: `internal/sandbox` renders a macOS Seatbelt profile and runs
  a child process under `/usr/bin/sandbox-exec`. Reads are allowed except for the
  built-in protected paths; writes are confined to the project directory, temporary
  directories, and the per-user cache directories; `.git/` is readable but not
  writable; the network is denied.
- Bash runs every command in the sandbox when the system has one. The tool schema
  gains a `sandbox` field (`null` or `true` confines, `false` opts out); confirmations
  show a `Sandbox` line; the tool description and the default `SOUL.md` explain the
  limits. `tools.Call` and `policy.Call` gain `Sandboxed`, and `accept-edits` runs a
  sandboxed command without asking, as it runs confined file writes. An opt-out
  command asks in every mode except `bypass`.
- `mino mcp add --sandbox` confines a stdio MCP server with the network allowed and
  writes under its `cwd`; `mcp.json` stores `"sandbox": true`, and `mino mcp list`
  shows it. A sandboxed server that cannot be confined is reported unavailable.
- Eval runs confine Bash like chat. Three `sandbox-*` cases cover a protected read, a
  blocked network call, and a refused write under `.git/`.

### Changed

- The chat banner reads `Mino - Chapter 12: Sandbox`.
- Where `sandbox-exec` is missing, chat prints a startup notice and confirmations show
  `Sandbox: unavailable`; Bash then behaves as before. History stays at version 4:
  recorded arguments without a `sandbox` field mean the default.

## [0.11.0] - 2026-10-26

### Added

- Chapter 11 evals: `internal/eval` runs declared YAML cases through the real Agent
  loop and grades them with deterministic checks (`final_text_contains`,
  `final_text_not_contains`, `final_text_matches`, `tool_called`, `tool_not_called`,
  `tool_order`, `result_status`, `file_exists`, `file_contains`, `max_requests`,
  `max_tool_calls`). A case marked `live: false` asserts a scripted program path and
  is skipped by `--live`.
- `mino eval [PATH]` runs every `*.yaml` case in a directory (default `./evals`). Replay
  mode drives the loop with a scripted loopback Responses service: no key, no network,
  a fresh temporary workspace and home per run, the manual permission mode, and no MCP
  servers. `--live` uses `~/.mino/config.json`, runs each case several times (`--runs`,
  default 3), and reports pass rate, pass@k, pass^k, and token usage. `--filter`, `--out`
  (`report.json` plus one session log per run), and `--timeout` refine a run; any
  failing case exits with status 1.
- A repository `evals/` suite covering approved and denied Bash, a protected `.env`,
  file creation, and an unknown tool. `scripts/check.sh` runs it after the build.
- The Agent emits a `response_completed` event carrying the service's token usage after
  each validated response. The terminal gateway ignores it; `mino eval` sums it.

### Changed

- The chat banner reads `Mino - Chapter 11: Evals`, and `mino help` lists `eval`.

### Fixed

- Book build: override Vite with 6.4.5 and refresh the lockfile (`dompurify`, `source-map-js`)
  so `npm audit --audit-level=moderate` passes again and the tutorial site can be rebuilt
  and published. VitePress stays at 1.6.4; two low-severity KaTeX advisories remain.

## [0.10.0] - 2026-10-26

### Added

- Chapter 10 guardrails: every live tool call passes one policy decision (allow, ask,
  or deny) before inspection or confirmation. Denied calls return a `denied` result
  explaining the policy and never run; protected files are never opened, even to
  build a confirmation.
- Three permission modes, chosen in chat with `/mode manual|accept-edits|bypass`
  (`/mode` alone shows the current one). `manual`, the mode at every start, asks before
  every call except confined reads. `accept-edits` also runs `write_file` and
  `edit_file` without asking. `bypass` runs every call, including Bash and MCP, without
  asking. Loosening the mode requires confirmation in an interactive terminal;
  tightening it never asks. Each session has its own mode: a new session starts in
  `manual`, `/resume` returns to the mode that session had, and `/clear` keeps it.
  Modes last until Mino exits and are never saved or recorded in history.
- Built-in protected paths, refused in every mode: `.env`, `.env.*`, `*.pem`, `*.key`,
  `*.p12`, SSH keys, `.netrc`, `.npmrc`, `.pypirc`, and the `.git/`, `.ssh/`, `.aws/`,
  and `.gnupg/` directories, matched on every path component and ignoring case.
- Calls that a mode runs without asking still print the tool and its confirmation
  fields, such as the Bash command or the file path and preview.
- Claude Code project notes (`CLAUDE.md`) and a `book-writer` subagent that loads the
  same writing contract as the Codex `book_writer` role.

### Changed

- File tools refuse symbolic links in directory components, so a project link cannot
  rename a protected directory.
- There is no policy file; protected paths cannot be extended and no tool can be
  denied by name. There is no OS sandbox; Bash and MCP processes remain outside path
  rules.
- History records remain version 4; no new record kinds or statuses.
- Extend the default SOUL with permission-mode guidance, preserving existing user SOUL
  files.

## [0.9.0] - 2026-10-21

### Added

- Chapter 09 MCP tools: connect remote Streamable HTTP servers such as Context7 and
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
  when their server is offline or removed, so sessions stay readable; `chapter-08`
  cannot load sessions that started MCP calls.
- A tool result that becomes `unknown` during a live session is acknowledged before the
  next message is sent, instead of requiring a restart; declining keeps the message unsent.
- Add the official Go MCP SDK (v1.8.0) and complete third-party notices.
- Extend the default SOUL with MCP guidance, preserving existing user SOUL files.

## [0.8.0] - 2026-10-21

### Added

- Chapter 08 project file tools: `read_file`, `write_file`, and `edit_file`, confined
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
  remain version 4, but `chapter-07` cannot load sessions that started file tools.
- Extend the default SOUL with file tool guidance, preserving existing user SOUL files.

## [0.7.0] - 2026-10-21

### Added

- Chapter 07 project Skills: discover only direct `.agents/skills/<name>/SKILL.md`
  files in the captured startup project. Parse YAML name/description with a pinned
  YAML parser; do not scan home-directory skills or preload bodies into instructions.
- Persist a sorted catalog once per session, including empty catalogs. Restore it
  on restart and `/resume`, retain it through `/clear` and compaction, and scan anew
  for `/new`. Sessions created by earlier releases initialize their catalog on first
  use. The skill instruction prefix stays stable; service-side cache hits are not
  guaranteed.
- Add `skill_view` with a strict name argument. Read the current complete file only
  for registered skills, without confirmation; save paired calls/results through the
  existing Agent loop. Bash remains separately approved with the existing limits.
- Confine skill reads to the captured project, reject unsafe/special files, and bound
  file, metadata, and catalog sizes. Missing directories are empty; malformed skills
  fail discovery. Skill text is task guidance and grants no tool authorization.

### Changed

- Write history version 4 and read versions 1–4. The new `session_skills` record is
  synced before use. Earlier releases cannot read new records; preserve private
  backups before upgrading if rollback is needed.
- Extend the default SOUL with Skills guidance, preserving existing user SOUL files.

## [0.6.0] - 2026-10-21

### Added

- Chapter 06 combines manual `/compact` and automatic context compaction. Keep
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

### Changed

- Write history record version 3 while reading versions 1–3. Older releases cannot
  read new records; retain private backups before upgrading if rollback is needed.
- Set an explicit output-token limit and disable server truncation on Responses
  requests. The local estimate is deliberately conservative, not exact tokenization.
- Update the default SOUL and terminal help. Existing SOUL files stay untouched.

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
