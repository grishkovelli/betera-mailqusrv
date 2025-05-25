package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"mailqusrv/internal/config"
	"mailqusrv/internal/entities"
)

type emailRepo interface {
	BatchUpdateStatus(ctx context.Context, ids []int, status string) error
	LockPendingFailed(ctx context.Context, batchSize int) ([]entities.Email, error)
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	MarkStuckEmailsAsPending(ctx context.Context, seconds int) error
}

type Pool struct {
	conf config.Worker
	repo emailRepo
}

func NewPool(conf config.Worker, repo emailRepo) *Pool {
	return &Pool{conf, repo}
}

func (p *Pool) Run(ctx context.Context) {
	// check stuck emails
	go p.processStuckEmails(ctx)

	// run workers
	for range p.conf.PoolSize {
		go p.startWorker(ctx)
	}
}

func (p *Pool) startWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down")
			return
		default:
			var emails []entities.Email

			err := p.repo.WithTransaction(ctx, func(ctx context.Context) error {
				mails, err := p.repo.LockPendingFailed(ctx, p.conf.BatchSize)
				if err != nil {
					return fmt.Errorf("failed to get pending/failed emails: %w", err)
				}
				if len(mails) == 0 {
					return nil
				}

				ids := make([]int, len(mails))
				for i, m := range mails {
					ids[i] = m.ID
				}

				if err := p.repo.BatchUpdateStatus(ctx, ids, entities.Processing); err != nil {
					return fmt.Errorf("failed to update status to processing: %w", err)
				}

				emails = mails
				return nil
			})

			if err == nil && len(emails) > 0 {
				for status, ids := range processEmails(emails) {
					if err := p.repo.BatchUpdateStatus(ctx, ids, status); err != nil {
						log.Printf("failed to update status to %s: %v\n", status, err)
					}
				}
			} else if err != nil {
				log.Printf("transaction error: %v\n", err)
			}

			time.Sleep(time.Second)
		}
	}
}

// processStuckEmails changes the status from 'processing' back to 'pending' for emails that were not processed due to worker crashes
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

// processEmails simulates emails sending. Every second processing fails.
func processEmails(emails []entities.Email) map[string][]int {
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
