package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestSaveRoundTripAndContinuedReplay(t *testing.T) {
	for _, military := range []bool{false, true} {
		s := New(42)
		if military {
			s = warScenario(42)
		}
		for i := 0; i < 24; i++ {
			if military {
				commandWar(t, s)
			}
			if err := s.EndMonth(Simulation); err != nil {
				t.Fatal(err)
			}
		}
		// A policy changed since the last month must survive a codec round-trip.
		if err := s.SetForestPriority(0, 2); err != nil {
			t.Fatal(err)
		}
		before := *s
		before.Events = append([]Event(nil), s.Events...)
		data, err := s.MarshalSave()
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := UnmarshalSave(data)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(s, loaded) || !reflect.DeepEqual(before, *s) {
			t.Fatal("snapshot lost private state or mutated live game")
		}
		for i := 0; i < 60 && !s.AllExtinct(); i++ {
			if military {
				commandWar(t, s)
				commandWar(t, loaded)
			}
			if err := s.EndMonth(Simulation); err != nil {
				t.Fatal(err)
			}
			if err := loaded.EndMonth(Simulation); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s, loaded) {
				t.Fatal("loading changed later resources, growth, fog, events or military outcomes")
			}
		}
		for i := 0; i < 32; i++ {
			if s.rng.Int63() != loaded.rng.Int63() {
				t.Fatal("random position was not restored")
			}
		}
	}
}
func TestSavePreservesMonthStartAfterPaidAction(t *testing.T) {
	s := New(7)
	if err := s.Apply(0, Action{Kind: BuildFarm, Target: s.Tribes[0].Capital}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	p := s.Tribes[0].Capital
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: March, Target: Point{p.X, p.Y - 1}}); err != nil {
		t.Fatal(err)
	}
	data, err := s.MarshalSave()
	if err != nil {
		t.Fatal(err)
	}
	copy, err := UnmarshalSave(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, copy) {
		t.Fatal("paid AP or pending army order was lost")
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if err := copy.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].LastChange != copy.Tribes[0].LastChange || s.Tribes[0].LastChange.Wood != -FarmWood {
		t.Fatal("month-start stock was replaced by current stock")
	}
}
func TestSaveRejectsMalformedAndInvalidData(t *testing.T) {
	good, _ := New(42).MarshalSave()
	for _, data := range [][]byte{nil, []byte("null"), []byte("{}"), good[:len(good)/2], append(append([]byte(nil), good...), []byte(" {}")...), bytes.Replace(good, []byte(`"version": 1`), []byte(`"version": 999`), 1)} {
		if _, err := UnmarshalSave(data); err == nil {
			t.Fatalf("accepted malformed archive %.30s", data)
		}
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(good, &obj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"tiles", "tribes"} {
		bad := bytes.Replace(good, obj[key], []byte(`[]`), 1)
		if _, err := UnmarshalSave(bad); err == nil {
			t.Fatalf("accepted incomplete %s", key)
		}
	}
	s := New(42)
	s.Tribes[0].Food = -1
	if _, err := s.MarshalSave(); err == nil {
		t.Fatal("saved invalid resources")
	}
}
func FuzzSaveDecodeDoesNotPanic(f *testing.F) {
	data, _ := New(42).MarshalSave()
	f.Add(data)
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte("null"))
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := UnmarshalSave(data)
		if err == nil {
			if s == nil {
				t.Fatal("nil successful load")
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
			if _, err := s.MarshalSave(); err != nil {
				t.Fatal(err)
			}
		}
	})
}
