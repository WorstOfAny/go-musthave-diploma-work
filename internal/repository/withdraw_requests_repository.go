package repository

import(
	"fmt"
	"context"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	pgx "github.com/jackc/pgx/v5"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
)

type withdrawRequestRepo struct {
	db *pgxpool.Pool
	repository *Repository
}

// Интерфейс для работы с репозиторием запросов на списание средств
type WithdrawRequestRepository interface {
	List(context.Context, int) ([]models.WithdrawRequest, error)
	Create(context.Context, models.WithdrawRequest) error
}

// Возвращает историю запросов на списание средств
func (wrr *withdrawRequestRepo) List(ctx context.Context, userID int) ([]models.WithdrawRequest, error) {
	rows, err := retry(
		func() (any, error) {
			rows, err := wrr.db.Query(
				ctx,
				"SELECT * FROM withdraw_requests WHERE user_id = @uid ORDER BY processed_at DESC",
				pgx.NamedArgs{ "uid": userID, },
			)

			if err != nil {
				return nil, fmt.Errorf("failed to fetch objects from db: %w", err)
			}

			return rows, nil
		},
		3,
	)

	if err != nil {
		return nil, fmt.Errorf("failed query: %w", err)
	}

	withdrawRequests, err := pgx.CollectRows[models.WithdrawRequest](rows.(pgx.Rows), pgx.RowToStructByName)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return withdrawRequests, nil
}

// Регистрация запроса на списание средств
func (wrr *withdrawRequestRepo) Create(ctx context.Context, wr models.WithdrawRequest) error {
	_, err := retry(
		func() (any, error) {
			_, err := wrr.db.Exec(
				ctx,
				"INSERT INTO withdraw_requests (user_id, order_number, sum_value) VALUES (@uid, @orNum, @sum)",
				pgx.NamedArgs{ "sum": wr.Sum, "orNum": wr.Order, "uid": wr.UID },
			)

			if err != nil {
				return nil, fmt.Errorf("failed register withdraw request, %w", err)
			}

			return nil, nil
		},
		3,
	)

	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	return nil
}
