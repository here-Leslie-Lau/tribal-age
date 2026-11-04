package game

import (
	"reflect"
	"testing"
)

func TestLongSimulationAndActionReplay(t *testing.T) {
	seeds := []int64{-9223372036854775808, -100, -1, 0, 1, 2, 7, 42, 99, 2026, 12345, 9223372036854775807}
	for _, seed := range seeds {
		stepped, automatic := New(seed), New(seed)
		for month := 0; month < 120; month++ {
			beforePopulation := automatic.Tribes
			// Exercise the public action API individually, checking every transition.
			for id := range stepped.Tribes {
				for action := 0; action < MonthlyAP; action++ {
					v, err := stepped.View(id)
					if err != nil {
						t.Fatal(err)
					}
					a, ok := ChooseAction(v)
					if !ok {
						break
					}
					if err := stepped.Apply(id, a); err != nil {
						t.Fatalf("seed %d month %d: illegal AI action: %v", seed, month, err)
					}
					if err := stepped.Validate(); err != nil {
						t.Fatalf("seed %d month %d after action: %v", seed, month, err)
					}
				}
			}
			// Used AP cannot be spent again. Automatic scheduling must reach exactly the same state/events.
			if err := stepped.EndMonth(Simulation); err != nil {
				t.Fatalf("seed %d month %d: %v", seed, month, err)
			}
			if err := automatic.EndMonth(Simulation); err != nil {
				t.Fatal(err)
			}
			if err := stepped.Validate(); err != nil {
				t.Fatal(err)
			}
			if err := automatic.Validate(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(stepped, automatic) {
				t.Fatalf("seed %d month %d: action replay differs from automatic simulation", seed, month)
			}
			for id := range stepped.Tribes {
				if !reflect.DeepEqual(stepped.VisibleEvents(id, 0), automatic.VisibleEvents(id, 0)) {
					t.Fatal("event output not reproducible")
				}
				v, err := automatic.View(id)
				if err != nil {
					t.Fatal(err)
				}
				before := beforePopulation[id]
				maxBirths := (before.GrowthProgress + growthRate(before.Population, CarryingCapacity(v))) / GrowthScale
				if v.Self.Population > before.Population+maxBirths {
					t.Fatal("population exceeded this month's nonlinear growth bound")
				}
				if v.Self.Alive() && v.Self.Population > CarryingCapacity(v) {
					t.Fatal("new population exceeded territory capacity")
				}
				production := ProductionFor(v, v.Calendar.Season())
				if len(production.Farms)+len(production.Forests) > v.Self.WorkSlots() {
					t.Fatal("production assignments exceeded available workers")
				}
			}
		}
		if stepped.Calendar.Elapsed != 120 || stepped.Calendar.Year() != 11 || stepped.Calendar.Month() != 3 {
			t.Fatalf("seed %d did not actually complete 120 months", seed)
		}
	}
}

func FuzzActionInvariants(f *testing.F) {
	f.Add([]byte{0, 2, 2, 0, 1, 2, 2, 0, 4, 0, 0, 0})
	f.Add([]byte{2, 2, 3, 0, 3, 0, 0, 0, 4, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255})
	p := New(42).Tribes[0].Capital
	f.Add([]byte{byte(BuildFarm), byte(p.X), byte(p.Y), 1, byte(Explore), byte(p.X), byte(p.Y - 1), 1, 4, byte(Simulation), 0, 0})
	var growing []byte
	for month := 0; month < 12; month++ {
		growing = append(growing, 4, byte(Simulation), 0, 0)
	}
	f.Add(growing)
	f.Add([]byte{6, 4, 0, 1, 4, byte(Simulation), 0, 0, 6, 0, 0, 1, 4, byte(Simulation), 0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		s := New(42)
		for i := 0; i+3 < min(len(data), 512); i += 4 {
			before := *s
			before.Events = append([]Event(nil), s.Events...)
			var err error
			kind := data[i] % 7
			switch kind {
			case 4:
				err = s.EndMonth(Mode(data[i+1] % 3))
			case 6:
				priority := int(int8(data[i+1]))
				if data[i+2]&1 != 0 {
					priority = int(data[i+1]) * 2
				}
				err = s.SetForestPriority(int(data[i+3]%5)-1, priority)
			default:
				a := Action{Kind: ActionKind(kind), Target: Point{int(int8(data[i+1])), int(int8(data[i+2]))}}
				err = s.Apply(int(data[i+3]%5)-1, a)
			}
			if err != nil && !reflect.DeepEqual(before, *s) {
				t.Fatal("rejection changed state")
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		}
		if s.rng.Int63() != New(42).rng.Int63() {
			t.Fatal("actions or settlement advanced the map RNG")
		}
	})
}
