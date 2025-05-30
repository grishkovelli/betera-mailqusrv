package worker

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/grishkovelli/betera-mailqusrv/config"
	"github.com/grishkovelli/betera-mailqusrv/internal/entities"
)

// mockEmailRepo implements the emailRepo interface for testing.
type mockEmailRepo struct {
	emails            []entities.Email
	updateStatusCalls int
	lockEmailsCalls   int
	transactionCalls  int
	markStuckCalls    int

	updateStatusErr error
	lockEmailsErr   error
	transactionErr  error
	markStuckErr    error
}

func (m *mockEmailRepo) BatchUpdateStatus(_ context.Context, _ []int, _ string) error {
	m.updateStatusCalls++
	return m.updateStatusErr
}

func (m *mockEmailRepo) LockPendingFailed(_ context.Context, _ int) ([]entities.Email, error) {
	m.lockEmailsCalls++
	if m.lockEmailsErr != nil {
		return nil, m.lockEmailsErr
	}
	return m.emails, nil
}

func (m *mockEmailRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	m.transactionCalls++
	if m.transactionErr != nil {
		return m.transactionErr
	}
	return fn(ctx)
}

func (m *mockEmailRepo) MarkStuckEmailsAsPending(_ context.Context, _ int) error {
	m.markStuckCalls++
	return m.markStuckErr
}

func newConf() config.Worker {
	return config.Worker{
		PoolSize:           1,
		BatchSize:          2,
		StuckCheckInterval: 1,
	}
}

func newLogger() (*bytes.Buffer, *slog.Logger) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	return buf, logger
}

func TestSendEmails(t *testing.T) {
	tests := []struct {
		name   string
		emails []entities.Email
		want   map[string][]int
	}{
		{
			name:   "empty emails",
			emails: []entities.Email{},
			want: map[string][]int{
				entities.Sent:   {},
				entities.Failed: {},
			},
		},
		{
			name: "multiple emails",
			emails: []entities.Email{
				{ID: 1, To: "test1@example.com", Status: entities.Pending},
				{ID: 2, To: "test2@example.com", Status: entities.Pending},
				{ID: 3, To: "test3@example.com", Status: entities.Pending},
			},
			want: map[string][]int{
				entities.Sent:   {1, 3},
				entities.Failed: {2},
			},
		},
	}

	_, logger := newLogger()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sendEmails(tt.emails, logger)
			if len(got[entities.Sent]) != len(tt.want[entities.Sent]) {
				t.Errorf(
					"sendEmails() sent count = %v, want %v",
					len(got[entities.Sent]),
					len(tt.want[entities.Sent]),
				)
			}
			if len(got[entities.Failed]) != len(tt.want[entities.Failed]) {
				t.Errorf(
					"sendEmails() failed count = %v, want %v",
					len(got[entities.Failed]),
					len(tt.want[entities.Failed]),
				)
			}
		})
	}
}

func TestPool_StartWorker(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	mockRepo := &mockEmailRepo{
		emails: []entities.Email{
			{ID: 1, To: "test1@example.com", Status: entities.Pending},
			{ID: 2, To: "test2@example.com", Status: entities.Pending},
		},
	}

	_, logger := newLogger()
	pool := NewPool(newConf(), mockRepo, logger)
	pool.Run(ctx)

	<-ctx.Done()

	if mockRepo.lockEmailsCalls == 0 {
		t.Error("LockPendingFailed was not called")
	}
	if mockRepo.updateStatusCalls == 0 {
		t.Error("BatchUpdateStatus was not called")
	}
	if mockRepo.transactionCalls == 0 {
		t.Error("WithTransaction was not called")
	}
}

func TestPool_ProcessStuckEmails(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 1100*time.Millisecond)
	defer cancel()

	wantLogs := []string{
		`level=INFO msg="email status change" id=1 addr=test1@example.com from=processing to=sent`,
		`level=INFO msg="email status change" id=2 addr=test2@example.com from=processing to=failed`,
		`level=INFO msg="email status change" id=1 addr=test1@example.com from=processing to=sent`,
		`level=INFO msg="email status change" id=2 addr=test2@example.com from=processing to=failed`,
	}
	mockRepo := &mockEmailRepo{
		emails: []entities.Email{
			{ID: 1, To: "test1@example.com", Status: entities.Processing},
			{ID: 2, To: "test2@example.com", Status: entities.Processing},
		},
	}

	buf, logger := newLogger()
	pool := NewPool(newConf(), mockRepo, logger)
	pool.Run(ctx)

	<-ctx.Done()

	if mockRepo.markStuckCalls == 0 {
		t.Error("MarkStuckEmailsAsPending was not called")
	}

	s := buf.String()
	for _, l := range wantLogs {
		if !strings.Contains(s, l) {
			t.Errorf("Unexpected log output %s", s)
		}
	}
}

func TestPool_TransactionErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	const wantLog = `level=ERROR msg="transaction failed" error="invalid input syntax"`

	mockRepo := &mockEmailRepo{
		transactionErr: errors.New("invalid input syntax"),
	}

	buf, logger := newLogger()
	pool := NewPool(newConf(), mockRepo, logger)
	pool.Run(ctx)

	<-ctx.Done()

	if mockRepo.updateStatusCalls > 0 {
		t.Error("BatchUpdateStatus was called")
	}

	if s := buf.String(); !strings.Contains(s, wantLog) {
		t.Errorf("Unexpected log output %s", s)
	}
}
