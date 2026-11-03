package game

// WorkSlots is derived, never stored separately from the population it describes.
func (t Tribe) WorkSlots() int {
	if !t.Alive() {
		return 0
	}
	return (t.Population - t.Soldiers) / WorkersPerPlot
}

type Production struct {
	Slots     int
	Food      int
	Wood      int
	Farms     []Point
	Forests   []Point
	Unstaffed []Point
}

// ProductionFor is pure. View cells already have deterministic coordinate order.
func ProductionFor(v View, season Season) Production {
	p := Production{Slots: v.Self.WorkSlots()}
	if !v.Self.Alive() {
		return p
	}
	farms, forests := productionCandidates(v, season)
	reservedForests := min(v.Self.ForestPriority, p.Slots, len(forests))
	farmCount := min(p.Slots-reservedForests, len(farms))
	forestCount := min(p.Slots-farmCount, len(forests))
	p.Farms, p.Forests = farms[:farmCount], forests[:forestCount]
	p.Unstaffed = append(p.Unstaffed, farms[farmCount:]...)
	p.Unstaffed = append(p.Unstaffed, forests[forestCount:]...)
	p.Food = farmCount * FarmYield(season)
	p.Wood = forestCount * ForestYield
	return p
}

func productionCandidates(v View, season Season) (farms, forests []Point) {
	for _, c := range v.Cells {
		if c.Tile.Owner != v.Self.ID {
			continue
		}
		if c.Tile.Farm == FarmReady && FarmYield(season) > 0 {
			farms = append(farms, c.Point)
		}
		if c.Tile.Terrain == Forest {
			forests = append(forests, c.Point)
		}
	}
	return
}
