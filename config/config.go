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

type Config struct {
	DB     DBConfig      `toml:"db"`
	Logger LoggerOptions `toml:"logger"`
}

func LoadConfig(path string) (*Config, error) {

	fmt.Println("config path: ")
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
