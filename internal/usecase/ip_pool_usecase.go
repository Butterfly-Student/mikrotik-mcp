package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/repository"
)

type IPPoolUseCase struct {
	repo   repository.IPPoolRepository
	logger *zap.Logger
}

func NewIPPoolUseCase(repo repository.IPPoolRepository, logger *zap.Logger) *IPPoolUseCase {
	return &IPPoolUseCase{repo: repo, logger: logger}
}

func (uc *IPPoolUseCase) ListPools(ctx context.Context) (*dto.ListIPPoolResponse, error) {
	pools, err := uc.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ip pools: %w", err)
	}

	responses := make([]dto.IPPoolResponse, len(pools))
	for i, p := range pools {
		responses[i] = dto.IPPoolResponse{
			ID:       p.ID,
			Name:     p.Name,
			Ranges:   p.Ranges,
			NextPool: p.NextPool,
			Comment:  p.Comment,
		}
	}
	return &dto.ListIPPoolResponse{Pools: responses, Total: len(responses)}, nil
}

func (uc *IPPoolUseCase) ListUsed(ctx context.Context) (*dto.ListIPPoolUsedResponse, error) {
	used, err := uc.repo.GetUsed(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ip pool used: %w", err)
	}

	responses := make([]dto.IPPoolUsedResponse, len(used))
	for i, u := range used {
		responses[i] = dto.IPPoolUsedResponse{
			Pool:    u.Pool,
			Address: u.Address,
			Owner:   u.Owner,
			Info:    u.Info,
		}
	}
	return &dto.ListIPPoolUsedResponse{Used: responses, Total: len(responses)}, nil
}

func (uc *IPPoolUseCase) CreatePool(ctx context.Context, req dto.CreateIPPoolRequest) error {
	if req.Name == "" {
		return fmt.Errorf("pool name is required")
	}
	if req.Ranges == "" {
		return fmt.Errorf("pool ranges is required")
	}
	if err := uc.repo.Create(ctx, req); err != nil {
		return fmt.Errorf("failed to create ip pool: %w", err)
	}
	uc.logger.Info("ip pool created", zap.String("name", req.Name))
	return nil
}

func (uc *IPPoolUseCase) UpdatePool(ctx context.Context, req dto.UpdateIPPoolRequest) error {
	if req.ID == "" {
		return fmt.Errorf("pool id is required")
	}
	if err := uc.repo.Update(ctx, req); err != nil {
		return fmt.Errorf("failed to update ip pool: %w", err)
	}
	uc.logger.Info("ip pool updated", zap.String("id", req.ID))
	return nil
}

func (uc *IPPoolUseCase) DeletePool(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("pool id is required")
	}
	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete ip pool: %w", err)
	}
	uc.logger.Info("ip pool deleted", zap.String("id", id))
	return nil
}
