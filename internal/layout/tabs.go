package layout

import (
	"path/filepath"

	"fmt"
	"github.com/bencode/jun/internal/history"
	"path"
	"slices"
	"strings"
)

func (s *State) OpenView(target RegionID, kind ViewKind, projectName string) (ViewID, error) {
	r := s.Region(target)
	if r == nil {
		return 0, fmt.Errorf("region not found")
	}
	if s.Fixed(target) {
		return 0, errFixed
	}
	if kind == ViewFile || kind == ViewTranscript || kind == ViewDiff || !slices.Contains(tabKinds, kind) {
		return 0, fmt.Errorf("view kind %s cannot be opened in a tab", kind)
	}
	if !slices.ContainsFunc(s.Projects, func(p *Project) bool { return p.Name == projectName }) {
		return 0, fmt.Errorf("source project not found: %s", projectName)
	}
	id := s.appendView(r, kind, projectName)
	s.reveal(r)
	return id, nil
}

// OpenFile shows the file in the target region: the view there showing it,
// else the preview view, which it replaces, else a new view.
func (s *State) OpenFile(target RegionID, project string, ref FileRef, preview ViewID) (ViewID, error) {
	r := s.Region(target)
	if r == nil {
		return 0, fmt.Errorf("region not found")
	}
	if s.Fixed(target) {
		return 0, errFixed
	}
	if !slices.ContainsFunc(s.Projects, func(p *Project) bool { return p.Name == project }) {
		return 0, fmt.Errorf("source project not found: %s", project)
	}
	if !validFilePath(ref.Path) || ref.Source == "" {
		return 0, fmt.Errorf("invalid file %q in %q", ref.Path, ref.Source)
	}
	shows := func(v *View) bool { return v.Kind == ViewFile && v.ProjectName == project && *v.File == ref }
	isPreview := func(v *View) bool { return v.ID == preview && v.Kind == ViewFile }
	if i := slices.IndexFunc(r.Views, shows); i >= 0 {
		r.Active = r.Views[i].ID
	} else if i := slices.IndexFunc(r.Views, isPreview); i >= 0 {
		r.Views[i].ProjectName, r.Views[i].File = project, &ref
		r.Active = preview
	} else {
		s.appendView(r, ViewFile, project)
		r.Views[len(r.Views)-1].File = &ref
	}
	s.reveal(r)
	return r.Active, nil
}

// OpenTranscript shows the session in the target region: the view there
// showing it, else the preview view, which it replaces when that is a
// transcript too, else a new view.
func (s *State) OpenTranscript(target RegionID, project string, ref SessionRef, preview ViewID) (ViewID, error) {
	r := s.Region(target)
	if r == nil {
		return 0, fmt.Errorf("region not found")
	}
	if s.Fixed(target) {
		return 0, errFixed
	}
	if !slices.ContainsFunc(s.Projects, func(p *Project) bool { return p.Name == project }) {
		return 0, fmt.Errorf("source project not found: %s", project)
	}
	if err := validSession(ref); err != nil {
		return 0, err
	}
	shows := func(v *View) bool { return v.Transcript != nil && v.ProjectName == project && *v.Transcript == ref }
	isPreview := func(v *View) bool { return v.ID == preview && v.Kind == ViewTranscript }
	if i := slices.IndexFunc(r.Views, shows); i >= 0 {
		r.Active = r.Views[i].ID
	} else if i := slices.IndexFunc(r.Views, isPreview); i >= 0 {
		r.Views[i].ProjectName, r.Views[i].Transcript = project, &ref
		r.Active = preview
	} else {
		s.appendView(r, ViewTranscript, project)
		r.Views[len(r.Views)-1].Transcript = &ref
	}
	s.reveal(r)
	return r.Active, nil
}

// validSession checks a session reference: an agent, a plain ID and an
// absolute file.
func validSession(ref SessionRef) error {
	if !history.Known(ref.Agent) || !history.ValidID(ref.ID) || !filepath.IsAbs(ref.Path) {
		return fmt.Errorf("invalid session %q of %q", ref.ID, ref.Agent)
	}
	return nil
}

