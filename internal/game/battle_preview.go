package game

// EngagementEstimate is conditional: one selected army attacks the currently
// visible defenders where they stand. It cannot predict enemy orders or aid.
type EngagementEstimate struct {
	Available          bool
	OwnLoss, EnemyLoss int
}

func AttackEstimate(v View, slot int) EngagementEstimate {
	if slot < 0 || slot >= MaxArmies {
		return EngagementEstimate{}
	}
	a := v.Self.Armies[slot]
	if a.People == 0 || a.Order.Kind != Attack {
		return EngagementEstimate{}
	}
	forces := []fighter{{Owner: v.Self.ID, Slot: slot, People: a.People, From: a.Position, To: a.Order.Target, Supply: armySupply(v, slot)}}
	forces = append(forces, visibleDefenders(v, a.Order.Target)...)
	if len(forces) == 1 {
		return EngagementEstimate{}
	}
	result := resolveClashes(forces)
	e := EngagementEstimate{Available: true, OwnLoss: result.Losses[0]}
	for _, loss := range result.Losses[1:] {
		e.EnemyLoss += loss
	}
	return e
}
func visibleDefenders(v View, p Point) []fighter {
	var forces []fighter
	tile, _ := armyKnownTile(v, p)
	for _, a := range v.Armies {
		if a.Owner != v.Self.ID && a.Position == p {
			forces = append(forces, fighter{Owner: a.Owner, Slot: a.Slot, People: a.People, From: p, To: p, Supply: a.Supply, Forest: tile.Terrain == Forest})
		}
	}
	return forces
}
