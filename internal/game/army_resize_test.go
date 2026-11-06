package game

import "testing"

func TestResizingAtCapitalPreservesQueuedIntent(t *testing.T) {
	s := scenarioState()
	if err := s.SetArmySize(0, 0, 3); err != nil {
		t.Fatal(err)
	}
	order := ArmyOrder{Kind: March, Target: Point{2, 2}}
	if err := s.SetArmyOrder(0, 0, order); err != nil {
		t.Fatal(err)
	}
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Armies[0].Order != order {
		t.Fatal("reinforcing silently canceled a queued order")
	}
}
