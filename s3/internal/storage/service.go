package storage

import (
	"os"

	"github.com/tickloop/kilo/internal/common"
)

type ShardRegistry interface {
	GetShard(key string) ([]byte, error)
	PutShard(key string, data []byte) error
}

// v1 - basic servie
// no replication
type ShardRegisty_v1 struct {
	dataDir string
}

func (s *ShardRegisty_v1) GetShard(key string) ([]byte, error) {
	path, err := common.SafePathJoin(s.dataDir, key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *ShardRegisty_v1) PutShard(key string, data []byte) error {
	path, err := common.SafePathJoin(s.dataDir, key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
