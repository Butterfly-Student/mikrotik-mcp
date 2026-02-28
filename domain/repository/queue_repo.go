package repository

import (
	"context"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

type QueueRepository interface {
	GetAllSimple(ctx context.Context) ([]entity.SimpleQueue, error)
	GetAllTree(ctx context.Context) ([]entity.QueueTree, error)
	CreateSimple(ctx context.Context, req dto.CreateSimpleQueueRequest) error
	CreateTree(ctx context.Context, req dto.CreateQueueTreeRequest) error
	DeleteSimple(ctx context.Context, id string) error
	DeleteTree(ctx context.Context, id string) error
}
