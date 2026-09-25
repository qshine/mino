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
