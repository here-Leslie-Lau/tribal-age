package tui

import (
	"image/color"
	"math"
	"regexp"
	"strconv"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

func TestMapPaletteHasReadableNominalContrast(t *testing.T) {
	tokens := append([]mapTone{fogTone, unclaimedTone}, territoryTones[:]...)
	tokens = append(tokens, terrainTones[:]...)
	for _, token := range tokens {
		encoded := (mapStyle{profile: colorprofile.TrueColor}).paint("f", token, false)
		rgb := regexp.MustCompile(`38;2;(\d+);(\d+);(\d+);48;2;(\d+);(\d+);(\d+)`).FindStringSubmatch(encoded)
		if len(rgb) != 7 {
			t.Fatal("palette did not encode a foreground and full-cell background")
		}
		fg := rgbColor(rgb[1:4])
		bg := rgbColor(rgb[4:7])
		if contrast(fg, bg) < 4.5 {
			t.Fatalf("RGB map text lacks contrast: %q", encoded)
		}
		encoded = (mapStyle{profile: colorprofile.ANSI}).paint("f", token, false)
		basic := regexp.MustCompile(`\x1b\[(\d+);(\d+)m`).FindStringSubmatch(encoded)
		if len(basic) != 3 {
			t.Fatal("basic terminal received unexpected style")
		}
		foreground, _ := strconv.Atoi(basic[1])
		background, _ := strconv.Atoi(basic[2])
		// ANSI bright codes are 60 above their normal equivalents.
		foreground -= 30
		background -= 40
		if foreground >= 60 {
			foreground -= 52
		}
		if background >= 60 {
			background -= 52
		}
		if contrast(ansi.BasicColor(foreground), ansi.BasicColor(background)) < 4.5 {
			t.Fatalf("basic palette lacks nominal contrast: %q", encoded)
		}
	}
}

func rgbColor(parts []string) color.Color {
	var values [3]uint8
	for i, part := range parts {
		value, _ := strconv.Atoi(part)
		values[i] = uint8(value)
	}
	return color.RGBA{R: values[0], G: values[1], B: values[2], A: 255}
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	linear := func(v uint32) float64 {
		f := float64(v) / 65535
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

func contrast(a, b color.Color) float64 {
	x, y := luminance(a), luminance(b)
	return (max(x, y) + 0.05) / (min(x, y) + 0.05)
}
