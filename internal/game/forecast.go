package game

// Forecast describes this month's production and food payment without changing
// stocks, population, progress, or the calendar. Queued military outcomes are not predicted.
type Forecast struct {
	Production Production
	Demand     int
	NetFood    int // Production minus demand, even if stocks cannot cover it.
	FoodAfter  int
	Shortage   int
	Growth     GrowthInfo
}

func ForecastFor(v View) Forecast {
	v.Cells = append([]Cell(nil), v.Cells...)
	for i := range v.Cells {
		if v.Cells[i].Tile.Owner == v.Self.ID {
			v.Cells[i].Tile.Farm = v.Cells[i].Tile.advanceFarm(v.Calendar.Elapsed, v.BlockedProduction(v.Cells[i].Point))
		}
	}
	p := ProductionFor(v, v.Calendar.Season())
	f := Forecast{Production: p, Demand: FoodDemand(v.Self.Population, v.Self.Soldiers)}
	f.NetFood = p.Food - f.Demand
	balance := v.Self.Food + f.NetFood
	f.FoodAfter, f.Shortage = max(0, balance), max(0, -balance)
	// Eligibility uses food after payment, not the pre-settlement inventory.
	v.Self.Food = f.FoodAfter
	if f.Shortage > 0 {
		v.Self.HungerMonths++
	} else {
		v.Self.HungerMonths = 0
	}
	f.Growth = GrowthFor(v)
	return f
}
