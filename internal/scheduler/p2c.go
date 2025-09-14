package scheduler

import (
	"math"
	"math/rand"
	"time"

	"github.com/enesyesil/parallax/internal/core"
)


type P2C struct {
	D int 
}


func (s *P2C) Name() string { 
	if s.D <= 1 {
		return "p1"
	}
	if s.D == 2 {
		return "p2c"
	}
	return "jbsq_d"
}


func (s *P2C) Choose(ws []core.WorkerInfo) *core.WorkerInfo {
	n := len(ws)
	if n == 0 {
		return nil
	}

	if n == 1 || s.D <= 1 {
		return &ws[0]
	}

	d := s.D

	if d > n {
		d = n
	}
	
	seen := make(map[int]struct{}, d)
	bestIdx := -1

	for picked := 0; picked < d; {
		i := rand.Intn(n) 

		if _, ok := seen[i]; ok {
			continue
		}

		seen[i] = struct{}{}
		
		if bestIdx == -1 {
			bestIdx = i
		} else {
			bestIdx = betterIdx(ws, bestIdx, i)
		}

		picked++

	}

	return &ws[bestIdx]

}

func effEMA(x float64) float64 {
	
	if x <= 0 {
		return math.MaxFloat64
	}
	return x
}

func betterIdx(ws []core.WorkerInfo, a, b int) int {

	wa, wb := ws[a], ws[b]

	if wa.InFlight < wb.InFlight { return a }

	if wa.InFlight > wb.InFlight { return b }

	ea, eb := effEMA(wa.LoadEMA), effEMA(wb.LoadEMA)

	if ea < eb { return a }

	if ea > eb { return b }
	
	if rand.Intn(2) == 0 {
		 return a 
		}

	return b
}

func init() {
	
	_ = time.Now 
}
