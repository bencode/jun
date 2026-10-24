# Jun

A native, GPU-rendered workspace for working with terminal agents. Built with Go and [MyGo](https://github.com/egoist/mygo), without a WebView.

Jun currently provides the macOS layout skeleton and configuration recovery: projects rooted at local directories, Work/Review perspectives, one to three columns, a shared bottom region, and an independent Global region. Each region holds movable, closable tabs.

Opening a new project starts Work with one Terminal placeholder. Review starts with one empty column; the bottom and Global regions start empty and hidden. Real terminals arrive in I02, command entrypoints in I03, Agent overview in I04, and file/diff Review in I05.

## Development

Requires macOS, Xcode command line tools, and Go 1.27.1 or newer. Use an isolated development config so development and packaged builds do not share saved layouts:

```sh
JUN_CONFIG_PATH="$PWD/priv/config.dev.json" go run ./cmd/jun
go test ./...
go vet ./...
go tool mygo build -platform darwin/arm64 -skip-dmg
```

Open a project through the rail’s **+** button or **File → Open Project…**. Confirm the directory’s name, switch Work/Review using the framed selector in the first tab strip, and use each region’s **⋯** menu to open a Terminal placeholder or change the layout. Drag tabs to reorder or move between regions. Moving to a hidden bottom/Global region through the menu reveals it. The upper-right center control hides or restores Global.

The **Language** menu switches between English (default) and Simplified Chinese for the current session. Repeated launches of the same build focus its existing window; New Window is deferred to a separate iteration.

## Saved configuration

The default macOS file is `~/Library/Application Support/Jun/config.json`. `JUN_CONFIG_PATH` overrides the file location; relative paths resolve against the launch directory. MyGo distinguishes development and packaged builds for its single-instance behavior, so use the override when running them alongside each other.

The JSON document contains `version: 1` and a `layout` snapshot. It saves the ordered projects, names/CWDs, selected project by name, each perspective’s columns and bottom region, tab order and selection, split percentages, project rail expansion, and the independent Global region. Global views retain their source project when switching projects. Hidden regions are saved too.

Changes save after 300 ms of idle time, and pending changes flush when the window loses focus, hides, or closes. Writes use a same-directory temporary file and atomic replacement. Startup restores an existing snapshot instead of replacing it with defaults. A missing project directory preserves the project and layout and reports the problem when entering that project.

An unreadable, invalid, or unsupported configuration is preserved; Jun reports the error and disables saving for that run. Fix the file and restart. Other save failures preserve pending changes and offer Retry Save or Continue Editing. If saving fails during exit, the window stays open with Retry Save, Cancel Quit, and Quit Without Saving choices. Continuing editing pauses retries until another layout change or an exit attempt.

Only layout descriptions are restored. GPU/UI objects, scroll offsets, language choice, native window geometry, and terminal processes are not saved. I01.5 restores placeholders; terminal process recovery is not implemented. Configuration is read at startup; quit Jun before editing it manually. Earlier `state.json`, `settings.json`, and `window-state.json` files are left untouched.

## Structure

```text
cmd/jun          application entrypoint
internal/app     native UI, directory picker, save scheduling and error handling
internal/config  versioned JSON file loading and atomic replacement
internal/layout  pure Go project/perspective/region/view model and validation
```

Dependencies flow from `cmd/jun` to `app`, then from `app` to `config`, `layout`, and MyGo. `config` uses `layout` and standard-library JSON/filesystem APIs; `layout` uses only the standard library. There is no dedicated filesystem sidebar: a file tree is an ordinary view in a perspective.

Planning documents, the frozen HTML prototype, research and local review artifacts belong under the ignored `priv/` directory. Generated `.mygo/` and `build/` directories are also ignored.
