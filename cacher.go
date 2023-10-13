package caches

import (
	"context"
	"time"
)

type Cacher interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Store(ctx context.Context, key string, val []byte, ttl time.Duration) error
	DeleteKey(ctx context.Context, key string) error
	DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error
}
