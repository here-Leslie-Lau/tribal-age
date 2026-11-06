package game

import (
	"strings"
	"testing"
)

func TestDefeatIsReportedSeparatelyFromPopulationDeath(t *testing.T) {
	s := scenarioState()
	p := s.Tribes[0].Capital
	s.Tribes[1].Armies[0] = Army{People: 5, Position: p}
	s.revealArmies(1)
	if err := s.SetArmyOrder(1, 0, ArmyOrder{Kind: Occupy}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{s.EndMonth(Interactive), s.Apply(0, Action{Kind: Train})} {
		if err == nil || !strings.Contains(err.Error(), "战败") {
			t.Fatal("surviving defeated population was reported as dead")
		}
	}
}
