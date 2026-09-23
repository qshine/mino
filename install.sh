#!/bin/bash
set -euo pipefail
umask 077

fail() { printf 'Error: %s\n' "$*" >&2; exit 1; }

if [[ $# -gt 1 ]]; then
  fail 'Usage: bash install.sh [vMAJOR.MINOR.PATCH]'
fi
if [[ ${1:-} == --help || ${1:-} == -h ]]; then
  printf 'Usage: bash install.sh [vMAJOR.MINOR.PATCH]\nInstalls the latest release by default.\n'
  exit 0
fi
[[ $(uname -s) == Darwin ]] || fail 'Mino currently supports macOS only.'
macos_version=$(sw_vers -productVersion)
macos_major=${macos_version%%.*}
if [[ ! $macos_major =~ ^[0-9]+$ ]] || (( macos_major < 13 )); then
  fail 'Mino requires macOS 13 or later.'
fi
case $(uname -m) in
  arm64) architecture=arm64 ;;
  x86_64) architecture=amd64 ;;
  *) fail 'Unsupported Mac architecture.' ;;
esac
[[ ${HOME:-} == /* && -d ${HOME:-} ]] || fail 'Your home directory must exist and have an absolute path.'
command -v gh >/dev/null 2>&1 || fail 'Install GitHub CLI (brew install gh), then run gh auth login.'

repository=qshine/mino
release_tag=${1:-latest}
if [[ $release_tag == latest ]]; then
  release_tag=$(gh api --hostname github.com "repos/$repository/releases/latest" --jq .tag_name) || fail 'Cannot find a release. Run gh auth login and check your repository access.'
fi
release_version=${release_tag#v}
version_pattern='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
[[ $release_version =~ $version_pattern ]] || fail 'Version must have the form v0.1.0 or 0.1.0.'
release_tag="v$release_version"
asset="mino_${release_version}_darwin_${architecture}.tar.gz"

install_root="$HOME/.mino"
bin_directory="$install_root/bin"
for directory in "$install_root" "$bin_directory"; do
  [[ ! -L $directory && ( ! -e $directory || -d $directory ) ]] || fail "$directory must be a directory, not a file or symbolic link."
done
[[ ! -L $bin_directory/mino && ( ! -e $bin_directory/mino || -f $bin_directory/mino ) ]] || fail 'The existing mino executable must be a regular file.'

download_directory=$(mktemp -d "${TMPDIR:-/tmp}/mino-install.XXXXXX")
staged_binary=''
cleanup() {
  rm -rf "$download_directory"
  if [[ -n $staged_binary ]]; then rm -f "$staged_binary"; fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

printf 'Downloading Mino %s for macOS %s...\n' "$release_version" "$architecture"
gh release download "$release_tag" --repo "github.com/$repository" \
  --pattern "$asset" --pattern checksums.txt --dir "$download_directory" || fail 'Download failed. Your existing installation has not been changed.'
expected_checksum=$(awk -v name="$asset" '$2 == name { print $1 }' "$download_directory/checksums.txt")
[[ $expected_checksum =~ ^[a-f0-9]{64}$ ]] || fail 'Missing or invalid SHA-256 checksum.'
actual_checksum=$(shasum -a 256 "$download_directory/$asset")
[[ ${actual_checksum%% *} == "$expected_checksum" ]] || fail 'Checksum mismatch. Your existing installation has not been changed.'

mkdir -p "$bin_directory"
chmod 700 "$install_root" "$bin_directory"
# Stage on the same filesystem so the final replacement is atomic.
staged_binary=$(mktemp "$bin_directory/.mino.XXXXXX")
tar -xOzf "$download_directory/$asset" mino > "$staged_binary"
chmod 755 "$staged_binary"
installed_version=$("$staged_binary" version) || fail 'The downloaded executable could not run on this Mac.'
[[ $installed_version == "mino $release_version" ]] || fail 'The executable version does not match the release.'
mv -f "$staged_binary" "$bin_directory/mino"
staged_binary=''

# Keep future terminals ready to use Mino. The bootstrap command also exports
# PATH for the current terminal, since a child script cannot change its parent.
profile=''
case ${SHELL:-/bin/zsh} in
  */zsh) profile="${ZDOTDIR:-$HOME}/.zshrc" ;;
  */bash) profile="$HOME/.bash_profile" ;;
esac
path_line="export PATH=\"\$HOME/.mino/bin:\$PATH\""
if [[ -n $profile ]]; then
  if [[ -L $profile || ( -e $profile && ! -f $profile ) ]]; then
    printf 'Add this line to your shell profile: %s\n' "$path_line"
  elif ! grep -Fqx "$path_line" "$profile" 2>/dev/null; then
    if ! printf '\n# Mino command\n%s\n' "$path_line" >> "$profile"; then
      printf 'Add this line to your shell profile: %s\n' "$path_line"
    fi
  fi
fi
printf 'Installed mino %s at %s/mino\n' "$release_version" "$bin_directory"
printf 'Open a new terminal, or run: %s\n' "$path_line"
printf 'Run mino to start. Settings are stored in ~/.mino/config.json.\n'
