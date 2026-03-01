//go:build integration

package mikrotik

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// integrationConfig loads connection settings from env / .env file.
// Priority: env vars → .env file → defaults (host: 192.168.233.1, user: admin).
func integrationConfig(t *testing.T) Config {
	t.Helper()
	_ = godotenv.Load("../../../.env") // best-effort; ignore missing file

	pass := os.Getenv("MIKROTIK_PASSWORD")
	if pass == "" {
		pass = os.Getenv("MIKROTIK_PASS")
	}
	if pass == "" {
		t.Skip("skipping integration test: MIKROTIK_PASSWORD not set")
	}

	host := os.Getenv("MIKROTIK_HOST")
	if host == "" {
		host = "192.168.233.1"
	}
	port := 8728
	if v := os.Getenv("MIKROTIK_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port) //nolint:errcheck
	}
	user := os.Getenv("MIKROTIK_USER")
	if user == "" {
		user = "admin"
	}

	return Config{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		Timeout:  10 * time.Second,
	}
}

// integrationClient creates a connected Client with automatic cleanup.
func integrationClient(t *testing.T) *Client {
	t.Helper()
	cfg := integrationConfig(t)
	logger, _ := zap.NewDevelopment()

	const (
		attemptTimeout = 8 * time.Second
		retryInterval  = 5 * time.Second
		totalTimeout   = 30 * time.Second
	)
	deadline := time.Now().Add(totalTimeout)
	var lastErr error
	for attempt := 1; time.Now().Before(deadline); attempt++ {
		c := NewClient(cfg, logger)
		ctx, cancel := context.WithTimeout(context.Background(), attemptTimeout)
		err := c.Connect(ctx)
		cancel()
		if err == nil {
			t.Logf("connected to %s after %d attempt(s)", cfg.Host, attempt)
			t.Cleanup(func() { c.Close() })
			return c
		}
		lastErr = err
		t.Logf("attempt %d failed: %v — retry in %s", attempt, err, retryInterval)
		time.Sleep(retryInterval)
	}
	t.Fatalf("could not connect to %s: %v", cfg.Host, lastErr)
	return nil
}

// firstRunningInterface returns the name of the first running non-disabled interface.
func firstRunningInterface(t *testing.T, client *Client) string {
	t.Helper()
	repo := NewInterfaceRepository(client)
	ifaces, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	for _, iface := range ifaces {
		if iface.Running && !iface.Disabled {
			return iface.Name
		}
	}
	t.Skip("no running interface found on router")
	return ""
}

// ─── Connection & Async mode ──────────────────────────────────────────────────

func TestIntegration_Connect(t *testing.T) {
	c := integrationClient(t)
	assert.NotNil(t, c)
}

func TestIntegration_Async_IsAsync(t *testing.T) {
	c := integrationClient(t)
	assert.True(t, c.IsAsync(), "connection must be in async mode after Connect()")
}

func TestIntegration_RunContext_WithTimeout(t *testing.T) {
	c := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, err := c.RunContext(ctx, "/system/identity/print")
	require.NoError(t, err)
	assert.NotEmpty(t, reply.Re)
}

func TestIntegration_RunMany_Concurrent(t *testing.T) {
	c := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Five independent commands fired concurrently over the same async connection.
	commands := [][]string{
		{"/system/resource/print"},
		{"/system/identity/print"},
		{"/ip/pool/print"},
		{"/interface/print"},
		{"/ip/firewall/filter/print"},
	}

	start := time.Now()
	replies, errs := c.RunMany(ctx, commands)
	elapsed := time.Since(start)

	for i, err := range errs {
		require.NoError(t, err, "command %d (%s) failed", i, commands[i][0])
	}
	assert.NotEmpty(t, replies[0].Re, "resource/print must return data")
	assert.NotEmpty(t, replies[1].Re, "identity/print must return data")
	assert.NotEmpty(t, replies[3].Re, "interface/print must return data")
	t.Logf("5 concurrent commands finished in %s via single async conn", elapsed)
}

// ─── IP Pool ──────────────────────────────────────────────────────────────────

func TestIntegration_IPPool_GetAll(t *testing.T) {
	repo := NewIPPoolRepository(integrationClient(t))

	pools, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	t.Logf("found %d IP pools:", len(pools))
	for _, p := range pools {
		assert.NotEmpty(t, p.Name)
		t.Logf("  %-20s ranges=%-30s next=%s", p.Name, p.Ranges, p.NextPool)
	}
}

