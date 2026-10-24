# Jun

A native, GPU-rendered workspace for working with terminal agents. Built with Go and [MyGo](https://github.com/egoist/mygo), without a WebView.

Jun currently provides the macOS terminal workflow and configuration recovery: projects rooted at local directories, named perspectives, one to three columns, a shared bottom region, and an independent Global region. Each region holds movable, closable tabs.

Opening a new project creates one perspective named Work with one real terminal, running your login shell in the project directory. Additional perspectives start with one empty column; the bottom and Global regions start empty and hidden. Command entrypoints arrive in I03, Agent overview in I04, and file/diff Review in I05.

## Development

Requires macOS 13 or newer, Xcode command line tools, and Go 1.27.1 or newer. Use an isolated development config so development and packaged builds do not share saved layouts:

```sh
JUN_CONFIG_PATH="$PWD/priv/config.dev.json" go run ./cmd/jun
go test ./...
go vet ./...
go tool mygo build -platform darwin/arm64 -skip-dmg
```

Open a project through the rail’s **+** button or **File → Open Project…**. Confirm the directory’s name, switch perspectives using the layers button near the bottom of the project rail, and use **⋯ → Open View** in any region to open a terminal or a preview view. An empty region offers the same **Open View** menu; column actions remain in that column’s **⋯**, and bottom visibility lives in the Perspective menu. Drag tabs to reorder or move between regions. Moving a tab to a hidden bottom/Global region through its context menu reveals it. The upper-right center control hides or restores Global.

The gear button below Perspective opens **Settings**, offering the same English (default) and Simplified Chinese choices as the native **Language** menu. Language applies to the current session. Perspective names retain their spelling when the UI language changes. Repeated launches of the same build focus its existing window; New Window is deferred to a separate iteration.

## Perspectives

The **Perspective** menu lists the current project’s perspectives in creation order. **New Perspective…** asks for a name, then selects a new empty perspective without starting a terminal. **Rename Perspective…** changes the current name while preserving its tabs, layout and terminal sessions. Names must be nonempty and unique within the project, ignoring case. Other projects can reuse the same names.

The rail shows icons when collapsed and the current perspective name when expanded; its tooltip includes the project and full perspective name. Perspective is disabled without an open project. Switching projects restores that project’s selected perspective. Global remains independent. Deleting, duplicating and reordering perspectives are deferred.

## Opening views

Columns, the shared bottom region, and Global all offer the same **Open View** menu: **New Terminal**, **Agents (Preview)**, and **Files (Preview)**. New projects still start with one terminal. Agents and Files currently show explicitly labeled layout previews; Agent data and file reading arrive in later iterations. There is no file-opening action yet.

Opening the same preview type and source project again in a region selects its existing tab. Other regions can hold independent views. Files previews retain the project they were opened from when moved into Global or when switching projects. Without a selected project, Global can open an Agents preview; New Terminal and Files are disabled.

## Layout and tab actions

Each column’s **⋯** offers **Add Column to Right** and **Remove This Column**, with one to three columns allowed. Removing a column transfers its tabs to the left neighbor (the right neighbor for the first column) and keeps the removed column’s active tab selected. Empty-column removal preserves the destination’s selection. These actions never terminate terminal sessions.

The **Perspective** menu switches perspectives and shows or hides the shared bottom region. The bottom and Global menus open/reopen views and hide their own region; they do not change center columns or another region’s layout. Showing bottom returns focus to the previously active region; hiding an active region returns focus to the first center column.

Click a tab to select it, use its **×** to close it, or right-click it for **Close Tab** and **Move Tab To**. Right-clicking an inactive tab does not select it; the chosen action targets that tab. Move destinations include the project, perspective and column/bottom name. Closing an inactive tab preserves the group’s selection; closing the active tab selects its right neighbor, then its left neighbor. An empty region stays open. Window controls operate on the application window.

## Terminals and background sessions

Use **⋯ → Open View → New Terminal** or **⌘T** to create a terminal in the active region. **⌘W** closes its current view; **⌃Tab / ⌃⇧Tab** cycle that region’s tabs. The active region follows keyboard focus, including the bottom and Global regions. Switching project or perspective focuses its first column. Closing the last tab leaves the region open; **⌘Q** quits the application.

Switching projects/perspectives, hiding a region, moving a view, or closing its tab keeps the shell running. **⋯ → Reopen Terminal** lists closed terminals belonging to the current project and reopens the same session with its output and scrollback. Shell exit preserves output and displays the exit code. Quitting Jun closes all sessions; there is no background daemon or process recovery across app restarts.

A terminal moved into Global retains its source project. Creating a new terminal in Global uses the currently selected project’s directory. Changing directory inside a shell does not change the project root. User shell startup scripts can change their own directory.

MyGo’s Ghostty plugin provides GPU rendering, input method composition, copy/paste, scrolling and resizing, with a 10 MiB scrollback budget per terminal. Run installed Claude Code, Codex, Pi or other CLIs directly in the shell. Jun does not make AI API calls.

The packaged app includes `libghostty-vt`; development commands may download the pinned library into MyGo’s user cache on first use. Library loading failures stop startup with an error.

## Saved configuration

The default macOS file is `~/Library/Application Support/Jun/config.json`. `JUN_CONFIG_PATH` overrides the file location; relative paths resolve against the launch directory. MyGo distinguishes development and packaged builds for its single-instance behavior, so use the override when running them alongside each other.

The JSON document contains `version: 2` and a `layout` snapshot. It saves the ordered projects, names/CWDs, selected project by name, ordered perspectives with stable IDs and names, the selected perspective, each perspective’s columns and bottom region, tab order and selection, split percentages, project rail expansion, and the independent Global region. Global views retain their source project when switching projects. Hidden regions are saved too.

Version 1 configurations are converted in memory, retaining both Work and Review, all tab/region IDs, selections, source projects, and hidden layouts. Loading does not rewrite the file; the next normal save writes version 2. Older Jun versions cannot load version 2. Quit the previous Jun before using the upgraded app with the same configuration.

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

Dependencies flow from `cmd/jun` to `app`, then from `app` to `config`, `layout`, `terminal`, and MyGo. `terminal` uses MyGo’s terminal plugin. `config` uses `layout` and standard-library JSON/filesystem APIs; `layout` uses only the standard library. There is no dedicated filesystem sidebar: a file tree is an ordinary view in a perspective.

Planning documents, the frozen HTML prototype, research and local review artifacts belong under the ignored `priv/` directory. Generated `.mygo/` and `build/` directories are also ignored.
