package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/tickloop/kilo/internal/common"
)

type Config struct {
	ServiceType string
	Logger      *slog.Logger
}

func LoadConfig() (*Config, error) {
	svc_type, ok := os.LookupEnv("KILO_SERVICE_TYPE")
	if !ok {
		return nil, fmt.Errorf("env variable missing: KILO_SERVICE_TYPE")
	}

	cfg := Config{
		ServiceType: svc_type,
		Logger:      common.NewLogger(),
	}
	return &cfg, nil
}