// OpenDiff shows the comparison in the target region: in the Diff view
// there of the same source, else in a new one.
func (s *State) OpenDiff(target RegionID, project string, ref DiffRef) (ViewID, error) {
	r := s.Region(target)
	if r == nil {
		return 0, fmt.Errorf("region not found")
	}
	if s.Fixed(target) {
		return 0, errFixed
	}
	if !slices.ContainsFunc(s.Projects, func(p *Project) bool { return p.Name == project }) {
		return 0, fmt.Errorf("source project not found: %s", project)
	}
	if err := validDiff(ref); err != nil {
		return 0, err
	}
	shows := func(v *View) bool {
		return v.Kind == ViewDiff && v.ProjectName == project && v.Diff != nil && v.Diff.Source == ref.Source
	}
	if i := slices.IndexFunc(r.Views, shows); i >= 0 {
		r.Active = r.Views[i].ID
	} else {
		s.appendView(r, ViewDiff, project)
		r.Views[len(r.Views)-1].Diff = &ref
	}
	s.reveal(r)
	return r.Active, nil
}

func validDiff(ref DiffRef) error {
	hex := func(s string) bool { return strings.Trim(strings.ToLower(s), "0123456789abcdef") == "" }
	if ref.Source == "" || ref.Base != "" && (strings.HasPrefix(ref.Base, "-") || strings.ContainsAny(ref.Base, " \t\n\x00")) ||
		ref.Commit != "" && (len(ref.Commit) < 4 || len(ref.Commit) > 64 || !hex(ref.Commit)) || ref.Uncommitted && ref.Commit != "" {
		return fmt.Errorf("invalid diff of %q", ref.Source)
	}
	return nil
}

// validFilePath reports whether name is a slash-separated path relative to
// a repository's top level, inside it.
func validFilePath(name string) bool {
	return name != "" && path.Clean(name) == name && !path.IsAbs(name) && name != ".." && !strings.HasPrefix(name, "../")
}

var errFixed = fmt.Errorf("views of Global and side regions cannot be opened, moved or closed")

func (s *State) SelectView(region RegionID, view ViewID) error {
	r := s.Region(region)
	if r == nil || !slices.ContainsFunc(r.Views, func(v *View) bool { return v.ID == view }) {
		return fmt.Errorf("view is not in region")
	}
	r.Active = view
	return nil
}

func (s *State) locate(view ViewID) (*Region, int) {
	for _, r := range s.regions() {
		if i := slices.IndexFunc(r.Views, func(v *View) bool { return v.ID == view }); i >= 0 {
			return r, i
		}
	}
	return nil, -1
}

func (s *State) MoveView(view ViewID, target RegionID, before ViewID) error {
	source, i := s.locate(view)
	destination := s.Region(target)
	if source == nil || destination == nil {
		return fmt.Errorf("view or target region not found")
	}
	if s.Fixed(source.ID) || s.Fixed(destination.ID) {
		return errFixed
	}
	if before != 0 && !slices.ContainsFunc(destination.Views, func(v *View) bool { return v.ID == before }) {
		return fmt.Errorf("destination tab not found")
	}
	if before == view {
		return nil
	}
	v := source.Views[i]
	removeView(source, i)
	index := len(destination.Views)
	if before != 0 {
		index = slices.IndexFunc(destination.Views, func(v *View) bool { return v.ID == before })
	}
	destination.Views = slices.Insert(destination.Views, index, v)
	destination.Active = view
	s.reveal(destination)
	return nil
}

func (s *State) CloseView(view ViewID) error {
	r, i := s.locate(view)
	if r == nil {
		return fmt.Errorf("view not found")
	}
	if s.Fixed(r.ID) {
		return errFixed
	}
	removeView(r, i)
	return nil
}

func removeView(r *Region, i int) {
	active := r.Views[i].ID == r.Active
	r.Views = slices.Delete(r.Views, i, i+1)
	if !active {
		return
	}
	r.Active = 0
	if len(r.Views) > 0 {
		r.Active = r.Views[min(i, len(r.Views)-1)].ID
	}
}

func (s *State) reveal(r *Region) {
	if r == s.Global {
		s.GlobalVisible = true
	}
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			if r == perspective.Bottom {
				perspective.BottomVisible = true
			}
		}
	}
}

// RevealView selects the view in its region and shows the region: the
// region's project and perspective become current, and a hidden bottom
// region or Global appears.
func (s *State) RevealView(view ViewID) (RegionID, error) {
	r, _ := s.locate(view)
	if r == nil {
		return 0, fmt.Errorf("view not found")
	}
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			if r == perspective.Bottom || slices.Contains(perspective.Columns, r) {
				s.ActiveProject, p.ActivePerspective = p.Name, perspective.ID
			}
		}
	}
	r.Active = view
	s.reveal(r)
	return r.ID, nil
}
