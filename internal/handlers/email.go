package handlers

import (
	"context"
	"net/http"

	"mailqusrv/internal/config"
	"mailqusrv/internal/entities"
)

type emailService interface {
	Create(ctx context.Context, p entities.CreateEmail) error
	GetByStatus(ctx context.Context, status string, limit int) ([]entities.Email, error)
}

type EmailHandler struct {
	cfg          config.Server
	emailService emailService
}

func NewEmailHandler(cfg config.Server, srv emailService) *EmailHandler {
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
	case entities.Pending, entities.Sent, entities.Failed:
		return true
	default:
		return false
	}
}
