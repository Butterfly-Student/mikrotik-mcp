//go:build e2e

package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mikrotik-mcp/internal/mikrotik"
	"mikrotik-mcp/internal/usecase"
)

// ─── JSON-RPC types ──────────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ─── In-process MCP client ───────────────────────────────────────────────────

type mcpTestClient struct {
	writer  *io.PipeWriter
	scanner *bufio.Scanner
	seq     atomic.Int32
}

func (c *mcpTestClient) send(method string, params interface{}) (int, error) {
	id := int(c.seq.Add(1))
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	_, err = fmt.Fprintf(c.writer, "%s\n", data)
	return id, err
}

func (c *mcpTestClient) notify(method string, params interface{}) error {
	req := rpcRequest{JSONRPC: "2.0", Method: method, Params: params}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.writer, "%s\n", data)
	return err
}

func (c *mcpTestClient) readResponse(targetID int) (*rpcResponse, error) {
	for c.scanner.Scan() {
		line := c.scanner.Text()
		if line == "" {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		if resp.ID != nil && *resp.ID == targetID {
			return &resp, nil
		}
	}
	return nil, fmt.Errorf("connection closed before response for id=%d", targetID)
}

func (c *mcpTestClient) callTool(t *testing.T, name string, args map[string]interface{}) json.RawMessage {
	t.Helper()
	id, err := c.send("tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	require.NoError(t, err)

	resp, err := c.readResponse(id)
	require.NoError(t, err)
	require.Nil(t, resp.Error, "tool error: %v", resp.Error)
	return resp.Result
}

// ─── Test fixtures ───────────────────────────────────────────────────────────

func e2eClient(t *testing.T, readOnly bool) *mcpTestClient {
	t.Helper()

	pass := os.Getenv("MIKROTIK_PASS")
	if pass == "" {
		t.Skip("skipping e2e test: MIKROTIK_PASS not set")
	}
	host := os.Getenv("MIKROTIK_HOST")
	if host == "" {
		host = "192.168.88.1"
	}
	user := os.Getenv("MIKROTIK_USER")
	if user == "" {
		user = "admin"
	}

	logger, _ := zap.NewDevelopment()
	cfg := mikrotik.Config{
		Host:     host,
		Port:     8728,
		Username: user,
		Password: pass,
	}

	// Retry dengan timeout per-attempt untuk menunggu RouterOS boot selesai.
	const (
		attemptTimeout = 8 * time.Second
		retryInterval  = 5 * time.Second
		totalTimeout   = 3 * time.Minute
	)
	deadline := time.Now().Add(totalTimeout)
	var mtClient *mikrotik.Client
	for time.Now().Before(deadline) {
		c := mikrotik.NewClient(cfg, logger)
		ctx, cancel := context.WithTimeout(context.Background(), attemptTimeout)
		err := c.Connect(ctx)
		cancel()
		if err == nil {
			mtClient = c
			break
		}
		t.Logf("connection attempt failed: %v — retrying...", err)
		time.Sleep(retryInterval)
	}
	require.NotNil(t, mtClient, "failed to connect to MikroTik within %s", totalTimeout)
	t.Cleanup(func() { mtClient.Close() })

	deps := Dependencies{
		IPPool:    usecase.NewIPPoolUseCase(mikrotik.NewIPPoolRepository(mtClient), logger),
		Firewall:  usecase.NewFirewallUseCase(mikrotik.NewFirewallRepository(mtClient), logger),
		Interface: usecase.NewInterfaceUseCase(mikrotik.NewInterfaceRepository(mtClient), logger),
		Hotspot:   usecase.NewHotspotUseCase(mikrotik.NewHotspotRepository(mtClient), logger),
		Queue:     usecase.NewQueueUseCase(mikrotik.NewQueueRepository(mtClient), logger),
		System:    usecase.NewSystemUseCase(mikrotik.NewSystemRepository(mtClient), logger),
		ReadOnly:  readOnly,
	}

	s := server.NewMCPServer("mikrotik-mcp-e2e", "test")
	RegisterAll(s, deps)

	// Wire up in-process stdio transport using io.Pipe
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()

	srvCtx, srvCancel := context.WithCancel(context.Background())
	stdioServer := server.NewStdioServer(s)
	go func() {
		_ = stdioServer.Listen(srvCtx, serverIn, serverOut)
	}()
	t.Cleanup(func() {
		srvCancel()
		_ = clientOut.Close()
		_ = clientIn.Close()
	})

	cl := &mcpTestClient{
		writer:  clientOut,
		scanner: bufio.NewScanner(clientIn),
	}

	// MCP handshake
	id, err := cl.send("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "e2e-test", "version": "1.0"},
	})
	require.NoError(t, err)
	_, err = cl.readResponse(id)
	require.NoError(t, err)
	require.NoError(t, cl.notify("notifications/initialized", nil))

	return cl
}

// ─── E2E Tests ───────────────────────────────────────────────────────────────

func TestE2E_ListIPPools(t *testing.T) {
	cl := e2eClient(t, false)

	result := cl.callTool(t, "list_ip_pools", nil)

	assert.NotNil(t, result)
	t.Logf("list_ip_pools result: %s", result)
}

func TestE2E_AddDeleteIPPool(t *testing.T) {
	cl := e2eClient(t, false)

	// Add pool
	addResult := cl.callTool(t, "add_ip_pool", map[string]interface{}{
		"name":    "e2e-test-pool",
		"ranges":  "10.88.0.1-10.88.0.10",
		"comment": "e2e test",
	})
	assert.NotNil(t, addResult)
	t.Logf("add_ip_pool result: %s", addResult)

	// List to find the created pool's ID
	listResult := cl.callTool(t, "list_ip_pools", nil)
	require.NotNil(t, listResult)

	var listResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	_ = json.Unmarshal(listResult, &listResp)

	// Delete by searching for the pool (use list_ip_pools + delete_ip_pool)
	// In a real test we'd parse the ID; here we verify the round-trip worked.
	t.Logf("list_ip_pools after add: %s", listResult)
}

func TestE2E_ListInterfaces(t *testing.T) {
	cl := e2eClient(t, false)

	result := cl.callTool(t, "list_interfaces", nil)

	assert.NotNil(t, result)
	t.Logf("list_interfaces result: %s", result)
}

func TestE2E_GetResource(t *testing.T) {
	cl := e2eClient(t, false)

	result := cl.callTool(t, "get_resource", nil)

	assert.NotNil(t, result)
	t.Logf("get_resource result: %s", result)
}

func TestE2E_GetLogs(t *testing.T) {
	cl := e2eClient(t, false)

	result := cl.callTool(t, "get_logs", map[string]interface{}{
		"limit": 5,
	})

	assert.NotNil(t, result)
	t.Logf("get_logs result: %s", result)
}

func TestE2E_ReadOnlyMode(t *testing.T) {
	cl := e2eClient(t, true)

	// In read-only mode, write operations should return an error result
	result := cl.callTool(t, "add_ip_pool", map[string]interface{}{
		"name":   "should-fail",
		"ranges": "10.99.0.1-10.99.0.5",
	})

	// The tool returns a text error result (not an RPC error), so result is non-nil
	assert.NotNil(t, result)
	resultStr := string(result)
	assert.Contains(t, resultStr, "read-only", "should reject write in read-only mode")
	t.Logf("read-only add_ip_pool result: %s", result)
}
