package repos

import (
	"context"

	"github.com/grishkovelli/betera-mailqusrv/internal/entities"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EmailRepo handles all database operations related to emails
type EmailRepo struct {
	db *pgxpool.Pool
}

// NewEmailRepo creates a new instance of EmailRepo
func NewEmailRepo(db *pgxpool.Pool) *EmailRepo {
	return &EmailRepo{db: db}
}

// Create inserts a new email record into the database and returns the created email
func (r *EmailRepo) Create(ctx context.Context, email entities.CreateEmail) (entities.Email, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO emails (to_address, subject, body)
		VALUES ($1, $2, $3)
		RETURNING id, to_address, subject, body, status
	`, email.To, email.Subject, email.Body)
	if err != nil {
		return entities.Email{}, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByName[entities.Email])
}

// GetByStatus retrieves emails with the specified status, using cursor-based pagination
func (r *EmailRepo) GetByStatus(ctx context.Context, status string, limit, cursor int) ([]entities.Email, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, to_address, subject, body, status
		FROM emails
		WHERE id > $1
			AND status = $2
		ORDER BY id
		LIMIT $3
	`, cursor, status, limit)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByName[entities.Email])
}

// WithTransaction executes the provided function within a database transaction
func (r *EmailRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = fn(ctx)
	return err
}

// LockPendingFailed locks and retrieves a batch of pending or failed emails for processing
func (r *EmailRepo) LockPendingFailed(ctx context.Context, batchSize int) ([]entities.Email, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, to_address, subject, body, status
		FROM emails
		WHERE status IN ('pending', 'failed')
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[entities.Email])
}

// BatchUpdateStatus updates the status of multiple emails by their IDs
func (r *EmailRepo) BatchUpdateStatus(ctx context.Context, ids []int, status string) error {
	if len(ids) == 0 {
		return nil
	}

	_, err := r.db.Exec(ctx, `
		UPDATE emails
		SET status = $1,
				updated_at = NOW()
		WHERE id = ANY($2)
	`, status, ids)
	return err
}

// MarkStuckEmailsAsPending resets the status of emails that have been in 'processing' state for too long
func (r *EmailRepo) MarkStuckEmailsAsPending(ctx context.Context, seconds int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE emails
		SET status = 'pending',
				updated_at = NOW()
		WHERE status = 'processing'
		AND updated_at < NOW() - ($1 * INTERVAL '1 second')
	`, seconds)

	return err
}
