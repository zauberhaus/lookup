#!/usr/bin/env bash
# Print a markdown report of the available updates: a newer Go release, how
# old the dependencies in go.mod are, and the dependencies that have a newer
# version. Exits with 1 when go.mod was last changed more than FAIL_LIMIT days
# ago.
#
# Requires go, git, curl and jq.
set -euo pipefail

cd "$(dirname "$0")/.."

NOTICE_LIMIT=7
INFO_LIMIT=45
WARN_LIMIT=75
FAIL_LIMIT=90

# Modules that are not reported: our own (they may be private) and ones that
# only publish pseudo-versions.
IGNORE='^(github\.com/zauberhaus/|golang\.org/x/exp($|/)|google\.golang\.org/genproto($|/))'

for tool in go git curl jq; do
    command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 2; }
done

# release prints the major.minor part of a Go version: go1.26.3 -> 1.26
release() {
    local version="${1#go}"
    echo "$version" | sed -E 's/^([0-9]+\.[0-9]+).*/\1/'
}

# plural prints "1 day" or "N days"
plural() {
    if [[ $1 -eq 1 ]]; then echo "1 day"; else echo "$1 days"; fi
}

# Newest Go release, compared with the version go.mod asks for.
LATEST_GO="$(curl -fsS "https://go.dev/VERSION?m=text" | head -1 || true)"
MOD_GO="$(go list -m -f '{{.GoVersion}}')"
NEW_GO_VERSION=
if [[ "$LATEST_GO" == go* && -n "$MOD_GO" && "$(release "$LATEST_GO")" != "$(release "$MOD_GO")" ]]; then
    NEW_GO_VERSION="${LATEST_GO#go}"
fi

# Age of go.mod in days, by its last commit.
UPDATED="$(git log -1 --format=%ct -- go.mod)"
if [[ -z "$UPDATED" ]]; then
    UPDATED="$(date +%s)" # not committed yet
fi
AGE=$(( ($(date +%s) - UPDATED) / 86400 ))

if (( AGE > FAIL_LIMIT )); then
    echo "go.mod is too old ($AGE > $FAIL_LIMIT days)" >&2
    exit 1
fi

RESULT="# Available Updates"

if [[ -n "$NEW_GO_VERSION" ]]; then
    RESULT+=$'\n\n> [!important]\n> A new go version '"$NEW_GO_VERSION"' is available.'
fi

if (( AGE > WARN_LIMIT )); then
    RESULT+=$'\n\n> [!warning]\n> Last update was '"$(plural "$AGE")"' ago\n> Please update the dependencies in the go.mod file.\n> This check will fail after '"$FAIL_LIMIT"' days.'
elif (( AGE > INFO_LIMIT )); then
    RESULT+=$'\n\n> [!important]\n> Last update was '"$(plural "$AGE")"$' ago\n> Please update the dependencies in the go.mod file.'
elif (( AGE > NOTICE_LIMIT )); then
    RESULT+=$'\n\n> [!note]\n> Last update was '"$(plural "$AGE")"' ago'
fi

# modules prints the modules that provide a package this project builds
# (without -test) or builds and tests (with -test). The full module graph also
# holds modules only our dependencies' tests need, which go get never moves.
modules() {
    go list -deps "$@" -f '{{with .Module}}{{.Path}}{{end}}' ./... | sort -u | grep -v '^$' | grep -Ev "$IGNORE" || true
}

# table prints the markdown table rows of the modules that have a newer version.
table() {
    local mods="$1"
    [[ -n "$mods" ]] || return 0
    # shellcheck disable=SC2086 # the module list must expand into arguments
    go list -m -u -json $mods | jq -r 'select(.Update != null) | "|\(.Path)|\(.Version)|\(.Update.Version)|"'
}

# add_table appends a heading and a table to the result.
add_table() {
    local title="$1" rows="$2"
    RESULT+=$'\n\n'"## $title"$'\n\n| Package | Current | Update |\n|---------|---------|--------|'
    if [[ -n "$rows" ]]; then
        RESULT+=$'\n'"$rows"
    fi
}

CODE_MODULES="$(modules)"
# modules only the tests import: in the test list, but not in the code list
TEST_MODULES="$(comm -13 <(echo "$CODE_MODULES") <(modules -test))"

add_table "Dependencies" "$(table "$CODE_MODULES")"
add_table "Test dependencies" "$(table "$TEST_MODULES")"

printf '%s\n' "$RESULT"
