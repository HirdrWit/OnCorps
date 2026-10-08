package runner

import (
	"errors"
	"fmt"
	"log"
	"math"
	"slices"

	"github.com/hirdrwit/oncorp/internal/config"
	icsv "github.com/hirdrwit/oncorp/internal/csv"
)

// Runner runs the enabled checks on the loaded data and writes the
// flagged rows.
//
// AI-assisted (Claude Code): doc comment.
type Runner struct {
	Data         *icsv.Store
	ResultWriter *icsv.ResultWriter
	Config       *config.Settings
}

// New returns a Runner that reads from data, writes to rw, and uses cfg.
//
// AI-assisted (Claude Code): doc comment.
func New(data *icsv.Store, rw *icsv.ResultWriter, cfg config.Settings) (*Runner, error) {
	runner := &Runner{
		Data:         data,
		ResultWriter: rw,
		Config:       &cfg,
	}
	return runner, nil
}

// lookback returns the index of the row to compare rows[i] against.
type lookback func(rows []icsv.Row, i int) (int, bool)

// Execute runs each enabled check and writes its results. If a write
// fails, Execute logs the error and continues with the next check. It
// returns all write errors joined.
//
// AI-assisted (Claude Code): doc comment.
func (r *Runner) Execute() error {
	checks := []struct {
		name string
		cfg  config.Check
		back lookback
	}{
		{"day_over_day", r.Config.Checks.DayOverDay, calendarBack(0, 0, 1)},
		{"week_over_week", r.Config.Checks.WeekOverWeek, calendarBack(0, 0, 7)},
	}

	var errs []error
	for _, c := range checks {
		if c.cfg.Enabled {
			results := r.runChecker(c.name, c.cfg, c.back)
			if err := r.ResultWriter.Write(results); err != nil {
				log.Printf("write %s results: %v", c.name, err)
				errs = append(errs, fmt.Errorf("write %s: %w", c.name, err))
			}
		}
	}

	log.Printf("completed. path: %s", r.ResultWriter.Path)
	return errors.Join(errs...)
}

// runChecker compares each valid row in each file with the row that
// backFunc selects. It returns a result for each change that is larger
// than the threshold for that ticker.
//
// AI-assisted (Claude Code): doc comment.
func (r *Runner) runChecker(name string, cfg config.Check, backFunc lookback) []icsv.Result {
	var results []icsv.Result
	for _, file := range r.Data.Files {
		source := "default"
		threshold := cfg.ThresholdPct
		if override, ok := cfg.Overrides[file.Name]; ok {
			threshold = override
			source = "override"
		}

		for i, today := range file.Rows {
			if !today.Valid {
				continue
			}

			comparisonIndex, ok := backFunc(file.Rows, i)
			if !ok {
				continue
			}

			previous := file.Rows[comparisonIndex]
			change := today.Value - previous.Value
			percentChange := change / previous.Value * 100
			if math.Abs(percentChange) > threshold {
				thisResult := icsv.Result{
					Ticker:          file.Name,
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
				results = append(results, thisResult)
			}
		}
	}
	return results
}

// maxGapDays caps how far before the target date we search for a valid
// comparison row, so a hole in the data isn't reported as a one-day or
// one-week move. 4 days covers a weekend plus a holiday on either side.
//
// AI-assisted (Claude Code): code review suggested the date-based gap cap
// (maxGapDays/oldest) replacing the row-count cap; reviewed and applied by hand.
const maxGapDays = 4

// calendarBack returns a lookback that finds the last valid row on or
// before the date that is years, months, and days before rows[i]. It
// returns false if no valid row exists within maxGapDays of that date.
//
// AI-assisted (Claude Code): doc comment.
func calendarBack(years, months, days int) lookback {
	return func(rows []icsv.Row, i int) (int, bool) {
		target := rows[i].Date.AddDate(-years, -months, -days)
		oldest := target.AddDate(0, 0, -maxGapDays)
		for j, row := range slices.Backward(rows[:i]) {
			if row.Date.After(target) {
				continue
			}
			if row.Date.Before(oldest) {
				return -1, false
			}
			if row.Valid {
				return j, true
			}
		}
		return -1, false
	}
}

// getDirection returns "down" when v is less than 0. Otherwise it
// returns "up".
//
// AI-assisted (Claude Code): doc comment.
func getDirection(v float64) string {
	if v < 0 {
		return "down"
	}
	return "up"
}
