package game

import (
	"strings"
	"testing"
)

func TestOccupiedPlotPausesWorkAndExplainsLostProduction(t *testing.T) {
	s := scenarioState()
	p := s.Tribes[0].Capital
	s.Tiles[p.Y][p.X].Farm = FarmReady - 1
	s.Tribes[0].Armies[0] = Army{People: 5, Position: Point{2, 2}}
	s.revealArmies(0)
	s.Tribes[1].Armies[0] = Army{People: 5, Position: p}
	s.revealArmies(1)
	if err := s.SetArmyOrder(1, 0, ArmyOrder{Kind: Attack, Target: Point{2, 2}}); err != nil {
		t.Fatal(err)
	}
	v, _ := s.View(0)
	f := ForecastFor(v)
	if f.Production.Food != 0 {
		t.Fatal("preview operated a farm under hostile occupation")
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tiles[p.Y][p.X].Farm != FarmReady-1 || !strings.Contains(strings.Join(s.VisibleEvents(0, 0), "\n"), "敌军驻扎") {
		t.Fatal("occupation advanced construction or silently removed production")
	}
}
