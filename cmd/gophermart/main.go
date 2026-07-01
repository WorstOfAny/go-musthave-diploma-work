package main

import(
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/go-chi/chi/v5"
	"net/http"
	"context"
	"os/signal"
	"syscall"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/repository"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/handler"
	"github.com/WorstOfAny/go-musthave-diploma-work/internal/agent"
	"errors"
	"golang.org/x/sync/errgroup"
	"time"
	"os"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg := &config{}
	err := parseFlags(cfg)

	if err != nil {
		panic(fmt.Errorf("can't parse environment: %w", err))
	}

	if err := run(cfg); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Info().Msg("server gracefully shutted down")
			return
		}
		if errors.Is(err, http.ErrServerClosed) {
			log.Info().Msg("server gracefully shutted down")
			return
		}
		log.Debug().Err(err).Msg("server crashed")
		panic(err)
	}
}

func run(cfg *config) error {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	repo, err := repository.NewRepository(ctx, &cfg.RepoConfig)
	if err != nil {
		return fmt.Errorf("failed initialize repository: %w", err)
	}

	var secret struct {
		key []byte `yaml:"key"`
	}

	file, err := os.Open("../../config/secret.yml")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed open file with secret :%w", err)
	}

	if file != nil {
		dec := yaml.NewDecoder(file)

		if err := dec.Decode(&secret); err != nil {
			return fmt.Errorf("failed parse yaml :%w", err)
		}
	}

	c := handler.NewUsersController(repo, secret.key)
	r := chi.NewRouter()
	c.ApplyTo(r)

	a := agent.NewAgent(repo, cfg.AccrualAddr)
	g, errGrCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return runServer(errGrCtx, cfg.ServerAddr, r)
	})

	g.Go(func() error {
		return a.Start(errGrCtx)
	})

	return g.Wait()
}

func runServer(ctx context.Context, addr string, r *chi.Mux) error {
	log.Info().Msgf("Server working on: %s", addr)
	srv := http.Server{Addr: addr, Handler: r}

	go func() {
		<- ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Info().Err(err).Msg("server forced to shutdown")
		}
	}()

	return srv.ListenAndServe()
}
