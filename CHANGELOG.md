# Changelog

Versions follow the tutorial chapters: `0.<chapter>.<patch>`. Git tags use a
`v` prefix. Published releases are kept unchanged; fixes get a new patch release.

## [0.1.1] - 2026-09-23

- Fix private release installation and updates when GitHub's release-by-tag
  response omits uploaded assets; retrieve files through the dedicated assets API.
- Preserve the existing executable when an asset is missing or listing fails.

## [0.1.0] - 2026-09-23

- Chapter 01: independent terminal conversations using the OpenAI Responses API.
- English prompts, hidden API key entry, and reusable settings in `~/.mino/config.json`.
- macOS installation for Apple Silicon and Intel, without a local Go installation.
- `mino version` and `mino update`, with verified downloads and preserved settings.
- Optional `AGENTS.md` instructions from the current working directory.
- English and Simplified Chinese introductions, plus automated tests and releases.
