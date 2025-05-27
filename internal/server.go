package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grishkovelli/betera-mailqusrv/internal/config"
	"github.com/grishkovelli/betera-mailqusrv/internal/db"
	"github.com/grishkovelli/betera-mailqusrv/internal/handlers"
	"github.com/grishkovelli/betera-mailqusrv/internal/repos"
	"github.com/grishkovelli/betera-mailqusrv/internal/services"
	"github.com/grishkovelli/betera-mailqusrv/internal/worker"
)

// Run initializes and starts the server with database connection, worker pool,
// and HTTP server. It handles graceful shutdown on system signals.
func Run() {
	cfg := config.NewConfig()

	dbConn, err := db.NewPgxPool(cfg.DB.URL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp := newWorkerPool(cfg.Worker, dbConn)
	go wp.Run(ctx)

	srv := startServer(cfg, dbConn)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err = srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v\n", err)
	}

	log.Println("shutdown complete.")
}

// startServer creates and starts an HTTP server with the given configuration
// and database connection. It returns the server instance.
func startServer(cfg config.Config, dbConn *pgxpool.Pool) *http.Server {
	s := &http.Server{
		Addr:              fmt.Sprintf(":%v", cfg.Server.Port),
		Handler:           routes(cfg, dbConn),
		ReadHeaderTimeout: time.Duration(cfg.Server.ReadHeaderTimeout) * time.Second,
	}

	go func() {
		log.Printf("Server is running on port %v\n", cfg.Server.Port)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	return s
}

// routes sets up and returns the HTTP router with all application endpoints
// configured with their respective handlers.
func routes(cfg config.Config, db *pgxpool.Pool) *http.ServeMux {
	mux := http.NewServeMux()

	emailRepo := repos.NewEmailRepo(db)
	emailSrv := services.NewEmailService(emailRepo)
	emailHdr := handlers.NewEmailHandler(cfg.Server, emailSrv)

	mux.HandleFunc("GET /emails", emailHdr.List)
	mux.HandleFunc("POST /send-email", emailHdr.Send)

	return mux
}

// newWorkerPool creates and returns a new worker pool instance with the given
// configuration and database connection.
func newWorkerPool(c config.Worker, d *pgxpool.Pool) *worker.Pool {
	return worker.NewPool(c, repos.NewEmailRepo(d))
}
