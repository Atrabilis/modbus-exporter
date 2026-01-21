package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/atrabilis/modbus-exporter/internal/config"
	imodbus "github.com/atrabilis/modbus-exporter/internal/modbus"
	gomodbus "github.com/goburrow/modbus"
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

	for _, dev := range cfg.Devices {
		if dev.Protocol != "modbus-tcp" {
			log.Printf("device %s: unsupported protocol %s", dev.Name, dev.Protocol)
			continue
		}

		addr := fmt.Sprintf("%s:%d", dev.Address, dev.Port)
		log.Printf("connecting to device %s at %s", dev.Name, addr)

		handler := gomodbus.NewTCPClientHandler(addr)
		handler.Timeout = dev.Timeout
		handler.IdleTimeout = 10 * time.Second

		if err := handler.Connect(); err != nil {
			log.Printf("device %s: connect error: %v", dev.Name, err)
			continue
		}
		defer handler.Close()

		client := gomodbus.NewClient(handler)

		for _, slave := range dev.Slaves {
			handler.SlaveId = byte(slave.ID)

			for _, reg := range slave.Registers {
				effective := reg.Address - slave.Offset
				if effective < 0 {
					log.Printf(
						"device=%s slave=%d register=%d offset=%d -> negative effective address",
						dev.Name,
						slave.ID,
						reg.Address,
						slave.Offset,
					)
					continue
				}

				log.Printf(
					"reading device=%s slave=%d register=%d (effective=%d) words=%d function=%d datatype=%s",
					dev.Name,
					slave.ID,
					reg.Address,
					effective,
					reg.Words,
					reg.Function,
					reg.Datatype,
				)

				var raw []byte

				switch reg.Function {
				case 3:
					raw, err = client.ReadHoldingRegisters(
						uint16(effective),
						uint16(reg.Words),
					)
				case 4:
					raw, err = client.ReadInputRegisters(
						uint16(effective),
						uint16(reg.Words),
					)
				default:
					log.Printf("unsupported function code %d", reg.Function)
					continue
				}

				if err != nil {
					log.Printf(
						"read error device=%s slave=%d register=%d: %v",
						dev.Name,
						slave.ID,
						reg.Address,
						err,
					)
					continue
				}

				var value float64

				switch reg.Datatype {
				case "F32BE":
					value = float64(imodbus.F32BE(raw))
				case "F32LE":
					value = float64(imodbus.F32LE(raw))
				case "F32CDAB":
					value = float64(imodbus.F32CDAB(raw))
				case "F32BADC":
					value = float64(imodbus.F32BADC(raw))
				case "S64BE":
					value = float64(imodbus.S64BE(raw))
				case "U64BE":
					value = float64(imodbus.U64BE(raw))
				case "F64BE":
					value = imodbus.F64BE(raw)
				default:
					log.Printf("unsupported datatype %q", reg.Datatype)
					continue
				}

				log.Printf(
					"value device=%s slave=%d %s = %.6f %s",
					dev.Name,
					slave.ID,
					reg.Name,
					value,
					reg.Unit,
				)
			}
		}
	}

	log.Printf("modbus test finished")
}
