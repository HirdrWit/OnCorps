package runner

import (
	"github.com/hirdrwit/oncorp/internal/config"
	icsv "github.com/hirdrwit/oncorp/internal/csv"
)

type Runner struct {
	Data   *icsv.Store
	Config *config.Settings
}

func New(data *icsv.Store, cfg config.Settings) (*Runner, error) {
	return &Runner{Data: data, Config: &cfg}, nil
}

func (r *Runner) Execute() error {
	return nil
}
