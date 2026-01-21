package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/atrabilis/modbus-exporter/internal/config"
)

func main() {
	configPath := flag.String(
		"config",
		"config/example.yml",
		"Path to configuration file",
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	// Minimal proof that config was parsed correctly
	fmt.Printf("Config loaded successfully\n")
	fmt.Printf("Poll interval: %s\n", cfg.PollInterval)
	fmt.Printf("Devices: %d\n", len(cfg.Devices))

	for _, d := range cfg.Devices {
		fmt.Printf("- device=%s protocol=%s address=%s:%d slaves=%d\n",
			d.Name,
			d.Protocol,
			d.Address,
			d.Port,
			len(d.Slaves),
		)
	}
}
