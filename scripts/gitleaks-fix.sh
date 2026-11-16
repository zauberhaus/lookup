#!/usr/bin/env bash
# cspell:words gitleaks zricethezav gitleaksignore
# List the gitleaks findings (working tree and git history) and mark the false
# positives as accepted in .gitleaksignore, which gitleaks reads on its own.
#
# A real secret cannot be fixed by a script: revoke or rotate it first, then
# remove it from the code. A secret in the history stays reachable until the
# history is rewritten (git filter-repo, then a force push), so this script
# never touches history and never ignores anything without asking.
#
# Usage: scripts/gitleaks-fix.sh [-y] [-n]
#   -y  accept every finding without asking (only after reviewing the list)
#   -n  list the findings only, change nothing

set -eu

ASSUME_YES=false
LIST_ONLY=false

while getopts "yn" opt; do
  case $opt in
    y) ASSUME_YES=true ;;
    n) LIST_ONLY=true ;;
    *) echo "usage: $0 [-y] [-n]" >&2; exit 2 ;;
  esac
done

cd "$(dirname "$0")/.."

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

# Same as security.sh: the installed binary, else the latest version via go run.
run_gitleaks() {
  if command -v gitleaks >/dev/null 2>&1; then
    gitleaks "$@"
  else
    go run "github.com/zricethezav/gitleaks/v8@latest" "$@"
  fi
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Exit code 1 only means "leaks found"; the reports are what matters here.
run_gitleaks dir --no-banner --redact --exit-code 0 -f json -r "$tmp/dir.json" . >"$tmp/run.log" 2>&1 || true
run_gitleaks git --no-banner --redact --exit-code 0 -f json -r "$tmp/git.json" . >>"$tmp/run.log" 2>&1 || true

# One tab separated line per finding: fingerprint, rule, location, commit.
jq -r -s '
  add // [] | unique_by(.Fingerprint)[]
  | [.Fingerprint, .RuleID, "\(.File):\(.StartLine)", (.Commit // "" | .[0:8])]
  | @tsv' "$tmp/dir.json" "$tmp/git.json" >"$tmp/findings.tsv"

count=$(wc -l <"$tmp/findings.tsv" | tr -d ' ')
if [ "$count" -eq 0 ]; then
  echo "No gitleaks findings."
  exit 0
fi

echo "$count finding(s):"
awk -F'\t' '{ printf "  %-28s %-40s %s\n", $2, $3, ($4 == "" ? "working tree" : "commit " $4) }' "$tmp/findings.tsv"
echo
$LIST_ONLY && exit 0

touch .gitleaksignore
added=0
while IFS=$'\t' read -r fingerprint rule location commit <&3; do
  grep -qxF "$fingerprint" .gitleaksignore && continue
  if ! $ASSUME_YES; then
    printf 'Accept %s at %s as a false positive? [y/N] ' "$rule" "$location"
    read -r answer
    [ "$answer" = y ] || [ "$answer" = Y ] || continue
  fi
  printf '# %s at %s (accepted %s)\n%s\n' "$rule" "$location" "$(date +%F)" "$fingerprint" >>.gitleaksignore
  added=$((added + 1))
done 3<"$tmp/findings.tsv"

echo "Added $added fingerprint(s) to .gitleaksignore."
echo "Real secrets: rotate them, remove them from the code, and rewrite the history if they were committed."
