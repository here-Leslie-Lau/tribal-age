package game

import (
	"reflect"
	"strings"
	"testing"
)

func reportAt(t *testing.T, s *State, id, month int) MilitaryReport {
	t.Helper()
	for _, r := range s.MilitaryHistory(id) {
		if r.Month == month {
			return r
		}
	}
	t.Fatalf("missing tribe %d report for month %d", id, month)
	return MilitaryReport{}
}
func TestBattleReportUsesActualClashAndSurvivesArmyRemoval(t *testing.T) {
	s := scenarioState()
	s.Tribes[0].Soldiers = 20
	s.Tribes[1].Soldiers = 15
	s.Tribes[1].Capital = Point{3, 3}
	ownTile(s, 1, Point{3, 3}, Grassland, FarmNone)
	s.Tribes[0].Armies[0] = Army{People: 20, Position: Point{2, 3}, Order: ArmyOrder{Kind: Attack, Target: Point{3, 3}}}
	s.Tribes[1].Armies[0] = Army{People: 15, Position: Point{3, 3}}
	s.revealArmies(0)
	s.revealArmies(1)
	s.resolveMilitary()
	r := reportAt(t, s, 0, 0)
	if len(r.Battles) != 1 || len(r.Battles[0].Armies) != 2 {
		t.Fatalf("missing grouped opponents: %+v", r)
	}
	own, enemy := r.Battles[0].Losses(0)
	if own != 5 || enemy != 6 || s.Tribes[0].Armies[0].People != 15 || s.Tribes[1].Armies[0].People != 9 {
		t.Fatal("report disagrees with actual 20 vs 15 grass clash")
	}
	text := strings.Join(r.Lines(), "\n")
	for _, word := range []string{"20 → 15", "15 → 9", "阵亡比（我:敌）5:6", "未推进"} {
		if !strings.Contains(text, word) {
			t.Fatalf("missing result %q in %s", word, text)
		}
	}
	copy := s.MilitaryHistory(0)
	copy[0].Battles[0].Armies[0].Before = 999
	copy[0].Changes[0].Before = 999
	if !reflect.DeepEqual(r, reportAt(t, s, 0, 0)) {
		t.Fatal("history aliases live report")
	}
	if len(s.MilitaryHistory(2)) != 0 || s.MilitaryHistory(-1) != nil {
		t.Fatal("unrelated tribe obtained private battle report")
	}
}
func TestArmyDisappearanceHasOrderedNonCombatReasons(t *testing.T) {
	s := scenarioState()
	s.Tribes[0].Population = 1
	s.Tribes[0].Soldiers = 1
	s.Tribes[0].Armies[0] = Army{People: 1, Position: Point{7, 3}, Unsupplied: 1}
	s.revealArmies(0)
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	r := reportAt(t, s, 0, 0)
	if len(r.Changes) != 1 || r.Changes[0].Cause != SupplyLoss || r.Changes[0].Before != 1 || r.Changes[0].After != 0 || !r.Destroyed() {
		t.Fatalf("cutoff disappearance unexplained: %+v", r)
	}
	if !strings.Contains(strings.Join(r.Lines(), "\n"), "断供") {
		t.Fatal("cutoff was reported as combat")
	}
	s = scenarioState()
	s.Tribes[0].Population = 6
	s.Tribes[0].Food = 0
	s.Tribes[0].HungerMonths = 1
	s.Tribes[0].Armies[2] = Army{People: 5, Position: s.Tribes[0].Capital}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	r = reportAt(t, s, 0, 0)
	if len(r.Changes) != 1 || r.Changes[0].Slot != 2 || r.Changes[0].Cause != FamineLoss || r.Changes[0].Before != 5 || r.Changes[0].After != 3 {
		t.Fatal("famine not attributed to actual army")
	}
	s = scenarioState()
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	if err := s.SetArmySize(0, 0, 0); err != nil {
		t.Fatal(err)
	}
	r = reportAt(t, s, 0, 0)
	if r.Destroyed() || s.Tribes[0].ReserveSoldiers() != 5 || r.Changes[1].Cause != Reorganized {
		t.Fatal("voluntary disband counted as death")
	}
}
func TestMixedBattleReportDoesNotDoubleCountOrExposeRemoteFighters(t *testing.T) {
	s := scenarioState()
	// A meets B; B is also stopped by C beyond A's direct contact.
	s.Tribes[0].Armies[0] = Army{People: 5, Position: Point{2, 3}, Order: ArmyOrder{Kind: Attack, Target: Point{3, 3}}}
	s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{3, 3}, Order: ArmyOrder{Kind: March, Target: Point{4, 3}}}
	s.Tribes[2].Armies[0] = Army{People: 5, Position: Point{5, 3}, Order: ArmyOrder{Kind: March, Target: Point{4, 3}}}
	s.resolveMilitary()
	r := reportAt(t, s, 0, 0)
	if len(r.Battles) != 1 || !r.Battles[0].Partial || len(r.Battles[0].Armies) != 2 {
		t.Fatalf("chain was omitted or leaked distant C: %+v", r)
	}
	if !strings.Contains(r.Summary(), "已知") {
		t.Fatal("partial enemy losses look like complete-world casualties")
	}
	for _, a := range r.Battles[0].Armies {
		if a.Owner == 2 {
			t.Fatal("remote fighter leaked through contact chain")
		}
	}
	s = scenarioState()
	s.Tribes[0].Soldiers = 10
	s.Tribes[0].Armies[0] = Army{People: 5, Position: Point{2, 3}, Order: ArmyOrder{Kind: Attack, Target: Point{3, 3}}}
	s.Tribes[0].Armies[1] = Army{People: 5, Position: Point{3, 2}, Order: ArmyOrder{Kind: Attack, Target: Point{3, 3}}}
	s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{3, 3}}
	s.resolveMilitary()
	r = reportAt(t, s, 0, 0)
	if len(r.Battles) != 1 || len(r.Battles[0].Armies) != 3 {
		t.Fatal("one engagement duplicated per army")
	}
	own, enemy := r.Battles[0].Losses(0)
	if own != 2 || enemy != 3 {
		t.Fatalf("wrong mixed casualties %d %d", own, enemy)
	}
}
