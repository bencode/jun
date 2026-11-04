# Terminals

Jun's terminals run your login shell. Agents such as Claude Code, Codex and Pi run inside them as they would in any terminal.

## What a terminal is

- **Shell**: your login shell, started in the project's directory. The directory belongs to the tab: `cd` inside the shell does not change the project.
- **Rendering**: MyGo's Ghostty plugin, on the GPU. It supports input method composition, copy and paste, scrolling and resizing.
- **Scrollback**: 10 MiB per terminal.
- **Text size**: shared with file tabs, 13 by default. ⌘+ and ⌘− change it, ⌘0 goes back to 13, and Jun saves it as `font.size` in the [settings file](settings.md).
- **Theme**: Jun's own, or any Ghostty theme, including those in `~/.config/ghostty/themes`. See [Settings](settings.md).
- **No AI of its own**: Jun makes no AI API calls. It runs the CLIs you have installed.
- **Agent tools**: each terminal has `JUN_API`, `JUN_BIN` and `JUN_TERMINAL` set, for programs in it to ask Jun about its terminals. See [Agent tools](agent-tools.md).

## Shells keep running

- These keep the shell running:
  - switching projects or perspectives;
  - hiding a region;
  - moving the tab;
  - closing the tab.
- **⋯ → Reopen Terminal** lists the current project's closed terminals. It reopens the same session, with its output and scrollback.
- The **Agents** view lists agents in closed tabs as well; clicking one reopens its terminal. See [Agents](agents-view.md).

## When a shell ends

- A shell that exits (`exit`, ⌃D, or the process ending, with any exit code) closes its tab and ends its session. It cannot be reopened.
- A command that returns to the shell leaves the tab open.
- A shell that ends in the background does not change the project, perspective, regions or focus.
- When the focused shell ends, the focus goes to the tab to its right, else the one to its left. A region left empty stays open, with its menu focused.
- A shell that fails to start keeps its tab, which shows the error and a **Retry** button.

## Quitting and restarting

- **Quitting Jun**, or closing its window, ends every shell. Jun has no background daemon.
- When programs still run in some terminals, Jun lists them first ("codex — project · tab") and asks. Cancel takes Enter and Esc.
- **On the next start**, Jun starts a fresh shell for every saved terminal tab, in each tab's own project directory. This includes tabs in hidden regions and in other projects and perspectives.
- Earlier processes and their output are not restored.
- A directory that no longer exists keeps its tab, with a **Retry** button.

## Ghostty's library

- The packaged app includes `libghostty-vt`.
- Development runs (`go run`, `go test`) download the pinned library into MyGo's cache on first use, and check its SHA-256.
- If the library cannot load, Jun stops at startup with an error.
