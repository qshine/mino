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
mino_installer="$(gh api --hostname github.com -H 'Accept: application/vnd.github.raw+json' 'repos/qshine/mino/contents/install.sh?ref=main')" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
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

A custom service must support Responses API streaming: `stream: true` requests and `text/event-stream` responses. Mino rejects endpoints that return only complete JSON. Enter the API prefix, such as `https://gateway.example.com/v1`, without appending `/responses` or `/chat/completions`. Remote addresses require HTTPS.

Settings are saved in `~/.mino/config.json`. Later launches enter chat directly when settings are complete. Updates preserve them. The key is stored locally in plain text, with directory permissions `0700` and file permissions `0600`.

To change the service, model, or key, edit that file and restart. An empty field is prompted again; Ctrl+C during setup leaves existing settings unchanged. Mino does not read project-local configuration, `.env`, or `OPENAI_*` environment variables. Keep the configuration file out of the repository.

Enter a question after `You>`. Real questions contact your configured service and may incur charges under its terms. Entering only `/exit` checks startup without calling the model.

## Check the version and update

```bash
mino version
mino update
```

Updates still require GitHub access to the repository. Download or asset-verification failures preserve the existing executable, and your model settings remain unchanged.

The reissued `v0.1.0` adds terminal streaming; earlier builds with the same version number wait for the complete answer. Run `mino update v0.1.0` to obtain the streaming release **even if `mino version` already reports `0.1.0`**. If the earlier updater fails, use the installation command above. Both paths preserve your configuration and custom `~/.mino/SOUL.md`. This replacement is an explicit owner-authorized exception; see the [release reset note](./releases.md#chapter-01-baseline-reset).

## Mino identity

After settings are complete, Mino loads `~/.mino/SOUL.md` once before starting chat. If the file is missing, Mino creates it from the bundled default identity. Existing content is preserved across restarts and updates; incomplete or cancelled configuration does not create the file.

The default describes Mino as a terminal assistant that helps with questions, explanations, writing, and code supplied by the user. It asks the model to use your language and describe its limits honestly. Edit the user file to change this guidance, then restart Mino; there is no live reload. The complete text is sent to your configured model service in `instructions`, so keep credentials out of it.

Use a regular file containing non-empty UTF-8 text, at most 64 KiB. Mino rejects directories, symbolic links, and invalid text before sending any model request. It uses directory permissions `0700` and file permissions `0600`. Mino ignores both `AGENTS.md` and `SOUL.md` in the working directory: repository `AGENTS.md` is for development, while root `SOUL.md` is only the build-time default.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository and run from its directory:

```bash
gh repo clone qshine/mino
cd mino
go run ./cmd/mino
```

`cmd/mino/main.go` is the executable entry point. Application code and its tests live together in the `mino` package under `internal/`; `app.go` connects startup to the terminal loop. Start with `app.go`, then follow `terminal.go` into `responses.go`.

The entry point imports `github.com/qshine/mino/internal` as `mino`, so the call remains `mino.Main(version)`.

Root `assets.go` embeds `install.sh` and the default `SOUL.md`. `internal/cli.go` uses `assets.InstallerScript` for updates, and `internal/soul.go` uses `assets.DefaultSoul` to initialize the user identity. Both work without a source checkout.

The module files remain at the repository root. `go.mod` pins the official `github.com/openai/openai-go/v3` SDK to v3.66.0, and `go.sum` records dependency checksums. Go downloads the dependencies when you first build. You do not need a separate SDK installation.

Downloaded and source builds share your home-directory settings. Automated tests use temporary directories and mock model services; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md) to see what the program and model each do during a question and answer.
