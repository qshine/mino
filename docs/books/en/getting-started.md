---
prev:
  text: About this book
  link: /
next:
  text: 01 A terminal conversation
  link: /chapters/01-terminal-chat
---

# Setup and installation

This page prepares the `chapter-10` release (application version `0.10.0`), published on 2026-10-19. It includes terminal conversation, saved history, approved Bash execution, separate sessions, manual and automatic context compaction, project Skills, project file tools, MCP tools, guardrails, and evals. Use the tag named in each lesson when running its experiments.

Chapter 05 uses a default context window of **128,000 tokens**, with a [configuration override](#configure-the-context-window). Before upgrading existing sessions, read the [history compatibility notes](#chapter-05-history-compatibility): new records use version 3.

Chapter 06 adds [project Skills](#try-chapter-06-from-source). Before upgrading existing sessions, also read the [Chapter 06 history compatibility notes](#chapter-06-history-compatibility): new records use version 4.

Chapter 07 adds [project file tools](#try-chapter-07-from-source). Before using it on existing sessions, read the [Chapter 07 history compatibility notes](#chapter-07-history-compatibility).

Chapter 08 adds [MCP tools](#try-chapter-08-from-source). Before using it on existing sessions, read the [Chapter 08 history compatibility notes](#chapter-08-history-compatibility).

Chapter 09 adds [guardrails](#try-chapter-09-from-source): one tool policy and an optional `~/.mino/policy.json`.

Chapter 10 adds [evals](#try-chapter-10-from-source): `mino eval` runs declared cases through the Agent loop and grades the result. It keeps record version 4; see its [history compatibility notes](#chapter-10-history-compatibility).

## Prepare your Mac

Mino supports macOS 13 or later, with packages for Apple Silicon and Intel. The installer selects your architecture. You do not need Go to run a downloaded release.

The repository and release downloads are public. Installation uses macOS's `curl`; you do not need GitHub CLI or a GitHub login. A model API key is needed only when configuring the model service on first launch.

## Install with one command

Run this in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-10 && export PATH="$HOME/.mino/bin:$PATH"
```

This command fetches the current installer from `main` and selects `chapter-10`. Omit `-- chapter-10` to install the latest published release instead. The installer downloads the macOS package and checksum file from one resolved release tag.

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
mino update chapter-10
```

`mino update` without an argument selects the latest release. The updated installer also accepts `0.10.0` or `v0.10.0` as aliases for `chapter-10`. Download or verification failures preserve the existing executable. See the [release and patch rules](./releases.md#version-policy).

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

## Try Chapter 05 from source

Chapter 05 is available as the [chapter-05 release](https://github.com/qshine/mino/releases/tag/chapter-05), with application version `0.5.0`. To run the matching source, select its tag from the repository root:

```bash
git checkout chapter-05
go run ./cmd/mino
```

The banner identifies `Chapter 05: Context Compaction`, while the source version remains `dev`. The existing [Go setup](#learn-from-the-source) and [Responses tool requirements](#try-chapter-03-from-source) still apply. This version also requires the service to accept `max_output_tokens`, `truncation: "disabled"`, and tool-free summary requests with `tool_choice: "none"`. Local mock tests verify these fields; they do not establish compatibility with your provider.

At the next `You>` prompt, `/compact` summarizes earlier turns in the selected session while retaining its latest two replayable turns. Automatic budget checks use the same process before requests, including after tools. Unlike the session-selection commands, `/compact` can call the model and incur a charge. If no older turns exist, it makes no request. See [Chapter 05](./chapters/05-context-compaction.md) for the interaction, limits, and tests.

Existing SOUL files remain untouched. If yours claims that Mino always supplies the full history or cannot summarize, update those capability statements while keeping your custom guidance. Restart after editing. Summaries are lossy context, not new instructions or tool authorization.

### Configure the context window

Chapter 05 adds the optional `context_window` field to `~/.mino/config.json`. It is a non-negative integer measured in tokens: omission or `0` uses **128K (128,000 tokens)**, and a positive value overrides that default. Negative numbers, fractions, strings, and `null` are rejected. Setup does not prompt for this optional field.

For example, if your deployment supports a 64,000-token window, add the following field to the existing JSON object and restart. This is a field excerpt, not a replacement configuration: keep your existing `base_url`, `api_key`, and `model` fields.

```json
{
  "context_window": 64000
}
```

Startup reports `Context window: 64000 tokens (config).` With the field omitted or zero, it reports `Context window: 128000 tokens (default).` These are local settings; startup makes no model-metadata request. The value does not expand the service's real capacity. Set it to the actual window supported by your model and deployment.

Mino reserves output space and uses a byte-based estimate for input, so this setting is neither a file-size limit nor an exact count of consumed tokens. An overly small value can reject even an otherwise valid question; an overly large value can still lead to service-side errors. The [budget explanation](./chapters/05-context-compaction.md#_3-check-the-budget-before-each-request) describes the threshold and bounded failures.

### Chapter 05 history compatibility

Close other Mino processes and keep private pre-upgrade copies of your session logs and `active-session.json` before running this version on existing conversations. Chapter 05 reads record versions 1–3 and writes new records as `v: 3`, including ordinary turns. Once new records are appended, earlier releases cannot load the affected log, even if you have not used `/compact`. Do not relabel version 3 records as version 2.

To return to an older executable, close Mino, preserve the newer files separately, and restore the matching pre-upgrade files. Summaries are saved in the selected session's existing JSONL file; there is no separate summary file to copy. Legacy migration and session recovery still apply as described [above](#chapter-04-storage-and-migration).

Compaction appends a summary without deleting original messages, so it does not reduce disk usage or erase sensitive content. Both the earlier material sent for summarization and the resulting summary may contain private information. Keep session files and recovery copies private. Changing the configured service also changes the recipient of future summary and chat requests.

## Try Chapter 06 from source

Chapter 06 is available as the [chapter-06 release](https://github.com/qshine/mino/releases/tag/chapter-06), with application version `0.6.0`. To run the matching source, select its tag from the repository root:

```bash
git checkout chapter-06
go run ./cmd/mino
```

The banner identifies `Chapter 06: Project Skills`; the source version is `dev`. Existing Go, Responses tool, and compaction requirements still apply. This run uses your home-directory settings and sessions; read [Chapter 06 history compatibility](#chapter-06-history-compatibility) before using existing conversations. The no-cost mock checks are in [the lesson](./chapters/06-skills.md#_4-check-the-request-not-the-answer-s-claim).

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

Each `SKILL.md` must be a regular file of at most 64 KiB. A read requires non-empty UTF-8 text without NUL. The serialized catalog, including its project path, is limited to 32 KiB. An absent skills directory gives an empty catalog; direct directories without `SKILL.md` and symbolic-link child directories are ignored. Malformed candidates, symbolic-link files, special files, and symbolic links in the `.agents/skills` directory path are rejected. A discovery error prevents initialization; a failed `/new` leaves the selected session unchanged.

Names and descriptions stay fixed for the session, including after `/resume`, restart, and confirmed `/clear`; only `/new` discovers a fresh catalog. `skill_view` reads the current body from the saved project location. It accepts only catalog names, not arbitrary paths, and does not automatically read referenced files or execute scripts. Existing `~/.mino/SOUL.md` content remains untouched and is loaded once per process startup; update obsolete capability statements yourself and restart if needed.

### Chapter 06 history compatibility

Before running Chapter 06 on existing conversations, close Mino and keep private copies of session logs and `active-session.json`. It reads record versions 1–4 and writes all new records as `v: 4`. A session without a catalog gets a durable `session_skills` snapshot when first initialized by this version, even before its next question. An empty snapshot also uses version 4, so merely starting chat can make the selected log unreadable by earlier releases.

To roll back, stop Mino, preserve the newer files separately, and restore the matching pre-upgrade copies. Do not relabel version 4 records. `/clear` removes the selected conversation and summary while retaining its skill snapshot; it is not a format downgrade.

The catalog sends the captured project path, skill names, and descriptions to the configured model service. Requested full files enter both the session log and model context as tool results. Keep secrets out of skills and keep session logs private; this data can also enter later summary requests. See [Chapter 06](./chapters/06-skills.md) for fixed metadata, current file reads, and the limits of cache-related verification.

## Try Chapter 07 from source

Chapter 07 is available as the [chapter-07 release](https://github.com/qshine/mino/releases/tag/chapter-07), with application version `0.7.0`. An installed `mino` started in your project directory already includes the file tools. To run the matching source, select its tag from the repository root, build, then start it in the project you want it to work on:

```bash
git checkout chapter-07
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 07: Project File Tools`, the source version is `dev`, and the startup hint ends with `Bash commands and file writes require approval; project file reads do not.` Existing Go, Responses tool, compaction, and skill requirements still apply. The no-cost mock checks are in [the lesson](./chapters/07-file-tools.md#_4-check-the-approval-flow-with-mock-tests).

The file tools work only inside the directory where Mino starts, the same directory Bash uses; `/new` and `/resume` do not change it. `read_file` runs without approval, so anything it reads, including a `.env` file in the project, is sent to the configured model service and saved in the session log. Start Mino in a project directory, not your home directory: if the starting directory contains `~/.mino`, every file tool returns an error result instead of reading or writing.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations tells the model to prefer the file tools over Bash for project files and to read a file before editing it. If your SOUL describes Bash as the only tool, add similar guidance yourself and restart.

### Chapter 07 history compatibility

Chapter 07 keeps record version 4 and adds no record kinds, so it opens Chapter 06 sessions without migration. The reverse is not true. Once any file tool has run in a session, including an unapproved read, `chapter-06` cannot load that session: it reports a started call for an unavailable tool. Before trying Chapter 07 on existing conversations, close Mino and keep private copies of session logs and `active-session.json`. To roll back, stop Mino, preserve the newer files separately, and restore the matching copies.

## Try Chapter 08 from source

Chapter 08 is available as the [chapter-08 release](https://github.com/qshine/mino/releases/tag/chapter-08), with application version `0.8.0`. An installed `mino` already includes `mino mcp` and the MCP tools. To run the matching source, select its tag from the repository root, build, then start it in a project directory:

```bash
git checkout chapter-08
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. The banner reads `Mino - Chapter 08: MCP Tools`, the source version is `dev`, and the startup hint ends with `Bash commands, file writes, and MCP tool calls require approval; project file reads do not.` Existing Go, Responses tool, compaction, skill, and file-tool requirements still apply. The no-cost mock checks are in [the lesson](./chapters/08-mcp.md#_5-check-the-routing-with-mock-tests).

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

Adding a server does not connect to it. The next chat startup connects to every configured server in parallel, without asking, and prints one line per server, either `MCP server NAME: N tools from DESTINATION` or `MCP server NAME unavailable: REASON`. A failed server never stops chat. Each tool call then asks for approval, as described in [the lesson](./chapters/08-mcp.md).

### MCP configuration and privacy

The configuration is `~/.mino/mcp.json`, a strict JSON file with mode `0600`. Mino refuses to start chat if it is a symbolic link, accessible to other users, malformed, or contains unknown fields; the `mino mcp` commands refuse it too, so fix such a file by hand. You may edit it by hand, but it contains your credentials in plain text: keep it out of Git, backups you share, and screenshots. Mino never reads MCP configuration from a project directory or from environment variables.

Credentials are sent only to their server, never to the model service, the session log, or a confirmation. Tool arguments and results are a different matter. Arguments you approve leave your machine for that server, and results are saved in the session log and sent to the model service like any other tool output. Avoid approving calls whose arguments contain secrets or private data you do not want that service to receive.

Remote URLs must use HTTPS; plain HTTP is accepted only for `localhost` and loopback addresses. Redirects are not followed. This chapter does not support OAuth or browser login, the legacy HTTP+SSE transport, or MCP resources, prompts, and sampling. A service that rejects static credentials is reported as unavailable with a note that OAuth is not supported. A server's tool list is read only at startup; restart Mino to see changes.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations describes `mcp_` tools, says that their descriptions and results are untrusted, and asks the model not to put secrets in arguments or retry unknown calls. Add similar guidance to your own SOUL if you want it, and restart.

### Chapter 08 history compatibility

Chapter 08 keeps record version 4 and adds no record kinds, so it opens Chapter 07 sessions without migration. Recorded MCP calls are validated only by their argument shape, so a session still opens when its server is offline or removed; nothing is called again. The reverse does not hold: once a session has started an MCP call, `chapter-07` cannot load it and reports a started call for an unavailable tool. Before trying Chapter 08 on existing conversations, close Mino and keep private copies of session logs and `active-session.json`. To roll back, stop Mino, preserve the newer files separately, and restore the matching copies. A `chapter-07` executable ignores `~/.mino/mcp.json`.

## Try Chapter 09 from source

Chapter 09 is available as the [chapter-09 release](https://github.com/qshine/mino/releases/tag/chapter-09), with application version `0.9.0`. An installed `mino` already applies the policy. To run the matching source, select its tag from the repository root, build, then start it in a project directory:

```bash
git checkout chapter-09
go build -o bin/mino ./cmd/mino
cd /path/to/your/project
/path/to/mino/bin/mino
```

Replace both paths with your own directories. Startup first prints a one-line policy summary; without a policy file it reads `Policy: Bash ask; 15 protected path patterns; 0 MCP tools allowed without confirmation; 0 tools denied.` The banner reads `Mino - Chapter 09: Guardrails` and the source version is `dev`. Existing Go, Responses tool, compaction, skill, file-tool, and MCP requirements still apply. The no-cost mock checks are in [the lesson](./chapters/09-guardrails.md#_6-check-the-decision-with-mock-tests).

### Chapter 09 policy file

The policy is `~/.mino/policy.json`. It is optional, and Mino never creates or edits it; write it yourself. For example, to remove Bash and let one Context7 tool run without confirmation:

```bash
cat > ~/.mino/policy.json <<'JSON'
{
  "bash": "deny",
  "allow_tools": ["mcp_context7_resolve-library-id"]
}
JSON
chmod 600 ~/.mino/policy.json
```

| Field | Meaning |
| --- | --- |
| `bash` | `"ask"` (default): each command needs approval. `"deny"`: Bash is not offered to the model and every call is refused. |
| `protected_paths` | Patterns added to the defaults. A pattern without `/` matches a file name, such as `*.sqlite`; a pattern ending in `/` matches a directory name anywhere in the path, such as `secrets/`. Patterns cannot contain other slashes. |
| `allow_tools` | Exact MCP tool names, as shown in confirmations, that run without asking. Only names starting with `mcp_` are accepted. |
| `deny_tools` | Exact tool names that are refused and not offered to the model, for example `mcp_tickets_create` or `read_file`. |

The defaults always apply: `.env`, `.env.*`, `*.pem`, `*.key`, `*.p12`, `id_rsa*`, `id_ecdsa*`, `id_ed25519*`, `.netrc`, `.npmrc`, `.pypirc`, and the `.git/`, `.ssh/`, `.aws/`, and `.gnupg/` directories. File tools never read or write a matching path, and there is no prompt to override this. Matching ignores case, as the default macOS file system does, so `.ENV` is protected too.

The file must be a regular file, not a symbolic link, at most 64 KiB, and not writable by group or other users. It must be strict JSON with only the fields above. If any check fails, Mino prints the reason and does not start chat; fix the file or remove it. Mino reads it once at startup and never reads a policy from a project; restart after editing it. A name in `allow_tools` or `deny_tools` that matches no available tool, such as a typo or a server that failed to connect, produces a `Policy warning:` line at startup.

The file tools also change in one way: a path that passes through a symbolic link in any directory component is refused, even when the link stays inside the project. If your project reaches files through a linked directory, use the real path.

Existing `~/.mino/SOUL.md` files remain untouched. The default SOUL for new installations adds a paragraph asking the model to report policy blocks without working around them. Add similar guidance to your own SOUL if you want it, and restart. The policy is enforced either way.

The policy covers which tools run and which files the file tools may touch. It is not a sandbox: an approved Bash command or a configured MCP server runs with your account's permissions. See [the lesson](./chapters/09-guardrails.md#_5-what-the-policy-cannot-see) for these limits.

### Chapter 09 history compatibility

Chapter 09 keeps record version 4 and adds no record kinds or result statuses. A call refused by policy is saved like a call you declined: the model's request followed by a `denied` result. It opens Chapter 08 sessions without migration, and `chapter-08` can still open sessions written by Chapter 09. Rolling back removes the policy, though: `chapter-08` ignores `~/.mino/policy.json`, so protected paths and denied tools are no longer enforced. As before, keep private copies of session logs and `active-session.json` before switching versions.

## Try Chapter 10 from source

Chapter 10 is available as the [chapter-10 release](https://github.com/qshine/mino/releases/tag/chapter-10), with application version `0.10.0`; an installed `mino` already includes `mino eval`. To run the matching source, select its tag from the repository root, build, and run the repository's own cases from there:

```bash
git checkout main
go build -o bin/mino ./cmd/mino
bin/mino eval ./evals
```

The chat banner reads `Mino - Chapter 10: Evals`, `mino help` lists `eval`, and `mino eval help` prints the usage. The default replay mode drives the Agent loop with a scripted local model: it needs no API key or network, and it never reads `~/.mino`, so it cannot touch your settings or sessions. Each case runs in a fresh temporary workspace and home directory that are deleted afterwards. The existing Go requirement still applies; the lesson is [Chapter 10](./chapters/10-evals.md).

| Flag | Meaning |
| --- | --- |
| `PATH` | One case file, or a directory whose `*.yaml` and `*.yml` files are read in name order, not recursively. Default `./evals`. |
| `--live` | Use the model from `~/.mino/config.json` and `~/.mino/SOUL.md` as the default instructions. The configuration must be complete; the command does not prompt. Live runs cost real requests. |
| `--runs N` | Runs per case, 1 to 32. Default 1, or 3 with `--live`. A case's own `runs` field overrides it. |
| `--filter NAME` | Run only the named case. |
| `--out DIR` | Write `report.json` and each run's session log as `DIR/<case>/run-N.jsonl`. The directory may not be inside `~/.mino`. |
| `--timeout DURATION` | Limit for one run, 1s to 30m. Default 2m. |

The exit status is 1 when any case fails any run. Every run uses the default tool policy and connects no MCP servers, whatever your `~/.mino/policy.json` and `~/.mino/mcp.json` say. Bash, when a case approves it, runs in the temporary workspace with your account's permissions; the harness is not a sandbox.

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

It mirrors the repository's `write-then-read.yaml`. Run it alone with `bin/mino eval --filter hello-file ./evals`; observed on `main` under a temporary home directory, it prints one passing row with two requests and one tool call, and the temporary home still contains no `.mino` directory afterwards.

`name` and `prompt` are required; `prompt` is at most 16 KiB. `name` uses lowercase ASCII letters and digits with single hyphens, at most 64 bytes. `approve` is `all` (the default), `none`, or a list of tool names to approve. `instructions` replaces the default instructions for this case. `workspace.files` seeds relative paths before every run, at most 64 files of 64 KiB each. `model` is the replay script of at most 16 steps, each either `call: {tool, arguments}` or `text`, optionally with `expect_input_contains`, a substring that must appear in the request body the model receives at that step; it may be omitted only when you run with `--live`. `graders` holds 1 to 32 checks, all of which must pass: `final_text_contains`, `final_text_matches` (an RE2 pattern of at most 1 KiB), `tool_called`, `tool_not_called`, `tool_order`, `result_status` (`call` is 1-based over every call the model requested), `file_exists`, `file_contains`, `max_requests`, and `max_tool_calls`. Unknown fields fail loading, a directory holds at most 256 cases, and a case file is at most 64 KiB.

The tools a case can call are the chat tools without MCP: `bash`, `read_file`, `write_file`, `edit_file`, and `skill_view`, which reads skills seeded under `.agents/skills` in the workspace. Case files are project data that you run deliberately, like project skills; they are never loaded from `~/.mino`.

### Chapter 10 history compatibility

Chapter 10 keeps record version 4 and adds no record kinds or result statuses. The only runtime addition is an in-memory `response_completed` event that carries token usage to `mino eval` and is never written to a session log. `chapter-09` can therefore open sessions written by Chapter 10, and Chapter 10 opens Chapter 09 sessions without migration. Eval runs write their logs only to temporary directories, or to the `--out` directory you name, never under `~/.mino/sessions`.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository, select `chapter-10`, and run from its directory:

```bash
git clone https://github.com/qshine/mino.git
cd mino
git checkout chapter-10
go run ./cmd/mino
```

Application code and tests live under `internal/`. The chapter snapshots share `internal/gateway/` for terminal interaction and `internal/agent/` for model requests and, from Chapter 02, history. Chapter 03 adds tools under `internal/tools/`. Configuration and identity loading remain in `internal/`; no code-generation step is needed.

The module files remain at the repository root. `go.mod` pins the official `github.com/openai/openai-go/v3` SDK to v3.66.0, and `go.sum` records dependency checksums. Go downloads the dependencies when you first build. You do not need a separate SDK installation.

Downloaded and source builds share your home-directory settings. Chapter 02 and Chapter 03 source runs use your home-directory history; Chapter 04 and later source runs use and migrate it as described above. Automated tests use temporary directories and mock model services, leaving your real history untouched; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md), [Chapter 02: JSONL history](./chapters/02-jsonl-history.md), [Chapter 03: tools and Bash](./chapters/03-tools-and-bash.md), [Chapter 04: multiple sessions](./chapters/04-jsonl-sessions.md), [Chapter 05: context compaction](./chapters/05-context-compaction.md), [Chapter 06: Skills](./chapters/06-skills.md), [Chapter 07: file tools](./chapters/07-file-tools.md), or [Chapter 08: MCP tools](./chapters/08-mcp.md). Run each chapter's experiments at its stated source version; the request and history formats differ.
