package caches

import (
	"time"
)

type Cacher interface {
	Get(key string) ([]byte, error)
	Set(key string, val []byte, ttl time.Duration) error
	Delete(key string) error
	DeleteWithPrefix(keyPrefix string) error
}
