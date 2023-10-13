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

func (c *cacherMock) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	c.init()
	c.store.Store(key, val)
	return nil
}

func (c *cacherMock) Delete(ctx context.Context, key string) error {
	c.init()
	c.store.Delete(key)
	return nil
}

func (c *cacherMock) DeleteWithPrefix(ctx context.Context, keyPrefix string) error {
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

func (c *cacherStoreErrorMock) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return errors.New("store-error")
}

func (c *cacherStoreErrorMock) Delete(ctx context.Context, key string) error {
	c.init()
	c.store.Delete(key)
	return nil
}

func (c *cacherStoreErrorMock) DeleteWithPrefix(ctx context.Context, keyPrefix string) error {
	return nil
}
