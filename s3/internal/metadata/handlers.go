package metadata

import "net/http"

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
type MetadataService_v1 struct{}

func (m *MetadataService_v1) NewServeMux() http.Handler {
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

func (m *MetadataService_v1) GetObject(w http.ResponseWriter, r *http.Request)    {}
func (m *MetadataService_v1) PutObject(w http.ResponseWriter, r *http.Request)    {}
func (m *MetadataService_v1) ListObjects(w http.ResponseWriter, r *http.Request)  {}
func (m *MetadataService_v1) DeleteObject(w http.ResponseWriter, r *http.Request) {}
func (m *MetadataService_v1) CreateBucket(w http.ResponseWriter, r *http.Request) {}
func (m *MetadataService_v1) ListBuckets(w http.ResponseWriter, r *http.Request)  {}
func (m *MetadataService_v1) DeleteBucket(w http.ResponseWriter, r *http.Request) {}
