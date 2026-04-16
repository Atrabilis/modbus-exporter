package store

import (
	"strings"
	"sync"
	"time"
)

type Sample struct {
	Value     float64
	Timestamp time.Time

	// Identidad
	Device      string
	SlaveID     int
	SlaveName   string
	Register    int
	Name        string
	Unit        string
	IpAddress   string
	MetricName  string
	DeviceLabels map[string]string
	SlaveLabels  map[string]string

	// StringValue es distinto de nil solo para registros UTF8 (valor en etiqueta, no numérico).
	StringValue *string
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

	// Clonar string para no depender del puntero recibido (p. ej. variable de bucle en el poller).
	if sample.StringValue != nil {
		cloned := strings.Clone(*sample.StringValue)
		sample.StringValue = &cloned
	}
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
