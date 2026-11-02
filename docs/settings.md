# Settings

Jun keeps the user's preferences in one YAML file, `~/.config/jun/settings.yaml`. The `JUN_SETTINGS_PATH` environment variable names another file, for development builds and tests. The layout — projects, perspectives, tabs, splits — stays in Jun's own configuration and is not a setting.

## Editing

Jun ▸ Settings… (⌘,) or Open Settings File in the rail's Settings menu opens the file in a tab of the current perspective. Before the file exists, the tab offers a commented template. ⌘S or Save writes the file, and the settings apply at once.

Jun also follows the file when another editor changes it. While the tab has unsaved changes of its own, it keeps them and says the file changed on disk; Reload takes the file's version. Closing the last settings tab with unsaved changes asks first.

A file with an error is not applied: Jun keeps the settings it last read and shows the error at the top of the settings tab. Unknown keys are errors, so a misspelled key does not pass unnoticed.

## Keys

```yaml
language: zh       # en | zh; en when unset
theme: Dracula     # a Ghostty theme's name; Jun's own when unset
font:
  size: 14         # terminals' and file tabs' text, 8 to 40; 13 when unset
claude:
  configDirs:      # Claude Code configuration folders Sessions lists
    - ~/.claude
    - ~/work/claude-accounts/team
```

`claude.configDirs` lists the folders whose sessions Sessions shows, each the folder Claude Code would be given in `CLAUDE_CONFIG_DIR`. When it is unset, Sessions reads Claude Code's own: `CLAUDE_CONFIG_DIR` from Jun's environment, else `~/.claude`. An app opened from the Dock does not see variables a shell or a launcher sets, which is what the setting is for. Resuming a session from another folder starts Claude Code with `CLAUDE_CONFIG_DIR` set to it.

## Quick entries

Some settings have their own controls, which write the file, changing only their key and keeping the rest, comments included:

| Setting | Control |
|---|---|
| `language` | Settings ▸ Language, or the Language menu |
| `theme` | Settings ▸ Theme… |
| `font.size` | ⌘+ / ⌘− / ⌘0 |

Earlier versions kept the language, theme and text size in the layout. The first start moves them into the settings file, unless it already has its own.
