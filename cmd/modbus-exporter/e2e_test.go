package main

// End-to-end tests: build the real binary, run it against a fake Modbus TCP
// device (github.com/simonvetter/modbus server, test-only dependency) and
// scrape /metrics. The output is parsed with Prometheus' own text parser
// (github.com/prometheus/common/expfmt), so a format error that would make
// Prometheus reject the whole scrape fails the test.
//
// If MODBUS_EXPORTER_E2E_COVERDIR is set, the binary is built with -cover and
// writes its coverage data there when it exits.

import (
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	mbserver "github.com/simonvetter/modbus"
)

// fakeDevice serves holding and input registers from maps.
type fakeDevice struct {
	mu      sync.Mutex
	holding map[uint16]uint16
	input   map[uint16]uint16
}

func (f *fakeDevice) read(m map[uint16]uint16, addr, qty uint16) []uint16 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]uint16, qty)
	for i := range out {
		out[i] = m[addr+uint16(i)]
	}
	return out
}

func (f *fakeDevice) HandleCoils(*mbserver.CoilsRequest) ([]bool, error) {
	return nil, mbserver.ErrIllegalFunction
}
func (f *fakeDevice) HandleDiscreteInputs(*mbserver.DiscreteInputsRequest) ([]bool, error) {
	return nil, mbserver.ErrIllegalFunction
}
func (f *fakeDevice) HandleHoldingRegisters(r *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if r.IsWrite {
		return nil, mbserver.ErrIllegalFunction
	}
	return f.read(f.holding, r.Addr, r.Quantity), nil
}
func (f *fakeDevice) HandleInputRegisters(r *mbserver.InputRegistersRequest) ([]uint16, error) {
	return f.read(f.input, r.Addr, r.Quantity), nil
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().(*net.TCPAddr).Port
}

func startDevice(t *testing.T, dev *fakeDevice) (int, func()) {
	t.Helper()
	for attempt := 0; attempt < 5; attempt++ {
		port := freePort(t)
		srv, err := mbserver.NewServer(&mbserver.ServerConfiguration{
			URL: fmt.Sprintf("tcp://127.0.0.1:%d", port), Timeout: 30 * time.Second, MaxClients: 10,
			Logger: log.New(io.Discard, "", 0),
		}, dev)
		if err != nil {
			t.Fatal(err)
		}
		if srv.Start() == nil {
			var once sync.Once
			stop := func() { once.Do(func() { _ = srv.Stop() }) }
			t.Cleanup(stop)
			return port, stop
		}
	}
	t.Fatal("could not start the fake Modbus device")
	return 0, nil
}

// text packs a string into big-endian registers, padded with pad bytes.
func text(s string, words int, pad byte) []uint16 {
	b := []byte(s)
	for len(b) < 2*words {
		b = append(b, pad)
	}
	out := make([]uint16, words)
	for i := range out {
		out[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return out
}

func set(m map[uint16]uint16, addr uint16, words ...uint16) {
	for i, w := range words {
		m[addr+uint16(i)] = w
	}
}

type exporter struct {
	cmd  *exec.Cmd
	url  string
	logs *strings.Builder
}

// startExporter builds and runs the binary with a config that polls the fake
// device every 200 ms, and waits until /health answers.
func startExporter(t *testing.T, devicePort int, registers string) *exporter {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "modbus-exporter")
	args := []string{"build", "-o", bin}
	coverDir := os.Getenv("MODBUS_EXPORTER_E2E_COVERDIR")
	if coverDir != "" {
		args = append(args, "-cover", "-coverpkg=github.com/atrabilis/modbus-exporter/...")
	}
	if out, err := exec.Command("go", append(args, ".")...).CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	cfg := fmt.Sprintf(`poll_interval: 200ms
devices:
  - name: "inverter_block"
    protocol: "modbus-tcp"
    address: "127.0.0.1"
    port: %d
    timeout: 500ms
    slaves:
      - name: "inv1"
        slave_id: 1
        offset: 0
        modbus_registers:
%s`, devicePort, registers)
	cfgPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	listen := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	logs := &strings.Builder{}
	cmd := exec.Command(bin, "-config", cfgPath, "-listen", listen)
	cmd.Stdout, cmd.Stderr = logs, logs
	if coverDir != "" {
		cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverDir)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// SIGTERM lets the binary return from main and write coverage data.
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
		}
	})

	e := &exporter{cmd: cmd, url: "http://" + listen, logs: logs}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(e.url + "/health"); err == nil {
			_ = resp.Body.Close()
			return e
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("exporter did not start\n%s", logs.String())
	return nil
}

