package game

import (
	"strings"
	"testing"
)

func TestMilitaryChangesReconcileBattleCutoffAndFamineInOrder(t *testing.T) {
	s := scenarioState()
	t0 := &s.Tribes[0]
	t0.Population = 5
	t0.Soldiers = 5
	t0.Food = 0
	t0.HungerMonths = 1
	t0.Armies[0] = Army{People: 5, Position: Point{7, 3}, Unsupplied: 1}
	s.Tribes[1].Armies[0] = Army{People: 1, Position: Point{8, 3}, Order: ArmyOrder{Kind: Attack, Target: Point{7, 3}}}
	s.resolveMilitary()
	s.consume(0)
	r := reportAt(t, s, 0, 0)
	want := []ArmyChangeCause{CombatLoss, SupplyLoss, FamineLoss}
	if len(r.Changes) != 3 {
		t.Fatalf("missing a settlement stage: %+v", r)
	}
	remaining := 5
	for i, c := range r.Changes {
		if c.Cause != want[i] || c.Before != remaining || c.After >= c.Before {
			t.Fatalf("incorrect causal chain: %+v", r.Changes)
		}
		remaining = c.After
	}
	if remaining != t0.Armies[0].People || remaining != t0.Soldiers || remaining != t0.Population {
		t.Fatal("casualty ledger does not conserve headcount")
	}
	if err := validateSavedReports(s); err != nil {
		t.Fatal(err)
	}
}
func TestReserveFamineAndDefeatAreNotMissingOrCombatKills(t *testing.T) {
	s := scenarioState()
	s.Tribes[0].Population = 5
	s.Tribes[0].Food = 0
	s.Tribes[0].HungerMonths = 1
	s.consume(0)
	r := reportAt(t, s, 0, 0)
	if len(r.Changes) != 1 || r.Changes[0].Slot != ReserveSlot || r.Changes[0].Before-r.Changes[0].After != 2 {
		t.Fatal("reserve soldier famine was not explained")
	}
	s = invaderOnCapital()
	s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{10, 5}}
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: Occupy}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	r = reportAt(t, s, 1, 0)
	if !r.Destroyed() || len(r.Changes) != 1 || r.Changes[0].Cause != DefeatDisbanded || s.Tribes[1].Population != 30 {
		t.Fatal("defeat removal was unreported or treated as death")
	}
	if !strings.Contains(strings.Join(r.Lines(), "\n"), "未计为战死") {
		t.Fatal("defeat report misled player about surviving soldiers")
	}
}
func TestBattleReportForestSupplyAndZeroCasualtiesAreExplicit(t *testing.T) {
	s := scenarioState()
	s.Tribes[0].Armies[0] = Army{People: 5, Position: Point{2, 3}}
	s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{3, 3}, Order: ArmyOrder{Kind: Attack, Target: Point{2, 3}}}
	s.Tiles[3][2].Terrain = Forest
	s.resolveMilitary()
	r := reportAt(t, s, 0, 0)
	if !r.Battles[0].Armies[0].ForestDefense || r.Battles[0].Armies[1].Supply != CutOff {
		t.Fatal("combat conditions were discarded")
	}
	text := strings.Join(r.Lines(), "\n")
	if !strings.Contains(text, "森林驻守防御") || !strings.Contains(text, "断供") {
		t.Fatal("defense/supply conditions not explained")
	}
	b := BattleReport{Armies: []BattleArmy{{Owner: 0, Before: 5}, {Owner: 1, Before: 5, Lost: 2}}}
	if b.ratio(0) != "我军无阵亡；敌军阵亡2人。" {
		t.Fatal("zero loss ratio is misleading or divides by zero")
	}
}
