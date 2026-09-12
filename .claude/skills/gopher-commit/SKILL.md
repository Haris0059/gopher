---
name: gopher-commit
description: ALWAYS use this instead of the global commit-message skill when committing in the gopher repo. Use this skill whenever creating, writing, or generating a git commit message in this repo, or any time the user says "commit", "git commit", "stage and commit", "push changes", or asks to save/finalize code changes to version control. Generates properly formatted commit messages by analyzing staged changes via git diff --staged, and bumps the project's PATCH version to match Haris0059's commit count before committing. Always use before running any git commit command.
---

# Generating Commit Messages (gopher project)

Use this instead of the global `commit-message` skill in this repo: same message
format, plus a mandatory version-bump step. The rule is also documented in
`CLAUDE.md` ("Versioning") and enforced by `.claude/hooks/pre-commit-check.sh`,
which denies a commit whose staged version is wrong.

## Process
1. Run `git diff --staged` to see all staged changes
2. Analyze what changed and why
3. Generate a commit message following the format below
4. Bump the version (see below), staging the changed version files
5. Only then run `git commit`

## Version bump

Gopher's version is `MAJOR.MINOR.PATCH` (e.g. `0.3.024`). MAJOR.MINOR is a
manual decision — never change it here. PATCH is always the zero-padded
(minimum 3 digits) count of commits authored by `Haris0059` in this repo,
**including the commit about to be made**.

1. Get the current version from `cmd/gopher/main.go` (`const Version = "..."`)
   and split off MAJOR.MINOR.
2. Count existing commits by the user and add 1 for the commit in progress:
   ```
   n=$(( $(git log --author="Haris0059" --oneline | wc -l) + 1 ))
   ```
3. Zero-pad `n` to at least 3 digits (e.g. `24` → `024`; `104` stays `104`)
   and set `NEW_VERSION = MAJOR.MINOR.<padded n>`.
4. If `NEW_VERSION` equals the version already in the files, skip the rest
   of this section (nothing to bump — e.g. a second commit-message dry run).
5. Update the `Version` constant to `NEW_VERSION` in all four places it is
   duplicated:
   - `cmd/gopher/main.go`
   - `internal/cli/repl.go`
   - `pkg/doctor/diagnostic.go`
   - `pkg/ui/components/welcome.go`
6. `git add` those four files so the bump lands in the same commit as the
   rest of the staged changes.

Skip this whole section for a commit not authored by `Haris0059` (check
`git config user.name` first) — the patch count is this user's alone.

## Format

### Summary line (under 50 characters)
- Present tense imperative mood: "Add", "Fix", "Update", "Remove"
- Specific and descriptive — no generic messages

### Example
```
Add JWT authentication middleware
```

## Forbidden
- No "Generated with Claude Code"
- No "Co-Authored-By: Claude noreply@anthropic.com"
- No vague messages like "Update files" or "Fix issues"
- No Body in commit messages
