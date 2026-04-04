package storage

import "net/http"

type StorageService interface {
	NewServeMux() http.Handler
	GetShard(w http.ResponseWriter, r *http.Request)
	PutShard(w http.ResponseWriter, r *http.Request)
}

type StorageService_v1 struct{}

func (s *StorageService_v1) NewServeMux() http.Handler {
	hldr := http.NewServeMux()
	hldr.HandleFunc("GET /{key}", s.GetShard)
	hldr.HandleFunc("PUT /{key}", s.PutShard)
	return hldr
}

func (s *StorageService_v1) GetShard(w http.ResponseWriter, r *http.Request) {}
func (s *StorageService_v1) PutShard(w http.ResponseWriter, r *http.Request) {}
