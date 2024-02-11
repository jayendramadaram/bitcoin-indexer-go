package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type DBConfig struct {
	URI      string `toml:"uri"`
	Database string `toml:"database"`
}

type LoggerOptions struct {
	Level               []string `toml:"level"`
	LogBackTraceEnabled bool     `toml:"log_backtrace_enabled"`
}

type IndexConfig struct {
	HeaderFirstMode bool `toml:"mode"`
}

type Config struct {
	DB          DBConfig      `toml:"db"`
	Logger      LoggerOptions `toml:"logger"`
	IndexConfig IndexConfig   `toml:"indexCfg"`
}

func LoadConfig(path string) (*Config, error) {

	var config Config
	metaData, err := toml.DecodeFile(path, &config)
	if err != nil {
		return nil, err
	}

	if len(metaData.Undecoded()) > 0 {
		return nil, (fmt.Errorf("undecoded fields: %v", metaData.Undecoded()))
	}

	return &config, nil
}
