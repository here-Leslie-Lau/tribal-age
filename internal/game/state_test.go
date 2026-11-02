package game

import (
	"math/rand"
	"testing"
)

// Small, independent scenarios use an otherwise empty grassland map.
func scenarioState() *State {
	s := &State{Seed: 42, rng: rand.New(rand.NewSource(42))}
	for y := range s.Tiles {
		for x := range s.Tiles[y] {
			s.Tiles[y][x] = Tile{Terrain: Grassland, Owner: Unowned}
		}
	}
	for id := range s.Tribes {
		p := Point{X: 2 + id*8, Y: 3}
		s.Tribes[id] = newTribe(id, p)
		s.Tiles[p.Y][p.X].Owner = id
	}
	return s
}

func TestInitialTribe(t *testing.T) {
	tribe := newTribe(0, Point{2, 3})
	if tribe.Population != 30 || tribe.Soldiers != 5 || tribe.Food != 60 || tribe.Wood != 20 || tribe.AP != 2 {
		t.Fatalf("unexpected initial tribe: %+v", tribe)
	}
	count := 0
	for _, row := range tribe.Explored {
		for _, known := range row {
			if known {
				count++
			}
		}
	}
	if count != 1 || !tribe.Explored[3][2] {
		t.Fatal("only the capital should initially be explored")
	}
}

func TestStateInvariants(t *testing.T) {
	if err := scenarioState().Validate(); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		breakState func(*State)
	}{
		{"food", func(s *State) { s.Tribes[0].Food = -1 }},
		{"wood", func(s *State) { s.Tribes[0].Wood = -1 }},
		{"population", func(s *State) { s.Tribes[0].Population = -1 }},
		{"soldiers", func(s *State) { s.Tribes[0].Soldiers = 31 }},
		{"negative soldiers", func(s *State) { s.Tribes[0].Soldiers = -1 }},
		{"action points", func(s *State) { s.Tribes[0].AP = 3 }},
		{"negative action points", func(s *State) { s.Tribes[0].AP = -1 }},
		{"tribe identity", func(s *State) { s.Tribes[0].ID = 2 }},
		{"capital", func(s *State) { s.Tribes[0].Capital.X = -1 }},
		{"lost capital", func(s *State) { s.Tiles[3][2].Owner = Unowned }},
		{"ownership", func(s *State) { s.Tiles[0][0].Owner = 99 }},
		{"unexplored territory", func(s *State) { s.Tribes[0].Explored[3][2] = false }},
		{"terrain", func(s *State) { s.Tiles[0][0].Terrain = Terrain(99) }},
		{"impassable territory", func(s *State) { s.Tiles[3][2].Terrain = Ocean }},
		{"farm on forest", func(s *State) { s.Tiles[3][2] = Tile{Terrain: Forest, Owner: 0, Farm: FarmReady} }},
		{"unowned farm", func(s *State) { s.Tiles[0][0].Farm = FarmPlanned }},
		{"farm stage", func(s *State) { s.Tiles[3][2].Farm = FarmStage(99) }},
		{"calendar", func(s *State) { s.Calendar.Elapsed = -1 }},
		{"unclean extinction", func(s *State) { s.Tribes[0].Population = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scenarioState()
			tt.breakState(s)
			if err := s.Validate(); err == nil {
				t.Fatal("invalid state accepted")
			}
		})
	}
}

func TestCalendar(t *testing.T) {
	c := Calendar{}
	if c.Year() != 1 || c.Month() != 3 || c.Season() != Spring {
		t.Fatalf("wrong opening date: %+v", c)
	}
	seasons := []Season{Spring, Spring, Spring, Summer, Summer, Summer, Autumn, Autumn, Autumn, Winter, Winter, Winter}
	for i, want := range seasons {
		if c.Season() != want || c.Month() != (i+2)%12+1 {
			t.Fatalf("month %d: got %v", i, c)
		}
		c.Advance()
	}
	for c.Elapsed < 120 {
		c.Advance()
	}
	if c.Year() != 11 || c.Month() != 3 {
		t.Fatalf("120 months should end at year 11, March: %+v", c)
	}
	c = Calendar{Elapsed: 9}
	if c.Year() != 1 || c.Month() != 12 {
		t.Fatal("December boundary")
	}
	c.Advance()
	if c.Year() != 2 || c.Month() != 1 {
		t.Fatal("year boundary")
	}
}
