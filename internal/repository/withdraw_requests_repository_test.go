package repository

import(
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	"context"
	"os/signal"
	"syscall"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestWithdrawRequestCreate(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE withdraw_requests, users CASCADE;")

	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	orderNum := "12345674"
	sum := 100.0

	type want struct {
		returnError bool
	}

	testcases := []struct{
		name string
		wr models.WithdrawRequest
		want want
	}{
		{
			name: "create withdraw request", want: want {}, wr: models.WithdrawRequest{Order: orderNum, UID: &id, Sum: &sum},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err = repo.WithdrawRequests.Create(ctx, tc.wr)
			if tc.want.returnError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithdrawRequestList(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE withdraw_requests, users CASCADE;")

	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }
	id2, err := repo.Users.Register(ctx, models.User{Login: "user2", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	sum := 100.0
	sum2 := 1100.0

	withdrawRequests := []models.WithdrawRequest{
		models.WithdrawRequest{Order: "12345674", UID: &id, Sum: &sum},
		models.WithdrawRequest{Order: "7992738", UID: &id, Sum: &sum2},
	}

	err = repo.WithdrawRequests.Create(ctx, withdrawRequests[0])
	if err != nil { t.Fatal(err) }
	err = repo.WithdrawRequests.Create(ctx, withdrawRequests[1])
	if err != nil { t.Fatal(err) }

	type want struct {
		returnValue []models.WithdrawRequest
	}

	testcases := []struct{
		name string
		userID int
		want want
	}{
		{
			name: "withdraw requests exist", want: want { returnValue: withdrawRequests },  userID: id,
		},
		{
			name: "withdraw requests does not exist", want: want { returnValue: []models.WithdrawRequest{} }, userID: id2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			wrs, err := repo.WithdrawRequests.List(ctx, tc.userID)
			assert.NoError(t, err)

			sortOpt := cmpopts.SortSlices(func(a,b models.WithdrawRequest) bool {
				return a.Order < b.Order
			})

			ignoreOpt := cmpopts.IgnoreFields(models.WithdrawRequest{}, "ProcessedAt")

			if diff := cmp.Diff(tc.want.returnValue, wrs, sortOpt, ignoreOpt); diff != "" {
				t.Errorf("Содержимое не совпадает (-want +got):\n%s", diff)
			}
		})
	}
}
