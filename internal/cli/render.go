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
	var b strings.Builder
	fmt.Fprintf(&b, "%s | 行动点：%d/%d\n", v.Calendar, t.AP, game.MonthlyAP)
	fmt.Fprintf(&b, "%s | 首都：(%d,%d) | 人口：%d | 士兵：%d | 平民：%d\n", t.Name, t.Capital.X, t.Capital.Y, t.Population, t.Soldiers, t.Population-t.Soldiers)
	fmt.Fprintf(&b, "粮食：%d（上月净变化 %+d） 木材：%d（上月净变化 %+d）\n", t.Food, t.LastChange.Food, t.Wood, t.LastChange.Wood)
	territory, farms, building := counts(v)
	fmt.Fprintf(&b, "领土：%d 格 | 完工农田：%d | 在建农田：%d\n", territory, farms, building)
	for _, c := range v.Cells {
		if c.Tile.Owner != t.ID {
			continue
		}
		fmt.Fprintf(&b, "  (%d,%d) %s / 我方领土 / 农田：%s\n", c.Point.X, c.Point.Y, c.Tile.Terrain, farmLabel(c.Tile.Farm))
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
		state := "存活"
		if !t.Alive() {
			state = "灭亡"
		}
		v, _ := s.View(id) // id comes from the fixed tribe array, so it is always valid.
		territory, farms, building := counts(v)
		fmt.Fprintf(&b, "%s [%s] 人口 %d 士兵 %d 粮食 %d 木材 %d 领土 %d 农田 %d 在建 %d\n", t.Name, state, t.Population, t.Soldiers, t.Food, t.Wood, territory, farms, building)
	}
	return b.String()
}
