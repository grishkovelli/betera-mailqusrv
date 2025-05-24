package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"mailqusrv/internal/config"
	hdr "mailqusrv/internal/handlers"
	"mailqusrv/internal/repos"
	src "mailqusrv/internal/services"
)

func Setup(cfg config.Config, db *pgxpool.Pool) *http.ServeMux {
	mux := http.NewServeMux()

	emailRepo := repos.NewEmailRepo(db)
	emailSrv := src.NewEmailService(emailRepo)
	emailHdr := hdr.NewEmailHandler(cfg.Server, emailSrv)

	mux.HandleFunc("GET /emails", emailHdr.List)
	mux.HandleFunc("POST /send-email", emailHdr.Send)

	return mux
}
