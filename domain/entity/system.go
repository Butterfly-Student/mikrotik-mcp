package entity

type SystemResource struct {
	Uptime           string
	Version          string
	BuildTime        string
	FreeMemory       int64
	TotalMemory      int64
	CPU              string
	CPUCount         int
	CPUFrequency     int
	CPULoad          int
	FreeHDDSpace     int64
	TotalHDDSpace    int64
	ArchitectureName string
	BoardName        string
	Platform         string
}

type SystemLog struct {
	ID      string
	Time    string
	Topics  string
	Message string
}

type SystemIdentity struct {
	Name string
}

type RouterBoard struct {
	IsRouterBoard   bool
	BoardName       string
	Model           string
	SerialNumber    string
	FirmwareType    string
	FactoryFirmware string
	CurrentFirmware string
	UpgradeFirmware string
}
