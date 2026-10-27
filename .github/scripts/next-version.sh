#!/usr/bin/env bash
# Prints the tag HEAD releases as (V-3), or nothing when no release is due.
#   next-version.sh          the next release, vX.Y.Z
#   next-version.sh --rc     the next release candidate, vX.Y.Z-rc.N (V-8)
#   next-version.sh --notes  the release notes: commit subjects grouped by type
set -euo pipefail

mode="${1:-}"
stable='^v[0-9]+\.[0-9]+\.[0-9]+$'
head=$(git rev-parse HEAD)

# The highest stable tag behind HEAD; a tag on HEAD itself is this release, run again.
last_tag() {
  git tag --merged HEAD --list 'v*' | { grep -E "$stable" || true; } | while read -r t; do
    [ "$(git rev-list -n1 "$t")" = "$head" ] || echo "$t"
  done | sort -t. -k1.2,1n -k2,2n -k3,3n | tail -n1
}

# Bump level of one commit: 3 major, 2 minor, 1 patch, 0 none.
level() {
  local subject="$1" body="$2" re='^([A-Za-z]+)(\([^)]*\))?(!)?: '
  if printf '%s\n' "$body" | grep -Eq '^BREAKING[ -]CHANGE: '; then echo 3; return; fi
  [[ $subject =~ $re ]] || { echo 0; return; }
  [ -z "${BASH_REMATCH[3]}" ] || { echo 3; return; }
  case "$(printf '%s' "${BASH_REMATCH[1]}" | tr '[:upper:]' '[:lower:]')" in
    feat) echo 2 ;;
    fix | perf | revert) echo 1 ;;
    *) echo 0 ;;
  esac
}

last=$(last_tag)
range=HEAD
[ -z "$last" ] || range="$last..HEAD"
commits=$(git rev-list --no-merges "$range")

if [ "$mode" = --notes ]; then
  breaking='' feat='' fix='' other=''
  for c in $commits; do
    subject=$(git log -1 --format=%s "$c")
    case "$(level "$subject" "$(git log -1 --format=%b "$c")")$subject" in
      3*) breaking+="- $subject"$'\n' ;;
      2*) feat+="- $subject"$'\n' ;;
      1fix* | 1Fix*) fix+="- $subject"$'\n' ;;
      *) other+="- $subject"$'\n' ;;
    esac
  done
  for section in "Breaking changes:$breaking" "Features:$feat" "Fixes:$fix" "Other changes:$other"; do
    [ -n "${section#*:}" ] && printf '## %s\n\n%s\n' "${section%%:*}" "${section#*:}"
  done
  exit 0
fi

bump=0
for c in $commits; do
  l=$(level "$(git log -1 --format=%s "$c")" "$(git log -1 --format=%b "$c")")
  [ "$l" -le "$bump" ] || bump=$l
done
[ "$bump" -gt 0 ] || exit 0

if [ -z "$last" ]; then
  next=0.1.0
else
  IFS=. read -r major minor patch <<<"${last#v}"
  case $bump in
    3) next="$((major + 1)).0.0" ;;
    2) next="$major.$((minor + 1)).0" ;;
    1) next="$major.$minor.$((patch + 1))" ;;
  esac
fi

if [ "$mode" != --rc ]; then
  echo "v$next"
  exit 0
fi

# A candidate already on HEAD is this run again; otherwise the next free number.
rcs=$(git tag --list "v$next-rc.*" | grep -E "^v${next//./\\.}-rc\.[0-9]+$" || true)
for t in $rcs; do
  [ "$(git rev-list -n1 "$t")" != "$head" ] || { echo "$t"; exit 0; }
done
n=0
for t in $rcs; do
  [ "${t##*.}" -le "$n" ] || n=${t##*.}
done
echo "v$next-rc.$((n + 1))"
