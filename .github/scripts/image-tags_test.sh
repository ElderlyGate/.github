#!/usr/bin/env bash
# Tests image-tags.sh.
set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/image-tags.sh"
failed=0

expect() {
  local name="$1" want="$2" got
  shift 2
  got=$(bash "$script" "$@" 2>&1) || got="exit $?: $got"
  if [ "$got" = "$want" ]; then
    echo "ok   $name"
  else
    echo "FAIL $name: want '$want', got '$got'"
    failed=1
  fi
}

expect "a release is its version and latest" "ghcr.io/elderlygate/e-designer:7.0.1
ghcr.io/elderlygate/e-designer:latest" v7.0.1 ElderlyGate/e-designer

expect "a candidate is never latest" "ghcr.io/elderlygate/e-designer:7.1.0-rc.2" v7.1.0-rc.2 ElderlyGate/e-designer

expect "a name given wins over the repository" "registry.example.com/team/board:1.0.0
registry.example.com/team/board:latest" v1.0.0 ElderlyGate/e-designer registry.example.com/team/board

expect "an empty name is the repository's" "ghcr.io/elderlygate/kanban-app:0.1.0
ghcr.io/elderlygate/kanban-app:latest" v0.1.0 ElderlyGate/kanban-app ""

expect "a tag that is no version is refused" "exit 1: image-tags: 7.0.1 is not a version vX.Y.Z or vX.Y.Z-rc.N (V-4, V-8)" 7.0.1 ElderlyGate/e-designer

exit "$failed"
