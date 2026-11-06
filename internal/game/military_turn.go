package game

import "fmt"

func (s *State) militarySnapshot() []fighter {
	var f []fighter
	for owner, t := range s.Tribes {
		for slot, a := range t.Armies {
			if a.People == 0 {
				continue
			}
			to := a.Position
			if a.Order.Kind == March || a.Order.Kind == Attack {
				to = a.Order.Target
			}
			f = append(f, fighter{Owner: owner, Slot: slot, People: a.People, From: a.Position, To: to, Supply: s.supplyAt(owner, a.Position), Forest: s.Tiles[a.Position.Y][a.Position.X].Terrain == Forest})
		}
	}
	return f
}
func (s *State) militaryAudience(p Point) uint8 {
	var audience uint8
	for id := range s.Tribes {
		if s.canSeeArmy(id, p) {
			audience |= 1 << id
		}
	}
	return audience
}
func (s *State) resolveMilitary() {
	f := s.militarySnapshot()
	r := resolveClashes(f)
	s.emitMilitaryResults(f, r)
	s.applyMilitaryResults(f, r)
	s.occupyAfterClashes(f, r)
	s.cleanBattleDeaths(f)
	for id := range s.Tribes {
		for slot := range s.Tribes[id].Armies {
			s.Tribes[id].Armies[slot].Order = ArmyOrder{}
		}
		s.revealArmies(id)
	}
}
func (s *State) emitMilitaryResults(f []fighter, r clashResult) {
	// Capture audiences while all observers still occupy their pre-battle positions.
	for i, a := range f {
		audience := s.militaryAudience(a.From) | s.militaryAudience(r.Positions[i]) | 1<<a.Owner
		text := fmt.Sprintf("%s军队%c行军至(%d,%d)。", s.Tribes[a.Owner].Name, 'A'+a.Slot, r.Positions[i].X, r.Positions[i].Y)
		if r.Engaged[i] {
			text = fmt.Sprintf("%s军队%c交锋损失%d人，剩余%d；补给%s。", s.Tribes[a.Owner].Name, 'A'+a.Slot, r.Losses[i], a.People-r.Losses[i], a.Supply)
		}
		if r.Engaged[i] || a.From != r.Positions[i] {
			s.Events = append(s.Events, Event{Month: s.Calendar.Elapsed, Audience: audience, Text: text})
		}
	}
}
func (s *State) applyMilitaryResults(f []fighter, r clashResult) {
	for i, a := range f {
		s.loseArmyPeople(a.Owner, a.Slot, r.Losses[i])
		army := &s.Tribes[a.Owner].Armies[a.Slot]
		if army.People == 0 {
			continue
		}
		army.Position = r.Positions[i]
		s.settleArmySupply(a.Owner, a.Slot, a.Supply)
	}
}
func (s *State) cleanBattleDeaths(f []fighter) {
	var participants [TribeCount]bool
	for _, a := range f {
		participants[a.Owner] = true
	}
	for id, participated := range participants {
		if participated && s.Tribes[id].Population == 0 {
			s.clearExtinct(id)
		}
	}
}
func (s *State) loseArmyPeople(id, slot, n int) {
	t := &s.Tribes[id]
	a := &t.Armies[slot]
	n = min(n, a.People)
	a.People -= n
	t.Population -= n
	t.Soldiers -= n
	if a.People == 0 {
		*a = Army{}
	}
}
func (s *State) settleArmySupply(id, slot int, level SupplyLevel) {
	a := &s.Tribes[id].Armies[slot]
	if level != CutOff {
		a.Unsupplied = 0
		return
	}
	a.Unsupplied++
	if a.Unsupplied <= SupplyGraceMonths {
		return
	}
	loss := (a.People*SupplyAttritionPercent + 99) / 100
	s.emitOwn(id, fmt.Sprintf("军队%c持续断供%d个月，损失%d人。", 'A'+slot, a.Unsupplied, loss))
	s.loseArmyPeople(id, slot, loss)
}

// Reserve soldiers absorb military famine losses first. Then trim deployed
// armies from C back to A in a documented, reproducible order.
func (s *State) reconcileArmies(id int) {
	t := &s.Tribes[id]
	excess := -t.ReserveSoldiers()
	for slot := MaxArmies - 1; slot >= 0 && excess > 0; slot-- {
		a := &t.Armies[slot]
		loss := min(excess, a.People)
		a.People -= loss
		excess -= loss
		if a.People == 0 {
			*a = Army{}
		}
	}
}
