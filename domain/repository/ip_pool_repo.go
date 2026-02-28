package repository

import (
	"context"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

type IPPoolRepository interface {
	GetAll(ctx context.Context) ([]entity.IPPool, error)
	GetUsed(ctx context.Context) ([]entity.IPPoolUsed, error)
	GetByName(ctx context.Context, name string) (*entity.IPPool, error)
	Create(ctx context.Context, req dto.CreateIPPoolRequest) error
	Update(ctx context.Context, req dto.UpdateIPPoolRequest) error
	Delete(ctx context.Context, id string) error
}
