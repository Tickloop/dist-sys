package storage

import (
	"fmt"
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
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

func (s *StorageService_v1) PutShard(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	data := make([]byte, 64*1024) // 64 KB max shard size

	if _, err := r.Body.Read(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error writing shard: %s", err.Error())
	}

	if err := s.reg.PutShard(key, data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error writing shard: %s", err.Error())
	}

	w.WriteHeader(http.StatusAccepted)
}
