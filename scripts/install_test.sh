#!/bin/sh
set -eu

script=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/install.sh
repo=$(sed -n 's/^REPO="\([^"]*\)"$/\1/p' "$script")
[ "$repo" = "r6c/ainovel-cli" ] || {
  echo "install.sh 使用了错误仓库: $repo" >&2
  exit 1
}
grep -F "https://raw.githubusercontent.com/$repo/" "$script" >/dev/null
grep -F "github.com/$repo/cmd/ainovel-cli@latest" "$(dirname -- "$script")/../README.md" >/dev/null
