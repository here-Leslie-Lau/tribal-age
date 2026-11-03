package tui

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

// Read effective reverse-video state before the cell's first visible character.
// Cut keeps surrounding SGR; RGB channel values must not be read as attributes.
func reversedCell(cell string) bool {
	prefix := regexp.MustCompile(`^(?:\x1b\[[0-9;]*m)*`).FindString(cell)
	prefix = regexp.MustCompile(`(?:38|48);(?:2;[0-9]+;[0-9]+;[0-9]+|5;[0-9]+)`).ReplaceAllString(prefix, "")
	reversed := false
	for _, sgr := range regexp.MustCompile(`\x1b\[([0-9;]*)m`).FindAllStringSubmatch(prefix, -1) {
		for _, code := range strings.Split(sgr[1], ";") {
			switch code {
			case "", "0", "27":
				reversed = false
			case "7":
				reversed = true
			}
		}
	}
	return reversed
}

func TestCursorHighlightIsIndependentOfOwnerAndResetsOnMovement(t *testing.T) {
	for _, profile := range []colorprofile.Profile{colorprofile.ANSI, colorprofile.ANSI256, colorprofile.TrueColor} {
		m := colorModel(t, ownershipState(), profile)
		before := snapshot(m.state)
		capital := m.cursor
		for mode := 0; mode < 2; mode++ {
			m.cursor = capital
			for _, keys := range [][]string{nil, {"down"}, {"right"}, {"right"}, {"up", "left", "up", "left", "up", "left", "up", "left", "up", "left", "up", "left"}} {
				old := m.cursor
				press(m, keys...)
				cell := mapCell(m, m.cursor)
				text := strings.TrimSpace(ansi.Strip(cell))
				if !reversedCell(cell) || !strings.HasPrefix(text, "[") || !strings.HasSuffix(text, "]") || ansi.StringWidth(cell) != 4 {
					t.Fatalf("%v mode %d has no readable framed highlight at %v: %q", profile, mode, m.cursor, cell)
				}
				if old != m.cursor && reversedCell(mapCell(m, old)) {
					t.Fatal("cursor highlight remained on previous cell")
				}
			}
			press(m, "v")
		}
		if !reflect.DeepEqual(before, *m.state) {
			t.Fatal("highlight or movement changed the game")
		}
	}
}
