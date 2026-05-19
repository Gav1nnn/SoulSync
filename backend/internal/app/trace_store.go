package app

import "sync"

type TraceStore struct {
	mu     sync.Mutex
	traces []Trace
}

func NewTraceStore() *TraceStore {
	return &TraceStore{
		traces: make([]Trace, 0, 16),
	}
}

func (s *TraceStore) Append(trace Trace) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.traces = append(s.traces, trace)
}
