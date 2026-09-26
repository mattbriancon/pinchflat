#!/usr/bin/env bash
# Builds the CSS/JS served from internal/web/static/assets (committed, so a
# plain `go build` needs no Node or Tailwind). Replaces esbuild + the
# tailwind mix task. Usage: web/assets/build.sh [--minify]
set -euo pipefail
cd "$(dirname "$0")"
out=../../internal/web/static/assets
mkdir -p "$out"
TAILWIND=${TAILWIND:-./tailwindcss}
if [ ! -x "$TAILWIND" ]; then
  arch=$(uname -m); case "$arch" in x86_64) arch=x64;; aarch64|arm64) arch=arm64;; esac
  curl -sSL -o tailwindcss "https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.3/tailwindcss-linux-$arch"
  chmod +x tailwindcss
fi
"$TAILWIND" -c tailwind.config.js -i css/app.css -o "$out/app.css" ${1:-}
# No bundler: scripts are concatenated in dependency order. Alpine goes last
# and starts itself.
cat vendor/topbar.js vendor/htmx.min.js js/tabs.js js/alpine_helpers.js js/app.js > "$out/app.js"
cp vendor/alpine.min.js "$out/alpine.min.js"
echo "built $out"
