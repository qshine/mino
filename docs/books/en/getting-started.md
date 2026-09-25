---
prev:
  text: About this book
  link: /
next:
  text: 01 A terminal conversation
  link: /chapters/01-terminal-chat
---

# Setup and installation

This page prepares the `chapter-01` release (application version `0.1.0`), published on 2026-10-21. It covers the book up to Chapter 01; the [chapter roadmap](./plan-todo-chapters.md) lists what each chapter adds. Use the tag named in each lesson when running its experiments.


## Prepare your Mac

Mino supports macOS 13 or later, with packages for Apple Silicon and Intel. The installer selects your architecture. You do not need Go to run a downloaded release.

The repository and release downloads are public. Installation uses macOS's `curl`; you do not need GitHub CLI or a GitHub login. A model API key is needed only when configuring the model service on first launch.

## Install with one command

Run this in Bash or zsh:

```bash
mino_installer="$(curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/qshine/mino/main/install.sh)" && bash -c "$mino_installer" -- chapter-01 && export PATH="$HOME/.mino/bin:$PATH"
```

This command fetches the current installer from `main` and selects `chapter-01`. Omit `-- chapter-01` to install the latest published release instead. The installer downloads the macOS package and checksum file from one resolved release tag.

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

Mino sends an ordered `input` list and requests `reasoning.encrypted_content`. A compatible service must accept these fields and accept its returned response items on subsequent requests. For a text-only response that omits output items on successful completion, Mino uses the completed streamed text as the saved answer.

Settings are saved in `~/.mino/config.json`. Later launches enter chat directly when settings are complete. Updates preserve them. The key is stored locally in plain text, with directory permissions `0700` and file permissions `0600`.

To change the service, model, or key, edit that file and restart. An empty field is prompted again; Ctrl+C during setup leaves existing settings unchanged. Mino does not read project-local configuration, `.env`, or `OPENAI_*` environment variables. Keep the configuration file out of the repository.

Enter a question after `You>`. Real questions contact your configured service and may incur charges under its terms. Entering only `/exit` checks startup without calling the model.

## Check the version and update

If Mino was installed before the chapter-tag migration, rerun the installation command above **once**, even if the displayed numeric version is unchanged. The old embedded updater cannot resolve `chapter-*` tags. Reinstallation preserves your settings, custom `~/.mino/SOUL.md`, and any history.

After installing a chapter-tag release, you can check the numeric version and select this chapter again:

```bash
mino version
mino update chapter-01
```

`mino update` without an argument selects the latest release. The updated installer also accepts `0.1.0` or `v0.1.0` as aliases for `chapter-01`. Download or verification failures preserve the existing executable. Releases installed before 2026-10-21 use the earlier chapter numbering; see [republished releases](./releases.md#republished-releases). See the [release and patch rules](./releases.md#version-policy) for version numbers.

## Mino identity

After settings are complete, Mino loads `~/.mino/SOUL.md` once before starting chat. If the file is missing, Mino creates it from the bundled default identity. Existing content is preserved across restarts and updates; incomplete or cancelled configuration does not create the file.

The default describes Mino as a terminal assistant that helps with questions, explanations, writing, and code supplied by the user. It asks the model to use your language and describe its limits honestly. Edit the user file to change this guidance, then restart Mino; there is no live reload. The complete text is sent to your configured model service in `instructions`, so keep credentials out of it.

The Chapter 02 default describes supplied conversation history. The Chapter 03 default also describes paired tool results and Bash commands that require approval. The Chapter 04 default adds that an unknown result is not success and must never be retried automatically. Mino preserves existing identity files. When changing chapters, update only the capability statements that differ in the version you will run: independent questions end in Chapter 02, while tools arrive in the Chapter 03 release. Keep your other custom instructions and restart after editing.

Use a regular file containing non-empty UTF-8 text, at most 64 KiB. Mino rejects directories, symbolic links, and invalid text before sending any model request. It uses directory permissions `0700` and file permissions `0600`. Mino ignores both `AGENTS.md` and `SOUL.md` in the working directory: repository `AGENTS.md` is for development, while root `SOUL.md` is only the build-time default.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository, select `chapter-01`, and run from its directory:

```bash
git clone https://github.com/qshine/mino.git
cd mino
git checkout chapter-01
go run ./cmd/mino
```

Application code and tests live under `internal/`. `internal/gateway/` handles terminal interaction, and `internal/agent/` handles model requests. Configuration and identity loading remain in `internal/`; no code-generation step is needed.

The module files remain at the repository root. `go.mod` pins the official `github.com/openai/openai-go/v3` SDK to v3.66.0, and `go.sum` records dependency checksums. Go downloads the dependencies when you first build. You do not need a separate SDK installation.

Downloaded and source builds share your home-directory settings. Automated tests use temporary home directories and mock model services, leaving your real `~/.mino` files untouched; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md) and follow the chapters in order. Run each chapter's experiments at its stated source version; the request and history formats differ.
