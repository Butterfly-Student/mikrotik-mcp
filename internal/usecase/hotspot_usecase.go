package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/repository"
)

type HotspotUseCase struct {
	repo   repository.HotspotRepository
	logger *zap.Logger
}

func NewHotspotUseCase(repo repository.HotspotRepository, logger *zap.Logger) *HotspotUseCase {
	return &HotspotUseCase{repo: repo, logger: logger}
}

func (uc *HotspotUseCase) ListServers(ctx context.Context) (*dto.ListHotspotServerResponse, error) {
	servers, err := uc.repo.GetServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get hotspot servers: %w", err)
	}

	responses := make([]dto.HotspotServerResponse, len(servers))
	for i, s := range servers {
		responses[i] = dto.HotspotServerResponse{
			ID:          s.ID,
			Name:        s.Name,
			Interface:   s.Interface,
			AddressPool: s.AddressPool,
			Profile:     s.Profile,
			Disabled:    s.Disabled,
		}
	}
	return &dto.ListHotspotServerResponse{Servers: responses, Total: len(responses)}, nil
}

func (uc *HotspotUseCase) ListUsers(ctx context.Context) (*dto.ListHotspotUserResponse, error) {
	users, err := uc.repo.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get hotspot users: %w", err)
	}

	responses := make([]dto.HotspotUserResponse, len(users))
	for i, u := range users {
		responses[i] = dto.HotspotUserResponse{
			ID:              u.ID,
			Name:            u.Name,
			Server:          u.Server,
			Profile:         u.Profile,
			MacAddress:      u.MacAddress,
			IPAddress:       u.IPAddress,
			Comment:         u.Comment,
			Disabled:        u.Disabled,
			LimitBytesIn:    u.LimitBytesIn,
			LimitBytesOut:   u.LimitBytesOut,
			LimitBytesTotal: u.LimitBytesTotal,
			LimitUptime:     u.LimitUptime,
			Uptime:          u.Uptime,
			BytesIn:         u.BytesIn,
			BytesOut:        u.BytesOut,
		}
	}
	return &dto.ListHotspotUserResponse{Users: responses, Total: len(responses)}, nil
}

func (uc *HotspotUseCase) ListActiveUsers(ctx context.Context) (*dto.ListHotspotActiveResponse, error) {
	active, err := uc.repo.GetActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get hotspot active users: %w", err)
	}

	responses := make([]dto.HotspotActiveResponse, len(active))
	for i, a := range active {
		responses[i] = dto.HotspotActiveResponse{
			ID:         a.ID,
			Server:     a.Server,
			User:       a.User,
			Address:    a.Address,
			MacAddress: a.MacAddress,
			LoginBy:    a.LoginBy,
			Uptime:     a.Uptime,
			IdleTime:   a.IdleTime,
			BytesIn:    a.BytesIn,
			BytesOut:   a.BytesOut,
		}
	}
	return &dto.ListHotspotActiveResponse{Active: responses, Total: len(responses)}, nil
}

func (uc *HotspotUseCase) AddUser(ctx context.Context, req dto.CreateHotspotUserRequest) error {
	if req.Name == "" {
		return fmt.Errorf("username is required")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	if err := uc.repo.AddUser(ctx, req); err != nil {
		return fmt.Errorf("failed to add hotspot user: %w", err)
	}
	uc.logger.Info("hotspot user added", zap.String("name", req.Name))
	return nil
}

func (uc *HotspotUseCase) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("user id is required")
	}
	if err := uc.repo.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("failed to delete hotspot user: %w", err)
	}
	uc.logger.Info("hotspot user deleted", zap.String("id", id))
	return nil
}

func (uc *HotspotUseCase) KickActiveUser(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("active session id is required")
	}
	if err := uc.repo.KickActiveUser(ctx, id); err != nil {
		return fmt.Errorf("failed to kick hotspot user: %w", err)
	}
	uc.logger.Info("hotspot active user kicked", zap.String("id", id))
	return nil
}
