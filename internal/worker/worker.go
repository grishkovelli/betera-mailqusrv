package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grishkovelli/betera-mailqusrv/internal/config"
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
	conf config.Worker
	repo emailRepo
}

// NewPool creates a new worker pool with the provided configuration and repository.
func NewPool(conf config.Worker, repo emailRepo) *Pool {
	return &Pool{conf, repo}
}

// Run starts the worker pool by launching multiple worker goroutines
// and a goroutine to handle stuck emails.
func (p *Pool) Run(ctx context.Context) {
	// check stuck emails
	go p.processStuckEmails(ctx)

	// run workers
	for range p.conf.PoolSize {
		go p.startWorker(ctx)
	}
}

// startWorker runs a single worker that processes emails in a loop
// It fetches pending/failed emails, marks them as processing, and then processes them.
func (p *Pool) startWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down")
			return
		default:
			p.processEmails(ctx)
			time.Sleep(time.Second)
		}
	}
}

// processEmails handles a single batch of email processing.
// It first selects and marks emails as processing in a transaction,
// then processes them by sending and updating their status.
// If there's an error during the transaction or no emails are found,
// it returns early without processing.
func (p *Pool) processEmails(ctx context.Context) {
	emails, err := p.selectAndMarkEmails(ctx)

	if err != nil {
		log.Printf("transaction error: %v\n", err)
		return
	}

	if len(emails) == 0 {
		return
	}

	p.sendAndUpdateEmails(ctx, emails)
}

// sendAndUpdateEmails processes a batch of emails by sending them
// and updating their status in the database.
// It groups emails by their final status (sent/failed) and performs
// batch updates to minimize database operations.
func (p *Pool) sendAndUpdateEmails(ctx context.Context, emails []entities.Email) {
	for status, ids := range sendEmails(emails) {
		if err := p.repo.BatchUpdateStatus(ctx, ids, status); err != nil {
			log.Printf("failed to update status to %s: %v\n", status, err)
		}
	}
}

// selectAndMarkEmails retrieves a batch of pending or failed emails from the database,
// marks them as "processing" within a single transaction, and returns them.
// If no emails are found, it returns an empty slice with no error.
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

// processStuckEmails periodically checks for and handles emails that are stuck in processing state
// It runs in a separate goroutine and marks emails as pending if they've been in processing state
// for longer than the configured interval.
func (p *Pool) processStuckEmails(ctx context.Context) {
	tkr := time.NewTicker(time.Second * time.Duration(p.conf.StuckCheckInterval))
	defer tkr.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("stuck emails processing shutting down")
			return
		case <-tkr.C:
			if err := p.repo.MarkStuckEmailsAsPending(ctx, p.conf.StuckCheckInterval); err != nil {
				log.Printf("failed to update stuck emails: %v\n", err)
			}
		}
	}
}

// sendEmails simulates the processing of emails by randomly marking them as sent or failed
// Returns a map of status to email IDs that were processed.
func sendEmails(emails []entities.Email) map[string][]int {
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
		log.Printf("%d %s %s -> %s\n", email.ID, email.To, email.Status, status)
	}

	return result
}
