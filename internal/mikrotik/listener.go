package mikrotik

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"mikrotik-mcp/domain/entity"
)

type trafficMonitor struct {
	cancel context.CancelFunc
}

type listenerManager struct {
	client   *Client
	monitors map[string]*trafficMonitor
	mu       sync.Mutex
}

func newListenerManager(client *Client) *listenerManager {
	return &listenerManager{
		client:   client,
		monitors: make(map[string]*trafficMonitor),
	}
}

func (lm *listenerManager) StartTrafficMonitor(ctx context.Context, iface string, ch chan<- entity.TrafficStat) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if _, exists := lm.monitors[iface]; exists {
		return fmt.Errorf("traffic monitor already running for interface %s", iface)
	}

	listen, err := lm.client.ListenArgs([]string{
		"/interface/monitor-traffic",
		"=interface=" + iface,
	})
	if err != nil {
		return fmt.Errorf("failed to start traffic monitor for %s: %w", iface, err)
	}

	monCtx, cancel := context.WithCancel(ctx)
	lm.monitors[iface] = &trafficMonitor{cancel: cancel}

	go func() {
		defer listen.Cancel() //nolint:errcheck
		defer func() {
			lm.mu.Lock()
			delete(lm.monitors, iface)
			lm.mu.Unlock()
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

func (lm *listenerManager) StopTrafficMonitor(_ context.Context, iface string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	mon, exists := lm.monitors[iface]
	if !exists {
		return fmt.Errorf("no traffic monitor running for interface %s", iface)
	}
	mon.cancel()
	delete(lm.monitors, iface)
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
