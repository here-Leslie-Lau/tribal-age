package game

import "testing"

func TestAIPreparesFarmForNextPopulationThreshold(t *testing.T) {
	s := growthScenario()
	s.Tribes[0].Population = 40
	s.Tribes[0].Food = 100
	s.Tribes[0].Wood = 20
	ownTile(s, 0, Point{2, 2}, Forest, FarmNone)
	ownTile(s, 0, Point{3, 2}, Forest, FarmNone)
	ownTile(s, 0, Point{4, 3}, Grassland, FarmNone)
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := ChooseAction(v)
	if !ok || a != (Action{BuildFarm, Point{4, 3}}) {
		t.Fatalf("AI must prepare a third farm before population 41: %+v", a)
	}
	if err := s.Apply(0, a); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAIDoesNotClaimForestBeyondSeasonalLaborCapacity(t *testing.T) {
	s := productionScenario()
	s.Tribes[0].Food = 100
	s.Tribes[0].Wood = 100
	ownTile(s, 0, Point{4, 3}, Forest, FarmNone)
	s.Tiles[2][1] = Tile{Terrain: Forest, Owner: Unowned}
	s.Tribes[0].Explored[2][1] = true
	s.Tribes[0].Explored[4][1] = true
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := ChooseAction(v)
	if ok && a.Kind == Claim {
		t.Fatalf("AI spent wood on land with no labor or capacity need: %+v", a)
	}
}

func TestAIDoesNotBuildWithoutWorkers(t *testing.T) {
	s := scenarioState()
	s.Tribes[0].Population = 5
	s.Tribes[0].Soldiers = 5
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := ChooseAction(v)
	if ok && a.Kind == BuildFarm {
		t.Fatal("AI built a farm that no civilians can operate")
	}
}

func TestAIExpandsWhenPopulationHitsCapacity(t *testing.T) {
	s := growthScenario()
	s.Tribes[0].Population = 40
	s.Tribes[0].Food = 100
	s.Tribes[0].Wood = 20
	ownTile(s, 0, Point{2, 2}, Forest, FarmNone)
	s.Tribes[0].Explored[2][3] = true
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := ChooseAction(v)
	if !ok || a != (Action{Claim, Point{3, 2}}) {
		t.Fatalf("AI should open growth capacity: %+v", a)
	}
}
