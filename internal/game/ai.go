package game

import "fmt"

type aiEconomy struct{ farms, forests, bareGrass, required int }

func assessEconomy(v View) aiEconomy {
	annualYield := MonthsPerSeason * (FarmYield(Spring) + FarmYield(Summer) + FarmYield(Autumn) + FarmYield(Winter))
	need := FoodDemand(v.Self.Population, v.Self.Soldiers) * MonthsPerYear
	e := aiEconomy{required: (need + annualYield - 1) / annualYield}
	for _, c := range v.Cells {
		if c.Tile.Owner != v.Self.ID {
			continue
		}
		if c.Tile.Farm != FarmNone {
			e.farms++
		}
		if c.Tile.Terrain == Forest {
			e.forests++
		}
		if c.Tile.Terrain == Grassland && c.Tile.Farm == FarmNone {
			e.bareGrass++
		}
	}
	return e
}

// ChooseAction is pure: its only input is the tribe's information-limited view.
func ChooseAction(v View) (Action, bool) {
	e := assessEconomy(v)
	var tiles [Height][Width]Tile
	for _, c := range v.Cells {
		tiles[c.Point.Y][c.Point.X] = c.Tile
	}
	bestScore := 0
	var best Action
	for _, a := range v.Legal {
		score := actionScore(a, tiles[a.Target.Y][a.Target.X], v.Self.Wood, e)
		if score > bestScore {
			best, bestScore = a, score
		}
	}
	return best, bestScore > 0
}

func actionScore(a Action, tile Tile, wood int, e aiEconomy) int {
	switch a.Kind {
	case BuildFarm:
		if e.farms == 0 {
			return 100
		}
		if e.forests > 0 && e.farms < e.required {
			return 70
		}
	case Claim:
		return claimScore(tile.Terrain, wood, e)
	case Explore:
		if e.forests == 0 {
			return 80
		}
		if e.farms < e.required && e.bareGrass == 0 {
			return 50
		}
		return 10
	}
	return 0
}

func claimScore(terrain Terrain, wood int, e aiEconomy) int {
	if e.forests == 0 {
		if terrain == Forest {
			return 90
		}
		return 0
	}
	if e.farms < e.required {
		if e.bareGrass == 0 && terrain == Grassland {
			return 60
		}
		return 0
	}
	// Keep one farm's cost in reserve before optional expansion.
	if wood < ClaimWood+FarmWood {
		return 0
	}
	if terrain == Forest {
		return 40
	}
	return 30
}

func (s *State) runAI(id int) error {
	for step := 0; step < MonthlyAP; step++ {
		view, err := s.View(id)
		if err != nil {
			return err
		}
		a, ok := ChooseAction(view)
		if !ok {
			return nil
		}
		if err := s.Apply(id, a); err != nil {
			return fmt.Errorf("AI 行动失败: %w", err)
		}
	}
	return nil
}
