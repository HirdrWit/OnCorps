package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Row is a single dated observation. Valid is false when the source value
// is blank (e.g. a market holiday).
type Row struct {
	Date  time.Time
	Value float64
	Valid bool
}

// File holds the contents of a single CSV file.
type File struct {
	Name string
	Rows []Row
}

// Store holds all CSV files loaded from a folder, keyed by file name.
type Store struct {
	Files map[string]*File
}

// LoadDir reads every .csv file in dir (non-recursive) into a Store.
func LoadDir(dir string) (*Store, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	store := &Store{Files: make(map[string]*File)}

	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".csv") {
			continue
		}

		f, err := read(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		store.Files[f.Name] = f
	}

	return store, nil
}

const dateLayout = "2006-01-02"

func read(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()

	r := csv.NewReader(file)
	r.FieldsPerRecord = 2

	header, err := r.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("encountered empty csv file %q", path)
	}
	if err != nil {
		return nil, fmt.Errorf("parse header %q: %w", path, err)
	}

	f := &File{Name: header[1]}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", path, err)
		}

		line, _ := r.FieldPos(0)

		date, err := time.Parse(dateLayout, record[0])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: parse date: %w", path, line, err)
		}

		row := Row{Date: date}
		if record[1] != "" {
			row.Value, err = strconv.ParseFloat(record[1], 64)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: parse value: %w", path, line, err)
			}
			row.Valid = true
		}

		f.Rows = append(f.Rows, row)
	}

	return f, nil
}
