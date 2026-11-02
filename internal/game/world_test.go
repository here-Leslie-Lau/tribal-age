package game

import (
	"reflect"
	"testing"
)

func TestGeneratedWorld(t *testing.T) {
	for seed := int64(-20); seed < 80; seed++ {
		s := New(seed)
		if err := s.Validate(); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if !reflect.DeepEqual(s, New(seed)) {
			t.Fatalf("seed %d not reproducible", seed)
		}
		var terrains [5]bool
		for _, row := range s.Tiles {
			for _, tile := range row {
				terrains[tile.Terrain] = true
			}
		}
		for terrain, found := range terrains {
			if !found {
				t.Fatalf("seed %d missing terrain %d", seed, terrain)
			}
		}
		for id, tribe := range s.Tribes {
			p := tribe.Capital
			if s.Tiles[p.Y][p.X].Terrain != Grassland {
				t.Fatal("capital not on grassland")
			}
			grass, forest, owned := 0, 0, 0
			for _, n := range p.Neighbors() {
				if !n.Valid() {
					continue
				}
				tile := s.Tiles[n.Y][n.X]
				if tile.Owner != Unowned {
					continue
				}
				if tile.Terrain == Grassland {
					grass++
				}
				if tile.Terrain == Forest {
					forest++
				}
			}
			for _, row := range s.Tiles {
				for _, tile := range row {
					if tile.Owner == id {
						owned++
					}
				}
			}
			if grass == 0 || forest == 0 || owned != 1 {
				t.Fatalf("seed %d invalid starting space", seed)
			}
			for other := id + 1; other < TribeCount; other++ {
				q := s.Tribes[other].Capital
				if distance(p, q) < 6 || !reachable(s, p, q) {
					t.Fatalf("seed %d capitals too close or disconnected", seed)
				}
			}
		}
	}
	if New(1).Tiles == New(2).Tiles {
		t.Fatal("different seeds made identical worlds")
	}
}

func distance(a, b Point) int { return abs(a.X-b.X) + abs(a.Y-b.Y) }
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func reachable(s *State, from, to Point) bool {
	seen := [Height][Width]bool{}
	queue := []Point{from}
	seen[from.Y][from.X] = true
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == to {
			return true
		}
		for _, n := range p.Neighbors() {
			if n.Valid() && !seen[n.Y][n.X] && s.Tiles[n.Y][n.X].Terrain.Passable() {
				seen[n.Y][n.X] = true
				queue = append(queue, n)
			}
		}
	}
	return false
}
