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
	IsDeleted bool      `json:"-"`
}

type bucketRecord struct {
	Bucket
	IsDeleted bool `json:"is_deleted"`
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
		bucket_record := bucketRecord{}
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &bucket_record); err != nil {
			cfg.Logger.Error("Failed to unmarshal bucket from registry file", "line", line, "error", err)
			return &Registry{Buckets: make(map[string]Bucket)}, err
		}

		if _, ok := reg.Buckets[bucket_record.Name]; ok && bucket_record.IsDeleted {
			// check if the bucket already exists and was deleted
			delete(reg.Buckets, bucket_record.Name)
		} else if !ok {
			// does not exist - store it in memory
			reg.Buckets[bucket_record.Name] = bucket_record.Bucket
		}
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

	if err := json.NewEncoder(f).Encode(bucketRecord{Bucket: b, IsDeleted: b.IsDeleted}); err != nil {
		// TODO: Research error types that can be returned by f.Write and handle them appropriately
		cfg.Logger.Error("Failed to write bucket to registry file", "bucket", b.Name, "error", err)
		return err
	}

	return nil
}

// TODO: This method has two failure modes -
// need to communicate better which error occured
func (r *Registry) CreateBucket(name string) error {
	if _, ok := r.Buckets[name]; ok {
		return fmt.Errorf("Bucket %s already exists", name)
	}
	b := Bucket{
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := b.Serialize(); err != nil {
		return fmt.Errorf("Failed to serialize bucket %s: %v", name, err)
	}
	r.Buckets[name] = b
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
	b, ok := r.Buckets[name]
	if !ok {
		return fmt.Errorf("Bucket %s does not exist", name)
	}
	b.IsDeleted = true
	if err := b.Serialize(); err != nil {
		b.IsDeleted = false
		return fmt.Errorf("Failed to delete bucket %s", b.Name)
	}
	delete(r.Buckets, name)
	return nil
}

func (r *Registry) GetBucket(name string) (Bucket, error) {
	b, ok := r.Buckets[name]
	if !ok {
		return Bucket{}, fmt.Errorf("Bucket %s does not exist", name)
	}
	return b, nil
}
