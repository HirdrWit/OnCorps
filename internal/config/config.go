package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Settings struct {
	Checks Checks `yaml:"checks"`
}

type Checks struct {
	DayOverDay   Check `yaml:"day_over_day"`
	WeekOverWeek Check `yaml:"week_over_week"`
}

type Check struct {
	Enabled      bool               `yaml:"enabled"`
	ThresholdPct float64            `yaml:"threshold_pct"`
	Overrides    map[string]float64 `yaml:"overrides"`
}

func Load(filename string) (*Settings, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", filename, err)
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", filename, err)
	}
	return &settings, nil
}
