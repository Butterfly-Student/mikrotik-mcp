package dto

type SystemResourceResponse struct {
	Uptime           string `json:"uptime"`
	Version          string `json:"version"`
	CPULoad          int    `json:"cpu_load_percent"`
	CPUCount         int    `json:"cpu_count"`
	FreeMemory       string `json:"free_memory"`
	TotalMemory      string `json:"total_memory"`
	FreeHDDSpace     string `json:"free_hdd_space"`
	TotalHDDSpace    string `json:"total_hdd_space"`
	BoardName        string `json:"board_name"`
	ArchitectureName string `json:"architecture"`
	Platform         string `json:"platform"`
}

type GetLogsRequest struct {
	Topics string `json:"topics,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type SystemLogResponse struct {
	ID      string `json:"id"`
	Time    string `json:"time"`
	Topics  string `json:"topics"`
	Message string `json:"message"`
}

type ListSystemLogResponse struct {
	Logs  []SystemLogResponse `json:"logs"`
	Total int                 `json:"total"`
}

type SystemIdentityResponse struct {
	Name string `json:"name"`
}
