package repository

import(
	"github.com/rs/zerolog/log"
	pgconn "github.com/jackc/pgx/v5/pgconn"
	"strconv"
	"errors"
	"time"
	"fmt"
	"context"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/migrations"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type Repository struct {
	Users UserRepository
	Orders OrderRepository
	Balances BalanceRepository
	WithdrawRequests WithdrawRequestRepository
}

type RepoConfig struct {
	DatabaseDSN string `env:"DATABASE_URI" yaml:"database_uri"`
}

func NewRepository(ctx context.Context, cfg *RepoConfig) (*Repository, error) {
	sourceDriver, err := iofs.New(migrations.MigrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize source driver for migrator: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrations: %w", err)
	}
	err = m.Up()

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db connections pool: %w", err)
	}

	go func(ctx context.Context, db *pgxpool.Pool) {
		<-ctx.Done()
		db.Close()
	}(ctx, db)

	repo := Repository{}
	repo.Users = &userRepo{db: db, repository: &repo}
	repo.Orders = &orderRepo{db: db, repository: &repo}
	repo.Balances = &balanceRepo{db: db, repository: &repo}
	repo.WithdrawRequests = &withdrawRequestRepo{db: db, repository: &repo}

	return &repo, nil
}

func retry(request func() (any, error), maxRetries int) (any, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		log.Debug().Str("db attempt", strconv.Itoa(attempt)).Msg("db retry")
		res, err := request()
		if err == nil { return res, nil }

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code[:2] == "08" {
			<-time.After(time.Duration(2 * (attempt + 1) - 1) * time.Second)
			continue
		}

		return nil, fmt.Errorf("failed request: %w", err)
	}

	return nil, errors.New("db unreachable")
}
