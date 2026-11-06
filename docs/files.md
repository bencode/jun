# Files

**Files**, in the side bar, shows the project's Git repository by branch, and opens its files read-only in tabs.

## The tree

- **Root**: the repository's top level. A project outside any repository shows that instead.
- **Branches**: each local branch is an item.
  - The branch checked out in the project's directory comes first, and is open.
  - The other branches follow, by their latest commit, newest first, grouped by its day: today, yesterday, or its date, as in [Sessions](sessions.md).
  - Each row ends with its latest commit's time of day. Changes not yet committed do not count.
- **Worktrees**
  - A branch checked out in a worktree is read from that directory, so its uncommitted changes and new files show too. Ignored files are left out.
  - The tree follows the worktree's changes as they happen.
  - Detached worktrees show by directory.
- **Branches checked out nowhere** show their latest commit, marked *committed*.
- **⟳** reads the branches and worktrees again.
- Files highlights the row of the file in the current tab.

## Files outside the project

Any file can open in a tab, wherever it is, in two ways:

- **Drag it from Finder** onto a region's tab content: a file, a Diff, or an empty region. Each file opens in a tab of its own. Dropped on a terminal, a file goes to the terminal as its path; dropped on the rail, only a directory opens.
- **An agent** runs [`jun tool open file:<path>`](agent-tools.md#open-and-close).

A file in one of the project's worktrees shows as if opened in Files. Any other is read from its directory: it is read again when it is opened again, not as it changes, and its images and links reach only that directory and those below it.

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

## Notes

**+** in a file tab's header, or ⌘⇧M, adds a note to a [review](reviews.md) on the lines selected; clicking a line number does too. An image is noted as a whole. Reading a Markdown file takes no notes: switch to Source.

## How files show

| File | Shows as |
|---|---|
| Markdown | a page to read, as wide as View ▸ Reading Width says |
| PNG, JPEG, GIF, WebP, BMP, SVG | the picture, as large as it is, at most as wide as the tab, with its size in the header |
| Other text | the source, highlighted |
| Other binary files, files over 10 MB | a note |

- **Markdown**
  - Its images load from the same branch: relative to the file, or with a leading slash, relative to the repository.
  - Web links open in the browser. Links to the repository's files open in Jun, as a click in Files does.
  - Links to a heading within the page do nothing yet.
  - **Go to heading** (⌘⇧O, or View ▸ Go to Heading…): the command palette at `@` lists the headings of the active tab's page, indented by level. Typing filters them; Enter scrolls the heading to the top.
  - **Reading width**: View ▸ Reading Width, or Reading Width in the command palette: narrow (640 points), medium (760, the default), wide (960) or full, the tab's width. Saved as `markdown.width` in the [settings file](settings.md).
  - Go to heading is on while the active tab shows a Markdown page, and Reading Width also on a [session's tab](sessions.md): otherwise their menu items are dimmed and the palette leaves them out.
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

### Branches (`b:`)

- A search starting with `b:` finds branches instead of files: every branch whose name, with its worktree's directory name, holds every word after `b:`, ignoring case.
  - For example, `b:preview keep` finds `fix/session-preview-keep-frame`. `b:` alone lists every branch.
- **Results**: grouped by day, as the tree is.
- **Choosing one**: a click, a double click or Enter on a branch, or Enter in the field for the first, leaves the search. The tree then opens that branch, chooses it and scrolls it to the top.
- Down moves into the results.
