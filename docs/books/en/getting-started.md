---
prev:
  text: About this book
  link: /
next:
  text: 01 A terminal conversation
  link: /chapters/01-terminal-chat
---

# Setup and installation

Use this page to prepare Mino. Chapter 01 then follows one question from terminal input to a model reply.

## Prepare your Mac

Mino supports macOS 13 or later, with packages for Apple Silicon and Intel. The installer selects your architecture. You do not need Go to run a downloaded release.

The repository is currently private. Downloads require a GitHub account with repository access and [GitHub CLI](https://cli.github.com/). If you use Homebrew, install it with `brew install gh`, then sign in once:

```bash
gh auth login --hostname github.com
```

Your GitHub login downloads the program. The model API key entered later accesses the model service. These are separate credentials.

## Install with one command

Run this in Bash or zsh:

```bash
mino_installer="$(gh api --hostname github.com -H 'Accept: application/vnd.github.raw+json' 'repos/qshine/mino/contents/internal/install.sh?ref=main')" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
```

The installer verifies the package's SHA-256 checksum and executable version, installs `~/.mino/bin/mino`, and configures your terminal's command search path. Installation creates `~/.mino`; the configuration file is created after you complete setup on first launch.

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

A custom service must support the Responses API. Enter its API prefix, such as `https://gateway.example.com/v1`, without appending `/responses` or `/chat/completions`. Remote addresses require HTTPS.

Settings are saved in `~/.mino/config.json`. Later launches enter chat directly when settings are complete. Updates preserve them. The key is stored locally in plain text, with directory permissions `0700` and file permissions `0600`.

To change the service, model, or key, edit that file and restart. An empty field is prompted again; Ctrl+C during setup leaves existing settings unchanged. Mino does not read project-local configuration, `.env`, or `OPENAI_*` environment variables. Keep the configuration file out of the repository.

Enter a question after `You>`. Real questions contact your configured service and may incur charges under its terms. Entering only `/exit` checks startup without calling the model.

## Check the version and update

```bash
mino version
mino update
```

Updates still require GitHub access to the repository. Download or asset-verification failures preserve the existing executable, and your model settings remain unchanged.

Chapter 01 was reset to a new `v0.1.0` baseline with the official SDK. If you installed the earlier `v0.1.0` or `v0.1.1`, run `mino update v0.1.0` to install the reissued build. If the earlier updater fails, use the installation command above; existing configuration is preserved. See the [release reset note](./releases.md#chapter-01-baseline-reset).

## Optional project instructions

An `AGENTS.md` in the current working directory can supply instructions to the model. Mino reads it once at startup, without searching parent directories; restart after editing it. The file is optional, so an installed Mino can start in a directory with no project files. Its contents are sent to the configured model service: do not put credentials in it.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository and run from its directory:

```bash
gh repo clone qshine/mino
cd mino
go run ./cmd/mino
```

`cmd/mino/main.go` is the executable entry point. Application code and its tests live together in the `mino` package under `internal/`; `app.go` connects startup to the terminal loop. The installer lives there too so `cli.go` can embed it for `mino update`. Start with `app.go`, then follow `terminal.go` into `responses.go`.

The entry point imports `github.com/qshine/mino/internal` as `mino`, so the call remains `mino.Main(version)`.

The module files remain at the repository root. `go.mod` pins the official `github.com/openai/openai-go/v3` SDK to v3.66.0, and `go.sum` records dependency checksums. Go downloads the dependencies when you first build. You do not need a separate SDK installation.

Downloaded and source builds share your home-directory settings. Automated tests use temporary directories and mock model services; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md) to see what the program and model each do during a question and answer.
