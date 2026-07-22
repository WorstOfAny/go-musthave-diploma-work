package repository

import(
	"context"
	"os"
	"gopkg.in/yaml.v3"
)

func initRepo(ctx context.Context, cfgFile string) (*Repository, *RepoConfig, error) {
	file, err := os.Open(cfgFile)
	if err != nil { return nil, nil,  err }

	defer file.Close()

	var cfg RepoConfig
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&cfg); err != nil { return nil, nil, err }

	repo, err := NewRepository(ctx, &cfg)
	if err != nil { return nil, nil, err }

	return repo, &cfg, nil
}
