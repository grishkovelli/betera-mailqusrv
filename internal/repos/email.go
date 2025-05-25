package repos

import (
	"context"
	"mailqusrv/internal/entities"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailRepo struct {
	db *pgxpool.Pool
}

func NewEmailRepo(db *pgxpool.Pool) *EmailRepo {
	return &EmailRepo{db: db}
}

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

// TODO: add pagination
func (r *EmailRepo) GetByStatus(ctx context.Context, status string, limit int) ([]entities.Email, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, to_address, subject, body, status
		FROM emails
		WHERE status = $1 LIMIT $2
	`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[entities.Email])
}

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
