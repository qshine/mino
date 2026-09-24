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

The repository and release downloads are public. Installation uses macOS's `curl`; you do not need GitHub CLI or a GitHub login. A model API key is needed only when configuring the model service on first launch.

## Install with one command

Run this in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" && export PATH="$HOME/.mino/bin:$PATH"
```

This command uses the **Unreleased installer on `main`** to download the latest published application, currently the `v0.1.0` streaming reissue. It resolves one release tag, then downloads its macOS package and checksum file over HTTPS.

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

Settings are saved in `~/.mino/config.json`. Later launches enter chat directly when settings are complete. Updates preserve them. The key is stored locally in plain text, with directory permissions `0700` and file permissions `0600`.

To change the service, model, or key, edit that file and restart. An empty field is prompted again; Ctrl+C during setup leaves existing settings unchanged. Mino does not read project-local configuration, `.env`, or `OPENAI_*` environment variables. Keep the configuration file out of the repository.

Enter a question after `You>`. Real questions contact your configured service and may incur charges under its terms. Entering only `/exit` checks startup without calling the model.

## Check the version and update

```bash
mino version
```

To install the latest release without a GitHub login, rerun the installation command above. Download or verification failures preserve the existing executable; installation also preserves your model settings and custom `~/.mino/SOUL.md`.

The published `v0.1.0` still embeds the earlier updater: its `mino update` command requires GitHub CLI and a GitHub login. The public-download updater is Unreleased and will be included in a future application release. See [public downloads and the embedded updater](./releases.md#public-downloads-and-the-embedded-updater) for the distinction.

Earlier builds also report `0.1.0` but wait for the complete answer. If you used one of those builds, rerun the installation command to obtain streaming **even if `mino version` already reports `0.1.0`**. The [release reset note](./releases.md#chapter-01-baseline-reset) explains this owner-authorized replacement.

## Mino identity

After settings are complete, Mino loads `~/.mino/SOUL.md` once before starting chat. If the file is missing, Mino creates it from the bundled default identity. Existing content is preserved across restarts and updates; incomplete or cancelled configuration does not create the file.

The default describes Mino as a terminal assistant that helps with questions, explanations, writing, and code supplied by the user. It asks the model to use your language and describe its limits honestly. Edit the user file to change this guidance, then restart Mino; there is no live reload. The complete text is sent to your configured model service in `instructions`, so keep credentials out of it.

Use a regular file containing non-empty UTF-8 text, at most 64 KiB. Mino rejects directories, symbolic links, and invalid text before sending any model request. It uses directory permissions `0700` and file permissions `0600`. Mino ignores both `AGENTS.md` and `SOUL.md` in the working directory: repository `AGENTS.md` is for development, while root `SOUL.md` is only the build-time default.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository and run from its directory:

```bash
git clone https://github.com/qshine/mino.git
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
