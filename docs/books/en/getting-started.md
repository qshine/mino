---
prev:
  text: About this book
  link: /
next:
  text: 01 A terminal conversation
  link: /chapters/01-terminal-chat
---

# Setup and installation

This page prepares the `chapter-15` release (application version `0.15.0`), published on 2026-10-29. It covers the book up to Chapter 15; the [chapter roadmap](./plan-todo-chapters.md) lists what each chapter adds. Use the tag named in each lesson when running its experiments.

Chapter 02 [saves one conversation locally](#local-history-in-chapter-02) and restores it at startup.

Chapter 03 adds [approved Bash execution](#try-chapter-03-from-source). New records use version 2, which `chapter-02` cannot read.

Chapter 04 [recovers interrupted tool turns](#try-chapter-04-from-source) and can ask you to acknowledge an unknown result before chat continues.

Chapter 05 adds [multiple sessions](#try-chapter-05-from-source). Before creating a second session, read [Chapter 05 session storage](#chapter-05-session-storage): earlier releases keep only one conversation.

Chapter 06 uses a default context window of **128,000 tokens**, with a [configuration override](#configure-the-context-window). Before upgrading existing sessions, read the [history compatibility notes](#chapter-06-history-compatibility): new records use version 3.

Chapter 07 adds [project Skills](#try-chapter-07-from-source). Before upgrading existing sessions, also read the [Chapter 07 history compatibility notes](#chapter-07-history-compatibility): new records use version 4.

Chapter 08 adds [project file tools](#try-chapter-08-from-source). Before using it on existing sessions, read the [Chapter 08 history compatibility notes](#chapter-08-history-compatibility).

Chapter 09 adds [MCP tools](#try-chapter-09-from-source). Before using it on existing sessions, read the [Chapter 09 history compatibility notes](#chapter-09-history-compatibility).

Chapter 10 adds [guardrails](#try-chapter-10-from-source): one decision for every tool call, built-in protected paths, and [permission modes](#chapter-10-permission-modes) chosen with `/mode`.

Chapter 11 adds [evals](#try-chapter-11-from-source): `mino eval` runs declared cases through the Agent loop and grades the result. It keeps record version 4; see its [history compatibility notes](#chapter-11-history-compatibility).

Chapter 12 adds an [operating-system sandbox for Bash](#try-chapter-12-from-source) on macOS. It keeps record version 4; see its [history compatibility notes](#chapter-12-history-compatibility) before opening a Chapter 12 session with an older release.

Chapter 13 adds [user memory](#try-chapter-13-from-source) across sessions and projects. It keeps record version 4; see its [history compatibility notes](#chapter-13-history-compatibility) before opening a Chapter 13 session with an older release.

Chapter 14 adds [subagents](#try-chapter-14-from-source) that work in their own context. It keeps record version 4; see its [history compatibility notes](#chapter-14-history-compatibility) before opening a Chapter 14 session with an older release.

Chapter 15 adds [project subagent roles](#try-chapter-15-from-source). New records use version 5, which `chapter-14` cannot read; read its [history compatibility notes](#chapter-15-history-compatibility) before using it on existing sessions.

## Prepare your Mac

Mino supports macOS 13 or later, with packages for Apple Silicon and Intel. The installer selects your architecture. You do not need Go to run a downloaded release.

The repository and release downloads are public. Installation uses macOS's `curl`; you do not need GitHub CLI or a GitHub login. A model API key is needed only when configuring the model service on first launch.

## Install with one command

Run this in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-15 && export PATH="$HOME/.mino/bin:$PATH"
```

This command fetches the current installer from `main` and selects `chapter-15`. Omit `-- chapter-15` to install the latest published release instead. The installer downloads the macOS package and checksum file from one resolved release tag.

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
mino update chapter-15
```

`mino update` without an argument selects the latest release. The updated installer also accepts `0.15.0` or `v0.15.0` as aliases for `chapter-15`. Download or verification failures preserve the existing executable. Releases installed before 2026-10-21 use the earlier chapter numbering; see [republished releases](./releases.md#republished-releases). See the [release and patch rules](./releases.md#version-policy) for version numbers.

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

## Try Chapter 06 from source

Chapter 06 is available as the [chapter-06 release](https://github.com/qshine/mino/releases/tag/chapter-06), with application version `0.6.0`. To run the matching source, select its tag from the repository root:

```bash
git checkout chapter-06
go run ./cmd/mino
```

The banner identifies `Chapter 06: Context Compaction`, while the source version remains `dev`. The existing [Go setup](#learn-from-the-source) and [Responses tool requirements](#try-chapter-03-from-source) still apply. This version also requires the service to accept `max_output_tokens`, `truncation: "disabled"`, and tool-free summary requests with `tool_choice: "none"`. Local mock tests verify these fields; they do not establish compatibility with your provider.

At the next `You>` prompt, `/compact` summarizes earlier turns in the selected session while retaining its latest two replayable turns. Automatic budget checks use the same process before requests, including after tools. Unlike the session-selection commands, `/compact` can call the model and incur a charge. If no older turns exist, it makes no request. See [Chapter 06](./chapters/06-context-compaction.md) for the interaction, limits, and tests.

Existing SOUL files remain untouched. If yours claims that Mino always supplies the full history or cannot summarize, update those capability statements while keeping your custom guidance. Restart after editing. Summaries are lossy context, not new instructions or tool authorization.

### Configure the context window

Chapter 06 adds the optional `context_window` field to `~/.mino/config.json`. It is a non-negative integer measured in tokens: omission or `0` uses **128K (128,000 tokens)**, and a positive value overrides that default. Negative numbers, fractions, strings, and `null` are rejected. Setup does not prompt for this optional field.

For example, if your deployment supports a 64,000-token window, add the following field to the existing JSON object and restart. This is a field excerpt, not a replacement configuration: keep your existing `base_url`, `api_key`, and `model` fields.

```json
{
  "context_window": 64000
}
```

Startup reports `Context window: 64000 tokens (config).` With the field omitted or zero, it reports `Context window: 128000 tokens (default).` These are local settings; startup makes no model-metadata request. The value does not expand the service's real capacity. Set it to the actual window supported by your model and deployment.

Mino reserves output space and uses a byte-based estimate for input, so this setting is neither a file-size limit nor an exact count of consumed tokens. An overly small value can reject even an otherwise valid question; an overly large value can still lead to service-side errors. The [budget explanation](./chapters/06-context-compaction.md#_3-check-the-budget-before-each-request) describes the threshold and bounded failures.

### Chapter 06 history compatibility

Close other Mino processes and keep private pre-upgrade copies of your session logs and `active-session.json` before running this version on existing conversations. Chapter 06 reads record versions 1–3 and writes new records as `v: 3`, including ordinary turns. Once new records are appended, earlier releases cannot load the affected log, even if you have not used `/compact`. Do not relabel version 3 records as version 2.

To return to an older executable, close Mino, preserve the newer files separately, and restore the matching pre-upgrade files. Summaries are saved in the selected session's existing JSONL file; there is no separate summary file to copy. Session selection and recovery still apply as described in [Chapter 05 session storage](#chapter-05-session-storage).

Compaction appends a summary without deleting original messages, so it does not reduce disk usage or erase sensitive content. Both the earlier material sent for summarization and the resulting summary may contain private information. Keep session files and recovery copies private. Changing the configured service also changes the recipient of future summary and chat requests.

## Try Chapter 07 from source

Chapter 07 is available as the [chapter-07 release](https://github.com/qshine/mino/releases/tag/chapter-07), with application version `0.7.0`. To run the matching source, select its tag from the repository root:

```bash
git checkout chapter-07
go run ./cmd/mino
```

The banner identifies `Chapter 07: Project Skills`; the source version is `dev`. Existing Go, Responses tool, and compaction requirements still apply. This run uses your home-directory settings and sessions; read [Chapter 07 history compatibility](#chapter-07-history-compatibility) before using existing conversations. The no-cost mock checks are in [the lesson](./chapters/07-skills.md#_4-check-the-request-not-the-answer-s-claim).

Skills are discovered only under `.agents/skills/` in the directory where you start Mino. The program does not search parent directories or the user's `~/.agents/skills/`. For a fresh session, prepare the files before starting; if Mino restores an initialized session, use `/new` to discover them. An empty catalog is also saved, so adding a file does not update that session.

### Write a project skill

Create `.agents/skills/review/SKILL.md` under the project directory with this example content:

```text
---
name: review
description: Review code changes using tests and concrete evidence.
---
# Review a change

Read the supplied change and its tests. Identify a concrete failure scenario
for each finding. Separate confirmed issues from uncertainty. If you need
more evidence, request it; do not claim a test passed without its result.
```

At `You>`, ask “Use the review skill to review this change” and supply a small change. The model can request `skill_view` with `{"name":"review"}`. The read requires no confirmation; any subsequent Bash command still requires separate approval. Mino displays the full file as a tool result before the model continues.

Use direct child directories only; discovery is not recursive. The YAML header must start and end with `---`, with its closing delimiter within 16 KiB. `name` must match the directory, use lowercase ASCII letters or digits separated by single hyphens, and contain at most 64 characters. `description` must be a non-empty YAML string of at most 1,024 UTF-8 bytes; quoted and block strings are accepted. Other metadata is not included in the catalog.

Each `SKILL.md` must be a regular file of at most 64 KiB. A read requires non-empty UTF-8 text without NUL. The serialized catalog, including its project path, is limited to 32 KiB. An absent skills directory gives an empty catalog; direct directories without `SKILL.md` and symbolic-link child directories are ignored. Malformed candidates, symbolic-link files, special files, and symbolic links in the `.agents/skills` directory path are rejected. A discovery error prevents initialization; a failed `/new` leaves the selected session unchanged. These are the Chapter 07 rules: version 0.13.0 counts the description limit in characters and skips an unusable skill with a notice instead of failing, as described in [skill discovery changes](#skill-discovery-changes).

Names and descriptions stay fixed for the session, including after `/resume`, restart, and confirmed `/clear`; only `/new` discovers a fresh catalog. `skill_view` reads the current body from the saved project location. It accepts only catalog names, not arbitrary paths, and does not automatically read referenced files or execute scripts. Existing `~/.mino/SOUL.md` content remains untouched and is loaded once per process startup; update obsolete capability statements yourself and restart if needed.

### Chapter 07 history compatibility

Before running Chapter 07 on existing conversations, close Mino and keep private copies of session logs and `active-session.json`. It reads record versions 1–4 and writes all new records as `v: 4`. A session without a catalog gets a durable `session_skills` snapshot when first initialized by this version, even before its next question. An empty snapshot also uses version 4, so merely starting chat can make the selected log unreadable by earlier releases.

To roll back, stop Mino, preserve the newer files separately, and restore the matching pre-upgrade copies. Do not relabel version 4 records. `/clear` removes the selected conversation and summary while retaining its skill snapshot; it is not a format downgrade.

The catalog sends the captured project path, skill names, and descriptions to the configured model service. Requested full files enter both the session log and model context as tool results. Keep secrets out of skills and keep session logs private; this data can also enter later summary requests. See [Chapter 07](./chapters/07-skills.md) for fixed metadata, current file reads, and the limits of cache-related verification.

## Try Chapter 08 from source

Chapter 08 is available as the [chapter-08 release](https://github.com/qshine/mino/releases/tag/chapter-08), with application version `0.8.0`. An installed `mino` started in your project directory already includes the file tools. To run the matching source, select its tag from the repository root, build, then start it in the project you want it to work on:

```bash
git checkout chapter-08
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 08: Project File Tools`, the source version is `dev`, and the startup hint ends with `Bash commands and file writes require approval; project file reads do not.` Existing Go, Responses tool, compaction, and skill requirements still apply. The no-cost mock checks are in [the lesson](./chapters/08-file-tools.md#_4-check-the-approval-flow-with-mock-tests).

The file tools work only inside the directory where Mino starts, the same directory Bash uses; `/new` and `/resume` do not change it. `read_file` runs without approval, so anything it reads, including a `.env` file in the project, is sent to the configured model service and saved in the session log. Start Mino in a project directory, not your home directory: if the starting directory contains `~/.mino`, every file tool returns an error result instead of reading or writing.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations tells the model to prefer the file tools over Bash for project files and to read a file before editing it. If your SOUL describes Bash as the only tool, add similar guidance yourself and restart.

### Chapter 08 history compatibility

Chapter 08 keeps record version 4 and adds no record kinds, so it opens Chapter 07 sessions without migration. The reverse is not true. Once any file tool has run in a session, including an unapproved read, `chapter-07` cannot load that session: it reports a started call for an unavailable tool. Before trying Chapter 08 on existing conversations, close Mino and keep private copies of session logs and `active-session.json`. To roll back, stop Mino, preserve the newer files separately, and restore the matching copies.

## Try Chapter 09 from source

Chapter 09 is available as the [chapter-09 release](https://github.com/qshine/mino/releases/tag/chapter-09), with application version `0.9.0`. An installed `mino` already includes `mino mcp` and the MCP tools. To run the matching source, select its tag from the repository root, build, then start it in a project directory:

```bash
git checkout chapter-09
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 09: MCP Tools`, the source version is `dev`, and the startup hint ends with `Bash commands, file writes, and MCP tool calls require approval; project file reads do not.` Existing Go, Responses tool, compaction, skill, and file-tool requirements still apply. The no-cost mock checks are in [the lesson](./chapters/09-mcp.md#_5-check-the-routing-with-mock-tests).

### Add, list, and remove MCP servers

Servers are managed outside chat with `mino mcp` (or `bin/mino mcp` from the checkout). Add a remote Streamable HTTP server by name and URL:

```bash
mino mcp add context7 https://mcp.context7.com/mcp
```

```text
Added MCP server context7 (https://mcp.context7.com/mcp). It connects the next time chat starts; each tool call asks for approval.
```

If a service requires an API key, add `--bearer` to store an `Authorization: Bearer` header, or `--header NAME` (repeatable) for another header. A local stdio server takes its command after `--`, plus optional `--env NAME` (repeatable) and `--cwd DIR`:

```bash
mino mcp add context7 https://mcp.context7.com/mcp --bearer
mino mcp add local --env API_TOKEN -- /path/to/mcp-server
mino mcp list
mino mcp remove local
```

Replace `/path/to/mcp-server` with a server program you trust; Mino will start it every time chat starts. The command is resolved to an absolute path when you add it and run directly, without a shell, with its arguments. The working directory defaults to your home directory. Server names use lowercase letters, digits, and single hyphens, at most 32 characters; at most 16 servers can be configured. Run `mino mcp help` for the full usage.

Header and environment values are typed with echo off, so adding them requires an interactive terminal. They are never printed again: `mino mcp list` shows only their names. For example, after adding Context7 with a bearer token, it prints a line like this (illustrative output):

```text
context7  http  https://mcp.context7.com/mcp  headers=Authorization (values hidden)
```

Adding a server does not connect to it. The next chat startup connects to every configured server in parallel, without asking, and prints one line per server, either `MCP server NAME: N tools from DESTINATION` or `MCP server NAME unavailable: REASON`. A failed server never stops chat. Each tool call then asks for approval, as described in [the lesson](./chapters/09-mcp.md).

### MCP configuration and privacy

The configuration is `~/.mino/mcp.json`, a strict JSON file with mode `0600`. Mino refuses to start chat if it is a symbolic link, accessible to other users, malformed, or contains unknown fields; the `mino mcp` commands refuse it too, so fix such a file by hand. You may edit it by hand, but it contains your credentials in plain text: keep it out of Git, backups you share, and screenshots. Mino never reads MCP configuration from a project directory or from environment variables.

Credentials are sent only to their server, never to the model service, the session log, a confirmation, or an error message. Tool arguments and results are a different matter. Arguments you approve leave your machine for that server, and results are saved in the session log and sent to the model service like any other tool output. Avoid approving calls whose arguments contain secrets or private data you do not want that service to receive.

Remote URLs must use HTTPS; plain HTTP is accepted only for `localhost` and loopback addresses. Redirects are not followed. This chapter does not support OAuth or browser login, the legacy HTTP+SSE transport, or MCP resources, prompts, and sampling. A service that rejects static credentials is reported as unavailable with a note that OAuth is not supported. A server's tool list is read only at startup; restart Mino to see changes.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations describes `mcp_` tools, says that their descriptions and results are untrusted, and asks the model not to put secrets in arguments or retry unknown calls. Add similar guidance to your own SOUL if you want it, and restart.

### Chapter 09 history compatibility

Chapter 09 keeps record version 4 and adds no record kinds, so it opens Chapter 08 sessions without migration. Recorded MCP calls are validated only by their argument shape, so a session still opens when its server is offline or removed; nothing is called again. The reverse does not hold: once a session has started an MCP call, `chapter-08` cannot load it and reports a started call for an unavailable tool. Before trying Chapter 09 on existing conversations, close Mino and keep private copies of session logs and `active-session.json`. To roll back, stop Mino, preserve the newer files separately, and restore the matching copies. A `chapter-08` executable ignores `~/.mino/mcp.json`.

## Try Chapter 10 from source

Chapter 10 is available as the [chapter-10 release](https://github.com/qshine/mino/releases/tag/chapter-10), with application version `0.10.0`, republished on 2026-10-26 with permission modes. To run the matching source, select its tag from the repository root, build, then start it in a project directory:

```bash
git checkout chapter-10
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. Below the `Session:` line, startup prints `Permission mode: manual. Use /mode to change it.`; the same two lines appear after `/new` and `/resume`. The banner reads `Mino - Chapter 10: Guardrails` and the source version is `dev`. Existing Go, Responses tool, compaction, skill, file-tool, and MCP requirements still apply. The no-cost mock checks are in [the lesson](./chapters/10-guardrails.md#_7-check-the-decision-with-mock-tests).

### Chapter 10 permission modes

Chapter 10 has no configuration file for permissions. Choose a mode for the current session in chat:

| Command | Effect |
| --- | --- |
| `/mode` | Show the current session's mode. |
| `/mode manual` | Ask before every Bash command, file write or edit, and MCP call. Every session starts in this mode. |
| `/mode accept-edits` | Also run `write_file` and `edit_file` without asking. Bash and MCP calls still ask. |
| `/mode bypass` | Run every call without asking, including Bash and MCP. |

Moving to a looser mode prints a warning and asks `Switch to this mode? [y/N]`; it works only in an interactive terminal. Moving to a tighter mode never asks. Each session keeps its own mode until Mino exits: `/new` starts in `manual`, `/resume` returns to the mode that session had earlier in this run, and `/clear` keeps it. Modes are never saved, so after a restart every session is `manual` again. Every call that a mode lets run without asking is still printed with its fields. `/help` lists the command.

Protected paths apply in every mode, including `bypass`: `.env`, `.env.*`, `*.pem`, `*.key`, `*.p12`, `id_rsa*`, `id_ecdsa*`, `id_ed25519*`, `.netrc`, `.npmrc`, `.pypirc`, and the `.git/`, `.ssh/`, `.aws/`, and `.gnupg/` directories. File tools never read or write a matching path, and there is no prompt or setting to override this. Matching ignores case, as the default macOS file system does, so `.ENV` is protected too.

The file tools also change in one way: a path that passes through a symbolic link in any directory component is refused, even when the link stays inside the project. If your project reaches files through a linked directory, use the real path.

If you installed `0.10.0` before it was republished, its startup prints a `Policy:` line; run `mino update chapter-10` to replace it. The republished release ignores `~/.mino/policy.json`. Settings such as `"bash": "deny"` or `deny_tools` no longer apply: every tool is offered again, and in `manual` mode each Bash, write, and MCP call asks. Tools that `allow_tools` let run unasked now ask too. You may delete the file.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations describes the permission modes and asks the model to report blocks without working around them. Add similar guidance to your own SOUL if you want it, and restart. The decision is enforced either way.

The modes decide which calls need your answer, and protected paths limit the file tools. Neither is a sandbox: a Bash command or a configured MCP server that runs does so with your account's permissions, and in `bypass` mode a Bash command such as `cat .env` runs without asking. See [the lesson](./chapters/10-guardrails.md#_6-what-the-policy-cannot-see) for these limits.

### Chapter 10 history compatibility

Chapter 10 keeps record version 4 and adds no record kinds or result statuses. The permission mode is never written to a session. A call refused by the decision is saved like a call you declined: the model's request followed by a `denied` result. It opens Chapter 09 sessions without migration, and `chapter-09` can still open sessions written by Chapter 10. Rolling back removes the guardrails, though: `chapter-09` has no protected paths, so its `read_file` reads `.env` without asking. As before, keep private copies of session logs and `active-session.json` before switching versions.

## Try Chapter 11 from source

Chapter 11 is available as the [chapter-11 release](https://github.com/qshine/mino/releases/tag/chapter-11), with application version `0.11.0`, published on 2026-10-26; an installed `mino` already includes `mino eval`. To run the matching source, select its tag from the repository root, build, and run the repository's own cases from there:

```bash
git checkout chapter-11
go build -o bin/mino ./cmd/mino
bin/mino eval ./evals
```

The chat banner reads `Mino - Chapter 11: Evals`, `mino help` lists `eval`, and `mino eval help` prints the usage. The default replay mode drives the Agent loop with a scripted local model: it needs no API key or network, and it never reads `~/.mino`, so it cannot touch your settings or sessions. Each case runs in a fresh temporary workspace and home directory that are deleted afterwards. The existing Go requirement still applies; the lesson is [Chapter 11](./chapters/11-evals.md).

| Flag | Meaning |
| --- | --- |
| `PATH` | One case file, or a directory whose `*.yaml` and `*.yml` files are read in name order, not recursively. Default `./evals`. |
| `--live` | Use the model from `~/.mino/config.json`, with `~/.mino/SOUL.md` as the default instructions, creating the bundled SOUL first if it is missing, as chat does. The configuration must be complete; the command does not prompt. Cases marked `live: false` are skipped and named in one line before the table. Live runs cost real requests. |
| `--runs N` | Runs per case, 1 to 32. Default 1, or 3 with `--live`. A case's own `runs` field overrides it. |
| `--filter NAME` | Run only the named case. |
| `--out DIR` | Write `report.json` and each run's session log as `DIR/<case>/run-N.jsonl`. The directory may not be inside `~/.mino`. |
| `--timeout DURATION` | Limit for one run, 1s to 30m. Default 2m. |

The exit status is 1 when any case fails any run. Every run starts in the `manual` permission mode and connects no MCP servers, whatever your `~/.mino/mcp.json` says; `/mode` is a chat command and never applies to an eval run, and protected paths such as `.env` are refused exactly as in chat. Bash, when a case approves it, runs in the temporary workspace with your account's permissions; the harness is not a sandbox.

### Write an eval case

Create `evals/hello-file.yaml` in your project. The file name without its extension must equal `name`:

```yaml
name: hello-file
description: An approved write creates the file the user asked for.
prompt: Create hello.txt containing hi.
approve: [write_file]
model:
  - call: {tool: write_file, arguments: {path: hello.txt, content: "hi\n", mode: create}}
  - text: hello.txt now contains hi.
graders:
  - tool_order: [write_file]
  - result_status: {call: 1, status: completed}
  - file_contains: {path: hello.txt, text: hi}
  - max_requests: 2
```

It mirrors the repository's `write-then-read.yaml`. Run it alone with `bin/mino eval --filter hello-file ./evals`; observed on the Chapter 11 source under a temporary home directory, it prints one passing row with two requests and one tool call, and the temporary home still contains no `.mino` directory afterwards.

`name` and `prompt` are required; `prompt` is at most 16 KiB. `name` uses lowercase ASCII letters and digits with single hyphens, at most 64 bytes. `approve` is `all` (the default), `none`, or a list of tool names to approve. `instructions` replaces the default instructions for this case. `workspace.files` seeds relative paths before every run, at most 64 files of 64 KiB each. `model` is the replay script of at most 16 steps, each either `call: {tool, arguments}` or `text`, optionally with `expect_input_contains`, a substring that must appear in the request body the model receives at that step; it may be omitted only when you run with `--live`. `live: false` marks a replay-only case, one that asserts a scripted path a real model would not take; `--live` skips it, and it must have a `model` script. Omitting the field means the case also runs live. `graders` holds 1 to 32 checks, all of which must pass: `final_text_contains`, `final_text_not_contains`, `final_text_matches` (an RE2 pattern of at most 1 KiB), `tool_called`, `tool_not_called`, `tool_order`, `result_status` (`call` is 1-based over every call the model requested), `file_exists`, `file_contains`, `max_requests`, and `max_tool_calls`. A case meant to run live should grade an outcome a real model can reach, such as a written file or an absent secret, rather than one phrase. Unknown fields fail loading, a directory holds at most 256 cases, and a case file is at most 64 KiB.

The tools a case can call are the chat tools without MCP: `bash`, `read_file`, `write_file`, `edit_file`, and `skill_view`, which reads skills seeded under `.agents/skills` in the workspace. Case files are project data that you run deliberately, like project skills; they are never loaded from `~/.mino`.

### Chapter 11 history compatibility

Chapter 11 keeps record version 4 and adds no record kinds or result statuses. The only runtime addition is an in-memory `response_completed` event that carries token usage to `mino eval` and is never written to a session log. `chapter-10` can therefore open sessions written by Chapter 11, and Chapter 11 opens Chapter 10 sessions without migration; the only thing `chapter-10` lacks is the `mino eval` command. Eval runs write their logs only to temporary directories, or to the `--out` directory you name, never under `~/.mino/sessions`.

## Try Chapter 12 from source

Chapter 12 is available as the [chapter-12 release](https://github.com/qshine/mino/releases/tag/chapter-12), with application version `0.12.0`, published on 2026-10-26; an installed `mino` already runs Bash sandboxed. To run the matching source, select its tag from the repository root, build, and start the executable in a project directory:

```bash
git checkout chapter-12
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 12: Sandbox`; an installed release reports version `0.12.0`, while a source build reports `dev`. Existing Go, Responses tool, compaction, skill, file-tool, MCP, and guardrail requirements still apply; the lesson is [Chapter 12](./chapters/12-sandbox.md), and its no-cost checks are in [section 5](./chapters/12-sandbox.md#_5-check-it-with-evals-and-tests).

On macOS with `/usr/bin/sandbox-exec`, every Bash command runs confined by default: it can read files except the protected paths, write only inside the project directory, the system temporary directories, `~/Library/Caches`, and `~/.cache`, cannot write under `.git/`, and cannot open network connections. Each Bash confirmation, and each unasked Bash line in `accept-edits` or `bypass`, shows a `Sandbox` field with that summary. The model may set `sandbox: false` for a command that needs the network or a write elsewhere, such as `git commit`; Mino then shows `Sandbox: "off: the model asked to run this command unconfined."` and asks you in every mode except `bypass`. Because a confined command can only change the project, temporary, and cache directories, `/mode accept-edits` now runs sandboxed Bash without asking, as it runs `write_file` and `edit_file`; `manual` still asks for every command. The sandbox does not limit reads of unprotected files, does not confine `http` MCP servers, and does not confine a command you approve with `sandbox: false`.

If `sandbox-exec` is missing, startup prints `Sandbox unavailable on this system: Bash commands run with your account permissions.`, confirmations show `Sandbox: "unavailable on this system."`, and Bash behaves as in Chapter 11. There is no flag to turn the sandbox off, and no Linux sandbox in this chapter.

A local stdio MCP server can be confined with `mino mcp add NAME --sandbox --cwd DIR -- COMMAND`; `mino mcp list` shows `sandbox` on its line. It runs with writes limited to its `cwd`, temporary, and cache directories, protected paths blocked, and the network allowed. A sandboxed server that cannot be confined is reported unavailable at startup rather than started unconfined, and `--sandbox` is refused for `http` servers and on systems without a sandbox.

Eval cases run confined like chat. The three `sandbox-*` cases in `evals/` need a sandbox and fail without one; `bash scripts/check.sh` therefore passes only on macOS.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations describes the sandbox and tells the model to request `sandbox: false` only with a reason. Add similar guidance to your own SOUL if you want it, and restart; the sandbox is enforced either way.

### Chapter 12 history compatibility

Chapter 12 keeps record version 4 and adds no record kinds or result statuses; nothing about the sandbox is written to a session. A confined command is recorded as `{"command":"..."}`, exactly as Chapters 03–11 recorded every command, so Chapter 12 opens their sessions without migration and `chapter-11` can open a Chapter 12 session that contains only confined commands. The exception is an approved opt-out: its start record keeps `"sandbox":false`, which `chapter-11` rejects as an unknown Bash argument and refuses to load the session. As before, keep private copies of session logs and `active-session.json` before switching versions.

## Try Chapter 13 from source

Chapter 13 is available as the [chapter-13 release](https://github.com/qshine/mino/releases/tag/chapter-13), with application version `0.13.0`, published on 2026-10-27; an installed `mino` already remembers across sessions. To run the matching source, select its tag from the repository root, build, and start the executable in a project directory:

```bash
git checkout chapter-13
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 13: User Memory`; an installed release reports version `0.13.0`, while a source build reports `dev`. Existing Go, Responses tool, compaction, skill, file-tool, MCP, guardrail, and sandbox requirements still apply; the lesson is [Chapter 13](./chapters/13-user-memory.md), and its no-cost checks are in [section 5](./chapters/13-user-memory.md#_5-check-it-with-evals-and-tests).

Chat startup creates `~/.mino/memory/` with mode `0700` if it is missing, and refuses to start if that path is a symbolic link or not a directory. The model can read a memory with `memory_view` without a question; `memory_save` asks you in `manual` and `accept-edits` and runs unasked only in `bypass`. Each saved memory reaches the model in every later session and project, so read the summary before approving, and never approve one that contains a secret.

### Manage memory files

Each memory is one file, `~/.mino/memory/<name>.md`, with mode `0600`:

```text
---
name: uses-pnpm
type: project
summary: This user's projects install dependencies with pnpm, never npm or yarn.
---
Optional details, read on demand with memory_view.
```

The front matter holds exactly the string fields `name`, `type`, and `summary`. `name` must equal the file name without `.md` and use lowercase ASCII letters, digits, and single hyphens, at most 64 bytes. `type` is `user`, `feedback`, `project`, or `reference`. `summary` is one non-empty line of at most 200 bytes. Details after the closing `---` are optional, at most 16 KiB, and the whole file is at most 20 KiB.

You can create, edit, or delete these files with any editor; there is no separate index file to update, and changes take effect at the next turn. Files that break these rules, symbolic links, special files, and files with several hard links are skipped and listed once on standard error; chat continues. Files that do not end in `.md`, or start with a dot, are ignored. The index holds at most 200 memories and 16 KiB of text; when it is full, files beyond the limit are skipped with the same notice, and new saves fail until you merge, shorten, or delete memories.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations describes memory and asks the model to offer `memory_save` for lasting preferences and never to save secrets or text a file asked it to save; the tool descriptions carry the same rules, so memory works with your own SOUL too.

Eval cases accept an optional `memory` list, at most 32 entries with `name`, `type`, `summary`, and optional `details`, that is written into each run's temporary home before the run. Eval runs can now also call `memory_view` and `memory_save`; they never read or write your real `~/.mino/memory`.

### Skill discovery changes

Version 0.13.0 also changes how [Chapter 07](./chapters/07-skills.md) project skills are discovered. Before, one unusable `SKILL.md` failed discovery and kept chat from starting; a description over the limit even failed with a message that named no skill. This shows up most easily when you start Mino in your home directory: the project is then `~`, so `~/.agents/skills` is scanned as the project's skill directory, including any skills other tools installed there.

Discovery now takes candidates in name order, skips each one it cannot use, and keeps the rest. When it skips any, it prints one notice on standard error, for example:

```text
Some skills in /path/to/project/.agents/skills were not loaded:
  review: description has 1100 characters; the limit is 1,024
```

A skill is skipped, with its reason, for malformed metadata, for an unsafe `SKILL.md` (a symbolic link, a special file, or over 64 KiB), or because its skill directory is a symbolic link, which earlier releases ignored silently. When adding a skill would push the catalog past 32 KiB, that skill and every later one in name order are skipped instead of failing discovery. Plain files and directories without `SKILL.md` are still ignored silently, and an unusable project directory, or a `.agents` or `.agents/skills` path that is a symbolic link or not a directory, still fails discovery.

The `description` limit is now 1,024 characters, as in the Agent Skills format, instead of 1,024 UTF-8 bytes, so a Chinese description is no longer held to about 341 characters.

The notice appears only when a session gets its catalog: at startup for a session without one, and after each `/new`. The catalog stays frozen per session, so a restart or `/resume` neither rescans nor repeats the notice. After you fix a skill, use `/new` to load it.

### Chat input editing

Version 0.13.0 edits the chat line in Mino instead of leaving it to the terminal driver. Earlier releases let the macOS driver handle Delete, and it erases one screen column per character. A Chinese character takes two columns, so deleting one left half of it on screen, and what you saw no longer matched what would be sent. With the terminal's UTF-8 input mode (`iutf8`) off, Delete even removed a single byte and left a broken character.

In an interactive terminal, chat now turns off the driver's line editing and echo while it runs and restores your terminal settings when it exits, including after Ctrl+C. At `You>` and at every question asked during chat (tool approvals, `/mode`, `/clear`, and recovery acknowledgement):

- Delete or Backspace erases the last character by its display width, also across wrapped rows; Ctrl+U erases the line, and Ctrl+W the previous word.
- Enter submits, Ctrl+D on an empty line ends input, and Ctrl+C still quits.
- Arrow keys and other escape sequences are discarded instead of entering the line as text such as `^[[D`. There is still no cursor movement or input history.
- Text typed while an answer streams appears where you typed it and again after the next prompt, where you can edit it.

Widths are approximate for some emoji sequences, and resizing the window while you type a line takes effect only at the next prompt. Piped or redirected input and the configuration prompts, including the hidden API key, still use the driver's line reading.

### Chapter 13 history compatibility

Chapter 13 keeps record version 4 and adds no record kinds or result statuses. The memory index is rebuilt from the files at every turn and never written to a session log, so Chapter 13 opens sessions from earlier chapters without migration. A session in which a `memory_view` or `memory_save` call started records that call like any other tool call, and `chapter-12` refuses to load such a session with `History contains a started call for an unavailable tool`. Older releases ignore `~/.mino/memory/`. Because skill descriptions are now counted in characters, a session whose saved catalog holds a description longer than 1,024 bytes is also rejected by `chapter-12` and earlier, which still count bytes. As before, keep private copies of session logs and `active-session.json` before switching versions.

## Try Chapter 14 from source

Chapter 14 is available as the [chapter-14 release](https://github.com/qshine/mino/releases/tag/chapter-14), with application version `0.14.0`, published on 2026-10-28; an installed `mino` can already delegate work to subagents. To run the matching source, select its tag from the repository root, build, and start the executable in a project directory:

```bash
git checkout chapter-14
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 14: Subagents`; an installed release reports version `0.14.0`, while a source build reports `dev`. Existing Go, Responses tool, compaction, skill, file-tool, MCP, guardrail, sandbox, and memory requirements still apply; the lesson is [Chapter 14](./chapters/14-subagents.md), and its no-cost checks are in [section 5](./chapters/14-subagents.md#_5-check-it-with-evals-and-tests).

The model can now call `spawn_agent` with 1 to 10 tasks. The call asks you in `manual` and `accept-edits` and runs unasked only in `bypass`; approving it does not approve anything the subagents do. Each subagent's own tool calls follow the session's permission mode, and those that need approval ask one at a time, with a first `Subagent` field naming the task. A turn can start at most 16 subagents. Each subagent sends its own model requests, so one delegation can make many more requests than an ordinary turn.

Subagent logs are kept at `~/.mino/sessions/<session-id>.agents/<child-id>.jsonl`, with mode `0700` for the directory and `0600` for the files. `/sessions` does not list them and `/resume` cannot open them; `/clear` deletes them with the session's history.

### Configure subagent concurrency

At most 3 subagents run at once by default. The optional `max_concurrent_subagents` field in `~/.mino/config.json` changes that: an integer from 1 to 10, where omission or `0` keeps the default. `null`, negative numbers, and values above 10 stop startup with `max_concurrent_subagents in ~/.mino/config.json must be an integer from 1 to 10`; strings and fractions are rejected too. Setup does not prompt for this field.

For example, to run one subagent at a time, add the following field to the existing JSON object and restart. This is a field excerpt, not a replacement configuration:

```json
{
  "max_concurrent_subagents": 1
}
```

Mino does not retry failed model requests, so a rate-limit error fails that subagent's task. If your provider limits concurrent requests, lower this value. `mino eval --live` uses the same value; replay runs always run subagents one at a time.

### The sandbox blocks ~/.mino

On macOS with `sandbox-exec`, sandboxed Bash commands and stdio MCP servers added with `--sandbox` can no longer read or write `~/.mino`, which holds your API key, session logs, and memories. The `Sandbox` field now reads `on: writes only inside <project>, temporary and cache directories; protected paths and ~/.mino blocked; no network.`, and `mino mcp` help says the same for `--sandbox`. A sandboxed command such as `cat ~/.mino/config.json` now fails with `Operation not permitted`; a command approved with `sandbox: false` can still read the directory. If a sandboxed stdio server's `cwd` is inside `~/.mino`, move it elsewhere: the server can no longer use it.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations describes `spawn_agent` and the `~/.mino` restriction; the tool description carries the same guidance about self-contained tasks, so delegation works with your own SOUL too.

Eval runs can now call `spawn_agent`, and their subagents' logs stay in the run's temporary session directory. Token totals in `--live` reports include the subagents' requests.

### Chapter 14 history compatibility

Chapter 14 keeps record version 4 and adds no record kinds or result statuses. A whole batch is one `spawn_agent` start and one result in the parent's log, and each subagent's log is an ordinary session file in the `.agents` directory beside it. Chapter 14 opens sessions from earlier chapters without migration. A session in which a `spawn_agent` call started records that call like any other tool call, and `chapter-13` refuses to load such a session with `History contains a started call for an unavailable tool`. Older releases ignore the `.agents` directories, and their `/clear` leaves them in place. As before, keep private copies of session logs and `active-session.json` before switching versions.

## Try Chapter 15 from source

Chapter 15 is available as the [chapter-15 release](https://github.com/qshine/mino/releases/tag/chapter-15), with application version `0.15.0`, published on 2026-10-29; an installed `mino` can already delegate to project subagent roles. To run the matching source, select its tag from the repository root, build, and start the executable in a project directory:

```bash
git checkout chapter-15
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 15: Custom Subagents`; an installed release reports version `0.15.0`, while a source build reports `dev`. Existing Go, Responses tool, compaction, skill, file-tool, MCP, guardrail, sandbox, memory, and subagent requirements still apply; the lesson is [Chapter 15](./chapters/15-custom-subagents.md), and its no-cost checks are in [section 6](./chapters/15-custom-subagents.md#_6-check-it-with-evals-and-tests).

`spawn_agent` tasks can now name a project role. Approval works as in Chapter 14: the delegation asks in `manual` and `accept-edits`, and each subagent's own calls follow the session's mode. A role can only narrow a subagent's tools.

### Define a project subagent

Each role is one file, `.agents/agents/<name>.md`, directly in the directory where you start Mino. Mino never reads definitions from parent directories or your home directory. Plain files that do not end in `.md`, and directories, are ignored.

```text
---
name: explorer
description: Reads project files and reports what it finds, with file paths. Use for read-only lookups.
tools:
  - read_file
  - skill_view
---
Read only the files the task needs. Answer in a few sentences and give the
file path for every claim.
```

- `name` is a YAML string equal to the file name without `.md`, using lowercase ASCII letters, digits, and single hyphens, at most 64 bytes.
- `description` is a non-empty YAML string of at most 1,024 characters. It is all the parent model learns about when to use the role.
- `tools` is optional: a YAML list of 1 to 64 distinct tool names, such as `bash`, `read_file`, `write_file`, `edit_file`, `skill_view`, `memory_view`, or an MCP tool's `mcp_<server>_<tool>` name. Omit it to keep every tool a subagent normally has. `spawn_agent` and `memory_save` cannot be listed. A listed tool that is unavailable when the subagent starts is left out and named in that task's result.
- Other front-matter keys, such as a model or a permission mode written for another tool, are ignored.
- Everything after the closing `---` is the role text, sent to the role's subagent with each task. It can be empty.

The file is limited to 64 KiB, its front matter must close within 16 KiB, and the serialized catalog, including the project path, to 32 KiB. A definition that breaks these rules, a symbolic link, or one that no longer fits the catalog (taken in name order) is skipped, and one notice names each with its reason, for example:

```text
Some subagent definitions in /path/to/project/.agents/agents were not loaded:
  reviewer.md: tools cannot include spawn_agent; subagents never get it
```

An unusable project directory, or an `.agents` or `.agents/agents` path that is a symbolic link or not a directory, fails discovery, as it does for skills. A missing `.agents/agents` gives an empty catalog.

The names, descriptions, and tool lists are saved once per session and kept across restart, `/resume`, `/clear`, and compaction; only `/new` scans again, and the notice appears only then or when a session first gets its catalog. The role text is read from the file each time a subagent starts. The catalog sends the project path, names, descriptions, and tool lists to the configured model service; keep secrets out of definitions.

### Built-in skills and skill directories

Mino now embeds a built-in skill, `create-subagent`, that guides the model in writing a definition with you. It appears in every newly scanned skill catalog, marked `"builtin":true`; a project skill named `create-subagent` replaces it, with a line in the skill notice. Skill discovery is otherwise unchanged.

`skill_view` results now start with a short header: the skill's name, its absolute directory (or `none` for a built-in skill), the Bash working directory, and the rule that the skill's relative paths are relative to its own directory. Because the header shares the 64 KiB tool output limit, a `SKILL.md` close to 64 KiB now returns `failed` with `SKILL.md is too large to return with its header; keep it below 63 KiB.`

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations describes built-in skills, the skill directory rule, and the `agent` field in `spawn_agent` tasks; the tool descriptions carry the same guidance, so roles work with your own SOUL too.

Eval cases can seed definitions like any other file, by listing `.agents/agents/<name>.md` under `workspace.files`.

### Chapter 15 history compatibility

Chapter 15 writes every new record as `v: 5` and reads versions 1–5. The new `session_agents` record holds a session's subagent catalog and exists only at version 5. A session without one gets it as soon as Chapter 15 first uses it, even before its next question, so merely starting chat or resuming a session makes that log unreadable by `chapter-14` and earlier, which stop with an `Invalid history` error at the first version 5 line. `/clear` also rewrites the cleared log at version 5. Subagent logs are written at version 5 too.

Chapter 15 opens sessions from earlier chapters without migration. Their recorded `spawn_agent` tasks, saved as plain strings by `chapter-14`, still replay. A session whose skill catalog was saved by an earlier release keeps that catalog, so it has no built-in `create-subagent` skill until you start a new session with `/new`.

Before switching versions, keep private copies of session logs and `active-session.json`. To roll back, stop Mino, preserve the newer files separately, and restore the pre-upgrade copies; do not relabel version 5 records.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository, select `chapter-15`, and run from its directory:

```bash
git clone https://github.com/qshine/mino.git
cd mino
git checkout chapter-15
go run ./cmd/mino
```

Application code and tests live under `internal/`. `internal/gateway/` handles terminal interaction, and `internal/agent/` handles model requests and saved history. Chapter 03 adds tools under `internal/tools/`. Configuration and identity loading remain in `internal/`; no code-generation step is needed.

The module files remain at the repository root. `go.mod` pins the official `github.com/openai/openai-go/v3` SDK to v3.66.0 and the official `github.com/modelcontextprotocol/go-sdk` to v1.8.0, and `go.sum` records dependency checksums. Go downloads the dependencies when you first build. You do not need a separate SDK installation.

Downloaded and source builds share your home-directory settings and the conversations saved in `~/.mino/sessions/`. Automated tests use temporary home directories and mock model services, leaving your real `~/.mino` files untouched; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md) and follow the chapters in order. Run each chapter's experiments at its stated source version; the request and history formats differ.
