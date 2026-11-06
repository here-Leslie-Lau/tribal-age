package game

import "testing"

func invaderOnCapital() *State {
	s := scenarioState()
	p := s.Tribes[1].Capital
	s.Tiles[p.Y][p.X].Farm = FarmReady
	s.Tribes[0].Armies[0] = Army{People: 5, Position: p}
	s.Tribes[0].Food = 100
	s.revealArmies(0)
	return s
}
func TestOccupationPreservesFarmAndRelocatesWithoutCreatingCapacity(t *testing.T) {
	s := invaderOnCapital()
	p := s.Tribes[1].Capital
	fallback := Point{p.X, p.Y + 1}
	ownTile(s, 1, fallback, Grassland, FarmNone)
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: Occupy}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tiles[p.Y][p.X].Owner != 0 || s.Tiles[p.Y][p.X].Farm != FarmReady || s.Tribes[0].Food != 93 {
		t.Fatal("occupation destroyed a farm, transferred a treasury or produced before recovery")
	}
	v0, _ := s.View(0)
	v1, _ := s.View(1)
	if CarryingCapacity(v0) != 60 || CarryingCapacity(v1) != 5 || s.Tribes[1].Capital != fallback || !s.Tribes[1].Alive() {
		t.Fatal("capital identity teleported its 30-person settlement")
	}
	if ForecastFor(v0).Production.Food != 4 {
		t.Fatal("captured farm was not available in the following month")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestMilitaryDefeatRetainsPopulationAndCapturedFacilities(t *testing.T) {
	s := invaderOnCapital()
	p := s.Tribes[1].Capital
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: Occupy}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	t1 := s.Tribes[1]
	if !t1.Defeated || t1.Alive() || t1.Population != 30 || t1.Soldiers != 0 || t1.Food != 0 || s.Tiles[p.Y][p.X].Farm != FarmReady {
		t.Fatal("defeat was treated as mass death, annexation or scorched-earth cleanup")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestSupplyNetworkAndConsecutiveCutoff(t *testing.T) {
	s := scenarioState()
	ownTile(s, 0, Point{3, 3}, Grassland, FarmNone)
	ownTile(s, 0, Point{4, 3}, Grassland, FarmNone)
	ownTile(s, 0, Point{8, 3}, Grassland, FarmNone)
	if s.supplyAt(0, Point{5, 3}) != Supplied || s.supplyAt(0, Point{8, 3}) != CutOff {
		t.Fatal("isolated land became a magic supply source")
	}
	s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{3, 3}}
	s.revealArmies(1)
	if s.supplyAt(0, Point{5, 3}) != CutOff {
		t.Fatal("an enemy did not cut the supply connection")
	}
	s = scenarioState()
	s.Tribes[0].Armies[0] = Army{People: 5, Position: Point{7, 3}}
	s.revealArmies(0)
	for month := 1; month <= 2; month++ {
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
	}
	if s.Tribes[0].Armies[0].People != 4 || s.Tribes[0].Soldiers != 4 || s.Tribes[0].Population != 29 {
		t.Fatal("continued cutoff did not consistently reduce people and soldiers")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestMutualCombatDeathCleansUpAndFamineReconcilesArmies(t *testing.T) {
	s := scenarioState()
	for id, p := range []Point{{2, 3}, {3, 3}} {
		s.Tribes[id].Population = 1
		s.Tribes[id].Soldiers = 1
		s.Tribes[id].Armies[0] = Army{People: 1, Position: p}
		s.revealArmies(id)
	}
	if err := s.SetArmyOrder(1, 0, ArmyOrder{Kind: Attack, Target: Point{2, 3}}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Population != 0 || s.Tribes[1].Population != 0 {
		t.Fatal("battle was not simultaneous")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s = scenarioState()
	s.Tribes[0].Population = 6
	s.Tribes[0].Food = 0
	s.Tribes[0].HungerMonths = 1
	s.Tribes[0].Armies[0] = Army{People: 5, Position: s.Tribes[0].Capital}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].ReserveSoldiers() < 0 || s.Tribes[0].Armies[0].People != 3 {
		t.Fatal("famine left more deployed soldiers than surviving people")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
