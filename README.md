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

**Current progress:** This snapshot contains released Chapters 01–02, tagged
`chapter-01` through `chapter-02` with application versions `0.1.0` through `0.2.0`.
Chapter 01 introduces streaming terminal conversations; Chapter 02 adds JSONL
history and restart recovery.
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
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-02 && export PATH="$HOME/.mino/bin:$PATH"
```

The installer selects your Mac architecture, downloads this chapter release,
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

Settings contain `base_url`, `api_key`, and `model`. The key is stored locally
in plain text, with directory permissions `0700` and file permissions `0600`.
Edit the file to change settings, or clear a field to be asked for it again.
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

Install `chapter-02` to use Chapter 02, or run `go run ./cmd/mino` from this checkout.
Mino appends conversation records to `~/.mino/history.jsonl` and restores completed
turns at startup. One local file holds one conversation; `turn_id` associates the
records within each turn. Streamed answers remain immediate. Failed or interrupted turns
are retained as records but excluded from later requests; requests are never
automatically retried.

History and recovery backups contain private conversation data. Files use `0600`
permissions, and only one Mino process can use the history at a time. History
write failures stop chat. An incomplete tail is backed up before repair; invalid
records in the middle stop startup. Limits are 16 MiB per record and 64 MiB per
file; context compaction and `/new` are later chapters.

Existing `~/.mino/SOUL.md` files are preserved. If yours still says every question
is independent, edit that sentence to reflect supplied conversation history; the
updated bundled [SOUL.md](SOUL.md) provides an example. To start fresh before
session commands exist, quit Mino and move `history.jsonl` to a private backup
location. Keep recovery backups private too.

## Version and updates

```bash
mino version             # Show the installed version
mino update              # Install the latest published release
mino update chapter-02   # Install this snapshot's chapter
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
- [Chapter roadmap](docs/books/en/plan-todo-chapters.md)
- Entry point: `cmd/mino/main.go`; application code and tests: `internal/`.
- Application composition: `internal/app.go`; terminal interaction: `internal/gateway/`;
  model requests, Agent handling, and JSONL history: `internal/agent/`. Configuration and SOUL remain in `internal/`.
- The SDK handles API communication. Mino owns the terminal flow; tool execution and the Agent loop remain future chapters.
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
