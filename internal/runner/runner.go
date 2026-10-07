package runner

import (
	"fmt"
	"time"

	"github.com/hirdrwit/oncorp/internal/config"
	icsv "github.com/hirdrwit/oncorp/internal/csv"
)

type Runner struct {
	Data   *icsv.Store
	Config *config.Settings
}

type flag struct {
	Ticker          string
	Check           string // e.g. "day_over_day", "week_over_week"
	PrevDate        time.Time
	CurrDate        time.Time
	PrevValue       float64
	CurrValue       float64
	AbsChange       float64
	PctChange       float64
	ThresholdPct    float64
	ThresholdSource string // "default" or "override"
	Direction       string // "up" or "down"
}

func New(data *icsv.Store, cfg config.Settings) (*Runner, error) {
	return &Runner{Data: data, Config: &cfg}, nil
}

func (r *Runner) Execute() error {

	flags := []flag{}
	if r.Config.Checks.DayOverDay.Enabled {
		flags = append(flags, r.dayOverDay()...)
	}
	if r.Config.Checks.WeekOverWeek.Enabled {
		//		r.weekOverWeek()
	}

	for _, flag := range flags {
		fmt.Println(flag)
	}
	return nil
}
