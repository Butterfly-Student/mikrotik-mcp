package repository

import (
	"context"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

type SystemRepository interface {
	GetResource(ctx context.Context) (*entity.SystemResource, error)
	GetLogs(ctx context.Context, req dto.GetLogsRequest) ([]entity.SystemLog, error)
	GetIdentity(ctx context.Context) (*entity.SystemIdentity, error)
	Reboot(ctx context.Context) error
}
