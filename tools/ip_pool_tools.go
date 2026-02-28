package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/internal/usecase"
)

func RegisterIPPoolTools(s *server.MCPServer, uc *usecase.IPPoolUseCase, readOnly bool) {
	// list_ip_pools
	s.AddTool(
		mcp.NewTool("list_ip_pools",
			mcp.WithDescription("Menampilkan semua IP pool yang ada di MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListPools(ctx)
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

	// list_ip_pool_used
	s.AddTool(
		mcp.NewTool("list_ip_pool_used",
			mcp.WithDescription("Menampilkan IP address yang sedang digunakan dari pool MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListUsed(ctx)
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

	// add_ip_pool
	s.AddTool(
		mcp.NewTool("add_ip_pool",
			mcp.WithDescription("Menambahkan IP pool baru ke MikroTik"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Nama IP pool"),
			),
			mcp.WithString("ranges",
				mcp.Required(),
				mcp.Description("Range IP, contoh: 192.168.1.100-192.168.1.200"),
			),
			mcp.WithString("next_pool",
				mcp.Description("Nama pool berikutnya jika pool ini habis (opsional)"),
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
			ranges := req.GetString("ranges", "")
			if name == "" || ranges == "" {
				return mcp.NewToolResultError("name and ranges are required"), nil
			}
			err := uc.CreatePool(ctx, dto.CreateIPPoolRequest{
				Name:     name,
				Ranges:   ranges,
				NextPool: req.GetString("next_pool", ""),
				Comment:  req.GetString("comment", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("IP pool '%s' berhasil dibuat", name)), nil
		},
	)

	// update_ip_pool
	s.AddTool(
		mcp.NewTool("update_ip_pool",
			mcp.WithDescription("Mengubah ranges atau comment IP pool yang sudah ada"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID pool (contoh: *1)"),
			),
			mcp.WithString("ranges",
				mcp.Description("Range IP baru (opsional)"),
			),
			mcp.WithString("next_pool",
				mcp.Description("Nama pool berikutnya (opsional)"),
			),
			mcp.WithString("comment",
				mcp.Description("Keterangan baru (opsional)"),
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
			err := uc.UpdatePool(ctx, dto.UpdateIPPoolRequest{
				ID:       id,
				Ranges:   req.GetString("ranges", ""),
				NextPool: req.GetString("next_pool", ""),
				Comment:  req.GetString("comment", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("IP pool '%s' berhasil diupdate", id)), nil
		},
	)

	// delete_ip_pool
	s.AddTool(
		mcp.NewTool("delete_ip_pool",
			mcp.WithDescription("Menghapus IP pool dari MikroTik"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID pool yang akan dihapus (contoh: *1)"),
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
			err := uc.DeletePool(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("IP pool '%s' berhasil dihapus", id)), nil
		},
	)
}
