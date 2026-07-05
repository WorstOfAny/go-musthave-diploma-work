package main

import(
	"flag"
	"github.com/caarlos0/env/v11"
	"fmt"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/repository"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/agent"
)

type config struct {
	ServerAddr string `env:"RUN_ADDRESS"`
	AccrualAddr string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	RepoConfig repository.RepoConfig
	SecretPath string `env:"SECRET_PATH"`
	AgentConfig agent.Config
}

func parseFlags(cfg *config) (err error) {
	flag.StringVar(&cfg.ServerAddr, "a", "", "server address")
	flag.StringVar(&cfg.AccrualAddr, "r", "", "accrual system address")
	flag.StringVar(&cfg.RepoConfig.DatabaseDSN, "d", "", "database uri")
	flag.StringVar(&cfg.SecretPath, "secret-path", "config/secret.yml", "path to yml file with secret key")
	flag.IntVar(&cfg.AgentConfig.WorkerCount, "worker-count", 10, "worker count for processing orders")
	flag.IntVar(&cfg.AgentConfig.FetchOrderInterval, "fetch-order-interval", 5, "fetch orders for processing interval")
	flag.Parse()

	err = env.Parse(cfg)
	if err != nil {
		return fmt.Errorf("failed to read configuration from environment variables: %w", err)
	}

	return nil
}
