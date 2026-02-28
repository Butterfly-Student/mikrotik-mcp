package dto

type InterfaceResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	MacAddress string `json:"mac_address"`
	MTU        int    `json:"mtu"`
	Running    bool   `json:"running"`
	Disabled   bool   `json:"disabled"`
	Comment    string `json:"comment,omitempty"`
}

type ListInterfaceResponse struct {
	Interfaces []InterfaceResponse `json:"interfaces"`
	Total      int                 `json:"total"`
}

type WatchTrafficRequest struct {
	Interface string `json:"interface"`
	Seconds   int    `json:"seconds"`
}

type TrafficStatResponse struct {
	Interface          string `json:"interface"`
	RxBps              int64  `json:"rx_bps"`
	TxBps              int64  `json:"tx_bps"`
	RxPacketsPerSecond int64  `json:"rx_pps,omitempty"`
	TxPacketsPerSecond int64  `json:"tx_pps,omitempty"`
	Timestamp          string `json:"timestamp"`
}

type WatchTrafficResponse struct {
	Interface string                `json:"interface"`
	Samples   []TrafficStatResponse `json:"samples"`
	Duration  int                   `json:"duration_seconds"`
}
