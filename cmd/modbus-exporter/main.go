package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/atrabilis/modbus-exporter/internal/config"
	"github.com/goburrow/modbus"
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

	log.Printf("config loaded, poll_interval=%s", cfg.PollInterval)

	// --- VERY SIMPLE MODBUS TEST ---
	for _, dev := range cfg.Devices {
		if dev.Protocol != "modbus-tcp" {
			log.Printf("device %s: unsupported protocol %s", dev.Name, dev.Protocol)
			continue
		}

		address := fmt.Sprintf("%s:%d", dev.Address, dev.Port)
		log.Printf("connecting to device %s at %s", dev.Name, address)

		handler := modbus.NewTCPClientHandler(address)
		handler.Timeout = dev.Timeout
		handler.SlaveId = 1 // will iterate later
		handler.IdleTimeout = 10 * time.Second

		if err := handler.Connect(); err != nil {
			log.Printf("device %s: connect error: %v", dev.Name, err)
			continue
		}
		defer handler.Close()

		client := modbus.NewClient(handler)

		// Take the FIRST slave and FIRST register only (for now)
		slave := dev.Slaves[0]
		handler.SlaveId = byte(slave.ID)

		reg := slave.Registers[0]
		log.Printf(
			"reading device=%s slave=%d register=%d function=%d",
			dev.Name,
			slave.ID,
			reg.Address,
			reg.Function,
		)

		var raw []byte

		switch reg.Function {
		case 3:
			raw, err = client.ReadHoldingRegisters(reg.Address, 2)
		case 4:
			raw, err = client.ReadInputRegisters(reg.Address, 2)
		default:
			log.Printf("unsupported function code %d", reg.Function)
			continue
		}

		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}

		log.Printf("raw bytes: %x", raw)
	}

	log.Printf("modbus test finished")
}
