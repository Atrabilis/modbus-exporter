package httpserver

import (
	"strings"
	"testing"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// Any label value, whatever bytes a device sends, must produce a line that
// Prometheus accepts and reads back as the same text (invalid UTF-8 replaced
// by U+FFFD).
func FuzzLabelValue(f *testing.F) {
	for _, seed := range []string{
		"A2250644025", "", `back\slash`, `double"quote`, "line\nfeed", "tab\tchar",
		"SN\xff\xff\xff", "\x00\x01\x1f", "carriage\rreturn", "unicode ñ °C", "\\n literal",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, v string) {
		line := "modbus_register_info{value=" + labelValue(v) + "} 1\n"
		parser := expfmt.NewTextParser(model.UTF8Validation)
		families, err := parser.TextToMetricFamilies(strings.NewReader(line))
		if err != nil {
			t.Fatalf("Prometheus rejects %q: %v (line %q)", v, err, line)
		}
		got := families["modbus_register_info"].GetMetric()[0].GetLabel()[0].GetValue()
		if want := strings.ToValidUTF8(v, "�"); got != want {
			t.Fatalf("value %q read back as %q, want %q", v, got, want)
		}
	})
}
