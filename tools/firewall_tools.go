package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/internal/usecase"
)

func RegisterFirewallTools(s *server.MCPServer, uc *usecase.FirewallUseCase, readOnly bool) {
	// list_firewall_rules
	s.AddTool(
		mcp.NewTool("list_firewall_rules",
			mcp.WithDescription("Menampilkan semua firewall filter rules di MikroTik"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := uc.ListRules(ctx)
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

	// add_firewall_rule
	s.AddTool(
		mcp.NewTool("add_firewall_rule",
			mcp.WithDescription("Menambahkan firewall filter rule baru ke MikroTik"),
			mcp.WithString("chain",
				mcp.Required(),
				mcp.Description("Chain: input, forward, atau output"),
			),
			mcp.WithString("action",
				mcp.Required(),
				mcp.Description("Action: accept, drop, reject, jump, log, passthrough, fasttrack-connection, tarpit"),
			),
			mcp.WithString("src_address",
				mcp.Description("IP/subnet sumber, contoh: 192.168.1.0/24"),
			),
			mcp.WithString("dst_address",
				mcp.Description("IP/subnet tujuan"),
			),
			mcp.WithString("src_address_list",
				mcp.Description("Nama address-list sumber"),
			),
			mcp.WithString("dst_address_list",
				mcp.Description("Nama address-list tujuan"),
			),
			mcp.WithString("protocol",
				mcp.Description("Protokol: tcp, udp, icmp, dll."),
			),
			mcp.WithString("src_port",
				mcp.Description("Port sumber, contoh: 80,443 atau 1000-2000"),
			),
			mcp.WithString("dst_port",
				mcp.Description("Port tujuan, contoh: 80,443"),
			),
			mcp.WithString("in_interface",
				mcp.Description("Interface asal paket masuk, contoh: ether1"),
			),
			mcp.WithString("out_interface",
				mcp.Description("Interface tujuan keluar"),
			),
			mcp.WithString("connection_state",
				mcp.Description("State koneksi: new, established, related, invalid"),
			),
			mcp.WithString("comment",
				mcp.Description("Keterangan rule"),
			),
			mcp.WithString("place_before",
				mcp.Description("Posisi insert sebelum rule ID tertentu (opsional)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if readOnly {
				return mcp.NewToolResultError("operation not permitted in read-only mode"), nil
			}
			chain := req.GetString("chain", "")
			action := req.GetString("action", "")
			if chain == "" || action == "" {
				return mcp.NewToolResultError("chain and action are required"), nil
			}
			err := uc.CreateRule(ctx, dto.CreateFirewallRuleRequest{
				Chain:           chain,
				Action:          action,
				SrcAddress:      req.GetString("src_address", ""),
				DstAddress:      req.GetString("dst_address", ""),
				SrcAddressList:  req.GetString("src_address_list", ""),
				DstAddressList:  req.GetString("dst_address_list", ""),
				Protocol:        req.GetString("protocol", ""),
				SrcPort:         req.GetString("src_port", ""),
				DstPort:         req.GetString("dst_port", ""),
				InInterface:     req.GetString("in_interface", ""),
				OutInterface:    req.GetString("out_interface", ""),
				ConnectionState: req.GetString("connection_state", ""),
				Comment:         req.GetString("comment", ""),
				PlaceBefore:     req.GetString("place_before", ""),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Firewall rule chain=%s action=%s berhasil dibuat", chain, action)), nil
		},
	)

	// delete_firewall_rule
	s.AddTool(
		mcp.NewTool("delete_firewall_rule",
			mcp.WithDescription("Menghapus firewall rule berdasarkan ID. WAJIB sertakan confirm=true"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID rule yang akan dihapus (contoh: *5)"),
			),
			mcp.WithBoolean("confirm",
				mcp.Required(),
				mcp.Description("Harus true untuk konfirmasi penghapusan"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if readOnly {
				return mcp.NewToolResultError("operation not permitted in read-only mode"), nil
			}
			if !req.GetBool("confirm", false) {
				return mcp.NewToolResultError("Penghapusan dibatalkan. Sertakan confirm=true untuk melanjutkan"), nil
			}
			id := req.GetString("id", "")
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			err := uc.DeleteRule(ctx, id)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Firewall rule '%s' berhasil dihapus", id)), nil
		},
	)

	// toggle_firewall_rule
	s.AddTool(
		mcp.NewTool("toggle_firewall_rule",
			mcp.WithDescription("Mengaktifkan atau menonaktifkan firewall rule"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID rule (contoh: *5)"),
			),
			mcp.WithBoolean("disabled",
				mcp.Required(),
				mcp.Description("true = nonaktifkan, false = aktifkan"),
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
			disabled := req.GetBool("disabled", false)
			err := uc.ToggleRule(ctx, id, disabled)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			status := "diaktifkan"
			if disabled {
				status = "dinonaktifkan"
			}
			return mcp.NewToolResultText(fmt.Sprintf("Firewall rule '%s' berhasil %s", id, status)), nil
		},
	)
}
