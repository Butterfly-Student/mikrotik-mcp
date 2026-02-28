package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/internal/usecase"
)

func RegisterSystemTools(s *server.MCPServer, uc *usecase.SystemUseCase, readOnly bool) {
	// get_resource
	s.AddTool(
		mcp.NewTool("get_resource",
			mcp.WithDescription("Mendapatkan informasi resource sistem MikroTik: CPU load, RAM, storage, uptime, versi RouterOS"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.GetResource(ctx)
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

	// get_identity
	s.AddTool(
		mcp.NewTool("get_identity",
			mcp.WithDescription("Mendapatkan nama/hostname router MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.GetIdentity(ctx)
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

	// get_logs
	s.AddTool(
		mcp.NewTool("get_logs",
			mcp.WithDescription("Mengambil log sistem dari MikroTik. Bisa difilter berdasarkan topik."),
			mcp.WithString("topics",
				mcp.Description("Filter topik log, contoh: firewall, dhcp, hotspot, system, warning, error (opsional)"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Jumlah maksimal log yang dikembalikan (default: 50, max: 500)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			limit := req.GetInt("limit", 50)
			if limit > 500 {
				limit = 500
			}
			result, err := uc.GetLogs(ctx, dto.GetLogsRequest{
				Topics: req.GetString("topics", ""),
				Limit:  limit,
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

	// reboot_router
	s.AddTool(
		mcp.NewTool("reboot_router",
			mcp.WithDescription("Mereboot MikroTik router. PERINGATAN: Router akan tidak dapat diakses selama beberapa menit. WAJIB sertakan confirm=true"),
			mcp.WithBoolean("confirm",
				mcp.Required(),
				mcp.Description("Harus true untuk mengkonfirmasi reboot router"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if readOnly {
				return mcp.NewToolResultError("operation not permitted in read-only mode"), nil
			}
			if !req.GetBool("confirm", false) {
				return mcp.NewToolResultError("Reboot dibatalkan. Sertakan confirm=true untuk melanjutkan"), nil
			}
			if err := uc.Reboot(ctx); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText("Router sedang reboot... Koneksi akan terputus sementara."), nil
		},
	)
}
