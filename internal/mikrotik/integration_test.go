//go:build integration

package mikrotik

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
)

// integrationClient returns a connected Client using env vars, or skips the test.
// It retries the connection for up to 3 minutes to accommodate slow Docker boot.
func integrationClient(t *testing.T) *Client {
	t.Helper()

	pass := os.Getenv("MIKROTIK_PASS")
	if pass == "" {
		t.Skip("skipping integration test: MIKROTIK_PASS not set")
	}

	host := os.Getenv("MIKROTIK_HOST")
	if host == "" {
		host = "192.168.88.1"
	}
	port := 8728
	if v := os.Getenv("MIKROTIK_PORT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			t.Fatalf("invalid MIKROTIK_PORT: %v", err)
		}
	}
	user := os.Getenv("MIKROTIK_USER")
	if user == "" {
		user = "admin"
	}

	cfg := Config{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
	}
	logger, _ := zap.NewDevelopment()

	// Retry dengan timeout per-attempt untuk menunggu RouterOS boot selesai.
	const (
		attemptTimeout = 8 * time.Second
		retryInterval  = 5 * time.Second
		totalTimeout   = 3 * time.Minute
	)
	deadline := time.Now().Add(totalTimeout)
	var lastErr error
	attempt := 0
	for time.Now().Before(deadline) {
		attempt++
		client := NewClient(cfg, logger)
		ctx, cancel := context.WithTimeout(context.Background(), attemptTimeout)
		err := client.Connect(ctx)
		cancel()
		if err == nil {
			t.Logf("connected to RouterOS after %d attempt(s)", attempt)
			t.Cleanup(func() { client.Close() })
			return client
		}
		lastErr = err
		t.Logf("attempt %d: %v — retry in %s", attempt, err, retryInterval)
		time.Sleep(retryInterval)
	}
	t.Fatalf("failed to connect to MikroTik after %s: %v", totalTimeout, lastErr)
	return nil
}

func TestIntegration_Connect(t *testing.T) {
	client := integrationClient(t)
	assert.NotNil(t, client)
}

func TestIntegration_IPPool_GetAll(t *testing.T) {
	client := integrationClient(t)
	repo := NewIPPoolRepository(client)

	pools, err := repo.GetAll(context.Background())

	require.NoError(t, err)
	t.Logf("found %d IP pools", len(pools))
	for _, p := range pools {
		assert.NotEmpty(t, p.Name)
	}
}

func TestIntegration_IPPool_CreateDelete(t *testing.T) {
	client := integrationClient(t)
	repo := NewIPPoolRepository(client)

	const testPool = "test-integration-pool"
	req := dto.CreateIPPoolRequest{
		Name:    testPool,
		Ranges:  "10.99.0.1-10.99.0.10",
		Comment: "integration test",
	}

	// Create
	err := repo.Create(context.Background(), req)
	require.NoError(t, err, "Create should succeed")

	// Verify it exists
	pools, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	var created *string
	for _, p := range pools {
		if p.Name == testPool {
			created = &p.ID
			break
		}
	}
	require.NotNil(t, created, "created pool should appear in list")

	// Delete
	err = repo.Delete(context.Background(), *created)
	require.NoError(t, err, "Delete should succeed")
}

func TestIntegration_Firewall_GetAll(t *testing.T) {
	client := integrationClient(t)
	repo := NewFirewallRepository(client)

	rules, err := repo.GetAll(context.Background())

	require.NoError(t, err)
	t.Logf("found %d firewall rules", len(rules))
}

func TestIntegration_Interface_GetAll(t *testing.T) {
	client := integrationClient(t)
	repo := NewInterfaceRepository(client)

	ifaces, err := repo.GetAll(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, ifaces, "router should have at least one interface")
	t.Logf("found %d interfaces", len(ifaces))
}

func TestIntegration_System_GetResource(t *testing.T) {
	client := integrationClient(t)
	repo := NewSystemRepository(client)

	res, err := repo.GetResource(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, res.Version)
	assert.NotEmpty(t, res.BoardName)
	t.Logf("RouterOS %s on %s, CPU load: %d%%", res.Version, res.BoardName, res.CPULoad)
}

func TestIntegration_System_GetLogs(t *testing.T) {
	client := integrationClient(t)
	repo := NewSystemRepository(client)

	logs, err := repo.GetLogs(context.Background(), dto.GetLogsRequest{Limit: 10})

	require.NoError(t, err)
	t.Logf("retrieved %d log entries", len(logs))
}
