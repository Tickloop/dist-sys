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
	"strconv"

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
	mux.HandleFunc("GET /bucket/{name}", GetBucketHandler)
	mux.HandleFunc("DELETE /bucket/{name}", DeleteBucketHandler)
	return mux
}

type CreateBucketRequest struct {
	Name string `json:"name"`
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
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 100 // default limit
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0 // default offset
	}

	buckets := reg.ListBuckets(limit, offset)
	json.NewEncoder(w).Encode(buckets)
}

func GetBucketHandler(w http.ResponseWriter, r *http.Request) {
	cfg.Logger.Info("GetBucketHandler invoked")
	name := r.PathValue("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Empty bucket name"))
		return
	}

	b, err := reg.GetBucket(name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	json.NewEncoder(w).Encode(b)
}

func DeleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	cfg.Logger.Info("DeleteBucketHandler invoked")
	name := r.PathValue("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Empty bucket name"))
		return
	}

	if err := reg.DeleteBucket(name); err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
}
