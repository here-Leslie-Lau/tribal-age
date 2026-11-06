package game

import (
	"reflect"
	"testing"
)

func TestArmyFormationOrdersAndPrivacy(t *testing.T) {
	s := scenarioState()
	if err := s.SetArmySize(0, 0, 3); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Population != 30 || s.Tribes[0].Soldiers != 5 || s.Tribes[0].ReserveSoldiers() != 2 || s.Tribes[0].AP != 1 {
		t.Fatal("formation manufactured people or charged the wrong cost")
	}
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	before := *s
	before.Events = append([]Event(nil), s.Events...)
	if err := s.SetArmySize(0, 0, 5); err != nil || !reflect.DeepEqual(before, *s) {
		t.Fatal("same size was not idempotent")
	}
	order := ArmyOrder{Kind: March, Target: Point{2, 2}}
	if err := s.SetArmyOrder(0, 0, order); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Armies[0].Position != s.Tribes[0].Capital || s.Tribes[0].AP != 0 {
		t.Fatal("setting an order moved soldiers or charged economic AP")
	}
	before = *s
	before.Events = append([]Event(nil), s.Events...)
	if err := s.SetArmyOrder(0, 0, order); err != nil || !reflect.DeepEqual(before, *s) {
		t.Fatal("same order was not idempotent")
	}
	for _, attempt := range []func() error{
		func() error { return s.SetArmySize(-1, 0, 1) }, func() error { return s.SetArmySize(0, 3, 1) }, func() error { return s.SetArmySize(0, 1, 6) },
		func() error { return s.SetArmyOrder(0, 1, order) }, func() error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: March, Target: Point{9, 9}}) },
	} {
		if attempt() == nil || !reflect.DeepEqual(before, *s) {
			t.Fatal("rejected military operation mutated state")
		}
	}
	s.Tribes[1].Armies[0] = Army{People: 5, Position: s.Tribes[1].Capital, Order: ArmyOrder{Kind: March, Target: Point{10, 2}}}
	s.revealArmies(1)
	s.Tribes[0].Explored[3][10] = true
	v, _ := s.View(0)
	for _, a := range v.Armies {
		if a.Owner == 1 {
			t.Fatal("remembered terrain exposed a distant enemy army")
		}
	}
	v.Self.Armies[0].People = 1
	if s.Tribes[0].Armies[0].People != 5 {
		t.Fatal("view shared military data")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
