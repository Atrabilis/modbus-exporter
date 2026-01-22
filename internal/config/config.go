package config

import (
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
	Flags    []string      `yaml:"flags,omitempty"`
	Slaves   []Slave       `yaml:"slaves"`
}

type Slave struct {
	Name      string     `yaml:"name"`
	SlaveID   int        `yaml:"slave_id"`
	Offset    int        `yaml:"offset"`
	Registers []Register `yaml:"modbus_registers"`
}

type Register struct {
	Register     int          `yaml:"register"`
	FunctionCode int          `yaml:"function_code"`
	Name         string       `yaml:"name"`
	Description  string       `yaml:"description"`
	Words        int          `yaml:"words"`
	Datatype     string       `yaml:"datatype"`
	Unit         string       `yaml:"unit"`
	Gain         float64      `yaml:"gain"`
	Flags        RegisterFlag `yaml:"flags,omitempty"`
}

type RegisterFlag struct {
	ModuleNumber int    `yaml:"module_number,omitempty"`
	ModuleLabel  string `yaml:"module_label,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
