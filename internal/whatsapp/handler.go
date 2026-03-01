package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"mikrotik-mcp/pkg/format"
)

// Processor adalah interface yang diimplementasikan oleh Orchestrator.
// Dipisahkan agar Handler mudah di-mock dalam testing.
type Processor interface {
	Process(ctx context.Context, phone, accessLevel, text string) (string, error)
}

// MessageSender adalah interface untuk pengiriman pesan ke WhatsApp.
type MessageSender interface {
	SendText(ctx context.Context, phone, text string) error
	DelayedStatus(ctx context.Context, phone, msg string, delay time.Duration) func()
}

type Handler struct {
	orch          Processor
	sender        MessageSender
	auth          *Middleware
	webhookSecret string
	logger        *zap.Logger
}

func NewHandler(orch Processor, sender MessageSender, auth *Middleware, webhookSecret string, logger *zap.Logger) *Handler {
	return &Handler{
		orch:          orch,
		sender:        sender,
		auth:          auth,
		webhookSecret: webhookSecret,
		logger:        logger,
	}
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verifikasi HMAC jika webhook_secret dikonfigurasi
	if h.webhookSecret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !h.verifySignature(body, sig) {
			h.logger.Warn("webhook signature mismatch — request ditolak")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var p GowaWebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Hanya proses event "message"
	if p.Event != "message" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Abaikan: pesan dari diri sendiri, grup, pesan kosong
	if p.Payload.IsFromMe || p.IsGroup() || strings.TrimSpace(p.Payload.Body) == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Balas 200 segera ke gowa — tidak boleh timeout
	w.WriteHeader(http.StatusOK)

	go h.process(p)
}

// verifySignature memvalidasi header X-Hub-Signature-256: sha256={hex}
func (h *Handler) verifySignature(body []byte, sig string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(sig, prefix) {
		return false
	}
	got, err := hex.DecodeString(sig[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(got, expected)
}

func (h *Handler) process(p GowaWebhookPayload) {
	phone := p.ExtractPhone()
	text := strings.TrimSpace(p.Payload.Body)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Auth check
	if !h.auth.IsAuthorized(phone) {
		h.logger.Warn("unauthorized access attempt", zap.String("phone", phone))
		_ = h.sender.SendText(ctx, phone, "Nomor Anda tidak terdaftar untuk menggunakan layanan ini.")
		return
	}

	// Rate limit
	if !h.auth.Allow(phone) {
		_ = h.sender.SendText(ctx, phone, "Terlalu banyak permintaan. Tunggu sebentar.")
		return
	}

	// Kirim "sedang memproses" jika lebih dari 3 detik
	stopStatus := h.sender.DelayedStatus(ctx, phone, "_Sedang memproses..._", 3*time.Second)

	accessLevel := h.auth.GetAccessLevel(phone)
	response, err := h.orch.Process(ctx, phone, accessLevel, text)
	stopStatus()

	if err != nil {
		h.logger.Error("process failed", zap.String("phone", phone), zap.Error(err))
		_ = h.sender.SendText(ctx, phone, "Terjadi kesalahan. Silakan coba lagi.")
		return
	}

	// Kirim response — multi-chunk jika panjang
	chunks := format.SplitLongMessage(response)
	for i, chunk := range chunks {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}
		if err := h.sender.SendText(ctx, phone, chunk); err != nil {
			h.logger.Error("send failed", zap.Error(err))
			break
		}
	}
}
