package repository

import(
	"testing"
	"github.com/stretchr/testify/assert"
	"context"
	"os/signal"
	"syscall"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	pgx "github.com/jackc/pgx/v5"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
)

func TestGet(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }
	defer db.Exec(ctx, "TRUNCATE balances, users CASCADE;")

	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	type want struct {
		returnValue *models.Balance
	}

	testcases := []struct{
		name string
		userID int
		want want
	}{
		{
			name: "balance exist",
			want: want { returnValue: &models.Balance{UID: &id, Current: new(float64), Withdrawn: new(float64)} },
			userID: id,
		},
		{
			name: "balance does not exist",
			want: want { returnValue: nil },
			userID: 2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if err != nil { t.Fatal(err) }

			bal, err := repo.Balances.Get(ctx, tc.userID)

			assert.NoError(t, err)
			assert.Equal(t, tc.want.returnValue, bal)
		})
	}
}

func TestWithdraw(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE balances, users CASCADE;")
	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	_, err = db.Exec(
		ctx,
		"UPDATE balances set current = 1000, withdrawn = 1000 WHERE user_id = @uid",
		pgx.NamedArgs{ "uid": id },
	)
	if err != nil { t.Fatal(err) }

	sum := 100.0
	sum2 := 1100.0
	orderNum := "12345674"

	type want struct {
		returnError bool
		err balanceRepoError
	}

	testcases := []struct{
		name string
		wr models.WithdrawRequest
		want want
	}{
		{
			name: "withdraw 100",
			want: want {},
			wr: models.WithdrawRequest{UID: &id, Order: orderNum, Sum: &sum},
		},
		{
			name: "withdraw 1100",
			want: want {returnError: true, err: ErrBalanceNotEnough},
			wr: models.WithdrawRequest{UID: &id, Order: orderNum, Sum: &sum2},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err = repo.Balances.Withdraw(ctx, tc.wr)
			if tc.want.returnError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.want.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
