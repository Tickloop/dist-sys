package metadata

import (
	"crypto/sha256"
	"fmt"

	"github.com/tickloop/kilo/internal/clients"
	"github.com/tickloop/kilo/internal/config"
)

type Object struct {
	Key    string
	Data   []byte
	Size   int
	shards []shard
}

type shard struct {
	idx int
	key string
}

type Bucket struct {
	Key     string
	Objects map[string]Object
}

type MetadataRegistry interface {
	// Public API for handlers to use
	// other things it needs to handle:
	// persistance - serialize/deserialize
	// consistency - share state within nodes
	GetObject(bucket, key string) (Object, error)
	PutObject(bucket, key string, data []byte) error
	ListObjects(bucket string) ([]Object, error)
	DeleteObject(bucket, key string) error
	CreateBucket(key string) error
	ListBuckets() ([]Bucket, error)
	DeleteBucket(key string) error
}

type MetadataRegistry_v1 struct {
	buckets       map[string]Bucket
	storageClient clients.StorageServieClient
}

func InitMetadataRegistry(cfg *config.Config) MetadataRegistry {
	return &MetadataRegistry_v1{
		buckets:       make(map[string]Bucket),
		storageClient: clients.InitStorageServiceClient(cfg.StorageServiceUrl),
	}
}

func (m *MetadataRegistry_v1) GetObject(bucket, key string) (Object, error) {
	b, ok := m.buckets[bucket]
	if !ok {
		return Object{}, fmt.Errorf("bucket %s not found", bucket)
	}

	obj, ok := b.Objects[key]
	if !ok {
		return Object{}, fmt.Errorf("object %s not found", key)
	}

	// collect shards and stitch them
	data := make([]byte, obj.Size)
	for i, shard := range obj.shards {
		// TODO: can add caching for object shards - cache invalidation will be a fun puzzle
		chunk, err := m.storageClient.GetShard(shard.key)
		if err != nil {
			return Object{}, fmt.Errorf("getting chunk: %w", err)
		}

		// TODO: can this be streamed?
		copy(data[i*config.SHARD_SIZE:], chunk)
	}

	// TODO: check memory allocation of this object
	return Object{Key: obj.Key, Data: data, Size: obj.Size, shards: obj.shards}, nil
}

func (m *MetadataRegistry_v1) PutObject(bucket, key string, data []byte) error {
	b, ok := m.buckets[bucket]
	if !ok {
		return fmt.Errorf("bucket not found: %s", bucket)
	}

	// if key already in bucket - overwrite old data
	// build shards
	idx := 0
	var shards []shard
	for i := 0; i < len(data); i += config.SHARD_SIZE {
		_end := min(len(data), i+config.SHARD_SIZE)
		chunk := make([]byte, config.SHARD_SIZE)
		copy(chunk, data[i:_end])
		key := fmt.Sprintf("%x", sha256.Sum256(chunk))

		// send chunks
		err := m.storageClient.PutShard(key, chunk)
		if err != nil {
			// TODO: cleanup old chunks that have been persisted
			return fmt.Errorf("creating chunk: %w", err)
		}

		shards = append(shards, shard{idx: idx, key: key})
		idx += 1
	}

	// now create object metadata and persist in registry
	obj := Object{
		Key:    key,
		Size:   len(data),
		shards: shards,
	}
	b.Objects[key] = obj
	return nil
}

func (m *MetadataRegistry_v1) ListObjects(bucket string) ([]Object, error) {
	b, ok := m.buckets[bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", bucket)
	}

	// TODO: this needs to be a tree structure
	objects := make([]Object, 0)
	for _, obj := range b.Objects {
		objects = append(objects, obj)
	}
	return objects, nil
}

func (m *MetadataRegistry_v1) DeleteObject(bucket, key string) error {
	b, ok := m.buckets[bucket]
	if !ok {
		return fmt.Errorf("bucket not found: %s", bucket)
	}

	// TODO: also need to clean-up shards
	delete(b.Objects, key)
	return nil
}

func (m *MetadataRegistry_v1) CreateBucket(bucket string) error {
	if _, ok := m.buckets[bucket]; ok {
		return fmt.Errorf("Duplicate bucket key: %s", bucket)
	}
	m.buckets[bucket] = Bucket{Key: bucket, Objects: make(map[string]Object)}
	return nil
}

func (m *MetadataRegistry_v1) ListBuckets() ([]Bucket, error) {
	buckets := make([]Bucket, 0)
	for _, bucket := range m.buckets {
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

func (m *MetadataRegistry_v1) DeleteBucket(bucket string) error {
	// TODO: need to check if there are any objects and clean them up
	delete(m.buckets, bucket)
	return nil
}
