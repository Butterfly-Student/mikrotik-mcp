package mikrotik

import (
	"context"
	"fmt"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type ipPoolRepository struct {
	client *Client
}

func NewIPPoolRepository(client *Client) repository.IPPoolRepository {
	return &ipPoolRepository{client: client}
}

func (r *ipPoolRepository) GetAll(ctx context.Context) ([]entity.IPPool, error) {
	reply, err := r.client.Run("/ip/pool/print")
	if err != nil {
		return nil, fmt.Errorf("ip pool print: %w", err)
	}

	pools := make([]entity.IPPool, 0, len(reply.Re))
	for _, s := range reply.Re {
		pools = append(pools, entity.IPPool{
			ID:       s.Map[".id"],
			Name:     s.Map["name"],
			Ranges:   s.Map["ranges"],
			NextPool: s.Map["next-pool"],
			Comment:  s.Map["comment"],
		})
	}
	return pools, nil
}

func (r *ipPoolRepository) GetUsed(ctx context.Context) ([]entity.IPPoolUsed, error) {
	reply, err := r.client.Run("/ip/pool/used/print")
	if err != nil {
		return nil, fmt.Errorf("ip pool used print: %w", err)
	}

	used := make([]entity.IPPoolUsed, 0, len(reply.Re))
	for _, s := range reply.Re {
		used = append(used, entity.IPPoolUsed{
			Pool:    s.Map["pool"],
			Address: s.Map["address"],
			Owner:   s.Map["owner"],
			Info:    s.Map["info"],
		})
	}
	return used, nil
}

func (r *ipPoolRepository) GetByName(ctx context.Context, name string) (*entity.IPPool, error) {
	reply, err := r.client.Run("/ip/pool/print", "?name="+name)
	if err != nil {
		return nil, fmt.Errorf("ip pool find by name: %w", err)
	}
	if len(reply.Re) == 0 {
		return nil, fmt.Errorf("ip pool not found: %s", name)
	}
	s := reply.Re[0]
	return &entity.IPPool{
		ID:       s.Map[".id"],
		Name:     s.Map["name"],
		Ranges:   s.Map["ranges"],
		NextPool: s.Map["next-pool"],
		Comment:  s.Map["comment"],
	}, nil
}

func (r *ipPoolRepository) Create(ctx context.Context, req dto.CreateIPPoolRequest) error {
	args := []string{"/ip/pool/add", "=name=" + req.Name, "=ranges=" + req.Ranges}
	if req.NextPool != "" {
		args = append(args, "=next-pool="+req.NextPool)
	}
	if req.Comment != "" {
		args = append(args, "=comment="+req.Comment)
	}
	_, err := r.client.Run(args...)
	if err != nil {
		return fmt.Errorf("ip pool add: %w", err)
	}
	return nil
}

func (r *ipPoolRepository) Update(ctx context.Context, req dto.UpdateIPPoolRequest) error {
	args := []string{"/ip/pool/set", "=.id=" + req.ID}
	if req.Ranges != "" {
		args = append(args, "=ranges="+req.Ranges)
	}
	if req.NextPool != "" {
		args = append(args, "=next-pool="+req.NextPool)
	}
	if req.Comment != "" {
		args = append(args, "=comment="+req.Comment)
	}
	_, err := r.client.Run(args...)
	if err != nil {
		return fmt.Errorf("ip pool set: %w", err)
	}
	return nil
}

func (r *ipPoolRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Run("/ip/pool/remove", "=.id="+id)
	if err != nil {
		return fmt.Errorf("ip pool remove: %w", err)
	}
	return nil
}
