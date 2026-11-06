#!/usr/bin/env bash
# Fails a change that alters code without changing the design (G-PLAN-9).
#   design-check.sh <base> [labels]
# <base> is the revision to compare HEAD with; labels are the pull request's labels, comma-separated.
# A change passes when it carries the label no-design-change, when it changes no code, or when it also
# changes a file under .designer/ (the board's own docker-compose.yml, .env and .gitignore do not count).
# Code is every file except the design, cards, Markdown, docs/, constitution/, .github/ and the licence.
set -euo pipefail

base="${1:?usage: design-check.sh <base> [labels]}"
labels=",${2:-},"

if [[ $labels == *",no-design-change,"* ]]; then
  echo "The label no-design-change says this change leaves the architecture as it is."
  exit 0
fi

code=() design=()
while IFS= read -r file; do
  case "$file" in
    .designer/docker-compose.yml | .designer/.env | .designer/.gitignore) ;;
    .designer/*) design+=("$file") ;;
    .kanban/* | *.md | docs/* | constitution/* | .github/* | LICENSE* | .gitignore) ;;
    *) code+=("$file") ;;
  esac
done < <(git diff --name-only "$base"...HEAD)

if [ ${#code[@]} -eq 0 ]; then
  echo "No code changed: nothing to check."
  exit 0
fi
if [ ${#design[@]} -gt 0 ]; then
  echo "Code and design changed together (${#code[@]} code files, ${#design[@]} design files)."
  exit 0
fi

echo "Code changed, the design did not (G-PLAN-9):"
printf '  %s\n' "${code[@]}"
echo "Update the design under .designer/ in this pull request, or label it no-design-change when the architecture stays as it is."
exit 1
