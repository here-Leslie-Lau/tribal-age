package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"tribal-age/internal/game"
	"tribal-age/internal/tui"
)

const usage = "用法：\n  tribal-age play --seed 42          方向键界面\n  tribal-age play --cli --seed 42    原逐行命令\n  tribal-age simulate --seed 42 --months 120\n"

func Run(args []string, input io.Reader, output io.Writer) error {
	o, err := parseOptions(args)
	if errors.Is(err, flag.ErrHelp) {
		return writeText(output, usage)
	}
	if err != nil {
		return err
	}
	s := game.New(o.seed)
	if o.mode == "play" {
		if o.lineMode || !tui.CanRun(input, output) {
			return play(s, input, output)
		}
		return tui.Run(s, input, output)
	}
	return simulate(s, o.months, output)
}

type options struct {
	mode     string
	seed     int64
	months   int
	lineMode bool
}

func parseOptions(args []string) (options, error) {
	if len(args) == 0 {
		return options{}, flag.ErrHelp
	}
	mode := args[0]
	switch mode {
	case "help", "--help", "-h":
		return options{}, flag.ErrHelp
	case "play", "simulate":
	default:
		return options{}, fmt.Errorf("未知模式 %q；请使用 play 或 simulate", mode)
	}
	o := options{mode: mode, months: 120}
	flags := flag.NewFlagSet(mode, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Int64Var(&o.seed, "seed", 42, "地图随机种子")
	if mode == "simulate" {
		flags.IntVar(&o.months, "months", 120, "模拟月数（非负整数）")
	} else {
		flags.BoolVar(&o.lineMode, "cli", false, "使用原逐行命令")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("不接受额外参数：%s", strings.Join(flags.Args(), " "))
	}
	return o, nil
}

func play(s *game.State, input io.Reader, output io.Writer) error {
	v, err := s.View(0)
	if err != nil {
		return err
	}
	if err := writeText(output, renderStatus(v)+renderMap(v)+commandHelp); err != nil {
		return err
	}
	if !s.Tribes[0].Alive() {
		return writeText(output, "生存结束。\n")
	}
	scanner := bufio.NewScanner(input)
	for {
		if err := writeText(output, "> "); err != nil {
			return err
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("读取命令失败：%w", err)
			}
			return nil
		}
		c, err := parseCommand(scanner.Text())
		if err == nil && c.name == "quit" {
			return writeText(output, "已退出。\n")
		}
		var text string
		if err == nil {
			text, err = execute(s, c)
		}
		if err != nil {
			text = "错误：" + err.Error() + "\n"
		}
		if err := writeText(output, text); err != nil {
			return err
		}
		if !s.Tribes[0].Alive() {
			return writeText(output, "生存结束。\n")
		}
	}
}

func execute(s *game.State, c command) (string, error) {
	if c.name == "" {
		return "", nil
	}
	if c.name == "help" {
		return commandHelp, nil
	}
	from := len(s.Events)
	switch c.name {
	case "map", "status":
	case "work":
		if err := s.SetForestPriority(0, c.forests); err != nil {
			return "", err
		}
	case "end":
		if err := s.EndMonth(game.Interactive); err != nil {
			return "", err
		}
	default:
		if err := s.Apply(0, c.action); err != nil {
			return "", err
		}
	}
	v, err := s.View(0)
	if err != nil {
		return "", err
	}
	if c.name == "map" {
		return renderMap(v), nil
	}
	text := renderStatus(v)
	for _, event := range s.VisibleEvents(0, from) {
		text += "- " + event + "\n"
	}
	return text, nil
}

func simulate(s *game.State, months int, output io.Writer) error {
	if months < 0 {
		return fmt.Errorf("months 必须是非负整数")
	}
	start := s.Calendar.Elapsed
	for i := 0; i < months && !s.AllExtinct(); i++ {
		if err := s.EndMonth(game.Simulation); err != nil {
			return err
		}
		if err := s.Validate(); err != nil {
			return fmt.Errorf("模拟状态无效：%w", err)
		}
	}
	return writeText(output, renderSummary(s, months, s.Calendar.Elapsed-start))
}

func writeText(w io.Writer, text string) error {
	n, err := io.WriteString(w, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}
