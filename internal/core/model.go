package core

import "time"

type Job struct {
	ID       string
	CostMS   int
	Payload  []byte
	Enqueued time.Time
}

type WorkerInfo struct {
	ID       string
	Addr     string // http://host:port
	LastSeen time.Time

	LoadEMA  float64
	InFlight int
}

type Policy string

const (
	PolicyRR Policy = "rr"
	PolicyLeastLoad Policy = "ll"
	PolicyP2C       Policy = "p2c" 
)
