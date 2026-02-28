package mikrotik

import (
	"testing"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/stretchr/testify/assert"
)

func TestParseTrafficSentence_FullData(t *testing.T) {
	s := &proto.Sentence{
		Map: map[string]string{
			"rx-bits-per-second":    "10000000",
			"tx-bits-per-second":    "5000000",
			"rx-packets-per-second": "1500",
			"tx-packets-per-second": "800",
			"rx-drops-per-second":   "3",
			"tx-drops-per-second":   "1",
		},
	}

	stat := parseTrafficSentence(s, "ether1")

	assert.Equal(t, "ether1", stat.Interface)
	assert.Equal(t, int64(10_000_000), stat.RxBitsPerSecond)
	assert.Equal(t, int64(5_000_000), stat.TxBitsPerSecond)
	assert.Equal(t, int64(1500), stat.RxPacketsPerSecond)
	assert.Equal(t, int64(800), stat.TxPacketsPerSecond)
	assert.Equal(t, int64(3), stat.RxDropsPerSecond)
	assert.Equal(t, int64(1), stat.TxDropsPerSecond)
}

func TestParseTrafficSentence_ZeroValues(t *testing.T) {
	s := &proto.Sentence{
		Map: map[string]string{},
	}

	stat := parseTrafficSentence(s, "ether2")

	assert.Equal(t, "ether2", stat.Interface)
	assert.Equal(t, int64(0), stat.RxBitsPerSecond)
	assert.Equal(t, int64(0), stat.TxBitsPerSecond)
	assert.Equal(t, int64(0), stat.RxPacketsPerSecond)
	assert.Equal(t, int64(0), stat.TxPacketsPerSecond)
	assert.Equal(t, int64(0), stat.RxDropsPerSecond)
	assert.Equal(t, int64(0), stat.TxDropsPerSecond)
}

func TestParseTrafficSentence_InterfaceName(t *testing.T) {
	s := &proto.Sentence{Map: map[string]string{}}

	stat := parseTrafficSentence(s, "wlan1-gateway")

	assert.Equal(t, "wlan1-gateway", stat.Interface)
}

func TestParseTrafficSentence_TimestampRecent(t *testing.T) {
	s := &proto.Sentence{Map: map[string]string{}}

	stat := parseTrafficSentence(s, "ether1")

	assert.False(t, stat.Timestamp.IsZero(), "Timestamp should be set")
}
