package format

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitLongMessage_ShortMessage(t *testing.T) {
	msg := "Hello, MikroBot!"
	chunks := SplitLongMessage(msg)

	assert.Len(t, chunks, 1)
	assert.Equal(t, msg, chunks[0])
}

func TestSplitLongMessage_ExactLimit(t *testing.T) {
	msg := strings.Repeat("a", maxMessageLen)
	chunks := SplitLongMessage(msg)

	assert.Len(t, chunks, 1)
	assert.Equal(t, msg, chunks[0])
}

func TestSplitLongMessage_OneLonger(t *testing.T) {
	msg := strings.Repeat("a", maxMessageLen+1)
	chunks := SplitLongMessage(msg)

	assert.Len(t, chunks, 2)
	totalLen := 0
	for _, c := range chunks {
		totalLen += len(c)
	}
	assert.Equal(t, maxMessageLen+1, totalLen)
}

func TestSplitLongMessage_SplitsOnNewline(t *testing.T) {
	// Buat pesan yang tepat melebihi limit dengan newline di tengah
	half := strings.Repeat("x", maxMessageLen/2)
	msg := half + "\n" + half + strings.Repeat("y", 100)

	chunks := SplitLongMessage(msg)

	assert.GreaterOrEqual(t, len(chunks), 2)
	// Tidak ada chunk yang melebihi maxMessageLen
	for i, c := range chunks {
		assert.LessOrEqual(t, len(c), maxMessageLen,
			"chunk %d has length %d > %d", i, len(c), maxMessageLen)
	}
}

func TestSplitLongMessage_PreservesContent(t *testing.T) {
	// Pastikan konten tidak hilang setelah split
	line := strings.Repeat("ab", 100) + "\n"
	msg := strings.Repeat(line, 50) // ~10200 chars

	chunks := SplitLongMessage(msg)

	combined := strings.Join(chunks, "\n")
	// Semua karakter 'a' dan 'b' harus tetap ada
	assert.Equal(t, strings.Count(msg, "a"), strings.Count(combined, "a"))
	assert.Equal(t, strings.Count(msg, "b"), strings.Count(combined, "b"))
}

func TestSplitLongMessage_EmptyString(t *testing.T) {
	chunks := SplitLongMessage("")
	assert.Len(t, chunks, 1)
	assert.Equal(t, "", chunks[0])
}

func TestSplitLongMessage_MultipleChunks(t *testing.T) {
	// 3x limit harus menghasilkan setidaknya 3 chunk
	msg := strings.Repeat("z", maxMessageLen*3+1)
	chunks := SplitLongMessage(msg)

	assert.GreaterOrEqual(t, len(chunks), 3)
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c), maxMessageLen)
	}
}
