package mikrotik

import (
	"context"
	"fmt"
	"strconv"

	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
	"mikrotik-mcp/domain/repository"
)

type firewallRepository struct {
	client *Client
}

func NewFirewallRepository(client *Client) repository.FirewallRepository {
	return &firewallRepository{client: client}
}

func (r *firewallRepository) GetAll(ctx context.Context) ([]entity.FirewallRule, error) {
	reply, err := r.client.Run("/ip/firewall/filter/print")
	if err != nil {
		return nil, fmt.Errorf("firewall filter print: %w", err)
	}

	rules := make([]entity.FirewallRule, 0, len(reply.Re))
	for _, s := range reply.Re {
		bytes, _ := strconv.ParseInt(s.Map["bytes"], 10, 64)
		packets, _ := strconv.ParseInt(s.Map["packets"], 10, 64)

		rules = append(rules, entity.FirewallRule{
			ID:              s.Map[".id"],
			Chain:           s.Map["chain"],
			Action:          s.Map["action"],
			SrcAddress:      s.Map["src-address"],
			DstAddress:      s.Map["dst-address"],
			SrcAddressList:  s.Map["src-address-list"],
			DstAddressList:  s.Map["dst-address-list"],
			Protocol:        s.Map["protocol"],
			SrcPort:         s.Map["src-port"],
			DstPort:         s.Map["dst-port"],
			InInterface:     s.Map["in-interface"],
			OutInterface:    s.Map["out-interface"],
			ConnectionState: s.Map["connection-state"],
			Comment:         s.Map["comment"],
			Disabled:        s.Map["disabled"] == "true",
			Dynamic:         s.Map["dynamic"] == "true",
			Log:             s.Map["log"] == "true",
			LogPrefix:       s.Map["log-prefix"],
			Bytes:           bytes,
			Packets:         packets,
		})
	}
	return rules, nil
}

func (r *firewallRepository) Create(ctx context.Context, req dto.CreateFirewallRuleRequest) error {
	args := []string{
		"/ip/firewall/filter/add",
		"=chain=" + req.Chain,
		"=action=" + req.Action,
	}
	if req.SrcAddress != "" {
		args = append(args, "=src-address="+req.SrcAddress)
	}
	if req.DstAddress != "" {
		args = append(args, "=dst-address="+req.DstAddress)
	}
	if req.SrcAddressList != "" {
		args = append(args, "=src-address-list="+req.SrcAddressList)
	}
	if req.DstAddressList != "" {
		args = append(args, "=dst-address-list="+req.DstAddressList)
	}
	if req.Protocol != "" {
		args = append(args, "=protocol="+req.Protocol)
	}
	if req.SrcPort != "" {
		args = append(args, "=src-port="+req.SrcPort)
	}
	if req.DstPort != "" {
		args = append(args, "=dst-port="+req.DstPort)
	}
	if req.InInterface != "" {
		args = append(args, "=in-interface="+req.InInterface)
	}
	if req.OutInterface != "" {
		args = append(args, "=out-interface="+req.OutInterface)
	}
	if req.ConnectionState != "" {
		args = append(args, "=connection-state="+req.ConnectionState)
	}
	if req.Comment != "" {
		args = append(args, "=comment="+req.Comment)
	}
	if req.PlaceBefore != "" {
		args = append(args, "=place-before="+req.PlaceBefore)
	}

	_, err := r.client.Run(args...)
	if err != nil {
		return fmt.Errorf("firewall filter add: %w", err)
	}
	return nil
}

func (r *firewallRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Run("/ip/firewall/filter/remove", "=.id="+id)
	if err != nil {
		return fmt.Errorf("firewall filter remove: %w", err)
	}
	return nil
}

func (r *firewallRepository) Toggle(ctx context.Context, id string, disabled bool) error {
	var cmd string
	if disabled {
		cmd = "/ip/firewall/filter/disable"
	} else {
		cmd = "/ip/firewall/filter/enable"
	}
	_, err := r.client.Run(cmd, "=.id="+id)
	if err != nil {
		return fmt.Errorf("firewall filter toggle: %w", err)
	}
	return nil
}
