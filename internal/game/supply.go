package game

type SupplyLevel uint8

const (
	Supplied SupplyLevel = iota
	Strained
	CutOff
)

func (l SupplyLevel) String() string {
	switch l {
	case Supplied:
		return "正常"
	case Strained:
		return "吃紧"
	default:
		return "断供"
	}
}
func (l SupplyLevel) Percent() int {
	switch l {
	case Supplied:
		return SupplyNormalPercent
	case Strained:
		return SupplyStrainedPercent
	default:
		return SupplyCutPercent
	}
}

// supplyAt uses the actual board, but only the observable condition is exposed.
// No enemy stocks or routes are included in the player's information.
func (s *State) supplyAt(id int, p Point) SupplyLevel {
	distances := s.supplyDistances(id)
	d := distances[p.Y][p.X]
	if d < 0 || d > SupplyStrainedRange {
		return CutOff
	}
	if d > SupplyNormalRange {
		return Strained
	}
	return Supplied
}
func (s *State) supplyDistances(id int) [Height][Width]int {
	blocked := s.supplyBlockers(id)
	root := s.Tribes[id].Capital
	if !s.Tribes[id].Alive() || blocked[root.Y][root.X] || s.Tiles[root.Y][root.X].Owner != id {
		return emptyDistances()
	}
	home := floodDistances([]Point{root}, Width*Height, func(p Point) bool { return !blocked[p.Y][p.X] && s.Tiles[p.Y][p.X].Owner == id })
	return floodDistances(reachedCells(home), SupplyStrainedRange, func(p Point) bool { return !blocked[p.Y][p.X] && s.Tiles[p.Y][p.X].Terrain.Passable() })
}
func (s *State) supplyBlockers(id int) [Height][Width]bool {
	var blocked [Height][Width]bool
	for owner, t := range s.Tribes {
		if owner == id {
			continue
		}
		for _, a := range t.Armies {
			if a.People > 0 {
				blocked[a.Position.Y][a.Position.X] = true
			}
		}
	}
	return blocked
}
func emptyDistances() [Height][Width]int {
	var d [Height][Width]int
	for y := range d {
		for x := range d[y] {
			d[y][x] = -1
		}
	}
	return d
}
func reachedCells(d [Height][Width]int) []Point {
	var cells []Point
	for y, row := range d {
		for x, distance := range row {
			if distance >= 0 {
				cells = append(cells, Point{x, y})
			}
		}
	}
	return cells
}
func floodDistances(seeds []Point, limit int, allowed func(Point) bool) [Height][Width]int {
	d := emptyDistances()
	queue := append([]Point(nil), seeds...)
	for _, p := range seeds {
		d[p.Y][p.X] = 0
	}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		if d[p.Y][p.X] >= limit {
			continue
		}
		for _, n := range p.Neighbors() {
			if n.Valid() && d[n.Y][n.X] < 0 && allowed(n) {
				d[n.Y][n.X] = d[p.Y][p.X] + 1
				queue = append(queue, n)
			}
		}
	}
	return d
}
