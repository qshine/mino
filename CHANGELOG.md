# Changelog

Versions follow the tutorial chapters: `0.<chapter>.<patch>`. Git tags use a
`v` prefix. After this owner-requested Chapter 01 reset, published releases stay
unchanged and fixes receive new patch versions.

## [Unreleased]

### Added

- Chapter 02 conversation history in `~/.mino/history.jsonl`, with one conversation
  per file, ordered turns paired by `turn_id`, and automatic restart recovery.
  Completed turns are supplied to subsequent Responses requests while answers
  continue to stream immediately. Preserve returned output items, including
  assistant phase and encrypted reasoning state, for stateless replay.
- Private history files, a single-writer lock, recovery backups for incomplete
  tails, validation of record order and turn pairing, and bounded file sizes.
  Failed and interrupted turns remain recorded without entering future context;
  storage failures stop chat and never silently retry a model request.
- Bilingual Chapter 02 lessons and deterministic tests for restart continuity,
  streaming, cancellation, disk failures, truncation recovery, and file locking.
  Chapter 02 targets `v0.2.0`; no release tag or package has been published.

### Changed

- Remove the per-record `session_id` from the unreleased Chapter 02 format;
  Chapter 04 will identify each session by its JSONL filename. Earlier development
  histories with this field are rejected unchanged; back up the file and remove
  only that top-level field from each record before reuse.
- Center the book writer's rules and bilingual Chapter 01 on user/model
  interactions, replacing file, function, and SDK walkthroughs with request,
  streaming response, completion, and context explanations.
- Download public macOS releases with `curl`, without GitHub CLI or a GitHub
  login, while preserving checksum/version verification and existing settings.
  The published `v0.1.0` embeds the earlier updater; rerun the README installer
  for public downloads until a new application release includes this change.
- Update the bilingual installation guides for the public repository and add
  direct online reading links to the bilingual GitHub Pages book website.

### Fixed

- Prevent Mermaid from automatically reprocessing diagrams already rendered by
  the book components, avoiding intermittent syntax errors after page loads.

## [0.1.0] - 2026-09-24

Chapter 01 is reissued as `v0.1.0` at the owner's request, replacing the original
`v0.1.0` and `v0.1.1` releases. Install this baseline with `mino update v0.1.0`;
if an earlier updater fails, use the updated README installation command.
Both paths preserve existing configuration and user identity edits. This reissue
includes application files directly in `internal/`, the root installer, and
`SOUL.md`. Run the update even if `mino version` already reports `0.1.0`:
the version number is unchanged by this owner-requested reissue.

At the owner's explicit request, `v0.1.0` is reissued again with terminal
streaming. Run `mino update v0.1.0` even if already on `0.1.0` to replace the
earlier non-streaming build. This release replacement is an owner-authorized
exception; subsequent fixes use new patch versions.

Streaming update:

- Display Responses API text and refusal fragments in the terminal as they arrive.
  Streaming is enabled by default and requires a `text/event-stream` endpoint.
- Keep partial answers visible on interruption or failure, report incomplete
  streams, and avoid printing the completed answer twice. Preserve cancellation,
  timeout, response-size limits, and terminal control-character filtering.
- Add a deterministic test proving terminal output appears before generation
  completes, plus stream failure, cancellation, refusal, and size-limit tests.

Existing Chapter 01 baseline:

- Independent terminal conversations through the OpenAI Responses API, using
  the official OpenAI Go SDK v3.66.0. Mino retains control of interaction flow;
  tool execution and the Agent loop remain planned.
- A small `cmd/mino` entry point and application code and tests in `internal`.
  Start source builds with `go run ./cmd/mino`.
- English prompts, hidden API key entry, and reusable settings in `~/.mino/config.json`.
  Project configuration and `OPENAI_*` environment variables are not used.
- Explicit request limits, no automatic retries or redirects, and redacted errors.
- macOS installation for Apple Silicon and Intel, without a local Go installation.
  The installer is now root `install.sh`; root `assets.go` embeds it so
  `mino update` works from any directory without a source checkout.
- `mino version` and `mino update`, with verified downloads and preserved settings.
  Private release downloads use the dedicated GitHub assets API, retaining the
  original `v0.1.1` fix for incomplete asset listings.
- An editable `~/.mino/SOUL.md` identity replaces working-directory `AGENTS.md`
  instructions. On first configured startup, seed the file from the bundled root
  `SOUL.md`; preserve user edits and load them on restart. Mino no longer reads
  project instruction files. Validate identity text before sending it in the
  Responses API `instructions` field.
- Updated English and Simplified Chinese tutorials, plus automated tests and releases.
