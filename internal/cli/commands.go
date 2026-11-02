package cli

import (
	"fmt"
	"strconv"
	"strings"

	"tribal-age/internal/game"
)

const commandHelp = "命令：help | map | status | explore x y | claim x y | farm x y | train | end | quit\n坐标从 0 开始；只有 end 推进月份。\n"

type command struct {
	name   string
	action game.Action
}

func parseCommand(line string) (command, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return command{}, nil
	}
	c := command{name: strings.ToLower(parts[0])}
	switch c.name {
	case "explore", "claim", "farm":
		if len(parts) != 3 {
			return command{}, fmt.Errorf("用法：%s x y", c.name)
		}
		x, err := strconv.Atoi(parts[1])
		if err != nil {
			return command{}, fmt.Errorf("x 必须是整数")
		}
		y, err := strconv.Atoi(parts[2])
		if err != nil {
			return command{}, fmt.Errorf("y 必须是整数")
		}
		kind := map[string]game.ActionKind{"explore": game.Explore, "claim": game.Claim, "farm": game.BuildFarm}[c.name]
		c.action = game.Action{Kind: kind, Target: game.Point{X: x, Y: y}}
	case "help", "map", "status", "train", "end", "quit":
		if len(parts) != 1 {
			return command{}, fmt.Errorf("%s 不接受额外参数", c.name)
		}
		c.action.Kind = game.Train
	default:
		return command{}, fmt.Errorf("未知命令 %q，请输入 help", parts[0])
	}
	return c, nil
}
