#!/bin/sh
# Prepare an Orca worktree for TuneTrail work (ISSUE-129). Runs from the new worktree.
# .agents/ (personas and skills) and backlog/docs/estudos-locais/ are kept out of git on
# purpose, so a fresh worktree would not have them: link them from the main checkout.
set -eu

here=$(git rev-parse --show-toplevel)
main=$(git worktree list --porcelain | awk '/^worktree /{print $2; exit}')

if [ "$here" != "$main" ]; then
	for path in .agents backlog/docs/estudos-locais; do
		if [ -e "$main/$path" ] && [ ! -e "$here/$path" ]; then
			mkdir -p "$(dirname "$here/$path")"
			ln -s "$main/$path" "$here/$path"
			echo "linked $path from the main checkout"
		fi
	done
fi

# Report missing tools instead of failing: an agent can still read and plan without them.
for tool in go node python3 backlog; do
	command -v "$tool" >/dev/null 2>&1 || echo "warning: $tool not found on PATH" >&2
done
echo "TuneTrail worktree ready: $here"
