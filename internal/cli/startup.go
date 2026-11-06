package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"tribal-age/internal/game"
	"tribal-age/internal/savegame"
	"tribal-age/internal/tui"
)

func runPlay(o options, input io.Reader, output io.Writer) (err error) {
	store, err := savegame.OpenDefault()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	if !o.lineMode && tui.CanRun(input, output) {
		return tui.RunSaved(o.seed, store, input, output)
	}
	scanner := bufio.NewScanner(input)
	s, err := chooseLineGame(o.seed, store, scanner, output)
	if err != nil || s == nil {
		return err
	}
	return playScanner(s, scanner, output, store.Save)
}
func chooseLineGame(seed int64, store *savegame.Store, scanner *bufio.Scanner, out io.Writer) (*game.State, error) {
	saved, problem := store.Load()
	if errors.Is(problem, os.ErrNotExist) {
		return game.New(seed), nil
	}
	for {
		if err := writeText(out, lineStartupSummary(saved, problem)); err != nil {
			return nil, err
		}
		if !scanner.Scan() {
			return nil, scanner.Err()
		}
		state, done, err := lineGameChoice(scanner.Text(), seed, saved, scanner, out)
		if done || err != nil {
			return state, err
		}
	}
}

func lineGameChoice(choice string, seed int64, saved *game.State, scanner *bufio.Scanner, out io.Writer) (*game.State, bool, error) {
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "1":
		return saved, saved != nil, nil
	case "2":
		accepted, err := confirmLineNew(scanner, out)
		if err != nil || !accepted {
			return nil, false, err
		}
		return game.New(seed), true, nil
	case "3", "q", "quit":
		return nil, true, nil
	}
	return nil, false, nil
}
func confirmLineNew(scanner *bufio.Scanner, out io.Writer) (bool, error) {
	if err := writeText(out, "确认开始新游戏？新局首月结算成功才覆盖旧存档。[Y]确认 [其余]取消\n"); err != nil {
		return false, err
	}
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	return strings.EqualFold(strings.TrimSpace(scanner.Text()), "y"), nil
}
func lineStartupSummary(saved *game.State, problem error) string {
	text := "部落纪元\n[1]继续游戏 [2]新游戏 [3]退出\n"
	if problem != nil {
		return text + fmt.Sprintf("无法读取存档：%v；原档保留。\n", problem)
	}
	return text + fmt.Sprintf("存档：%s，人口%d，种子%d\n", saved.Calendar, saved.Tribes[0].Population, saved.Seed)
}
