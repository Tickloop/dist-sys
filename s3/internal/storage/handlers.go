package storage

import (
	"fmt"
	"io"
	"net/http"

	"github.com/tickloop/kilo/internal/config"
)

type StorageService interface {
	NewServeMux() http.Handler
	GetShard(w http.ResponseWriter, r *http.Request)
	PutShard(w http.ResponseWriter, r *http.Request)
}

type StorageService_v1 struct {
	reg *ShardRegisty_v1
	cfg *config.Config
}

func (s *StorageService_v1) NewServeMux(cfg *config.Config) http.Handler {
	// init registry
	s.cfg = cfg
	s.reg = &ShardRegisty_v1{
		dataDir: cfg.DataDir,
	}

	// register handlers
	hldr := http.NewServeMux()
	hldr.HandleFunc("GET /{key}", s.GetShard)
	hldr.HandleFunc("PUT /{key}", s.PutShard)
	return hldr
}

func (s *StorageService_v1) GetShard(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	data, err := s.reg.GetShard(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error loading shard: %s", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *StorageService_v1) PutShard(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	r.Body = http.MaxBytesReader(w, r.Body, config.SHARD_SIZE)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	if err := s.reg.PutShard(key, data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error writing shard: %s", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}
