# Chapter 05 tasks

- [x] Configuration and startup: omitted/zero use 128,000; positive overrides; invalid values fail; no model lookup. Focused config/application tests passed.
- [x] Summary storage: version 3 append/replay, complete-turn coverage, preserved logs and atomic memory, failure/recovery tests passed.
- [x] Manual compaction: `/compact`, no tools, recent-turn retention, repeated summaries, session isolation and restart. Mock request tests passed.
- [x] Automatic compaction: count every request, reserve output, early trigger, bounded oversized/failure behavior and combined request limit. Mock loop/tool tests passed.
- [x] Review and verify: `GOCACHE=/tmp/mino-ch05-go-cache bash scripts/check.sh` passed formatting, syntax, vet, race tests and build; reviewed diff and failure boundaries.
- [x] Book handoff: `book_writer` updated both languages; reviewed facts, translations and version scope, updated README/changelog/SOUL/navigation, and passed the final `npm run book:build`. Browser checks covered both diagrams, light/dark modes, horizontal scrolling, a narrow viewport and language switching. No diagram/runtime errors; the preview retains an existing favicon 404 and the build retains its large-bundle warning.
- [x] CLI smoke: built executable with temporary HOME and a local mock service verified the 128,000 default, `/compact`, tool-free summary requests, retained recent context and summary restoration after restart. No live model was called.
- [x] Release preparation: packaged `chapter-05` for both macOS architectures, verified the native executable reports `0.5.0`, checked package contents and SHA-256 checksums, and passed `npm audit --audit-level=moderate` with no vulnerabilities. Updated release metadata for the user-authorized tag and release.
