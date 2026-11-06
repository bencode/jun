package layout

import (
	"fmt"
	"slices"
)

// GlobalArrange is how Global shows its views.
type GlobalArrange string

const (
	GlobalSide    GlobalArrange = ""        // side by side, each as high as Global
	GlobalStacked GlobalArrange = "stacked" // one above the other, each as wide as Global
)

// globalShares are the views' weights until the user drags them.
var globalShares = map[ViewKind]float32{ViewAgents: 60, ViewMessages: 40}

// maximumShare is far past what drags reach, for the weights to add up.
const maximumShare float32 = 1e6

// minimumShare is the part of two neighbors a drag leaves the smaller one.
const minimumShare float32 = 0.01

// shown is the views of a fixed region it shows, in order.
func shown(views []*View) []*View {
	return slices.DeleteFunc(slices.Clone(views), func(v *View) bool { return !v.Show })
}

// Shown is what the region shows: a fixed region those chosen, tabs all.
func (s *State) Shown(r *Region) []*View {
	if s.Fixed(r.ID) {
		return shown(r.Views)
	}
	return slices.Clone(r.Views)
}

// GlobalViews is what Global shows, in order.
func (s *State) GlobalViews() []*View { return shown(s.Global.Views) }

// ToggleGlobalView shows or hides a view of Global. With Global hidden, it
// shows Global with the view. The last view shown hides Global instead, and
// comes back with it.
func (s *State) ToggleGlobalView(id ViewID) error {
	i := slices.IndexFunc(s.Global.Views, func(v *View) bool { return v.ID == id })
	if i < 0 {
		return fmt.Errorf("view %d is not in Global", id)
	}
	v := s.Global.Views[i]
	if !s.GlobalVisible {
		s.GlobalVisible, v.Show = true, true
		return nil
	}
	if views := s.GlobalViews(); len(views) == 1 && views[0] == v {
		s.GlobalVisible = false
		return nil
	}
	v.Show = !v.Show
	s.settleGlobal()
	return nil
}

// ToggleGlobalArrange switches between side by side and stacked.
func (s *State) ToggleGlobalArrange() {
	s.GlobalArrange = map[GlobalArrange]GlobalArrange{GlobalSide: GlobalStacked, GlobalStacked: GlobalSide}[s.GlobalArrange]
}

// TotalShare is the views' weights together.
func TotalShare(views []*View) float32 {
	total := float32(0)
	for _, v := range views {
		total += v.Share
	}
	return total
}

// FirstShare is the percentage of the views' room the first of them takes.
func FirstShare(views []*View) float32 { return 100 * views[0].Share / TotalShare(views) }

// ResizeGlobalView gives the first of the views this percentage of their
// room, moving it from or to the second alone, so that a divider changes
// only the views on either side of it.
func ResizeGlobalView(views []*View, percent float32) {
	pair := views[0].Share + views[1].Share
	first := min(max(percent/100*TotalShare(views), minimumShare*pair), (1-minimumShare)*pair)
	views[0].Share, views[1].Share = first, pair-first
}

// settleGlobal keeps a view shown, as a preset that dropped the only one
// shown would not, and moves the active view off a hidden one.
func (s *State) settleGlobal() {
	if len(s.Global.Views) == 0 {
		return
	}
	if len(s.GlobalViews()) == 0 {
		s.Global.Views[0].Show = true
	}
	views := s.GlobalViews()
	if !slices.ContainsFunc(views, func(v *View) bool { return v.ID == s.Global.Active }) {
		s.Global.Active = views[0].ID
	}
}

// fillGlobalShares gives the views without one their default share.
func (s *State) fillGlobalShares() {
	for _, v := range s.Global.Views {
		if v.Share == 0 {
			v.Share = globalShares[v.Kind]
		}
	}
}

// moveMessagesFields moves what earlier versions kept on the state to
// Global's views.
func (s *State) moveMessagesFields() {
	for _, v := range s.Global.Views {
		if v.Kind == ViewAgents && s.AgentsPercent != 0 {
			v.Share = s.AgentsPercent
		}
		if v.Kind == ViewMessages {
			v.Show = v.Show && !(s.MessagesHidden && len(s.GlobalViews()) > 1)
			if s.AgentsPercent != 0 {
				v.Share = 100 - s.AgentsPercent
			}
		}
	}
	s.AgentsPercent, s.MessagesHidden = 0, false
}

// validGlobal checks what only fixed regions' views have.
func (s *State) validGlobal() error {
	if !slices.Contains([]GlobalArrange{GlobalSide, GlobalStacked}, s.GlobalArrange) {
		return fmt.Errorf("unknown globalArrange %q", s.GlobalArrange)
	}
	for _, r := range s.regions() {
		fixed := s.Fixed(r.ID)
		for _, v := range r.Views {
			if r != s.Global && v.Share != 0 {
				return fmt.Errorf("view %d is not in Global but has its share", v.ID)
			}
			if !fixed && v.Show {
				return fmt.Errorf("view %d is a tab, which always shows", v.ID)
			} else if fixed && r != s.Global && !v.Show && slices.Contains(shownKinds, v.Kind) {
				return fmt.Errorf("view %d of a side region must show", v.ID)
			}
			if v.Share < 0 || v.Share > maximumShare {
				return fmt.Errorf("view %d: share must be between 0 and %g", v.ID, maximumShare)
			}
		}
	}
	if len(s.Global.Views) > 0 && len(s.GlobalViews()) == 0 {
		return fmt.Errorf("Global shows no view")
	}
	return nil
}
