package entity

type SimpleQueue struct {
	ID             string
	Name           string
	Target         string
	DstAddress     string
	Parent         string
	Priority       int
	Queue          string
	MaxLimit       string
	LimitAt        string
	BurstLimit     string
	BurstThreshold string
	BurstTime      string
	PacketMarks    string
	Comment        string
	Disabled       bool
	Dynamic        bool
	Invalid        bool
	Rate           string
	BytesTotal     int64
	PacketsTotal   int64
	Dropped        int64
}

type QueueTree struct {
	ID             string
	Name           string
	Parent         string
	PacketMark     string
	Priority       int
	MaxLimit       string
	LimitAt        string
	BurstLimit     string
	BurstThreshold string
	BurstTime      string
	Queue          string
	Comment        string
	Disabled       bool
}
