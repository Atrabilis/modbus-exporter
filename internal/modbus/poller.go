package modbus

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/atrabilis/modbus-exporter/internal/config"
	"github.com/atrabilis/modbus-exporter/internal/store"
	gomodbus "github.com/goburrow/modbus"
)

type Poller struct {
	cfg   *config.Config
	store *store.Store
	debug bool
}

func NewPoller(cfg *config.Config, store *store.Store, debug bool) *Poller {
	return &Poller{
		cfg:   cfg,
		store: store,
		debug: debug,
	}
}

func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	// primer poll inmediato
	p.pollOnce()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollOnce()
		}
	}
}

func (p *Poller) pollOnce() {
	for _, dev := range p.cfg.Devices {
		if dev.Protocol != "modbus-tcp" {
			continue
		}

		addr := fmt.Sprintf("%s:%d", dev.Address, dev.Port)

		handler := gomodbus.NewTCPClientHandler(addr)
		handler.Timeout = dev.Timeout
		handler.IdleTimeout = 10 * time.Second

		if err := handler.Connect(); err != nil {
			log.Printf(
				"modbus: device=%s addr=%s connect error: %v",
				dev.Name,
				addr,
				err,
			)
			continue
		}
		if p.debug {
			log.Printf(
				"modbus: device=%s addr=%s connected",
				dev.Name,
				addr,
			)
		}
		client := gomodbus.NewClient(handler)

		for _, slave := range dev.Slaves {
			handler.SlaveId = byte(slave.SlaveID)

			if p.debug {
				log.Printf(
					"modbus: device=%s slave=%d start",
					dev.Name,
					slave.SlaveID,
				)
			}

			for _, reg := range slave.Registers {
				effective := reg.Register - slave.Offset
				if effective < 0 {
					log.Printf(
						"modbus: device=%s slave=%d register=%d offset=%d -> negative effective address",
						dev.Name,
						slave.SlaveID,
						reg.Register,
						slave.Offset,
					)
					continue
				}

				if p.debug {
					log.Printf(
						"modbus: device=%s slave=%d register=%d effective=%d words=%d function=%d",
						dev.Name,
						slave.SlaveID,
						reg.Register,
						effective,
						reg.Words,
						reg.FunctionCode,
					)
				}

				var raw []byte
				var err error

				switch reg.FunctionCode {
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
					continue
				}

				if err != nil {
					log.Printf(
						"modbus: read error device=%s slave=%d register=%d: %v",
						dev.Name,
						slave.SlaveID,
						reg.Register,
						err,
					)
					continue
				}

				value, ok := decode(reg.Datatype, raw)
				if !ok {
					log.Printf(
						"modbus: unsupported datatype device=%s slave=%d register=%d datatype=%s",
						dev.Name,
						slave.SlaveID,
						reg.Register,
						reg.Datatype,
					)
					continue
				}

				value *= reg.Gain

				if p.debug {
					log.Printf(
						"modbus: device=%s slave=%d register=%d name=%s value=%.6f %s",
						dev.Name,
						slave.SlaveID,
						reg.Register,
						reg.Name,
						value,
						reg.Unit,
					)
				}

				p.store.Set(store.Sample{
					Value:     value,
					Timestamp: time.Now().UTC(),
					Device:    dev.Name,
					SlaveID:   slave.SlaveID,
					Register:  reg.Register,
					Name:      reg.Name,
					Unit:      reg.Unit,
					IpAddress: dev.Address,
				})
			}
		}

		handler.Close()
	}
}

func decode(datatype string, raw []byte) (float64, bool) {
	switch datatype {

	// ---- Integer ----
	case "U8":
		return float64(U8(raw)), true

	case "U16":
		return float64(U16(raw)), true

	case "S16":
		return float64(S16(raw)), true

	case "U32":
		return float64(U32(raw)), true

	case "S32":
		return float64(S32(raw)), true

	case "U32LE":
		return float64(U32LE(raw)), true

	case "S32LE":
		return float64(S32LE(raw)), true

	case "U64BE":
		return float64(U64BE(raw)), true

	case "S64BE":
		return float64(S64BE(raw)), true

	// ---- Float ----
	case "F32BE":
		return float64(F32BE(raw)), true

	case "F32LE":
		return float64(F32LE(raw)), true

	case "F32CDAB":
		return float64(F32CDAB(raw)), true

	case "F32BADC":
		return float64(F32BADC(raw)), true

	case "F64BE":
		return F64BE(raw), true

	// ---- String (special case) ----
	case "UTF8":
		// No es numérico → no exportable como gauge
		// pero NO es error de lectura
		return 0, false

	default:
		return 0, false
	}
}
