# Runtime of the Security workflow: Go plus the tools scripts/security.sh runs,
# installed once instead of compiled on every run. The versions come from
# scripts/security-tools.env (passed as build arguments by the Security
# workflow, which builds the image when this file or the versions change).
# Build locally with:
#
#   docker build -f docker/security/Dockerfile \
#     $(sed -n 's/^\([A-Z_]*\)=\(.*\)$/--build-arg \1=\2/p' scripts/security-tools.env) \
#     -t lookup-security .
FROM golang:1.27

ARG GOVULNCHECK_VERSION
ARG GOSEC_VERSION
ARG GITLEAKS_VERSION

# GOBIN keeps the binaries out of GOPATH, which the workflow's module cache may
# shadow. GOTOOLCHAIN=auto lets go.mod ask for a newer Go than the image has.
ENV GOBIN=/usr/local/bin \
    GOTOOLCHAIN=auto

RUN test -n "$GOVULNCHECK_VERSION" && test -n "$GOSEC_VERSION" && test -n "$GITLEAKS_VERSION" \
 && go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} \
 && go install github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION} \
 && go install github.com/zricethezav/gitleaks/v8@${GITLEAKS_VERSION} \
 && go clean -cache -modcache
