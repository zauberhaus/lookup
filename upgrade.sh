#!/usr/bin/env bash
# Upgrade the Go module dependencies in go.mod, tidy, and verify the result
# still builds and passes the tests.
# Usage: upgrade.sh [-c|--check] [-p|--patch] [-n|--no-test] [-h|--help]
#
#   -c, --check     list the available updates and exit without changing anything
#   -p, --patch     only upgrade to the latest patch release of each dependency
#   -n, --no-test   skip the test run after upgrading (build and vet still run)
set -euo pipefail

CHECK=0
PATCH=0
RUN_TESTS=1

usage() {
    sed -n '2,8p' "$0" | sed 's/^# \{0,1\}//'
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -c|--check)   CHECK=1 ;;
        -p|--patch)   PATCH=1 ;;
        -n|--no-test) RUN_TESTS=0 ;;
        -h|--help)    usage; exit 0 ;;
        *)            echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
    esac
    shift
done

cd "$(dirname "$0")"

# modules lists every module that provides a package this project builds or
# tests. The full module graph ('go list -m all') also contains modules that
# only our dependencies' own tests need — go get never moves those, so
# reporting them would promise upgrades that never happen.
modules() {
    go list -deps -test -f '{{with .Module}}{{.Path}}{{end}}' ./... | sort -u | grep -v '^$'
}

# outdated prints "<module> <version> -> <new version>" for each module with a
# newer release. The Update field is only set when one exists, so an up to date
# dependency prints nothing.
outdated() {
    local mods
    mods="$(modules)"

    # shellcheck disable=SC2086 # the module list must expand into arguments
    go list -u -m -f '{{if .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' $mods |
        grep -v '^$' || true
}

versions() {
    go list -m -f '{{.Path}} {{.Version}}' all
}

# patch_version prints the newest release of a module that keeps its current
# major and minor version, or nothing if there is none. 'go get -u=patch' is
# the obvious way to do this, but current toolchains fail to resolve module
# paths for it ("can't query version patch of module ..."), so the patch
# release is looked up and pinned explicitly.
patch_version() {
    local mod="$1" prefix
    prefix="${2%.*}."
    prefix="${prefix//./\\.}"

    # grep exits non-zero when the module has no release on the current minor
    # version, which is a normal answer here, not a failure.
    go list -m -versions "$mod" | tr ' ' '\n' | grep "^${prefix}" | sort -V | tail -1 || true
}

if [[ $CHECK -eq 1 ]]; then
    echo "==> checking for updates"
    updates="$(outdated)"

    if [[ -z "$updates" ]]; then
        echo "all dependencies are up to date"
    else
        echo "$updates"
    fi

    exit 0
fi

# Only used for the summary at the end, so a stale or incomplete go.sum must
# not abort the upgrade that would repair it.
before="$(versions 2>/dev/null || true)"

if [[ $PATCH -eq 1 ]]; then
    echo "==> upgrading dependencies to their latest patch release"

    targets=""
    while read -r mod current _; do
        [[ -z "$mod" ]] && continue

        patch="$(patch_version "$mod" "$current")"
        if [[ -n "$patch" && "$patch" != "$current" ]]; then
            targets="$targets $mod@$patch"
        fi
    done <<< "$(outdated)"

    if [[ -z "$targets" ]]; then
        echo "no patch releases available"
    else
        # shellcheck disable=SC2086 # the targets must expand into arguments
        go get $targets
    fi
else
    echo "==> upgrading dependencies"
    go get -u ./...
fi

echo "==> tidying go.mod"
go mod tidy

echo "==> building"
go build ./...

echo "==> vetting"
go vet ./...

if [[ $RUN_TESTS -eq 1 ]]; then
    echo "==> testing"
    go test ./...
fi

if [[ -n "$before" ]]; then
    echo "==> changed modules"
    # Compare the module list from before the upgrade with the current one and
    # report every version that moved, was added, or was dropped.
    diff <(echo "$before") <(versions) |
        awk '/^</ {print "  - " $2 " " $3} /^>/ {print "  + " $2 " " $3}' || true
fi

echo
echo "done — review with 'git diff go.mod go.sum', undo with 'git checkout go.mod go.sum'"
