package game

import (
	"reflect"
	"testing"
)

func TestViewAndAIDoNotSeeHiddenInformation(t *testing.T) {
	s := scenarioState()
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Cells) != 1 || v.Cells[0].Point != s.Tribes[0].Capital {
		t.Fatal("view contains unexplored cells")
	}
	a, ok := ChooseAction(v)
	if !ok || a.Kind != BuildFarm {
		t.Fatal("AI should first establish food production")
	}
	// Changing hidden terrain, ownership, enemy stock and enemy fog must not affect the view or decision.
	s.Tiles[0][0].Terrain = Ocean
	s.Tiles[0][0].Owner = 1
	s.Tribes[1].Explored[0][0] = true
	s.Tribes[1].Food = 999
	s.Tribes[1].Wood = 888
	s.Tribes[1].Population = 55
	s.Tribes[1].GrowthProgress = 2
	s.Tribes[1].ForestPriority = 4
	other, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	b, found := ChooseAction(other)
	if !reflect.DeepEqual(v, other) || !found || a != b {
		t.Fatal("hidden information influenced AI")
	}
	v.Self.Food = 0
	v.Self.Explored[0][0] = true
	v.Cells[0].Tile.Owner = 2
	v.Legal[0].Kind = Train
	if s.Tribes[0].Food != 60 || s.Tribes[0].Explored[0][0] || s.Tiles[3][2].Owner != 0 {
		t.Fatal("view shares mutable state")
	}
	if _, err := s.View(-1); err == nil {
		t.Fatal("invalid view accepted")
	}
	if _, err := s.View(TribeCount); err == nil {
		t.Fatal("invalid view accepted")
	}
}

func TestAIUsesLegalActionsAndCoordinateOrder(t *testing.T) {
	s := scenarioState()
	s.Tiles[3][2].Farm = FarmPlanned
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := ChooseAction(v)
	if !ok || a != (Action{Explore, Point{2, 2}}) {
		t.Fatalf("wrong exploration tie-break: %+v", a)
	}
	if err := s.Apply(0, a); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Tribes[0].AP = 0
	v, err = s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ChooseAction(v); ok {
		t.Fatal("AI acted without AP")
	}
}

func TestMonthlyAIDispatch(t *testing.T) {
	t.Run("all three tribes once", func(t *testing.T) {
		s := New(42)
		if err := s.EndMonth(Simulation); err != nil {
			t.Fatal(err)
		}
		for _, tribe := range s.Tribes {
			if tribe.Wood != 10 || tribe.Food != 53 || tribe.AP != 2 || s.Tiles[tribe.Capital.Y][tribe.Capital.X].Farm.Progress() != 1 {
				t.Fatalf("AI skipped, repeated, or used different rules: %+v", tribe)
			}
		}
	})
	t.Run("player not automated or replayed", func(t *testing.T) {
		s := New(42)
		p := s.Tribes[0].Capital
		if err := s.Apply(0, Action{BuildFarm, p}); err != nil {
			t.Fatal(err)
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Wood != 10 || s.Tribes[0].Food != 53 || s.Tribes[0].AP != 2 {
			t.Fatal("player action replayed")
		}
		if s.Tribes[0].Explored[p.Y-1][p.X] {
			t.Fatal("AI also controlled player")
		}
		if s.Tribes[1].Wood != 10 || s.Tribes[2].Wood != 10 {
			t.Fatal("AI tribes did not act")
		}
	})
}
