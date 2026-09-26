# Chapter 05 implementation plan

The approved bilingual roadmap is the feature specification: one chapter for manual `/compact` and automatic context compaction, a default 128,000-token context window, and a `context_window` configuration override. There is no model-metadata discovery. Release: `chapter-05`, version 0.5.0; source builds remain `dev`.

## Implementation decisions

- Keep configuration in `internal/config.go`, composition in `internal/app.go`, and budgeting/compaction in `internal/agent/compact.go`. Gateway renders existing events; SessionManager routes `/compact` and serializes it with other commands and turns.
- Use serialized UTF-8 input, instructions, and tool-definition bytes as a conservative local token estimate, with framing allowance. This deliberately overcounts typical text and is not an exact tokenizer or a guarantee for arbitrary endpoints. Reserve one eighth of the window for output, capped at 8,192 tokens, and set `max_output_tokens` accordingly. Trigger automatic compaction at 80% of the remaining input budget; reject requests above that input budget.
- Retain the latest two replayable turns and the complete active turn. Summarize the earlier prefix together with any previous summary through a tool-free Responses request. Both summary and normal requests must fit their budgets; unsupported or oversized work reports a bounded error. One summary request per trigger, at most one automatic compaction per user turn, included in the existing eight-request turn limit.
- Append version 3 `context_compaction` records with summary text and the covered turn-end sequence. Continue reading versions 1 and 2; write version 3 records. Validate coverage against whole replayable turns, increasing coverage, settled tool results, and acknowledged recovery. Commit memory only after append and sync. Preserve original records and recover summaries on restart/resume; `/clear` also clears the summary.
- Supply summaries as labeled assistant context, never system instructions or tool authorization. The summarizer must preserve goals, constraints, decisions, pending work, provenance, and uncertain outcomes. Empty, refused, malformed, oversized, or non-shrinking summaries leave old context intact. Failed storage stops chat.

## Build order and verification

1. Configuration/default wiring and startup reporting; focused configuration tests.
2. Durable whole-turn summary transitions and recovery; history tests with write/sync failures and invalid records.
3. Manual command and summary request; mock HTTP tests for retained context, failures, no tool execution, isolation and restart.
4. Automatic budget checks before every request, including after tools; mock tests for thresholds, limits and oversized input/results.
5. Full `bash scripts/check.sh`, required `book_writer` documentation review, bilingual Chapter 05, README/SOUL/changelog and navigation updates; `npm run book:build` and browser inspection of changed diagrams.

Tests live beside implementation, use Go's standard testing package, temporary homes/session files and local mock model servers. Use ordinary Go formatting and explicit errors, for example `if err := session.append(record); err != nil { return &StorageError{err} }`. Preserve existing user edits and published tags/assets; do not use real credentials or paid models. The user authorized implementation, then explicitly requested pushing the chapter, tagging it and creating its release. Use the existing release workflow and leave repository publishing settings unchanged.
