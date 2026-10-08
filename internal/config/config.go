package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"
)

// Settings is the decoded configuration for all checks.
//
// AI-assisted (Claude Code): doc comment.
type Settings struct {
	Checks Checks
}

// Checks holds the configuration for each check.
//
// AI-assisted (Claude Code): doc comment.
type Checks struct {
	DayOverDay   Check
	WeekOverWeek Check
}

// Check is the configuration for one check. Overrides maps a ticker to a
// threshold that replaces ThresholdPct for that ticker.
//
// AI-assisted (Claude Code): doc comment.
type Check struct {
	Enabled      bool
	ThresholdPct float64
	Overrides    map[string]float64
}

// fileSettings mirrors config.yaml. Pointer fields let Load tell a missing
// key apart from an explicit false or 0, which plain values cannot.
type fileSettings struct {
	Checks *fileChecks `yaml:"checks"`
}

// fileChecks mirrors the checks key in config.yaml.
//
// AI-assisted (Claude Code): doc comment.
type fileChecks struct {
	DayOverDay   *fileCheck `yaml:"day_over_day"`
	WeekOverWeek *fileCheck `yaml:"week_over_week"`
}

// fileCheck mirrors one check in config.yaml. A nil field means that the
// key is missing.
//
// AI-assisted (Claude Code): doc comment.
type fileCheck struct {
	Enabled      *bool              `yaml:"enabled"`
	ThresholdPct *float64           `yaml:"threshold_pct"`
	Overrides    map[string]float64 `yaml:"overrides"`
}

// Load reads and strictly decodes filename. Unknown or duplicate keys are
// rejected, and every check must set both enabled and threshold_pct, so a
// typo or omission fails loudly instead of decoding to a zero value.
//
// AI-assisted (Claude Code): strict decoding and required-key checks via the
// pointer-based fileSettings types.
func Load(filename string) (*Settings, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", filename, err)
	}

	var raw fileSettings
	if err := yaml.UnmarshalStrict(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", filename, err)
	}

	settings, err := raw.settings()
	if err != nil {
		return nil, fmt.Errorf("config %q: %w", filename, err)
	}
	return settings, nil
}

// settings converts f into Settings. It returns an error for each
// required key that is missing.
//
// AI-assisted (Claude Code): doc comment.
func (f fileSettings) settings() (*Settings, error) {
	if f.Checks == nil {
		return nil, errors.New(`missing required key "checks"`)
	}
	dod, dodErr := f.Checks.DayOverDay.check("checks.day_over_day")
	wow, wowErr := f.Checks.WeekOverWeek.check("checks.week_over_week")
	if err := errors.Join(dodErr, wowErr); err != nil {
		return nil, err
	}
	return &Settings{Checks: Checks{DayOverDay: dod, WeekOverWeek: wow}}, nil
}

// check converts c into a Check. path is the YAML key path for error
// messages. It returns an error for each required key that is missing.
//
// AI-assisted (Claude Code): doc comment.
func (c *fileCheck) check(path string) (Check, error) {
	if c == nil {
		return Check{}, fmt.Errorf("missing required key %q", path)
	}
	var errs []error
	if c.Enabled == nil {
		errs = append(errs, fmt.Errorf("missing required key %q", path+".enabled"))
	}
	if c.ThresholdPct == nil {
		errs = append(errs, fmt.Errorf("missing required key %q", path+".threshold_pct"))
	}
	if err := errors.Join(errs...); err != nil {
		return Check{}, err
	}
	return Check{Enabled: *c.Enabled, ThresholdPct: *c.ThresholdPct, Overrides: c.Overrides}, nil
}

// Validate checks what decoding cannot: every threshold and override must be
// greater than 0, and every override must name one of the loaded tickers.
// All problems are reported together.
//
// AI-assisted (Claude Code): Validate and validate, including the
// case-insensitive "did you mean" hint for override tickers.
func (s *Settings) Validate(tickers []string) error {
	return errors.Join(
		s.Checks.DayOverDay.validate("checks.day_over_day", tickers),
		s.Checks.WeekOverWeek.validate("checks.week_over_week", tickers),
	)
}

// validate returns an error for each threshold, override, or override
// ticker in c that is not valid. path is the YAML key path for error messages.
//
// AI-assisted (Claude Code): doc comment.
func (c Check) validate(path string, tickers []string) error {
	var errs []error
	// !(x > 0) rather than x <= 0 so that NaN is rejected too.
	if !(c.ThresholdPct > 0) {
		errs = append(errs, fmt.Errorf("%s.threshold_pct must be > 0, got %v", path, c.ThresholdPct))
	}

	for _, ticker := range slices.Sorted(maps.Keys(c.Overrides)) {
		if pct := c.Overrides[ticker]; !(pct > 0) {
			errs = append(errs, fmt.Errorf("%s.overrides.%s must be > 0, got %v", path, ticker, pct))
		}
		if slices.Contains(tickers, ticker) {
			continue
		}
		if i := slices.IndexFunc(tickers, func(t string) bool { return strings.EqualFold(t, ticker) }); i >= 0 {
			errs = append(errs, fmt.Errorf("%s.overrides: unknown ticker %q (did you mean %q?)", path, ticker, tickers[i]))
		} else {
			errs = append(errs, fmt.Errorf("%s.overrides: unknown ticker %q (loaded: %s)", path, ticker, strings.Join(tickers, ", ")))
		}
	}
	return errors.Join(errs...)
}
