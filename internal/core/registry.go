package core

import (
	"sync"
	"time"
)

type Registry struct {
	mu      sync.RWMutex
	workers []WorkerInfo
	loadEMA map[string]float64   // workerID -> EMA ms
	infl    map[string]int       // workerID -> inflight
	alpha   float64     
}

const defaultEMAms = 75.0


func NewRegistry() *Registry {
	return &Registry{
		loadEMA: make(map[string]float64),
		infl:    make(map[string]int),
		alpha:   0.8,
	}
}

func (r *Registry) Upsert(w WorkerInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w.LastSeen = time.Now()
	for i := range r.workers {
		if r.workers[i].ID == w.ID {
			r.workers[i] = w
			return
		}
	}
	r.workers = append(r.workers, w)

	if _, ok := r.loadEMA[w.ID]; !ok {
		r.loadEMA[w.ID] = defaultEMAms
	}

}

func (r *Registry) Snapshot() []WorkerInfo {

	r.mu.RLock()
	defer r.mu.RUnlock()

	cp := make([]WorkerInfo, len(r.workers))


	for i, w := range r.workers {
		w.LoadEMA = r.loadEMA[w.ID]
		w.InFlight = r.infl[w.ID]
		cp[i] = w
	}


	return cp

}

func (r *Registry) MarkStart(id string) {
	r.mu.Lock()
	r.infl[id]++
	r.mu.Unlock()
}

func (r *Registry) MarkFinish(id string, durMS int) {
	r.mu.Lock()
	if r.infl[id] > 0 {
		r.infl[id]--
	}

	old := r.loadEMA[id]
	alpha := r.alpha

	newEMA := alpha*old + (1.0-alpha)*float64(durMS)
	r.loadEMA[id] = newEMA

	r.mu.Unlock()
}
