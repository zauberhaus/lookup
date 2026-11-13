#!/bin/sh
# cspell:words govulncheck gosec securego
# Run go vet (suspicious constructs), govulncheck (known vulnerabilities), gosec (static security analysis)
# and gitleaks (hardcoded secrets in the working tree and git history).
set -u

status=0

# Tool versions are pinned in security-tools.env so a run does not depend on
# what "latest" resolves to (CI once picked an old govulncheck whose
# golang.org/x/tools no longer builds with new Go).
. "$(dirname "$0")/security-tools.env"

# run_tool uses the installed binary (the Security workflow image has them) and
# otherwise builds the pinned version with "go run".
run_tool() {
  bin=$1 pkg=$2
  shift 2
  if command -v "$bin" >/dev/null 2>&1; then
    "$bin" "$@"
  else
    go run "$pkg" "$@"
  fi
}

cd "$(dirname "$0")/.."

echo "==> go vet"
go vet ./... || status=1

echo "==> govulncheck"
run_tool govulncheck golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION ./... || status=1

echo "==> gosec"
run_tool gosec github.com/securego/gosec/v2/cmd/gosec@$GOSEC_VERSION ./... || status=1

echo "==> gitleaks (working tree)"
run_tool gitleaks github.com/zricethezav/gitleaks/v8@$GITLEAKS_VERSION dir --no-banner --redact . || status=1

echo "==> gitleaks (git history)"
run_tool gitleaks github.com/zricethezav/gitleaks/v8@$GITLEAKS_VERSION git --no-banner --redact . || status=1

if [ $status -ne 0 ]; then
  echo "FAILED!"
  exit 1
fi
echo "OK"
