package runner

import (
	"math"

	"github.com/hirdrwit/oncorp/internal/csv"
)

const (
	defaultSource  = "default"
	overrideSource = "override"
)

func (r *Runner) dayOverDay() []flag {
	overrides := r.Config.Checks.DayOverDay.Overrides

	dodFlags := []flag{}
	for name, content := range r.Data.Files {
		source := defaultSource
		threshold := r.Config.Checks.DayOverDay.ThresholdPct
		if override, ok := overrides[name]; ok {
			threshold = override
			source = overrideSource
		}

		fileFlags := []flag{}
		var previous *csv.Row
		for _, today := range content.Rows {
			// step 1: check if valid row (holidays are skipped)
			if !today.Valid {
				continue
			}

			// step 2: check if this is the first valid value
			if previous == nil {
				previous = &today
				continue
			}

			// step 3: check if percent change exceeds threshold
			change := today.Value - previous.Value
			percentChange := change / previous.Value * 100
			if math.Abs(percentChange) > threshold {
				thisResult := flag{
					Ticker:          name,
					Check:           "day_over_day",
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
				fileFlags = append(fileFlags, thisResult)
			}

			// step 4: update previous
			previous = &today
		}
		dodFlags = append(dodFlags, fileFlags...)
	}
	return dodFlags
}

func getDirection(v float64) string {
	if v < 0 {
		return "down"
	}
	return "up"
}
