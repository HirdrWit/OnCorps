package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CSVFile holds the contents of a single CSV file.
type File struct {
	Name    string
	Headers []string
	Rows    [][]string
}

// CSVStore holds all CSV files loaded from a folder, keyed by file name.
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
		store.Files[e.Name()] = f
	}

	return store, nil
}

func read(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()

	r := csv.NewReader(file)
	r.FieldsPerRecord = -1 // allow rows with differing column counts

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("encountered empty csv file %q", path)
	}

	f := &File{
		Name:    records[0][1],
		Headers: records[0],
		Rows:    records[1:],
	}
	return f, nil
}
