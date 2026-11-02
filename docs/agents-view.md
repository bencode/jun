# Agents

The **Agents** view, in Global, shows every coding agent running in Jun's terminals, across all projects. Global is shared by every project, so the view stays put while you switch projects.

## Which agents

- Claude Code (`claude`), Codex (`codex`) and Pi (`pi`).
- Jun recognizes an agent by its terminal's foreground process, which it checks every second.
- Jun reads an agent's status from the terminal's title and screen. These are heuristics, and a new version of an agent may show something they miss.

## Order and status

| Status | Meaning | Order |
|---|---|---|
| Waiting | The agent asks you something, such as a permission or a choice | first |
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

- **Click an agent** to show its terminal tab, switching project and perspective as needed.
  - When its tab was closed, the terminal reopens in its project, in the column or bottom region you focused last.
- **Right-click an agent** → **Review Changes** opens a [Diff](diff.md) of the branch the agent works on. This is offered for agents of the current project.
