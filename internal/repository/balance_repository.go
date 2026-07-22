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

type balanceRepoError string

func (bre balanceRepoError) Error() string {
	return string(bre)
}

// Ошибка возвращается, если недостаточно средств для списания

const ErrBalanceNotEnough = balanceRepoError("balance not enough")

type balanceRepo struct {
	db *pgxpool.Pool
	repository *Repository
}

// Интерфейс для работы с балансом пользователя

type BalanceRepository interface {
	Get(context.Context, int) (*models.Balance, error)
	Withdraw(context.Context, models.WithdrawRequest) error
	Create(context.Context, int) error
	Update(context.Context, models.Order) error
}

// Создание баланса для пользователя, необходимо наличие его идентификатора
func (br *balanceRepo) Create(ctx context.Context, userID int) error {
	_, err := retry(
		func() (any, error) {
			_, err := br.db.Exec(
				ctx,
				"INSERT INTO balances (user_id) VALUES (@uid)",
				pgx.NamedArgs{ "uid": userID, },
			)

			if err != nil {
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

// Получение баланса пользователя, необходимо наличие идентификатора

func (br *balanceRepo) Get(ctx context.Context, userID int) (*models.Balance, error) {
	rows, err := retry(
		func() (any, error) {
			rows, err := br.db.Query(
				ctx,
				"SELECT user_id, current, withdrawn FROM balances WHERE user_id = @uid",
				pgx.NamedArgs{ "uid": userID, },
			)

			if err != nil {
				return nil, fmt.Errorf("failed to fetch object from db: %w", err)
			}

			return rows, nil
		},
		3,
	)

	if err != nil {
		return nil, fmt.Errorf("failed query: %w", err)
	}

	balance, err := pgx.CollectExactlyOneRow[models.Balance](rows.(pgx.Rows), pgx.RowToStructByName)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return &balance, nil
}


// Снятие средств с баланса пользователя в пользу оплаты заказа, необходим запрос на снятие, тут же регистрируется запрос на снятие
func (br *balanceRepo) Withdraw(ctx context.Context, wr models.WithdrawRequest)  error {
	_, err := retry(
		func() (any, error) {
			tx, err := br.db.Begin(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to start transaction: %w", err)
			}
			defer tx.Rollback(ctx)
			_, err = tx.Exec(
				ctx,
				"UPDATE balances SET current = current - @sum, withdrawn = withdrawn + @sum WHERE user_id = @uid",
				pgx.NamedArgs{ "sum": wr.Sum, "uid": wr.UID },
			)
			
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23514" {
					return 0, ErrBalanceNotEnough
				}
				return 0, fmt.Errorf("failed to update balance: %w", err)
			}

			_, err = tx.Exec(
				ctx,
				"INSERT INTO withdraw_requests (user_id, order_number, sum_value) VALUES (@uid, @orNum, @sum)",
				pgx.NamedArgs{ "sum": wr.Sum, "orNum": wr.Order, "uid": wr.UID },
			)
			if err != nil {
				return nil, fmt.Errorf("failed register withdraw request, %w", err)
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

// Обновление баланса пользователя после получение информации от системы расчетов, необходим заказ, который обработан системой расчётов
func (br *balanceRepo) Update(ctx context.Context, o models.Order) error {
	_, err := retry(
		func() (any, error) {
			_, err := br.db.Exec(
				ctx,
				"UPDATE balances SET current = current + @accrual WHERE user_id = @uid",
				pgx.NamedArgs{ "accrual": o.Accrual, "uid": o.UID },
			)

			if err != nil {
				return nil, fmt.Errorf("failed to update balance: %w", err)
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
