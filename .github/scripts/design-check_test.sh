#!/usr/bin/env bash
# Tests design-check.sh against throwaway git repositories.
set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/design-check.sh"
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com
root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT
failed=0

# repo makes a repository with one commit and prints nothing; change commits the files it names on top.
repo() { cd "$(mktemp -d "$root/repo.XXXX")" && git init -q -b main && touch keep && git add . && git commit -q -m start; }
change() {
  local f
  for f in "$@"; do mkdir -p "$(dirname "$f")" && echo x >> "$f"; done
  git add . && git commit -q -m change
}

expect() {
  local name="$1" want="$2" got=0
  shift 2
  bash "$script" HEAD~1 "$@" > /dev/null || got=$?
  if [ "$got" = "$want" ]; then
    echo "ok   $name"
  else
    echo "FAIL $name: want exit $want, got $got"
    failed=1
  fi
}

repo; change main.go;                                expect "code without design fails" 1
repo; change main.go .designer/kanban/kanban.yml;    expect "code with design passes" 0
repo; change main.go;                                expect "the label passes" 0 "bug,no-design-change"
repo; change main.go;                                expect "another label does not" 1 "bug"
repo; change README.md docs/a.md .kanban/cards/c.yml; expect "docs and cards only pass" 0
repo; change constitution/tech-stack.md .github/workflows/ci.yml; expect "constitution and workflows pass" 0
repo; change main.go .designer/docker-compose.yml .designer/.env; expect "the board's own files are not the design" 1
repo; change .designer/kanban/layout.yml;            expect "a design change alone passes" 0

exit $failed
