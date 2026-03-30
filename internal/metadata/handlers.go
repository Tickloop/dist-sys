package metadata

// The following functions to be supported:
// - CreateBucket
// - ListBuckets
// - GetBucket
// - DeleteBucket
// - CreateObject
// - GetObject
// - DeleteObject

import (
	"encoding/json"
	"net/http"

	"github.com/tickloop/kilo/internal/config"
)

var (
	cfg *config.Config = config.LoadConfig()
)

func NewRouter(cf *config.Config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /bucket", CreateBucketHandler)
	mux.HandleFunc("GET /buckets", ListBucketsHandler)
	return mux
}

type CreateBucketRequest struct {
	Name string `json:"name"`
}

type ListBucketRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func CreateBucketHandler(w http.ResponseWriter, r *http.Request) {
	cfg.Logger.Info("CreateBucketHandler invoked")
	var req CreateBucketRequest
	json.NewDecoder(r.Body).Decode(&req)

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bucket name is required"))
		return
	}

	if err := CreateBucket(req.Name); err != nil {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("Duplicate bucket name"))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func ListBucketsHandler(w http.ResponseWriter, r *http.Request) {
	cfg.Logger.Info("ListBucketsHandler invoked")
	var req ListBucketRequest
	json.NewDecoder(r.Body).Decode(&req)

	if req.Limit <= 0 {
		req.Limit = 25
	}

	buckets := ListBuckets(req.Limit, req.Offset)
	json.NewEncoder(w).Encode(buckets)
}
