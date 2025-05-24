package worker

import (
	"context"
	"fmt"
	"log"
	"mailqusrv/internal/config"
	"mailqusrv/internal/entities"
	"time"
)

const (
	failed     = "failed"
	pending    = "pending"
	processing = "processing"
	sent       = "sent"
)

type emailRepo interface {
	BatchUpdateStatus(ctx context.Context, ids []int, status string) error
	GetPendingOrFailed(ctx context.Context, batchSize int) ([]entities.Email, error)
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

func (p *Pool) Run() {
	go p.processStuckEmails()

	for range p.conf.PoolSize {
		go func() {
			ctx := context.Background()

			for {
				emails := make([]entities.Email, 0, p.conf.BatchSize)

				err := p.repo.WithTransaction(ctx, func(ctx context.Context) error {
					mails, err := p.repo.GetPendingOrFailed(ctx, p.conf.BatchSize)
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

					if err := p.repo.BatchUpdateStatus(ctx, ids, processing); err != nil {
						return fmt.Errorf("failed to update status to processing: %w", err)
					}

					emails = mails
					return nil
				})

				if err != nil {
					continue
				}

				for status, ids := range processEmails(emails) {
					p.repo.BatchUpdateStatus(ctx, ids, status)
				}

				time.Sleep(time.Second)
			}
		}()
	}
}

func (p *Pool) processStuckEmails() {
	ctx := context.Background()
	tkr := time.NewTicker(time.Second * time.Duration(p.conf.StuckCheckInterval))
	defer tkr.Stop()

	for {
		select {
		case <-tkr.C:
			if err := p.repo.MarkStuckEmailsAsPending(ctx, p.conf.StuckCheckInterval); err != nil {
				log.Printf("failed to update stuck emails: %v\n", err)
			}
		}
	}
}

func processEmails(emails []entities.Email) map[string][]int {
	// simulate processing
	time.Sleep(time.Microsecond * 200)

	result := map[string][]int{
		sent:   make([]int, 0, len(emails)/2+1),
		failed: make([]int, 0, len(emails)/2+1),
	}

	for i, email := range emails {
		status := failed
		// every second one will be a failure
		if i%2 == 0 {
			result[sent] = append(result[sent], email.ID)
			status = sent
		} else {
			result[failed] = append(result[failed], email.ID)
		}

		log.Printf("%d %s %s -> %s\n", email.ID, email.To, email.Status, status)
	}

	return result
}
