package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grishkovelli/betera-mailqusrv/config"
	"github.com/grishkovelli/betera-mailqusrv/internal/entities"
)

type emailRepo interface {
	BatchUpdateStatus(ctx context.Context, ids []int, status string) error
	LockPendingFailed(ctx context.Context, batchSize int) ([]entities.Email, error)
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	MarkStuckEmailsAsPending(ctx context.Context, seconds int) error
}

// Pool represents a worker pool that processes emails concurrently.
type Pool struct {
	conf   config.Worker
	repo   emailRepo
	logger *slog.Logger
}

// NewPool creates a new worker pool with the provided configuration and repository.
func NewPool(conf config.Worker, repo emailRepo, logger *slog.Logger) *Pool {
	return &Pool{conf, repo, logger}
}

// Run starts the worker pool by launching multiple worker goroutines and a goroutine to handle stuck emails.
func (p *Pool) Run(ctx context.Context) {
	// check stuck emails
	go p.processStuckEmails(ctx)

	// run workers
	for range p.conf.PoolSize {
		go p.startWorker(ctx)
	}
}

// startWorker runs a single worker that processes emails in a loop until the context is cancelled.
func (p *Pool) startWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.logger.InfoContext(ctx, "worker shutting down")
			return
		default:
			p.processEmails(ctx)
			time.Sleep(time.Second)
		}
	}
}

// processEmails handles a single batch of email processing by selecting, marking, and processing emails.
func (p *Pool) processEmails(ctx context.Context) {
	emails, err := p.selectAndMarkEmails(ctx)

	if err != nil {
		p.logger.ErrorContext(ctx, "transaction failed", "error", err)
		return
	}

	if len(emails) == 0 {
		return
	}

	p.sendAndUpdateEmails(ctx, emails)
}

// sendAndUpdateEmails processes a batch of emails by sending them and updating their status in the database.
func (p *Pool) sendAndUpdateEmails(ctx context.Context, emails []entities.Email) {
	for status, ids := range sendEmails(emails, p.logger) {
		if err := p.repo.BatchUpdateStatus(ctx, ids, status); err != nil {
			p.logger.ErrorContext(ctx, "failed to update status", "error", err)
		}
	}
}

// selectAndMarkEmails retrieves pending/failed emails and marks them as processing within a transaction.
func (p *Pool) selectAndMarkEmails(ctx context.Context) ([]entities.Email, error) {
	var emails []entities.Email

	err := p.repo.WithTransaction(ctx, func(ctx context.Context) error {
		var err error
		emails, err = p.repo.LockPendingFailed(ctx, p.conf.BatchSize)
		if err != nil {
			return fmt.Errorf("failed to get pending/failed emails: %w", err)
		}
		if len(emails) == 0 {
			return nil
		}

		ids := make([]int, len(emails))
		for i, m := range emails {
			ids[i] = m.ID
		}

		if err = p.repo.BatchUpdateStatus(ctx, ids, entities.Processing); err != nil {
			return fmt.Errorf("failed to update status to processing: %w", err)
		}

		return nil
	})
	return emails, err
}

// processStuckEmails periodically checks for and handles emails that are stuck in processing state.
func (p *Pool) processStuckEmails(ctx context.Context) {
	tkr := time.NewTicker(time.Second * time.Duration(p.conf.StuckCheckInterval))
	defer tkr.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.InfoContext(ctx, "stuck emails processing shutting down")
			return
		case <-tkr.C:
			if err := p.repo.MarkStuckEmailsAsPending(ctx, p.conf.StuckCheckInterval); err != nil {
				p.logger.InfoContext(ctx, "failed to update stuck emails", "error", err)
			}
		}
	}
}

// sendEmails simulates email processing by randomly marking emails as sent or failed.
func sendEmails(emails []entities.Email, logger *slog.Logger) map[string][]int {
	result := map[string][]int{
		entities.Sent:   make([]int, 0, len(emails)/2+1),
		entities.Failed: make([]int, 0, len(emails)/2+1),
	}

	for i, email := range emails {
		status := entities.Failed
		if i%2 == 0 {
			status = entities.Sent
		}

		result[status] = append(result[status], email.ID)

		logger.Info("email status change",
			"id", email.ID,
			"addr", email.To,
			"from", email.Status,
			"to", status)
	}

	return result
}
