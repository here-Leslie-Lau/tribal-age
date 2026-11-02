package game

import "fmt"

type Season uint8

const (
	Winter Season = iota
	Spring
	Summer
	Autumn
)

func (s Season) String() string {
	return [...]string{"冬季", "春季", "夏季", "秋季"}[s]
}

// Calendar stores one number so that year and month cannot drift apart.
// Its zero value is year 1, March, before any month has been settled.
type Calendar struct {
	Elapsed int
}

func (c Calendar) Year() int      { return (c.Elapsed+StartingMonth-1)/MonthsPerYear + 1 }
func (c Calendar) Month() int     { return (c.Elapsed+StartingMonth-1)%MonthsPerYear + 1 }
func (c Calendar) Season() Season { return Season(c.Month() % MonthsPerYear / MonthsPerSeason) }
func (c *Calendar) Advance()      { c.Elapsed++ }
func (c Calendar) String() string {
	return fmt.Sprintf("第 %d 年 %d 月（%s） 第 %d 回合", c.Year(), c.Month(), c.Season(), c.Elapsed+1)
}
