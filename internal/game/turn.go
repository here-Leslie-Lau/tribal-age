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
		return fmt.Errorf("玩家部落已经灭亡")
	}
	first := 1
	if mode == Simulation {
		first = 0
	}
	for id := first; id < TribeCount; id++ {
		if s.Tribes[id].Alive() {
			if err := s.runAI(id); err != nil {
				return err
			}
		}
	}
	s.advanceFarms()
	s.produce()
	var newlyExtinct []int
	for id := range s.Tribes {
		if s.Tribes[id].Alive() {
			s.consume(id)
			if !s.Tribes[id].Alive() {
				newlyExtinct = append(newlyExtinct, id)
			}
		}
	}
	s.Calendar.Advance()
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
	return nil
}

func (s *State) advanceFarms() {
	for y := range s.Tiles {
		for x := range s.Tiles[y] {
			tile := &s.Tiles[y][x]
			if tile.Farm == FarmNone || tile.Farm == FarmReady {
				continue
			}
			tile.Farm++
			message := fmt.Sprintf("农田 (%d,%d) 推进至 %d/%d。", x, y, tile.Farm.Progress(), FarmMonths)
			if tile.Farm == FarmReady {
				message = fmt.Sprintf("农田 (%d,%d) 已完工（%d/%d）。", x, y, FarmMonths, FarmMonths)
			}
			s.emitAt(tile.Owner, Point{x, y}, message)
		}
	}
}

func (s *State) produce() {
	var gains [TribeCount]Stock
	for _, row := range s.Tiles {
		for _, tile := range row {
			if tile.Owner == Unowned {
				continue
			}
			if tile.Farm == FarmReady {
				gains[tile.Owner].Food += FarmYield(s.Calendar.Season())
			}
			if tile.Terrain == Forest {
				gains[tile.Owner].Wood += ForestYield
			}
		}
	}
	for id, gain := range gains {
		if !s.Tribes[id].Alive() {
			continue
		}
		s.Tribes[id].Food += gain.Food
		s.Tribes[id].Wood += gain.Wood
		s.emitOwn(id, fmt.Sprintf("生产入库：粮食 +%d，木材 +%d。", gain.Food, gain.Wood))
	}
}

func (s *State) consume(id int) {
	t := &s.Tribes[id]
	demand := FoodDemand(t.Population, t.Soldiers)
	paid := min(t.Food, demand)
	t.Food -= paid
	s.emitOwn(id, fmt.Sprintf("本月粮耗 %d，实际支付 %d。", demand, paid))
	if paid == demand {
		return
	}
	loss := min(t.Population, demand-paid)
	oldSoldiers := t.Soldiers
	t.Population -= loss
	t.Soldiers = min(t.Soldiers, t.Population)
	s.emitOwn(id, fmt.Sprintf("饥荒：缺粮 %d，人口减少 %d，士兵减少 %d。", demand-paid, loss, oldSoldiers-t.Soldiers))
}

func (s *State) clearExtinct(id int) {
	t := &s.Tribes[id]
	s.emitAt(id, t.Capital, fmt.Sprintf("%s灭亡，领土与项目已清理。", t.Name))
	t.Food, t.Wood, t.Soldiers, t.AP = 0, 0, 0, 0
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
