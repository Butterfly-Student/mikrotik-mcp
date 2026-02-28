package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/repository"
)

type QueueUseCase struct {
	repo   repository.QueueRepository
	logger *zap.Logger
}

func NewQueueUseCase(repo repository.QueueRepository, logger *zap.Logger) *QueueUseCase {
	return &QueueUseCase{repo: repo, logger: logger}
}

func (uc *QueueUseCase) ListSimpleQueues(ctx context.Context) (*dto.ListSimpleQueueResponse, error) {
	queues, err := uc.repo.GetAllSimple(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get simple queues: %w", err)
	}

	responses := make([]dto.SimpleQueueResponse, len(queues))
	for i, q := range queues {
		responses[i] = dto.SimpleQueueResponse{
			ID:             q.ID,
			Name:           q.Name,
			Target:         q.Target,
			MaxLimit:       q.MaxLimit,
			LimitAt:        q.LimitAt,
			BurstLimit:     q.BurstLimit,
			BurstThreshold: q.BurstThreshold,
			BurstTime:      q.BurstTime,
			Parent:         q.Parent,
			Priority:       q.Priority,
			Comment:        q.Comment,
			Disabled:       q.Disabled,
			Rate:           q.Rate,
		}
	}
	return &dto.ListSimpleQueueResponse{Queues: responses, Total: len(responses)}, nil
}

func (uc *QueueUseCase) ListTreeQueues(ctx context.Context) (*dto.ListQueueTreeResponse, error) {
	queues, err := uc.repo.GetAllTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree queues: %w", err)
	}

	responses := make([]dto.QueueTreeResponse, len(queues))
	for i, q := range queues {
		responses[i] = dto.QueueTreeResponse{
			ID:         q.ID,
			Name:       q.Name,
			Parent:     q.Parent,
			PacketMark: q.PacketMark,
			MaxLimit:   q.MaxLimit,
			Priority:   q.Priority,
			Comment:    q.Comment,
			Disabled:   q.Disabled,
		}
	}
	return &dto.ListQueueTreeResponse{Queues: responses, Total: len(responses)}, nil
}

func (uc *QueueUseCase) AddSimpleQueue(ctx context.Context, req dto.CreateSimpleQueueRequest) error {
	if req.Name == "" {
		return fmt.Errorf("queue name is required")
	}
	if req.Target == "" {
		return fmt.Errorf("queue target is required (e.g. 192.168.1.100/32)")
	}
	if err := uc.repo.CreateSimple(ctx, req); err != nil {
		return fmt.Errorf("failed to create simple queue: %w", err)
	}
	uc.logger.Info("simple queue created", zap.String("name", req.Name), zap.String("target", req.Target))
	return nil
}

func (uc *QueueUseCase) AddTreeQueue(ctx context.Context, req dto.CreateQueueTreeRequest) error {
	if req.Name == "" {
		return fmt.Errorf("queue name is required")
	}
	if req.Parent == "" {
		return fmt.Errorf("parent is required (e.g. global, ether1)")
	}
	if err := uc.repo.CreateTree(ctx, req); err != nil {
		return fmt.Errorf("failed to create tree queue: %w", err)
	}
	uc.logger.Info("tree queue created", zap.String("name", req.Name))
	return nil
}

func (uc *QueueUseCase) DeleteSimpleQueue(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("queue id is required")
	}
	if err := uc.repo.DeleteSimple(ctx, id); err != nil {
		return fmt.Errorf("failed to delete simple queue: %w", err)
	}
	uc.logger.Info("simple queue deleted", zap.String("id", id))
	return nil
}

func (uc *QueueUseCase) DeleteTreeQueue(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("queue id is required")
	}
	if err := uc.repo.DeleteTree(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tree queue: %w", err)
	}
	uc.logger.Info("tree queue deleted", zap.String("id", id))
	return nil
}
