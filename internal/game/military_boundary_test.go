package game

import (
	"reflect"
	"testing"
)

func TestHiddenArmyDoesNotChangeRejectedCivilianErrors(t *testing.T) {
	a, b := scenarioState(), scenarioState()
	p := Point{14, 7}
	b.Tribes[1].Armies[0] = Army{People: 5, Position: p}
	b.revealArmies(1)
	for _, kind := range []ActionKind{Claim, BuildFarm} {
		ea, eb := a.Apply(0, Action{Kind: kind, Target: p}), b.Apply(0, Action{Kind: kind, Target: p})
		if ea == nil || eb == nil || ea.Error() != eb.Error() {
			t.Fatal("rejected civilian operation disclosed a hidden army")
		}
	}
	va, _ := a.View(0)
	vb, _ := b.View(0)
	if !reflect.DeepEqual(va, vb) {
		t.Fatal("hidden enemy military state changed the player's view")
	}
}
func TestMilitaryRecordValidation(t *testing.T) {
	for _, change := range []func(*State){
		func(s *State) { s.Tribes[0].Armies[0].People = 6 },
		func(s *State) { s.Tribes[0].Armies[0].People = -1 },
		func(s *State) { s.Tribes[0].Armies[0].Position = Point{-1, 3} },
		func(s *State) { s.Tribes[0].Armies[0].Position = Point{0, 0}; s.Tiles[0][0].Terrain = Ocean },
		func(s *State) { s.Tribes[0].Armies[0].Unsupplied = -1 },
		func(s *State) { s.Tribes[0].Armies[0].Order = ArmyOrder{Kind: ArmyOrderKind(255)} },
		func(s *State) { s.Tribes[0].Armies[0].Order = ArmyOrder{Kind: March, Target: Point{-1, -1}} },
		func(s *State) { s.Tribes[0].Armies[0].Order = ArmyOrder{Kind: Hold, Target: Point{3, 3}} },
		func(s *State) { s.Tribes[0].Armies[0].Order = ArmyOrder{Kind: Occupy, Target: Point{3, 3}} },
		func(s *State) { s.Tribes[0].Armies[1].Position = Point{3, 3} },
		func(s *State) {
			s.Tribes[1].Armies[0] = Army{People: 5, Position: s.Tribes[0].Capital}
			s.revealArmies(1)
		},
		func(s *State) { s.Tiles[3][2].RestUntil = -1 },
		func(s *State) { s.Tiles[3][2].RestUntil = 100 },
	} {
		s := scenarioState()
		if err := s.SetArmySize(0, 0, 5); err != nil {
			t.Fatal(err)
		}
		change(s)
		if s.Validate() == nil {
			t.Fatal("invalid military record accepted")
		}
	}
}
