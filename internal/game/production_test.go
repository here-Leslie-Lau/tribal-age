package game

import (
	"reflect"
	"strings"
	"testing"
)

func ownTile(s *State, id int, p Point, terrain Terrain, farm FarmStage) {
	s.Tiles[p.Y][p.X] = Tile{Terrain: terrain, Owner: id, Farm: farm, Settlement: s.Tiles[p.Y][p.X].Settlement}
	s.Tribes[id].Explored[p.Y][p.X] = true
}

func productionScenario() *State {
	s := scenarioState()
	ownTile(s, 0, Point{2, 3}, Grassland, FarmReady)
	ownTile(s, 0, Point{3, 3}, Grassland, FarmReady)
	for _, p := range []Point{{1, 3}, {2, 2}, {3, 2}, {4, 2}} {
		ownTile(s, 0, p, Forest, FarmNone)
	}
	return s
}

func TestLaborLimitsProduction(t *testing.T) {
	s := productionScenario()
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	before := v
	before.Cells = append([]Cell(nil), v.Cells...)
	before.Legal = append([]Action(nil), v.Legal...)
	got := ProductionFor(v, Spring)
	if got.Slots != 5 || got.Food != 8 || got.Wood != 6 {
		t.Fatalf("wrong spring production: %+v", got)
	}
	if !reflect.DeepEqual(got.Farms, []Point{{2, 3}, {3, 3}}) || !reflect.DeepEqual(got.Forests, []Point{{2, 2}, {3, 2}, {4, 2}}) || !reflect.DeepEqual(got.Unstaffed, []Point{{1, 3}}) {
		t.Fatalf("must prioritize farms, then coordinates: %+v", got)
	}
	if !reflect.DeepEqual(v, before) {
		t.Fatal("production calculation mutated its input")
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Food != 61 || s.Tribes[0].Wood != 26 {
		t.Fatalf("monthly settlement ignored labor: %+v", s.Tribes[0])
	}
	if !strings.Contains(strings.Join(s.VisibleEvents(0, 0), "\n"), "人手不足") {
		t.Fatal("unstaffed production was not explained")
	}
	if strings.Contains(strings.Join(s.VisibleEvents(1, 0), "\n"), "人手不足") {
		t.Fatal("private labor status leaked")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWinterLaborMovesFromFarmsToForest(t *testing.T) {
	s := productionScenario()
	s.Calendar.Elapsed = 9
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	got := ProductionFor(v, Winter)
	if got.Food != 0 || got.Wood != 8 || len(got.Farms) != 0 || len(got.Forests) != 4 || len(got.Unstaffed) != 0 {
		t.Fatalf("winter farms consumed workers: %+v", got)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Food != 53 || s.Tribes[0].Wood != 28 {
		t.Fatal("wrong winter labor settlement")
	}
}

func TestSoldiersDoNotWork(t *testing.T) {
	s := productionScenario()
	if s.Tribes[0].WorkSlots() != 5 {
		t.Fatal("wrong initial labor")
	}
	if err := s.Apply(0, Action{Kind: Train}); err != nil {
		t.Fatal(err)
	}
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	got := ProductionFor(v, Spring)
	if s.Tribes[0].Population != 30 || s.Tribes[0].WorkSlots() != 4 || got.Food != 8 || got.Wood != 4 {
		t.Fatal("training did not reduce available workers")
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Food != 55 || s.Tribes[0].Wood != 24 {
		t.Fatal("wrong training and production costs")
	}
}

func TestIncompleteFarmsAndOtherTerritoryDoNotUseLabor(t *testing.T) {
	s := productionScenario()
	s.Tiles[3][3].Farm = FarmPlanned
	s.Tribes[0].Explored[3][10] = true
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	got := ProductionFor(v, Spring)
	if got.Food != 4 || got.Wood != 8 || len(got.Unstaffed) != 0 {
		t.Fatalf("unfinished or foreign plots consumed workers: %+v", got)
	}
	for _, tt := range []struct{ population, soldiers int }{{4, 0}, {30, 30}, {0, 0}} {
		v.Self.Population, v.Self.Soldiers = tt.population, tt.soldiers
		got := ProductionFor(v, Spring)
		if got.Slots != 0 || got.Food != 0 || got.Wood != 0 {
			t.Fatalf("production without workers: %+v", got)
		}
	}
}
