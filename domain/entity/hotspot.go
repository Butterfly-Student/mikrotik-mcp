package entity

type HotspotServer struct {
	ID               string
	Name             string
	Interface        string
	AddressPool      string
	Profile          string
	IdleTimeout      string
	KeepaliveTimeout string
	LoginTimeout     string
	AddressesPerMac  int
	Disabled         bool
	Invalid          bool
	HTTPS            bool
}

type HotspotUser struct {
	ID              string
	Name            string
	Server          string
	Profile         string
	MacAddress      string
	IPAddress       string
	Comment         string
	Disabled        bool
	LimitBytesIn    int64
	LimitBytesOut   int64
	LimitBytesTotal int64
	LimitUptime     string
	Uptime          string
	BytesIn         int64
	BytesOut        int64
	BytesTotal      int64
}

type HotspotUserProfile struct {
	ID              string
	Name            string
	RateLimit       string
	SharedUsers     int
	IdleTimeout     string
	KeepaliveTimeout string
	SessionTimeout  string
	AddressPool     string
}

type HotspotActive struct {
	ID               string
	Server           string
	User             string
	Domain           string
	Address          string
	MacAddress       string
	LoginBy          string
	Uptime           string
	IdleTime         string
	SessionTimeLeft  string
	BytesIn          int64
	BytesOut         int64
	PacketsIn        int64
	PacketsOut       int64
}

type HotspotHost struct {
	ID         string
	Address    string
	MacAddress string
	Server     string
	ToAddress  string
	Status     string
	Authorized bool
	Bypassed   bool
	Uptime     string
	BytesIn    int64
	BytesOut   int64
}
