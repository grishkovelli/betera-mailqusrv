package handlers

import (
	"context"
	"net/http"

	"mailqusrv/internal/config"
	"mailqusrv/internal/entities"
	srv "mailqusrv/internal/services"
)

type EmailHandler struct {
	cfg          config.Server
	emailService *srv.EmailService
}

func NewEmailHandler(cfg config.Server, srv *srv.EmailService) *EmailHandler {
	return &EmailHandler{cfg, srv}
}

func (h *EmailHandler) Send(w http.ResponseWriter, r *http.Request) {
	params := entities.CreateEmail{}

	if err := validateParams(r, &params); err != nil {
		renderError(w, http.StatusBadRequest, err)
		return
	}

	ctx := context.Background()
	if err := h.emailService.Create(ctx, params); err != nil {
		renderError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// TODO: add pagination using ID range
func (h *EmailHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	if !validateEmailStatus(status) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	emails, err := h.emailService.GetByStatus(ctx, status, h.cfg.PageSize)
	if err != nil {
		renderError(w, http.StatusInternalServerError, err)
		return
	}

	renderJSON(w, http.StatusOK, emails)
}

func validateEmailStatus(status string) bool {
	switch status {
	case "pending", "sent", "failed":
		return true
	default:
		return false
	}
}
