package services

import (
	"context"

	"github.com/grishkovelli/betera-mailqusrv/internal/entities"
)

type emailRepo interface {
	Create(ctx context.Context, email entities.CreateEmail) (entities.Email, error)
	GetByStatus(ctx context.Context, status string, limit, cursor int) ([]entities.Email, error)
}

// EmailService handles business logic for email operations
type EmailService struct {
	repo emailRepo
}

// NewEmailService creates a new instance of EmailService with the provided repository
func NewEmailService(repo emailRepo) *EmailService {
	return &EmailService{repo: repo}
}

// Create creates a new email record in the system
func (s *EmailService) Create(ctx context.Context, p entities.CreateEmail) error {
	_, err := s.repo.Create(ctx, p)
	return err
}

// GetByStatus retrieves a list of emails filtered by their status
// limit specifies the maximum number of records to return
// cursor is used for pagination
func (s *EmailService) GetByStatus(ctx context.Context, status string, limit, cursor int) ([]entities.Email, error) {
	return s.repo.GetByStatus(ctx, status, limit, cursor)
}
