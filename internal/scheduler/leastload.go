package scheduler

import "github.com/enesyesil/parallax/internal/core"

type LeastLoad struct{}

func (s *LeastLoad) Name() string {

	 return "ll" 
	
	}

func (s *LeastLoad) Choose(ws []core.WorkerInfo) *core.WorkerInfo {

	if len(ws) == 0 {
		return nil
	}

	best := ws[0]

	// heuristic: prefer lower inflight first, then lower EMA
	for _, w := range ws[1:] {

		if w.InFlight < best.InFlight || (w.InFlight == best.InFlight && w.LoadEMA < best.LoadEMA) {
			best = w
			
		}
	}
	return &best
}
