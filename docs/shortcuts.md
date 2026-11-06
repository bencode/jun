# Keyboard Shortcuts

**Settings ▸ Keyboard Shortcuts…**, or **Keyboard Shortcuts** in the command palette (⌘K), lists every shortcut below. Search it by action, in English or Chinese, or by keys, written as symbols or words: `close`, `关闭`, `⌘W` and `cmd+w` all find Close tab. Menus, buttons and the command palette show the shortcut beside an action that has one.

Jun follows the conventions of Ghostty, iTerm2 and web browsers. Its shortcuts use ⌘, except ⌃Tab and ⌃⇧Tab, which cycle tabs. Every other ⌃ combination reaches programs running in the terminal, such as Claude Code, Codex and vim, unchanged.

The modifiers form a ladder: ⌘⇧ for tabs, ⌘⌥ for perspectives, ⌃⌘ for projects, with `[` and `]` for the previous and next one; digits select by position, with ⌘ for tabs. ⌘[ and ⌘] go back and forward, as in a browser.

Symbols: ⌘ Command, ⌥ Option, ⌃ Control, ⇧ Shift.

## Tabs

Tab shortcuts act on the active region: the side bar, column, bottom region or Global that holds keyboard focus. In Global they move between the views shown. The side bar’s tabs and Global’s views are fixed: there ⌘W does nothing, and ⌘T opens the terminal in the column or bottom region focused last, else the first column.

| Action | Shortcut |
|---|---|
| New terminal | ⌘T |
| Close tab (the terminal keeps running in the background) | ⌘W |
| Next / previous tab | ⌘⇧] / ⌘⇧[ or ⌃Tab / ⌃⇧Tab |
| Tab 1–8 | ⌘1–⌘8 |
| Last tab | ⌘9 |

Closing a region’s last tab leaves the region open. A tab’s context menu also shows ⌘W, but its Close Tab closes the tab clicked, while ⌘W closes the active one.

## Regions

| Action | Shortcut |
|---|---|
| Focus the region to the left / right | ⌘⌥← / ⌘⌥→ |
| Focus the region above / below | ⌘⌥↑ / ⌘⌥↓ |
| Show / hide the side bar | ⌘B |

Arrow keys move by position. The side bar sits to the left of the first column and the bottom region, Global to the right of the last column. The bottom region sits below the columns; ⌘⌥↑ from the bottom region focuses the first column, and so does ⌘⌥→ from the side bar. Arrows stop at the edge. Hiding the focused side bar focuses the first column. Each perspective keeps whether its side bar shows.

## Perspectives

| Action | Shortcut |
|---|---|
| Next / previous perspective | ⌘⌥] / ⌘⌥[ |
| Perspective 1–9 | ⌘⌥1–⌘⌥9 |
| New Perspective… | ⌘⇧N |

Perspectives are ordered as in the Perspective menu. Switching focuses the first column.

## Projects

| Action | Shortcut |
|---|---|
| Next / previous project | ⌃⌘] / ⌃⌘[ |
| Project 1–9 | ⌃⌘1–⌃⌘9 |
| Open Project… (the command palette at `~/`) | ⌘O |

Projects are ordered as in the project rail. Switching restores the project's selected perspective and focuses its first column.

## Back and forward

| Action | Shortcut |
|---|---|
| Back | ⌘[ |
| Forward | ⌘] |

Jun remembers where you have been: the tab you work in, in the column or bottom region focused last, across perspectives and projects. Each move there, by a click, a shortcut or the command palette, is one step, however much it switched on the way: showing a terminal of another project counts once. Focusing the side bar or Global, or peeking an agent, leaves you where you are.

Closing a tab is no step: back and forward skip closed tabs, and the steps ahead stay. Moving somewhere new after going back drops the steps ahead, as in a browser. At either end, back and forward do nothing. Jun keeps the last 100 steps back until it quits.

## Files

| Action | Shortcut |
|---|---|
| Search files in Files | ⌘P |
| In the search field: preview the first match, and move into the matches | ↓ |
| In the search field: open the first match in a tab of its own | Enter |

⌘P shows the side bar, chooses Files and focuses its search field. It does nothing until a project is open.

## Agents

| Action | Shortcut |
|---|---|
| Peek the next waiting agent; with none waiting, the current agent | ⌘J |
| Show the peeked agent's tab | ⌘↩ |
| Close the peek | ⌘W |

While an agent is peeked, Esc and every other key without ⌘ go to the agent. See [Agents](agents-view.md#peek).

## Text size

| Action | Shortcut |
|---|---|
| Larger text | ⌘+ or ⌘= |
| Smaller text | ⌘− |
| Default text size (13) | ⌘0 |

Terminals, file tabs and Messages share one text size, from 8 to 40. Jun saves it as `font.size` in the [settings file](settings.md).

## Command palette

| Action | Shortcut |
|---|---|
| Command palette | ⌘K |
| Review a branch's changes, in the palette | ⌘⇧D |
| Go to a heading of the Markdown tab, in the palette at `@` | ⌘⇧O |
| Move among items | ↑ / ↓ |
| Go into the highlighted directory | Tab |
| Run the highlighted item | Enter |
| Close | Esc |

## Terminal keys

The terminal keeps these macOS keys:

| Action | Shortcut |
|---|---|
| Copy / paste / select all | ⌘C / ⌘V / ⌘A |
| Scroll a page up / down | ⌘PageUp / ⌘PageDown or ⇧PageUp / ⇧PageDown |
| Scroll to the top / bottom | ⌘Home / ⌘End |

In a full-screen program, such as vim, which has no scrollback, the scrolling keys reach the program. ⌘K opens the command palette rather than clearing the screen; ⌃L clears it in most shells, which is the shell’s key, not Jun’s. ⌘⇧D reviews changes rather than splitting the terminal, and ⌘+, ⌘− and ⌘0 change Jun’s text size.

## Window and app

| Action | Shortcut |
|---|---|
| Settings | ⌘, |
| Save settings, in the settings tab | ⌘S |
| Undo / redo | ⌘Z / ⌘⇧Z |
| Cut / paste and match style | ⌘X / ⌘⌥⇧V |
| Full screen | ⌃⌘F |
| Minimize | ⌘M |
| Quit Jun | ⌘Q |

Undo, redo, cut and paste and match style come from the Edit menu and act on the text being edited. ⌘Q asks first while programs run in terminals; quitting ends the terminals.

## Rules

- Next and previous wrap around.
- A position beyond the last item does nothing.
- Perspective and project shortcuts do nothing until a project is open.
- While a dialog is open, such as the command palette or the Keyboard Shortcuts panel, Jun’s own shortcuts do nothing. The menu bar’s still reach their menus: ⌘B shows or hides the side bar, the Edit menu acts on the dialog’s text, and ⌘Q, ⌘M and ⌃⌘F work as ever. ⌘K, ⌘O and ⌘⇧D open the command palette behind the dialog, where it shows once the dialog closes.
