package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/repository"
)

type SystemUseCase struct {
	repo   repository.SystemRepository
	logger *zap.Logger
}

func NewSystemUseCase(repo repository.SystemRepository, logger *zap.Logger) *SystemUseCase {
	return &SystemUseCase{repo: repo, logger: logger}
}

func (uc *SystemUseCase) GetResource(ctx context.Context) (*dto.SystemResourceResponse, error) {
	res, err := uc.repo.GetResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system resource: %w", err)
	}

	return &dto.SystemResourceResponse{
		Uptime:           res.Uptime,
		Version:          res.Version,
		CPULoad:          res.CPULoad,
		CPUCount:         res.CPUCount,
		FreeMemory:       formatBytes(res.FreeMemory),
		TotalMemory:      formatBytes(res.TotalMemory),
		FreeHDDSpace:     formatBytes(res.FreeHDDSpace),
		TotalHDDSpace:    formatBytes(res.TotalHDDSpace),
		BoardName:        res.BoardName,
		ArchitectureName: res.ArchitectureName,
		Platform:         res.Platform,
	}, nil
}

func (uc *SystemUseCase) GetLogs(ctx context.Context, req dto.GetLogsRequest) (*dto.ListSystemLogResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 50
	}

	logs, err := uc.repo.GetLogs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get system logs: %w", err)
	}

	responses := make([]dto.SystemLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = dto.SystemLogResponse{
			ID:      l.ID,
			Time:    l.Time,
			Topics:  l.Topics,
			Message: l.Message,
		}
	}
	return &dto.ListSystemLogResponse{Logs: responses, Total: len(responses)}, nil
}

func (uc *SystemUseCase) GetIdentity(ctx context.Context) (*dto.SystemIdentityResponse, error) {
	identity, err := uc.repo.GetIdentity(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system identity: %w", err)
	}
	return &dto.SystemIdentityResponse{Name: identity.Name}, nil
}

func (uc *SystemUseCase) Reboot(ctx context.Context) error {
	if err := uc.repo.Reboot(ctx); err != nil {
		return fmt.Errorf("failed to reboot router: %w", err)
	}
	uc.logger.Warn("router reboot initiated")
	return nil
}

func formatBytes(b int64) string {
	const (
		_          = iota
		KB float64 = 1 << (10 * iota)
		MB
		GB
	)
	fb := float64(b)
	switch {
	case fb >= GB:
		return fmt.Sprintf("%.2f GB", fb/GB)
	case fb >= MB:
		return fmt.Sprintf("%.2f MB", fb/MB)
	case fb >= KB:
		return fmt.Sprintf("%.2f KB", fb/KB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
