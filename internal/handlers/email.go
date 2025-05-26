package handlers

import (
	"context"
	"net/http"
	"slices"
	"strconv"

	"github.com/grishkovelli/betera-mailqusrv/internal/config"
	"github.com/grishkovelli/betera-mailqusrv/internal/entities"
)

// emailService defines the interface for email-related operations
type emailService interface {
	Create(ctx context.Context, p entities.CreateEmail) error
	GetByStatus(ctx context.Context, status string, limit, cursor int) ([]entities.Email, error)
}

// EmailHandler handles HTTP requests related to email operations
type EmailHandler struct {
	cfg          config.Server
	emailService emailService
}

// NewEmailHandler creates a new instance of EmailHandler
func NewEmailHandler(cfg config.Server, srv emailService) *EmailHandler {
	return &EmailHandler{cfg, srv}
}

// Send handles the HTTP request to create and queue a new email
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

// List handles the HTTP request to retrieve emails by their status
func (h *EmailHandler) List(w http.ResponseWriter, r *http.Request) {
	var cursor int
	status := r.URL.Query().Get("status")

	if c := r.URL.Query().Get("cursor"); c != "" {
		v, err := strconv.Atoi(c)
		if err != nil {
			renderError(w, http.StatusBadRequest, err)
			return
		}

		cursor = v
	}

	if !validateEmailStatus(status) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	emails, err := h.emailService.GetByStatus(ctx, status, h.cfg.PageSize, cursor)
	if err != nil {
		renderError(w, http.StatusInternalServerError, err)
		return
	}

	renderJSON(w, http.StatusOK, emails)
}

// validateEmailStatus checks if the provided status is valid
func validateEmailStatus(status string) bool {
	return slices.Contains([]string{entities.Pending, entities.Sent, entities.Failed}, status)
}
