package game

import "fmt"

type Mode uint8

const (
	Interactive Mode = iota
	Simulation
)

func (s *State) EndMonth(mode Mode) error {
	if mode != Interactive && mode != Simulation {
		return fmt.Errorf("未知运行模式")
	}
	if s.AllExtinct() {
		return fmt.Errorf("所有部落已经灭亡")
	}
	if mode == Interactive && !s.Tribes[0].Alive() {
		return fmt.Errorf("玩家部落已经%s", s.Tribes[0].Status())
	}
	if err := s.runMonthlyAI(mode); err != nil {
		return err
	}
	s.resolveMilitary()
	s.advanceFarms()
	s.produce()
	newlyExtinct := s.settlePopulation()
	s.Calendar.Advance()
	s.finishMonth(newlyExtinct)
	return nil
}

func (s *State) advanceFarms() {
	for y := range s.Tiles {
		for x := range s.Tiles[y] {
			tile := &s.Tiles[y][x]
			next := tile.advanceFarm(s.Calendar.Elapsed, s.hostileAt(tile.Owner, Point{x, y}))
			if next == tile.Farm {
				continue
			}
			tile.Farm = next
			message := fmt.Sprintf("农田 (%d,%d) 推进至 %d/%d。", x, y, tile.Farm.Progress(), FarmMonths)
			if tile.Farm == FarmReady {
				message = fmt.Sprintf("农田 (%d,%d) 已完工（%d/%d）。", x, y, FarmMonths, FarmMonths)
			}
			s.emitAt(tile.Owner, Point{x, y}, message)
		}
	}
}

func (s *State) produce() {
	for id := range s.Tribes {
		if !s.Tribes[id].Alive() {
			continue
		}
		v, _ := s.View(id) // id comes from the fixed tribe array.
		s.explainPausedProduction(v)
		p := ProductionFor(v, s.Calendar.Season())
		s.Tribes[id].Food += p.Food
		s.Tribes[id].Wood += p.Wood
		s.emitOwn(id, fmt.Sprintf("生产入库：粮食 +%d，木材 +%d。", p.Food, p.Wood))
		for _, point := range p.Unstaffed {
			s.emitOwn(id, fmt.Sprintf("生产地块 (%d,%d) 人手不足，本月停产。", point.X, point.Y))
		}
	}
}

func (s *State) consume(id int) {
	t := &s.Tribes[id]
	demand := FoodDemand(t.Population, t.Soldiers)
	paid := min(t.Food, demand)
	t.Food -= paid
	s.emitOwn(id, fmt.Sprintf("本月粮耗 %d，实际支付 %d。", demand, paid))
	if paid == demand {
		t.HungerMonths = 0
		return
	}
	t.HungerMonths++
	t.GrowthProgress = 0
	if t.HungerMonths <= HungerGraceMonths {
		s.emitOwn(id, fmt.Sprintf("饥荒宽限：缺粮 %d，连续缺粮 %d 月，本月不减员。", demand-paid, t.HungerMonths))
		return
	}
	loss := min(t.Population, demand-paid)
	oldSoldiers := t.Soldiers
	oldReserve := t.ReserveSoldiers()
	t.Population -= loss
	t.Soldiers = min(t.Soldiers, t.Population)
	reserveLoss := min(oldReserve, oldSoldiers-t.Soldiers)
	s.recordArmyChange(id, ReserveSlot, Army{People: oldReserve, Position: t.Capital}, oldReserve-reserveLoss, FamineLoss)
	s.reconcileArmies(id)
	s.emitOwn(id, fmt.Sprintf("饥荒：连续缺粮 %d 月，缺粮 %d，人口减少 %d，士兵减少 %d。", t.HungerMonths, demand-paid, loss, oldSoldiers-t.Soldiers))
}

func (s *State) clearExtinct(id int) {
	t := &s.Tribes[id]
	s.emitAt(id, t.Capital, fmt.Sprintf("%s灭亡，领土与项目已清理。", t.Name))
	t.Food, t.Wood, t.Soldiers, t.AP = 0, 0, 0, 0
	t.GrowthProgress, t.HungerMonths = 0, 0
	t.ForestPriority = 0
	t.Armies = [MaxArmies]Army{}
	for y := range s.Tiles {
		for x := range s.Tiles[y] {
			tile := &s.Tiles[y][x]
			if tile.Owner == id {
				tile.Owner = Unowned
				tile.Farm = FarmNone
			}
		}
	}
}

func (s *State) runMonthlyAI(mode Mode) error {
	first := 1
	if mode == Simulation {
		first = 0
	}
	for id := first; id < TribeCount; id++ {
		if s.Tribes[id].Alive() {
			if err := s.prepareDefense(id); err != nil {
				return err
			}
			if err := s.runAI(id); err != nil {
				return err
			}
		}
	}
	return s.planMilitaryAI(first)
}

func (s *State) settlePopulation() []int {
	var newlyExtinct []int
	for id := range s.Tribes {
		if s.Tribes[id].Alive() {
			s.consume(id)
			if !s.Tribes[id].Alive() {
				newlyExtinct = append(newlyExtinct, id)
			}
		}
	}
	for id := range s.Tribes {
		if s.Tribes[id].Alive() {
			s.grow(id)
		}
	}
	return newlyExtinct
}

func (s *State) finishMonth(newlyExtinct []int) {
	for _, id := range newlyExtinct {
		s.clearExtinct(id)
	}
	for id := range s.Tribes {
		if s.Tribes[id].Alive() {
			s.Tribes[id].AP = MonthlyAP
		}
		t := &s.Tribes[id]
		t.LastChange = Stock{t.Food - t.monthStart.Food, t.Wood - t.monthStart.Wood}
		t.monthStart = Stock{t.Food, t.Wood}
	}
}
