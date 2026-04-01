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
	reg *Registry
)

func NewRouter(cf *config.Config) *http.ServeMux {
	reg, _ = InitRegistry()
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

	if len(req.Name) > 1024 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bucket name must be less than 1024 characters"))
		return
	}

	if err := reg.CreateBucket(req.Name); err != nil {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(err.Error()))
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

	buckets := reg.ListBuckets(req.Limit, req.Offset)
	json.NewEncoder(w).Encode(buckets)
}
