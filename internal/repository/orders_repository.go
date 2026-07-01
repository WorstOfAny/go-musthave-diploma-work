package repository

import(
	"fmt"
	"context"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	pgx "github.com/jackc/pgx/v5"
	pgconn "github.com/jackc/pgx/v5/pgconn"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	"errors"
)

type orderRepoError string

func (ore orderRepoError) Error() string {
	return string(ore)
}

// Ошибки при работе с заказами пользователя
// ErrOrderExist - такой заказ уже зарегистрирован
// ErrOrderLoadedByOtherUser - такой заказ зарегистрирован другим пользователем
const ErrOrderExist = orderRepoError("order already uploaded")
const ErrOrderLoadedByOtherUser = orderRepoError("order uploaded by other user")

type orderRepo struct {
	db *pgxpool.Pool
	repository *Repository
}

// Интерфейс для работы с репозиторием заказов
type OrderRepository interface {
	Create(context.Context, models.Order) error
	List(context.Context, int) ([]models.Order, error)
	ListProcessing(context.Context) ([]models.Order, error)
	Update(context.Context, models.Order) error
}

// Регистрация заказа, проверка, создан ли заказ другим пользователем
func (or *orderRepo) Create(ctx context.Context, o models.Order) error {
	_, err := retry(
		func() (any, error) {
			_, err := or.db.Exec(
				ctx,
				"INSERT INTO orders (user_id, number) VALUES (@uid, @number)",
				pgx.NamedArgs{ "uid": o.UID, "number": o.Number, },
			)

			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					var sameUser bool
					err := or.db.QueryRow(
						ctx,
						"SELECT user_id = @uid FROM orders WHERE number = @number",
						pgx.NamedArgs{ "uid": o.UID, "number": o.Number, },
					).Scan(&sameUser)

					if err != nil {
						return nil, fmt.Errorf("failed to fetch order from db: %w", err)
					}

					if sameUser { return nil, ErrOrderExist }
					return nil, ErrOrderLoadedByOtherUser
				}
				return nil, fmt.Errorf("failed to insert object to db: %w", err)
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

// Получение списка заказов пользователя
func (or *orderRepo) List(ctx context.Context, userID int) ([]models.Order, error) {
	rows, err := retry(
		func() (any, error) {
			rows, err := or.db.Query(
				ctx,
				"SELECT user_id, number, status, accrual, uploaded_at FROM orders WHERE user_id = @uid ORDER BY uploaded_at DESC",
				pgx.NamedArgs{ "uid": userID },
			)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, nil
				}
				return nil, fmt.Errorf("failed to insert object to db: %w", err)
			}

			return rows, nil
		},
		3,
	)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	orders, err := pgx.CollectRows[models.Order](rows.(pgx.Rows), pgx.RowToStructByName)

	if err != nil {
		return nil, fmt.Errorf("failed to parse data from db to objs: %w", err)
	}

	return orders, nil
}

// Получение списка заказов, которые не прошли проверку в системе расчётов
func (or *orderRepo) ListProcessing(ctx context.Context) ([]models.Order, error) {
	rows, err := retry(
		func() (any, error) {
			rows, err := or.db.Query(
				ctx,
				"SELECT user_id, number, status, accrual, uploaded_at FROM orders WHERE status IN ('NEW', 'PROCESSING') ORDER BY uploaded_at ASC",
			)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, nil
				}
				return nil, fmt.Errorf("failed to fetch objects from db: %w", err)
			}

			return rows, nil
		},
		3,
	)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	orders, err := pgx.CollectRows[models.Order](rows.(pgx.Rows), pgx.RowToStructByName)

	if err != nil {
		return nil, fmt.Errorf("failed to parse data from db to objs: %w", err)
	}

	return orders, nil
}

// Обновление заказа после получения информации из системы расчётов
func (or *orderRepo) Update(ctx context.Context, o models.Order) error {
	_, err := retry(
		func() (any, error) {
			tx, err := or.db.Begin(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to start transaction: %w", err)
			}
			defer tx.Rollback(ctx)
			_, err = or.db.Exec(
				ctx,
				"UPDATE orders SET status = @status, accrual = @accrual WHERE number = @number",
				pgx.NamedArgs{ "number": o.Number, "status": o.Status, "accrual": o.Accrual },
			)

			if err != nil {
				return nil, fmt.Errorf("failed to update order: %w", err)
			}

			if o.Accrual != nil {
				err := or.repository.Balances.Update(ctx, o)
				if err != nil {
					return nil, fmt.Errorf("failed to update balance: %w", err)
				}
			}

			err = tx.Commit(ctx)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
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
