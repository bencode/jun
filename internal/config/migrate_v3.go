package config

import (
	"bytes"
	"encoding/json"

	"github.com/bencode/jun/internal/layout"
)

// migrateV3 reads a version 3 layout, whose fixed regions show a view unless
// it says hidden, where version 4 shows only one that says show. It works on
// the JSON, as no View holds the old field: what it does not know, it leaves
// for decoding to report.
func migrateV3(data json.RawMessage) (*layout.State, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber() // IDs go past what a float64 holds
	var doc map[string]any
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}
	showViews(doc["global"])
	for _, project := range objects(doc["projects"]) {
		for _, perspective := range objects(project["perspectives"]) {
			showViews(perspective["side"])
		}
	}
	converted, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var state *layout.State
	err = decodeJSON(converted, &state)
	return state, err
}

func showViews(region any) {
	r, _ := region.(map[string]any)
	for _, view := range objects(r["views"]) {
		hidden, ok := view["hidden"].(bool)
		if _, set := view["hidden"]; set && !ok {
			continue
		}
		delete(view, "hidden")
		view["show"] = !hidden
	}
}

// objects is the JSON objects of an array, of anything else none.
func objects(value any) []map[string]any {
	items, _ := value.([]any)
	var result []map[string]any
	for _, item := range items {
		if object, ok := item.(map[string]any); ok {
			result = append(result, object)
		}
	}
	return result
}