// scrape returns the parsed metric families, failing the test if Prometheus
// would reject the payload.
func (e *exporter) scrape(t *testing.T) map[string]*dto.MetricFamily {
	t.Helper()
	resp, err := http.Get(e.url + "/metrics")
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("Prometheus would reject this scrape: %v\n%s", err, body)
	}
	return families
}

// waitFor scrapes until cond is true.
func (e *exporter) waitFor(t *testing.T, cond func(map[string]*dto.MetricFamily) bool) map[string]*dto.MetricFamily {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		families := e.scrape(t)
		if cond(families) {
			return families
		}
		if time.Now().After(deadline) {
			t.Fatalf("condition not met before the deadline\n%s", e.logs.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func label(m *dto.Metric, name string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == name {
			return l.GetValue()
		}
	}
	return ""
}

func byName(families map[string]*dto.MetricFamily, family string) map[string]*dto.Metric {
	out := map[string]*dto.Metric{}
	if f := families[family]; f != nil {
		for _, m := range f.GetMetric() {
			out[label(m, "name")+label(m, "register")] = m
		}
	}
	return out
}

func f32(v float32) []uint16 {
	b := math.Float32bits(v)
	return []uint16{uint16(b >> 16), uint16(b)}
}

func TestE2EExposesDecodedValues(t *testing.T) {
	dev := &fakeDevice{holding: map[uint16]uint16{}, input: map[uint16]uint16{}}
	set(dev.holding, 100, 2345)                         // U16, gain 0.1 -> 234.5
	set(dev.holding, 101, 0xFF38)                       // S16 -> -200
	set(dev.holding, 102, 0x0001, 0x1170)               // U32 -> 70000
	set(dev.holding, 104, 0x1170, 0x0001)               // U32LE -> 70000
	set(dev.holding, 106, 0xFFFF, 0xFF38)               // S32 -> -200
	set(dev.holding, 108, f32(50.25)...)                // F32BE
	set(dev.holding, 120, text("A2250644025", 8, 0)...) // serial padded with NUL
	set(dev.input, 200, 7)                              // U16 from input registers
	port, _ := startDevice(t, dev)

	e := startExporter(t, port, `          - {register: 100, function_code: 3, name: ac_voltage, words: 1, datatype: U16, unit: V, gain: 0.1}
          - {register: 101, function_code: 3, name: temperature, words: 1, datatype: S16, unit: C, gain: 1}
          - {register: 102, function_code: 3, name: energy, words: 2, datatype: U32, unit: kWh, gain: 1}
          - {register: 104, function_code: 3, name: energy_le, words: 2, datatype: U32LE, unit: kWh, gain: 1}
          - {register: 106, function_code: 3, name: power, words: 2, datatype: S32, unit: W, gain: 1}
          - {register: 108, function_code: 3, name: frequency, words: 2, datatype: F32BE, unit: Hz, gain: 1}
          - {register: 120, function_code: 3, name: serial_id, words: 8, datatype: UTF-8, unit: ""}
          - {register: 200, function_code: 4, name: status, words: 1, datatype: U16, unit: "", gain: 1}
`)
	families := e.waitFor(t, func(f map[string]*dto.MetricFamily) bool {
		return len(byName(f, "modbus_value")) == 7 && len(byName(f, "modbus_register_info")) == 1
	})

	values := byName(families, "modbus_value")
	want := map[string]float64{
		"ac_voltage100": 234.5, "temperature101": -200, "energy102": 70000, "energy_le104": 70000,
		"power106": -200, "frequency108": 50.25, "status200": 7,
	}
	for key, w := range want {
		m := values[key]
		if m == nil {
			t.Errorf("%s missing", key)
			continue
		}
		if got := m.GetGauge().GetValue(); math.Abs(got-w) > 1e-6 {
			t.Errorf("%s = %v, want %v", key, got, w)
		}
	}
	m := values["ac_voltage100"]
	if label(m, "device") != "inverter_block" || label(m, "slave") != "1" || label(m, "slave_name") != "inv1" ||
		label(m, "unit") != "V" || label(m, "ip_address") != "127.0.0.1" {
		t.Errorf("unexpected labels: %v", m.GetLabel())
	}
	if got := label(byName(families, "modbus_register_info")["serial_id120"], "value"); got != "A2250644025" {
		t.Errorf("serial_id value label = %q, want A2250644025", got)
	}
	if n := len(families["modbus_sample_age_seconds"].GetMetric()); n != 8 {
		t.Errorf("expected an age series per register (8), got %d", n)
	}
}

// Devices pad text registers with 0xFF or spaces and may send control bytes.
// Whatever they send, the scrape must stay valid: one bad label value would
// make Prometheus reject every metric of the exporter.
func TestE2EStringRegisterWithOddBytesKeepsTheScrapeValid(t *testing.T) {
	dev := &fakeDevice{holding: map[uint16]uint16{}, input: map[uint16]uint16{}}
	set(dev.holding, 100, 500)
	set(dev.holding, 120, text("SN\t\"12\\3", 8, 0xFF)...)
	port, _ := startDevice(t, dev)

	e := startExporter(t, port, `          - {register: 100, function_code: 3, name: ac_voltage, words: 1, datatype: U16, unit: V, gain: 1}
          - {register: 120, function_code: 3, name: serial_id, words: 8, datatype: UTF-8, unit: ""}
`)
	families := e.waitFor(t, func(f map[string]*dto.MetricFamily) bool {
		return len(byName(f, "modbus_register_info")) == 1
	})
	if v := byName(families, "modbus_value")["ac_voltage100"]; v == nil || v.GetGauge().GetValue() != 500 {
		t.Fatalf("the numeric register must still be exposed, got %v", v)
	}
	got := label(byName(families, "modbus_register_info")["serial_id120"], "value")
	if !strings.HasPrefix(got, "SN\t\"12\\3") {
		t.Fatalf("the readable part of the string must survive, got %q", got)
	}
}

// A device that stops answering keeps its last value, and the age grows: the
// DataStale alert (Prometheus rule 50_data_freshness.yml) relies on it.
func TestE2EDeviceDownKeepsLastValueAndAgeGrows(t *testing.T) {
	dev := &fakeDevice{holding: map[uint16]uint16{}, input: map[uint16]uint16{}}
	set(dev.holding, 100, 2345)
	port, stop := startDevice(t, dev)

	e := startExporter(t, port, `          - {register: 100, function_code: 3, name: ac_voltage, words: 1, datatype: U16, unit: V, gain: 0.1}
`)
	e.waitFor(t, func(f map[string]*dto.MetricFamily) bool { return len(byName(f, "modbus_value")) == 1 })

	stop()
	time.Sleep(1500 * time.Millisecond) // several poll intervals without the device
	families := e.scrape(t)
	if v := byName(families, "modbus_value")["ac_voltage100"]; v == nil || math.Abs(v.GetGauge().GetValue()-234.5) > 1e-9 {
		t.Fatalf("the last value must stay exposed, got %v", v)
	}
	age := families["modbus_sample_age_seconds"].GetMetric()[0].GetGauge().GetValue()
	if age < 1 {
		t.Fatalf("sample age must grow while the device is down, got %v s", age)
	}
	if !strings.Contains(e.logs.String(), "connect error") && !strings.Contains(e.logs.String(), "read error") {
		t.Fatalf("expected the failed polls to be logged\n%s", e.logs.String())
	}
}
