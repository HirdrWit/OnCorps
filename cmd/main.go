package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/hirdrwit/oncorp/internal/config"
	icsv "github.com/hirdrwit/oncorp/internal/csv"
	"github.com/hirdrwit/oncorp/internal/runner"
)

var (
	cfgPath = flag.String("config", "config.yaml", "path to check config")
	inDir   = flag.String("in", "input", "directory of input CSVs")
	outDir  = flag.String("out", "output", "directory for result CSVs")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	store, err := icsv.LoadDir(*inDir)
	if err != nil {
		return fmt.Errorf("unable to load input: %v", err)
	}

	if err := cfg.Validate(store.Tickers()); err != nil {
		return fmt.Errorf("error validating config %q:\n%v", *cfgPath, err)
	}

	rw, err := icsv.NewResultWriter(*outDir)
	if err != nil {
		return fmt.Errorf("unable to create result writer: %v", err)
	}
	defer func() {
		if cerr := rw.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close results %q: %v", rw.Path, cerr))
		}
	}()

	r, err := runner.New(store, rw, *cfg)
	if err != nil {
		return fmt.Errorf("unable to create runner: %v", err)
	}

	if err := r.Execute(); err != nil {
		return fmt.Errorf("error running checks: %v", err)
	}
	return nil
}
