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
