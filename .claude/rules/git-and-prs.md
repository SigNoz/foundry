# Git & pull request conventions

These apply to every commit, push, and PR in this repo.

## No AI attribution
Never add `Co-Authored-By: Claude …`, "Generated with Claude Code", or any
other AI/Claude attribution to commit messages, PR bodies, code comments, or
generated docs. Stop the commit body before any attribution trailer. If a
commit already carries one, `git commit --amend` to strip it.

## Push explicitly
Always push with the remote and branch named: `git push origin <branch>` — even
when the branch already tracks an upstream. Never use bare `git push`. For
force-pushes use `git push origin <branch> --force-with-lease`.

## PR descriptions follow the repo template
Fill in `.github/pull_request_template.md` — the `Features` / `Fixes` /
`Refactors` / `Tests` / `Chores` sections. Only keep the sections your change
actually touches; drop the rest along with their `<placeholder>` lines. Keep
each entry human-readable and to the point.

Do **not** add `## Test plan`, `## Testing`, or any checklist section on top of
the template. If verification matters, mention it inline.

## Branching
Commit or push only when asked. If you're on `main`, branch first.
