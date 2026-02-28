package dto

type CreateHotspotUserRequest struct {
	Name            string `json:"name"`
	Password        string `json:"password"`
	Server          string `json:"server,omitempty"`
	Profile         string `json:"profile,omitempty"`
	MacAddress      string `json:"mac_address,omitempty"`
	IPAddress       string `json:"ip_address,omitempty"`
	LimitBytesIn    int64  `json:"limit_bytes_in,omitempty"`
	LimitBytesOut   int64  `json:"limit_bytes_out,omitempty"`
	LimitBytesTotal int64  `json:"limit_bytes_total,omitempty"`
	LimitUptime     string `json:"limit_uptime,omitempty"`
	Comment         string `json:"comment,omitempty"`
}

type HotspotUserResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Server          string `json:"server"`
	Profile         string `json:"profile"`
	MacAddress      string `json:"mac_address,omitempty"`
	IPAddress       string `json:"ip_address,omitempty"`
	Comment         string `json:"comment,omitempty"`
	Disabled        bool   `json:"disabled"`
	LimitBytesIn    int64  `json:"limit_bytes_in,omitempty"`
	LimitBytesOut   int64  `json:"limit_bytes_out,omitempty"`
	LimitBytesTotal int64  `json:"limit_bytes_total,omitempty"`
	LimitUptime     string `json:"limit_uptime,omitempty"`
	Uptime          string `json:"uptime,omitempty"`
	BytesIn         int64  `json:"bytes_in,omitempty"`
	BytesOut        int64  `json:"bytes_out,omitempty"`
}

type ListHotspotUserResponse struct {
	Users []HotspotUserResponse `json:"users"`
	Total int                   `json:"total"`
}

type HotspotActiveResponse struct {
	ID          string `json:"id"`
	Server      string `json:"server"`
	User        string `json:"user"`
	Address     string `json:"address"`
	MacAddress  string `json:"mac_address"`
	LoginBy     string `json:"login_by"`
	Uptime      string `json:"uptime"`
	IdleTime    string `json:"idle_time,omitempty"`
	BytesIn     int64  `json:"bytes_in"`
	BytesOut    int64  `json:"bytes_out"`
}

type ListHotspotActiveResponse struct {
	Active []HotspotActiveResponse `json:"active"`
	Total  int                     `json:"total"`
}

type HotspotServerResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Interface   string `json:"interface"`
	AddressPool string `json:"address_pool"`
	Profile     string `json:"profile"`
	Disabled    bool   `json:"disabled"`
}

type ListHotspotServerResponse struct {
	Servers []HotspotServerResponse `json:"servers"`
	Total   int                     `json:"total"`
}
