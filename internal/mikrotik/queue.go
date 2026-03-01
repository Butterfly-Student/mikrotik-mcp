package mikrotik

import (
	"context"
	"fmt"
	"strconv"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type queueRepository struct {
	client *Client
}

func NewQueueRepository(client *Client) repository.QueueRepository {
	return &queueRepository{client: client}
}

func (r *queueRepository) GetAllSimple(ctx context.Context) ([]entity.SimpleQueue, error) {
	reply, err := r.client.RunContext(ctx, "/queue/simple/print")
	if err != nil {
		return nil, fmt.Errorf("queue simple print: %w", err)
	}

	queues := make([]entity.SimpleQueue, 0, len(reply.Re))
	for _, s := range reply.Re {
		priority, _ := strconv.Atoi(s.Map["priority"])
		queues = append(queues, entity.SimpleQueue{
			ID:             s.Map[".id"],
			Name:           s.Map["name"],
			Target:         s.Map["target"],
			DstAddress:     s.Map["dst-address"],
			Parent:         s.Map["parent"],
			Priority:       priority,
			Queue:          s.Map["queue"],
			MaxLimit:       s.Map["max-limit"],
			LimitAt:        s.Map["limit-at"],
			BurstLimit:     s.Map["burst-limit"],
			BurstThreshold: s.Map["burst-threshold"],
			BurstTime:      s.Map["burst-time"],
			PacketMarks:    s.Map["packet-marks"],
			Comment:        s.Map["comment"],
			Disabled:       s.Map["disabled"] == "true",
			Dynamic:        s.Map["dynamic"] == "true",
			Invalid:        s.Map["invalid"] == "true",
		})
	}
	return queues, nil
}

func (r *queueRepository) GetAllTree(ctx context.Context) ([]entity.QueueTree, error) {
	reply, err := r.client.RunContext(ctx, "/queue/tree/print")
	if err != nil {
		return nil, fmt.Errorf("queue tree print: %w", err)
	}

	queues := make([]entity.QueueTree, 0, len(reply.Re))
	for _, s := range reply.Re {
		priority, _ := strconv.Atoi(s.Map["priority"])
		queues = append(queues, entity.QueueTree{
			ID:             s.Map[".id"],
			Name:           s.Map["name"],
			Parent:         s.Map["parent"],
			PacketMark:     s.Map["packet-mark"],
			Priority:       priority,
			MaxLimit:       s.Map["max-limit"],
			LimitAt:        s.Map["limit-at"],
			BurstLimit:     s.Map["burst-limit"],
			BurstThreshold: s.Map["burst-threshold"],
			BurstTime:      s.Map["burst-time"],
			Queue:          s.Map["queue"],
			Comment:        s.Map["comment"],
			Disabled:       s.Map["disabled"] == "true",
		})
	}
	return queues, nil
}

func (r *queueRepository) CreateSimple(ctx context.Context, req dto.CreateSimpleQueueRequest) error {
	args := []string{
		"/queue/simple/add",
		"=name=" + req.Name,
		"=target=" + req.Target,
	}
	if req.MaxLimit != "" {
		args = append(args, "=max-limit="+req.MaxLimit)
	}
	if req.LimitAt != "" {
		args = append(args, "=limit-at="+req.LimitAt)
	}
	if req.BurstLimit != "" {
		args = append(args, "=burst-limit="+req.BurstLimit)
	}
	if req.BurstThreshold != "" {
		args = append(args, "=burst-threshold="+req.BurstThreshold)
	}
	if req.BurstTime != "" {
		args = append(args, "=burst-time="+req.BurstTime)
	}
	if req.Parent != "" {
		args = append(args, "=parent="+req.Parent)
	}
	if req.Priority > 0 {
		args = append(args, "=priority="+strconv.Itoa(req.Priority))
	}
	if req.Comment != "" {
		args = append(args, "=comment="+req.Comment)
	}

	_, err := r.client.RunArgsContext(ctx, args)
	if err != nil {
		return fmt.Errorf("queue simple add: %w", err)
	}
	return nil
}

func (r *queueRepository) CreateTree(ctx context.Context, req dto.CreateQueueTreeRequest) error {
	args := []string{
		"/queue/tree/add",
		"=name=" + req.Name,
		"=parent=" + req.Parent,
	}
	if req.PacketMark != "" {
		args = append(args, "=packet-mark="+req.PacketMark)
	}
	if req.MaxLimit != "" {
		args = append(args, "=max-limit="+req.MaxLimit)
	}
	if req.LimitAt != "" {
		args = append(args, "=limit-at="+req.LimitAt)
	}
	if req.Priority > 0 {
		args = append(args, "=priority="+strconv.Itoa(req.Priority))
	}
	if req.Comment != "" {
		args = append(args, "=comment="+req.Comment)
	}

	_, err := r.client.RunArgsContext(ctx, args)
	if err != nil {
		return fmt.Errorf("queue tree add: %w", err)
	}
	return nil
}

func (r *queueRepository) DeleteSimple(ctx context.Context, id string) error {
	_, err := r.client.RunContext(ctx, "/queue/simple/remove", "=.id="+id)
	if err != nil {
		return fmt.Errorf("queue simple remove: %w", err)
	}
	return nil
}

func (r *queueRepository) DeleteTree(ctx context.Context, id string) error {
	_, err := r.client.RunContext(ctx, "/queue/tree/remove", "=.id="+id)
	if err != nil {
		return fmt.Errorf("queue tree remove: %w", err)
	}
	return nil
}
