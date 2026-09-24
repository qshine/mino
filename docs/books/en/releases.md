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

At the owner's explicit request, Chapter 01 is reissued as `v0.1.0` with terminal
streaming, replacing earlier non-streaming `v0.1.0` and intermediate Chapter 01
builds, including `v0.1.1`. The baseline includes the
official OpenAI Go SDK, `cmd/mino` entry point, `internal/` application package,
root `install.sh`, and editable `~/.mino/SOUL.md` identity. `assets.go` embeds the
installer and default identity. The private-download fix from `v0.1.1` is retained.

If you used any earlier build, reinstall with the current
[README installation command](https://github.com/qshine/mino#install).
Do this even if `mino version` already reports `0.1.0`: the version number is
unchanged. Reinstallation preserves your configuration and custom
`~/.mino/SOUL.md`. Replacing the published tag and assets is an explicit
owner-authorized exception for this reissue. Subsequent fixes use new patch tags
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
uploads succeed. Users then receive the version by rerunning the installation
command or using `mino update`, subject to the [embedded updater's requirements](#public-downloads-and-the-embedded-updater).
The changelog entry supplies the release notes. Tags without a matching entry
fail packaging. Normal branch pushes run [CI](https://github.com/qshine/mino/blob/main/.github/workflows/ci.yml)
and do not publish a release.

GitHub provides the build machines and public release downloads; no personal
server is required.
The publishing job uses its built-in `GITHUB_TOKEN` with `contents: write`;
no personal access token or model API key needs to be added to Actions secrets.

## Local packaging and recovery

To inspect packages before publishing:

```bash
bash scripts/package.sh v0.1.0
```

Run this from the repository root and choose a version already described in
`CHANGELOG.md`. Packaging builds the current working tree, not a checkout of the
named tag. Output goes into the ignored `dist/` directory; it does not create a
tag or a release.

If a run fails, inspect its logs. A failed asset upload can leave a draft;
inspect and remove that draft before rerunning the failed publishing job.
Do not move a published tag or replace a published asset. Fixes receive a new
patch tag instead.

Users can reinstall with the [current installation command](./getting-started.md#install-with-one-command).
It replaces only the executable and preserves `~/.mino/config.json` and custom
`~/.mino/SOUL.md`. To select a particular release from a source checkout, run
`bash install.sh v0.1.0` from the repository root.

## Public downloads and the embedded updater

The repository is public. The README command uses `curl` to fetch root
`install.sh` from `main`, without GitHub CLI or a GitHub login. This installer
change is **Unreleased**; it downloads existing published application packages.
It resolves the latest release once, downloads the package and `checksums.txt`
from the same tag, verifies SHA-256 and the executable's version, and only then
replaces the installed executable. A specified version skips latest-release
discovery.

`assets.go` embeds the installer when the application is built, and
`mino update` runs that embedded copy. The published `v0.1.0` therefore still
uses the earlier GitHub CLI updater and requires a GitHub login, including for
`mino update v0.1.0`. Updating the script on `main` does not change a published
binary. Until a new application release includes the public-download updater,
rerun the README installation command to update without GitHub CLI.

Installation creates `~/.mino/bin/mino`. First launch asks for model settings
and creates `~/.mino/config.json` only when they are complete. The installer
adds a PATH entry to `.zshrc` (respecting `ZDOTDIR`) or `.bash_profile`; it does
not overwrite a profile. If the profile is a symlink or cannot be written, it
prints the line to add manually. The bootstrap command updates the current
terminal's PATH as well.
