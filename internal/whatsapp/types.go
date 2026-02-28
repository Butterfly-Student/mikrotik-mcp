package whatsapp

import "strings"

// GowaWebhookPayload adalah struktur webhook dari gowa (go-whatsapp-web-multidevice)
// Format baru gowa v8: {"event":"message","device_id":"...","payload":{...}}
type GowaWebhookPayload struct {
	Event    string         `json:"event"`
	DeviceID string         `json:"device_id"`
	Payload  MessagePayload `json:"payload"`
}

type MessagePayload struct {
	ID         string `json:"id"`
	ChatID     string `json:"chat_id"`
	From       string `json:"from"`
	FromName   string `json:"from_name"`
	Timestamp  string `json:"timestamp"`
	IsFromMe   bool   `json:"is_from_me"`
	Body       string `json:"body"`
}

// ExtractPhone mengambil nomor WA bersih dari JID (e.g. "6281234@s.whatsapp.net" → "6281234")
func (p *GowaWebhookPayload) ExtractPhone() string {
	from := p.Payload.From
	if idx := strings.Index(from, "@"); idx != -1 {
		return from[:idx]
	}
	return from
}

// IsGroup cek apakah pesan dari grup
func (p *GowaWebhookPayload) IsGroup() bool {
	return strings.HasSuffix(p.Payload.ChatID, "@g.us")
}

// GowaSendRequest adalah body request ke gowa POST /send/message
type GowaSendRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}
