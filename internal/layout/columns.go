package layout

import (
	"fmt"
	"slices"
)

func (s *State) AddColumn() (RegionID, error) {
	p := s.CurrentPerspective()
	if p == nil {
		return 0, fmt.Errorf("no active perspective")
	}
	if len(p.Columns) >= 3 {
		return 0, fmt.Errorf("maximum of three columns")
	}
	r := s.newRegion()
	p.Columns = append(p.Columns, r)
	rebalance(p)
	return r.ID, nil
}

func (s *State) RemoveColumn(id RegionID) error {
	p := s.CurrentPerspective()
	if p == nil {
		return fmt.Errorf("no active perspective")
	}
	i := slices.IndexFunc(p.Columns, func(r *Region) bool { return r.ID == id })
	if i < 0 {
		return fmt.Errorf("column not found")
	}
	if len(p.Columns) == 1 {
		return fmt.Errorf("the last column cannot be removed")
	}
	destination := max(0, i-1)
	if i == 0 {
		destination = 1
	}
	target, source := p.Columns[destination], p.Columns[i]
	target.Views = append(target.Views, source.Views...)
	if target.Active == 0 {
		target.Active = source.Active
	}
	p.Columns = slices.Delete(p.Columns, i, i+1)
	rebalance(p)
	return nil
}

func rebalance(p *Perspective) {
	p.ColumnSplits = nil
	for remaining := len(p.Columns); remaining > 1; remaining-- {
		p.ColumnSplits = append(p.ColumnSplits, 100/float32(remaining))
	}
}
