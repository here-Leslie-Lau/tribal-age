package game

import (
	"encoding/json"
	"testing"
)

func TestSaveBoundaryRejectsDamagedStructureAndUnsafeRecords(t *testing.T) {
	good, err := New(42).MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	tribe := func(o map[string]any) map[string]any { return o["tribes"].([]any)[0].(map[string]any) }
	for _, tt := range []struct {
		name   string
		damage func(map[string]any)
	}{
		{"unknown field", func(o map[string]any) { o["unexpected"] = true }},
		{"missing seed", func(o map[string]any) { delete(o, "seed") }},
		{"missing calendar", func(o map[string]any) { delete(o, "calendar") }},
		{"short map row", func(o map[string]any) { o["tiles"].([]any)[0] = []any{} }},
		{"missing fog", func(o map[string]any) { tribe(o)["Explored"] = []any{} }},
		{"short fog row", func(o map[string]any) { tribe(o)["Explored"].([]any)[0] = []any{} }},
		{"missing army roster", func(o map[string]any) { tribe(o)["Armies"] = []any{} }},
		{"oversized army roster", func(o map[string]any) { tribe(o)["Armies"] = append(tribe(o)["Armies"].([]any), map[string]any{}) }},
		{"missing initial stock", func(o map[string]any) { delete(tribe(o), "month_start") }},
		{"negative initial stock", func(o map[string]any) { tribe(o)["month_start"].(map[string]any)["Food"] = -1 }},
		{"impossible soldiers", func(o map[string]any) { tribe(o)["Soldiers"] = 31 }},
		{"unsafe arithmetic", func(o map[string]any) { tribe(o)["Population"] = 1e16 }},
		{"terminal escape", func(o map[string]any) { tribe(o)["Name"] = "\x1b[2J" }},
		{"negative calendar", func(o map[string]any) { o["calendar"].(map[string]any)["Elapsed"] = -1 }},
		{"invalid event", func(o map[string]any) {
			o["events"] = []any{map[string]any{"Month": 0, "Audience": 0, "Text": "lost audience"}}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var o map[string]any
			if err := json.Unmarshal(good, &o); err != nil {
				t.Fatal(err)
			}
			tt.damage(o)
			bad, err := json.Marshal(o)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := UnmarshalSave(bad); err == nil {
				t.Fatal("damaged archive was accepted")
			}
		})
	}
	if _, err := UnmarshalSave(make([]byte, MaxSaveBytes+1)); err == nil {
		t.Fatal("oversized save accepted")
	}
	var s *State
	if _, err := s.MarshalSave(); err == nil {
		t.Fatal("nil game was saved")
	}
	s = New(42)
	s.Tribes[0].Name = "\x1b[2J"
	if _, err := s.MarshalSave(); err == nil {
		t.Fatal("encoder emitted an archive that its own boundary rejects")
	}
}
