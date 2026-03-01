package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Sender struct {
	gowaURL    string
	deviceID   string
	username   string
	password   string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewSender(gowaURL, deviceID, username, password string, logger *zap.Logger) *Sender {
	return &Sender{
		gowaURL:  gowaURL,
		deviceID: deviceID,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (s *Sender) SendText(ctx context.Context, phone, text string) error {
	// Pastikan phone dalam format JID
	jid := phone
	if len(phone) > 0 && phone[len(phone)-1] != 't' {
		jid = phone + "@s.whatsapp.net"
	}

	body, _ := json.Marshal(GowaSendRequest{
		Phone:   jid,
		Message: text,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.gowaURL+"/send/message", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.username != "" {
		req.SetBasicAuth(s.username, s.password)
	}
	if s.deviceID != "" {
		req.Header.Set("X-Device-Id", s.deviceID)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gowa send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gowa status %d", resp.StatusCode)
	}
	return nil
}

// DelayedStatus mengirim pesan status setelah delay, return func untuk cancel
func (s *Sender) DelayedStatus(ctx context.Context, phone, msg string, delay time.Duration) func() {
	t := time.AfterFunc(delay, func() {
		if err := s.SendText(ctx, phone, msg); err != nil {
			s.logger.Warn("delayed status send failed", zap.Error(err))
		}
	})
	return func() { t.Stop() }
}
