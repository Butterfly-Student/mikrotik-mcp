package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/internal/usecase"
)

func RegisterQueueTools(s *server.MCPServer, uc *usecase.QueueUseCase, readOnly bool) {
	// list_queues (simple + tree)
	s.AddTool(
		mcp.NewTool("list_queues",
			mcp.WithDescription("Menampilkan semua simple queue dan tree queue di MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			simple, err := uc.ListSimpleQueues(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			tree, err := uc.ListTreeQueues(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result := map[string]any{
				"simple_queues": simple,
				"tree_queues":   tree,
			}
			out, err := mcp.NewToolResultJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return out, nil
		},
	)

	// list_simple_queues
	s.AddTool(
		mcp.NewTool("list_simple_queues",
			mcp.WithDescription("Menampilkan semua simple queue di MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListSimpleQueues(ctx)
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

	// add_queue (simple queue)
	s.AddTool(
		mcp.NewTool("add_queue",
			mcp.WithDescription("Menambahkan simple queue baru untuk membatasi bandwidth client. Format max_limit: upload/download contoh: 1M/5M"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Nama queue"),
			),
			mcp.WithString("target",
				mcp.Required(),
				mcp.Description("IP target, contoh: 192.168.1.100/32 atau 192.168.1.0/24"),
			),
			mcp.WithString("max_limit",
				mcp.Description("Batas maksimal bandwidth format upload/download, contoh: 1M/5M atau 512k/2M"),
			),
			mcp.WithString("limit_at",
				mcp.Description("Guaranteed rate format upload/download, contoh: 256k/512k"),
			),
			mcp.WithString("burst_limit",
				mcp.Description("Burst max rate format upload/download, contoh: 2M/10M"),
			),
			mcp.WithString("burst_threshold",
				mcp.Description("Threshold on/off burst, contoh: 512k/2M"),
			),
			mcp.WithString("burst_time",
				mcp.Description("Burst averaging period, contoh: 8s/8s"),
			),
			mcp.WithString("parent",
				mcp.Description("Nama parent queue (untuk HTB)"),
			),
			mcp.WithNumber("priority",
				mcp.Description("Prioritas 1-8 (1=tertinggi)"),
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
			target := req.GetString("target", "")
			if name == "" || target == "" {
				return mcp.NewToolResultError("name and target are required"), nil
			}
			err := uc.AddSimpleQueue(ctx, dto.CreateSimpleQueueRequest{
				Name:           name,
				Target:         target,
				MaxLimit:       req.GetString("max_limit", ""),
				LimitAt:        req.GetString("limit_at", ""),
				BurstLimit:     req.GetString("burst_limit", ""),
				BurstThreshold: req.GetString("burst_threshold", ""),
				BurstTime:      req.GetString("burst_time", ""),
				Parent:         req.GetString("parent", ""),
				Priority:       req.GetInt("priority", 0),
				Comment:        req.GetString("comment", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Simple queue '%s' untuk target '%s' berhasil dibuat", name, target)), nil
		},
	)

	// add_queue_tree
	s.AddTool(
		mcp.NewTool("add_queue_tree",
			mcp.WithDescription("Menambahkan queue tree baru (untuk traffic shaping berbasis packet mark)"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Nama queue tree"),
			),
			mcp.WithString("parent",
				mcp.Required(),
				mcp.Description("Parent: global, interface (contoh: ether1), atau nama queue parent"),
			),
			mcp.WithString("packet_mark",
				mcp.Description("Packet mark dari /ip/firewall/mangle"),
			),
			mcp.WithString("max_limit",
				mcp.Description("Batas maksimal bandwidth, contoh: 2M"),
			),
			mcp.WithString("limit_at",
				mcp.Description("Guaranteed rate"),
			),
			mcp.WithNumber("priority",
				mcp.Description("Prioritas 1-8"),
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
			parent := req.GetString("parent", "")
			if name == "" || parent == "" {
				return mcp.NewToolResultError("name and parent are required"), nil
			}
			err := uc.AddTreeQueue(ctx, dto.CreateQueueTreeRequest{
				Name:       name,
				Parent:     parent,
				PacketMark: req.GetString("packet_mark", ""),
				MaxLimit:   req.GetString("max_limit", ""),
				LimitAt:    req.GetString("limit_at", ""),
				Priority:   req.GetInt("priority", 0),
				Comment:    req.GetString("comment", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Queue tree '%s' berhasil dibuat", name)), nil
		},
	)

	// delete_queue
	s.AddTool(
		mcp.NewTool("delete_queue",
			mcp.WithDescription("Menghapus simple queue dari MikroTik"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID simple queue (contoh: *1)"),
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
			err := uc.DeleteSimpleQueue(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Simple queue '%s' berhasil dihapus", id)), nil
		},
	)
}
