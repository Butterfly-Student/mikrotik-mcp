package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/internal/usecase"
)

func RegisterInterfaceTools(s *server.MCPServer, uc *usecase.InterfaceUseCase) {
	// list_interfaces
	s.AddTool(
		mcp.NewTool("list_interfaces",
			mcp.WithDescription("Menampilkan semua network interface di MikroTik beserta status running/disabled"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListInterfaces(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out, err := mcp.NewToolResultJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return out, nil
		},
	)

	// watch_traffic
	s.AddTool(
		mcp.NewTool("watch_traffic",
			mcp.WithDescription("Memonitor traffic realtime pada interface tertentu selama N detik. Mengembalikan data rx/tx bps per detik."),
			mcp.WithString("interface",
				mcp.Required(),
				mcp.Description("Nama interface yang dimonitor, contoh: ether1, atau 'all' untuk semua"),
			),
			mcp.WithNumber("seconds",
				mcp.Required(),
				mcp.Description("Durasi monitoring dalam detik (1-60)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			iface := req.GetString("interface", "")
			seconds := req.GetInt("seconds", 5)
			if iface == "" {
				return mcp.NewToolResultError("interface is required"), nil
			}
			if seconds < 1 || seconds > 60 {
				return mcp.NewToolResultError("seconds harus antara 1 dan 60"), nil
			}

			result, err := uc.WatchTraffic(ctx, dto.WatchTrafficRequest{
				Interface: iface,
				Seconds:   seconds,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out, err := mcp.NewToolResultJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return out, nil
		},
	)
}
