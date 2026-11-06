package game

import "fmt"

func CarryingCapacity(v View) int {
	if !v.Self.Alive() {
		return 0
	}
	capacity := 0
	for _, c := range v.Cells {
		if c.Tile.Owner == v.Self.ID {
			capacity += c.Tile.Capacity()
		}
	}
	return capacity
}

type GrowthInfo struct {
	Capacity int
	Reserve  int
	Rate     int    // Eligible monthly growth, in GrowthScale units per person.
	Added    int    // Actual births allowed by space and post-growth food reserves.
	Progress int    // Fraction remaining after this settlement; always below GrowthScale.
	Reason   string // Empty means the current month's growth conditions are met.
}

// GrowthFor is the single source of monthly growth for settlement and previews.
// Its view must contain food and hunger after this month's consumption.
func GrowthFor(v View) GrowthInfo {
	t := v.Self
	g := GrowthInfo{Capacity: CarryingCapacity(v), Progress: t.GrowthProgress}
	if !t.Alive() {
		g.Reason, g.Progress = "部落已灭亡", 0
		return g
	}
	// Saturate only for extreme, over-capacity records; never wrap a food target.
	next := max(t.Population, t.Population+1)
	g.Reserve = growthReserve(next, t.Soldiers)
	switch {
	case t.HungerMonths > 0:
		g.Reason, g.Progress = "连续缺粮尚未解除", 0
	case t.Population >= g.Capacity:
		g.Reason, g.Progress = "领土承载力已满", 0
	case t.Food < g.Reserve:
		g.Reason = fmt.Sprintf("存粮不足，至少需要 %d 粮", g.Reserve)
	default:
		g = eligibleGrowth(t, g)
	}
	return g
}

func growthRate(population, capacity int) int {
	if population <= 0 || population >= capacity {
		return 0
	}
	return min(population*GrowthPopulationRate, (capacity-population)*GrowthSpaceRate)
}

func growthReserve(population, soldiers int) int {
	need := FoodDemand(population, soldiers)
	limit := int(^uint(0) >> 1)
	if need > limit/GrowthReserveMonths {
		return limit
	}
	return need * GrowthReserveMonths
}

func eligibleGrowth(t Tribe, g GrowthInfo) GrowthInfo {
	g.Rate = growthRate(t.Population, g.Capacity)
	total := t.GrowthProgress + g.Rate
	g.Added = min(total/GrowthScale, g.Capacity-t.Population)
	for g.Added > 0 && t.Food < growthReserve(t.Population+g.Added, t.Soldiers) {
		g.Added--
	}
	// Blocked whole births are discarded, never banked for a later month.
	g.Progress = total % GrowthScale
	if t.Population+g.Added == g.Capacity {
		g.Progress = 0
	}
	g.Reserve = growthReserve(t.Population+max(1, g.Added), t.Soldiers)
	return g
}

func (s *State) grow(id int) {
	v, _ := s.View(id) // id comes from the fixed tribe array.
	g := GrowthFor(v)
	t := &s.Tribes[id]
	planned := (t.GrowthProgress + g.Rate) / GrowthScale
	t.GrowthProgress = g.Progress
	if g.Reason != "" {
		s.emitOwn(id, fmt.Sprintf("人口增长暂停：%s；增长进度 %d/%d。", g.Reason, g.Progress, GrowthScale))
		return
	}
	if planned > g.Added {
		s.emitOwn(id, "部分新增人口受到空间或安全存粮限制，不会囤积到下月。")
	}
	if g.Added == 0 {
		s.emitOwn(id, fmt.Sprintf("人口增长进度 %d/%d（本月 +%d/%d）。", g.Progress, GrowthScale, g.Rate, GrowthScale))
		return
	}
	t.Population += g.Added
	s.emitOwn(id, fmt.Sprintf("%s人口自然增长：新增 %d 名平民，人口现为 %d；增长余数 %d/%d。", t.Name, g.Added, t.Population, g.Progress, GrowthScale))
}

func validatePopulationRecords(t Tribe) error {
	if t.GrowthProgress < 0 || t.GrowthProgress >= GrowthScale || t.HungerMonths < 0 {
		return fmt.Errorf("人口增长或连续缺粮记录无效")
	}
	if t.HungerMonths > 0 && t.GrowthProgress != 0 {
		return fmt.Errorf("缺粮期间不得积累增长进度")
	}
	if !t.Alive() && (t.GrowthProgress != 0 || t.HungerMonths != 0) {
		return fmt.Errorf("灭亡部落人口记录未清理")
	}
	return nil
}
