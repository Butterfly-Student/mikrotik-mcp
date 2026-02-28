package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/repository"
)

type FirewallUseCase struct {
	repo   repository.FirewallRepository
	logger *zap.Logger
}

func NewFirewallUseCase(repo repository.FirewallRepository, logger *zap.Logger) *FirewallUseCase {
	return &FirewallUseCase{repo: repo, logger: logger}
}

func (uc *FirewallUseCase) ListRules(ctx context.Context) (*dto.ListFirewallRuleResponse, error) {
	rules, err := uc.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall rules: %w", err)
	}

	responses := make([]dto.FirewallRuleResponse, len(rules))
	for i, r := range rules {
		responses[i] = dto.FirewallRuleResponse{
			ID:              r.ID,
			Chain:           r.Chain,
			Action:          r.Action,
			SrcAddress:      r.SrcAddress,
			DstAddress:      r.DstAddress,
			SrcAddressList:  r.SrcAddressList,
			DstAddressList:  r.DstAddressList,
			Protocol:        r.Protocol,
			SrcPort:         r.SrcPort,
			DstPort:         r.DstPort,
			InInterface:     r.InInterface,
			OutInterface:    r.OutInterface,
			ConnectionState: r.ConnectionState,
			Comment:         r.Comment,
			Disabled:        r.Disabled,
			Dynamic:         r.Dynamic,
			Bytes:           r.Bytes,
			Packets:         r.Packets,
		}
	}
	return &dto.ListFirewallRuleResponse{Rules: responses, Total: len(responses)}, nil
}

func (uc *FirewallUseCase) CreateRule(ctx context.Context, req dto.CreateFirewallRuleRequest) error {
	if req.Chain == "" {
		return fmt.Errorf("chain is required (input, forward, output)")
	}
	if req.Action == "" {
		return fmt.Errorf("action is required (accept, drop, reject, etc.)")
	}
	if err := uc.repo.Create(ctx, req); err != nil {
		return fmt.Errorf("failed to create firewall rule: %w", err)
	}
	uc.logger.Info("firewall rule created",
		zap.String("chain", req.Chain),
		zap.String("action", req.Action),
	)
	return nil
}

func (uc *FirewallUseCase) DeleteRule(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("rule id is required")
	}
	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete firewall rule: %w", err)
	}
	uc.logger.Info("firewall rule deleted", zap.String("id", id))
	return nil
}

func (uc *FirewallUseCase) ToggleRule(ctx context.Context, id string, disabled bool) error {
	if id == "" {
		return fmt.Errorf("rule id is required")
	}
	if err := uc.repo.Toggle(ctx, id, disabled); err != nil {
		return fmt.Errorf("failed to toggle firewall rule: %w", err)
	}
	uc.logger.Info("firewall rule toggled",
		zap.String("id", id),
		zap.Bool("disabled", disabled),
	)
	return nil
}