func TestIntegration_IPPool_CreateUpdateDelete(t *testing.T) {
	repo := NewIPPoolRepository(integrationClient(t))
	ctx := context.Background()
	const testPool = "test-integration-pool"

	require.NoError(t, repo.Create(ctx, dto.CreateIPPoolRequest{
		Name:    testPool,
		Ranges:  "10.99.0.1-10.99.0.10",
		Comment: "integration test — safe to delete",
	}))

	pools, err := repo.GetAll(ctx)
	require.NoError(t, err)
	var id string
	for _, p := range pools {
		if p.Name == testPool {
			id = p.ID
		}
	}
	require.NotEmpty(t, id, "created pool must appear in list")
	t.Logf("created pool id=%s", id)

	require.NoError(t, repo.Update(ctx, dto.UpdateIPPoolRequest{
		ID:      id,
		Ranges:  "10.99.0.1-10.99.0.20",
		Comment: "integration test updated",
	}))
	updated, err := repo.GetByName(ctx, testPool)
	require.NoError(t, err)
	assert.Equal(t, "10.99.0.1-10.99.0.20", updated.Ranges)

	require.NoError(t, repo.Delete(ctx, id))
	pools, err = repo.GetAll(ctx)
	require.NoError(t, err)
	for _, p := range pools {
		assert.NotEqual(t, testPool, p.Name, "pool must be gone after delete")
	}
}

func TestIntegration_IPPool_GetUsed(t *testing.T) {
	repo := NewIPPoolRepository(integrationClient(t))

	used, err := repo.GetUsed(context.Background())
	require.NoError(t, err)
	t.Logf("found %d used pool entries", len(used))
}

// ─── Firewall ─────────────────────────────────────────────────────────────────

func TestIntegration_Firewall_GetAll(t *testing.T) {
	repo := NewFirewallRepository(integrationClient(t))

	rules, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	t.Logf("found %d firewall rules:", len(rules))
	for _, r := range rules {
		t.Logf("  [%s] chain=%-10s action=%-10s disabled=%v", r.ID, r.Chain, r.Action, r.Disabled)
	}
}

func TestIntegration_Firewall_CreateToggleDelete(t *testing.T) {
	repo := NewFirewallRepository(integrationClient(t))
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, dto.CreateFirewallRuleRequest{
		Chain:   "forward",
		Action:  "passthrough",
		Comment: "integration-test-rule — safe to delete",
	}))

	rules, err := repo.GetAll(ctx)
	require.NoError(t, err)
	var id string
	for _, r := range rules {
		if r.Comment == "integration-test-rule — safe to delete" {
			id = r.ID
		}
	}
	require.NotEmpty(t, id)
	t.Logf("created firewall rule id=%s", id)

	require.NoError(t, repo.Toggle(ctx, id, true))
	rules, _ = repo.GetAll(ctx)
	for _, r := range rules {
		if r.ID == id {
			assert.True(t, r.Disabled, "rule should be disabled")
		}
	}

	require.NoError(t, repo.Toggle(ctx, id, false))
	rules, _ = repo.GetAll(ctx)
	for _, r := range rules {
		if r.ID == id {
			assert.False(t, r.Disabled, "rule should be enabled")
		}
	}

	require.NoError(t, repo.Delete(ctx, id))
}

// ─── Interface ────────────────────────────────────────────────────────────────

func TestIntegration_Interface_GetAll(t *testing.T) {
	repo := NewInterfaceRepository(integrationClient(t))

	ifaces, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, ifaces)
	t.Logf("found %d interfaces:", len(ifaces))
	for _, i := range ifaces {
		t.Logf("  %-20s type=%-12s running=%-5v disabled=%v", i.Name, i.Type, i.Running, i.Disabled)
	}
}

// ─── Hotspot ──────────────────────────────────────────────────────────────────

func TestIntegration_Hotspot_GetServers(t *testing.T) {
	repo := NewHotspotRepository(integrationClient(t))

	servers, err := repo.GetServers(context.Background())
	require.NoError(t, err)
	t.Logf("found %d hotspot servers", len(servers))
}

