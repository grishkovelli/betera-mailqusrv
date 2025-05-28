package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grishkovelli/betera-mailqusrv/config"
	"github.com/grishkovelli/betera-mailqusrv/internal/handlers"
	"github.com/grishkovelli/betera-mailqusrv/internal/repos"
	"github.com/grishkovelli/betera-mailqusrv/internal/services"
	"github.com/grishkovelli/betera-mailqusrv/internal/worker"
	"github.com/grishkovelli/betera-mailqusrv/pkg/postgres"
)

// Run initializes and starts the server with database connection, worker pool, and HTTP server. It handles graceful shutdown on system signals.
func Run() {
	cfg := config.NewConfig()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dbConn, err := postgres.NewPgxPool(cfg.DB)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp := newWorkerPool(cfg.Worker, dbConn, logger)
	go wp.Run(ctx)

	s := newServer(cfg.Server, dbConn, logger)
	go func() {
		logger.Info("server is running", "port", cfg.Server.Port)
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err = s.Shutdown(ctx); err != nil {
		logger.Error("server shutdown", "error", err)
	}

	logger.Info("server shutdown complete.")
}

// newMux sets up and returns the HTTP router with all application endpoints configured.
func newMux(cfg config.Server, dbConn *pgxpool.Pool) *http.ServeMux {
	mux := http.NewServeMux()

	emailRepo := repos.NewEmailRepo(dbConn)
	emailSrv := services.NewEmailService(emailRepo)
	emailHdr := handlers.NewEmailHandler(cfg, emailSrv)

	mux.HandleFunc("GET /emails", emailHdr.List)
	mux.HandleFunc("POST /send-email", emailHdr.Send)

	return mux
}

// newWorkerPool creates and returns a new worker pool instance with the given configuration.
func newWorkerPool(c config.Worker, d *pgxpool.Pool, l *slog.Logger) *worker.Pool {
	return worker.NewPool(c, repos.NewEmailRepo(d), l)
}

// newServer creates and returns a new HTTP server with the given configuration.
func newServer(cfg config.Server, dbConn *pgxpool.Pool, logger *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%v", cfg.Port),
		Handler:           loggingAccess(logger)(newMux(cfg, dbConn)),
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
	}
}
