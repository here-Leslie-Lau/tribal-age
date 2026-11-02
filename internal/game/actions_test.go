package game

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func TestExploreAndClaim(t *testing.T) {
	s := scenarioState()
	p := Point{2, 2}
	s.Tiles[p.Y][p.X].Terrain = Forest
	if err := s.Apply(0, Action{Kind: Explore, Target: p}); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].AP != 1 || !s.Tribes[0].Explored[2][2] || s.Tribes[1].Explored[2][2] || s.Tiles[2][2].Owner != Unowned {
		t.Fatal("exploration changed territory or another tribe's fog")
	}
	if err := s.Apply(0, Action{Kind: Claim, Target: p}); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].AP != 0 || s.Tribes[0].Wood != 15 || s.Tiles[2][2].Owner != 0 {
		t.Fatal("wrong claim cost/effect")
	}
	if len(s.VisibleEvents(0, 0)) != 2 || len(s.VisibleEvents(1, 0)) != 0 {
		t.Fatal("hidden actions leaked")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFarmAndTrain(t *testing.T) {
	s := scenarioState()
	p := s.Tribes[0].Capital
	if err := s.Apply(0, Action{Kind: BuildFarm, Target: p}); err != nil {
		t.Fatal(err)
	}
	if s.Tiles[p.Y][p.X].Farm != FarmPlanned || s.Tiles[p.Y][p.X].Farm.Progress() != 0 || s.Tribes[0].Wood != 10 || s.Tribes[0].AP != 1 {
		t.Fatal("farm should start at progress 0 and cost 10 wood, 1 AP")
	}
	if err := s.Apply(0, Action{Kind: Train}); err != nil {
		t.Fatal(err)
	}
	tribe := s.Tribes[0]
	if tribe.Population != 30 || tribe.Soldiers != 10 || tribe.Food != 55 || tribe.AP != 0 {
		t.Fatalf("wrong training result: %+v", tribe)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedActionsAreAtomic(t *testing.T) {
	tests := []struct {
		name   string
		id     int
		action Action
		setup  func(*State)
	}{
		{"negative tribe", -1, Action{Kind: Train}, nil},
		{"unknown tribe", TribeCount, Action{Kind: Train}, nil},
		{"unknown action", 0, Action{Kind: ActionKind(99)}, nil},
		{"negative x", 0, Action{Explore, Point{-1, 3}}, nil},
		{"large x", 0, Action{Explore, Point{Width, 3}}, nil},
		{"negative y", 0, Action{Explore, Point{2, -1}}, nil},
		{"large y", 0, Action{Explore, Point{2, Height}}, nil},
		{"diagonal", 0, Action{Explore, Point{1, 2}}, nil},
		{"remote", 0, Action{Explore, Point{5, 3}}, nil},
		{"repeat exploration", 0, Action{Explore, Point{2, 3}}, nil},
		{"explored is not territory", 0, Action{Explore, Point{2, 1}}, func(s *State) { s.Tribes[0].Explored[2][2] = true }},
		{"unknown claim", 0, Action{Claim, Point{2, 2}}, nil},
		{"claim owned", 0, Action{Claim, Point{2, 3}}, nil},
		{"claim impassable", 0, Action{Claim, Point{2, 2}}, func(s *State) { s.Tribes[0].Explored[2][2] = true; s.Tiles[2][2].Terrain = Ocean }},
		{"remote claim", 0, Action{Claim, Point{2, 1}}, func(s *State) { s.Tribes[0].Explored[1][2] = true }},
		{"claim wood", 0, Action{Claim, Point{2, 2}}, func(s *State) { s.Tribes[0].Explored[2][2] = true; s.Tribes[0].Wood = 4 }},
		{"farm not owned", 0, Action{BuildFarm, Point{2, 2}}, nil},
		{"farm on forest", 0, Action{BuildFarm, Point{2, 3}}, func(s *State) { s.Tiles[3][2].Terrain = Forest }},
		{"farm wood", 0, Action{BuildFarm, Point{2, 3}}, func(s *State) { s.Tribes[0].Wood = 9 }},
		{"farm under construction", 0, Action{BuildFarm, Point{2, 3}}, func(s *State) { s.Tiles[3][2].Farm = FarmPlanned }},
		{"farm complete", 0, Action{BuildFarm, Point{2, 3}}, func(s *State) { s.Tiles[3][2].Farm = FarmReady }},
		{"training food", 0, Action{Kind: Train}, func(s *State) { s.Tribes[0].Food = 4 }},
		{"training civilians", 0, Action{Kind: Train}, func(s *State) { s.Tribes[0].Population = 6 }},
		{"no AP", 0, Action{Kind: Train}, func(s *State) { s.Tribes[0].AP = 0 }},
		{"extinct", 0, Action{Kind: Train}, func(s *State) {
			s.Tribes[0].Population = 0
			s.Tribes[0].Soldiers = 0
			s.Tribes[0].Food = 0
			s.Tribes[0].Wood = 0
			s.Tribes[0].AP = 0
			s.Tiles[3][2].Owner = Unowned
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scenarioState()
			if tt.setup != nil {
				tt.setup(s)
			}
			s.Events = []Event{{Audience: 1, Text: "existing event"}}
			before := *s
			before.Events = append([]Event(nil), s.Events...)
			if err := s.Apply(tt.id, tt.action); err == nil {
				t.Fatal("illegal action accepted")
			}
			if !reflect.DeepEqual(before, *s) {
				t.Fatal("failure mutated game state")
			}
			if s.rng.Int63() != rand.New(rand.NewSource(42)).Int63() {
				t.Fatal("failure advanced RNG")
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUnknownClaimDoesNotRevealTerrain(t *testing.T) {
	for _, terrain := range []Terrain{Grassland, Forest, Ocean} {
		s := scenarioState()
		s.Tiles[2][2].Terrain = terrain
		err := s.Apply(0, Action{Claim, Point{2, 2}})
		if err == nil || !strings.Contains(err.Error(), "尚未探索") {
			t.Fatalf("hidden terrain leaked: %v", err)
		}
	}
}

func TestExploreImpassableAndEventVisibility(t *testing.T) {
	s := scenarioState()
	s.Tiles[2][2].Terrain = River
	if err := s.Apply(0, Action{Explore, Point{2, 2}}); err != nil {
		t.Fatal(err)
	}
	// Learning about the place later must not retroactively reveal its old events.
	s.Tribes[1].Explored[2][2] = true
	if len(s.VisibleEvents(1, 0)) != 0 {
		t.Fatal("past hidden event leaked after exploration")
	}
	s.Tribes[0].Explored[2][3] = true
	s.Tribes[1].Explored[2][3] = true
	// This square is not adjacent yet; use an actually adjacent shared square.
	s.Tribes[1].Explored[3][3] = true
	if err := s.Apply(0, Action{Explore, Point{3, 3}}); err != nil {
		t.Fatal(err)
	}
	if len(s.VisibleEvents(1, 0)) != 1 {
		t.Fatal("known local event missing")
	}
}
