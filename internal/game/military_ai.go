package game

// Defensive AI has no access to enemy orders, stores or armies outside its view.
func (s *State) prepareDefense(id int) error {
	v, _ := s.View(id)
	if _, ok := nearestArmyEnemy(v, v.Self.Capital); !ok {
		return nil
	}
	slot := defenseSlot(v.Self)
	if slot < 0 || v.Self.AP < ActionAP || s.hostileAt(id, v.Self.Capital) {
		return nil
	}
	if s.shouldRecruitDefense(id) {
		if err := s.Apply(id, Action{Kind: Train}); err != nil {
			return err
		}
	}
	t := s.Tribes[id]
	if t.ReserveSoldiers() == 0 {
		return nil
	}
	return s.SetArmySize(id, slot, min(MaxArmyPeople, t.ReserveSoldiers()+t.Armies[slot].People))
}
func defenseSlot(t Tribe) int {
	for slot, a := range t.Armies {
		if a.People > 0 {
			if a.Position == t.Capital && a.Order.Kind == Hold {
				return slot
			}
			return -1
		}
	}
	return 0
}
func (s *State) shouldRecruitDefense(id int) bool {
	t := s.Tribes[id]
	return t.ReserveSoldiers() == 0 && t.AP >= 2*ActionAP &&
		t.Food >= growthReserve(t.Population, t.Soldiers+TrainingSize) && s.checkAction(id, Action{Kind: Train}) == nil
}
func ArmyOrdersFor(v View) [MaxArmies]ArmyOrder {
	var orders [MaxArmies]ArmyOrder
	for slot, a := range v.Self.Armies {
		if a.People == 0 {
			continue
		}
		orders[slot] = armyOrderFor(v, slot, a)
	}
	return orders
}
func nearestArmyEnemy(v View, p Point) (ArmyInfo, bool) {
	var best ArmyInfo
	found := false
	for _, a := range v.Armies {
		if a.Owner != v.Self.ID && (!found || manhattan(p, a.Position) < manhattan(p, best.Position)) {
			best = a
			found = true
		}
	}
	return best, found
}
func armyKnownTile(v View, p Point) (Tile, bool) {
	for _, c := range v.Cells {
		if c.Point == p {
			return c.Tile, true
		}
	}
	return Tile{}, false
}
func armySupply(v View, slot int) SupplyLevel {
	for _, a := range v.Armies {
		if a.Owner == v.Self.ID && a.Slot == slot {
			return a.Supply
		}
	}
	return CutOff
}
func armyPathStep(v View, from, goal Point) (Point, bool) {
	passable := armyPassableCells(v, goal)
	if from == goal || !passable[goal.Y][goal.X] {
		return Point{}, false
	}
	distances := floodDistances([]Point{goal}, Width*Height, func(p Point) bool { return passable[p.Y][p.X] })
	if distances[from.Y][from.X] < 0 {
		return Point{}, false
	}
	for _, n := range from.Neighbors() {
		if n.Valid() && distances[n.Y][n.X] >= 0 && distances[n.Y][n.X] < distances[from.Y][from.X] {
			return n, true
		}
	}
	return Point{}, false
}
func armyPassableCells(v View, goal Point) [Height][Width]bool {
	var passable [Height][Width]bool
	for _, c := range v.Cells {
		passable[c.Point.Y][c.Point.X] = c.Tile.Terrain.Passable()
	}
	for _, a := range v.Armies {
		if a.Owner != v.Self.ID && a.Position != goal {
			passable[a.Position.Y][a.Position.X] = false
		}
	}
	return passable
}
func armyOrderFor(v View, slot int, a Army) ArmyOrder {
	if a.Order.Kind != Hold {
		return a.Order
	}
	goal := v.Self.Capital
	if enemy, ok := nearestArmyEnemy(v, a.Position); ok {
		goal = enemy.Position
	} else if canArmyOccupy(v, slot, a.Position) {
		return ArmyOrder{Kind: Occupy, Target: a.Position}
	}
	if step, ok := armyPathStep(v, a.Position, goal); ok {
		return ArmyOrder{Kind: ArmyApproachKind(v, step), Target: step}
	}
	return ArmyOrder{}
}
func canArmyOccupy(v View, slot int, p Point) bool {
	tile, known := armyKnownTile(v, p)
	return known && tile.Owner != v.Self.ID && tile.Owner != Unowned && armySupply(v, slot) != CutOff
}

// ArmyApproachKind chooses from current visible troops, never hidden orders.
func ArmyApproachKind(v View, p Point) ArmyOrderKind {
	for _, a := range v.Armies {
		if a.Owner != v.Self.ID && a.Position == p {
			return Attack
		}
	}
	return March
}
func (s *State) planMilitaryAI(first int) error {
	var plans [TribeCount][MaxArmies]ArmyOrder
	// Read every view before writing any plan. AI never receives opponents' orders.
	for id := first; id < TribeCount; id++ {
		if s.Tribes[id].Alive() {
			v, _ := s.View(id)
			plans[id] = ArmyOrdersFor(v)
		}
	}
	for id := first; id < TribeCount; id++ {
		for slot, a := range s.Tribes[id].Armies {
			if a.People > 0 {
				if err := s.SetArmyOrder(id, slot, plans[id][slot]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
