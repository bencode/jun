# Keyboard Shortcuts

Jun follows the conventions of Ghostty, iTerm2 and web browsers. Every shortcut uses ⌘, so ⌃ combinations reach programs running in the terminal, such as Claude Code, Codex and vim, unchanged.

The modifiers form a ladder: ⌘ for tabs, ⌘⌥ for perspectives, ⌃⌘ for projects. `[` and `]` move to the previous and next item; digits select by position.

## Tabs

Tab shortcuts act on the active region: the column, bottom region or Global that holds keyboard focus.

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

Arrow keys move by position. Global sits to the right of the last column. The bottom region sits below the columns; ⌘⌥↑ from the bottom region focuses the first column. Arrows stop at the edge. ⌘[ and ⌘] cycle through the visible regions in the order columns, bottom, Global, and wrap around.

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
| Open Project… | ⌘O |

Projects are ordered as in the project rail. Switching restores the project's selected perspective and focuses its first column.

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
| Clear screen | ⌘K |
| Larger / smaller / reset font | ⌘= / ⌘- / ⌘0 |
| Scroll page / to top or bottom | ⌘PageUp / ⌘PageDown / ⌘Home / ⌘End |

⌘Q quits Jun.
