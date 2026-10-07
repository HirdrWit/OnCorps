package main

import (
	"log"

	"github.com/hirdrwit/oncorp/internal/config"
	icsv "github.com/hirdrwit/oncorp/internal/csv"
	"github.com/hirdrwit/oncorp/internal/runner"
)

const (
	INPUT_DIR   = "input"
	CONFIG_FILE = "config.yaml"
)

func main() {

	store, err := icsv.LoadDir(INPUT_DIR)
	if err != nil {
		log.Fatalf("unable to load csv input, %v", err)
	}

	cfg, err := config.Load(CONFIG_FILE)
	if err != nil {
		log.Fatalf("unable to load config %v", err)
	}

	runner, err := runner.New(store, *cfg)
	if err != nil {
		log.Fatalf("unable to create new runner %v", err)
	}

	if err = runner.Execute(); err != nil {
		log.Fatalf("error executing script %v", err)
	}
}
