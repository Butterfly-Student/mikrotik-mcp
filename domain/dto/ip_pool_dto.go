package dto

type CreateIPPoolRequest struct {
	Name     string `json:"name"`
	Ranges   string `json:"ranges"`
	NextPool string `json:"next_pool,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type UpdateIPPoolRequest struct {
	ID       string `json:"id"`
	Ranges   string `json:"ranges,omitempty"`
	NextPool string `json:"next_pool,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type IPPoolResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Ranges   string `json:"ranges"`
	NextPool string `json:"next_pool,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

type ListIPPoolResponse struct {
	Pools []IPPoolResponse `json:"pools"`
	Total int              `json:"total"`
}

type IPPoolUsedResponse struct {
	Pool    string `json:"pool"`
	Address string `json:"address"`
	Owner   string `json:"owner"`
	Info    string `json:"info,omitempty"`
}

type ListIPPoolUsedResponse struct {
	Used  []IPPoolUsedResponse `json:"used"`
	Total int                  `json:"total"`
}
