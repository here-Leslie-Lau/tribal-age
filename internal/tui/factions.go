package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
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

func (s mapStyle) factionLines(v game.View, expanded bool, width int) []string {
	counts := knownTerritories(v)
	if expanded {
		return s.factionChart(v.Self.ID, counts, width)
	}
	return []string{compactFactionSummary(v.Self.ID, counts)}
}

func compactFactionSummary(self int, counts [game.TribeCount]int) string {
	parts := []string{fmt.Sprintf("已知领土 我%d", counts[self])}
	for id, n := range counts {
		if id == self {
			continue
		}
		count := "?"
		if n > 0 {
			count = strconv.Itoa(n)
		}
		parts = append(parts, fmt.Sprintf("%d:%s", id+1, count))
	}
	return strings.Join(parts, " ")
}

func (s mapStyle) factionChart(self int, counts [game.TribeCount]int, width int) []string {
	var labels [game.TribeCount]string
	labelWidth, largest := 0, 0
	for id, n := range counts {
		label := fmt.Sprintf("部落%d %d格", id+1, n)
		if id == self {
			label = fmt.Sprintf("我方%d格", n)
		} else if n == 0 {
			label = fmt.Sprintf("部落%d 未知", id+1)
		}
		labels[id] = label
		labelWidth, largest = max(labelWidth, ansi.StringWidth(label)), max(largest, n)
	}
	barWidth := min(12, max(0, width-labelWidth-4))
	lines := []string{"大陆局势（已知）"}
	for id, n := range counts {
		bar := territoryBar(n, largest, barWidth)
		if id != self && n == 0 {
			bar = "?"
		}
		lines = append(lines, s.paint("  ", territoryTones[id], false)+" "+fitRow(labels[id], labelWidth)+" "+bar)
	}
	return lines
}

// All bars share the largest known count; a nonzero faction stays visible.
func territoryBar(count, largest, width int) string {
	if count == 0 || width == 0 {
		return ""
	}
	length := max(1, (count*width+largest/2)/largest)
	return strings.Repeat("#", length)
}
