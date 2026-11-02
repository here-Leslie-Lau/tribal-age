package game

import "math/rand"

func New(seed int64) *State {
	s := &State{Seed: seed, rng: rand.New(rand.NewSource(seed))}
	choices := [...]Terrain{Grassland, Grassland, Grassland, Grassland, Grassland, Forest, Forest, Forest, Mountain, River, Ocean}
	for y := range s.Tiles {
		for x := range s.Tiles[y] {
			terrain := choices[s.rng.Intn(len(choices))]
			if x == 0 || y == 0 || x == Width-1 || y == Height-1 {
				terrain = Ocean
			}
			s.Tiles[y][x] = Tile{Terrain: terrain, Owner: Unowned}
		}
	}
	// Disjoint horizontal bands guarantee spacing without rejection sampling.
	var capitals [TribeCount]Point
	band := Width / TribeCount
	for id := range capitals {
		capitals[id] = Point{band*id + 2 + s.rng.Intn(band-MinSpawnDistance+1), 3 + s.rng.Intn(Height-6)}
		if id > 0 {
			s.connect(capitals[id-1], capitals[id])
		}
	}
	for id, p := range capitals {
		s.Tiles[p.Y][p.X] = Tile{Terrain: Grassland, Owner: id}
		s.Tiles[p.Y-1][p.X].Terrain = Forest
		s.Tiles[p.Y+1][p.X].Terrain = Grassland
		s.Tribes[id] = newTribe(id, p)
	}
	// These cells are outside all paths and starting neighborhoods.
	s.Tiles[1][1].Terrain = Mountain
	s.Tiles[1][2].Terrain = River
	return s
}

func (s *State) connect(from, to Point) {
	p := from
	for p.X != to.X {
		s.Tiles[p.Y][p.X].Terrain = Grassland
		if p.X < to.X {
			p.X++
		} else {
			p.X--
		}
	}
	for p.Y != to.Y {
		s.Tiles[p.Y][p.X].Terrain = Grassland
		if p.Y < to.Y {
			p.Y++
		} else {
			p.Y--
		}
	}
	s.Tiles[p.Y][p.X].Terrain = Grassland
}
