package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"mailqusrv/internal/config"
	"mailqusrv/internal/db"
	"mailqusrv/internal/handlers"
	"mailqusrv/internal/repos"
	"mailqusrv/internal/services"
	"mailqusrv/internal/worker"
)

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.NewConfig()

	dbConn, err := db.NewPgxPool(cfg.DB.URL())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	wp := newWorkerPool(cfg.Worker, dbConn)
	go wp.Run(ctx)

	srv := startServer(cfg, dbConn)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}

	log.Println("shutdown complete.")
}

func startServer(cfg config.Config, dbConn *pgxpool.Pool) *http.Server {
	s := &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.Server.Port),
		Handler: routes(cfg, dbConn),
	}

	go func() {
		fmt.Printf("Server is running on port %v\n", cfg.Server.Port)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	return s
}

func routes(cfg config.Config, db *pgxpool.Pool) *http.ServeMux {
	mux := http.NewServeMux()

	emailRepo := repos.NewEmailRepo(db)
	emailSrv := services.NewEmailService(emailRepo)
	emailHdr := handlers.NewEmailHandler(cfg.Server, emailSrv)

	mux.HandleFunc("GET /emails", emailHdr.List)
	mux.HandleFunc("POST /send-email", emailHdr.Send)

	return mux
}

func newWorkerPool(c config.Worker, d *pgxpool.Pool) *worker.Pool {
	return worker.NewPool(c, repos.NewEmailRepo(d))
}
