package game

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestVersionOneSaveLoadsWithoutInventingPastReports(t *testing.T) {
	s := New(42)
	for i := 0; i < 3; i++ {
		if err := s.EndMonth(Simulation); err != nil {
			t.Fatal(err)
		}
	}
	data, err := s.MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	var old map[string]any
	if err := json.Unmarshal(data, &old); err != nil {
		t.Fatal(err)
	}
	old["version"] = 1
	delete(old, "reports")
	delete(old, "reports_since")
	data, err = json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := UnmarshalSave(data)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ReportsSince != 3 || len(loaded.Reports) != 0 || !reflect.DeepEqual(s.Tribes, loaded.Tribes) || !reflect.DeepEqual(s.Events, loaded.Events) {
		t.Fatal("old state or messages changed, or past reports were invented")
	}
	if err := loaded.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	data, err = loaded.MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := UnmarshalSave(data)
	if err != nil || !reflect.DeepEqual(loaded, restored) {
		t.Fatal("new-format report or old-history boundary did not survive save")
	}
}
func TestReportSaveRejectsInvalidFacts(t *testing.T) {
	valid := New(42)
	if err := valid.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		change func(*State)
	}{
		{"future history", func(s *State) { s.ReportsSince = 1 }},
		{"foreign owner", func(s *State) { s.Reports[0].Owner = TribeCount }},
		{"future month", func(s *State) { s.Reports[0].Month = 1 }},
		{"duplicate month", func(s *State) { s.Reports = append(s.Reports, s.Reports[0]) }},
		{"unknown cause", func(s *State) { s.Reports[0].Changes[0].Cause = 255 }},
		{"negative before", func(s *State) { s.Reports[0].Changes[0].Before = -1 }},
		{"bad slot", func(s *State) { s.Reports[0].Changes[0].Slot = MaxArmies }},
		{"bad location", func(s *State) { s.Reports[0].Changes[0].Position = Point{-1, 0} }},
		{"unexplained combat", func(s *State) {
			s.Reports[0].Changes[0] = ArmyChange{Slot: 0, Position: Point{2, 3}, Before: 5, After: 0, Cause: CombatLoss}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data, err := valid.MarshalSave()
			if err != nil {
				t.Fatal(err)
			}
			s, err := UnmarshalSave(data)
			if err != nil {
				t.Fatal(err)
			}
			tt.change(s)
			if _, err := s.MarshalSave(); err == nil {
				t.Fatal("encoder accepted invalid report")
			}
			// Exercise the external boundary too, bypassing the guarded encoder.
			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatal(err)
			}
			raw["reports"] = s.Reports
			raw["reports_since"] = s.ReportsSince
			bad, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := UnmarshalSave(bad); err == nil {
				t.Fatal("decoder accepted invalid report")
			}
		})
	}
}
