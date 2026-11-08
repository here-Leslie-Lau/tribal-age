package cli

import (
	"fmt"
	"strconv"
	"strings"

	"tribal-age/internal/game"
)

const commandHelp = "命令：help | map | status | reports | explore x y | claim x y | farm x y | train | work wood n | work auto | end | quit\n坐标从 0 开始；只有 end 推进月份。reports回看己方战报，不花行动点。work wood n 优先采木 n 格，work auto 恢复自动；安排会沿用且不花行动点。\n"

type command struct {
	name    string
	action  game.Action
	forests int
}

func parseCommand(line string) (command, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return command{}, nil
	}
	c := command{name: strings.ToLower(parts[0])}
	switch c.name {
	case "explore", "claim", "farm":
		action, err := parseTarget(c.name, parts[1:])
		if err != nil {
			return command{}, err
		}
		c.action = action
	case "work":
		forests, err := parseWork(parts[1:])
		if err != nil {
			return command{}, err
		}
		c.forests = forests
	case "help", "map", "status", "reports", "train", "end", "quit":
		if len(parts) != 1 {
			return command{}, fmt.Errorf("%s 不接受额外参数", c.name)
		}
		c.action.Kind = game.Train
	default:
		return command{}, fmt.Errorf("未知命令 %q，请输入 help", parts[0])
	}
	return c, nil
}

func parseTarget(name string, args []string) (game.Action, error) {
	if len(args) != 2 {
		return game.Action{}, fmt.Errorf("用法：%s x y", name)
	}
	x, err := strconv.Atoi(args[0])
	if err != nil {
		return game.Action{}, fmt.Errorf("x 必须是整数")
	}
	y, err := strconv.Atoi(args[1])
	if err != nil {
		return game.Action{}, fmt.Errorf("y 必须是整数")
	}
	kind := map[string]game.ActionKind{"explore": game.Explore, "claim": game.Claim, "farm": game.BuildFarm}[name]
	return game.Action{Kind: kind, Target: game.Point{X: x, Y: y}}, nil
}

func parseWork(args []string) (int, error) {
	if len(args) == 1 && strings.ToLower(args[0]) == "auto" {
		return 0, nil
	}
	if len(args) != 2 || strings.ToLower(args[0]) != "wood" {
		return 0, fmt.Errorf("用法：work wood 数量 | work auto")
	}
	forests, err := strconv.Atoi(args[1])
	if err != nil {
		return 0, fmt.Errorf("采木数量必须是整数")
	}
	return forests, nil
}
