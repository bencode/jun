package app

import (
	"errors"
	"maps"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
)

func (a *app) reapExitedTerminals(focused layout.ViewID) bool {
	closed, cleanupErr := a.terminals.ReapExited()
	if len(closed) == 0 && cleanupErr == nil {
		return false
	}
	maps.DeleteFunc(a.leads, func(led, lead terminal.ID) bool { return slices.Contains(closed, led) || slices.Contains(closed, lead) })
	maps.DeleteFunc(a.names, func(id terminal.ID, _ string) bool { return slices.Contains(closed, id) })
	var destination layout.RegionID
	modal := a.palette.open || a.projectForm.open || a.perspectiveForm.open || a.projectRename.open || a.confirmation.open || a.errorOpen ||
		a.persistence != nil && a.persistence.errorOpen
	if !modal && focused != 0 && slices.Contains(closed, a.bindings[focused]) {
		if r := a.targetRegion(); r != nil && r.Active == focused {
			destination = r.ID
		}
	}
	errs := []error{cleanupErr}
	changed := false
	for view, session := range a.bindings {
		if !slices.Contains(closed, session) {
			continue
		}
		if err := a.state.CloseView(view); err != nil {
			errs = append(errs, err)
		} else {
			changed = true
		}
		delete(a.bindings, view)
		delete(a.scroll, view)
		delete(a.terminalErrors, view)
	}
	if changed {
		a.layoutChanged()
	}
	if destination != 0 {
		a.focusRegion(destination)
	}
	err := errors.Join(errs...)
	a.report(err)
	return changed || err != nil
}
