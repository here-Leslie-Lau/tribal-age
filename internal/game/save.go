package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
)

const SaveVersion = 1
const MaxSaveBytes = 32 << 20

// Version 1 stores rule state, never a limited player View or a terminal model.
// The RNG is used only by New; replaying that version's generator restores its
// exact post-generation position. Turn-time randomness requires a new format.
type savedGame struct {
	Version  int          `json:"version"`
	Seed     *int64       `json:"seed"`
	Calendar *Calendar    `json:"calendar"`
	Tiles    [][]Tile     `json:"tiles"`
	Tribes   []savedTribe `json:"tribes"`
	Events   []Event      `json:"events"`
}
type savedTribe struct {
	Tribe
	Explored   [][]bool
	Armies     []Army
	MonthStart *Stock `json:"month_start"`
}

func (s *State) MarshalSave() ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("游戏状态为空")
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if err := validateSavedRecords(s); err != nil {
		return nil, err
	}
	record := savedGame{Version: SaveVersion, Seed: &s.Seed, Calendar: &s.Calendar, Events: s.Events}
	for _, row := range s.Tiles {
		record.Tiles = append(record.Tiles, append([]Tile(nil), row[:]...))
	}
	for _, t := range s.Tribes {
		entry := savedTribe{Tribe: t, MonthStart: &t.monthStart, Armies: append([]Army(nil), t.Armies[:]...)}
		for _, row := range t.Explored {
			entry.Explored = append(entry.Explored, append([]bool(nil), row[:]...))
		}
		record.Tribes = append(record.Tribes, entry)
	}
	return json.MarshalIndent(record, "", "  ")
}
func UnmarshalSave(data []byte) (*State, error) {
	if len(data) > MaxSaveBytes {
		return nil, fmt.Errorf("存档超过大小限制")
	}
	var record savedGame
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return nil, fmt.Errorf("存档JSON无效：%w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("存档末尾存在额外内容")
	}
	if record.Version != SaveVersion {
		return nil, fmt.Errorf("不支持的存档版本%d（当前%d）", record.Version, SaveVersion)
	}
	return record.restore()
}
func (r savedGame) restore() (*State, error) {
	if r.Seed == nil || r.Calendar == nil || len(r.Tiles) != Height || len(r.Tribes) != TribeCount {
		return nil, fmt.Errorf("存档缺少完整地图、部落或日历")
	}
	s := New(*r.Seed)
	s.Calendar = *r.Calendar
	s.Events = r.Events
	if err := r.restoreTiles(s); err != nil {
		return nil, err
	}
	if err := r.restoreTribes(s); err != nil {
		return nil, err
	}
	if err := validateSavedRecords(s); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("存档规则状态无效：%w", err)
	}
	return s, nil
}
func (r savedTribe) restore() (Tribe, error) {
	if r.MonthStart == nil || len(r.Explored) != Height || len(r.Armies) != MaxArmies {
		return Tribe{}, fmt.Errorf("存档缺少月初库存、独立迷雾或完整军队记录")
	}
	t := r.Tribe
	t.monthStart = *r.MonthStart
	copy(t.Armies[:], r.Armies)
	for y, row := range r.Explored {
		if len(row) != Width {
			return Tribe{}, fmt.Errorf("存档迷雾宽度无效")
		}
		copy(t.Explored[y][:], row)
	}
	return t, nil
}

const saveNumberLimit = int(^uint(0)>>1) / 1000000

func validateSavedRecords(s *State) error {
	if s.Calendar.Elapsed < 0 || s.Calendar.Elapsed > saveNumberLimit {
		return fmt.Errorf("存档日历超出安全范围")
	}
	for _, t := range s.Tribes {
		if err := validateSavedTribe(t); err != nil {
			return err
		}
	}
	for _, e := range s.Events {
		if !validSavedEvent(e, s.Calendar.Elapsed) {
			return fmt.Errorf("存档事件记录无效")
		}
	}
	return nil
}
func validateSavedTribe(t Tribe) error {
	if !safeSavedText(t.Name, 128) {
		return fmt.Errorf("存档部落名称无效")
	}
	for _, n := range []int{t.Population, t.Soldiers, t.Food, t.Wood, t.monthStart.Food, t.monthStart.Wood, t.HungerMonths} {
		if n < 0 || n > saveNumberLimit {
			return fmt.Errorf("存档人口或库存超出安全范围")
		}
	}
	return nil
}
func validSavedEvent(e Event, elapsed int) bool {
	return e.Month >= 0 && e.Month <= elapsed && e.Audience != 0 && e.Audience>>TribeCount == 0 && safeSavedText(e.Text, 4096)
}
func (r savedGame) restoreTiles(s *State) error {
	for y, row := range r.Tiles {
		if len(row) != Width {
			return fmt.Errorf("存档地图宽度无效")
		}
		copy(s.Tiles[y][:], row)
	}
	return nil
}
func (r savedGame) restoreTribes(s *State) error {
	for id, entry := range r.Tribes {
		t, err := entry.restore()
		if err != nil {
			return err
		}
		s.Tribes[id] = t
	}
	return nil
}
func safeSavedText(s string, limit int) bool {
	return s != "" && len(s) <= limit && strings.IndexFunc(s, unicode.IsControl) < 0
}
