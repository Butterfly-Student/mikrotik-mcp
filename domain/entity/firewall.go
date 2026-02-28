package entity

type FirewallRule struct {
	ID         string
	Chain      string
	Action     string
	SrcAddress string
	DstAddress string
	SrcAddressList string
	DstAddressList string
	Protocol   string
	SrcPort    string
	DstPort    string
	InInterface  string
	OutInterface string
	ConnectionState string
	Comment    string
	Disabled   bool
	Dynamic    bool
	Log        bool
	LogPrefix  string
	Bytes      int64
	Packets    int64
}
