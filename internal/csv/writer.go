package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var csvHeader = []string{
	"ticker", "check", "prev_date", "curr_date", "prev_value", "curr_value",
	"abs_change", "pct_change", "threshold_pct", "threshold_source", "direction",
}

// Result is one flagged row, produced by any check runner.
type Result struct {
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

func (r Result) record() []string {
	const dateFmt = "2006-01-02"
	return []string{
		r.Ticker,
		r.Check,
		r.PrevDate.Format(dateFmt),
		r.CurrDate.Format(dateFmt),
		strconv.FormatFloat(r.PrevValue, 'f', 2, 64),
		strconv.FormatFloat(r.CurrValue, 'f', 2, 64),
		strconv.FormatFloat(r.AbsChange, 'f', 2, 64),
		strconv.FormatFloat(r.PctChange, 'f', 4, 64),
		strconv.FormatFloat(r.ThresholdPct, 'f', 2, 64),
		r.ThresholdSource,
		r.Direction,
	}
}

// ResultWriter writes results to a new CSV file for each run. It is not
// safe for concurrent use; checks run one after another and share one writer.
type ResultWriter struct {
	f    *os.File
	w    *csv.Writer
	Path string
}

// NewResultWriter creates dir if needed, then creates a new timestamped
// results file in it (e.g. 20261008T113631_result.csv) and writes the header.
// It fails rather than overwrite if a file with that name already exists.
//
// AI-assisted (Claude Code): colon-free file name, create-exclusive instead
// of append mode, MkdirAll, and the Path field.
func NewResultWriter(dir string) (*ResultWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir %q: %w", dir, err)
	}

	fileName := time.Now().Format("20060102T150405") + "_result.csv"
	fullPath := filepath.Join(dir, fileName)

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create %q: %w", fullPath, err)
	}

	rw := &ResultWriter{f: f, w: csv.NewWriter(f), Path: fullPath}

	if err := rw.w.Write(csvHeader); err != nil {
		f.Close()
		return nil, fmt.Errorf("write header to %q: %w", fullPath, err)
	}
	rw.w.Flush()
	if err := rw.w.Error(); err != nil {
		f.Close()
		return nil, fmt.Errorf("write header to %q: %w", fullPath, err)
	}
	return rw, nil
}

// Write appends all results as rows and flushes them to the file.
func (rw *ResultWriter) Write(results []Result) error {
	for _, r := range results {
		if err := rw.w.Write(r.record()); err != nil {
			return fmt.Errorf("write record for %s: %w", r.Ticker, err)
		}
	}
	rw.w.Flush()
	return rw.w.Error()
}

func (rw *ResultWriter) Close() error {
	rw.w.Flush()
	if err := rw.w.Error(); err != nil {
		rw.f.Close()
		return err
	}
	return rw.f.Close()
}
