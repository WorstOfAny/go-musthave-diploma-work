package repository

import(
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	"context"
	"os/signal"
	"syscall"
)

func TestRegister(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE users CASCADE;")

	type want struct {
		returnError bool
	}

	testcases := []struct{
		name string
		user models.User
		want want
	}{
		{
			name: "user does not exist", want: want { returnError: false }, user: models.User{Login: "user1", Password: "11111111"},
		},
		{
			name: "user exist", want: want { returnError: true }, user: models.User{Login: "user1", Password: "11111111"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := repo.Users.Register(ctx, tc.user)
			if tc.want.returnError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrLoginExist)
				assert.Equal(t, id, 0)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, id, 0)
			}
		})
	}
}

func TestAuth(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE users CASCADE;")

	_, err = repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111"})
	if err != nil { t.Fatal(err)}

	type want struct {
		returnError bool
	}

	testcases := []struct{
		name string
		user models.User
		want want
	}{
		{
			name: "user do not exist", want: want { returnError: true }, user: models.User{Login: "", Password: ""},
		},
		{
			name: "user exist, but credentials wrong", want: want { returnError: true }, user: models.User{Login: "user1", Password: "1"},
		},
		{
			name: "user exist, credentials valid", want: want { returnError: false }, user: models.User{Login: "user1", Password: "11111111"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := repo.Users.Auth(ctx, tc.user)
			if tc.want.returnError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrWrongCredentials)
				assert.Equal(t, id, 0)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, id, 0)
			}
		})
	}
}

func TestExistByID (t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }
	defer db.Exec(ctx, "TRUNCATE users CASCADE;")
	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111"})
	if err != nil { t.Fatal(err) }

	type want struct {
		returnError bool
		returnValue bool
	}

	testcases := []struct{
		name string
		id int
		want want
	}{
		{
			name: "user do not exist", want: want {}, id: id - 1,
		},
		{
			name: "user exist", want: want { returnValue: true }, id: id,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			exist, err := repo.Users.ExistByID(ctx, tc.id)
			assert.NoError(t, err)
			assert.Equal(t, tc.want.returnValue, exist)
		})
	}
}
