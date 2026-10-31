# MyGo Dependency

Jun builds on [MyGo](https://github.com/egoist/mygo). Until a fix is released upstream, `go.mod` replaces it with a fork that carries the fix on top of the release Jun uses:

```text
replace github.com/egoist/mygo => github.com/bencode/mygo <v0.7.3 + fix>
```

Import paths stay `github.com/egoist/mygo`, and `go tool mygo` comes from the fork too.

## The fix

The terminal view kept glyphs shaped with fonts that MyGo's text system had let go of (it does so once more than 512 font sizes pile up), and drew them again. The app crashed in Core Text. The fix reshapes the view's rows when the text system's generation changes. It is 9 lines in `plugins/terminal`, plus a test (`TestRowsAfterFontsLetGo`).

| Branch in `bencode/mygo` | Based on | Purpose |
|---|---|---|
| `fix/terminal-forgotten-fonts` | `v0.7.3` | what Jun uses |
| `terminal-fonts-let-go` | upstream `main` | the pull request to `egoist/mygo` |

## Upgrading MyGo

1. If the new release already includes the fix, delete the `replace` line and run `go get github.com/egoist/mygo@<version>`. The fork is then no longer needed.
2. Otherwise, rebase the fix onto the new version on a new branch (no force-push), then point the `replace` at it and update the table above:

```sh
# in a clone of bencode/mygo, with egoist/mygo as the "upstream" remote
git fetch upstream --tags
git checkout -b fix/terminal-fonts-<version> <branch Jun uses now>
git rebase --onto <version> <version Jun uses now>
go test ./plugins/terminal
git push -u origin fix/terminal-fonts-<version>

# in Jun
go mod edit -replace=github.com/egoist/mygo=github.com/bencode/mygo@<commit>
go mod tidy && go test ./...
```

To follow upstream `main` instead of a release, use `upstream/main` in place of `<version>`.
