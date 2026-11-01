# Jun

A native, GPU-rendered workspace for working with terminal agents. Built with Go and [MyGo](https://github.com/egoist/mygo), without a WebView.

Jun currently provides the macOS terminal workflow and configuration recovery: projects rooted at local directories, named perspectives, one to three columns, a shared bottom region, a side bar, and an independent Global region. Columns and the bottom region hold movable, closable tabs; the side bar and Global hold fixed views.

Opening a new project creates one perspective named Work with one real terminal, running your login shell in the project directory, and its side bar shown. Additional perspectives start with one empty column; their bottom region starts empty and hidden, and their side bar hidden. Global starts hidden. Command entrypoints arrive in I03 and file/diff Review in I05.

## Development

Requires macOS 13 or newer, Xcode command line tools, and Go 1.27.3 or newer. Use an isolated development config so development and packaged builds do not share saved layouts:

```sh
JUN_CONFIG_PATH="$PWD/priv/config.dev.json" go run ./cmd/jun
go test ./...
go vet ./...
go tool mygo build -platform darwin/arm64 -skip-dmg
```

Press ⌘K for the command palette. Type a directory, such as `~/work/`, to open it as a project: Tab goes into the highlighted directory, and Enter opens it. Anything else filters commands, such as New Terminal or switching projects and perspectives. The rail’s **+** button and **File → Open Project…** (⌘O) open the palette at `~/`; its **Browse…** item opens the system folder picker. You can also drag a directory from Finder onto the rail, or run `jun <directory>` in a terminal, such as `jun .`, after adding an alias to your shell:

```sh
alias jun=/Applications/Jun.app/Contents/MacOS/Jun
```

The directory’s name becomes the project’s; Jun asks for another only when it is taken, and selects the project already open at a directory instead of opening it again. Switch perspectives using the layers button near the bottom of the project rail, and use **⋯ → Open View** in any region to open a terminal or a preview view. An empty region offers the same **Open View** menu; column actions remain in that column’s **⋯**, and bottom visibility lives in the Perspective menu. Drag tabs to reorder or move between regions. Moving a tab to a hidden bottom/Global region through its context menu reveals it. The upper-right center control hides or restores Global.

The gear button at the bottom of the rail opens **Settings ▸ Language**, offering the same English (default) and Simplified Chinese choices as the native **Language** menu. The language is saved with the layout and used on the next launch. **Settings ▸ Theme…**, or **Theme…** in the command palette, lists Jun’s own theme and Ghostty’s, including any in `~/.config/ghostty/themes`: type to filter, and the highlighted theme shows at once; Enter keeps it and Esc goes back. The theme colors the terminals, and Jun’s controls take their background and text from it, light or dark. **Wez** is WezTerm’s default colors. The theme is saved with the layout; one no longer found falls back to Jun’s, saying so. Perspective names retain their spelling when the UI language changes. Repeated launches of the same build focus its existing window; New Window is deferred to a separate iteration.

## Perspectives

The **Perspective** menu lists the current project’s perspectives in creation order. **New Perspective…** asks for a name, offering the first of Work, Review, Monitor, Debug and Notes the project lacks, then “Perspective” and a number, selected so typing replaces it; it then selects a new empty perspective without starting a terminal. **Rename Perspective…** changes the current name while preserving its tabs, layout and terminal sessions. Names must be nonempty and unique within the project, ignoring case. Other projects can reuse the same names. **Delete Perspective…** removes the current perspective with its tabs and selects its neighbor; when it shows terminals, Jun asks first and then closes them, with any agents running in them. A project keeps at least one perspective.

The view bar, at the right end of the last column’s tab bar, shows the current perspective’s name, which opens the **Perspective** menu, and its tooltip includes the project and full perspective name; in a narrow column the name gives way, down to its icon. Without an open project only the Global switch shows. Switching projects restores that project’s selected perspective. Global remains independent. Duplicating and reordering perspectives are deferred.

In the rail, each project shows as a colored tile of a two-character abbreviation: the initials of its name’s first two words, words parted by separators, camelCase and digits (`cc-mission-control` → CM, `jun2` → J2), or else its first two letters (`jun` → Ju). The color follows the name. Expanded, the rail adds the project’s name and its directory, with home as `~` and long paths cut to their last two parts. A project with agents waiting for you shows an orange dot on its tile, or their number when expanded. Right-click a project in the rail for **Rename Project…** and **Remove Project…**. Renaming keeps its perspectives, tabs and terminal sessions, with the same name rules as opening a project. Removing always asks first; it closes all of the project’s terminals, including closed tabs’ sessions and any agents running in them, and its views moved into other projects. The directory stays on disk.

## Opening views

Columns and the shared bottom region offer an **Open View** menu with **New Terminal**. New projects still start with one terminal. There is no file-opening action yet.

## Side bar and Global

The side bar, left of the columns, and Global, right of them, hold fixed views: their tabs cannot be closed, dragged, or moved elsewhere, and other tabs cannot be moved into them. Their **⋯** menu only hides the region. Each perspective has its own side bar with **Files** and **Sessions**. Show or hide it with **⌘B** or the Perspective menu, and drag its edge to resize it: it keeps its width as the window or Global changes. Global, shared by every project, holds **Agents**.

**Files** shows the project’s Git repository, from its top level, and nothing else: each local branch is an item, the one checked out in the project’s directory first and open, then the others by their latest commit, newest first. A branch checked out in a worktree is read from that directory, uncommitted changes and new files included, ignored ones left out, and follows its changes; one checked out nowhere shows its latest commit, marked committed. Detached worktrees show by directory. **⟳** reads the branches and worktrees again. A click on a file shows it, read-only, in a preview tab (its title in italics) in the column or bottom region last focused; the next click replaces it, and a double click or Enter keeps it as a tab of its own. Files shows the current file’s row. Its context menu copies paths and, for a worktree’s files, reveals them in Finder or opens them in their default app. A project outside any repository shows that instead. Markdown files show as a page to read, at most 760 points wide: their images load from the same branch, relative to the file or, with a leading slash, to the repository, and their links open web pages in the browser and the repository’s files in Jun, as a click in Files does (links to a heading within the page do nothing yet). Pictures (PNG, JPEG, GIF, WebP, BMP and SVG) show as large as they are, at most as wide as the tab, with their size in the header. The header’s switch shows a Markdown or SVG file’s source instead, for that tab. Terminals and file tabs share one text size, 13 by default: **⌘+** and **⌘−** make it larger or smaller, **⌘0** returns it to 13, and **Settings ▸ Font…** sets one, in a panel for the text of terminals and file tabs. Open tabs follow it at once, and Jun saves it. Other binary files and files over 10 MB show a note.

Files’ search field, which **⌘P** shows and focuses, finds files in one branch: the current file’s, else the one checked out in the project’s directory, or the one its menu chooses. It searches all of the branch’s files, not only the open folders, for paths holding every word typed, ignoring case: `layout fix` finds `internal/layout/fixed.go`. The matches show as a tree with every folder open, at most 500 of them; past that, typing more narrows them. A click, double click or Enter on a match opens it as in the tree; Down moves into the matches, and Enter in the field keeps the first one in a tab. Escape or clearing the field brings back the tree as it was.

**Sessions** lists the project’s Claude Code and Codex sessions: those started in its directory or one of its worktrees, by the day they were last active, the latest first, each with its title, the branch it started on and the time. Jun reads them from Claude Code’s configuration directory (`CLAUDE_CONFIG_DIR`, else `~/.claude`) and from `CODEX_HOME` (else `~/.codex`), only each file’s start and end, again as they change; **⟳** reads them again at once. Subagents’ sessions and sessions where nothing was asked are left out. The list shows the latest 500; the search field, which searches them all, keeps the sessions whose title or first or last prompt holds every word typed, ignoring case, again at most 500. A click shows a session read-only in a preview tab, as Files does, and a double click or Enter in a tab of its own: the prompts and answers, the latest at the bottom, with the tools’ work counted between them, read again as the session goes on. **Resume**, in that tab’s header or a session’s context menu, opens a terminal in the session’s directory and runs `claude --resume <id>` or `codex resume <id>` there.

**Diff** shows a branch's changes, for reviewing an agent's work before its pull request. Open one with **⌘⇧D**, which lists the project's branches in the command palette, those checked out in a worktree first, marked with the agent working there; the same **Review Changes** items are in ⌘K, in a branch's context menu in Files, and in an agent's context menu in Agents. It opens where new tabs go, in the current perspective: keep a perspective for reviewing, and switch to it first. A branch has one Diff tab in a region; opening it again chooses it. The header chooses what the branch is compared to: by default the branch it most likely forked from, of the local branches and the remote branches they track, the one fewest of its commits are not in, marked *guessed*. Its scope shows all the changes since then, including those not committed and new files not ignored, only those not committed yet, or one commit's. The files changed are listed by directory, each with its lines added and deleted; the one chosen shows side by side or, with **⇆**, unified, at the text size of file tabs. Binary files and files over 10 MB show no lines. A worktree's Diff is read again as its files change; **⟳** reads it again at once.

**Agents** shows every Claude Code, Codex and Pi running in Jun’s terminals, across all projects, with the most in need of you first: waiting, then working, then idle. When they all fit in the view, they show as cards, each with a read-only, live copy of the agent’s screen; otherwise they are listed under a heading for each status, in compact single-line rows when there are too many for normal rows, and with the idle ones folded under their heading when even those would not fit. Widen Global to fit more cards. The view’s **Auto ▾** menu can keep it to **Cards** or **List** instead, and remembers the choice. The agent in the terminal you focused last is marked as current, and scrolled into view when the view scrolls. Clicking an agent shows its tab, switching project and perspective as needed; when its tab was closed, the terminal reopens in its project, in the column or bottom region focused last. Jun recognizes an agent by the terminal’s foreground process, checked every second, and reads its status from its title and screen, as heuristics that can miss what a new agent version shows.

## Layout and tab actions

Each column’s **⋯** offers **Add Column to Right** and **Remove This Column**, with one to three columns allowed. Removing a column transfers its tabs to the left neighbor (the right neighbor for the first column) and keeps the removed column’s active tab selected. Empty-column removal preserves the destination’s selection. These actions never terminate terminal sessions.

The **Perspective** menu switches, creates, renames and deletes perspectives. Beside it, the view bar’s three switches show or hide the side bar (also **⌘B**), the shared bottom region and Global, each lit while its region shows; using them leaves focus where it was. The bottom menu opens/reopens views and hides its region, as the side bar and Global menus hide theirs; they do not change center columns or another region’s layout. Showing bottom returns focus to the previously active region; hiding an active region returns focus to the first center column.

Click a tab to select it, use its **×** to close it, or right-click it for **Close Tab** and **Move Tab To**. Right-clicking an inactive tab does not select it; the chosen action targets that tab. Move destinations include the project, perspective and column/bottom name. Closing an inactive tab preserves the group’s selection; closing the active tab selects its right neighbor, then its left neighbor. An empty region stays open. Window controls operate on the application window.

## Terminals and background sessions

Use **⋯ → Open View → New Terminal** or **⌘T** to create a terminal in the active region; from the side bar or Global, it goes to the column or bottom region focused last. **⌘W** closes the current tab of a column or the bottom region; **⌃Tab / ⌃⇧Tab** cycle the active region’s tabs. The active region follows keyboard focus, including the side bar, bottom and Global regions. Switching project or perspective focuses its first column. Closing the last tab leaves the region open; **⌘Q** quits the application. See [Keyboard shortcuts](docs/shortcuts.md) for tab, region, perspective and project navigation.

Switching projects/perspectives, hiding a region, moving a view, or manually closing its tab keeps the shell running. **⋯ → Reopen Terminal** lists closed terminals belonging to the current project and reopens the same session with its output and scrollback. A shell exiting (`exit`, ⌃D or the process ending, whatever its exit code) automatically closes its tab and releases its session, including manually hidden sessions; it cannot be reopened. A shell that fails to start keeps its tab, showing the error. A child command returning to the shell leaves the tab open. Background exits do not change project, perspective, region visibility or focus. A focused shell exiting focuses the next tab to its right, otherwise its left; an empty region remains open with its menu focused. Exited tabs are removed from saved layouts. Quitting Jun closes all sessions; there is no background daemon or process recovery across app restarts.

A terminal keeps its source project wherever it moves. Changing directory inside a shell does not change the project root. User shell startup scripts can change their own directory.

MyGo’s Ghostty plugin provides GPU rendering, input method composition, copy/paste, scrolling and resizing, with a 10 MiB scrollback budget per terminal. Run installed Claude Code, Codex, Pi or other CLIs directly in the shell. Jun does not make AI API calls.

The packaged app includes `libghostty-vt`; development commands may download the pinned library into MyGo’s user cache on first use. Library loading failures stop startup with an error.

## Saved configuration

The default macOS file is `~/Library/Application Support/Jun/config.json`. `JUN_CONFIG_PATH` overrides the file location; relative paths resolve against the launch directory. MyGo distinguishes development and packaged builds for its single-instance behavior, so use the override when running them alongside each other.

The JSON document contains `version: 3` and a `layout` snapshot. It saves the ordered projects, names/CWDs, selected project by name, ordered perspectives with stable IDs and names, the selected perspective, each perspective’s columns, bottom region and side bar, tab order and selection, the files file tabs show (a branch, or a detached worktree’s directory, and a path), split percentages and the side bar’s width, project rail expansion, the theme, the Agents view’s layout choice, and the independent Global region. Hidden regions are saved too. The side bar’s and Global’s views follow Jun’s presets: loading adds the views a preset gained and drops those it lost.

Older configurations are converted in memory. Version 1 becomes version 2, retaining both Work and Review, all tab/region IDs, selections, source projects, and hidden layouts. Version 2 becomes version 3: each perspective gets a hidden side bar, terminals in Global move to the end of the first column of their project’s current perspective, and Agents and Files tabs in columns and bottom regions are dropped, as are the placeholder File tabs of version 1. Loading does not rewrite the file; the next normal save writes version 3. Older Jun versions cannot load version 3. Quit the previous Jun before using the upgraded app with the same configuration.

Changes save after 300 ms of idle time, and pending changes flush when the window loses focus, hides, or closes. Writes use a same-directory temporary file and atomic replacement. Startup restores an existing snapshot instead of replacing it with defaults. A missing project directory preserves the project and layout and reports the problem when entering that project.

An unreadable, invalid, or unsupported configuration is preserved; Jun reports the error and disables saving for that run. Fix the file and restart. Other save failures preserve pending changes and offer Retry Save or Continue Editing. If saving fails during exit, the window stays open with Retry Save, Cancel Quit, and Quit Without Saving choices. Continuing editing pauses retries until another layout change or an exit attempt.

Only layout descriptions are restored. Startup creates fresh shells for every saved Terminal tab, including hidden regions and inactive projects/perspectives, using each tab’s source project directory. A missing directory preserves the tab with a Retry action. Previous processes, output, closed tabs, GPU/UI objects, scroll offsets, language choice, and native window geometry are not saved. Configuration is read at startup; quit Jun before editing it manually. Earlier `state.json`, `settings.json`, and `window-state.json` files are left untouched.

## Structure

```text
cmd/jun          application entrypoint
internal/app     native UI, directory picker, save scheduling and error handling
internal/config  versioned JSON file loading and atomic replacement
internal/layout  pure Go project/perspective/region/view model and validation
internal/terminal  runtime sessions and Ghostty resource lifecycle
```

Dependencies flow from `cmd/jun` to `app`, then from `app` to `config`, `layout`, `terminal`, and MyGo. `terminal` uses MyGo’s terminal plugin. `config` uses `layout` and standard-library JSON/filesystem APIs; `layout` uses only the standard library.

Planning documents, the frozen HTML prototype, research and local review artifacts belong under the ignored `priv/` directory. Generated `.mygo/` and `build/` directories are also ignored.
