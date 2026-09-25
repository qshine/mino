#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."

release_tag=${1:-}
chapter_pattern='^chapter-(0[1-9]|[1-9][0-9])(\.([1-9][0-9]*))?$'
if [[ $# != 1 || ! $release_tag =~ $chapter_pattern ]]; then
  printf 'Usage: bash scripts/package.sh chapter-NN[.PATCH]\n' >&2
  exit 1
fi
chapter=$((10#${BASH_REMATCH[1]}))
patch=${BASH_REMATCH[3]:-0}
release_version="0.$chapter.$patch"
mkdir -p dist
awk -v heading="## [$release_version]" '
  index($0, heading) == 1 { found = 1; next }
  found && /^## / { exit }
  found { print }
' CHANGELOG.md > dist/release-notes.md
if [[ ! -s dist/release-notes.md ]]; then
  printf 'Add release %s to CHANGELOG.md before packaging.\n' "$release_version" >&2
  exit 1
fi

staging_directory=$(mktemp -d "${TMPDIR:-/tmp}/mino-package.XXXXXX")
trap 'rm -rf "$staging_directory"' EXIT
cp LICENSE THIRD_PARTY_NOTICES.txt "$staging_directory/"
native_arch=$(go env GOARCH)
for architecture in arm64 amd64; do
  CGO_ENABLED=0 GOOS=darwin GOARCH="$architecture" go build -trimpath \
    -ldflags "-s -w -X main.version=$release_version" -o "$staging_directory/mino" ./cmd/mino
  if [[ $(uname -s) == Darwin && $architecture == "$native_arch" ]]; then
    [[ $("$staging_directory/mino" version) == "mino $release_version" ]]
  fi
  COPYFILE_DISABLE=1 tar -czf "dist/mino_${release_version}_darwin_${architecture}.tar.gz" \
    -C "$staging_directory" mino LICENSE THIRD_PARTY_NOTICES.txt
done
(
  cd dist
  shasum -a 256 "mino_${release_version}_darwin_arm64.tar.gz" \
    "mino_${release_version}_darwin_amd64.tar.gz" > checksums.txt
)
printf 'Built macOS release %s in dist/.\n' "$release_tag"
