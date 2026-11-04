package game

import (
	"reflect"
	"strings"
	"testing"
)

func TestGrowthForecastMatchesBatchesPauseAndCapacity(t *testing.T) {
	for _, tt := range []struct{ p, k, food, progress, added, remainder, rate int }{
		{100, 120, 100000, 0, 1, 500, 1500},
		{200, 240, 100000, 999, 3, 999, 3000},
		{104, 150, 88, 800, 1, 360, 1560},
		{30, 35, 30, 800, 0, 800, 0},
		{30, 35, 1, 800, 0, 0, 0},
		{30, 30, 100000, 900, 0, 0, 0},
	} {
		s := sizedGrowthState(t, tt.p, tt.k, tt.food, tt.progress)
		v, _ := s.View(0)
		before := *s
		f := ForecastFor(v)
		if f.Growth.Added != tt.added || f.Growth.Progress != tt.remainder || f.Growth.Rate != tt.rate {
			t.Fatalf("wrong growth preview for P=%d: %+v", tt.p, f.Growth)
		}
		if !reflect.DeepEqual(before, *s) || !reflect.DeepEqual(f, ForecastFor(v)) {
			t.Fatal("forecast mutated state or was nondeterministic")
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Population != tt.p+tt.added || s.Tribes[0].GrowthProgress != tt.remainder || s.Tribes[0].Food != f.FoodAfter {
			t.Fatal("growth preview differs from actual monthly settlement")
		}
		if tt.p == 104 && !strings.Contains(strings.Join(s.VisibleEvents(0, 0), "\n"), "不会囤积") {
			t.Fatal("discarded births lacked an explanation")
		}
	}
}

func TestPopulationRecordUnitsAndExtremeStoppedGrowth(t *testing.T) {
	s := sizedGrowthState(t, 30, 35, 100, 999)
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Tribes[0].GrowthProgress = 1000
	if s.Validate() == nil {
		t.Fatal("a whole unissued person was accepted as fractional progress")
	}
	s.Tribes[0].GrowthProgress = 0
	s.Tribes[0].Population = int(^uint(0) >> 1)
	s.Tribes[0].Soldiers = s.Tribes[0].Population
	v, _ := s.View(0)
	g := GrowthFor(v)
	if g.Reserve != int(^uint(0)>>1) || g.Added != 0 || g.Progress != 0 || !strings.Contains(g.Reason, "承载力") {
		t.Fatal("extreme stopped growth overflowed its reserve or capacity guard")
	}
}