func TestIntegration_Hotspot_GetUsers(t *testing.T) {
	repo := NewHotspotRepository(integrationClient(t))

	users, err := repo.GetUsers(context.Background())
	require.NoError(t, err)
	t.Logf("found %d hotspot users", len(users))
}

func TestIntegration_Hotspot_GetActiveUsers(t *testing.T) {
	repo := NewHotspotRepository(integrationClient(t))

	active, err := repo.GetActiveUsers(context.Background())
	require.NoError(t, err)
	t.Logf("found %d active hotspot sessions", len(active))
}

// ─── Queue ────────────────────────────────────────────────────────────────────

func TestIntegration_Queue_GetAllSimple(t *testing.T) {
	repo := NewQueueRepository(integrationClient(t))

	queues, err := repo.GetAllSimple(context.Background())
	require.NoError(t, err)
	t.Logf("found %d simple queues:", len(queues))
	for _, q := range queues {
		t.Logf("  %-20s target=%-20s max=%s", q.Name, q.Target, q.MaxLimit)
	}
}

func TestIntegration_Queue_GetAllTree(t *testing.T) {
	repo := NewQueueRepository(integrationClient(t))

	queues, err := repo.GetAllTree(context.Background())
	require.NoError(t, err)
	t.Logf("found %d queue tree entries", len(queues))
}

func TestIntegration_Queue_CreateDeleteSimple(t *testing.T) {
	repo := NewQueueRepository(integrationClient(t))
	ctx := context.Background()

	require.NoError(t, repo.CreateSimple(ctx, dto.CreateSimpleQueueRequest{
		Name:     "test-integration-queue",
		Target:   "10.99.99.1/32",
		MaxLimit: "1M/1M",
		Comment:  "integration test — safe to delete",
	}))

	queues, err := repo.GetAllSimple(ctx)
	require.NoError(t, err)
	var id string
	for _, q := range queues {
		if q.Name == "test-integration-queue" {
			id = q.ID
		}
	}
	require.NotEmpty(t, id, "created queue must appear in list")
	t.Logf("created queue id=%s", id)

	require.NoError(t, repo.DeleteSimple(ctx, id))
}

// ─── System ───────────────────────────────────────────────────────────────────

func TestIntegration_System_GetResource(t *testing.T) {
	repo := NewSystemRepository(integrationClient(t))

	res, err := repo.GetResource(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, res.Version)
	assert.NotEmpty(t, res.BoardName)
	assert.Greater(t, res.TotalMemory, int64(0))
	t.Logf("RouterOS %s on %s | uptime: %s | CPU: %d%% | RAM: %d/%d MB",
		res.Version, res.BoardName, res.Uptime,
		res.CPULoad, res.FreeMemory/1024/1024, res.TotalMemory/1024/1024)
}

func TestIntegration_System_GetIdentity(t *testing.T) {
	repo := NewSystemRepository(integrationClient(t))

	id, err := repo.GetIdentity(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, id.Name)
	t.Logf("router identity: %s", id.Name)
}

func TestIntegration_System_GetLogs(t *testing.T) {
	repo := NewSystemRepository(integrationClient(t))

	logs, err := repo.GetLogs(context.Background(), dto.GetLogsRequest{Limit: 10})
	require.NoError(t, err)
	t.Logf("retrieved %d log entries:", len(logs))
	for _, l := range logs {
		t.Logf("  [%s] %-20s %s", l.Time, l.Topics, l.Message)
	}
}

// ─── Listener / Traffic Monitor ───────────────────────────────────────────────

func TestIntegration_Listener_TrafficOnce(t *testing.T) {
	c := integrationClient(t)
	ifaceName := firstRunningInterface(t, c)
	t.Logf("monitoring interface: %s", ifaceName)

	// =once= asks RouterOS to send one sample then close the stream.
	lr, err := c.ListenArgs([]string{
		"/interface/monitor-traffic",
		"=interface=" + ifaceName,
		"=once=",
	})
	require.NoError(t, err)

	received := 0
	timeout := time.After(8 * time.Second)
loop:
	for {
		select {
		case sen, ok := <-lr.Chan():
			if !ok {
				break loop
			}
			rxBps := sen.Map["rx-bits-per-second"]
			txBps := sen.Map["tx-bits-per-second"]
			t.Logf("  sample %d: rx=%s bps  tx=%s bps", received+1, rxBps, txBps)
			received++
		case <-timeout:
			break loop
		}
	}

	_, err = lr.Cancel()
	assert.NoError(t, err)
	assert.Greater(t, received, 0, "must receive at least one traffic sample")
}

