# Releasing Mino

[English introduction](https://github.com/qshine/mino#readme) · [中文介绍](https://github.com/qshine/mino/blob/main/README.zh-CN.md)

This page covers Go application releases. For the book website, see
[writing and updating the book](./maintaining-the-book.md).

## Version policy

During the tutorial, use `0.<chapter>.<patch>`: `0.1.0` for Chapter 01,
`0.1.1` for its first fix, and `0.2.0` for Chapter 02. Git tags include `v`.
The release workflow passes the tag version into the binary with Go linker
flags. Development builds display `dev`; no source constant needs bumping.

## Chapter 01 baseline reset

At the owner's request, Chapter 01 is reissued as `v0.1.0`, replacing the original
`v0.1.0` and `v0.1.1` releases and tags. The new baseline introduces the official
OpenAI Go SDK, `cmd/mino` entry point, and `internal/mino` application package.
It also retains the private-download fix from the earlier `v0.1.1`.

If you used either earlier build, run `mino update v0.1.0` to install this baseline;
if its embedded updater fails, reinstall with the current
[README installation command](https://github.com/qshine/mino#install).
Both paths preserve your configuration. The version string alone cannot distinguish
the two `0.1.0` builds. This is a one-time reset; subsequent fixes use new patch tags
and leave published tags and assets unchanged.

## Publish a version

1. Finish the chapter or fix, add its entry to `CHANGELOG.md`, and commit it.
2. Run `bash scripts/check.sh`. Review the change and push it to `main`.
3. Create and push the next version tag. For a future patch after the reset, for example:

   ```bash
   git tag -a v0.1.2 -m 'Mino v0.1.2'
   git push origin v0.1.2
   ```

The [Release workflow](https://github.com/qshine/mino/blob/main/.github/workflows/release.yml) runs the checks again,
then builds `darwin/arm64` and `darwin/amd64` with CGO disabled. On macOS,
packaging runs the executable for the host architecture to check its version
and cross-builds the other macOS architecture.
Each archive contains `mino`, `LICENSE`, and `THIRD_PARTY_NOTICES.txt`, which
includes the Go runtime, SDK, and dependency licenses. A release has two archives
and a checksum file:

```text
mino_0.1.2_darwin_arm64.tar.gz
mino_0.1.2_darwin_amd64.tar.gz
checksums.txt
```

The workflow uploads to a draft release first and publishes only after the
uploads succeed. Users then receive the version through `mino update`.
The changelog entry supplies the release notes. Tags without a matching entry
fail packaging. Normal branch pushes run [CI](https://github.com/qshine/mino/blob/main/.github/workflows/ci.yml)
and do not publish a release.

GitHub provides the build machines and download storage; no personal server is
required. Private repositories consume the account's GitHub Actions allowance.
The publishing job uses its built-in `GITHUB_TOKEN` with `contents: write`;
no personal access token or model API key needs to be added to Actions secrets.

## Local packaging and recovery

To inspect packages before publishing:

```bash
bash scripts/package.sh v0.1.0
```

Choose a version already described in `CHANGELOG.md`. Output goes into the
ignored `dist/` directory. Local packaging does not create a tag or a release.

If a run fails, inspect its logs. A failed asset upload can leave a draft;
inspect and remove that draft before rerunning the failed publishing job.
Do not move a published tag or replace a published asset. Fixes receive a new
patch tag instead.

Users can reinstall the Chapter 01 baseline with `mino update v0.1.0`. This replaces
only the executable; it does not roll back or erase `~/.mino/config.json`.

## Private repository installation

The bootstrap command in the README uses authenticated `gh api` to read
`internal/mino/install.sh` from `main`. The installer and the embedded updater use `gh` to
resolve release IDs and download files through GitHub's dedicated release-assets
API, so an incomplete embedded asset list does not block installation. Users
must sign in with an account that can read that repository. Tokens remain managed by GitHub
CLI and are not copied into Mino's configuration.

Installation creates `~/.mino/bin/mino`. First launch asks for model settings
and creates `~/.mino/config.json` only when they are complete. The installer
adds a PATH entry to `.zshrc` (respecting `ZDOTDIR`) or `.bash_profile`; it does
not overwrite a profile. If the profile is a symlink or cannot be written, it
prints the line to add manually. The bootstrap command updates the current
terminal's PATH as well.
