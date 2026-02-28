package dto

type CreateFirewallRuleRequest struct {
	Chain           string `json:"chain"`
	Action          string `json:"action"`
	SrcAddress      string `json:"src_address,omitempty"`
	DstAddress      string `json:"dst_address,omitempty"`
	SrcAddressList  string `json:"src_address_list,omitempty"`
	DstAddressList  string `json:"dst_address_list,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	SrcPort         string `json:"src_port,omitempty"`
	DstPort         string `json:"dst_port,omitempty"`
	InInterface     string `json:"in_interface,omitempty"`
	OutInterface    string `json:"out_interface,omitempty"`
	ConnectionState string `json:"connection_state,omitempty"`
	Comment         string `json:"comment,omitempty"`
	PlaceBefore     string `json:"place_before,omitempty"`
}

type FirewallRuleResponse struct {
	ID              string `json:"id"`
	Chain           string `json:"chain"`
	Action          string `json:"action"`
	SrcAddress      string `json:"src_address,omitempty"`
	DstAddress      string `json:"dst_address,omitempty"`
	SrcAddressList  string `json:"src_address_list,omitempty"`
	DstAddressList  string `json:"dst_address_list,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	SrcPort         string `json:"src_port,omitempty"`
	DstPort         string `json:"dst_port,omitempty"`
	InInterface     string `json:"in_interface,omitempty"`
	OutInterface    string `json:"out_interface,omitempty"`
	ConnectionState string `json:"connection_state,omitempty"`
	Comment         string `json:"comment,omitempty"`
	Disabled        bool   `json:"disabled"`
	Dynamic         bool   `json:"dynamic"`
	Bytes           int64  `json:"bytes,omitempty"`
	Packets         int64  `json:"packets,omitempty"`
}

type ListFirewallRuleResponse struct {
	Rules []FirewallRuleResponse `json:"rules"`
	Total int                    `json:"total"`
}
