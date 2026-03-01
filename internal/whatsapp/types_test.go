package whatsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractPhone_WithSuffix(t *testing.T) {
	p := GowaWebhookPayload{
		Payload: MessagePayload{
			From: "6281234567890@s.whatsapp.net",
		},
	}
	assert.Equal(t, "6281234567890", p.ExtractPhone())
}

func TestExtractPhone_WithoutSuffix(t *testing.T) {
	p := GowaWebhookPayload{
		Payload: MessagePayload{
			From: "6281234567890",
		},
	}
	assert.Equal(t, "6281234567890", p.ExtractPhone())
}

func TestExtractPhone_Empty(t *testing.T) {
	p := GowaWebhookPayload{}
	assert.Equal(t, "", p.ExtractPhone())
}

func TestIsGroup_GroupChat(t *testing.T) {
	p := GowaWebhookPayload{
		Payload: MessagePayload{
			ChatID: "120363402106XXXXX@g.us",
		},
	}
	assert.True(t, p.IsGroup())
}

func TestIsGroup_PrivateChat(t *testing.T) {
	p := GowaWebhookPayload{
		Payload: MessagePayload{
			ChatID: "6281234567890@s.whatsapp.net",
		},
	}
	assert.False(t, p.IsGroup())
}

func TestIsGroup_Empty(t *testing.T) {
	p := GowaWebhookPayload{}
	assert.False(t, p.IsGroup())
}
