package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"tribal-age/internal/game"
)

func TestMapPreservesTerrainOwnershipAndWidth(t *testing.T) {
	s := game.New(42)
	for x, terrain := range []game.Terrain{game.Grassland, game.Forest, game.Mountain, game.River, game.Ocean} {
		s.Tribes[0].Explored[1][x+1] = true
		s.Tiles[1][x+1] = game.Tile{Terrain: terrain, Owner: game.Unowned}
	}
	p := s.Tribes[0].Capital
	s.Tribes[0].Explored[p.Y-1][p.X] = true
	s.Tiles[p.Y-1][p.X].Owner = 0
	s.Tiles[p.Y][p.X].Farm = game.FarmReady
	enemy := s.Tribes[1].Capital
	s.Tribes[0].Explored[enemy.Y][enemy.X] = true
	var out bytes.Buffer
	if err := play(s, strings.NewReader("map\nstatus\nquit\n"), &out); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{".-", "f-", "^-", "=-", "~-", "fP", ".@", ".2", "??", "已完工", "领土：2 格"} {
		if !strings.Contains(out.String(), token) {
			t.Fatalf("missing map/status token %q", token)
		}
	}
	for _, line := range strings.Split(out.String(), "\n") {
		for y := 0; y < game.Height; y++ {
			if strings.HasPrefix(line, fmt.Sprintf("%02d ", y)) && len(line) > 80 {
				t.Fatalf("map row exceeds 80 columns: %d", len(line))
			}
		}
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestShortWriteIsReported(t *testing.T) {
	if err := Run([]string{"help"}, strings.NewReader(""), shortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short output silently discarded: %v", err)
	}
}
