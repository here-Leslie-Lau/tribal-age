package game

import "fmt"

func (t Tile) Capacity() int {
	if t.Settlement {
		return CapitalCapacity
	}
	return CapacityPerPlot
}
func (t Tile) Resting(month int) bool { return t.RestUntil > month }
func (t Tile) advanceFarm(month int, blocked bool) FarmStage {
	if t.Resting(month) || blocked {
		return t.Farm
	}
	return t.Farm.advance()
}

func (s *State) occupyAfterClashes(f []fighter, r clashResult) {
	// All captures precede any relocation or defeat, allowing simultaneous captures.
	for i, before := range f {
		a := s.Tribes[before.Owner].Armies[before.Slot]
		if a.People == 0 || a.Order.Kind != Occupy || r.Engaged[i] {
			continue
		}
		p := a.Position
		tile := &s.Tiles[p.Y][p.X]
		if tile.Owner == before.Owner || tile.Owner == Unowned {
			continue
		}
		old := tile.Owner
		tile.Owner = before.Owner
		tile.RestUntil = s.Calendar.Elapsed + 1
		s.Tribes[before.Owner].Explored[p.Y][p.X] = true
		s.Events = append(s.Events, Event{Month: s.Calendar.Elapsed, Audience: s.militaryAudience(p) | 1<<old | 1<<before.Owner, Text: fmt.Sprintf("%s接收(%d,%d)，保留设施；本月休整，下月生产。", s.Tribes[before.Owner].Name, p.X, p.Y)})
	}
	for id := range s.Tribes {
		s.relocateCapital(id)
	}
}
func (s *State) relocateCapital(id int) {
	t := &s.Tribes[id]
	if !t.Alive() || s.Tiles[t.Capital.Y][t.Capital.X].Owner == id {
		return
	}
	for y, row := range s.Tiles {
		for x, tile := range row {
			if tile.Owner == id {
				t.Capital = Point{x, y}
				s.emitOwn(id, fmt.Sprintf("首都失守，迁都至(%d,%d)；原聚落设施留在原地，补给下月重算。", x, y))
				return
			}
		}
	}
	t.Defeated = true
	t.Soldiers, t.Food, t.Wood, t.AP = 0, 0, 0, 0
	t.GrowthProgress, t.HungerMonths, t.ForestPriority = 0, 0, 0
	t.Armies = [MaxArmies]Army{}
	s.emitOwn(id, fmt.Sprintf("%s失去全部领土，战败退出；尚存%d人未计为战死，也不归入胜方。", t.Name, t.Population))
}
func (v View) BlockedProduction(p Point) bool {
	for _, a := range v.Armies {
		if a.Owner != v.Self.ID && a.Position == p {
			return true
		}
	}
	return false
}

func (v View) ProductionPause(c Cell) string {
	if c.Tile.Resting(v.Calendar.Elapsed) {
		return "占领休整"
	}
	if v.BlockedProduction(c.Point) {
		return "敌军驻扎"
	}
	return ""
}
func (s *State) explainPausedProduction(v View) {
	for _, c := range v.Cells {
		if c.Tile.Owner != v.Self.ID {
			continue
		}
		if reason := v.ProductionPause(c); reason != "" {
			s.emitOwn(v.Self.ID, fmt.Sprintf("地块(%d,%d)%s，暂停生产和建设。", c.Point.X, c.Point.Y, reason))
		}
	}
}
