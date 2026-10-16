#!/usr/bin/env bash
#
# Release the current branch (normally dev) into main.
#
#   1. Runs all tests (tests.sh); everything must be committed.
#   2. Checks that all commits are pushed (it never pushes) and opens a PR
#      into main (or reuses the open one).
#   3. Waits for the PR checks, then merges it with a merge commit.
#   4. Follows the Release Check workflow that the merge triggers on main.
#      release-please then opens the release PR, which the workflow merges.
#   5. Waits for the new GitHub release to appear and prints its version.
#
# With -v the workflow is additionally dispatched to force that version.
#
# Requires an authenticated GitHub CLI (gh).

set -euo pipefail

BASE=main
WORKFLOW=release-please.yml
TIMEOUT=900

usage() {
  cat <<EOF
Usage: $(basename "$0") [-v VERSION] [-t SECONDS] [-n] [-y]

  -v VERSION  force the release version (e.g. 1.6.0) by dispatching the
              release workflow after the merge
  -t SECONDS  how long to wait for the new release (default: $TIMEOUT)
  -n          only create the PR; do not wait, merge or start a release
  -y          do not ask for confirmation before merging
  -h          show this help
EOF
}

log() { printf '==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

VERSION=
PR_ONLY=false
ASSUME_YES=false

while getopts "v:t:nyh" opt; do
  case "$opt" in
    v) VERSION=$OPTARG ;;
    t) TIMEOUT=$OPTARG ;;
    n) PR_ONLY=true ;;
    y) ASSUME_YES=true ;;
    h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

[[ $TIMEOUT =~ ^[0-9]+$ ]] || die "invalid timeout '$TIMEOUT' (expected seconds)"

