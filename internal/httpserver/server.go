package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	mux.HandleFunc("/health", s.handleHealth)
	log.Printf("http server listening on %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	samples := s.store.Snapshot()
	now := time.Now()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Comentarios Prometheus: HELP y TYPE para cada métrica.
	fmt.Fprint(w, "# HELP modbus_value Current value of the Modbus register (numeric).\n")
	fmt.Fprint(w, "# TYPE modbus_value gauge\n")
	fmt.Fprint(w, "# HELP modbus_sample_age_seconds Age in seconds since the last successful poll.\n")
	fmt.Fprint(w, "# TYPE modbus_sample_age_seconds gauge\n")
	fmt.Fprint(w, "# HELP modbus_register_info Info from UTF-8 string registers (string value in label).\n")
	fmt.Fprint(w, "# TYPE modbus_register_info gauge\n")

	for _, sm := range samples {
		if sm.StringValue != nil {
			// Registro UTF8: exponer como info (valor en etiqueta, gauge=1).
			fmt.Fprintf(
				w,
				"modbus_register_info{%s} 1\n",
				buildMetricLabels(sm, true, sm.StringValue),
			)
		} else {
			// Registro numérico.
			fmt.Fprintf(
				w,
				"modbus_value{%s} %f\n",
				buildMetricLabels(sm, false, nil),
				sm.Value,
			)
		}
	}

	for _, sm := range samples {
		age := now.Sub(sm.Timestamp).Seconds()
		fmt.Fprintf(
			w,
			"modbus_sample_age_seconds{%s} %f\n",
			buildAgeMetricLabels(sm),
			age,
		)
	}
}

func buildMetricLabels(sm store.Sample, includeValue bool, stringValue *string) string {
	labels := []string{
		fmt.Sprintf("device=%q", sm.Device),
		fmt.Sprintf("slave=%q", strconv.Itoa(sm.SlaveID)),
		fmt.Sprintf("slave_name=%q", sm.SlaveName),
		fmt.Sprintf("register=%q", strconv.Itoa(sm.Register)),
		fmt.Sprintf("name=%q", sm.Name),
		fmt.Sprintf("unit=%q", sm.Unit),
		fmt.Sprintf("ip_address=%q", sm.IpAddress),
	}

	if sm.ModuleNumber != 0 {
		labels = append(labels, fmt.Sprintf("module_number=%q", strconv.Itoa(sm.ModuleNumber)))
	}
	if sm.ModuleLabel != "" {
		labels = append(labels, fmt.Sprintf("module_label=%q", sm.ModuleLabel))
	}
	if includeValue && stringValue != nil {
		labels = append(labels, fmt.Sprintf("value=%q", *stringValue))
	}

	return strings.Join(labels, ",")
}

func buildAgeMetricLabels(sm store.Sample) string {
	labels := []string{
		fmt.Sprintf("device=%q", sm.Device),
		fmt.Sprintf("slave=%q", strconv.Itoa(sm.SlaveID)),
		fmt.Sprintf("slave_name=%q", sm.SlaveName),
		fmt.Sprintf("register=%q", strconv.Itoa(sm.Register)),
		fmt.Sprintf("ip_address=%q", sm.IpAddress),
	}

	if sm.ModuleNumber != 0 {
		labels = append(labels, fmt.Sprintf("module_number=%q", strconv.Itoa(sm.ModuleNumber)))
	}
	if sm.ModuleLabel != "" {
		labels = append(labels, fmt.Sprintf("module_label=%q", sm.ModuleLabel))
	}

	return strings.Join(labels, ",")
}
