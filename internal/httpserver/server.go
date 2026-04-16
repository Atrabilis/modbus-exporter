package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/atrabilis/modbus-exporter/internal/store"
)

type Server struct {
	addr             string
	store            *store.Store
	sampleAgeEnabled bool
}

func New(addr string, store *store.Store, sampleAgeEnabled bool) *Server {
	return &Server{
		addr:             addr,
		store:            store,
		sampleAgeEnabled: sampleAgeEnabled,
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
		labelSet := func() string {
			labels := map[string]string{
				"device":      sm.Device,
				"slave":       fmt.Sprintf("%d", sm.SlaveID),
				"slave_name":  sm.SlaveName,
				"register":    fmt.Sprintf("%d", sm.Register),
				"name":        sm.Name,
				"unit":        sm.Unit,
				"ip_address":  sm.IpAddress,
			}

			for k, v := range sm.DeviceLabels {
				labels["device_label_"+k] = v
			}
			for k, v := range sm.SlaveLabels {
				labels["slave_label_"+k] = v
			}

			return formatLabels(labels)
		}()

		if sm.StringValue != nil {
			fmt.Fprintf(w, "modbus_register_info{%s,value=%q} 1\n", labelSet, *sm.StringValue)
		} else {
			metricName := "modbus_value"
			if sm.MetricName != "" {
				metricName = sm.MetricName
			}
			fmt.Fprintf(w, "%s{%s} %f\n", metricName, labelSet, sm.Value)
		}
	}


	if s.sampleAgeEnabled {
		for _, sm := range samples {
			labelSet := map[string]string{
				"device":      sm.Device,
				"slave":       fmt.Sprintf("%d", sm.SlaveID),
				"slave_name":  sm.SlaveName,
				"register":    fmt.Sprintf("%d", sm.Register),
				"ip_address":  sm.IpAddress,
			}

			for k, v := range sm.DeviceLabels {
				labelSet["device_label_"+k] = v
			}
			for k, v := range sm.SlaveLabels {
				labelSet["slave_label_"+k] = v
			}

			age := now.Sub(sm.Timestamp).Seconds()
			fmt.Fprintf(
				w,
				"modbus_sample_age_seconds{%s} %f\n",
				formatLabels(labelSet),
				age,
			)
		}
	}
}

func formatLabels(labels map[string]string) string {
	first := true
	buf := ""
	for k, v := range labels {
		if !first {
			buf += ","
		}
		first = false
		buf += fmt.Sprintf("%s=\"%s\"", k, v)
	}
	return buf
}
