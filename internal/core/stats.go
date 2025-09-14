package core

import (
	"sort"
	"sync"
	"time"
)


type Stats struct {
	mu    sync.Mutex
	
	recent []sample
	
	window time.Duration
}

type sample struct {
	t  time.Time
	ls float64 
}

func NewStats(window time.Duration) *Stats {
	if window <= 0 {
		window = 10 * time.Second
	}
	return &Stats{window: window}
}


func (s *Stats) Add(t time.Time, latencySeconds float64) {
	cut := t.Add(-s.window)
	s.mu.Lock()
	// drop old
	i := 0
	for ; i < len(s.recent); i++ {
		if s.recent[i].t.After(cut) {
			break
		}
	}
	if i > 0 {
		s.recent = append([]sample{}, s.recent[i:]...)
	}
	
	s.recent = append(s.recent, sample{t: t, ls: latencySeconds})
	s.mu.Unlock()
}


func (s *Stats) Snapshot(now time.Time) (rps float64, p50, p95 float64, count int) {
	cut := now.Add(-s.window)
	s.mu.Lock()
	defer s.mu.Unlock()

	
	i := 0
	for ; i < len(s.recent); i++ {
		if s.recent[i].t.After(cut) {
			break
		}
	}
	if i > 0 {
		s.recent = append([]sample{}, s.recent[i:]...)
	}
	n := len(s.recent)
	count = n
	if n == 0 {
		return 0, 0, 0, 0
	}
	
	rps = float64(n) / s.window.Seconds()

	
	vals := make([]float64, n)
	for i := range s.recent {
		vals[i] = s.recent[i].ls
	}
	sort.Float64s(vals)

	idx50 := int(0.50*float64(n) + 0.5) - 1
	if idx50 < 0 {
		idx50 = 0
	}
	idx95 := int(0.95*float64(n) + 0.5) - 1
	if idx95 < 0 {
		idx95 = 0
	}
	p50, p95 = vals[idx50], vals[idx95]
	return
}
