#!/usr/bin/env bash
# The image a release pushes, and its tags, one reference per line: <name>:<version> as the archives name it (no v), and <name>:latest unless the tag is a candidate (V-8).
# Usage: image-tags.sh <tag> <owner/repo> [<name>]. <name> is the image input; empty, it is ghcr.io/<owner>/<repo>. Either is lowercased, as registries want.
set -euo pipefail

tag=${1:?the tag, vX.Y.Z or vX.Y.Z-rc.N}
repository=${2:?the repository, owner/repo}
name=${3:-}

if ! [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]]; then
  echo "image-tags: $tag is not a version vX.Y.Z or vX.Y.Z-rc.N (V-4, V-8)" >&2
  exit 1
fi
[ -n "$name" ] || name="ghcr.io/$repository"
name=$(printf %s "$name" | tr "[:upper:]" "[:lower:]")

echo "$name:${tag#v}"
case "$tag" in
  *-rc.*) ;;
  *) echo "$name:latest" ;;
esac
