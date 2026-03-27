package config

import (
	"log/slog"

	"github.com/tickloop/kilo/internal/common"
)

type Config struct {
	Logger *slog.Logger;
}


func LoadConfig() *Config {
	cfg := Config{
		Logger: common.NewLogger(),
	}
	return &cfg;
}
