package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/bencode/jun/internal/layout"
)

func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("expected one JSON document")
		}
		return err
	}
	return nil
}

func decodeLayout(version int, data json.RawMessage) (*layout.State, error) {
	if version == 1 {
		return migrateV1(data)
	}
	if version != 2 {
		return nil, fmt.Errorf("unsupported version %d", version)
	}
	var state *layout.State
	if err := decodeJSON(data, &state); err != nil {
		return nil, err
	}
	if err := state.Restore(); err != nil {
		return nil, err
	}
	return state, nil
}

type legacyState struct {
	Projects      []*legacyProject `json:"projects"`
	ActiveProject string           `json:"activeProject"`
	Global        *layout.Region   `json:"global"`
	GlobalVisible bool             `json:"globalVisible"`
	RailExpanded  bool             `json:"railExpanded"`
	CenterPercent float32          `json:"centerPercent"`
}

type legacyProject struct {
	Name              string                        `json:"name"`
	CWD               string                        `json:"cwd"`
	ActivePerspective string                        `json:"activePerspective"`
	Perspectives      map[string]*legacyPerspective `json:"perspectives"`
}

type legacyPerspective struct {
	Columns       []*layout.Region `json:"columns"`
	Bottom        *layout.Region   `json:"bottom"`
	BottomVisible bool             `json:"bottomVisible"`
	ColumnSplits  []float32        `json:"columnSplits"`
	TopPercent    float32          `json:"topPercent"`
}

func migrateV1(data json.RawMessage) (*layout.State, error) {
	var old *legacyState
	if err := decodeJSON(data, &old); err != nil {
		return nil, err
	}
	if old == nil {
		return nil, fmt.Errorf("layout is missing")
	}
	maximum, err := legacyMaximumID(old)
	if err != nil {
		return nil, err
	}
	if maximum >= math.MaxUint64-uint64(2*len(old.Projects)) {
		return nil, fmt.Errorf("no ID space for migrated perspectives")
	}
	state := &layout.State{ActiveProject: old.ActiveProject, Global: old.Global,
		GlobalVisible: old.GlobalVisible, RailExpanded: old.RailExpanded, CenterPercent: old.CenterPercent}
	for _, project := range old.Projects {
		p := &layout.Project{Name: project.Name, CWD: project.CWD}
		for i, kind := range []string{"work", "review"} {
			previous := project.Perspectives[kind]
			maximum++
			perspective := &layout.Perspective{ID: layout.PerspectiveID(maximum), Name: []string{"Work", "Review"}[i],
				Columns: previous.Columns, Bottom: previous.Bottom, BottomVisible: previous.BottomVisible,
				ColumnSplits: previous.ColumnSplits, TopPercent: previous.TopPercent}
			p.Perspectives = append(p.Perspectives, perspective)
			if project.ActivePerspective == kind {
				p.ActivePerspective = perspective.ID
			}
		}
		state.Projects = append(state.Projects, p)
	}
	if err := state.Restore(); err != nil {
		return nil, err
	}
	return state, nil
}

func legacyMaximumID(old *legacyState) (uint64, error) {
	regions := []*layout.Region{old.Global}
	for _, project := range old.Projects {
		if project == nil || len(project.Perspectives) != 2 ||
			(project.ActivePerspective != "work" && project.ActivePerspective != "review") {
			return 0, fmt.Errorf("invalid legacy project perspectives")
		}
		for _, kind := range []string{"work", "review"} {
			p := project.Perspectives[kind]
			if p == nil {
				return 0, fmt.Errorf("legacy perspective %q is missing", kind)
			}
			regions = append(regions, p.Columns...)
			regions = append(regions, p.Bottom)
		}
	}
	var maximum uint64
	for _, region := range regions {
		if region == nil {
			continue // Restore reports invalid regions after conversion.
		}
		maximum = max(maximum, uint64(region.ID))
		for _, view := range region.Views {
			if view != nil {
				maximum = max(maximum, uint64(view.ID))
			}
		}
	}
	return maximum, nil
}