func TestIntegration_Listener_ContinuousWithCancel(t *testing.T) {
	c := integrationClient(t)
	ifaceName := firstRunningInterface(t, c)

	lr, err := c.ListenArgs([]string{
		"/interface/monitor-traffic",
		"=interface=" + ifaceName,
	})
	require.NoError(t, err)

	// Collect 3 samples then cancel
	received := 0
	timeout := time.After(10 * time.Second)
collect:
	for {
		select {
		case sen, ok := <-lr.Chan():
			if !ok {
				break collect
			}
			t.Logf("  sample %d rx=%s tx=%s",
				received+1,
				sen.Map["rx-bits-per-second"],
				sen.Map["tx-bits-per-second"],
			)
			received++
			if received >= 3 {
				break collect
			}
		case <-timeout:
			break collect
		}
	}

	_, err = lr.Cancel()
	assert.NoError(t, err, "Cancel should succeed")
	assert.GreaterOrEqual(t, received, 1, "must receive at least 1 sample before cancel")
}

func TestIntegration_Listener_CancelContext(t *testing.T) {
	c := integrationClient(t)
	ifaceName := firstRunningInterface(t, c)

	lr, err := c.ListenArgs([]string{
		"/interface/monitor-traffic",
		"=interface=" + ifaceName,
	})
	require.NoError(t, err)

	// Wait for first sample
	select {
	case _, ok := <-lr.Chan():
		assert.True(t, ok, "first sentence should arrive")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: no traffic sentence received")
	}

	// Cancel with context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = lr.CancelContext(ctx)
	assert.NoError(t, err)
}

func TestIntegration_Listener_RunAndListenConcurrent(t *testing.T) {
	c := integrationClient(t)
	ifaceName := firstRunningInterface(t, c)

	// Start streaming while also firing batch commands — proves async mode works
	lr, err := c.ListenArgs([]string{
		"/interface/monitor-traffic",
		"=interface=" + ifaceName,
	})
	require.NoError(t, err)
	defer lr.Cancel() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	replies, errs := c.RunMany(ctx, [][]string{
		{"/system/resource/print"},
		{"/system/identity/print"},
		{"/ip/pool/print"},
	})
	for i, err := range errs {
		require.NoError(t, err, "RunMany command %d should succeed while listener active", i)
	}
	assert.NotEmpty(t, replies[0].Re)
	assert.NotEmpty(t, replies[1].Re)
	t.Log("RunMany succeeded while ListenArgs streaming — async confirmed")

	// Also verify we get a traffic sample
	select {
	case sen, ok := <-lr.Chan():
		require.True(t, ok)
		t.Logf("concurrent traffic sample: rx=%s tx=%s",
			sen.Map["rx-bits-per-second"], sen.Map["tx-bits-per-second"])
	case <-time.After(5 * time.Second):
		t.Fatal("no traffic sample received during concurrent test")
	}
}

func TestIntegration_Listener_TrafficMonitor_ViaRepo(t *testing.T) {
	c := integrationClient(t)
	ifaceName := firstRunningInterface(t, c)
	repo := NewInterfaceRepository(c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch := make(chan entity.TrafficStat, 8)
	err := repo.StartTrafficMonitor(ctx, ifaceName, ch)
	require.NoError(t, err)
	defer repo.StopTrafficMonitor(context.Background(), ifaceName) //nolint:errcheck

	// Collect up to 3 samples
	received := 0
	timeout := time.After(8 * time.Second)
loop:
	for {
		select {
		case stat := <-ch:
			t.Logf("  stat %d: iface=%s rx=%d bps tx=%d bps",
				received+1, stat.Interface, stat.RxBitsPerSecond, stat.TxBitsPerSecond)
			assert.Equal(t, ifaceName, stat.Interface)
			assert.False(t, stat.Timestamp.IsZero())
			received++
			if received >= 3 {
				break loop
			}
		case <-timeout:
			break loop
		case <-ctx.Done():
			break loop
		}
	}

	assert.Greater(t, received, 0, "must receive at least one stat from TrafficMonitor")
	t.Logf("received %d traffic stats from interface %s", received, ifaceName)
}