if [ -n "$VERSION" ]; then
  VERSION=${VERSION#v}
  [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "invalid version '$VERSION' (expected e.g. 1.2.3)"
fi

# poll_run prints the id of the first workflow run matching the given gh run
# list filters, waiting up to about a minute for it to appear.
poll_run() {
  local id i
  for i in $(seq 1 20); do
    id=$(gh run list --workflow "$WORKFLOW" "$@" --limit 1 --json databaseId --jq '.[0].databaseId // empty')
    if [ -n "$id" ]; then
      echo "$id"
      return 0
    fi
    sleep 3
  done
  return 1
}

# latest_release prints the tag of the latest GitHub release, or nothing if there is none.
latest_release() {
  gh release view --json tagName --jq .tagName 2>/dev/null || true
}

# wait_for_release waits until a release other than $1 is published (and, when
# a version was forced, until that version is published) and prints its tag.
wait_for_release() {
  local before=$1 deadline=$((SECONDS + TIMEOUT)) tag
  while [ "$SECONDS" -lt "$deadline" ]; do
    tag=$(latest_release)
    if [ -n "$tag" ] && [ "$tag" != "$before" ] && { [ -z "$VERSION" ] || [ "$tag" = "v$VERSION" ]; }; then
      echo "$tag"
      return 0
    fi
    sleep 10
  done
  return 1
}

# wait_for_checks blocks until the PR checks pass. A PR without any checks is fine.
wait_for_checks() {
  local out
  sleep 5
  if out=$(gh pr checks "$1" 2>&1); then
    return 0
  elif grep -q "no checks reported" <<<"$out"; then
    log "no checks reported"
    return 0
  fi
  gh pr checks "$1" --watch --interval 10
}

command -v gh >/dev/null || die "gh (GitHub CLI) is required"
gh auth status >/dev/null 2>&1 || die "gh is not authenticated; run 'gh auth login'"

HEAD_BRANCH=$(git branch --show-current)
[ -n "$HEAD_BRANCH" ] || die "detached HEAD; check out the branch to release"
[ "$HEAD_BRANCH" != "$BASE" ] || die "already on $BASE; check out the branch to release"

[ -z "$(git status --porcelain)" ] || die "uncommitted changes; commit or stash them first"

git fetch --quiet origin
if [ -n "$VERSION" ] && [ -n "$(git ls-remote --tags origin "refs/tags/v$VERSION")" ]; then
  die "tag v$VERSION already exists"
fi

RELEASE_BEFORE=$(latest_release)

AHEAD=$(git rev-list --count "origin/$BASE..HEAD")
[ "$AHEAD" -gt 0 ] || die "$HEAD_BRANCH has no commits that are not already in $BASE"

log "Running all tests"
"$(dirname "$0")/tests.sh"
[ -z "$(git status --porcelain)" ] || { git status --short >&2; die "the tests changed files; commit them first"; }

# The script never pushes; every commit must already be on the remote.
if ! git rev-parse --verify --quiet "origin/$HEAD_BRANCH" >/dev/null; then
  die "$HEAD_BRANCH does not exist on origin; push it first"
fi
UNPUSHED=$(git rev-list --count "origin/$HEAD_BRANCH..HEAD")
[ "$UNPUSHED" -eq 0 ] || die "$UNPUSHED commit(s) of $HEAD_BRANCH are not pushed; push them first"

PR=$(gh pr list --head "$HEAD_BRANCH" --base "$BASE" --state open --json number --jq '.[0].number // empty')
if [ -n "$PR" ]; then
  log "Using existing PR #$PR"
else
  BODY=$(git log --no-merges --format='- %s (%h)' "origin/$BASE..HEAD")
  log "Creating PR $HEAD_BRANCH -> $BASE ($AHEAD commits)"
  gh pr create --base "$BASE" --head "$HEAD_BRANCH" \
    --title "Merge $HEAD_BRANCH into $BASE" \
    --body "$BODY"
  PR=$(gh pr list --head "$HEAD_BRANCH" --base "$BASE" --state open --json number --jq '.[0].number')
fi
PR_URL=$(gh pr view "$PR" --json url --jq .url)

if $PR_ONLY; then
  log "PR ready: $PR_URL"
  exit 0
fi

log "Waiting for checks on PR #$PR"
wait_for_checks "$PR"

if ! $ASSUME_YES; then
  read -r -p "Merge $PR_URL into $BASE and start the release? [y/N] " answer
  [[ $answer =~ ^[Yy]$ ]] || die "aborted; PR #$PR is still open"
fi

log "Merging PR #$PR"
gh pr merge "$PR" --merge
MERGE_SHA=$(gh pr view "$PR" --json mergeCommit --jq .mergeCommit.oid)

# The merge to main triggers the workflow, unless no Go or module files changed.
log "Waiting for the release workflow triggered by the merge"
if RUN=$(poll_run --event push --commit "$MERGE_SHA"); then
  gh run watch "$RUN" --exit-status
elif [ -z "$VERSION" ]; then
  die "no workflow run was triggered by the merge (no Go or module files changed?); rerun with -v VERSION to dispatch one"
else
  log "No run triggered by the merge"
fi

if [ -n "$VERSION" ]; then
  log "Dispatching release workflow for $VERSION"
  STARTED=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  gh workflow run "$WORKFLOW" --ref "$BASE" -f "version=$VERSION"
  RUN=$(poll_run --event workflow_dispatch --created ">=$STARTED") || die "dispatched run did not appear"
  gh run watch "$RUN" --exit-status
fi

RELEASE_PR=$(gh pr list --base "$BASE" --head "release-please--branches--$BASE" --state open --json url --jq '.[0].url // empty')
if [ -n "$RELEASE_PR" ]; then
  log "Release PR: $RELEASE_PR"
else
  log "No open release PR; it was already merged by the workflow or there is nothing to release"
fi

log "Waiting for the new release (up to ${TIMEOUT}s, latest is ${RELEASE_BEFORE:-none})"
if TAG=$(wait_for_release "$RELEASE_BEFORE"); then
  log "Released $TAG: $(gh release view "$TAG" --json url --jq .url)"
else
  die "no new release after ${TIMEOUT}s; check ${RELEASE_PR:-the $WORKFLOW runs}"
fi
log "Done"
