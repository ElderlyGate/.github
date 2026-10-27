#!/usr/bin/env bash
# Tests next-version.sh against throwaway git repositories.
set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/next-version.sh"
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com
root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT
failed=0

repo() { cd "$(mktemp -d "$root/repo.XXXX")" && git init -q -b main; }
commit() { git commit -q --allow-empty -m "$1" ${2:+-m "$2"}; }
tag() { git tag -a "$1" -m "$1"; }

expect() {
  local name="$1" want="$2" got
  shift 2
  got=$(bash "$script" "$@")
  if [ "$got" = "$want" ]; then
    echo "ok   $name"
  else
    echo "FAIL $name: want '$want', got '$got'"
    failed=1
  fi
}

repo; commit "feat: start"; tag v1.2.3; commit "fix: a bug"
expect "fix only is a patch" v1.2.4

repo; commit "feat: start"; tag v1.2.3; commit "fix: a bug"; commit "feat: a thing"
expect "feat is a minor" v1.3.0

repo; commit "feat: start"; tag v1.2.3; commit "feat!: drop the old flag"
expect "feat! is a major" v2.0.0

repo; commit "feat: start"; tag v1.2.3; commit "fix: rename the flag" "BREAKING CHANGE: --old is gone"
expect "BREAKING CHANGE in the body is a major" v2.0.0

repo; commit "feat: start"; tag v1.2.3; commit "docs: words"; commit "chore: card"; commit "ci: cache"
expect "docs and chore alone make no release" ""

repo; commit "feat: start"; tag v1.2.3; commit "docs: words"; commit "fix: a bug"; commit "feat: a thing"; commit "chore: card"
expect "mixed takes the highest" v1.3.0

repo; commit "chore: init"; commit "feat: first"
expect "no tag starts at v0.1.0" v0.1.0

repo; commit "docs: readme"
expect "no tag and nothing to release" ""

repo; commit "feat: start"; tag v1.2.0; commit "feat: next"; tag v1.3.0-rc.1; commit "fix: a bug"
expect "rc tags are ignored" v1.3.0

repo; commit "feat: start"; tag v1.2.3; commit "feat(help): one layout"
expect "a scope is allowed" v1.3.0

repo; commit "feat: start"; tag v1.2.3; commit "fix(cli)!: exit codes"
expect "a scope with ! is a major" v2.0.0

repo; commit "feat: start"; tag v0.5.0; commit "feat!: new format"
expect "a breaking change in v0.x is a major" v1.0.0

repo; commit "feat: start"; tag v0.5.0; commit "feat: more"
expect "feat in v0.x is a minor" v0.6.0

repo; commit "feat: start"; tag v1.2.3; commit "refactor: split the parser"
expect "refactor alone makes no release" ""

repo; commit "feat: start"; tag v1.2.3; commit "perf: faster load"
expect "perf is a patch" v1.2.4

repo; commit "feat: start"; tag v1.9.0; tag v1.10.0; tag v4-before-watcher; commit "fix: a bug"
expect "tags sort as numbers and odd tags are ignored" v1.10.1

repo; commit "feat: start"; tag v1.2.3; commit "feat: a thing"; tag v1.3.0
expect "a tag on HEAD is the same release, run again" v1.3.0

repo; commit "Merge stuff"; commit "update readme"
expect "subjects that are not Conventional make no release" ""

repo; commit "feat: start"; tag v1.2.0; commit "feat: next"
expect "first release candidate" v1.3.0-rc.1 --rc
tag v1.3.0-rc.1
expect "the candidate on HEAD, run again" v1.3.0-rc.1 --rc
commit "fix: found in rc"
expect "second release candidate" v1.3.0-rc.2 --rc
expect "the release after candidates" v1.3.0

repo; commit "feat: start"; tag v1.2.3
commit "feat!: new format"; commit "feat: search"; commit "fix(cli): typo"; commit "docs: words"
want=$(printf '## Breaking changes\n\n- feat!: new format\n\n## Features\n\n- feat: search\n\n## Fixes\n\n- fix(cli): typo\n\n## Other changes\n\n- docs: words')
expect "notes are grouped by type" "$want" --notes

exit $failed
