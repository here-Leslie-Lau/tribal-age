package game

import (
	"encoding/json"
	"testing"
)

func TestExternalReportBoundaryRejectsMalformedBattleFacts(t *testing.T) {
	s := New(42)
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	s.Reports[0].Battles = []BattleReport{{Armies: []BattleArmy{{Owner: 0, Slot: 0, Before: 5, From: Point{2, 3}, Target: Point{2, 3}, Supply: Supplied}, {Owner: 1, Slot: 0, Before: 5, Lost: 1, From: Point{3, 3}, Target: Point{2, 3}, Supply: Supplied}}}}
	data, err := s.MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		change func(*State)
	}{
		{"no participants", func(s *State) { s.Reports[0].Battles[0].Armies = nil }},
		{"only opponent", func(s *State) { s.Reports[0].Battles[0].Armies[0].Owner = 2 }},
		{"only own", func(s *State) {
			s.Reports[0].Battles[0].Armies[1].Owner = 0
			s.Reports[0].Battles[0].Armies[1].Slot = 1
		}},
		{"duplicated participant", func(s *State) { s.Reports[0].Battles[0].Armies[1] = s.Reports[0].Battles[0].Armies[0] }},
		{"wrong faction", func(s *State) { s.Reports[0].Battles[0].Armies[1].Owner = -1 }},
		{"wrong army", func(s *State) { s.Reports[0].Battles[0].Armies[1].Slot = 3 }},
		{"no people", func(s *State) { s.Reports[0].Battles[0].Armies[1].Before = 0 }},
		{"too many dead", func(s *State) { s.Reports[0].Battles[0].Armies[1].Lost = 6 }},
		{"invalid route", func(s *State) { s.Reports[0].Battles[0].Armies[1].Target = Point{5, 5} }},
		{"invalid supply", func(s *State) { s.Reports[0].Battles[0].Armies[1].Supply = 255 }},
		{"moving forest defense", func(s *State) { s.Reports[0].Battles[0].Armies[1].ForestDefense = true }},
		{"reserve combat", func(s *State) { s.Reports[0].Changes[0].Slot = ReserveSlot }},
		{"fake loss with growth", func(s *State) { s.Reports[0].Changes[0].Cause = SupplyLoss }},
		{"partial defeat", func(s *State) {
			s.Reports[0].Changes[0] = ArmyChange{Slot: 0, Position: Point{2, 3}, Before: 5, After: 2, Cause: DefeatDisbanded}
		}},
		{"empty month", func(s *State) { s.Reports[0].Changes = nil; s.Reports[0].Battles = nil }},
		{"unchanged count", func(s *State) { s.Reports[0].Changes[0].Before = 5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			restored, err := UnmarshalSave(data)
			if err != nil {
				t.Fatal(err)
			}
			tt.change(restored)
			var obj map[string]any
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			obj["reports"] = restored.Reports
			bad, err := json.Marshal(obj)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := UnmarshalSave(bad); err == nil {
				t.Fatal("unsafe report accepted")
			}
		})
	}
}
