# Releasing Mino

This page covers application releases. For the book website, see [writing and updating the book](./maintaining-the-book.md).

Chapters 01–04 were published on **2026-09-26** as `chapter-01` through `chapter-04`, with macOS packages for Apple Silicon and Intel. This checkout contains the implementation through Chapter 03; each lesson links to its own release source.

## Version policy

Release tags identify chapters; executables and archive filenames retain numeric versions:

| Release tag | Application version |
| --- | --- |
| `chapter-01` | `0.1.0` |
| `chapter-02` | `0.2.0` |
| `chapter-03` | `0.3.0` |
| `chapter-04` | `0.4.0` |

An initial chapter release uses `chapter-NN`. A later fix uses `chapter-NN.PATCH`: for example, `chapter-04.1` produces version `0.4.1`. Future fixes receive new tags; do not move published tags or replace their assets. Prefer one complete commit per chapter, including code, tests, and both book languages.

Packaging derives `0.<chapter>.<patch>` from the tag and supplies it through Go linker flags. Source builds display `dev`. The changelog heading stays numeric, such as `## [0.4.1]`.

## Migrate an existing installation

**An updater installed before the chapter-tag migration cannot resolve `chapter-*` tags.** Rerun the [current installation command](./getting-started.md#install-with-one-command) once, even if the displayed numeric version is unchanged. This fetches the new installer and replaces the executable while preserving settings, custom `~/.mino/SOUL.md`, and any history. Updating a tag or the script on `main` does not change an installed binary.

After that installation, `mino update` selects the latest release and `mino update chapter-03` selects this chapter. The installer also accepts `0.3.0` or `v0.3.0` as aliases for `chapter-03`; these are version selectors, not additional Git tags.

## Publish a version

1. Finish the change, add the numeric version entry to `CHANGELOG.md`, and commit it.
2. Run `bash scripts/check.sh`, review the code and bilingual book, then push the reviewed change to `main`.
3. Create and push a new chapter or patch tag. For a future Chapter 02 patch:

   ```bash
   git tag -a chapter-02.1 -m 'Mino chapter-02.1'
   git push origin chapter-02.1
   ```

The [Release workflow](https://github.com/qshine/mino/blob/main/.github/workflows/release.yml) runs for `chapter-*` tags. It reruns checks and builds `darwin/arm64` and `darwin/amd64` with CGO disabled. On macOS, packaging also runs the host-architecture executable to verify its numeric version.

Each archive contains `mino`, `LICENSE`, and `THIRD_PARTY_NOTICES.txt`. The example patch would provide:

```text
mino_0.2.1_darwin_arm64.tar.gz
mino_0.2.1_darwin_amd64.tar.gz
checksums.txt
```

The workflow uploads to a draft release, then publishes after all uploads succeed. The matching numeric changelog entry supplies the notes; packaging fails if it is missing. Normal branch pushes run CI without publishing an application release. The publishing job uses GitHub's built-in `GITHUB_TOKEN` with `contents: write`; it needs no model API key.

## Local packaging and recovery

From the repository root at `chapter-03`, inspect this chapter's packages:

```bash
bash scripts/package.sh chapter-03
```

Packaging accepts the canonical chapter tag and builds the current working tree; it does not check out the tag. Output goes to the ignored `dist/` directory without creating a Git tag or release. If publishing fails, inspect the logs and any incomplete draft before rerunning the job.

## Public downloads and the embedded updater

The installer uses `curl`, without GitHub CLI or a GitHub login. It resolves the latest tag once unless you specify a release, then downloads the package and `checksums.txt` from that same tag. It verifies SHA-256 and the executable's numeric version before replacing `~/.mino/bin/mino`; download or verification failures preserve the existing executable.

Each chapter release embeds the installer used by `mino update`. Installation configures PATH through `.zshrc` (respecting `ZDOTDIR`) or `.bash_profile` without overwriting the profile. For a symlink or unwritable profile, it prints the line to add manually. First launch creates the model configuration only after you complete setup.
