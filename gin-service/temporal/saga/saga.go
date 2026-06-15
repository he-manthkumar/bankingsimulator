package saga

import "go.temporal.io/sdk/workflow"

type Saga struct {
	compensations []func(workflow.Context) error
}

func (s *Saga) AddCompensation(fn func(workflow.Context) error) {
	s.compensations = append(s.compensations, fn)
}

func (s *Saga) Compensate(ctx workflow.Context) {
	for i := len(s.compensations) - 1; i >= 0; i-- {
		if err := s.compensations[i](ctx); err != nil {
			workflow.GetLogger(ctx).Error("compensation step failed", "step", i, "error", err)
		}
	}
}
