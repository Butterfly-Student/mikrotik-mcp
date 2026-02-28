package repository

import (
	"context"

	"mikrotik-mcp/domain/entity"
)

type InterfaceRepository interface {
	GetAll(ctx context.Context) ([]entity.NetworkInterface, error)
	StartTrafficMonitor(ctx context.Context, iface string, ch chan<- entity.TrafficStat) error
	StopTrafficMonitor(ctx context.Context, iface string) error
}
