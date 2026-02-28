package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/internal/usecase"
)

func RegisterHotspotTools(s *server.MCPServer, uc *usecase.HotspotUseCase, readOnly bool) {
	// list_hotspot_servers
	s.AddTool(
		mcp.NewTool("list_hotspot_servers",
			mcp.WithDescription("Menampilkan semua hotspot server yang dikonfigurasi di MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListServers(ctx)
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

	// list_hotspot_users
	s.AddTool(
		mcp.NewTool("list_hotspot_users",
			mcp.WithDescription("Menampilkan semua user hotspot yang terdaftar di MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListUsers(ctx)
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

	// list_hotspot_active
	s.AddTool(
		mcp.NewTool("list_hotspot_active",
			mcp.WithDescription("Menampilkan sesi hotspot yang sedang aktif/online saat ini"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListActiveUsers(ctx)
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

	// add_hotspot_user
	s.AddTool(
		mcp.NewTool("add_hotspot_user",
			mcp.WithDescription("Menambahkan user hotspot baru ke MikroTik"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Username untuk hotspot"),
			),
			mcp.WithString("password",
				mcp.Required(),
				mcp.Description("Password user"),
			),
			mcp.WithString("server",
				mcp.Description("Nama hotspot server (default: all)"),
			),
			mcp.WithString("profile",
				mcp.Description("Nama user profile (default: default)"),
			),
			mcp.WithString("mac_address",
				mcp.Description("Binding MAC address opsional, contoh: AA:BB:CC:DD:EE:FF"),
			),
			mcp.WithString("limit_uptime",
				mcp.Description("Batas total waktu online, contoh: 1d, 8h"),
			),
			mcp.WithNumber("limit_bytes_total",
				mcp.Description("Batas total traffic dalam bytes (0 = unlimited)"),
			),
			mcp.WithString("comment",
				mcp.Description("Keterangan opsional"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if readOnly {
				return mcp.NewToolResultError("operation not permitted in read-only mode"), nil
			}
			name := req.GetString("name", "")
			password := req.GetString("password", "")
			if name == "" || password == "" {
				return mcp.NewToolResultError("name and password are required"), nil
			}
			limitBytesTotal := int64(req.GetInt("limit_bytes_total", 0))
			err := uc.AddUser(ctx, dto.CreateHotspotUserRequest{
				Name:            name,
				Password:        password,
				Server:          req.GetString("server", ""),
				Profile:         req.GetString("profile", ""),
				MacAddress:      req.GetString("mac_address", ""),
				LimitUptime:     req.GetString("limit_uptime", ""),
				LimitBytesTotal: limitBytesTotal,
				Comment:         req.GetString("comment", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Hotspot user '%s' berhasil ditambahkan", name)), nil
		},
	)

	// delete_hotspot_user
	s.AddTool(
		mcp.NewTool("delete_hotspot_user",
			mcp.WithDescription("Menghapus user hotspot dari MikroTik"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID user hotspot (contoh: *1)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if readOnly {
				return mcp.NewToolResultError("operation not permitted in read-only mode"), nil
			}
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			err := uc.DeleteUser(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Hotspot user '%s' berhasil dihapus", id)), nil
		},
	)

	// kick_hotspot_user
	s.AddTool(
		mcp.NewTool("kick_hotspot_user",
			mcp.WithDescription("Memaksa logout / kick user dari sesi hotspot aktif"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID sesi aktif (dari list_hotspot_active, contoh: *A1)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if readOnly {
				return mcp.NewToolResultError("operation not permitted in read-only mode"), nil
			}
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			err := uc.KickActiveUser(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("User dengan sesi '%s' berhasil di-kick", id)), nil
		},
	)
}
