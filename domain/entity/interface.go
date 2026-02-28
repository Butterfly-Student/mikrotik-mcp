package entity

import "time"

type NetworkInterface struct {
	ID             string
	Name           string
	Type           string
	MacAddress     string
	MTU            int
	Running        bool
	Disabled       bool
	Dynamic        bool
	Slave          bool
	Comment        string
	LastLinkUpTime string
	LinkDowns      int
	RxByte         int64
	TxByte         int64
	RxPacket       int64
	TxPacket       int64
	RxError        int64
	TxError        int64
	RxDrop         int64
	TxDrop         int64
}

type TrafficStat struct {
	Interface          string
	RxBitsPerSecond    int64
	TxBitsPerSecond    int64
	RxPacketsPerSecond int64
	TxPacketsPerSecond int64
	RxDropsPerSecond   int64
	TxDropsPerSecond   int64
	Timestamp          time.Time
}
