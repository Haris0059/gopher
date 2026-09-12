#!/usr/bin/env bash
# PreToolUse(Bash) hook: gate `git commit` in the gopher repo.
#   1. The gopher-commit skill must have run (it runs `git diff --staged`,
#      which the PostToolUse hook records as .git/.claude_commit_marker).
#   2. The staged Version in all four files must equal
#      MAJOR.MINOR.<zero-padded Haris0059 commit count + 1>.
# See CLAUDE.md "Versioning" and .claude/skills/gopher-commit/SKILL.md.

cmd=$(jq -r '.tool_input.command // empty')

# Only real git commit invocations: at start or after ; & | ( — not the words inside a string.
printf '%s' "$cmd" | grep -qE '(^|[;&|(]\s*)git(\s+-C\s+\S+)?\s+commit\b' || exit 0

deny() {
	jq -n --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
	exit 0
}

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

# 1. Skill marker (valid for 15 minutes, consumed on a successful check).
marker=.git/.claude_commit_marker
if [ ! -f "$marker" ]; then
	deny "Use the gopher-commit skill (Skill tool, skill: gopher-commit) before committing. It runs git diff --staged, bumps the version, and generates the message; run it even if the user just said commit it."
fi
age=$(( $(date +%s) - $(stat -c %Y "$marker" 2>/dev/null || stat -f %m "$marker") ))
if [ "$age" -ge 900 ]; then
	rm -f "$marker"
	deny "The gopher-commit skill marker expired. Re-run the gopher-commit skill (Skill tool, skill: gopher-commit), then commit."
fi

# 2. Version check — only for commits authored by Haris0059.
if [ "$(git config user.name)" = "Haris0059" ]; then
	files="cmd/gopher/main.go internal/cli/repl.go pkg/doctor/diagnostic.go pkg/ui/components/welcome.go"
	count=$(git log --author="Haris0059" --oneline 2>/dev/null | wc -l)
	# --amend replaces HEAD instead of adding a commit.
	printf '%s' "$cmd" | grep -qE '\s--amend\b' || count=$((count + 1))
	# -a / --all stages tracked files at commit time, so read the working tree.
	if printf '%s' "$cmd" | grep -qE '\s(-[a-zA-Z]*a[a-zA-Z]*|--all)\b'; then
		read_file() { cat "$1"; }
	else
		read_file() { git show ":$1"; }
	fi

	major_minor=""
	bad=""
	for f in $files; do
		v=$(read_file "$f" 2>/dev/null | sed -nE 's/^(const|var) Version = "([^"]+)".*/\2/p' | head -1)
		[ -z "$major_minor" ] && major_minor=${v%.*}
		expected="$major_minor.$(printf '%03d' "$count")"
		[ "$v" = "$expected" ] || bad="$bad $f=${v:-missing}"
	done
	if [ -n "$bad" ]; then
		deny "Version not bumped. This will be Haris0059 commit #$count, so all four Version constants must be staged as $expected. Found:$bad. Follow the Version bump section of the gopher-commit skill (update and git add the four files), then commit."
	fi
fi

rm -f "$marker"
exit 0
