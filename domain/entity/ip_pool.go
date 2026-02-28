package entity

type IPPool struct {
	ID       string
	Name     string
	Ranges   string
	NextPool string
	Comment  string
}

type IPPoolUsed struct {
	Pool    string
	Address string
	Owner   string
	Info    string
}
