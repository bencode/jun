# Files

**Files**, in the side bar, shows the project's Git repository by branch, and opens its files read-only in tabs.

## The tree

- **Root**: the repository's top level. A project outside any repository shows that instead.
- **Branches**: each local branch is an item.
  - The branch checked out in the project's directory comes first, and is open.
  - The other branches follow, by their latest commit, newest first.
- **Worktrees**
  - A branch checked out in a worktree is read from that directory, so its uncommitted changes and new files show too. Ignored files are left out.
  - The tree follows the worktree's changes as they happen.
  - Detached worktrees show by directory.
- **Branches checked out nowhere** show their latest commit, marked *committed*.
- **⟳** reads the branches and worktrees again.
- Files highlights the row of the file in the current tab.

## Opening files

- **Click** a file to show it in a preview tab, in the column or bottom region you focused last. The next click replaces the preview.
- **Double click** or **Enter** keeps the file in a tab of its own.
- **Right-click** an item:

| Item | Shown for |
|---|---|
| Review Changes | a branch; opens its [Diff](diff.md) |
| Copy Relative Path | every file and folder |
| Copy Path, Reveal in Finder | a worktree's branch, files and folders |
| Open with Default App | a worktree's files and folders |

## How files show

| File | Shows as |
|---|---|
| Markdown | a page to read, at most 760 points wide |
| PNG, JPEG, GIF, WebP, BMP, SVG | the picture, as large as it is, at most as wide as the tab, with its size in the header |
| Other text | the source, highlighted |
| Other binary files, files over 10 MB | a note |

- **Markdown**
  - Its images load from the same branch: relative to the file, or with a leading slash, relative to the repository.
  - Web links open in the browser. Links to the repository's files open in Jun, as a click in Files does.
  - Links to a heading within the page do nothing yet.
- **Show as**: the header's switch shows a Markdown or SVG file's source instead, for that tab.
- **Text size**: file tabs share it with terminals. ⌘+ and ⌘− change it, and ⌘0 goes back to 13.

## Search (⌘P)

- ⌘P shows the side bar, chooses Files and focuses its search field. It does nothing until a project is open.
- **Which branch**: the current file's, else the one checked out in the project's directory. The field's menu chooses another one.
- **Matches**: every file of the branch, not only the open folders, whose path holds every word you type, ignoring case.
  - For example, `layout fix` finds `internal/layout/fixed.go`.
- **Results**: a tree with every folder open, at most 500 files. Past that, typing more narrows them.
- **Keys**
  - A click, a double click or Enter on a match opens it, as in the tree.
  - Down moves into the matches.
  - Enter in the field keeps the first match in a tab.
  - Escape, or clearing the field, brings back the tree as it was.
