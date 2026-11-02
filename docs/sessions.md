# Sessions

**Sessions**, the side bar's first tab, lists the project's Claude Code and Codex sessions. Open one to read it, or resume it in a terminal.

## Which sessions

- Sessions started in the project's directory, or in one of its worktrees.
- **Claude Code**: read from the folders in `claude.configDirs` in the [settings file](settings.md).
  - With none set, Jun reads Claude Code's own folder: `CLAUDE_CONFIG_DIR` from Jun's environment, else `~/.claude`.
  - An app opened from the Dock does not see variables that a shell or a launcher sets. Set `claude.configDirs` when Claude Code keeps its sessions elsewhere.
- **Codex**: read from `CODEX_HOME`, else `~/.codex`.
- **Left out**: subagents' sessions, and sessions in which nothing was asked.
- **Reading**: Jun reads only the start and end of each session's file, and reads a file again only when it changes.
  - The list is read again when it is older than 30 seconds, or when the project's worktrees change.
  - **⟳** reads it again at once.

## The list

- Grouped by the day each session was last active, the latest first. At most the latest 500 show.
- **A row**

| Part | Shows |
|---|---|
| Icon | the agent's own mark: Claude Code or Codex |
| First line | the session's title: the agent's, else the first prompt's first line |
| Second line | `subdirectory · branch · time · duration` |
| Tooltip | the last prompt's first line, the session's ID, its directory |

- The second line leaves out what is unknown or does not apply:
  - The subdirectory is the session's directory within the project. It is left out at the project's own directory, or outside it.
  - The branch is the one the session started on.
  - The time is when the session was last active.
  - The duration runs from the session's start to its last activity, such as `25m`, `2h` or `1h 10m`.
- **Search**: the field above the list searches every session, not only the 500 shown. It keeps those whose title, first prompt or last prompt holds every word you type, ignoring case, again at most 500.

## Reading a session

- **Click** a session to show it read-only in a preview tab. **Double click** or **Enter** keeps it in a tab of its own.
- **The tab** shows:
  - the prompts and answers, the latest at the bottom;
  - between them, the number of tool steps the agent took;
  - in its header, the agent, the branch, the number of messages, and when the session started and was last active.
- The tab is read again as the session goes on.

## Resume and other actions

- **Resume**, in the session tab's header or a session's context menu:
  - opens a terminal in the session's directory;
  - runs `claude --resume <id>` or `codex resume <id>` there.
- **Claude Code sessions from another folder** resume with `CLAUDE_CONFIG_DIR` set to that folder, such as `CLAUDE_CONFIG_DIR='…' claude --resume <id>`. The default folder needs no prefix.
- **Copy Session ID**, in the header or the context menu, copies the session's ID.
- **A directory that no longer exists** cannot be resumed, and Jun says why.
