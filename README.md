# Mino

English | [简体中文](README.zh-CN.md)

A small terminal agent built chapter by chapter in Go. Chapter 01 provides
independent conversations through the OpenAI Responses API. Runs on **macOS 13+**,
with downloads for **Apple Silicon and Intel**.

## Install

No Go installation is needed. While this repository is private, you need a GitHub
account with access to `qshine/mino` and [GitHub CLI](https://cli.github.com/).
If you use Homebrew, install the CLI with `brew install gh`, then sign in once:

```bash
gh auth login --hostname github.com
```

Run this **one-line command** in Bash or zsh:

```bash
mino_installer="$(gh api --hostname github.com -H 'Accept: application/vnd.github.raw+json' 'repos/qshine/mino/contents/install.sh?ref=main')" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
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

Settings contain `base_url`, `api_key`, and `model`. The key is stored locally
in plain text, with directory permissions `0700` and file permissions `0600`.
Edit the file to change settings, or clear a field to be asked for it again.
Project-local configuration, `.env`, and `OPENAI_*` variables are not read.
Older project-local `miniagent.json` files can be moved to the new location if
no user configuration exists yet.

A custom API URL must support `/responses`; enter its API prefix, usually
ending in `/v1`. Remote services require HTTPS. If the current directory has an
`AGENTS.md`, Mino loads it as model instructions; it is optional.

Type a question and press Enter. Use `/exit`, Ctrl+D on an empty line, or Ctrl+C
to quit. Chapter 01 does not retain conversation history.

## Version and updates

```bash
mino version          # Show the installed version
mino update           # Install the latest published release
mino update v0.1.0    # Install a specific release, including a rollback
```

Updates manage `~/.mino/bin/mino`, reuse your GitHub login, and preserve your
configuration. A failed download or verification keeps the existing executable.

| Progress | Version | Git tag |
| --- | --- | --- |
| Chapter 01 | `0.1.0` | `v0.1.0` |
| Chapter 01 fixes | `0.1.1`, `0.1.2` | `v0.1.1`, `v0.1.2` |
| Chapter 02 | `0.2.0` | `v0.2.0` |
| Chapter 02 fixes | `0.2.1` | `v0.2.1` |

A push to `main` runs CI. A version tag runs tests, builds both macOS packages,
and publishes a GitHub Release with checksums. Git tags provide the version
embedded in release binaries; source builds report `dev`. Published tags and
assets stay unchanged. See [release instructions](docs/releases.md) and the
[changelog](CHANGELOG.md).

## Build and learn

Source development uses **Go 1.27.1** and the standard library, with no
third-party Go dependencies. An existing Go 1.21+ installation with the default
`GOTOOLCHAIN=auto` can download the required toolchain.

From the repository root:

```bash
go run .
bash scripts/check.sh
```

The check script verifies formatting and shell syntax, runs `go vet` and tests
with the race detector, then builds `bin/mino`. Tests use fake keys, temporary
home directories, local mock HTTP servers, and fake release downloads. They do
not call a paid model or modify your real configuration.

- [Chapter 01: terminal chat (Chinese)](docs/chapters/01-terminal-chat.md)
- [Chapter plan (Chinese)](docs/plan-todo-chapters.md)
- Reading order: `main.go` → `cli.go` → `config.go` / `config_prompt.go` → `terminal.go` → `responses.go`.
- [Contribution guidelines](AGENTS.md) · [MIT License](LICENSE)
