package app

import (
	"errors"
	"slices"

	"github.com/bencode/jun/internal/layout"
)

func (a *app) reapExitedTerminals(focused layout.ViewID) bool {
	closed, cleanupErr := a.terminals.ReapSuccessful()
	if len(closed) == 0 && cleanupErr == nil {
		return false
	}
	var destination layout.RegionID
	modal := a.projectForm.open || a.perspectiveForm.open || a.errorOpen ||
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
