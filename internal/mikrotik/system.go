package mikrotik

import (
	"context"
	"fmt"
	"strconv"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type systemRepository struct {
	client *Client
}

func NewSystemRepository(client *Client) repository.SystemRepository {
	return &systemRepository{client: client}
}

func (r *systemRepository) GetResource(ctx context.Context) (*entity.SystemResource, error) {
	reply, err := r.client.Run("/system/resource/print")
	if err != nil {
		return nil, fmt.Errorf("system resource print: %w", err)
	}
	if len(reply.Re) == 0 {
		return nil, fmt.Errorf("no system resource data")
	}

	s := reply.Re[0]
	freeMemory, _ := strconv.ParseInt(s.Map["free-memory"], 10, 64)
	totalMemory, _ := strconv.ParseInt(s.Map["total-memory"], 10, 64)
	cpuLoad, _ := strconv.Atoi(s.Map["cpu-load"])
	cpuCount, _ := strconv.Atoi(s.Map["cpu-count"])
	cpuFrequency, _ := strconv.Atoi(s.Map["cpu-frequency"])
	freeHDD, _ := strconv.ParseInt(s.Map["free-hdd-space"], 10, 64)
	totalHDD, _ := strconv.ParseInt(s.Map["total-hdd-space"], 10, 64)

	return &entity.SystemResource{
		Uptime:           s.Map["uptime"],
		Version:          s.Map["version"],
		BuildTime:        s.Map["build-time"],
		FreeMemory:       freeMemory,
		TotalMemory:      totalMemory,
		CPU:              s.Map["cpu"],
		CPUCount:         cpuCount,
		CPUFrequency:     cpuFrequency,
		CPULoad:          cpuLoad,
		FreeHDDSpace:     freeHDD,
		TotalHDDSpace:    totalHDD,
		ArchitectureName: s.Map["architecture-name"],
		BoardName:        s.Map["board-name"],
		Platform:         s.Map["platform"],
	}, nil
}

func (r *systemRepository) GetLogs(ctx context.Context, req dto.GetLogsRequest) ([]entity.SystemLog, error) {
	args := []string{"/log/print"}
	if req.Topics != "" {
		args = append(args, "?topics="+req.Topics)
	}

	reply, err := r.client.Run(args...)
	if err != nil {
		return nil, fmt.Errorf("log print: %w", err)
	}

	limit := req.Limit
	if limit <= 0 || limit > len(reply.Re) {
		limit = len(reply.Re)
	}

	logs := make([]entity.SystemLog, 0, limit)
	// Return most recent logs (last N entries)
	start := len(reply.Re) - limit
	if start < 0 {
		start = 0
	}
	for _, s := range reply.Re[start:] {
		logs = append(logs, entity.SystemLog{
			ID:      s.Map[".id"],
			Time:    s.Map["time"],
			Topics:  s.Map["topics"],
			Message: s.Map["message"],
		})
	}
	return logs, nil
}

func (r *systemRepository) GetIdentity(ctx context.Context) (*entity.SystemIdentity, error) {
	reply, err := r.client.Run("/system/identity/print")
	if err != nil {
		return nil, fmt.Errorf("system identity print: %w", err)
	}
	if len(reply.Re) == 0 {
		return nil, fmt.Errorf("no identity data")
	}
	return &entity.SystemIdentity{
		Name: reply.Re[0].Map["name"],
	}, nil
}

func (r *systemRepository) Reboot(ctx context.Context) error {
	_, err := r.client.Run("/system/reboot")
	if err != nil {
		return fmt.Errorf("system reboot: %w", err)
	}
	return nil
}
