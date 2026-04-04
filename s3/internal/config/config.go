package config

import (
	"log"
	"log/slog"
	"os"

	"github.com/tickloop/kilo/internal/common"
)

type Config struct {
	DataDir     string
	ServiceType string
	Logger      *slog.Logger
}

func must(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("Missing required env variable: %s", key)
	}
	return val
}

func LoadConfig() (*Config, error) {
	svc_type := must("KILO_SERVICE_TYPE")
	data_dir := must("KILO_DATA_DIR")

	cfg := Config{
		ServiceType: svc_type,
		DataDir:     data_dir,
		Logger:      common.NewLogger(),
	}
	return &cfg, nil
}
