package dto

type CreateSimpleQueueRequest struct {
	Name           string `json:"name"`
	Target         string `json:"target"`
	MaxLimit       string `json:"max_limit,omitempty"`
	LimitAt        string `json:"limit_at,omitempty"`
	BurstLimit     string `json:"burst_limit,omitempty"`
	BurstThreshold string `json:"burst_threshold,omitempty"`
	BurstTime      string `json:"burst_time,omitempty"`
	Priority       int    `json:"priority,omitempty"`
	Parent         string `json:"parent,omitempty"`
	Comment        string `json:"comment,omitempty"`
}

type SimpleQueueResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Target         string `json:"target"`
	MaxLimit       string `json:"max_limit"`
	LimitAt        string `json:"limit_at,omitempty"`
	BurstLimit     string `json:"burst_limit,omitempty"`
	BurstThreshold string `json:"burst_threshold,omitempty"`
	BurstTime      string `json:"burst_time,omitempty"`
	Parent         string `json:"parent,omitempty"`
	Priority       int    `json:"priority"`
	Comment        string `json:"comment,omitempty"`
	Disabled       bool   `json:"disabled"`
	Rate           string `json:"rate,omitempty"`
}

type ListSimpleQueueResponse struct {
	Queues []SimpleQueueResponse `json:"queues"`
	Total  int                   `json:"total"`
}

type CreateQueueTreeRequest struct {
	Name       string `json:"name"`
	Parent     string `json:"parent"`
	PacketMark string `json:"packet_mark,omitempty"`
	MaxLimit   string `json:"max_limit,omitempty"`
	LimitAt    string `json:"limit_at,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

type QueueTreeResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Parent     string `json:"parent"`
	PacketMark string `json:"packet_mark,omitempty"`
	MaxLimit   string `json:"max_limit"`
	Priority   int    `json:"priority"`
	Comment    string `json:"comment,omitempty"`
	Disabled   bool   `json:"disabled"`
}

type ListQueueTreeResponse struct {
	Queues []QueueTreeResponse `json:"queues"`
	Total  int                 `json:"total"`
}
