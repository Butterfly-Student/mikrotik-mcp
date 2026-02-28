package mcpclient

import (
	"context"
	"fmt"
	"time"

	mcpClient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// MCPCaller adalah interface yang digunakan bridge untuk memanggil MCP server.
// Memisahkan interface dari implementasi konkrit agar mudah di-mock dalam testing.
type MCPCaller interface {
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallResult, error)
}

type Client struct {
	c         *mcpClient.Client
	cancelSSE context.CancelFunc
	logger    *zap.Logger
}

func NewClient(serverURL string, logger *zap.Logger) (*Client, error) {
	c, err := mcpClient.NewSSEMCPClient(serverURL + "/sse")
	if err != nil {
		return nil, fmt.Errorf("create MCP SSE client: %w", err)
	}

	// sseCtx hidup selama Client hidup — jangan di-cancel sampai Close() dipanggil.
	// Ini yang menjaga SSE stream HTTP connection tetap terbuka.
	sseCtx, sseCancel := context.WithCancel(context.Background())

	if err := c.Start(sseCtx); err != nil {
		sseCancel()
		return nil, fmt.Errorf("start MCP client: %w", err)
	}

	// Initialize hanya perlu timeout pendek — ini request-response biasa.
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	if _, err := c.Initialize(initCtx, mcp.InitializeRequest{}); err != nil {
		sseCancel()
		return nil, fmt.Errorf("MCP initialize handshake: %w", err)
	}

	logger.Info("connected to MCP server", zap.String("url", serverURL))
	return &Client{c: c, cancelSSE: sseCancel, logger: logger}, nil
}

// ListTools mengambil semua tool yang tersedia dari MCP server
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	result, err := c.c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list MCP tools: %w", err)
	}

	tools := make([]Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		schema := map[string]interface{}{}
		if t.InputSchema.Type != "" {
			schema["type"] = t.InputSchema.Type
		}
		if t.InputSchema.Properties != nil {
			schema["properties"] = t.InputSchema.Properties
		}
		if len(t.InputSchema.Required) > 0 {
			schema["required"] = t.InputSchema.Required
		}
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	return tools, nil
}

// CallTool memanggil satu tool di MCP server
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := c.c.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("call MCP tool %s: %w", name, err)
	}

	cr := &CallResult{IsError: result.IsError}
	for _, block := range result.Content {
		if textContent, ok := block.(mcp.TextContent); ok {
			cr.Content = append(cr.Content, ContentBlock{Type: "text", Text: textContent.Text})
		}
	}
	return cr, nil
}

func (c *Client) Close() {
	c.cancelSSE()
	c.c.Close()
}
