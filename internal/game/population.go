package game

import "fmt"

func CarryingCapacity(v View) int {
	if !v.Self.Alive() {
		return 0
	}
	capacity := CapitalCapacity
	for _, c := range v.Cells {
		if c.Tile.Owner == v.Self.ID && c.Point != v.Self.Capital {
			capacity += CapacityPerPlot
		}
	}
	return capacity
}

type GrowthInfo struct {
	Capacity int
	Reserve  int
	Reason   string // Empty means the current month's growth conditions are met.
}

// GrowthFor is the single source of eligibility for settlement and player status.
func GrowthFor(v View) GrowthInfo {
	g := GrowthInfo{Capacity: CarryingCapacity(v)}
	if !v.Self.Alive() {
		g.Reason = "部落已灭亡"
		return g
	}
	g.Reserve = FoodDemand(v.Self.Population+GrowthPeople, v.Self.Soldiers) * GrowthReserveMonths
	switch {
	case v.Self.HungerMonths > 0:
		g.Reason = "连续缺粮尚未解除"
	case v.Self.Population+GrowthPeople > g.Capacity:
		g.Reason = "领土承载力已满"
	case v.Self.Food < g.Reserve:
		g.Reason = fmt.Sprintf("存粮不足，至少需要 %d 粮", g.Reserve)
	}
	return g
}

func (s *State) grow(id int) {
	v, _ := s.View(id) // id comes from the fixed tribe array.
	g := GrowthFor(v)
	t := &s.Tribes[id]
	if g.Reason != "" {
		t.GrowthProgress = 0
		s.emitOwn(id, fmt.Sprintf("人口增长暂停：%s。", g.Reason))
		return
	}
	t.GrowthProgress++
	if t.GrowthProgress < GrowthMonths {
		s.emitOwn(id, fmt.Sprintf("人口增长进度 %d/%d。", t.GrowthProgress, GrowthMonths))
		return
	}
	t.Population += GrowthPeople
	t.GrowthProgress = 0
	s.emitOwn(id, fmt.Sprintf("%s人口自然增长：新增 %d 名平民，人口现为 %d。", t.Name, GrowthPeople, t.Population))
}

func validatePopulationRecords(t Tribe) error {
	if t.GrowthProgress < 0 || t.GrowthProgress >= GrowthMonths || t.HungerMonths < 0 {
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
