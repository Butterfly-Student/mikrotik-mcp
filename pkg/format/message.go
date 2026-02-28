package format

import "strings"

const maxMessageLen = 4000

// SplitLongMessage memecah pesan panjang menjadi beberapa chunk
// untuk dikirim ke WhatsApp (batas ~4096 karakter)
func SplitLongMessage(text string) []string {
	if len(text) <= maxMessageLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxMessageLen {
			chunks = append(chunks, text)
			break
		}

		// Cari newline terakhir sebelum batas untuk potong yang rapi
		chunk := text[:maxMessageLen]
		lastNewline := strings.LastIndex(chunk, "\n")
		if lastNewline > maxMessageLen/2 {
			chunk = text[:lastNewline+1]
		}

		chunks = append(chunks, strings.TrimRight(chunk, "\n"))
		text = strings.TrimLeft(text[len(chunk):], "\n")
	}
	return chunks
}
