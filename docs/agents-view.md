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
- An agent [leads](agent-tools.md#join-and-leave) those it started and those that joined it. The list keeps a lead and the agents it leads together, as a group:
  - The group comes under the status of its most urgent agent: when one of its agents waits, the whole group comes with the waiting ones.
  - In the group, the lead comes first, and the agents it leads are set in below it, by number, with a line from it.
  - Only one level is set in: an agent led by one of those comes in the same group, and shows `↳ #8`, the number of its own lead.
  - An agent whose lead runs no agent any more comes on its own, under its own status, and shows `↳ #3`.

## Layout

- The agents are listed under a heading for each status, which counts the agents under it.
- A row shows the agent's number, then its [name](agent-tools.md#terminals-update) in bold when it has one, then its title.
- An agent with a [timer](agent-tools.md#timers) shows it before its status: `⏱ 30m · 12:30`, its interval and when it fires next, or `⏱ 15:00` for one that fires once.
- An agent with texts [held](agent-tools.md#send) while you type in its input shows how many: `✉ 2`.
- With too many for normal rows, the list uses compact single-line rows. When even those do not fit, the idle ones are folded under their heading.

## Current agent

- The agent in the terminal you focused last is marked with a bar at the left of its row.
- When the view scrolls, it scrolls the current agent into view.

## Actions

- **Click an agent** to [peek](#peek) at it.
- **⌘-click an agent** to show its terminal tab, switching project and perspective as needed.
  - When its tab was closed, the terminal reopens in its project, in the column or bottom region you focused last.
- **Right-click an agent** → **Review Changes** opens a [Diff](diff.md) of the branch the agent works on. This is offered for agents of the current project.

## Tasks

**Tasks**, in Global between Agents and Messages, is the goals and tasks agents keep with [`jun tool goal` and `task`](agent-tools.md#goal-and-task). It only shows them: agents change them.

- The bar counts the tasks doing and to do, and **All Tasks** opens the [tab of every task](#all-tasks).
- **Now**, on top, is the tasks doing, however long ago they started, and those that ended today, each with the title of its goal.
- **Goals** is the open goals as a tree, each with its tasks to do and doing. Goals show their number, such as `G2`, their period and their due date.
  - A task shows its state (○ to do, ◐ doing, ✓ done, × dropped), its number, such as `T4`, its [priority](agent-tools.md#goal-and-task) when it is `P0`, in red, or `P1`, in orange, its title, `ask N` when N messages about it still ask for you, and who follows it.
  - **Ended N** under a goal folds its tasks done and dropped; **Ended goals N**, at the bottom, folds the goals done and dropped with all their tasks.
- **Click a task** to choose it: the messages about it and the row of the agent that follows it light up. Click a fold, or a goal for its ended tasks, to open or close it.
- **Tab** from the title of Tasks, past its ⋯ if it has one, reaches the list: **↑** and **↓** move through it, **↩** opens what the task links to, else [peeks](#peek) at the agent that follows it, and **→** and **←** open and close a fold, or a goal's ended tasks.
- A message about a task has a tag with its number and title, such as `T4 Draft the plan`. Click the tag to show the task in Tasks, chosen, with Global and Tasks shown and its folds open.
- What is chosen and open lasts until Jun quits.

### All Tasks

**All Tasks**, in the command palette or the bar of Tasks, opens a tab of every task, ended ones too. Each perspective has one at most: opened again, it shows the one there.

- The filter: text found in the title, the outcome or the text; any of the four states, all when none is on; who follows it; its project.
- Tasks not ended come first, then those ended, by month, the latest first. Only those not ended show until you open a month; with text in the filter, all show.
- Each task shows its goal, its project and the day it ended.
- **↩** on a task shows it in Tasks. **→** and **←** open and close a month, as a click does.

## Messages

**Messages**, in Global, is what agents and loops tell you, so you need not open them to see how it goes. They send it with [`jun tool message`](agent-tools.md#message).

- Each message shows an icon for its kind, its number, its title, its age, its text and, when it links to something, a tag, the latest first.
  - The icons tell what asks for you, a warning, what is done and what is for your information. When an agent sees that you handled what asked for you, it turns the icon to done.
  - The number, such as `M12`, is what you and agents call the message by.
  - The tag names the project and the terminal, Diff or file: `#13 yimi`, `#13 yimi · Diff`, `yimi · notes.md`. `#13` is the number on the agent's row.
- **Replies** are what agents added to a message later, indented below it, each with its age. A reply brings its message to the top.
- **Click a message with a tag** to go to what it links to:

  | It links to | Click | ⌘-click |
  |---|---|---|
  | An agent's terminal | [Peeks](#peek) at it; clicked again, closes the peek | Shows its tab |
  | Another terminal | Shows its tab | Shows its tab |
  | A Diff or a file | Goes to its tab wherever it is, else opens one in its project | The same |
  | A project | Switches to it | The same |

  - A message whose terminal has ended, or whose project is gone, cannot be clicked.
- A long text shows four lines and a message its last three replies. **Show more** opens it where it is, and **Show less** folds it.
- Messages, like Agents and Tasks, keep the size of Jun's own text, whatever the [text size](shortcuts.md#text-size) of terminals.
- **Tab** from the title of Messages, past its ⋯ if it has one, reaches the list: **↑** and **↓** move through it, **↩** acts as a click, and **→** and **←** open and fold a long text.
- Messages last until Jun quits. It keeps the last 500.

Global shows its views together, without tabs:

- **⋯ → Agents**, **Tasks** and **Messages** show or hide each view. Its room goes to the other views. Hiding the last one hides Global, and it comes back with Global.
  - Messages keep coming while hidden.
- **⋯ → Side by Side** puts the views in columns, each as high as Global, and **Stacked** puts them one above the other.
- Drag the line between two views to resize them. Side by side, the left view keeps its dragged width as Global or the window changes; the right view takes the remaining room. When space is short, the left view temporarily narrows and returns to its saved width once there is room. Stacked views share the height by proportion.
- **Toggle Agents**, **Toggle Tasks**, **Toggle Messages** and **Toggle Global Layout** are in the View menu and the command palette. With Global hidden, the first three show it too.
- **⌃Tab** and **⌘1**–**⌘9** move between the views shown.

## Peek

Peek opens an agent's terminal in a drawer that slides out of Global's edge, so you can answer it without leaving the project and perspective you are in. The rest of the work area dims; Global stays bright, and the agent's row in it is outlined.

- **⌘J** peeks the first waiting agent. Pressed again, it moves to the next waiting one, in the list's order. On the last, the drawer's button reads **Close ⌘J**, and ⌘J closes it. With none waiting, ⌘J peeks the current agent.
- **Keys**: ⌘ shortcuts stay Jun's. Every other key goes to the agent, as in any terminal, **Esc** too: the drawer's header says **Esc → agent**, and after Esc it shows for a moment how to close the drawer.
- **Closing**: ✕, ⌘W, a click on the dimmed work area, or a click on the agent's row again. The focus goes back to where it was.
- **Open Tab ⌘↩** shows the agent's tab, as ⌘-click does. **Next ⌘J** goes to the next waiting agent.
- **Size**: the drawer is as tall as the work area, and as wide as the agent's tab, up to 62% of the work area (or 560 points where that is more). An agent from a wider tab redraws at the drawer's width, and again back in its tab.
- When the agent's tab is already on screen, a peek focuses the tab instead.
- The drawer closes when Jun moves the focus elsewhere, such as on switching project or perspective, and when the agent quits.
