package scheduler

import "github.com/enesyesil/parallax/internal/core"

type RRScheduler struct{
	 i int 
	
	}

func (s *RRScheduler) Name() string { 
	return "rr" 

}

func (s *RRScheduler) Choose(ws []core.WorkerInfo) *core.WorkerInfo {
	if len(ws) == 0 {
		return nil
	}
	w := ws[s.i%len(ws)]
	s.i++
	return &w
}
