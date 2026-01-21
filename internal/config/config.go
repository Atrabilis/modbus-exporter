package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PollInterval time.Duration `yaml:"poll_interval"`
	Devices      []Device      `yaml:"devices"`
}

type Device struct {
	Name     string        `yaml:"name"`
	Protocol string        `yaml:"protocol"`
	Address  string        `yaml:"address"`
	Port     int           `yaml:"port"`
	Timeout  time.Duration `yaml:"timeout"`
	Slaves   []Slave       `yaml:"slaves"`
}

type Slave struct {
	ID        int        `yaml:"id"`
	Name      string     `yaml:"name"`
	Registers []Register `yaml:"registers"`
}

type Register struct {
	Address  uint16 `yaml:"address"`
	Function uint8  `yaml:"function"`
	Name     string `yaml:"name"`
	Datatype string `yaml:"datatype"`
	Unit     string `yaml:"unit"`
}

// Load reads a YAML config file from path and unmarshals it into Config.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal yaml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate performs minimal sanity checks.
// Keep this strict but small; semantics come later.
func (c *Config) validate() error {
	if c.PollInterval <= 0 {
		return fmt.Errorf("config: poll_interval must be > 0")
	}

	if len(c.Devices) == 0 {
		return fmt.Errorf("config: at least one device must be defined")
	}

	for _, d := range c.Devices {
		if d.Name == "" {
			return fmt.Errorf("config: device name is required")
		}
		if d.Protocol == "" {
			return fmt.Errorf("config: device %q: protocol is required", d.Name)
		}
		if d.Address == "" {
			return fmt.Errorf("config: device %q: address is required", d.Name)
		}
		if d.Port <= 0 {
			return fmt.Errorf("config: device %q: invalid port", d.Name)
		}
		if d.Timeout <= 0 {
			return fmt.Errorf("config: device %q: timeout must be > 0", d.Name)
		}
		if len(d.Slaves) == 0 {
			return fmt.Errorf("config: device %q: at least one slave required", d.Name)
		}

		for _, s := range d.Slaves {
			if s.ID < 0 {
				return fmt.Errorf("config: device %q: slave id must be >= 0", d.Name)
			}
			if s.Name == "" {
				return fmt.Errorf("config: device %q: slave name is required", d.Name)
			}
			if len(s.Registers) == 0 {
				return fmt.Errorf("config: device %q slave %q: no registers defined", d.Name, s.Name)
			}

			for _, r := range s.Registers {
				if r.Name == "" {
					return fmt.Errorf("config: device %q slave %q: register name is required", d.Name, s.Name)
				}
				if r.Function == 0 {
					return fmt.Errorf("config: device %q slave %q: register %q: function is required", d.Name, s.Name, r.Name)
				}
			}
		}
	}

	return nil
}
