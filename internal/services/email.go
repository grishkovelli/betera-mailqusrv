package services

import (
	"context"
	"mailqusrv/internal/entities"
)

type emailRepo interface {
	Create(ctx context.Context, email entities.CreateEmail) (entities.Email, error)
	GetByStatus(ctx context.Context, status string, limit, cursor int) ([]entities.Email, error)
}

type EmailService struct {
	repo emailRepo
}

func NewEmailService(repo emailRepo) *EmailService {
	return &EmailService{repo: repo}
}

func (s *EmailService) Create(ctx context.Context, p entities.CreateEmail) error {
	_, err := s.repo.Create(ctx, p)
	return err
}

func (s *EmailService) GetByStatus(ctx context.Context, status string, limit, cursor int) ([]entities.Email, error) {
	return s.repo.GetByStatus(ctx, status, limit, cursor)
}
