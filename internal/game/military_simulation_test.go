package game

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func TestClashesAreIndependentOfArmyEnumeration(t *testing.T) {
	rng := rand.New(rand.NewSource(91))
	for trial := 0; trial < 400; trial++ {
		used := map[Point]bool{}
		var f []fighter
		for len(f) < 9 {
			p := Point{2 + rng.Intn(12), 2 + rng.Intn(8)}
			if used[p] {
				continue
			}
			used[p] = true
			to := p
			if rng.Intn(2) == 0 {
				to = p.Neighbors()[rng.Intn(4)]
			}
			f = append(f, fighter{Owner: len(f) % 3, People: 1 + rng.Intn(40), From: p, To: to, Supply: SupplyLevel(rng.Intn(3)), Forest: rng.Intn(2) == 0})
		}
		a := resolveClashes(f)
		permutation := rng.Perm(len(f))
		shuffled := make([]fighter, len(f))
		for i, j := range permutation {
			shuffled[i] = f[j]
		}
		b := resolveClashes(shuffled)
		for i, j := range permutation {
			if a.Losses[j] != b.Losses[i] || a.Positions[j] != b.Positions[i] || a.Engaged[j] != b.Engaged[i] {
				t.Fatal("army enumeration affected simultaneous resolution")
			}
		}
		for i := range f {
			if a.Losses[i] < 0 || a.Losses[i] > f[i].People {
				t.Fatal("invalid casualties")
			}
			for j := 0; j < i; j++ {
				if f[i].Owner != f[j].Owner && a.Positions[i] == a.Positions[j] {
					t.Fatal("blocked departure allowed hostile co-location")
				}
			}
		}
	}
}
func TestOccupationInterruptedByBattleAndPrivateCombatEvents(t *testing.T) {
	s := invaderOnCapital()
	p := s.Tribes[1].Capital
	back := Point{p.X, p.Y + 1}
	ownTile(s, 1, back, Grassland, FarmNone)
	s.Tribes[1].Armies[0] = Army{People: 5, Position: back}
	s.revealArmies(1)
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: Occupy}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetArmyOrder(1, 0, ArmyOrder{Kind: Attack, Target: p}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tiles[p.Y][p.X].Owner != 1 {
		t.Fatal("a contested occupation completed")
	}
	s = scenarioState()
	for y := range s.Tribes[0].Explored {
		for x := range s.Tribes[0].Explored[y] {
			s.Tribes[0].Explored[y][x] = true
		}
	}
	s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{12, 3}}
	s.revealArmies(1)
	s.Tribes[2].Armies[0] = Army{People: 5, Position: Point{13, 3}}
	s.revealArmies(2)
	if err := s.SetArmyOrder(1, 0, ArmyOrder{Kind: Attack, Target: Point{13, 3}}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(s.VisibleEvents(0, 0), "\n"), "军队") {
		t.Fatal("remembered terrain leaked an unseen battle")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
func warScenario(seed int64) *State {
	s := New(seed)
	for id := range s.Tribes {
		t := &s.Tribes[id]
		t.Population, t.Soldiers, t.Food, t.Wood = 100, 30, 100000, 1000
		t.Armies[0] = Army{People: 30, Position: t.Capital}
		s.revealArmies(id)
		for y := range t.Explored {
			for x := range t.Explored[y] {
				t.Explored[y][x] = true
			}
		}
	}
	return s
}

// A deterministic test commander initiates war; production AI is intentionally
// defensive in this version. This is a war-system replay, not a balance claim.
func commandWar(tb *testing.T, s *State) {
	tb.Helper()
	issue := func(slot int, order ArmyOrder) {
		tb.Helper()
		if err := s.SetArmyOrder(0, slot, order); err != nil {
			tb.Fatal(err)
		}
	}
	if !s.Tribes[0].Alive() {
		return
	}
	t := s.Tribes[0]
	for slot, a := range t.Armies {
		if a.People == 0 {
			continue
		}
		v, _ := s.View(0)
		if enemy, ok := nearestArmyEnemy(v, a.Position); ok && manhattan(a.Position, enemy.Position) == 1 {
			issue(slot, ArmyOrder{Kind: Attack, Target: enemy.Position})
			continue
		}
		tile := s.Tiles[a.Position.Y][a.Position.X]
		if tile.Owner != 0 && tile.Owner != Unowned {
			issue(slot, ArmyOrder{Kind: Occupy})
			continue
		}
		goal := Point{}
		found := false
		for _, c := range v.Cells {
			if c.Tile.Owner > 0 && (!found || manhattan(a.Position, c.Point) < manhattan(a.Position, goal)) {
				goal = c.Point
				found = true
			}
		}
		if found {
			if step, ok := armyPathStep(v, a.Position, goal); ok {
				issue(slot, ArmyOrder{Kind: March, Target: step})
			}
		}
	}
}
func TestMilitaryLongReplayAndPopulationAccounting(t *testing.T) {
	for _, seed := range []int64{0, 7, 42, 99} {
		a, b := warScenario(seed), warScenario(seed)
		for month := 0; month < 120 && !a.AllExtinct(); month++ {
			commandWar(t, a)
			commandWar(t, b)
			if err := a.EndMonth(Simulation); err != nil {
				t.Fatal(err)
			}
			if err := b.EndMonth(Simulation); err != nil {
				t.Fatal(err)
			}
			if err := a.Validate(); err != nil {
				t.Fatalf("seed %d month %d: %v", seed, month, err)
			}
			if !reflect.DeepEqual(*a, *b) {
				t.Fatal("military replay diverged")
			}
		}
		if a.Calendar.Elapsed != 120 {
			t.Fatalf("seed %d military replay stopped at %d months", seed, a.Calendar.Elapsed)
		}
		battle := false
		for _, e := range a.Events {
			if strings.Contains(e.Text, "交锋") {
				battle = true
			}
		}
		if !battle {
			t.Fatal("military replay did not actually exercise combat")
		}
	}
}
