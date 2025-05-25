package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mailqusrv/internal/config"
	"mailqusrv/internal/entities"
)

type MockEmailService struct {
	mock.Mock
}

var _ emailService = (*MockEmailService)(nil)

func (m *MockEmailService) Create(ctx context.Context, p entities.CreateEmail) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockEmailService) GetByStatus(ctx context.Context, status string, limit int) ([]entities.Email, error) {
	args := m.Called(ctx, status, limit)
	return args.Get(0).([]entities.Email), args.Error(1)
}

func TestEmailHandler_Send(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    entities.CreateEmail
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful email creation",
			requestBody: entities.CreateEmail{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			mockError:      nil,
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "invalid email address",
			requestBody: entities.CreateEmail{
				To:      "invalid-email",
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid subject",
			requestBody: entities.CreateEmail{
				To:      "test@example.com",
				Subject: "",
				Body:    "Test Body",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid body",
			requestBody: entities.CreateEmail{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: entities.CreateEmail{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
			},
			mockError:      errors.New("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockEmailService)
			handler := NewEmailHandler(config.Server{}, mockService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/send-email", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			if tt.mockError == nil && tt.expectedStatus == http.StatusAccepted {
				mockService.On("Create", mock.Anything, tt.requestBody).Return(nil)
			} else if tt.mockError != nil {
				mockService.On("Create", mock.Anything, tt.requestBody).Return(tt.mockError)
			}

			handler.Send(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestEmailHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		mockEmails     []entities.Email
		mockError      error
		expectedStatus int
	}{
		{
			name:   "successful list pending emails",
			status: entities.Pending,
			mockEmails: []entities.Email{
				{ID: 1, To: "test1@example.com", Subject: "Test 1", Body: "Body 1", Status: entities.Pending},
				{ID: 2, To: "test2@example.com", Subject: "Test 2", Body: "Body 2", Status: entities.Pending},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid status",
			status:         "invalid",
			mockEmails:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			status:         entities.Pending,
			mockEmails:     nil,
			mockError:      errors.New("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockEmailService)
			handler := NewEmailHandler(config.Server{PageSize: 10}, mockService)

			req := httptest.NewRequest(http.MethodGet, "/emails?status="+tt.status, nil)
			w := httptest.NewRecorder()

			if tt.expectedStatus == http.StatusOK {
				mockService.On("GetByStatus", mock.Anything, tt.status, 10).Return(tt.mockEmails, nil)
			} else if tt.mockError != nil {
				mockService.On("GetByStatus", mock.Anything, tt.status, 10).Return([]entities.Email{}, tt.mockError)
			}

			handler.List(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				var response []entities.Email
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockEmails, response)
			}
			mockService.AssertExpectations(t)
		})
	}
}
