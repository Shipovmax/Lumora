// Package provider содержит реализации domain.Sender по вендору push-доставки.
// Выбор вендора (Firebase Cloud Messaging, покрывает и Android, и iOS через
// один API) согласован с пользователем 2026-07-25 — см. ARCHITECTURE.md,
// Этап 10; domain.Sender остаётся вендор-независимым.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

const fcmEndpoint = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

// FCMSender реализует domain.Sender через FCM HTTP v1 API напрямую (без
// firebase-admin-go — единственное, что нужно из всего Admin SDK, это
// OAuth2-токен сервисного аккаунта и один HTTP POST, лишняя зависимость на
// весь SDK не оправдана).
//
// Учётные данные читаются лениво, при первом Send, а не в NewFCMSender: так
// же, как ANTHROPIC_API_KEY у ai.ClaudeProvider, они нужны только когда
// реально есть что отправлять — воркер должен запускаться одной командой
// (см. task.md, Этап 1) даже до того, как FCM_PROJECT_ID/FCM_CREDENTIALS_FILE
// настроены; ошибка возвращается только конкретной задаче notification:push.
type FCMSender struct {
	initOnce sync.Once
	initErr  error

	projectID  string
	httpClient *http.Client
	// endpoint — формат URL FCM (%s подставляется project ID); поле, а не
	// константа, чтобы тесты могли подменить его на httptest.Server.
	endpoint string
	// ready — тесты выставляют true вместе с httpClient/projectID/endpoint,
	// чтобы обойти ленивую инициализацию реальных credentials.
	ready bool
}

func NewFCMSender() *FCMSender {
	return &FCMSender{}
}

var _ domain.Sender = (*FCMSender)(nil)

func (s *FCMSender) init(ctx context.Context) error {
	if s.ready {
		return nil
	}

	s.initOnce.Do(func() {
		projectID := os.Getenv("FCM_PROJECT_ID")
		if projectID == "" {
			s.initErr = fmt.Errorf("fcm: FCM_PROJECT_ID is required")
			return
		}

		credsFile := os.Getenv("FCM_CREDENTIALS_FILE")
		if credsFile == "" {
			s.initErr = fmt.Errorf("fcm: FCM_CREDENTIALS_FILE is required")
			return
		}

		credsJSON, err := os.ReadFile(credsFile)
		if err != nil {
			s.initErr = fmt.Errorf("fcm: read credentials file: %w", err)
			return
		}

		creds, err := google.CredentialsFromJSON(ctx, credsJSON, fcmScope)
		if err != nil {
			s.initErr = fmt.Errorf("fcm: parse credentials: %w", err)
			return
		}

		s.projectID = projectID
		s.httpClient = oauth2.NewClient(ctx, creds.TokenSource)
		s.endpoint = fcmEndpoint
	})

	return s.initErr
}

type fcmMessage struct {
	Message fcmMessageBody `json:"message"`
}

type fcmMessageBody struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *FCMSender) Send(ctx context.Context, token string, msg domain.PushMessage) error {
	if err := s.init(ctx); err != nil {
		return err
	}

	payload, err := json.Marshal(fcmMessage{Message: fcmMessageBody{
		Token:        token,
		Notification: fcmNotification{Title: msg.Title, Body: msg.Body},
		Data:         msg.Data,
	}})
	if err != nil {
		return fmt.Errorf("fcm: marshal message: %w", err)
	}

	url := fmt.Sprintf(s.endpoint, s.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("fcm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	// FCM отвечает 404 (токен не существует) или 400 с кодом UNREGISTERED/
	// INVALID_ARGUMENT для токенов, которые больше не валидны (переустановка
	// приложения без ротации, отзыв разрешения) — это сигнал удалить токен из
	// хранилища (domain.ErrInvalidToken), а не транзиентная ошибка отправки,
	// которую стоит просто залогировать и оставить токен как есть.
	if resp.StatusCode == http.StatusNotFound || strings.Contains(string(body), "UNREGISTERED") {
		return domain.ErrInvalidToken
	}

	return fmt.Errorf("fcm: send push failed: status=%d body=%s", resp.StatusCode, body)
}
