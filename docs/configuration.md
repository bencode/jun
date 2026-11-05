# Configuration and saved state

Jun keeps two files. **Preferences** live in the settings file, which you edit. **State** lives in the layout file, which Jun writes.

| File | Holds | Default location | Override |
|---|---|---|---|
| `settings.yaml` | language, theme, text size, reading width, Claude Code folders | `~/.config/jun/settings.yaml` | `JUN_SETTINGS_PATH` |
| `config.json` | projects, perspectives, regions, tabs | `~/Library/Application Support/Jun/config.json` | `JUN_CONFIG_PATH` |

- The settings file is described in [Settings](settings.md).
- In both overrides, a relative path resolves against the directory Jun was launched from.

## What the layout file saves

- The projects, in order, with their names and directories, and the selected project.
- Each project's perspectives, in order, with their stable IDs and names, and the selected one.
- Each perspective's layout:
  - its columns, bottom region and side bar, including hidden ones;
  - the order of their tabs, and the selected tab;
  - the split percentages and the side bar's width.
- What tabs show:
  - the file a file tab shows: a branch, or a detached worktree's directory, and a path;
  - the session a session tab shows;
  - what a Diff tab compares.
- Global, its visibility, and the Agents view's layout choice.
- Whether the project rail is expanded.

## What it does not save

- Running processes and their output. Every saved terminal tab starts a fresh shell on the next launch; see [Terminals](terminals.md#quitting-and-restarting).
- Closed tabs, scroll offsets, and the window's size and position.

## How Jun saves

- Changes save after 300 ms without further changes.
- Pending changes are written at once when the window loses focus, hides or closes.
- Jun writes a temporary file in the same directory, then replaces the file atomically.
- Jun reads the file only at startup. Quit Jun before editing it by hand.
- MyGo treats development and packaged builds separately when it keeps Jun to a single instance. Give development builds their own `JUN_CONFIG_PATH`, so the two do not share one file.

## When something goes wrong

| Situation | What Jun does |
|---|---|
| The file is unreadable, invalid or of an unsupported version | Keeps the file untouched, reports the error, and does not save during that run. Fix the file and restart. |
| A save fails | Keeps the pending changes, and offers **Retry Save** or **Continue Editing**. Continuing pauses retries until the next change or an attempt to quit. |
| A save fails while quitting | Keeps the window open, with **Retry Save**, **Cancel Quit** and **Quit Without Saving**. |
| A project's directory is missing | Keeps the project and its layout, and reports the problem when you enter the project. |
| A terminal's directory is missing | Keeps the tab, with a **Retry** button. |

## Versions

- The layout file is `{"version": 3, "layout": …}`.
- Older files are converted in memory when loaded, and the next save writes version 3:
  - **Version 1 → 2** keeps the Work and Review perspectives, every tab and region ID, the selections, each tab's project, and hidden layouts.
  - **Version 2 → 3** gives each perspective a hidden side bar. Terminals in Global move to the end of the first column of their project's current perspective. Agents and Files tabs in columns and bottom regions are dropped, as are version 1's placeholder file tabs.
- Older versions of Jun cannot load version 3. Quit the previous Jun before running the upgraded app with the same file.
- Preferences that earlier versions kept in the layout file (`chinese`, `theme`, `textSize`) move into the settings file on the first start.
- Even older files, `state.json`, `settings.json` and `window-state.json`, are left untouched.
