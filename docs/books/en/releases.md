# Releasing Mino

This page covers application releases. For the book website, see [writing and updating the book](./maintaining-the-book.md).

This checkout corresponds to the [chapter-08](https://github.com/qshine/mino/releases/tag/chapter-08) release, application version **0.8.0**, published on **2026-10-21** with macOS packages for Apple Silicon and Intel. Each lesson uses its matching source tag, and each later chapter has its own release. Building the working tree does not create a release or change published tags.

## Republished releases

On 2026-10-21 the tutorial was restructured from nine chapters into ten, and every chapter tag and release was republished. The former Chapter 03 became Chapter 03 (the Agent loop) and Chapter 04 (recovering interrupted tool turns), and later chapters moved up by one, so a tag such as `chapter-05` now names a different lesson than before. Releases published before that date used the old numbering and are superseded. Replacing published tags was a one-time exception authorized by the owner; future fixes receive new patch tags.

An executable installed before 2026-10-21 still reports its old version number; run `mino update` to switch to a republished release. Sessions that earlier releases wrote under `~/.mino/sessions/` remain readable, but a legacy `~/.mino/history.jsonl` is not imported. Keep it as a private backup.

## Session compatibility

Each release reads the session records written by earlier chapters, but an older release may refuse a session once a newer one has written records it does not know. Before switching versions, close Mino and keep private copies of your session files.

Chapter 03 writes version 2 session records and reads versions 1–2; `chapter-02` cannot read a session after version 2 records are appended. See [Chapter 03 setup](./getting-started.md#try-chapter-03-from-source).

Chapter 04 keeps version 2, but `chapter-03` cannot open a session after Chapter 04 saves a `recovery_ack` record; see [Chapter 04 setup](./getting-started.md#try-chapter-04-from-source).

Chapter 05 also keeps version 2 and adds multiple sessions. Once `/new` has created a second session, Chapters 02–04 stop at startup; see [Chapter 05 session storage](./getting-started.md#chapter-05-session-storage).

Chapter 06 writes version 3 session records and reads versions 1–3. Before trying it or rolling back, follow the [history compatibility notes](./getting-started.md#chapter-06-history-compatibility).

Chapter 07 writes version 4 records and reads versions 1–4; see [Chapter 07 compatibility](./getting-started.md#chapter-07-history-compatibility) before using existing sessions or rolling back.

Chapter 08 keeps record version 4, but `chapter-07` cannot load a session that started a file-tool call; see [Chapter 08 compatibility](./getting-started.md#chapter-08-history-compatibility) before rolling back.

## Version policy

Release tags identify chapters; executables and archive filenames retain numeric versions:

| Release tag | Application version |
| --- | --- |
| `chapter-01` | `0.1.0` |
| `chapter-02` | `0.2.0` |
| `chapter-03` | `0.3.0` |
| `chapter-04` | `0.4.0` |
| `chapter-05` | `0.5.0` |
| `chapter-06` | `0.6.0` |
| `chapter-07` | `0.7.0` |
| `chapter-08` | `0.8.0` |
| `chapter-09` | `0.9.0` |
| `chapter-10` | `0.10.0` |

An initial chapter release uses `chapter-NN`. A later fix uses `chapter-NN.PATCH`: for example, `chapter-05.1` produces version `0.5.1`. Future fixes receive new tags; do not move published tags or replace their assets. The [2026-10-21 republication](#republished-releases) was a one-time exception. Prefer one complete commit per chapter, including code, tests, and both book languages.

Packaging derives `0.<chapter>.<patch>` from the tag and supplies it through Go linker flags. Source builds display `dev`. The changelog heading stays numeric, such as `## [0.5.1]`.

## Migrate an existing installation

**An updater installed before the chapter-tag migration cannot resolve `chapter-*` tags.** Rerun the [current installation command](./getting-started.md#install-with-one-command) once, even if the displayed numeric version is unchanged. This fetches the new installer and replaces the executable while preserving settings, custom `~/.mino/SOUL.md`, and any history. Updating a tag or the script on `main` does not change an installed binary.

After that installation, `mino update` selects the latest release and `mino update chapter-08` selects this chapter. The installer also accepts `0.8.0` or `v0.8.0` as aliases for `chapter-08`; these are version selectors, not additional Git tags.

## Publish a version

1. Finish the change, add the numeric version entry to `CHANGELOG.md`, and commit it.
2. Run `bash scripts/check.sh`, review the code and bilingual book, then push the reviewed change to `main`.
3. Create and push a new chapter or patch tag. For example, a Chapter 05 patch:

   ```bash
   git tag -a chapter-05.1 -m 'Mino chapter-05.1'
   git push origin chapter-05.1
   ```

The [Release workflow](https://github.com/qshine/mino/blob/main/.github/workflows/release.yml) runs for `chapter-*` tags. It reruns checks and builds `darwin/arm64` and `darwin/amd64` with CGO disabled. On macOS, packaging also runs the host-architecture executable to verify its numeric version.

Each archive contains `mino`, `LICENSE`, and `THIRD_PARTY_NOTICES.txt`. The example patch would provide:

```text
mino_0.5.1_darwin_arm64.tar.gz
mino_0.5.1_darwin_amd64.tar.gz
checksums.txt
```

The workflow uploads to a draft release, then publishes after all uploads succeed. The matching numeric changelog entry supplies the notes; packaging fails if it is missing. Normal branch pushes run CI without publishing an application release. The publishing job uses GitHub's built-in `GITHUB_TOKEN` with `contents: write`; it needs no model API key.

## Local packaging and recovery

From the repository root at `chapter-08`, inspect this chapter's packages:

```bash
bash scripts/package.sh chapter-08
```

Packaging accepts the canonical chapter tag and builds the current working tree; it does not check out the tag. Output goes to the ignored `dist/` directory without creating a Git tag or release. If publishing fails, inspect the logs and any incomplete draft before rerunning the job.

## Public downloads and the embedded updater

The installer uses `curl`, without GitHub CLI or a GitHub login. It resolves the latest tag once unless you specify a release, then downloads the package and `checksums.txt` from that same tag. It verifies SHA-256 and the executable's numeric version before replacing `~/.mino/bin/mino`; download or verification failures preserve the existing executable.

Each chapter release embeds the installer used by `mino update`. Installation configures PATH through `.zshrc` (respecting `ZDOTDIR`) or `.bash_profile` without overwriting the profile. For a symlink or unwritable profile, it prints the line to add manually. First launch creates the model configuration only after you complete setup.
