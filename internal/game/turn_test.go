package game

import (
	"reflect"
	"strings"
	"testing"
)

func TestMonthlyHandExamples(t *testing.T) {
	t.Run("initial consumption", func(t *testing.T) {
		s := scenarioState()
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		p := s.Tribes[0]
		if p.Food != 53 || p.Wood != 20 || p.Population != 30 || p.Soldiers != 5 || p.AP != 2 || s.Calendar.Month() != 4 {
			t.Fatalf("wrong initial month: %+v", p)
		}
		if p.LastChange != (Stock{-7, 0}) {
			t.Fatalf("wrong net change: %+v", p.LastChange)
		}
	})
	t.Run("farm completion and seasons", func(t *testing.T) {
		s := scenarioState()
		p := s.Tribes[0].Capital
		if err := s.Apply(0, Action{BuildFarm, p}); err != nil {
			t.Fatal(err)
		}
		for i, food := range []int{53, 46, 43, 42} {
			if err := s.EndMonth(Interactive); err != nil {
				t.Fatal(err)
			}
			if s.Tribes[0].Food != food || s.Tribes[0].Wood != 10 || s.Tiles[p.Y][p.X].Farm.Progress() != min(i+1, 3) {
				t.Fatalf("month %d: food %d, farm %v", i, s.Tribes[0].Food, s.Tiles[p.Y][p.X].Farm)
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("winter and year rollover", func(t *testing.T) {
		s := scenarioState()
		s.Calendar.Elapsed = 9
		s.Tribes[0].Food = 20
		s.Tribes[0].Wood = 0
		s.Tiles[3][2].Farm = FarmReady
		s.Tiles[2][2] = Tile{Terrain: Forest, Owner: 0}
		s.Tribes[0].Explored[2][2] = true
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Food != 13 || s.Tribes[0].Wood != 2 || s.Calendar.Year() != 2 || s.Calendar.Month() != 1 {
			t.Fatal("wrong winter settlement")
		}
	})
	t.Run("starvation and soldier clamp", func(t *testing.T) {
		for _, tt := range []struct{ pop, soldiers, food, wantPop, wantSoldiers int }{{30, 5, 3, 23, 5}, {6, 5, 0, 3, 3}} {
			s := scenarioState()
			s.Tribes[0].Population = tt.pop
			s.Tribes[0].Soldiers = tt.soldiers
			s.Tribes[0].Food = tt.food
			if err := s.EndMonth(Interactive); err != nil {
				t.Fatal(err)
			}
			if s.Tribes[0].Population != tt.pop || s.Tribes[0].Soldiers != tt.soldiers || s.Tribes[0].Food != 0 {
				t.Fatal("first shortage must grant a grace month without reducing population")
			}
			if err := s.EndMonth(Interactive); err != nil {
				t.Fatal(err)
			}
			if s.Tribes[0].Population != tt.wantPop || s.Tribes[0].Soldiers != tt.wantSoldiers || s.Tribes[0].Food != 0 {
				t.Fatalf("wrong starvation: %+v", s.Tribes[0])
			}
			if !strings.Contains(strings.Join(s.VisibleEvents(0, 0), "\n"), "饥荒") {
				t.Fatal("starvation unexplained")
			}
		}
	})
	t.Run("training consumption", func(t *testing.T) {
		s := scenarioState()
		if err := s.Apply(0, Action{Kind: Train}); err != nil {
			t.Fatal(err)
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Food != 47 || s.Tribes[0].Soldiers != 10 || s.Tribes[0].LastChange.Food != -13 {
			t.Fatal("wrong training month")
		}
	})
}

func TestAllSeasonYieldsAndRations(t *testing.T) {
	for _, tt := range []struct{ elapsed, yield int }{{0, 4}, {3, 6}, {6, 8}, {9, 0}} {
		s := scenarioState()
		s.Calendar.Elapsed = tt.elapsed
		s.Tiles[3][2].Farm = FarmReady
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Food != 60+tt.yield-7 {
			t.Fatalf("elapsed %d: food %d", tt.elapsed, s.Tribes[0].Food)
		}
	}
	for _, tt := range []struct{ pop, soldiers, want int }{{0, 0, 0}, {1, 0, 1}, {5, 5, 2}, {6, 5, 3}, {30, 5, 7}, {30, 10, 8}} {
		if FoodDemand(tt.pop, tt.soldiers) != tt.want {
			t.Fatalf("wrong rations: %+v", tt)
		}
	}
}

func TestNewForestProducesInClaimMonth(t *testing.T) {
	s := scenarioState()
	s.Tiles[2][2].Terrain = Forest
	for _, kind := range []ActionKind{Explore, Claim} {
		if err := s.Apply(0, Action{kind, Point{2, 2}}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Tribes[0].Wood != 15 {
		t.Fatal("production was credited before settlement")
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Wood != 17 || s.Tribes[0].LastChange.Wood != -3 {
		t.Fatal("new forest did not produce")
	}
}

func TestExtinctionCleanupAndTerminalMonth(t *testing.T) {
	s := scenarioState()
	for id := range s.Tribes {
		s.Tribes[id].Population = 1
		s.Tribes[id].Soldiers = 1
		s.Tribes[id].Food = 0
		s.Tribes[id].Wood = 7
		p := s.Tribes[id].Capital
		s.Tiles[p.Y][p.X].Farm = FarmPlanned
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.AllExtinct() || s.Calendar.Elapsed != 1 {
		t.Fatal("first shortage must not kill tribes")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if !s.AllExtinct() || s.Calendar.Elapsed != 2 {
		t.Fatal("continuous shortage should kill these tribes after two months")
	}
	for _, tribe := range s.Tribes {
		if tribe.Food != 0 || tribe.Wood != 0 || tribe.Soldiers != 0 || tribe.AP != 0 {
			t.Fatal("extinct resources not cleared")
		}
	}
	for _, row := range s.Tiles {
		for _, tile := range row {
			if tile.Owner != Unowned || tile.Farm != FarmNone {
				t.Fatal("extinct territory not cleared")
			}
		}
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	before := *s
	if err := s.EndMonth(Simulation); err == nil {
		t.Fatal("terminal state advanced")
	}
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("rejected terminal month mutated state")
	}
}

func TestSingleExtinctionDoesNotStopSimulation(t *testing.T) {
	s := scenarioState()
	s.Tribes[0].Population = 1
	s.Tribes[0].Soldiers = 1
	s.Tribes[0].Food = 0
	s.Tribes[0].Wood = 0
	if err := s.EndMonth(Simulation); err != nil {
		t.Fatal(err)
	}
	if !s.Tribes[0].Alive() {
		t.Fatal("first shortage must grant the player a grace month")
	}
	if err := s.EndMonth(Simulation); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Alive() || s.AllExtinct() {
		t.Fatal("wrong extinction scope")
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("newly extinct tribe with empty stocks was not cleaned: %v", err)
	}
	before := *s
	if err := s.EndMonth(Interactive); err == nil || !reflect.DeepEqual(before, *s) {
		t.Fatal("dead player can continue")
	}
	if err := s.EndMonth(Simulation); err != nil {
		t.Fatal(err)
	}
	if s.Calendar.Elapsed != 3 {
		t.Fatal("surviving simulation stopped")
	}
}

func TestInvalidModeIsAtomic(t *testing.T) {
	s := scenarioState()
	before := *s
	if err := s.EndMonth(Mode(99)); err == nil || !reflect.DeepEqual(before, *s) {
		t.Fatal("invalid mode not rejected atomically")
	}
}
