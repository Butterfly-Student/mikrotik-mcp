package repository

import (
	"context"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

type FirewallRepository interface {
	GetAll(ctx context.Context) ([]entity.FirewallRule, error)
	Create(ctx context.Context, req dto.CreateFirewallRuleRequest) error
	Delete(ctx context.Context, id string) error
	Toggle(ctx context.Context, id string, disabled bool) error
}
