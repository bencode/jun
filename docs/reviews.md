# Reviews

A **review** keeps what you note while reading a branch's changes and files. Jun sends nothing and asks for nothing: copy its address, and give it to an agent, with what you want done.

## Adding notes

| Where | How | What the note is on |
|---|---|---|
| [Diff](diff.md) | **+** on a line number, or drag line numbers, then ⌘⇧M | the lines, of the old or the new side |
| Source | a click on a line number, or select, then ⌘⇧M or **Add Note** in the context menu | the lines touched |
| Reading a Markdown file | select, then ⌘⇧M or **+** in the header | the text selected, and the lines of its paragraphs |
| Image | ⌘⇧M or **+** in the header | the whole file |

- The note is written under the view. ⌘↩ saves it, Esc cancels.
- **One review a branch**: notes on a branch, from its Diff and its files alike, go to its review in progress. A new one starts with the first note after the last is done.
- A note keeps the file, the lines and their text, what the view compared, and the branch's commit.
- **Uncommitted**: a note on text not committed yet is marked so. The commit does not hold that text; the text kept in the note does.

In a Diff, notes show under their lines. In the source, a dot and a tint mark their lines, and a click on the dot opens the review. Only the review in progress shows.

## Reviews

**Reviews**, in the side bar, lists the project's reviews by day, the latest first. It is hidden until you show it: with the side bar's **⋯**, **Show Reviews** in ⌘K, or a click on the count of notes in a Diff or file tab.

- **A row**: `●` in progress or `✓` done, the branch, then its commit, whether anything is uncommitted, the time it started and its number of notes.
- **Click** a review to show it in a preview tab. **Double click** or **Enter** keeps it in a tab of its own.
- **Right-click** to open it, copy its address, or delete it.

## A review's tab

- **Copy Address** copies the path of the review's file. Give it to an agent to read: a JSON file with each note's `path`, `lines`, `quote` and `text`.
- **Done** ends the review, which can then not be changed.
- **Click a note** to go to its place: the view shows again what it compared, and the text is looked for near where it was.
  - **Moved to line N** tells that the text moved.
  - **The text has changed** tells that it is gone: the view shows where it was.
  - When a Diff no longer changes the file, the file opens instead.
