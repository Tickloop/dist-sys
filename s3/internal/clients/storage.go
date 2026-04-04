package clients

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tickloop/kilo/internal/config"
)

// a clean SDK to GET and PUT shards
type StorageServieClient interface {
	GetShard(key string) ([]byte, error)
	PutShard(key string, data []byte) error
}

type StorageServieClient_v1 struct {
	base   string
	client *http.Client
}

func InitStorageServiceClient(baseUrl string) StorageServieClient {
	return &StorageServieClient_v1{
		client: &http.Client{Timeout: 5 * time.Second},
		base:   baseUrl,
	}
}

func (s *StorageServieClient_v1) GetShard(key string) ([]byte, error) {
	req, err := http.NewRequest("GET", s.base+"/"+key, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("response: %w", err)
	}
	defer resp.Body.Close()

	// discards error returned from the service
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	// shard max size 64KB
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	return data, nil
}

func (s *StorageServieClient_v1) PutShard(key string, data []byte) error {
	if len(data) > config.SHARD_SIZE {
		return fmt.Errorf("shard too large: %d", len(data))
	}

	req, err := http.NewRequest("PUT", s.base+"/"+key, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return nil
}
