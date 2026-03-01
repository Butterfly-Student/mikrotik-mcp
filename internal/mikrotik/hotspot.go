package mikrotik

import (
	"context"
	"fmt"
	"strconv"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type hotspotRepository struct {
	client *Client
}

func NewHotspotRepository(client *Client) repository.HotspotRepository {
	return &hotspotRepository{client: client}
}

func (r *hotspotRepository) GetServers(ctx context.Context) ([]entity.HotspotServer, error) {
	reply, err := r.client.RunContext(ctx, "/ip/hotspot/print")
	if err != nil {
		return nil, fmt.Errorf("hotspot server print: %w", err)
	}

	servers := make([]entity.HotspotServer, 0, len(reply.Re))
	for _, s := range reply.Re {
		addrPerMac, _ := strconv.Atoi(s.Map["addresses-per-mac"])
		servers = append(servers, entity.HotspotServer{
			ID:               s.Map[".id"],
			Name:             s.Map["name"],
			Interface:        s.Map["interface"],
			AddressPool:      s.Map["address-pool"],
			Profile:          s.Map["profile"],
			IdleTimeout:      s.Map["idle-timeout"],
			KeepaliveTimeout: s.Map["keepalive-timeout"],
			LoginTimeout:     s.Map["login-timeout"],
			AddressesPerMac:  addrPerMac,
			Disabled:         s.Map["disabled"] == "true",
			Invalid:          s.Map["invalid"] == "true",
			HTTPS:            s.Map["https"] == "true",
		})
	}
	return servers, nil
}

func (r *hotspotRepository) GetUsers(ctx context.Context) ([]entity.HotspotUser, error) {
	reply, err := r.client.RunContext(ctx, "/ip/hotspot/user/print")
	if err != nil {
		return nil, fmt.Errorf("hotspot user print: %w", err)
	}

	users := make([]entity.HotspotUser, 0, len(reply.Re))
	for _, s := range reply.Re {
		limitBytesIn, _ := strconv.ParseInt(s.Map["limit-bytes-in"], 10, 64)
		limitBytesOut, _ := strconv.ParseInt(s.Map["limit-bytes-out"], 10, 64)
		limitBytesTotal, _ := strconv.ParseInt(s.Map["limit-bytes-total"], 10, 64)
		bytesIn, _ := strconv.ParseInt(s.Map["bytes-in"], 10, 64)
		bytesOut, _ := strconv.ParseInt(s.Map["bytes-out"], 10, 64)
		bytesTotal, _ := strconv.ParseInt(s.Map["bytes-total"], 10, 64)

		users = append(users, entity.HotspotUser{
			ID:              s.Map[".id"],
			Name:            s.Map["name"],
			Server:          s.Map["server"],
			Profile:         s.Map["profile"],
			MacAddress:      s.Map["mac-address"],
			IPAddress:       s.Map["ip-address"],
			Comment:         s.Map["comment"],
			Disabled:        s.Map["disabled"] == "true",
			LimitBytesIn:    limitBytesIn,
			LimitBytesOut:   limitBytesOut,
			LimitBytesTotal: limitBytesTotal,
			LimitUptime:     s.Map["limit-uptime"],
			Uptime:          s.Map["uptime"],
			BytesIn:         bytesIn,
			BytesOut:        bytesOut,
			BytesTotal:      bytesTotal,
		})
	}
	return users, nil
}

func (r *hotspotRepository) GetActiveUsers(ctx context.Context) ([]entity.HotspotActive, error) {
	reply, err := r.client.RunContext(ctx, "/ip/hotspot/active/print")
	if err != nil {
		return nil, fmt.Errorf("hotspot active print: %w", err)
	}

	active := make([]entity.HotspotActive, 0, len(reply.Re))
	for _, s := range reply.Re {
		bytesIn, _ := strconv.ParseInt(s.Map["bytes-in"], 10, 64)
		bytesOut, _ := strconv.ParseInt(s.Map["bytes-out"], 10, 64)
		packetsIn, _ := strconv.ParseInt(s.Map["packets-in"], 10, 64)
		packetsOut, _ := strconv.ParseInt(s.Map["packets-out"], 10, 64)

		active = append(active, entity.HotspotActive{
			ID:              s.Map[".id"],
			Server:          s.Map["server"],
			User:            s.Map["user"],
			Domain:          s.Map["domain"],
			Address:         s.Map["address"],
			MacAddress:      s.Map["mac-address"],
			LoginBy:         s.Map["login-by"],
			Uptime:          s.Map["uptime"],
			IdleTime:        s.Map["idle-time"],
			SessionTimeLeft: s.Map["session-time-left"],
			BytesIn:         bytesIn,
			BytesOut:        bytesOut,
			PacketsIn:       packetsIn,
			PacketsOut:      packetsOut,
		})
	}
	return active, nil
}

func (r *hotspotRepository) AddUser(ctx context.Context, req dto.CreateHotspotUserRequest) error {
	args := []string{
		"/ip/hotspot/user/add",
		"=name=" + req.Name,
		"=password=" + req.Password,
	}
	if req.Server != "" {
		args = append(args, "=server="+req.Server)
	}
	if req.Profile != "" {
		args = append(args, "=profile="+req.Profile)
	}
	if req.MacAddress != "" {
		args = append(args, "=mac-address="+req.MacAddress)
	}
	if req.IPAddress != "" {
		args = append(args, "=ip-address="+req.IPAddress)
	}
	if req.LimitUptime != "" {
		args = append(args, "=limit-uptime="+req.LimitUptime)
	}
	if req.LimitBytesTotal > 0 {
		args = append(args, "=limit-bytes-total="+strconv.FormatInt(req.LimitBytesTotal, 10))
	}
	if req.LimitBytesIn > 0 {
		args = append(args, "=limit-bytes-in="+strconv.FormatInt(req.LimitBytesIn, 10))
	}
	if req.LimitBytesOut > 0 {
		args = append(args, "=limit-bytes-out="+strconv.FormatInt(req.LimitBytesOut, 10))
	}
	if req.Comment != "" {
		args = append(args, "=comment="+req.Comment)
	}

	_, err := r.client.RunArgsContext(ctx, args)
	if err != nil {
		return fmt.Errorf("hotspot user add: %w", err)
	}
	return nil
}

func (r *hotspotRepository) DeleteUser(ctx context.Context, id string) error {
	_, err := r.client.RunContext(ctx, "/ip/hotspot/user/remove", "=.id="+id)
	if err != nil {
		return fmt.Errorf("hotspot user remove: %w", err)
	}
	return nil
}

func (r *hotspotRepository) KickActiveUser(ctx context.Context, id string) error {
	_, err := r.client.RunContext(ctx, "/ip/hotspot/active/remove", "=.id="+id)
	if err != nil {
		return fmt.Errorf("hotspot active kick: %w", err)
	}
	return nil
}
