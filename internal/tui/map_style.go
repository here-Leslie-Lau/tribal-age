package tui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

type mapStyle struct {
	terrain bool
	profile colorprofile.Profile
}

func (s mapStyle) colored() bool { return s.profile >= colorprofile.ANSI }

func (s mapStyle) name() string {
	name := "领土视图"
	if s.terrain {
		name = "地形视图"
	}
	if !s.colored() {
		name += "(无色)"
	}
	return name
}

func (s mapStyle) tone(c *game.Cell) mapTone {
	if c == nil {
		return fogTone
	}
	if s.terrain {
		return terrainTones[c.Tile.Terrain]
	}
	if c.Tile.Owner == game.Unowned {
		return unclaimedTone
	}
	return territoryTones[c.Tile.Owner]
}

func (s mapStyle) paint(text string, t mapTone, selected bool) string {
	if !s.colored() {
		return text
	}
	var fg, bg color.Color = color.RGBA{R: 242, G: 244, B: 248, A: 255}, t.background
	if s.profile == colorprofile.ANSI {
		fg, bg = t.basicFG, t.basicBG
	}
	style := ansi.NewStyle().ForegroundColor(s.profile.Convert(fg)).BackgroundColor(s.profile.Convert(bg))
	if selected {
		style = style.Bold().Reverse(true)
	}
	return style.Styled(text)
}

func (s mapStyle) cell(c *game.Cell, self game.Tribe, selected bool) string {
	text := tileText(c, self)
	if s.colored() {
		if text[1] == '@' {
			text = "@"
		} else {
			text = text[:1]
		}
	}
	return s.markedCell(text, c, selected)
}
func (s mapStyle) markedCell(text string, c *game.Cell, selected bool) string {
	if selected {
		text = "[" + text + "]"
	} else {
		text = " " + text
	}
	// Paint padding too: adjacent cells of the same territory must join.
	return s.paint(fitRow(text, mapCellWidth), s.tone(c), selected)
}

func (s mapStyle) legend() string {
	if !s.colored() {
		return "[]光标 @首都 P我方 2/3他部落 .草原 f森林 ^山 =河 ~海 ?未知"
	}
	if s.terrain {
		var parts []string
		for i, t := range terrainTones {
			terrain := game.Terrain(i)
			parts = append(parts, s.paint(fmt.Sprintf(" %c%s ", terrain.Symbol(), terrain), t, false))
		}
		return strings.Join(parts, " ") + " ?未知 @首都 []光标"
	}
	return s.paint(" 我 ", territoryTones[0], false) + " " + s.paint(" 2 ", territoryTones[1], false) + " " +
		s.paint(" 3 ", territoryTones[2], false) + " " + s.paint(" 无主 ", unclaimedTone, false) + " | .草原 f森林 ^山 =河 ~海 ?未知 @首都 []光标"
}
