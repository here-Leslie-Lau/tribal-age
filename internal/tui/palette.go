package tui

import (
	"image/color"

	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

// RGB colors have fixed, contrasting 16-color equivalents for basic terminals.
// These value-only palettes are shared by map cells, legends and faction badges.
type mapTone struct {
	background       color.RGBA
	basicBG, basicFG ansi.BasicColor
}

func tone(r, g, b uint8, bg, fg ansi.BasicColor) mapTone {
	return mapTone{color.RGBA{R: r, G: g, B: b, A: 255}, bg, fg}
}

var territoryTones = [game.TribeCount]mapTone{
	tone(29, 77, 114, ansi.Blue, ansi.BrightWhite),
	tone(146, 77, 29, ansi.Yellow, ansi.Black),
	tone(101, 58, 123, ansi.Magenta, ansi.BrightWhite),
}

var terrainTones = [...]mapTone{
	game.Grassland: tone(45, 73, 48, ansi.Green, ansi.BrightWhite),
	game.Forest:    tone(26, 64, 58, ansi.Cyan, ansi.BrightWhite),
	game.Mountain:  tone(82, 75, 68, ansi.BrightBlack, ansi.Black),
	game.River:     tone(28, 68, 99, ansi.Blue, ansi.BrightWhite),
	game.Ocean:     tone(18, 43, 65, ansi.Black, ansi.BrightWhite),
}

var (
	unclaimedTone = tone(49, 58, 67, ansi.BrightBlack, ansi.Black)
	fogTone       = tone(20, 27, 34, ansi.Black, ansi.White)
)
