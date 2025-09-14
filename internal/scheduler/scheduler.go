package scheduler

import "github.com/enesyesil/parallax/internal/core"

type Scheduler interface {
	Name() string
	Choose([]core.WorkerInfo) *core.WorkerInfo
}
