package tools

import (
	"github.com/mark3labs/mcp-go/server"
	"mikrotik-mcp/internal/usecase"
)

type Dependencies struct {
	IPPool    *usecase.IPPoolUseCase
	Firewall  *usecase.FirewallUseCase
	Interface *usecase.InterfaceUseCase
	Hotspot   *usecase.HotspotUseCase
	Queue     *usecase.QueueUseCase
	System    *usecase.SystemUseCase
	ReadOnly  bool
}

func RegisterAll(s *server.MCPServer, deps Dependencies) {
	RegisterIPPoolTools(s, deps.IPPool, deps.ReadOnly)
	RegisterFirewallTools(s, deps.Firewall, deps.ReadOnly)
	RegisterInterfaceTools(s, deps.Interface)
	RegisterHotspotTools(s, deps.Hotspot, deps.ReadOnly)
	RegisterQueueTools(s, deps.Queue, deps.ReadOnly)
	RegisterSystemTools(s, deps.System, deps.ReadOnly)
}
