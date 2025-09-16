package core

import (
	"sync"
	"time"
)

type MicroReq struct {
	Cost    int
	Payload string
	RespCh  chan MicroResp
}

type MicroResp struct {
	OK     bool
	TookMS int
	Err    string
}


type Batcher struct {
	mu          sync.Mutex
	pending     []MicroReq
	window      time.Duration
	maxBatch    int
	maxCost     int
	processFunc func(batch []MicroReq) (tookMS int, err error) 
	stopCh      chan struct{}
}

func NewBatcher(window time.Duration, maxBatch, maxCost int, process func([]MicroReq) (int, error)) *Batcher {
	if window <= 0 {
		window = 20 * time.Millisecond
	}
	if maxBatch <= 0 {
		maxBatch = 8
	}
	if maxCost <= 0 {
		maxCost = 480 
	}
	return &Batcher{
		window:      window,
		maxBatch:    maxBatch,
		maxCost:     maxCost,
		processFunc: process,
		stopCh:      make(chan struct{}),
	}
}

func (b *Batcher) Enqueue(r MicroReq) {
	b.mu.Lock()
	b.pending = append(b.pending, r)
	b.mu.Unlock()
}

func (b *Batcher) Run() {
	ticker := time.NewTicker(b.window)
	defer ticker.Stop()
	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.flushWindow()
		}
	}
}

func (b *Batcher) Stop() { close(b.stopCh) }

func (b *Batcher) flushWindow() {
	
	b.mu.Lock()
	local := b.pending
	b.pending = nil
	b.mu.Unlock()
	if len(local) == 0 {
		return
	}


	i := 0
	for i < len(local) {
		bsz := 0
		bcost := 0
		j := i
		for j < len(local) && bsz < b.maxBatch && bcost+local[j].Cost <= b.maxCost {
			bsz++
			bcost += local[j].Cost
			j++
		}
		batch := local[i:j]

	
		tookMS, err := b.processFunc(batch)
		if err != nil {
			for _, req := range batch {
				req.RespCh <- MicroResp{OK: false, Err: err.Error()}
			}
		} else {
			for _, req := range batch {
				req.RespCh <- MicroResp{OK: true, TookMS: tookMS}
			}
		}
		i = j
	}
}
