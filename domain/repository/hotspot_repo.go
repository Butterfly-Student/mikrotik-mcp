package repository

import (
	"context"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

type HotspotRepository interface {
	GetServers(ctx context.Context) ([]entity.HotspotServer, error)
	GetUsers(ctx context.Context) ([]entity.HotspotUser, error)
	GetActiveUsers(ctx context.Context) ([]entity.HotspotActive, error)
	AddUser(ctx context.Context, req dto.CreateHotspotUserRequest) error
	DeleteUser(ctx context.Context, id string) error
	KickActiveUser(ctx context.Context, id string) error
}
