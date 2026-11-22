#!/bin/sh
# Stop hook: blocks Claude from finishing while code, docs, or example decks are out of sync.
input=$(cat)
case "$input" in *'"stop_hook_active":true'*|*'"stop_hook_active": true'*) exit 0 ;; esac

cd "${CLAUDE_PROJECT_DIR:-$(dirname "$0")/../..}" || exit 0
stamp=.git/drift-ok
if [ -f "$stamp" ] && [ -z "$(find . -path ./bin -prune -o \( -name '*.go' -o -name '*.md' -o -name go.mod \) -newer "$stamp" -print | head -1)" ]; then
	exit 0
fi

out=$( { gofmt -l . | sed 's/^/gofmt needed: /'; go vet ./... && go test -count=1 ./...; } 2>&1 )
status=$?
if [ $status -ne 0 ] || echo "$out" | grep -q '^gofmt needed'; then
	{
		echo "Drift check failed (gofmt / go vet / go test, incl. internal/docscheck which keeps README, docs/ and examples/ in sync with the code). Fix before finishing:"
		echo "$out" | grep -v '^ok \|no test files' | tail -60
	} >&2
	exit 2
fi
mkdir -p .git 2>/dev/null && touch "$stamp"
exit 0
