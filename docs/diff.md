# Diff

A **Diff** tab shows a branch's changes, for reviewing an agent's work before its pull request.

## Opening a Diff

| Entry | Where |
|---|---|
| ⌘⇧D | lists the project's branches in the command palette |
| **Review Changes: …** | in ⌘K |
| **Review Changes** | a branch's context menu in [Files](files.md) |
| **Review Changes** | an agent's context menu in [Agents](agents-view.md) |

- In the ⌘⇧D list, branches checked out in a worktree come first. Each is marked with where it is checked out (the project directory, or a worktree), and with the agent working there and its status.
- **Where it opens**: where new tabs go, in the current perspective. To review in a perspective of its own, switch to that perspective first.
- **One per region**: a branch has at most one Diff tab in a region. Opening it again chooses that tab.

## What it compares

- **Compare to** (in the header): by default, the branch this one most likely forked from.
  - Jun considers the local branches and the remote branches they track, and picks the one that lacks the fewest of this branch's commits.
  - Its menu lists them, marks the guess *(guessed)*, and lets you choose another one.

- **Scope** (in the header):

| Scope | Changes shown |
|---|---|
| All Changes | everything since the base, including uncommitted changes and new files that are not ignored |
| Uncommitted Only | only what is not committed yet |
| One Commit | the changes of one commit |

## Reading it

- The changed files are listed by directory, each with its lines added and deleted.
- The chosen file shows **side by side**, or **unified** with **⇆**, at the text size of file tabs.
- Binary files, and files over 10 MB, show no lines.
- A worktree's Diff is read again as its files change. **⟳** reads it again at once.
