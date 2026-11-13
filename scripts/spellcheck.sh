#!/bin/sh
set -e

cd "$(dirname "$0")"

cd ..

# package.json exists only to pin the cspell tooling below; it's
# gitignored, so recreate it on demand rather than committing it.
if [ ! -f package.json ]; then
	cat > package.json << 'EOF'
{
  "private": true,
  "devDependencies": {
    "cspell": "10.4.1",
    "@cspell/dict-en-au": "1.1.31",
    "@cspell/dict-en-gb": "5.0.38"
  }
}
EOF
fi

if [ -f package-lock.json ]; then
	npm ci --ignore-scripts
else
	npm install --ignore-scripts
fi

npx cspell -c cspell.config.yaml .