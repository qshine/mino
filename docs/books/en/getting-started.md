---
prev:
  text: About this book
  link: /
next:
  text: 01 A terminal conversation
  link: /chapters/01-terminal-chat
---

# Setup and installation

This page prepares the `chapter-04` release (application version `0.4.0`), published on 2026-09-26. This chapter includes terminal conversation, saved history, approved Bash execution, and separate sessions. Use the tag named in each lesson when running its experiments.

## Prepare your Mac

Mino supports macOS 13 or later, with packages for Apple Silicon and Intel. The installer selects your architecture. You do not need Go to run a downloaded release.

The repository and release downloads are public. Installation uses macOS's `curl`; you do not need GitHub CLI or a GitHub login. A model API key is needed only when configuring the model service on first launch.

## Install with one command

Run this in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-04 && export PATH="$HOME/.mino/bin:$PATH"
```

This command fetches the current installer from `main` and selects `chapter-04`. Omit `-- chapter-04` to install the latest published release instead. The installer downloads the macOS package and checksum file from one resolved release tag.

The installer verifies the package's SHA-256 checksum and executable version before replacing `~/.mino/bin/mino`, and configures your terminal's command search path. Installation creates `~/.mino`; the configuration file is created after you complete setup on first launch.

## Start for the first time

From any directory, run:

```bash
mino
```

The program asks only for missing settings. All application prompts are in English:

```text
API URL [https://api.openai.com/v1]:
Model:
API Key (input hidden):
```

Press Enter to use the official OpenAI API URL. **There is no default model**: enter a model your service supports and your account can access. The API key is also required, and its characters are hidden while you type.

A custom service must support Responses API streaming: `stream: true` requests and `text/event-stream` responses. Mino rejects endpoints that return only complete JSON. Enter the API prefix, such as `https://gateway.example.com/v1`, without appending `/responses` or `/chat/completions`. Remote addresses require HTTPS.

Mino sends an ordered `input` list and requests `reasoning.encrypted_content`. A compatible service must accept these fields and accept its returned response items on subsequent requests. Chapter 03 and later also require [Responses function-tool support](#try-chapter-03-from-source). For a text-only response that omits output items on successful completion, Mino uses the completed streamed text as the saved answer.

Settings are saved in `~/.mino/config.json`. Later launches enter chat directly when settings are complete. Updates preserve them. The key is stored locally in plain text, with directory permissions `0700` and file permissions `0600`.

To change the service, model, or key, edit that file and restart. An empty field is prompted again; Ctrl+C during setup leaves existing settings unchanged. Mino does not read project-local configuration, `.env`, or `OPENAI_*` environment variables. Keep the configuration file out of the repository.

Enter a question after `You>`. Real questions contact your configured service and may incur charges under its terms. Entering only `/exit` checks startup without calling the model.

## Check the version and update

If Mino was installed before the chapter-tag migration, rerun the installation command above **once**, even if `mino version` already reports `0.4.0`. The old embedded updater cannot resolve `chapter-*` tags. Reinstallation preserves your settings, custom `~/.mino/SOUL.md`, and any history.

After installing a chapter-tag release, you can check the numeric version and select this chapter again:

```bash
mino version
mino update chapter-04
```

`mino update` without an argument selects the latest release. The updated installer also accepts `0.4.0` or `v0.4.0` as aliases for `chapter-04`. Download or verification failures preserve the existing executable. See the [release and patch rules](./releases.md#version-policy).

## Mino identity

After settings are complete, Mino loads `~/.mino/SOUL.md` once before starting chat. If the file is missing, Mino creates it from the bundled default identity. Existing content is preserved across restarts and updates; incomplete or cancelled configuration does not create the file.

The default describes Mino as a terminal assistant that helps with questions, explanations, writing, and code supplied by the user. It asks the model to use your language and describe its limits honestly. Edit the user file to change this guidance, then restart Mino; there is no live reload. The complete text is sent to your configured model service in `instructions`, so keep credentials out of it.

The Chapter 02 default describes supplied conversation history. The Chapter 03 default also describes paired tool results and Bash commands that require approval. Mino preserves existing identity files. When changing chapters, update only the capability statements that differ in the version you will run: independent questions end in Chapter 02, while tools arrive in the Chapter 03 release. Keep your other custom instructions and restart after editing.

Use a regular file containing non-empty UTF-8 text, at most 64 KiB. Mino rejects directories, symbolic links, and invalid text before sending any model request. It uses directory permissions `0700` and file permissions `0600`. Mino ignores both `AGENTS.md` and `SOUL.md` in the working directory: repository `AGENTS.md` is for development, while root `SOUL.md` is only the build-time default.

## Local history in Chapter 02

After loading settings and identity, Mino `chapter-02` opens `~/.mino/history.jsonl` and restores completed turns. This file represents one conversation; records use `turn_id` to pair each input with its answer and ending. The file starts empty on first launch. Mino also opens a stable `~/.mino/history.lock` so only one process can use this history at a time. Close the other Mino process if startup reports that history is in use; the lock file can remain after exit and should not be deleted to bypass a running process.

Both files use `0600` permissions under the `0700` user directory. Mino rejects symbolic links and non-regular history or lock files. Conversation text is stored locally in plain text; configuration credentials and raw API errors are not copied into the history. Text you submit is saved, including any sensitive data you put in it. Keep history and recovery copies out of Git, screenshots, and shared logs.

Only completed exchanges are sent with later questions. Changing the model or service in your configuration does not create a separate history: subsequent requests send those completed exchanges to the newly configured service. `store: false` disables service-side response-object storage; it does not disable this local file or the transmission of context.

If startup repairs an incomplete or malformed final line, it reports a `history-recovery-*.jsonl` backup in `~/.mino/`, saved with `0600` permissions before the repair. A pending turn is marked interrupted and is not retried. Corruption in the middle, unknown record fields or versions, and invalid record order stop startup while preserving the history contents; use a known valid backup rather than removing arbitrary records.

History is limited to 16 MiB per record and 64 MiB per file. These are file limits, not the model's context budget. Mino reports a limit failure instead of silently dropping old records. If you need to start over manually, close Mino, keep a private backup, and move `history.jsonl` aside before restarting. In `chapter-02`, `/new`, `/clear`, and compaction are not implemented.

Earlier Unreleased development builds wrote `session_id` into every record. The current format rejects this unknown field and leaves the file contents unchanged; Mino does not migrate it automatically. To keep that history, exit Mino and make a private backup, then remove only the top-level `session_id` field from each line's JSON object. Preserve `v`, `seq`, `turn_id`, all other fields and values, record order, and each line's terminating newline, including the final one. Do not remove matching text inside messages or nested data. Alternatively, back up and move the old file aside to start a new conversation.

## Try Chapter 03 from source

Chapter 03 is available as `chapter-03` (application version `0.3.0`). To follow its source, use the [checkout steps below](#learn-from-the-source), select `chapter-03`, and run from the repository root:

```bash
git checkout chapter-03
go run ./cmd/mino
```

The startup banner identifies Chapter 03; source builds report `dev` for their version. Use an interactive terminal to approve commands. Input redirected from a file or pipe cannot approve execution. Each command starts in the directory where you launched Mino, resolved to a real absolute path, and the approval screen shows that directory.

The configured model and service must support Responses function tools, strict parameter schemas, `parallel_tool_calls: false`, and replay of `function_call` and `function_call_output` items alongside messages and any returned encrypted reasoning. A service that only streams text is insufficient. Local mock tests verify the request structure; they do not certify your provider's compatibility.

When upgrading, review [Mino identity](#mino-identity): existing `~/.mino/SOUL.md` instructions are preserved. The new default describes tool calls, untrusted tool output, and execution limits.

Bash starts without profile files. Its supplied environment consists of `HOME`, `LANG=en_US.UTF-8`, and this fixed `PATH`:

```text
/opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin
```

Programs installed elsewhere may need an absolute executable path. Mino does not inherit your shell's aliases or other environment variables, and does not copy API configuration into the child environment. **Approved Bash still has your account's filesystem and network access**, including access to configuration files that account can read; this is not a sandbox. Review the complete command and displayed environment before approving it.

The same `history.jsonl` now also records model tool calls, starts, and results. Chapter 03 reads Chapter 02's `v: 1` records and writes new `v: 2` records; it does not rewrite older records. Back up history privately before running the Chapter 03 release. Once version 2 records have been appended, `chapter-02` cannot read that file: close Mino and preserve it separately before restoring your pre-upgrade backup for an older executable. Do not relabel version 2 records as version 1.

Unlike Chapter 02's completed-only replay, failed or interrupted tool turns retain paired calls and results, with a runtime notice, because an operation may already have happened. Startup never reruns a command. A start without a saved result becomes `unknown` and requires acknowledgment in an interactive terminal; that gate persists until the acknowledgment is saved. See [Chapter 03's recovery explanation](./chapters/03-tools-and-bash.md#_4-save-enough-to-recover-an-action-honestly).

Commands and captured results are saved in plain text and supplied as context on later requests. They may contain sensitive content, including data a command reads from a file. Switching the configured service also sends this replayable history to the new service. Keep history and recovery copies private.

## Try Chapter 04 from source

Chapter 04 is available as the [chapter-04 release](https://github.com/qshine/mino/releases/tag/chapter-04), with application version `0.4.0`. The installed package includes multiple sessions. To run the matching source, use this tag from the repository root:

```bash
git checkout chapter-04
go run ./cmd/mino
```

The banner identifies `Chapter 04: Multiple Sessions`, and source builds still report `dev`. Go and model-service requirements are the same as in [the source setup](#learn-from-the-source) and [Chapter 03's tool notes](#try-chapter-03-from-source). Running this checkout uses your real home-directory settings and performs the migration below when applicable. Close older Mino processes and keep private backups before trying it.

Use `/new` to start a separate conversation, `/sessions` to list IDs, `/resume <id>` to select a complete 32-character lowercase hexadecimal ID, and `/help` to list commands. `/clear` requires typing the current session's exact ID in an interactive terminal; `/exit` exits. These commands do not call the model. If startup encounters an unknown tool result, it requires the existing recovery acknowledgment before continuing. See [Chapter 04](./chapters/04-jsonl-sessions.md) for the A → B → A experiment and clear-confirmation behavior.

Changing sessions does not reload `~/.mino/SOUL.md` or change the Bash directory captured at startup. Existing identity files remain untouched. If yours claims that Mino only has one conversation, update that guidance yourself and restart. All sessions use the same loaded model settings: changing the configured service and restarting sends the selected session's replayable history to that service on your next question.

## Chapter 04 storage and migration

Chapter 04 uses this layout under `~/.mino/`:

```text
sessions/<session_id>.jsonl
active-session.json
sessions.lock
history-migration.json     (present during an unfinished import)
history.jsonl             (retained legacy archive after import)
history.lock              (legacy lock used during import)
```

The user and sessions directories use `0700`; session logs, selection, migration marker, and locks use `0600`. The program rejects symbolic links and unexpected file types when opening them. Session content remains plain text and may contain private questions, commands, and tool output. Keep session files, archives, and recovery copies out of Git and shared logs. Each JSONL file retains the 16 MiB per-record and 64 MiB per-file limits; these are not model context budgets. Records still read `v: 1` and `v: 2`, with new records written as `v: 2`.

Mino holds `sessions.lock` for the whole run, so only one Mino process can use this session store, even if you intend to select different sessions. Close the other process instead of deleting its lock. The lock file can remain after exit.

On a fresh setup, startup creates a session. With a saved selection, it restores that session. A missing or damaged selection, an unavailable selected log, or a conflicting migration target can instead leave Mino waiting for a selection. Use `/sessions`, then `/resume` with an existing complete ID, or choose `/new`. The list shows IDs and modification times, not a validation report; loading still checks the log. Recovery of an incomplete tail makes a private `<session_id>-recovery-*.jsonl` backup in `sessions/`. Corruption in the middle or invalid records reject that log rather than silently discarding content; failed recovery writes stop the program.

When no sessions or active selection exist and `history.jsonl` is present, Mino acquires the legacy `history.lock`, validates and recovers the old history, then copies it to one session. A valid completed log is copied unchanged; pending turns and incomplete tails can require the established recovery records and backups before copying. The old file remains as a migration archive. The program keeps the legacy lock through copying and saving the new selection, so a still-running older Mino blocks migration.

Before copying, Mino saves the destination ID in `history-migration.json`. If importing is interrupted, retrying uses the same destination. An existing target must match the validated legacy bytes; differing content is left untouched and requires explicit selection. A successfully saved active selection takes precedence at startup, avoiding another import. Matching migration progress is cleaned up on recovery. Do not run an older version expecting shared history: older versions continue to use `history.jsonl`, and their changes do not synchronize with the new sessions.

`/clear` keeps the current session ID and other sessions, but replaces the active log with empty history after confirmation. It does not remove the old `history.jsonl` archive, recovery copies, or manual backups, and it does not undo commands or securely erase disk data. Saving or syncing session changes can fail; Mino stops chat instead of promising success. Preserve the files and inspect the reported error before restarting, when the program checks the saved state again.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository, select `chapter-04`, and run from its directory:

```bash
git clone https://github.com/qshine/mino.git
cd mino
git checkout chapter-04
go run ./cmd/mino
```

Application code and tests live under `internal/`. The chapter snapshots share `internal/gateway/` for terminal interaction and `internal/agent/` for model requests and, from Chapter 02, history. Chapter 03 adds tools under `internal/tools/`. Configuration and identity loading remain in `internal/`; no code-generation step is needed.

The module files remain at the repository root. `go.mod` pins the official `github.com/openai/openai-go/v3` SDK to v3.66.0, and `go.sum` records dependency checksums. Go downloads the dependencies when you first build. You do not need a separate SDK installation.

Downloaded and source builds share your home-directory settings. Chapter 02 and Chapter 03 source runs use your home-directory history; Chapter 04 source runs use and migrate it as described above. Automated tests use temporary directories and mock model services, leaving your real history untouched; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md), [Chapter 02: JSONL history](./chapters/02-jsonl-history.md), [Chapter 03: tools and Bash](./chapters/03-tools-and-bash.md), or [Chapter 04: multiple sessions](./chapters/04-jsonl-sessions.md). Run each chapter's experiments at its stated source version; the request and history formats differ.
