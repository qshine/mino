#!/bin/bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

if ! command -v npm >/dev/null 2>&1; then
  printf 'Install Node.js 24 (including npm) before reviewing the book.\n' >&2
  exit 1
fi

if [[ ! -x node_modules/.bin/vitepress || ! -x node_modules/.bin/vite ]]; then
  printf 'Installing book dependencies...\n'
  npm ci --ignore-scripts
fi

printf 'Building the book from this checkout...\n'
npm run book:build

# Vite otherwise prefers a running Chromium browser on macOS over the OS default.
if [[ "$(uname -s)" == Darwin ]]; then
  export BROWSER=open
else
  unset BROWSER
fi

printf 'Opening the book in your default browser. Press Ctrl+C to stop the preview.\n'
exec npm run book:preview -- --open /mino/zh/chapters/01-terminal-chat.html
