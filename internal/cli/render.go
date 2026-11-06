package cli

import (
	"fmt"
	"strings"

	"tribal-age/internal/game"
)

func renderMap(v game.View) string {
	var cells [game.Height][game.Width]*game.Cell
	for i := range v.Cells {
		c := &v.Cells[i]
		cells[c.Point.Y][c.Point.X] = c
	}
	var b strings.Builder
	b.WriteString("地图（每格：地形 + 归属）\n   ")
	for x := 0; x < game.Width; x++ {
		fmt.Fprintf(&b, "%02d ", x)
	}
	b.WriteByte('\n')
	for y := 0; y < game.Height; y++ {
		fmt.Fprintf(&b, "%02d ", y)
		for x := 0; x < game.Width; x++ {
			c := cells[y][x]
			if c == nil {
				b.WriteString("?? ")
				continue
			}
			owner := byte('-')
			if c.Tile.Owner == v.Self.ID {
				owner = 'P'
				if c.Point == v.Self.Capital && v.Self.Alive() {
					owner = '@'
				}
			} else if c.Tile.Owner != game.Unowned {
				owner = byte('1' + c.Tile.Owner)
			}
			fmt.Fprintf(&b, "%c%c ", c.Tile.Terrain.Symbol(), owner)
		}
		b.WriteByte('\n')
	}
	b.WriteString("图例：. 草原  f 森林  ^ 山地  = 河流  ~ 海洋  ?? 未探索\n")
	b.WriteString("归属：@ 我方首都  P 我方领土  2/3 其他部落  - 无人\n")
	return b.String()
}

func renderStatus(v game.View) string {
	t := v.Self
	f := game.ForecastFor(v)
	g, p := f.Growth, f.Production
	growthCondition := g.Reason
	if growthCondition == "" {
		growthCondition = "预计可积累增长进度"
	} else {
		growthCondition = "预计增长暂停：" + growthCondition
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s | 行动点：%d/%d\n", v.Calendar, t.AP, game.MonthlyAP)
	fmt.Fprintf(&b, "%s | 首都：(%d,%d) | 人口：%d/%d（承载力） | 士兵：%d | 平民：%d\n", t.Name, t.Capital.X, t.Capital.Y, t.Population, g.Capacity, t.Soldiers, t.Population-t.Soldiers)
	fmt.Fprintf(&b, "粮食：%d（上月净变化 %+d） 木材：%d（上月净变化 %+d）\n", t.Food, t.LastChange.Food, t.Wood, t.LastChange.Wood)
	territory, farms, building := counts(v)
	fmt.Fprintf(&b, "领土：%d 格 | 完工农田：%d | 在建农田：%d\n", territory, farms, building)
	fmt.Fprintf(&b, "增长进度：%d/%d | 安全存粮：%d | 本月预计+%d/%d，新增%d人 | %s\n", t.GrowthProgress, game.GrowthScale, g.Reserve, g.Rate, game.GrowthScale, g.Added, growthCondition)
	fmt.Fprintf(&b, "连续缺粮：%d 月（从第 %d 月起减员） | 每月口粮：%d\n", t.HungerMonths, game.HungerGraceMonths+1, game.FoodDemand(t.Population, t.Soldiers))
	b.WriteString(renderProduction(t, f))
	fmt.Fprintf(&b, "人手可经营 %d 格 | 月末预计安排：农田 %d 格，森林 %d 格\n", p.Slots, len(p.Farms), len(p.Forests))
	for _, point := range p.Unstaffed {
		fmt.Fprintf(&b, "  (%d,%d) 人手不足，预计停产\n", point.X, point.Y)
	}
	for _, c := range v.Cells {
		if c.Tile.Owner != t.ID {
			continue
		}
		fmt.Fprintf(&b, "  (%d,%d) %s / 我方领土 / 农田：%s\n", c.Point.X, c.Point.Y, c.Tile.Terrain, farmLabel(c.Tile.Farm))
	}
	return b.String()
}

func renderProduction(t game.Tribe, f game.Forecast) string {
	var b strings.Builder
	if t.ForestPriority == 0 {
		b.WriteString("生产安排：自动（先种田，再采木）\n")
	} else {
		fmt.Fprintf(&b, "优先采木：%d 格森林（剩余人手自动安排）\n", t.ForestPriority)
		if len(f.Production.Forests) < t.ForestPriority {
			fmt.Fprintf(&b, "采木目标暂未满足：实际经营 %d 格，森林或人手不足。\n", len(f.Production.Forests))
		}
	}
	fmt.Fprintf(&b, "本月预计产粮 %d，产木 %d；口粮 %d，粮食收支 %+d，预计结余 %d。\n", f.Production.Food, f.Production.Wood, f.Demand, f.NetFood, f.FoodAfter)
	if f.Shortage > 0 {
		fmt.Fprintf(&b, "缺粮预警：预计缺 %d 粮，将中断增长并计入连续缺粮；仍需 end 才结算。\n", f.Shortage)
	}
	return b.String()
}

func counts(v game.View) (territory, farms, building int) {
	for _, c := range v.Cells {
		if c.Tile.Owner != v.Self.ID {
			continue
		}
		territory++
		if c.Tile.Farm == game.FarmReady {
			farms++
		} else if c.Tile.Farm != game.FarmNone {
			building++
		}
	}
	return
}

func farmLabel(f game.FarmStage) string {
	if f == game.FarmNone {
		return "无"
	}
	if f == game.FarmReady {
		return "已完工"
	}
	return fmt.Sprintf("建造中 %d/%d", f.Progress(), game.FarmMonths)
}

func renderSummary(s *game.State, requested, actual int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "结算摘要（无胜利判定）\n种子：%d | 请求 %d 个月 | 实际推进 %d 个月\n%s\n", s.Seed, requested, actual, s.Calendar)
	for id, t := range s.Tribes {
		state := t.Status()
		v, _ := s.View(id) // id comes from the fixed tribe array, so it is always valid.
		territory, farms, building := counts(v)
		fmt.Fprintf(&b, "%s [%s] 人口 %d/%d 士兵 %d 粮食 %d 木材 %d 领土 %d 农田 %d 在建 %d 增长 %d/%d 连续缺粮 %d 月\n", t.Name, state, t.Population, game.CarryingCapacity(v), t.Soldiers, t.Food, t.Wood, territory, farms, building, t.GrowthProgress, game.GrowthScale, t.HungerMonths)
	}
	return b.String()
}
