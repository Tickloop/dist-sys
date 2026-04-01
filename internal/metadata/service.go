package metadata

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

const (
	// RegistryFilePath = "/var/lib/kilo/registry.jsonl"
	RegistryFilePath = "./registry.jsonl"
)

type Bucket struct {
	Name      string    `json:"name"` // max 1024 characters, must be unique
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Registry struct {
	Buckets map[string]Bucket
}

func InitRegistry() (*Registry, error) {
	f, err := os.OpenFile(RegistryFilePath, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		// TODO: Research error types that can be returned by os.OpenFile and handle them appropriately
		cfg.Logger.Error("Failed to open registry file", "error", err)
		return &Registry{Buckets: make(map[string]Bucket)}, err
	}
	defer f.Close()

	reg := &Registry{Buckets: make(map[string]Bucket)}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		bucket := Bucket{}
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &bucket); err != nil {
			cfg.Logger.Error("Failed to unmarshal bucket from registry file", "line", line, "error", err)
			return &Registry{Buckets: make(map[string]Bucket)}, err
		}
		reg.Buckets[bucket.Name] = bucket
	}
	return reg, nil
}

func (b Bucket) Serialize() error {
	f, err := os.OpenFile(RegistryFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		cfg.Logger.Error("Failed to open registry file", "error", err)
		return err
	}
	defer f.Close()

	jBucket, err := json.Marshal(b)
	if err != nil {
		cfg.Logger.Error("Failed to serialize bucket", "bucket", b.Name, "error", err)
		return err
	}

	if _, err := f.Write(jBucket); err != nil {
		// TODO: Research error types that can be returned by f.Write and handle them appropriately
		cfg.Logger.Error("Failed to write bucket to registry file", "bucket", b.Name, "error", err)
		return err
	}

	return nil
}

// TODO: This method has two failure modes -
// need to communicate better which error occured
func (r *Registry) CreateBucket(name string) error {
	if b, ok := r.Buckets[name]; ok {
		return fmt.Errorf("Bucket %s already exists", b.Name)
	}
	r.Buckets[name] = Bucket{
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.Buckets[name].Serialize(); err != nil {
		delete(r.Buckets, name)
		return fmt.Errorf("Failed to serialize bucket %s: %v", name, err)
	}
	return nil
}

func (r *Registry) ListBuckets(limit int, offset int) []Bucket {
	// naive-sort
	keys := make([]string, 0, len(r.Buckets))
	for k := range r.Buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buckets := []Bucket{}
	for _, key := range keys {
		buckets = append(buckets, r.Buckets[key])
	}

	if offset >= len(buckets) {
		return []Bucket{}
	}

	return buckets[offset:min(offset+limit, len(buckets))]
}

func (r *Registry) DeleteBucket(name string) error {
	if _, ok := r.Buckets[name]; !ok {
		return fmt.Errorf("Bucket %s does not exist", name)
	}
	delete(r.Buckets, name)
	return nil
}
