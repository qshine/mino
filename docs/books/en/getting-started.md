---
prev:
  text: About this book
  link: /
next:
  text: 01 A terminal conversation
  link: /chapters/01-terminal-chat
---

# Setup and installation

This page prepares the `chapter-05` release (application version `0.5.0`), published on 2026-10-21. It covers the book up to Chapter 05; the [chapter roadmap](./plan-todo-chapters.md) lists what each chapter adds. Use the tag named in each lesson when running its experiments.

Chapter 02 [saves one conversation locally](#local-history-in-chapter-02) and restores it at startup.

Chapter 03 adds [approved Bash execution](#try-chapter-03-from-source). New records use version 2, which `chapter-02` cannot read.

Chapter 04 [recovers interrupted tool turns](#try-chapter-04-from-source) and can ask you to acknowledge an unknown result before chat continues.

Chapter 05 adds [multiple sessions](#try-chapter-05-from-source). Before creating a second session, read [Chapter 05 session storage](#chapter-05-session-storage): earlier releases keep only one conversation.

## Prepare your Mac

Mino supports macOS 13 or later, with packages for Apple Silicon and Intel. The installer selects your architecture. You do not need Go to run a downloaded release.

The repository and release downloads are public. Installation uses macOS's `curl`; you do not need GitHub CLI or a GitHub login. A model API key is needed only when configuring the model service on first launch.

## Install with one command

Run this in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-05 && export PATH="$HOME/.mino/bin:$PATH"
```

This command fetches the current installer from `main` and selects `chapter-05`. Omit `-- chapter-05` to install the latest published release instead. The installer downloads the macOS package and checksum file from one resolved release tag.

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

If Mino was installed before the chapter-tag migration, rerun the installation command above **once**, even if the displayed numeric version is unchanged. The old embedded updater cannot resolve `chapter-*` tags. Reinstallation preserves your settings, custom `~/.mino/SOUL.md`, and any history.

After installing a chapter-tag release, you can check the numeric version and select this chapter again:

```bash
mino version
mino update chapter-05
```

`mino update` without an argument selects the latest release. The updated installer also accepts `0.5.0` or `v0.5.0` as aliases for `chapter-05`. Download or verification failures preserve the existing executable. Releases installed before 2026-10-21 use the earlier chapter numbering; see [republished releases](./releases.md#republished-releases). See the [release and patch rules](./releases.md#version-policy) for version numbers.

## Mino identity

After settings are complete, Mino loads `~/.mino/SOUL.md` once before starting chat. If the file is missing, Mino creates it from the bundled default identity. Existing content is preserved across restarts and updates; incomplete or cancelled configuration does not create the file.

The default describes Mino as a terminal assistant that helps with questions, explanations, writing, and code supplied by the user. It asks the model to use your language and describe its limits honestly. Edit the user file to change this guidance, then restart Mino; there is no live reload. The complete text is sent to your configured model service in `instructions`, so keep credentials out of it.

The Chapter 02 default describes supplied conversation history. The Chapter 03 default also describes paired tool results and Bash commands that require approval. The Chapter 04 default adds that an unknown result is not success and must never be retried automatically. Mino preserves existing identity files. When changing chapters, update only the capability statements that differ in the version you will run: independent questions end in Chapter 02, while tools arrive in the Chapter 03 release. Keep your other custom instructions and restart after editing.

Use a regular file containing non-empty UTF-8 text, at most 64 KiB. Mino rejects directories, symbolic links, and invalid text before sending any model request. It uses directory permissions `0700` and file permissions `0600`. Mino ignores both `AGENTS.md` and `SOUL.md` in the working directory: repository `AGENTS.md` is for development, while root `SOUL.md` is only the build-time default.

## Local history in Chapter 02

After loading settings and identity, Mino `chapter-02` opens its one saved conversation in `~/.mino/sessions/` and restores completed turns. On first launch it creates that directory and an empty `<session_id>.jsonl` file, where the session ID is 32 lowercase hexadecimal characters; later launches open the same file. Records use `turn_id` to pair each input with its answer and ending, and carry no `session_id` because the file name identifies the conversation. Mino also opens a stable `~/.mino/sessions.lock` before looking in the directory, so only one process can use the saved conversation at a time. Close the other Mino process if startup reports that history is in use; the lock file can remain after exit and should not be deleted to bypass a running process.

The session file and lock use `0600` permissions; `~/.mino/sessions/` and the user directory use `0700`. Mino rejects symbolic links and non-regular session or lock files, and refuses a `sessions` entry that is a file or symbolic link. Conversation text is stored locally in plain text; configuration credentials and raw API errors are not copied into the history. Text you submit is saved, including any sensitive data you put in it. Keep history and recovery copies out of Git, screenshots, and shared logs.

Only completed exchanges are sent with later questions. Changing the model or service in your configuration does not create a separate history: subsequent requests send those completed exchanges to the newly configured service. `store: false` disables service-side response-object storage; it does not disable this local file or the transmission of context.

If startup repairs an incomplete or malformed final line, it reports a `<session_id>-recovery-*.jsonl` backup beside the session file, saved with `0600` permissions before the repair. A pending turn is marked interrupted and is not retried. Corruption in the middle, unknown record fields or versions, and invalid record order stop startup while preserving the history contents; use a known valid backup rather than removing arbitrary records.

This release keeps exactly one conversation. If `~/.mino/sessions/` contains more than one file named by a session ID, startup stops with `~/.mino/sessions contains more than one conversation. This release keeps one conversation; use a release that supports multiple sessions.` and changes nothing. Recovery backups and other file names do not count. Move the extra files elsewhere, or run `chapter-05` or later, which manages several sessions.

History is limited to 16 MiB per record and 64 MiB per file. These are file limits, not the model's context budget. Mino reports a limit failure instead of silently dropping old records. If you need to start over manually, close Mino, keep a private backup, and move the session file out of `~/.mino/sessions/` before restarting; the next launch creates a new, empty conversation. In `chapter-02`, `/new`, `/clear`, and compaction are not implemented.

Releases published before 2026-10-21 kept this conversation in `~/.mino/history.jsonl`. Current releases do not read or migrate that file; keep it as a private backup.

## Try Chapter 03 from source

Chapter 03 is available as `chapter-03` (application version `0.3.0`). To follow its source, use the [checkout steps below](#learn-from-the-source), select `chapter-03`, and run from the repository root:

```bash
git checkout chapter-03
go run ./cmd/mino
```

The startup banner identifies `Chapter 03: Tools and the Agent Loop`; source builds report `dev` for their version. Use an interactive terminal to approve commands. Input redirected from a file or pipe cannot approve execution. Each command starts in the directory where you launched Mino, resolved to a real absolute path, and the approval screen shows that directory.

The configured model and service must support Responses function tools, strict parameter schemas, `parallel_tool_calls: false`, and replay of `function_call` and `function_call_output` items alongside messages and any returned encrypted reasoning. A service that only streams text is insufficient. Local mock tests verify the request structure; they do not certify your provider's compatibility.

When upgrading, review [Mino identity](#mino-identity): existing `~/.mino/SOUL.md` instructions are preserved. The new default describes tool calls, untrusted tool output, and execution limits.

Bash starts without profile files. Its supplied environment consists of `HOME`, `LANG=en_US.UTF-8`, and this fixed `PATH`:

```text
/opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin
```

Programs installed elsewhere may need an absolute executable path. Mino does not inherit your shell's aliases or other environment variables, and does not copy API configuration into the child environment. **Approved Bash still has your account's filesystem and network access**, including access to configuration files that account can read; this is not a sandbox. Review the complete command and displayed environment before approving it.

The same session file, `~/.mino/sessions/<session_id>.jsonl`, now also records model tool calls, starts, and results. Chapter 03 reads Chapter 02's `v: 1` records and writes new `v: 2` records; it does not rewrite older records. Back up the session file privately before running the Chapter 03 release. Once version 2 records have been appended, `chapter-02` cannot read that file: close Mino and preserve it separately before restoring your pre-upgrade backup for an older executable. Do not relabel version 2 records as version 1.

Like Chapter 02, Chapter 03 replays only completed turns. Startup never reruns a command: an unfinished tool turn is closed on disk with `not_executed` or `unknown` results and reported with a notice, but it is excluded from context, as are tool turns that failed or were cancelled. See [Chapter 03's restart behavior](./chapters/03-tools-and-bash.md#_4-2-what-a-restart-leaves-out).

Commands and captured results are saved in plain text and supplied as context on later requests. They may contain sensitive content, including data a command reads from a file. Switching the configured service also sends this replayable history to the new service. Keep history and recovery copies private.

## Try Chapter 04 from source

Chapter 04 is available as `chapter-04` (application version `0.4.0`). Select it with the [checkout steps below](#learn-from-the-source) and run from the repository root:

```bash
git checkout chapter-04
go run ./cmd/mino
```

The banner identifies `Chapter 04: Recovering Tool Turns`, and source builds report `dev`. Go, model-service, and Bash requirements are the same as in [Chapter 03's notes](#try-chapter-03-from-source). Running this checkout uses your real home-directory settings and session file.

Chapter 04 replays failed, cancelled, or interrupted tool turns with their paired results and a runtime notice. If the session contains an `unknown` result that has not been acknowledged, startup asks `Do you understand that these commands may already have run? [y/N]` before the banner. Answer in an interactive terminal; redirected or piped input cannot acknowledge, and any answer other than `y` or `yes` exits without saving an acknowledgement. The question returns on every start until a `recovery_ack` record is saved. Review the command's possible effects first; see [Chapter 04](./chapters/04-recovering-tool-turns.md).

The new default identity adds that an unknown result is not success and must never be retried automatically. Existing `~/.mino/SOUL.md` files are preserved; update that line yourself if you want it, then restart.

Chapter 04 keeps `v: 2` records and reads Chapter 03 session files as they are. Tool turns that Chapter 03 excluded from context are now replayed, and an `unknown` result left by Chapter 03 triggers the acknowledgement question. After a `recovery_ack` record has been written, `chapter-03` refuses to open the file because it does not know that record kind. Close Mino and keep a private backup of `~/.mino/sessions/` before switching between these releases.

## Try Chapter 05 from source

Chapter 05 is available as the [chapter-05 release](https://github.com/qshine/mino/releases/tag/chapter-05), with application version `0.5.0`. The installed package includes multiple sessions. To run the matching source, use this tag from the repository root:

```bash
git checkout chapter-05
go run ./cmd/mino
```

The banner identifies `Chapter 05: Multiple Sessions`, and source builds still report `dev`. Go and model-service requirements are the same as in [the source setup](#learn-from-the-source) and [Chapter 03's tool notes](#try-chapter-03-from-source). Running this checkout uses your real home-directory settings and the conversation saved by earlier chapters. Close other Mino processes and keep a private backup of `~/.mino/sessions/` before trying it.

Use `/new` to start a separate conversation, `/sessions` to list IDs, `/resume <id>` to select a complete 32-character lowercase hexadecimal ID, and `/help` to list commands. `/clear` requires typing the current session's exact ID in an interactive terminal; `/exit` exits. These commands do not call the model. If startup or `/resume` encounters an unknown tool result, Chapter 04's recovery acknowledgement is required before continuing. See [Chapter 05](./chapters/05-jsonl-sessions.md) for the A → B → A experiment and clear-confirmation behavior.

Changing sessions does not reload `~/.mino/SOUL.md` or change the Bash directory captured at startup. Existing identity files remain untouched. If yours claims that Mino only has one conversation, update that guidance yourself and restart. All sessions use the same loaded model settings: changing the configured service and restarting sends the selected session's replayable history to that service on your next question.

## Chapter 05 session storage

Chapter 05 uses this layout under `~/.mino/`:

```text
sessions/<session_id>.jsonl
active-session.json
sessions.lock
```

The user and sessions directories use `0700`; session logs, the selection file, and the lock use `0600`. The program rejects symbolic links and unexpected file types when opening them. Session content remains plain text and may contain private questions, commands, and tool output. Keep session files and recovery copies out of Git and shared logs. Each JSONL file retains the 16 MiB per-record and 64 MiB per-file limits; these are not model context budgets. Records still read `v: 1` and `v: 2`, with new records written as `v: 2`.

Mino holds `sessions.lock` for the whole run, so only one Mino process can use this session store, even if you intend to select different sessions. Chapters 02–04 use the same lock, so a running older Mino also blocks startup. Close the other process instead of deleting its lock. The lock file can remain after exit.

On a fresh setup, startup creates a session. With a saved selection, it restores that session. If `active-session.json` is absent and `sessions/` contains exactly one conversation, which is what Chapters 02–04 leave, Mino selects that file in place and saves the selection; the file is not copied or rewritten. A missing selection with several sessions, a damaged selection, or an unavailable selected log instead leaves Mino waiting for a selection. Use `/sessions`, then `/resume` with an existing complete ID, or choose `/new`. The list shows IDs and modification times, not a validation report; loading still checks the log. Recovery of an incomplete tail makes a private `<session_id>-recovery-*.jsonl` backup in `sessions/`. Corruption in the middle or invalid records reject that log rather than silently discarding content; failed recovery writes stop the program.

Chapters 02–04 support only one conversation and ignore `active-session.json`. Once `/new` has created a second session file, they stop at startup with `~/.mino/sessions contains more than one conversation`. To return to an older executable, close Mino, keep a private copy of `~/.mino/sessions/`, and leave only the conversation you want in that directory.

`/clear` keeps the current session ID and other sessions, but replaces the active log with empty history after confirmation. It does not remove recovery copies or manual backups, and it does not undo commands or securely erase disk data. Saving or syncing session changes can fail; Mino stops chat instead of promising success. Preserve the files and inspect the reported error before restarting, when the program checks the saved state again.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository, select `chapter-05`, and run from its directory:

```bash
git clone https://github.com/qshine/mino.git
cd mino
git checkout chapter-05
go run ./cmd/mino
```

Application code and tests live under `internal/`. `internal/gateway/` handles terminal interaction, and `internal/agent/` handles model requests and saved history. Chapter 03 adds tools under `internal/tools/`. Configuration and identity loading remain in `internal/`; no code-generation step is needed.

The module files remain at the repository root. `go.mod` pins the official `github.com/openai/openai-go/v3` SDK to v3.66.0, and `go.sum` records dependency checksums. Go downloads the dependencies when you first build. You do not need a separate SDK installation.

Downloaded and source builds share your home-directory settings and the conversations saved in `~/.mino/sessions/`. Automated tests use temporary home directories and mock model services, leaving your real `~/.mino` files untouched; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md) and follow the chapters in order. Run each chapter's experiments at its stated source version; the request and history formats differ.
