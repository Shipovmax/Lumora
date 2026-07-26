package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
)

func TestSendReturnsNilOnOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/projects/test-project/messages:send", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := &FCMSender{projectID: "test-project", httpClient: server.Client(), endpoint: server.URL + "/v1/projects/%s/messages:send", ready: true}

	err := sender.Send(context.Background(), "token-1", domain.PushMessage{Title: "T", Body: "B"})
	require.NoError(t, err)
}

func TestSendReturnsErrInvalidTokenOnNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	sender := &FCMSender{projectID: "test-project", httpClient: server.Client(), endpoint: server.URL + "/v1/projects/%s/messages:send", ready: true}

	err := sender.Send(context.Background(), "stale-token", domain.PushMessage{Title: "T"})
	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestSendReturnsErrInvalidTokenOnUnregisteredBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"status":"UNREGISTERED"}}`))
	}))
	defer server.Close()

	sender := &FCMSender{projectID: "test-project", httpClient: server.Client(), endpoint: server.URL + "/v1/projects/%s/messages:send", ready: true}

	err := sender.Send(context.Background(), "stale-token", domain.PushMessage{Title: "T"})
	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestSendFailsWithoutBlockingConstructionWhenEnvNotConfigured(t *testing.T) {
	t.Setenv("FCM_PROJECT_ID", "")
	t.Setenv("FCM_CREDENTIALS_FILE", "")

	sender := NewFCMSender()

	err := sender.Send(context.Background(), "token-1", domain.PushMessage{Title: "T"})
	require.Error(t, err)
}

func TestSendReturnsErrorOnOtherFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := &FCMSender{projectID: "test-project", httpClient: server.Client(), endpoint: server.URL + "/v1/projects/%s/messages:send", ready: true}

	err := sender.Send(context.Background(), "token-1", domain.PushMessage{Title: "T"})
	require.Error(t, err)
	require.False(t, errors.Is(err, domain.ErrInvalidToken))
}
