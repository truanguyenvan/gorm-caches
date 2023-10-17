package caches

import (
	"errors"
	"fmt"
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

func (c *cacherMock) Get(key string) ([]byte, error) {
	c.init()
	fmt.Println("START READ CACHE KEY: ", key)
	val, ok := c.store.Load(key)
	if !ok {
		return nil, nil
	}
	fmt.Println("READ CACHE KEY: ", key)

	return val.([]byte), nil
}

func (c *cacherMock) Set(key string, val []byte, ttl time.Duration) error {
	c.init()
	fmt.Println("SET CACHE KEY: ", key)
	c.store.Store(key, val)
	return nil
}

func (c *cacherMock) Delete(key string) error {
	c.init()
	c.store.Delete(key)
	return nil
}

func (c *cacherMock) DeleteWithPrefix(keyPrefix string) error {
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

func (c *cacherStoreErrorMock) Get(key string) ([]byte, error) {
	c.init()
	val, ok := c.store.Load(key)
	if !ok {
		return nil, nil
	}

	return val.([]byte), nil
}

func (c *cacherStoreErrorMock) Set(key string, val []byte, ttl time.Duration) error {
	return errors.New("store-error")
}

func (c *cacherStoreErrorMock) Delete(key string) error {
	c.init()
	c.store.Delete(key)
	return nil
}

func (c *cacherStoreErrorMock) DeleteWithPrefix(keyPrefix string) error {
	return nil
}
