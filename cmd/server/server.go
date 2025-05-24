package main

import (
	"fmt"
	"log"
	"net/http"

	server "mailqusrv/internal"
	"mailqusrv/internal/config"
	"mailqusrv/internal/db"
	"mailqusrv/internal/repos"
	"mailqusrv/internal/worker"
)

func main() {
	cfg := config.NewConfig()

	dbConn, _ := db.NewPgxPool(cfg.DB.URL())
	defer dbConn.Close()

	emailRepo := repos.NewEmailRepo(dbConn)
	wp := worker.NewPool(cfg.Worker, emailRepo)
	go wp.Run()

	port := fmt.Sprintf(":%v", cfg.Server.Port)
	fmt.Printf("Server is running on port %v\n", port)
	err := http.ListenAndServe(port, server.Setup(cfg, dbConn))

	log.Fatal(err)
}
