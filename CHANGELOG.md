# Changelog

Versions follow the tutorial chapters: `0.<chapter>.<patch>`. Git tags use a
`v` prefix. After this owner-requested Chapter 01 reset, published releases stay
unchanged and fixes receive new patch versions.

## [0.1.0] - 2026-09-23

Chapter 01 is reissued as `v0.1.0` at the owner's request, replacing the original
`v0.1.0` and `v0.1.1` releases. Install this baseline with `mino update v0.1.0`;
if an earlier updater fails, use the updated README installation command.
Both paths preserve existing configuration.

- Independent terminal conversations through the OpenAI Responses API, using
  the official OpenAI Go SDK v3.66.0. Mino retains control of interaction flow;
  tool execution and the Agent loop remain planned.
- A small `cmd/mino` entry point and application code and tests in `internal/mino`.
  Start source builds with `go run ./cmd/mino`.
- English prompts, hidden API key entry, and reusable settings in `~/.mino/config.json`.
  Project configuration and `OPENAI_*` environment variables are not used.
- Explicit request limits, no automatic retries or redirects, and redacted errors.
- macOS installation for Apple Silicon and Intel, without a local Go installation.
  The installer is now `internal/mino/install.sh`; use the updated README command.
- `mino version` and `mino update`, with verified downloads and preserved settings.
  Private release downloads use the dedicated GitHub assets API, retaining the
  original `v0.1.1` fix for incomplete asset listings.
- Optional `AGENTS.md` instructions from the current working directory.
- Updated English and Simplified Chinese tutorials, plus automated tests and releases.
