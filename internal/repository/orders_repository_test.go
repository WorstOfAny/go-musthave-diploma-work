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

func TestOrderCreate(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE orders, users CASCADE;")

	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	id2, err := repo.Users.Register(ctx, models.User{Login: "user2", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	orderNum := "12345674"
	loadedByOtherNum := "7992738"

	err = repo.Orders.Create(ctx, models.Order{Number: loadedByOtherNum, UID: &id2})
	if err != nil { t.Fatal(err) }

	type want struct {
		returnError bool
		err orderRepoError
	}

	testcases := []struct{
		name string
		order models.Order
		want want
	}{
		{
			name: "order does not exist", want: want { returnError: false }, order: models.Order{Number: orderNum, UID: &id},
		},
		{
			name: "order loaded by provided user", want: want { returnError: true, err: ErrOrderExist }, order: models.Order{Number: orderNum, UID: &id},
		},
		{
			name: "order loaded by other user", want: want { returnError: true, err: ErrOrderLoadedByOtherUser }, order: models.Order{Number: loadedByOtherNum, UID: &id},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err = repo.Orders.Create(ctx, tc.order)
			if tc.want.returnError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.want.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderList(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE orders, users CASCADE;")

	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }
	id2, err := repo.Users.Register(ctx, models.User{Login: "user2", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	orders := []models.Order{
		models.Order{Number: "12345674", UID: &id, Status: "NEW"},
		models.Order{Number: "7992738", UID: &id, Status: "NEW"},
	}

	err = repo.Orders.Create(ctx, orders[0])
	if err != nil { t.Fatal(err) }
	err = repo.Orders.Create(ctx, orders[1])
	if err != nil { t.Fatal(err) }

	type want struct {
		returnValue []models.Order
	}

	testcases := []struct{
		name string
		userID int
		want want
	}{
		{
			name: "orders exist", want: want { returnValue: orders },  userID: id,
		},
		{
			name: "orders does not exist", want: want { returnValue: []models.Order{} }, userID: id2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ors, err := repo.Orders.List(ctx, tc.userID)
			assert.NoError(t, err)

			sortOpt := cmpopts.SortSlices(func(a,b models.Order) bool {
				return a.Number < b.Number
			})

			ignoreOpt := cmpopts.IgnoreFields(models.Order{}, "UploadedAt")

			if diff := cmp.Diff(tc.want.returnValue, ors, sortOpt, ignoreOpt); diff != "" {
				t.Errorf("Содержимое не совпадает (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOrderUpdate(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE orders, users CASCADE;")

	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	order := models.Order{Number: "12345674", UID: &id, Status: "NEW"}

	err = repo.Orders.Create(ctx, order)
	if err != nil { t.Fatal(err) }

	type want struct {
		returnError bool
	}

	testcases := []struct{
		name string
		newStatus string
		want want
	}{
		{
			name: "update order", want: want {},  newStatus: "INVALID",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			order.Status = tc.newStatus
			err := repo.Orders.Update(ctx, order)
			assert.NoError(t, err)
		})
	}
}

func TestOrderListProcessing(t *testing.T) {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, cfg, err := initRepo(ctx, "config/db_test.yml")
	if err != nil { t.Fatal(err) }

	db, err := pgxpool.New(ctx, cfg.DatabaseDSN)
	if err != nil { t.Fatal(err) }

	defer db.Exec(ctx, "TRUNCATE orders, users CASCADE;")

	id, err := repo.Users.Register(ctx, models.User{Login: "user1", Password: "11111111",})
	if err != nil { t.Fatal(err) }

	orders := []models.Order{
		models.Order{Number: "12345674", UID: &id, Status: "NEW"},
		models.Order{Number: "7992738", UID: &id, Status: "INVALID"},
	}

	err = repo.Orders.Create(ctx, orders[0])
	if err != nil { t.Fatal(err) }
	err = repo.Orders.Create(ctx, orders[1])
	if err != nil { t.Fatal(err) }

	orders[1].Status = "INVALID"

	err = repo.Orders.Update(ctx, orders[1])
	if err != nil { t.Fatal(err) }

	type want struct {
		returnValue []models.Order
	}

	testcases := []struct{
		name string
		want want
	}{
		{
			name: "orders exist", want: want { returnValue: orders[:1] },
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ors, err := repo.Orders.ListProcessing(ctx)
			assert.NoError(t, err)

			ignoreOpt := cmpopts.IgnoreFields(models.Order{}, "UploadedAt")

			if diff := cmp.Diff(tc.want.returnValue, ors, nil, ignoreOpt); diff != "" {
				t.Errorf("Содержимое не совпадает (-want +got):\n%s", diff)
			}
		})
	}
}
