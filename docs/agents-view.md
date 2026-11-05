# Agents

The **Agents** view, in Global, shows every coding agent running in Jun's terminals, across all projects. Global is shared by every project, so the view stays put while you switch projects.

## Which agents

- Claude Code (`claude`), Codex (`codex`) and Pi (`pi`).
- Jun recognizes an agent by its terminal's foreground process, which it checks every second.
- Jun reads an agent's status from the terminal's title and screen. These are heuristics, and a new version of an agent may show something they miss.

## Order and status

| Status | Meaning | Order |
|---|---|---|
| Waiting | The agent asks you something, such as a permission or a choice, or Claude Code shows a notice that waits for Enter | first |
| Working | The agent is busy | second |
| Idle | The agent waits for a new prompt | last |

- The rail shows a project's waiting agents on its tile; see [Workspace](workspace.md#projects).

## Layout

- **Auto** (the default)
  - When every agent fits, they show as **cards**, each with a read-only, live copy of the agent's screen.
  - When they do not fit, they are **listed** under a heading for each status.
  - With too many for normal rows, the list uses compact single-line rows. When even those do not fit, the idle ones are folded under their heading.
  - Widen Global to fit more cards.
- The view's **Auto ▾** menu can keep it to **Cards** or **List**. The view remembers the choice.

## Current agent

- The agent in the terminal you focused last is marked **Current**.
- When the view scrolls, it scrolls the current agent into view.

## Actions

- **Click an agent** to [peek](#peek) at it.
- **⌘-click an agent** to show its terminal tab, switching project and perspective as needed.
  - When its tab was closed, the terminal reopens in its project, in the column or bottom region you focused last.
- **Right-click an agent** → **Review Changes** opens a [Diff](diff.md) of the branch the agent works on. This is offered for agents of the current project.

## Peek

Peek opens an agent's terminal in a drawer that slides out of Global's edge, so you can answer it without leaving the project and perspective you are in. The rest of the work area dims; Global stays bright, and its card for the agent says **Peeking**.

- **⌘J** peeks the first waiting agent. Pressed again, it moves to the next waiting one. On the last, the drawer's button reads **Close ⌘J**, and ⌘J closes it. With none waiting, ⌘J peeks the current agent.
- **Keys**: ⌘ shortcuts stay Jun's. Every other key goes to the agent, as in any terminal, **Esc** too: the drawer's header says **Esc → agent**, and after Esc it shows for a moment how to close the drawer.
- **Closing**: ✕, ⌘W, a click on the dimmed work area, or a click on the agent's card again. The focus goes back to where it was.
- **Open Tab ⌘↩** shows the agent's tab, as ⌘-click does. **Next ⌘J** goes to the next waiting agent.
- **Size**: the drawer is as tall as the work area, and as wide as the agent's tab, up to 62% of the work area (or 560 points where that is more). An agent from a wider tab redraws at the drawer's width, and again back in its tab.
- When the agent's tab is already on screen, a peek focuses the tab instead.
- The drawer closes when Jun moves the focus elsewhere, such as on switching project or perspective, and when the agent quits.
