---
prev:
  text: About this book
  link: /
next:
  text: 01 A terminal conversation
  link: /chapters/01-terminal-chat
---

# Setup and installation

Start by running Mino. Chapter 01 then follows the code through startup, a request, and an answer.

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

Settings are saved in `~/.mino/config.json`. Later launches enter chat directly when settings are complete. Updates preserve them. The key is stored locally in plain text, with directory permissions `0700` and file permissions `0600`.

Enter a question after `You>`. Real questions contact your configured service and may incur charges under its terms. Entering only `/exit` checks startup without calling the model.

## Check the version and update

```bash
mino version
mino update
```

Updates still require GitHub access to the repository. Download or asset-verification failures preserve the existing executable, and your model settings remain unchanged.

An optional `AGENTS.md` in the current directory can supply project instructions to the model. An installed Mino can start in a directory with no project files.

## Learn from the source

To run or change the source, use Go 1.27.1 or a newer compatible toolchain. Clone the repository and run from its directory:

```bash
gh repo clone qshine/mino
cd mino
go run .
```

Downloaded and source builds share your home-directory settings. Automated tests use temporary directories and mock model services; no real API key is needed:

```bash
bash scripts/check.sh
```

Continue to [Chapter 01: a terminal conversation](./chapters/01-terminal-chat.md) to see what the program and model each do during a question and answer.
