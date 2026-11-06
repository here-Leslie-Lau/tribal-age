package game

import (
	"reflect"
	"testing"
)

func TestMilitaryAIUsesVisiblePositionsNotEnemyPlans(t *testing.T) {
	s := scenarioState()
	p := s.Tribes[1].Capital
	enemy := Point{p.X - 1, p.Y}
	s.Tribes[0].Armies[0] = Army{People: 5, Position: enemy}
	s.revealArmies(0)
	s.prepareDefense(1)
	if s.Tribes[1].Armies[0].People != 5 || s.Tribes[1].AP != 1 {
		t.Fatal("AI failed to assemble a legal defense")
	}
	v, _ := s.View(1)
	before := *s
	before.Events = append([]Event(nil), s.Events...)
	orders := ArmyOrdersFor(v)
	if orders[0].Kind != Attack || orders[0].Target != enemy || !reflect.DeepEqual(before, *s) {
		t.Fatal("defensive planning was not pure or failed to engage")
	}
	s.Tribes[0].Armies[0].Order = ArmyOrder{Kind: March, Target: Point{p.X - 1, p.Y - 1}}
	v2, _ := s.View(1)
	if !reflect.DeepEqual(orders, ArmyOrdersFor(v2)) {
		t.Fatal("AI inspected a hidden enemy order")
	}
}
func TestEnemyOccupationBlocksCivilianConstruction(t *testing.T) {
	s := invaderOnCapital()
	id := 1
	p := s.Tribes[id].Capital
	for _, a := range []Action{{Kind: BuildFarm, Target: p}, {Kind: Train}} {
		s.Tiles[p.Y][p.X].Farm = FarmNone
		before := *s
		before.Events = append([]Event(nil), s.Events...)
		if s.Apply(id, a) == nil || !reflect.DeepEqual(before, *s) {
			t.Fatal("civilians built or recruited beneath an enemy army")
		}
	}
}
