package metadata

import (
	"fmt"
	"time"
)

type Bucket struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	Buckets = make(map[string]Bucket)
)

func CreateBucket(name string) error {
	if b, ok := Buckets[name]; ok {
		return fmt.Errorf("Bucket %s already exists", b.Name)
	}
	Buckets[name] = Bucket{
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func ListBuckets(limit int, offset int) []Bucket {
	buckets := []Bucket{}
	for _, bucket := range Buckets {
		buckets = append(buckets, bucket)
	}

	if offset >= len(buckets) {
		return []Bucket{}
	}

	return buckets[offset:min(offset+limit, len(buckets))]
}

func DeleteBucket(name string) error {
	if _, ok := Buckets[name]; !ok {
		return fmt.Errorf("Bucket %s does not exist", name)
	}
	delete(Buckets, name)
	return nil
}
