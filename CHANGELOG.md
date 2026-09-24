# Changelog

Versions follow the tutorial chapters: `0.<chapter>.<patch>`. Git tags use a
`v` prefix. After this owner-requested Chapter 01 reset, published releases stay
unchanged and fixes receive new patch versions.

## [Unreleased]

- Move `install.sh` to the repository root and update installation links.
  Root `assets.go` embeds it for `mino update`, which continues to work from
  any directory without a source checkout.
- Replace working-directory `AGENTS.md` instructions with an editable
  `~/.mino/SOUL.md` identity. On first configured startup, seed the file from the
  bundled root `SOUL.md`; preserve user edits and load them on restart.
  Mino no longer reads project instruction files. Validate identity text before
  sending it in the Responses API `instructions` field.

## [0.1.0] - 2026-09-24

Chapter 01 is reissued as `v0.1.0` at the owner's request, replacing the original
`v0.1.0` and `v0.1.1` releases. Install this baseline with `mino update v0.1.0`;
if an earlier updater fails, use the updated README installation command.
Both paths preserve existing configuration. This reissue includes the final
layout with application files directly in `internal/`.

- Independent terminal conversations through the OpenAI Responses API, using
  the official OpenAI Go SDK v3.66.0. Mino retains control of interaction flow;
  tool execution and the Agent loop remain planned.
- A small `cmd/mino` entry point and application code and tests in `internal`.
  Start source builds with `go run ./cmd/mino`.
- English prompts, hidden API key entry, and reusable settings in `~/.mino/config.json`.
  Project configuration and `OPENAI_*` environment variables are not used.
- Explicit request limits, no automatic retries or redirects, and redacted errors.
- macOS installation for Apple Silicon and Intel, without a local Go installation.
  The installer is now `internal/install.sh`; use the updated README command.
- `mino version` and `mino update`, with verified downloads and preserved settings.
  Private release downloads use the dedicated GitHub assets API, retaining the
  original `v0.1.1` fix for incomplete asset listings.
- Optional `AGENTS.md` instructions from the current working directory.
- Updated English and Simplified Chinese tutorials, plus automated tests and releases.
