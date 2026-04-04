package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/tickloop/kilo/internal/common"
	"github.com/tickloop/kilo/internal/config"
	"github.com/tickloop/kilo/internal/metadata"
	"github.com/tickloop/kilo/internal/storage"
)

func loadBanner(cfg *config.Config) {
	_banner := `███████      ████████  ████████████   ███████                  █████  █████
███████     ███████    ████████████   ███████                ███████  ███████
███████    ███████     ████████████   ███████              █████████  █████████
███████  ████████        ███████      ███████             ████████      ███████
███████ ███████          ███████      ███████             ████████      ███████
███████ ██████           ███████      ███████             ████████      ███████
███████ █████ ██         ███████      ███████             ████████      ███████
███████ ███  ████        ███████      ███████             ████████      ███████
███████ ██ ███████       ███████      ███████     ██████  ████████      ███████
███████ █  ████████    ████████████   ███████ ██████████   █████████  █████████
███████     ████████   ████████████   ███████ ██████████     ███████  ███████`

	for _, line := range strings.Split(_banner, "\n") {
		cfg.Logger.Info(line)
	}
}

func startServer(cfg *config.Config) {
	cfg.Logger.Info("Starting server...")
	var mux http.Handler

	switch cfg.ServiceType {
	case "METADATA":
		cfg.Logger.Info("Running metadata service...")
		mux = (&metadata.MetadataService_v1{}).NewServeMux()
	case "STORAGE":
		cfg.Logger.Info("Running storage service...")
		mux = (&storage.StorageService_v1{}).NewServeMux(cfg)
	default:
		cfg.Logger.Warn("Default case triggered in switch - should be impossible")
	}

	// main router
	// router := http.NewServeMux()
	// router.Handle("/api/v1/", http.StripPrefix("/api/v1", common.Chain(mux, common.MVerifyContentTypeHeader)))
	server := &http.Server{
		Addr: ":8080",
		Handler: common.Chain(
			mux,
			common.MLogPath,
		),
	}
	if err := server.ListenAndServe(); err != nil {
		cfg.Logger.Error(err.Error())
	}
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("ERR starting application\n%s", err.Error())
	}

	// Load banner on server start
	loadBanner(cfg)

	// Start the server
	startServer(cfg)
}
