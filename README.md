# Mino

English | [简体中文](README.zh-CN.md)

**Build your own agent from scratch in Go: understand each interaction,
then verify how it works with code and experiments.**

Mino is a chapter-by-chapter tutorial for readers new to agents, using Go and
the OpenAI Responses API. Each chapter starts with an observable interaction,
follows what the model receives and what the program handles, then uses a small
experiment to check the result and its limits. Reading the source requires
familiarity with Go variables, functions, and basic error handling; no prior
model API experience is needed.

**Read online:** [English book](https://qshine.github.io/mino/) · [简体中文教程](https://qshine.github.io/mino/zh/)

**Current progress:** This snapshot contains released Chapters 01–05, tagged
`chapter-01` through `chapter-05` with application versions `0.1.0` through `0.5.0`.
Chapter 01 introduces streaming terminal conversations; Chapter 02 adds JSONL
history and restart recovery. Chapter 03 adds individually approved Bash calls and the Agent loop. Chapter 04 adds isolated sessions, selection, and confirmed clearing.
Chapter 05 adds context compaction: manual `/compact`, automatic budget checks, and durable summaries.
Gateway and Agent directories are consistent from Chapter 01 onward. See the [chapter roadmap](docs/books/en/plan-todo-chapters.md).

**By qqling | AI Builder.** I want to build an agent of my own from scratch and
turn what I learn through ongoing research and practice into beginner tutorials.
Read [about the author](docs/books/en/about-author.md) for the story behind Mino
and links to follow or get in touch on X, GitHub, and Xiaohongshu.

## Install

Runs on **macOS 13+**, with downloads for **Apple Silicon and Intel**.
The repository and release downloads are public. No Go installation, GitHub CLI,
or GitHub login is needed; the installer uses the `curl` included with macOS.

Run this **one-line command** in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
```

The installer selects your Mac architecture, downloads the latest release,
verifies its SHA-256 checksum and version, and installs `~/.mino/bin/mino`.
It adds that directory to your zsh or Bash login configuration; the final
`export` also makes the command available in your current terminal. No `sudo`
is needed. Existing settings are preserved.

## Start

From any directory:

```bash
mino
```

On the first launch, enter the missing settings:

```text
API URL [https://api.openai.com/v1]:
Model:
API Key (input hidden):
```

Press Enter to accept the official OpenAI API URL. **The model has no default**
and must be entered. The API key is required and hidden while you type.
Complete settings are saved in `~/.mino/config.json`; subsequent launches go
straight to chat. Only missing fields are requested.

Settings contain `base_url`, `api_key`, and `model`; Chapter 05 also accepts optional `context_window`. The key is stored locally
in plain text, with directory permissions `0700` and file permissions `0600`.
Edit the file to change settings, or clear a required field to be asked for it again.
Project-local configuration, `.env`, and `OPENAI_*` variables are not read.
Older project-local `miniagent.json` files can be moved to the new location if
no user configuration exists yet.

A custom API URL must support `/responses`; enter its API prefix, usually
ending in `/v1`. The streaming build requires Responses streaming with
`text/event-stream`. Remote services require HTTPS.

After configuration is complete, chat startup creates `~/.mino/SOUL.md` from the
bundled [SOUL.md](SOUL.md) if it is missing. This file introduces Mino, its text
assistance, and the current limits. Edit your copy and restart Mino to change its
instructions; existing edits are preserved. Its contents are sent to the model
service, so keep credentials out of it. Mino does not load `AGENTS.md` or `SOUL.md`
from the working directory. `AGENTS.md` is only for developing this repository.
The reissued `chapter-01` includes this identity file.

Type a question and press Enter. Use `/exit`, Ctrl+D on an empty line, or Ctrl+C
to quit. Completed conversations are restored when you restart Mino.

Mino displays answer fragments
as they arrive after `Assistant>`. If a stream fails, the partial answer remains
visible and an error is shown.

## Conversation history (Chapter 02)

Version `chapter-02` introduces Chapter 02 history; Chapter 03 adds the tool-aware
recovery described below.
In Chapters 02 and 03, Mino appends conversation records to `~/.mino/history.jsonl` and restores completed
turns at startup. One local file holds one conversation; `turn_id` associates the
records within each turn. Streamed answers remain immediate. In `chapter-02`, failed
or interrupted turns are retained but excluded from later requests. Chapter 03
also restores paired tool results from stopped turns, since commands may have
already changed something. Requests and commands are never automatically retried.

History and recovery backups contain private conversation data. Files use `0600`
permissions, and only one Mino process can use the history at a time. History
write failures stop chat. An incomplete tail is backed up before repair; invalid
records in the middle stop startup. Limits are 16 MiB per record and 64 MiB per
file. Chapter 04 adds `/new`; Chapter 05 adds context compaction while retaining these original records and file limits.

Existing `~/.mino/SOUL.md` files are preserved. If yours still says every question
is independent, edit that sentence to reflect supplied conversation history; the
updated bundled [SOUL.md](SOUL.md) provides an example. To start fresh before
session commands exist, quit Mino and move `history.jsonl` to a private backup
location. Keep recovery backups private too.

## Bash and the Agent loop (Chapter 03)

Install `chapter-03` with `mino update chapter-03`, or check out that tag and run
`go run ./cmd/mino`. Ask “Which Go version is installed here?” When the model
requests Bash, Mino shows the escaped command, working directory, environment, and limits.
Enter `y` to approve that command or press Enter to deny it. Each call needs its
own approval; piped input cannot approve commands. Results go back to the model,
which can answer or request another tool call.

Tools live in the `internal/tools/` directory, starting with Bash. Commands have a
30 second timeout and a combined 64 KiB stdout/stderr limit; each turn allows
at most 8 model requests and 16 tool calls. Bash runs with your account
permissions. It can access files and networks; approval and process limits do
not provide an OS sandbox. See [Chapter 03](docs/books/en/chapters/03-tools-and-bash.md)
for the interaction, recovery behavior, and verification.

Chapters 03 and 04 write format `v: 2` while retaining support for Chapter 02's `v: 1`
records. Older releases cannot read a history containing new records; keep a
private backup before upgrading to `chapter-03` if you need to return to `chapter-02`.
An interrupted command with no saved result is marked `unknown`. Mino requires
you to acknowledge that it may already have run before chat resumes, and never
runs it again during recovery.

Existing `~/.mino/SOUL.md` files are preserved. If yours still says Mino cannot
execute tools, update those outdated capability sentences using the bundled
[SOUL.md](SOUL.md) as a reference, while keeping your own instructions.

## Multiple sessions (Chapter 04)

Install `chapter-04` with `mino update chapter-04` to use Chapter 04. Startup restores
the last active session and displays its ID. Each session has its own private
`~/.mino/sessions/<id>.jsonl`; only that session supplies request context.

| Command | Behavior |
| --- | --- |
| `/new` | Start an empty session and keep the previous one. |
| `/sessions` | List full IDs and update times; `*` marks the active session. |
| `/resume <id>` | Restore the full ID shown by `/sessions`. |
| `/clear` | Clear the current session after typing its complete ID in an interactive terminal. |
| `/help` | Show the available commands. |

The commands above do not call the model. Clearing keeps the session ID and other sessions;
it does not undo tool effects or delete archives and backups. One Mino process
holds the session store at a time. Switching keeps this launch's SOUL and Bash
working directory. If the saved selection is damaged, Mino asks you to choose a
session instead of silently starting over.

The first upgrade imports legacy `history.jsonl` under its old lock and retains
it as a private archive. Interrupted imports reuse their destination; conflicting
files are not overwritten. Old releases keep using the archive and do not see
new session activity. See [Chapter 04](docs/books/en/chapters/04-jsonl-sessions.md)
for isolation, recovery, and the request-level experiment.

## Context compaction (Chapter 05)

Install `chapter-05` with `mino update chapter-05`. `/compact` summarizes older complete
turns, retaining the latest two turns and any active tool steps in full. Future
requests use the summary plus retained turns; restart and `/resume` restore the
same context. `/clear` also clears the summary. Original JSONL records remain.

The context window defaults to **128K (128,000 tokens)**. To override it, add
`"context_window": 64000` to your existing `~/.mino/config.json` and restart.
Omission or `0` uses the default; negative, fractional, null, or non-integer
values are rejected. Startup displays the value and source, without querying
model metadata. Set it to match your service's actual capacity.

Mino estimates context conservatively from serialized UTF-8 bytes, including
instructions and tool definitions. This is not an exact token count. It reserves
one eighth of the window, capped at 8,192 tokens, for output and automatically
compacts above 80% of the remaining input budget, including after tool results.
Each turn allows one automatic summary request, counted within its eight model
requests. Summaries have no tools and cannot grant approval or resolve unknown
command outcomes.

Failed, empty, oversized, or non-shrinking summaries preserve the previous
context. Input or retained turns that still exceed the budget produce an error;
Mino does not silently truncate history or repeat a command. Summary generation
adds a model request and may lose details. It cannot compact history too large
to fit in one summary request.

Chapter 05 writes `v: 3` records and reads versions 1–3. Earlier releases cannot
read these new session records; keep a private backup before upgrading if you
need to return to Chapter 04. See the [Chapter 05 lesson](docs/books/en/chapters/05-context-compaction.md).

## Version and updates

```bash
mino version             # Show the installed version
mino update              # Install the latest published release
mino update chapter-05   # Install the latest published chapter
```

Git tags use `chapter-NN`; binaries keep `0.N.0`. For example, `chapter-04` reports
`mino 0.4.0`. Later fixes use `chapter-04.1` for `0.4.1`. The installer also accepts
numeric aliases such as `0.4.0` and `v0.4.0`.

Builds installed before 2026-09-26 may not recognize chapter tags. Rerun the
installation command above once to get the updated installer, even if the binary
version is unchanged. Installation and updates preserve configuration, SOUL, and
history; failed downloads or verification leave the existing executable in place.

| Chapter | Application version | Git tag |
| --- | --- | --- |
| Chapter 01 | `0.1.0` | [`chapter-01`](https://github.com/qshine/mino/releases/tag/chapter-01) |
| Chapter 02 | `0.2.0` | [`chapter-02`](https://github.com/qshine/mino/releases/tag/chapter-02) |
| Chapter 03 | `0.3.0` | [`chapter-03`](https://github.com/qshine/mino/releases/tag/chapter-03) |
| Chapter 04 | `0.4.0` | [`chapter-04`](https://github.com/qshine/mino/releases/tag/chapter-04) |
| Chapter 05 | `0.5.0` | [`chapter-05`](https://github.com/qshine/mino/releases/tag/chapter-05) |

A push to `main` runs CI. A chapter tag runs checks, builds both macOS packages,
and publishes their checksums. Source builds report `dev`. This tag migration is
owner-authorized; future fixes use new patch tags without replacing published content.
See [release instructions](docs/books/en/releases.md) and the [changelog](CHANGELOG.md).

## Build and learn

Source development uses **Go 1.27.1** and the official
[OpenAI Go SDK](https://github.com/openai/openai-go), pinned in `go.mod`.
An existing Go 1.21+ installation with the default
`GOTOOLCHAIN=auto` can download the required toolchain.

From the repository root:

```bash
go run ./cmd/mino
bash scripts/check.sh
```

The check script verifies formatting and shell syntax, runs `go vet` and tests
with the race detector, then builds `bin/mino`. Tests use fake keys, temporary
home directories, local mock HTTP servers, and fake release downloads. They do
not call a paid model or modify your real configuration.

- [Read the illustrated book](https://qshine.github.io/mino/) · [简体中文](https://qshine.github.io/mino/zh/)
- [Chapter 01: a terminal conversation](docs/books/en/chapters/01-terminal-chat.md)
- [Chapter 02: JSONL conversation history](docs/books/en/chapters/02-jsonl-history.md)
- [Chapter 03: tools and the Agent loop](docs/books/en/chapters/03-tools-and-bash.md)
- [Chapter 04: multiple sessions](docs/books/en/chapters/04-jsonl-sessions.md)
- [Chapter 05: context compaction](docs/books/en/chapters/05-context-compaction.md)
- [Chapter roadmap](docs/books/en/plan-todo-chapters.md)
- Entry point: `cmd/mino/main.go`; dependency assembly: `internal/app.go`.
- Read the core path: [`CLI.Run`](internal/gateway/cli.go) → [`SessionManager`](internal/agent/session_commands.go) → [`Agent.Handle` and `runLoop`](internal/agent/agent.go). `Handle` saves the user message before `runLoop` directly calls the Responses SDK, saves each complete response, and processes tools.
- [`Session`](internal/agent/session.go) owns in-memory history; [`history.go`](internal/agent/history.go) manages the JSONL file. Memory advances only after a successful write and sync.
- [`Tool`](internal/tools/tool.go) defines preparation and execution; Bash implements it in `internal/tools/bash.go`. Constructors inject dependencies. Gateway handles display and confirmation through the Agent contract.
- The SDK handles API communication. Mino owns terminal interaction, the Agent loop, approval, and local tool execution.
- [Contribution guidelines](AGENTS.md) · [MIT License](LICENSE)

## Preview the tutorial book

Book sources live in `docs/books/en/` and `docs/books/zh/`. Chapters focus on
agent interactions, with setup and release details in supporting pages.
With Node.js 24, run from the repository root:

```bash
./book_review.sh
```

The script installs missing book dependencies, builds this checkout, and opens
Chinese Chapter 01 in your default browser. It uses a free local port; keep the
terminal open and press Ctrl+C to stop the preview. Run it again after edits to
rebuild. For live preview while editing, use `npm run book:dev` instead.
`npm run book:build` checks internal links and builds both languages.

The project `book_writer` subagent maintains the book after Codex code changes.
GitHub Actions checks and publishes the book to GitHub Pages after relevant
updates to `main`. The online links above open the tutorial website.
See [the writing and publishing workflow](docs/books/en/maintaining-the-book.md).
