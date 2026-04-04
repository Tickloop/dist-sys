package config

import (
	"log"
	"log/slog"
	"os"

	"github.com/tickloop/kilo/internal/common"
)

const (
	SHARD_SIZE = 64 * 1024
)

type Config struct {
	DataDir           string
	ServiceType       string
	Port              string
	Logger            *slog.Logger
	StorageServiceUrl string
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
	storage_service_url := must("KILO_STORAGE_SERVICE_URL")
	port := must("KILO_PORT")

	cfg := Config{
		ServiceType:       svc_type,
		DataDir:           data_dir,
		Port:              port,
		StorageServiceUrl: storage_service_url,
		Logger:            common.NewLogger(),
	}
	return &cfg, nil
}
