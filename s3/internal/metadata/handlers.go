package metadata

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/tickloop/kilo/internal/config"
)

type MetadataService interface {
	NewServeMux() http.Handler
	GetObject(w http.ResponseWriter, r *http.Request)
	PutObject(w http.ResponseWriter, r *http.Request)
	ListObjects(w http.ResponseWriter, r *http.Request)
	DeleteObject(w http.ResponseWriter, r *http.Request)
	CreateBucket(w http.ResponseWriter, r *http.Request)
	ListBuckets(w http.ResponseWriter, r *http.Request)
	DeleteBucket(w http.ResponseWriter, r *http.Request)
}

// v1 - basic service
type MetadataService_v1 struct {
	reg MetadataRegistry
	cfg *config.Config
}

func (m *MetadataService_v1) NewServeMux(cfg *config.Config) http.Handler {
	// init registry
	m.cfg = cfg
	m.reg = InitMetadataRegistry(cfg)

	hldr := http.NewServeMux()

	hldr.HandleFunc("GET /{bucket}/{key...}", m.GetObject)
	hldr.HandleFunc("PUT /{bucket}/{key...}", m.PutObject)
	hldr.HandleFunc("DELETE /{bucket}/{key...}", m.DeleteObject)

	hldr.HandleFunc("GET /", m.ListBuckets)
	hldr.HandleFunc("GET /{bucket}", m.ListObjects)
	hldr.HandleFunc("PUT /{bucket}", m.CreateBucket)
	hldr.HandleFunc("DELETE /{bucket}", m.DeleteBucket)

	return hldr
}

func (m *MetadataService_v1) GetObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	object, err := m.reg.GetObject(bucket, key)
	// TODO: Better types for errors - this helps send better messages
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(object.Data)
}

func (m *MetadataService_v1) PutObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	// TODO: make this production grade to handle large file uploads
	data, err := io.ReadAll(r.Body)
	if err != nil {
		m.cfg.Logger.Error(err.Error())
		http.Error(w, "error reading file", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := m.reg.PutObject(bucket, key, data); err != nil {
		m.cfg.Logger.Error(err.Error())
		http.Error(w, "error uploading file", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (m *MetadataService_v1) ListObjects(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")

	objects, err := m.reg.ListObjects(bucket)
	if err != nil {
		m.cfg.Logger.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error fetching objects"))
		return
	}

	// TODO: Add a serialation layer for Objects
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(objects)
}

func (m *MetadataService_v1) DeleteObject(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	if err := m.reg.DeleteObject(bucket, key); err != nil {
		m.cfg.Logger.Error(err.Error())
		http.Error(w, "error deleting object", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (m *MetadataService_v1) CreateBucket(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	if err := m.reg.CreateBucket(bucket); err != nil {
		m.cfg.Logger.Error(err.Error())
		http.Error(w, "error creating bucket", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (m *MetadataService_v1) ListBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := m.reg.ListBuckets()
	if err != nil {
		m.cfg.Logger.Error(err.Error())
		http.Error(w, "error creating bucket", http.StatusInternalServerError)
		return
	}
	// TODO: serialization format for buckets
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(buckets)
}

func (m *MetadataService_v1) DeleteBucket(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	if err := m.reg.DeleteBucket(bucket); err != nil {
		m.cfg.Logger.Error(err.Error())
		http.Error(w, "error creating bucket", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
