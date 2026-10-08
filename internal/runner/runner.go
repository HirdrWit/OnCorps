package runner

import (
	"errors"
	"fmt"
	"log"
	"math"

	"github.com/hirdrwit/oncorp/internal/config"
	icsv "github.com/hirdrwit/oncorp/internal/csv"
)

type Runner struct {
	Data        *icsv.Store
	ResultWrite *icsv.ResultWriter
	Config      *config.Settings
}

func New(data *icsv.Store, rw *icsv.ResultWriter, cfg config.Settings) (*Runner, error) {
	runner := &Runner{
		Data:        data,
		ResultWrite: rw,
		Config:      &cfg,
	}
	return runner, nil
}

// lookback returns the index of the row to compare rows[i] against.
type lookback func(rows []icsv.Row, i int) (int, bool)

func (r *Runner) Execute() error {
	checks := []struct {
		name string
		cfg  config.Check
		back lookback
	}{
		{"day_over_day", r.Config.Checks.DayOverDay, calendarBack(0, 0, 1)},
		{"week_over_week", r.Config.Checks.WeekOverWeek, calendarBack(0, 0, 7)},
	}

	errs := make([]error, 0)
	for _, c := range checks {
		if c.cfg.Enabled {
			results := r.runChecker(c.name, c.cfg, c.back)
			if err := r.ResultWrite.Write(results); err != nil {
				log.Printf("write %s results: %v", c.name, err)
				errs = append(errs, fmt.Errorf("write %s: %w", c.name, err))
			}
		}
	}

	log.Printf("completed. path: %s", r.ResultWrite.Path)
	return errors.Join(errs...)
}

func (r *Runner) runChecker(name string, cfg config.Check, backFunc lookback) []icsv.Result {
	result := []icsv.Result{}
	for _, file := range r.Data.Files {
		source := "default"
		threshold := cfg.ThresholdPct
		if override, ok := cfg.Overrides[file.GetName()]; ok {
			threshold = override
			source = "override"
		}

		fileResult := []icsv.Result{}
		rows := file.GetRows()
		for i, today := range rows {
			if !today.Valid {
				continue
			}

			comparisonIndex, ok := backFunc(rows, i)
			if !ok {
				continue
			}

			previous := rows[comparisonIndex]
			change := today.Value - previous.Value
			percentChange := change / previous.Value * 100
			if math.Abs(percentChange) > threshold {
				thisResult := icsv.Result{
					Ticker:          file.GetName(),
					Check:           name,
					PrevDate:        previous.Date,
					CurrDate:        today.Date,
					PrevValue:       previous.Value,
					CurrValue:       today.Value,
					AbsChange:       change,
					PctChange:       percentChange,
					ThresholdPct:    threshold,
					ThresholdSource: source,
					Direction:       getDirection(percentChange),
				}
				fileResult = append(fileResult, thisResult)
			}
		}
		result = append(result, fileResult...)
	}
	return result
}

func calendarBack(years, months, days int) lookback {
	return func(rows []icsv.Row, i int) (int, bool) {
		target := rows[i].Date.AddDate(years*-1, months*-1, days*-1)
		for j := i - 1; j >= 0; j-- {
			if !rows[j].Date.After(target) && rows[j].Valid {
				return j, true
			}
		}
		return -1, false
	}
}

func getDirection(v float64) string {
	if v < 0 {
		return "down"
	}
	return "up"
}
