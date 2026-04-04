package storage

type ShardRegistry interface {
	GetShard(key string) ([]byte, error)
	PutShard(key string, data []byte) (error)
}

