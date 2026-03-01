package mikrotik

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type trafficMonitor struct {
	cancel context.CancelFunc
}

type interfaceRepository struct {
	client   *Client
	monitors map[string]*trafficMonitor
	mu       sync.Mutex
}

func NewInterfaceRepository(client *Client) repository.InterfaceRepository {
	return &interfaceRepository{
		client:   client,
		monitors: make(map[string]*trafficMonitor),
	}
}

func (r *interfaceRepository) GetAll(ctx context.Context) ([]entity.NetworkInterface, error) {
	reply, err := r.client.RunContext(ctx, "/interface/print")
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
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.monitors[iface]; exists {
		return fmt.Errorf("traffic monitor already running for interface %s", iface)
	}

	listen, err := r.client.ListenArgs([]string{
		"/interface/monitor-traffic",
		"=interface=" + iface,
	})
	if err != nil {
		return fmt.Errorf("failed to start traffic monitor for %s: %w", iface, err)
	}

	monCtx, cancel := context.WithCancel(ctx)
	r.monitors[iface] = &trafficMonitor{cancel: cancel}

	go func() {
		defer listen.Cancel() //nolint:errcheck
		defer func() {
			r.mu.Lock()
			delete(r.monitors, iface)
			r.mu.Unlock()
		}()

		for {
			select {
			case <-monCtx.Done():
				return
			case sentence, ok := <-listen.Chan():
				if !ok {
					return
				}
				stat := parseTrafficSentence(sentence, iface)
				select {
				case ch <- stat:
				default:
				}
			}
		}
	}()

	return nil
}

func (r *interfaceRepository) StopTrafficMonitor(_ context.Context, iface string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	mon, exists := r.monitors[iface]
	if !exists {
		return fmt.Errorf("no traffic monitor running for interface %s", iface)
	}
	mon.cancel()
	delete(r.monitors, iface)
	return nil
}

func parseTrafficSentence(s *proto.Sentence, iface string) entity.TrafficStat {
	rxBps, _ := strconv.ParseInt(s.Map["rx-bits-per-second"], 10, 64)
	txBps, _ := strconv.ParseInt(s.Map["tx-bits-per-second"], 10, 64)
	rxPps, _ := strconv.ParseInt(s.Map["rx-packets-per-second"], 10, 64)
	txPps, _ := strconv.ParseInt(s.Map["tx-packets-per-second"], 10, 64)
	rxDps, _ := strconv.ParseInt(s.Map["rx-drops-per-second"], 10, 64)
	txDps, _ := strconv.ParseInt(s.Map["tx-drops-per-second"], 10, 64)

	return entity.TrafficStat{
		Interface:          iface,
		RxBitsPerSecond:    rxBps,
		TxBitsPerSecond:    txBps,
		RxPacketsPerSecond: rxPps,
		TxPacketsPerSecond: txPps,
		RxDropsPerSecond:   rxDps,
		TxDropsPerSecond:   txDps,
		Timestamp:          time.Now(),
	}
}
