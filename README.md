# Mino

English | [简体中文](README.zh-CN.md)

A small terminal agent built chapter by chapter in Go. Chapter 01 provides
independent conversations through the OpenAI Responses API. Runs on **macOS 13+**,
with downloads for **Apple Silicon and Intel**.

**Read online:** [English book](https://qshine.github.io/mino/) · [简体中文教程](https://qshine.github.io/mino/zh/)

## Install

The repository and release downloads are public. No Go installation, GitHub CLI,
or GitHub login is needed; the installer uses the `curl` included with macOS.

Run this **one-line command** in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-01 && export PATH="$HOME/.mino/bin:$PATH"
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
to quit. Chapter 01 does not retain conversation history.

The reissued `chapter-01` streaming build displays answer fragments
as they arrive after `Assistant>`. If a stream fails, the partial answer remains
visible and an error is shown. Rerun the installation command above to get
streaming, even if your installed version already reports `0.1.0`; earlier
builds with this version number waited for the complete answer.

## Version and updates

```bash
mino version             # Show the installed version
mino update              # Install the latest published release
mino update chapter-01   # Install this snapshot's chapter
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
- [Chapter roadmap](docs/books/en/plan-todo-chapters.md)
- Entry point: `cmd/mino/main.go`; application code and tests: `internal/`.
- Application composition: `internal/app.go`; terminal interaction: `internal/gateway/`;
  model requests: `internal/agent/`. Configuration and SOUL remain in `internal/`.
- The SDK handles API communication. Mino owns the terminal flow; tool execution and the Agent loop remain future chapters.
- [Contribution guidelines](AGENTS.md) · [MIT License](LICENSE)

## Preview the tutorial book

Book sources live in `docs/books/en/` and `docs/books/zh/`. Chapters focus on
agent interactions, with setup and release details in supporting pages.
With Node.js 24, run `npm ci --ignore-scripts` and
`npm run book:dev`, then open the local address shown. `npm run book:build`
checks the site's internal links and builds both languages.

The project `book_writer` subagent maintains the book after Codex code changes.
GitHub Actions checks and publishes the book to GitHub Pages after relevant
updates to `main`. The online links above open the tutorial website.
See [the writing and publishing workflow](docs/books/en/maintaining-the-book.md).
