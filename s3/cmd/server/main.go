package main

import (
	"net/http"
	"strings"

	"github.com/tickloop/kilo/internal/common"
	"github.com/tickloop/kilo/internal/config"
	"github.com/tickloop/kilo/internal/metadata"
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
	service_mux := common.Chain(metadata.NewRouter(cfg), common.MVerifyContentTypeHeader)

	// main router
	router := http.NewServeMux()
	router.Handle("/api/v1/", http.StripPrefix("/api/v1", service_mux))
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	if err := server.ListenAndServe(); err != nil {
		cfg.Logger.Error(err.Error())
	}
}

func main() {
	// Load banner on server start
	cfg := config.LoadConfig()
	loadBanner(cfg)

	// Start the server
	startServer(cfg)
}
