package agent

import(
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/client"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/models"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/repository"
	"context"
	"golang.org/x/sync/errgroup"
	"time"
	"github.com/rs/zerolog/log"
	"fmt"
)

type agent struct {
	client *client.Client
	repo *repository.Repository
}

type Config struct {
	WorkerCount int `env:"WORKER_COUNT"`
	FetchOrderInterval int `env:"FETCH_ORDER_INTERVAL"`
}

// Создание нового агента для получения информации из системы расчёта и обновления заказов
func NewAgent(repo *repository.Repository, accrualAddr string) *agent {
	return &agent{
		repo: repo,
		client: client.NewClient(accrualAddr + "/api/orders/"),
	}
}

// Запуск агента
func (a *agent) Start(ctx context.Context, cfg Config) error {
	g, errGrCtx := errgroup.WithContext(ctx)
	ticker := time.NewTicker(time.Duration(cfg.FetchOrderInterval) * time.Second)
	defer ticker.Stop()

	fetchOrdersJobs := make(chan func(context.Context) ([]models.Order, error))
	g.Go(func() error {
		defer close(fetchOrdersJobs)
		for {
			select {
				case <-errGrCtx.Done(): return nil
				case <-ticker.C: fetchOrdersJobs <- a.repo.Orders.ListProcessing
			}
		}
	})
	ordersCh := make(chan models.Order, 10)

	g.Go(func() error {
		defer close(ordersCh)
		for {
			select {
				case j := <- fetchOrdersJobs:
					orders, err := j(errGrCtx)
					if err != nil {
						log.Error().Err(err).Msg("fetch order error")
						return err
					}
					for _, o := range orders { ordersCh <- o }

				case <-ctx.Done(): return nil
			}
		}
	})

	for i := 1; i <= cfg.WorkerCount ; i++ {
		g.Go(func() error {
			for order := range ordersCh {
				err := a.processOrder(errGrCtx, order)
				if err != nil { return err }
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("worker error: %w", err)
	}

	return nil
}

func (a *agent) processOrder(ctx context.Context, o models.Order) error {
	err := a.client.Fetch(&o)

	if err != nil {
		log.Error().Err(err).Msg("fetch order error")
		return err
	}

	err = a.repo.Orders.Update(ctx, o)

	if err != nil {
		return fmt.Errorf("failed update order: %w", err)
	}
	return nil
}
