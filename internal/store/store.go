package store

import (
	"sync"
	"time"
)

type Sample struct {
	Value     float64
	Timestamp time.Time

	// Identidad
	Device    string
	SlaveID   int
	Register  int
	Name      string
	Unit      string
	IpAddress string
}

type Store struct {
	mu sync.RWMutex

	// key: device/slave/register
	samples map[string]Sample
}

func New() *Store {
	return &Store{
		samples: make(map[string]Sample),
	}
}

func key(device string, slaveID int, register int) string {
	return device + "/" + string(rune(slaveID)) + "/" + string(rune(register))
}

func (s *Store) Set(sample Sample) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.samples[key(sample.Device, sample.SlaveID, sample.Register)] = sample
}

func (s *Store) Snapshot() []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Sample, 0, len(s.samples))
	for _, v := range s.samples {
		out = append(out, v)
	}
	return out
}
