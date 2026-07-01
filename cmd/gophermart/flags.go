package main

import(
	"flag"
	"github.com/caarlos0/env/v11"
	"fmt"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/repository"
)

type config struct {
	ServerAddr string `env:"RUN_ADDRESS"`
	AccrualAddr string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	RepoConfig repository.RepoConfig
}

func parseFlags(cfg *config) (err error) {
	flag.StringVar(&cfg.ServerAddr, "a", "", "server address")
	flag.StringVar(&cfg.AccrualAddr, "r", "", "accrual system address")
	flag.StringVar(&cfg.RepoConfig.DatabaseDSN, "d", "", "database uri")
	flag.Parse()

	err = env.Parse(cfg)
	if err != nil {
		return fmt.Errorf("failed to read configuration from environment variables: %w", err)
	}

	return nil
}
