# Jun

A native, GPU-rendered workspace for working with terminal agents. Built with Go and [MyGo](https://github.com/egoist/mygo), without a WebView.

I01 provides the layout skeleton on macOS: projects rooted at local directories, Work/Review perspectives, one to three columns, a shared bottom region, and an independent Global region. Each region holds movable, closable tabs. Projects and layout changes live in memory during the app session.

View content is explicitly a placeholder. Real terminals arrive in I02, persistence and command entrypoints in I03, Agent overview in I04, and file/diff Review in I05.

## Development

Requires macOS, Xcode command line tools, and Go 1.27.1 or newer.

```sh
go run ./cmd/jun
go test ./...
go vet ./...
go tool mygo build -platform darwin/arm64 -skip-dmg
```

Open a project through the rail’s **+** button or **File → Open Project…**. Confirm the directory’s name, switch Work/Review using the framed selector in the first tab strip, and use each region’s **⋯** menu to open a Terminal placeholder or change the layout. Drag tabs to reorder or move between regions. Moving to a hidden bottom/Global region through the menu reveals it. The upper-right center control hides or restores Global.

The **Language** menu switches between English (default) and Simplified Chinese for the current session.

## Structure

```text
cmd/jun          application entrypoint
internal/app     native window, UI composition, menus and directory picker
internal/layout  pure Go project/perspective/region/view model
```

Dependencies flow from `cmd/jun` to `app`, then from `app` to `layout` and MyGo. `layout` uses only the standard library. There is no dedicated filesystem sidebar: a file tree is an ordinary view in a perspective.

Planning documents, the frozen HTML prototype, research and local review artifacts belong under the ignored `priv/` directory. Generated `.mygo/` and `build/` directories are also ignored.
