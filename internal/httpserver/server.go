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
		fmt.Sprintf("device=%s", labelValue(sm.Device)),
		fmt.Sprintf("slave=%s", labelValue(strconv.Itoa(sm.SlaveID))),
		fmt.Sprintf("slave_name=%s", labelValue(sm.SlaveName)),
		fmt.Sprintf("register=%s", labelValue(strconv.Itoa(sm.Register))),
		fmt.Sprintf("name=%s", labelValue(sm.Name)),
		fmt.Sprintf("unit=%s", labelValue(sm.Unit)),
		fmt.Sprintf("ip_address=%s", labelValue(sm.IpAddress)),
	}

	if sm.ModuleNumber != 0 {
		labels = append(labels, fmt.Sprintf("module_number=%s", labelValue(strconv.Itoa(sm.ModuleNumber))))
	}
	if sm.ModuleLabel != "" {
		labels = append(labels, fmt.Sprintf("module_label=%s", labelValue(sm.ModuleLabel)))
	}
	if includeValue && stringValue != nil {
		labels = append(labels, fmt.Sprintf("value=%s", labelValue(*stringValue)))
	}

	return strings.Join(labels, ",")
}

func buildAgeMetricLabels(sm store.Sample) string {
	labels := []string{
		fmt.Sprintf("device=%s", labelValue(sm.Device)),
		fmt.Sprintf("slave=%s", labelValue(strconv.Itoa(sm.SlaveID))),
		fmt.Sprintf("slave_name=%s", labelValue(sm.SlaveName)),
		fmt.Sprintf("register=%s", labelValue(strconv.Itoa(sm.Register))),
		fmt.Sprintf("ip_address=%s", labelValue(sm.IpAddress)),
	}

	if sm.ModuleNumber != 0 {
		labels = append(labels, fmt.Sprintf("module_number=%s", labelValue(strconv.Itoa(sm.ModuleNumber))))
	}
	if sm.ModuleLabel != "" {
		labels = append(labels, fmt.Sprintf("module_label=%s", labelValue(sm.ModuleLabel)))
	}

	return strings.Join(labels, ",")
}

// labelEscaper applies the only escapes the Prometheus text format accepts in
// label values: backslash, double quote and line feed.
var labelEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

// labelValue quotes a label value for the Prometheus text format. Go's %q is
// not valid here: it writes escapes like \t or \xff that Prometheus rejects,
// and one bad value makes it drop the whole scrape. Bytes that are not valid
// UTF-8 (e.g. 0xFF padding in device strings) become U+FFFD.
func labelValue(v string) string {
	return `"` + labelEscaper.Replace(strings.ToValidUTF8(v, "\uFFFD")) + `"`
}
