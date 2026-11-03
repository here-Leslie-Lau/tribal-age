package tui

import (
	"fmt"
	"strings"

	"tribal-age/internal/game"
)

func mapLines(v game.View, cursor game.Point, l layout) []string {
	var known [game.Height][game.Width]*game.Cell
	for i := range v.Cells {
		c := &v.Cells[i]
		known[c.Point.Y][c.Point.X] = c
	}
	lines := []string{fmt.Sprintf("地图 x:%d..%d y:%d..%d", l.left, l.left+l.cols-1, l.top, l.top+l.rows-1)}
	var header strings.Builder
	header.WriteString("    ")
	for x := l.left; x < l.left+l.cols; x++ {
		fmt.Fprintf(&header, "%*d", mapCellWidth, x)
	}
	lines = append(lines, header.String())
	for y := l.top; y < l.top+l.rows; y++ {
		var row strings.Builder
		fmt.Fprintf(&row, "%02d |", y)
		for x := l.left; x < l.left+l.cols; x++ {
			marker := byte(' ')
			if cursor == (game.Point{X: x, Y: y}) {
				marker = '>'
			}
			fmt.Fprintf(&row, "%-*s", mapCellWidth, fmt.Sprintf("%c%s", marker, tileText(known[y][x], v.Self)))
		}
		lines = append(lines, row.String())
	}
	return lines
}

func tileText(c *game.Cell, self game.Tribe) string {
	if c == nil {
		return "??"
	}
	owner := byte('-')
	if c.Tile.Owner == self.ID {
		owner = 'P'
		if c.Point == self.Capital && self.Alive() {
			owner = '@'
		}
	} else if c.Tile.Owner != game.Unowned {
		owner = byte('1' + c.Tile.Owner)
	}
	return fmt.Sprintf("%c%c", c.Tile.Terrain.Symbol(), owner)
}

func selectedInfo(v game.View, cursor game.Point) string {
	prefix := fmt.Sprintf("选中 (%d,%d) ", cursor.X, cursor.Y)
	for _, c := range v.Cells {
		if c.Point != cursor {
			continue
		}
		owner := "无人占领"
		if c.Tile.Owner == v.Self.ID {
			owner = "自有领土"
		} else if c.Tile.Owner != game.Unowned {
			owner = fmt.Sprintf("部落%d领土", c.Tile.Owner+1)
		}
		return prefix + fmt.Sprintf("%s / %s / %s", c.Tile.Terrain, owner, farmInfo(c.Tile.Farm))
	}
	return prefix + "未探索（地形和归属未知）"
}

func farmInfo(f game.FarmStage) string {
	switch f {
	case game.FarmNone:
		return "无农田"
	case game.FarmReady:
		return "农田已完工"
	}
	return fmt.Sprintf("农田建造中 %d/%d", f.Progress(), game.FarmMonths)
}

func actionHints(v game.View, cursor game.Point) string {
	var hints []string
	for _, binding := range tileBindings {
		action := game.Action{Kind: binding.kind, Target: cursor}
		if !isLegal(v, action) {
			continue
		}
		cost := fmt.Sprintf("%d行动", game.ActionAP)
		if binding.wood > 0 {
			cost = fmt.Sprintf("%d木/", binding.wood) + cost
		}
		hints = append(hints, strings.ToUpper(binding.key)+binding.label+" "+cost)
	}
	if len(hints) == 0 {
		return "本格暂无可用操作；可移动光标或确认结束月份。"
	}
	return "可用：" + strings.Join(hints, "  ")
}

func isLegal(v game.View, action game.Action) bool {
	for _, legal := range v.Legal {
		if legal == action {
			return true
		}
	}
	return false
}
