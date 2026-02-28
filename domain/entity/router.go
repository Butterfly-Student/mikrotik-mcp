package entity

type Router struct {
	ID       string
	Name     string
	Host     string
	Port     int
	Username string
	UseTLS   bool
	Comment  string
}

type Connection struct {
	RouterID  string
	Connected bool
	Version   string
}
