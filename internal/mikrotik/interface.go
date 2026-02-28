package mikrotik

import (
	"context"
	"fmt"
	"strconv"

	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type interfaceRepository struct {
	client  *Client
	monitor *listenerManager
}

func NewInterfaceRepository(client *Client) repository.InterfaceRepository {
	return &interfaceRepository{
		client:  client,
		monitor: newListenerManager(client),
	}
}

func (r *interfaceRepository) GetAll(ctx context.Context) ([]entity.NetworkInterface, error) {
	reply, err := r.client.Run("/interface/print")
	if err != nil {
		return nil, fmt.Errorf("interface print: %w", err)
	}

	ifaces := make([]entity.NetworkInterface, 0, len(reply.Re))
	for _, s := range reply.Re {
		mtu, _ := strconv.Atoi(s.Map["mtu"])
		linkDowns, _ := strconv.Atoi(s.Map["link-downs"])

		ifaces = append(ifaces, entity.NetworkInterface{
			ID:             s.Map[".id"],
			Name:           s.Map["name"],
			Type:           s.Map["type"],
			MacAddress:     s.Map["mac-address"],
			MTU:            mtu,
			Running:        s.Map["running"] == "true",
			Disabled:       s.Map["disabled"] == "true",
			Dynamic:        s.Map["dynamic"] == "true",
			Slave:          s.Map["slave"] == "true",
			Comment:        s.Map["comment"],
			LastLinkUpTime: s.Map["last-link-up-time"],
			LinkDowns:      linkDowns,
		})
	}
	return ifaces, nil
}

func (r *interfaceRepository) StartTrafficMonitor(ctx context.Context, iface string, ch chan<- entity.TrafficStat) error {
	return r.monitor.StartTrafficMonitor(ctx, iface, ch)
}

func (r *interfaceRepository) StopTrafficMonitor(ctx context.Context, iface string) error {
	return r.monitor.StopTrafficMonitor(ctx, iface)
}
