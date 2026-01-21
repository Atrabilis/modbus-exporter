package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/atrabilis/modbus-exporter/internal/store"
)

type Server struct {
	addr  string
	store *store.Store
}

func New(addr string, store *store.Store) *Server {
	return &Server{
		addr:  addr,
		store: store,
	}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.handleMetrics)

	log.Printf("http server listening on %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	samples := s.store.Snapshot()

	// Content-Type correcto para Prometheus
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Métrica base por registro
	for _, sm := range samples {
		fmt.Fprintf(
			w,
			"modbus_value{device=%q,slave=%q,register=%q,name=%q,unit=%q,ip_address=%q} %f\n",
			sm.Device,
			fmt.Sprintf("%d", sm.SlaveID),
			fmt.Sprintf("%d", sm.Register),
			sm.Name,
			sm.Unit,
			sm.IpAddress,
			sm.Value,
		)
	}

	// Métrica de freshness (opcional pero muy útil)
	now := time.Now()
	for _, sm := range samples {
		age := now.Sub(sm.Timestamp).Seconds()
		fmt.Fprintf(
			w,
			"modbus_sample_age_seconds{device=%q,slave=%q,register=%q,ip_address=%q} %f\n",
			sm.Device,
			fmt.Sprintf("%d", sm.SlaveID),
			fmt.Sprintf("%d", sm.Register),
			sm.IpAddress,
			age,
		)
	}
}
