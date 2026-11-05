# Workspace: projects, perspectives, regions and tabs

How Jun's window is organized, and what each part does.

## Projects

A project is a local directory. The project rail, at the window's left edge, lists them.

- **Open a project**
  - ⌘K, then type a directory such as `~/work/`. Tab goes into the highlighted directory, and Enter opens it.
  - ⌘O, **File → Open Project…** or the rail's **+** open the palette at `~/`. **Browse…** opens the system folder picker.
  - Drag a directory from Finder onto the rail, one at a time. A file dragged to a region opens in a tab instead: see [Files](files.md#files-outside-the-project).
  - From a terminal, run `jun <directory>`, such as `jun .`. This needs the [`jun` alias](../README.md#optional-jun-command).
- **Name**: the directory's name. Jun asks for another one only when that name is already taken.
- **Already open**: opening the same directory again selects its project.
- **A new project** starts with one perspective, **Work**, which shows its side bar and one terminal in the project's directory.
- **Rail tile**: a colored tile with two characters.
  - The characters are the initials of the name's first two words; words are split at separators, camelCase and digits (`cc-mission-control` → CM, `jun2` → J2).
  - With one word, they are its first two letters (`jun` → Ju).
  - The color follows the name.
- **Expanded rail** (click the Jun logo at the rail's top, or **Toggle Project Names** in ⌘K): adds each project's name and directory. Home shows as `~`, and long paths are cut to their last two parts.
- **Waiting agents**: a project whose agents wait for you shows an orange dot on its tile, and their number when the rail is expanded.
- **No open panels**: a project with no tabs in any perspective appears dimmed until hovered or selected. Fixed side-bar views and terminals without an open tab do not count; projects with waiting agents stay bright.
- **Right-click a project**
  - **Rename Project…**: keeps its perspectives, tabs and terminals. The same name rules apply as when opening.
  - **Remove Project…**: always asks first. It closes all of the project's terminals, the agents running in them, and its views in other places. The directory stays on disk.
- **One window**: launching Jun again focuses the window it already has.

## Perspectives

A perspective is a named layout within a project, such as one for working and one for reviewing.

- **Perspective menu**: the name at the right end of the last column's tab bar. It lists the project's perspectives in creation order.
- **New Perspective…**
  - Suggests the first of Work, Review, Monitor, Debug and Notes that the project does not have yet, then "Perspective" with a number.
  - A new perspective starts with one empty column and no terminal.
- **Rename Perspective…**: keeps its tabs, layout and terminals.
- **Delete Perspective…**: selects a neighboring perspective. When the perspective shows terminals, Jun asks first, then closes them with any agents running in them. A project keeps at least one perspective.
- **Names** must not be empty and must be unique within the project, ignoring case. Other projects can use the same names.
- **Switching projects** returns to the perspective that project had selected.

## Regions

| Region | Where | Holds | Belongs to |
|---|---|---|---|
| Columns (1–3) | center | tabs: terminals, files, sessions, diffs, settings | the perspective |
| Bottom | below the columns | tabs, like a column | the perspective |
| Side bar | left of the columns | **Sessions**, **Files** (fixed) | the perspective |
| Global | right of the columns | **Agents** (fixed), with [Messages](agents-view.md#messages) below it | every project |

- **Columns**: each column's **⋯** offers **Add Column to Right** and **Remove This Column**.
  - Removing a column moves its tabs to the column on its left; the first column's tabs go to the right instead. Its terminals keep running.
- **Fixed regions**: the side bar's and Global's tabs cannot be closed or moved, and other tabs cannot be moved into them. Their **⋯** only hides the region, and in Global, [Messages](agents-view.md#messages).
- **Side bar**: ⌘B shows or hides it. Drag its edge to resize it.
  - It keeps its width as the window or Global changes.
  - When the columns would get narrower than their minimum, it gives way, and gets its width back once there is room.
- **View bar**: at the right end of the last column's tab bar.
  - The perspective menu, then three switches: side bar, bottom region and Global.
  - Each switch is lit while its region shows. Using them leaves the keyboard focus where it was.
  - In a narrow column, the perspective's name gives way, down to its icon.
- **Empty regions** stay open, and offer the **Open View** menu.

## Tabs

- **Open**
  - **⋯ → Open View** offers **New Terminal**.
  - ⌘T opens a terminal in the active region. From the side bar or Global, it goes to the column or bottom region you focused last.
- **Select**: click the tab, or use ⌃Tab / ⌃⇧Tab, ⌘1–⌘8, or ⌘9 for the last tab.
- **Close**: the tab's **×**, ⌘W, or right-click → **Close Tab**.
  - Closing a terminal's tab keeps its shell running; see [Terminals](terminals.md).
  - Closing the active tab selects the tab to its right, else the one to its left.
- **Move**
  - Drag a tab to reorder it, or drop it in another region.
  - Right-click → **Move Tab To** lists every column and bottom region by project and perspective. Moving a tab into a hidden region shows that region.
- **Right-clicking an inactive tab** does not select it; the action applies to that tab.
- **Preview tabs**, for files and sessions, show their title in italics. The next preview replaces them; a double click or Enter keeps the file or session in a tab of its own.

## Focus

- The active region is the one with the keyboard focus, including the side bar, the bottom region and Global.
- Switching project or perspective focuses its first column.
- Showing the bottom region gives the focus back to the region that had it. Hiding the active region focuses the first column.

See [Keyboard shortcuts](shortcuts.md) for every shortcut.
