package metadata

import (
	"time"

	"github.com/google/uuid"
)

type macTime struct {
	ModifyTime time.Time
	AccessTime time.Time
	CreateTime time.Time
}

type Object struct {
	Key string
	macTime
	shards []shard
	size   int
}

type shard struct {
	id  uuid.UUID
	idx int
}

type Bucket struct {
	Key     string
	Objects []Object
}

type MetadataRegistry interface {
	// Public API for handlers to use
	// other things it needs to handle:
	// persistance - serialize/deserialize
	// consistency - share state within nodes
	GetObject(key string) (Object, error)
	PutObject(key string, data []byte) error
	ListObjects() ([]Object, error)
	DeleteObject(key string) error
	CreateBucket(key string) error
	ListBuckets() ([]Bucket, error)
	DeleteBucket(key string) error
}
