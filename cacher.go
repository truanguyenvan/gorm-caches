package caches

import (
	"context"
	"time"
)

type Cacher interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteWithPrefix(ctx context.Context, keyPrefix string) error
}
