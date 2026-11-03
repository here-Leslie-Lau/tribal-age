package tui

import (
	"fmt"
	"strconv"
	"strings"

	"tribal-age/internal/game"
)

// Count the whole explored view, not the viewport or the underlying world.
func knownTerritories(v game.View) [game.TribeCount]int {
	var counts [game.TribeCount]int
	for _, c := range v.Cells {
		if c.Tile.Owner != game.Unowned {
			counts[c.Tile.Owner]++
		}
	}
	return counts
}

func (s mapStyle) factionLines(v game.View, expanded bool) []string {
	counts := knownTerritories(v)
	var lines []string
	if !expanded {
		parts := []string{fmt.Sprintf("已知领土 我%d", counts[v.Self.ID])}
		for id, n := range counts {
			if id == v.Self.ID {
				continue
			}
			count := "?"
			if n > 0 {
				count = strconv.Itoa(n)
			}
			parts = append(parts, fmt.Sprintf("%d:%s", id+1, count))
		}
		return []string{strings.Join(parts, " ")}
	}
	for id, n := range counts {
		label := fmt.Sprintf("部落%d 已发现%d格", id+1, n)
		if n == 0 {
			label = fmt.Sprintf("部落%d 暂无已知领地", id+1)
		}
		if id == v.Self.ID {
			label = fmt.Sprintf("我方%d格", n)
		}
		lines = append(lines, s.paint("  ", territoryTones[id], false)+" "+label)
	}
	return lines
}
