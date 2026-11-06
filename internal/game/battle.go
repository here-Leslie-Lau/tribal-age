package game

import "math/big"

type fighter struct {
	Owner, Slot, People int
	From, To            Point
	Supply              SupplyLevel
	Forest              bool
}
type clashResult struct {
	Positions []Point
	Engaged   []bool
	Losses    []int
}

// All intents are resolved from copies. Blocked departures propagate in batches,
// not in actor order, so a stopped army cannot be walked through by a third party.
func resolveClashes(f []fighter) clashResult {
	n := len(f)
	r := clashResult{Positions: make([]Point, n), Engaged: make([]bool, n), Losses: make([]int, n)}
	edges := make([][]bool, n)
	for i := range f {
		r.Positions[i] = f[i].To
		edges[i] = make([]bool, n)
	}
	for pass := 0; pass <= n; pass++ {
		blocked := collisionEdges(f, r.Positions, edges)
		changed := stopClashingArmies(f, blocked, &r)
		if !changed {
			break
		}
	}
	for i := range f {
		r.Losses[i] = incomingLoss(f, edges, i)
	}
	return r
}
func collides(a, b fighter, toA, toB Point) bool {
	return toA == toB || (toA == b.From && toB == a.From)
}

func collisionEdges(f []fighter, positions []Point, edges [][]bool) []bool {
	blocked := make([]bool, len(f))
	for i := range f {
		for j := i + 1; j < len(f); j++ {
			if f[i].Owner != f[j].Owner && collides(f[i], f[j], positions[i], positions[j]) {
				edges[i][j], edges[j][i] = true, true
				blocked[i], blocked[j] = true, true
			}
		}
	}
	return blocked
}
func stopClashingArmies(f []fighter, blocked []bool, r *clashResult) bool {
	changed := false
	for i := range f {
		if !blocked[i] {
			continue
		}
		r.Engaged[i] = true
		if r.Positions[i] != f[i].From {
			r.Positions[i] = f[i].From
			changed = true
		}
	}
	return changed
}

// Exact rational accumulation avoids division/rounding bonuses for army IDs.
// Each source divides its power by directly engaged enemy headcount, not flags.
func incomingLoss(f []fighter, edges [][]bool, victim int) int {
	damage := new(big.Rat)
	for source := range f {
		if !edges[source][victim] {
			continue
		}
		heads := 0
		for target := range f {
			if edges[source][target] {
				heads += f[target].People
			}
		}
		share := new(big.Rat).SetFrac64(int64(f[source].People)*int64(f[source].Supply.Percent())*int64(f[victim].People), int64(heads))
		damage.Add(damage, share)
	}
	defense := 100
	if f[victim].Forest && f[victim].From == f[victim].To {
		defense = ForestDefensePercent
	}
	damage.Mul(damage, big.NewRat(BattleDamagePercent, int64(100*defense)))
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(damage.Num(), damage.Denom(), rem)
	if rem.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return min(f[victim].People, int(q.Int64()))
}
