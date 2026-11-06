package game

import (
	"reflect"
	"testing"
)

func TestAttackEstimateReusesSoloClashAndDoesNotReadEnemyOrders(t *testing.T) {
	s := scenarioState()
	s.Tiles[3][2].Terrain = Forest
	s.Tribes[0].Soldiers = 10
	s.Tribes[0].Armies[0] = Army{People: 10, Position: Point{2, 3}}
	s.revealArmies(0)
	s.Tribes[1].Soldiers = 10
	s.Tribes[1].Armies[0] = Army{People: 10, Position: Point{3, 3}}
	s.revealArmies(1)
	if err := s.SetArmyOrder(1, 0, ArmyOrder{Kind: Attack, Target: Point{2, 3}}); err != nil {
		t.Fatal(err)
	}
	v, _ := s.View(1)
	before := *s
	before.Events = append([]Event(nil), s.Events...)
	estimate := AttackEstimate(v, 0)
	if !estimate.Available || estimate.OwnLoss != 3 || estimate.EnemyLoss != 1 || !reflect.DeepEqual(before, *s) {
		t.Fatalf("wrong isolated estimate: %+v", estimate)
	}
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: March, Target: Point{2, 2}}); err != nil {
		t.Fatal(err)
	}
	v2, _ := s.View(1)
	if AttackEstimate(v2, 0) != estimate {
		t.Fatal("estimate read a hidden defender order")
	}
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: Hold}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[1].Armies[0].People != 10-estimate.OwnLoss || s.Tribes[0].Armies[0].People != 10-estimate.EnemyLoss {
		t.Fatal("isolated estimate differs from actual same-rule clash")
	}
	if AttackEstimate(v, -1).Available || AttackEstimate(v, 1).Available {
		t.Fatal("missing armies produced a forecast")
	}
}
