package usecase

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type InterfaceUseCase struct {
	repo   repository.InterfaceRepository
	logger *zap.Logger
}

func NewInterfaceUseCase(repo repository.InterfaceRepository, logger *zap.Logger) *InterfaceUseCase {
	return &InterfaceUseCase{repo: repo, logger: logger}
}

func (uc *InterfaceUseCase) ListInterfaces(ctx context.Context) (*dto.ListInterfaceResponse, error) {
	ifaces, err := uc.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	responses := make([]dto.InterfaceResponse, len(ifaces))
	for i, iface := range ifaces {
		responses[i] = dto.InterfaceResponse{
			ID:         iface.ID,
			Name:       iface.Name,
			Type:       iface.Type,
			MacAddress: iface.MacAddress,
			MTU:        iface.MTU,
			Running:    iface.Running,
			Disabled:   iface.Disabled,
			Comment:    iface.Comment,
		}
	}
	return &dto.ListInterfaceResponse{Interfaces: responses, Total: len(responses)}, nil
}

func (uc *InterfaceUseCase) WatchTraffic(ctx context.Context, req dto.WatchTrafficRequest) (*dto.WatchTrafficResponse, error) {
	if req.Interface == "" {
		return nil, fmt.Errorf("interface name is required")
	}
	if req.Seconds <= 0 {
		return nil, fmt.Errorf("seconds must be greater than 0")
	}
	if req.Seconds > 60 {
		return nil, fmt.Errorf("seconds must not exceed 60")
	}

	ch := make(chan entity.TrafficStat, 100)
	monCtx, cancel := context.WithTimeout(ctx, time.Duration(req.Seconds+2)*time.Second)
	defer cancel()

	if err := uc.repo.StartTrafficMonitor(monCtx, req.Interface, ch); err != nil {
		return nil, fmt.Errorf("failed to start traffic monitor: %w", err)
	}
	defer func() {
		_ = uc.repo.StopTrafficMonitor(context.Background(), req.Interface)
	}()

	timer := time.NewTimer(time.Duration(req.Seconds) * time.Second)
	defer timer.Stop()

	var samples []dto.TrafficStatResponse
	for {
		select {
		case <-timer.C:
			return &dto.WatchTrafficResponse{
				Interface: req.Interface,
				Samples:   samples,
				Duration:  req.Seconds,
			}, nil
		case stat, ok := <-ch:
			if !ok {
				return &dto.WatchTrafficResponse{
					Interface: req.Interface,
					Samples:   samples,
					Duration:  req.Seconds,
				}, nil
			}
			samples = append(samples, dto.TrafficStatResponse{
				Interface:          stat.Interface,
				RxBps:              stat.RxBitsPerSecond,
				TxBps:              stat.TxBitsPerSecond,
				RxPacketsPerSecond: stat.RxPacketsPerSecond,
				TxPacketsPerSecond: stat.TxPacketsPerSecond,
				Timestamp:          stat.Timestamp.Format(time.RFC3339),
			})
		case <-monCtx.Done():
			return &dto.WatchTrafficResponse{
				Interface: req.Interface,
				Samples:   samples,
				Duration:  req.Seconds,
			}, nil
		}
	}
}
