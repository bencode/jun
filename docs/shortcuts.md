# Keyboard Shortcuts

Jun follows the conventions of Ghostty, iTerm2 and web browsers. Every shortcut uses ⌘, so ⌃ combinations reach programs running in the terminal, such as Claude Code, Codex and vim, unchanged.

The modifiers form a ladder: ⌘ for tabs, ⌘⌥ for perspectives, ⌃⌘ for projects. `[` and `]` move to the previous and next item; digits select by position.

## Tabs

Tab shortcuts act on the active region: the side bar, column, bottom region or Global that holds keyboard focus. The side bar’s and Global’s tabs are fixed: there ⌘W does nothing, and ⌘T opens the terminal in the column or bottom region focused last.

| Action | Shortcut |
|---|---|
| New terminal | ⌘T |
| Close tab (the shell keeps running in the background) | ⌘W |
| Next / previous tab | ⌘⇧] / ⌘⇧[ or ⌃Tab / ⌃⇧Tab |
| Tab 1–8 | ⌘1–⌘8 |
| Last tab | ⌘9 |

## Regions

| Action | Shortcut |
|---|---|
| Focus the region to the left / right | ⌘⌥← / ⌘⌥→ |
| Focus the region above / below | ⌘⌥↑ / ⌘⌥↓ |
| Previous / next region | ⌘[ / ⌘] |
| Show / hide the side bar | ⌘B |

Arrow keys move by position. The side bar sits to the left of the first column and the bottom region, Global to the right of the last column. The bottom region sits below the columns; ⌘⌥↑ from the bottom region focuses the first column, and so does ⌘⌥→ from the side bar. Arrows stop at the edge. ⌘[ and ⌘] cycle through the visible regions in the order side bar, columns, bottom, Global, and wrap around. Hiding the focused side bar focuses the first column.

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

## Command palette

| Action | Shortcut |
|---|---|
| Command palette | ⌘K |
| Move among items | ↑ / ↓ |
| Go into the highlighted directory | Tab |
| Run the highlighted item | Enter |
| Close | Esc |

## Rules

- Next and previous wrap around.
- A position beyond the last item does nothing.
- Perspective and project shortcuts do nothing until a project is open.
- Shortcuts are disabled while a dialog is open.

## Terminal keys

The terminal keeps these macOS keys:

| Action | Shortcut |
|---|---|
| Copy / paste / select all | ⌘C / ⌘V / ⌘A |
| Larger / smaller / reset font | ⌘= / ⌘- / ⌘0 |
| Scroll page / to top or bottom | ⌘PageUp / ⌘PageDown / ⌘Home / ⌘End |

⌘K opens the command palette rather than clearing the screen; ⌃L clears it in most shells.

⌘Q quits Jun.
