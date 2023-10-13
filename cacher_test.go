package caches

import (
	"context"
	"errors"
	"sync"
	"time"
)

type cacherMock struct {
	store *sync.Map
}

func (c *cacherMock) init() {
	if c.store == nil {
		c.store = &sync.Map{}
	}
}

func (c *cacherMock) Get(ctx context.Context, key string) ([]byte, error) {
	c.init()
	val, ok := c.store.Load(key)
	if !ok {
		return nil, nil
	}

	return val.([]byte), nil
}

func (c *cacherMock) Store(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	c.init()
	c.store.Store(key, val)
	return nil
}

func (c *cacherMock) DeleteKey(ctx context.Context, key string) error {
	c.init()
	c.store.Delete(key)
	return nil
}

func (c *cacherMock) DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error {
	return nil
}

type cacherStoreErrorMock struct {
	store *sync.Map
}

func (c *cacherStoreErrorMock) init() {
	if c.store == nil {
		c.store = &sync.Map{}
	}
}

func (c *cacherStoreErrorMock) Get(ctx context.Context, key string) ([]byte, error) {
	c.init()
	val, ok := c.store.Load(key)
	if !ok {
		return nil, nil
	}

	return val.([]byte), nil
}

func (c *cacherStoreErrorMock) Store(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return errors.New("store-error")
}

func (c *cacherStoreErrorMock) DeleteKey(ctx context.Context, key string) error {
	c.init()
	c.store.Delete(key)
	return nil
}

func (c *cacherStoreErrorMock) DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error {
	return nil
}
