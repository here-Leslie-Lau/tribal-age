package game

func battleKnowledge(f []fighter, r clashResult, owner int) []bool {
	known := make([]bool, len(f))
	for i, a := range f {
		if a.Owner != owner || !r.Engaged[i] {
			continue
		}
		known[i] = true
		for j, contact := range r.Contacts[i] {
			if contact {
				known[j] = true
			}
		}
	}
	return known
}

// Connected components of the known graph prevent duplicate casualty credit.
// Hidden contacts never pull their troops or positions into this observer's report.
func knownClashes(f []fighter, r clashResult, owner int) []BattleReport {
	known := battleKnowledge(f, r, owner)
	seen := make([]bool, len(f))
	var battles []BattleReport
	for start := range f {
		if !known[start] || seen[start] {
			continue
		}
		group := []int{start}
		seen[start] = true
		for next := 0; next < len(group); next++ {
			for j, contact := range r.Contacts[group[next]] {
				if contact && known[j] && !seen[j] {
					seen[j] = true
					group = append(group, j)
				}
			}
		}
		battles = append(battles, clashReport(f, r, known, group))
	}
	return battles
}
func clashReport(f []fighter, r clashResult, known []bool, group []int) BattleReport {
	var b BattleReport
	// Stable snapshot ordering, regardless of graph traversal order.
	member := make([]bool, len(f))
	for _, i := range group {
		member[i] = true
	}
	for i, a := range f {
		if !member[i] {
			continue
		}
		b.Armies = append(b.Armies, BattleArmy{a.Owner, a.Slot, a.People, r.Losses[i], a.From, a.To, a.Supply, a.Forest && a.From == a.To})
		for j, contact := range r.Contacts[i] {
			if contact && !known[j] {
				b.Partial = true
			}
		}
	}
	return b
}
func (s *State) recordClashes(f []fighter, r clashResult) {
	for owner := range s.Tribes {
		battles := knownClashes(f, r, owner)
		if len(battles) == 0 {
			continue
		}
		report := s.currentMilitaryReport(owner)
		report.Battles = append(report.Battles, battles...)
	}
}
