package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bencode/jun/internal/layout"
)

// fileAt is the file at the absolute path: of the source of the project's
// repository it is in, else of its own directory, which file views read as
// a worktree.
func (a *app) fileAt(project, path string) fileNode {
	if p := a.project(project); p != nil {
		s, ok := sourceAt(a.repoFor(p.CWD).sources, filepath.Dir(path))
		if name, err := filepath.Rel(s.Worktree, path); ok && err == nil {
			return fileNode{source: s.Key(), path: filepath.ToSlash(name)}
		}
	}
	return fileNode{source: filepath.Dir(path), path: filepath.Base(path)}
}

// showFile opens the file at the absolute path in a tab of its own, where
// new tabs go, or reads it again in the tab that shows it.
func (a *app) showFile(project, path string) error {
	if err := a.checkFile(path); err != nil {
		return err
	}
	target := a.tabsTarget()
	if target == nil {
		return errors.New(a.text("no region to open the file in", "没有可以打开文件的区域"))
	}
	a.openFile(project, a.fileAt(project, path), true)
	if v := activeView(target); v != nil && a.fileViews[v.ID] != nil {
		a.fileViews[v.ID].read = false
	}
	a.focusRegion(target.ID)
	return nil
}

// checkFile tells why the path is not a file Jun can open.
func (a *app) checkFile(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf(a.text("no file %s", "没有这个文件：%s"), path)
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf(a.text("%s is a directory", "%s 是目录"), path)
	}
	return nil
}

// openDroppedFiles opens files dropped on the region from Finder.
func (a *app) openDroppedFiles(region layout.RegionID, paths []string) {
	a.focusRegion(region)
	for _, path := range paths {
		if err := a.showFile(a.state.ActiveProject, path); err != nil {
			a.report(err)
		}
	}
}
