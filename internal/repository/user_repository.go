package repository

import(
	"fmt"
	"context"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	pgx "github.com/jackc/pgx/v5"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	pgconn "github.com/jackc/pgx/v5/pgconn"
	"errors"
	"github.com/rs/zerolog/log"
)

type userRepoError string

func (ure userRepoError) Error() string {
	return string(ure)
}
// Ошибки, возвращаемые репозиторием:
// ErrLoginExist - пользователь с таким логином уже зарегистрирован
// ErrWrongCredentials - неверные логин/пароль
// ErrNotFound - пользователь не найден
const ErrLoginExist = userRepoError("user with login already exist")
const ErrWrongCredentials = userRepoError("wrong pair login/password")
const ErrNotFound = userRepoError("user not found")

type userRepo struct {
	db *pgxpool.Pool
	repository *Repository
}

// Интерфейс для работы с репозиторием
type UserRepository interface {
	Register(context.Context, models.User) (int, error)
	Auth(context.Context, models.User) (int, error)
	ExistByID(context.Context, int) (bool, error)
}

// Регистрация пользователя, после успешной регистрации создается счёт пользователя
func (ur *userRepo) Register(ctx context.Context, u models.User) (int, error) {
	id, err := retry(
		func() (any, error) {
			tx, err := ur.db.Begin(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to start transaction: %w", err)
			}
			defer tx.Rollback(ctx)
			rows, err := tx.Query(
				ctx,
				"INSERT INTO users (login, password_hash) VALUES (@login, crypt(@password, gen_salt('bf', 14))) RETURNING id",
				pgx.NamedArgs{ "login": u.Login, "password": u.Password, },
			)

			if err != nil {
				return 0, fmt.Errorf("failed to insert object to db: %w", err)
			}

			id, err := pgx.CollectExactlyOneRow[int](rows, pgx.RowTo)

			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					return 0, ErrLoginExist
				}
				return 0, fmt.Errorf("failed to insert object to db: %w", err)
			}

			_, err = tx.Exec(
				ctx,
				"INSERT INTO balances (user_id) VALUES (@uid)",
				pgx.NamedArgs{ "uid": id, },
			)

			if err != nil {
				return 0, fmt.Errorf("failed to create balance for user: %w", err)
			}
			err = tx.Commit(ctx)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				return 0, fmt.Errorf("failed to commit transaction: %w", err)
			}

			return id, nil
		},
	3,
	)

	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}

	return id.(int), nil
}

// Аутентификация пользователя
func (ur *userRepo) Auth(ctx context.Context, u models.User) (int, error) {
	rows, err := retry(
		func() (any, error) {
			rows, err := ur.db.Query(
				ctx,
				"SELECT id, password_hash = (crypt(@password, password_hash)) AS password_match FROM users WHERE login = @login LIMIT 1",
				pgx.NamedArgs{ "login": u.Login, "password": u.Password, },
			)

			if err != nil {
				return nil, fmt.Errorf("failed to check password: %w", err)
			}

			return rows, nil
		},
		3,
	)

	if err != nil {
		return 0, fmt.Errorf("failed query: %w", err)
	}

	type PwMatch struct {
		PasswordMatch bool `db:"password_match"`
		ID int `db:"id"`
	}

	obj, err := pgx.CollectExactlyOneRow[PwMatch](rows.(pgx.Rows), pgx.RowToStructByName)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrWrongCredentials
		} else {
			return 0, fmt.Errorf("failed to parse obj from db to model: %w", err)
		}
	}

	if !obj.PasswordMatch {
		return 0, ErrWrongCredentials
	}

	return obj.ID, nil
}

//Проверка существования пользователя
func (ur *userRepo) ExistByID(ctx context.Context, id int) (bool, error) {
	_, err := retry(
		func() (any, error) {
			var exist int
			err := ur.db.QueryRow(
				ctx,
				"SELECT 1 FROM users WHERE id = @id LIMIT 1",
				pgx.NamedArgs{ "id": id },
			).Scan(&exist)

			if err != nil {
				return nil, fmt.Errorf("failed to fetch user by id: %w", err)
			}

			return nil, nil
		},
		3,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		log.Error().Err(err).Msg("error in ExistByID")
		return false, err
	}

	return true, nil
}
