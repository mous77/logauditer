package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type Cache struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewCache() *Cache {
	return &Cache{
		data: make(map[string]string),
	}
}

func (c *Cache) Get(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var err error
	ret, ok := c.data[key]
	if !ok {
		err = fmt.Errorf("%s", "not key.")
	}
	return ret, err
}

func (c *Cache) Set(key string, value string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.data[key] = value
	return nil
}

func (c *Cache) Keys() ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var keys = make([]string, 0, len(c.data))
	for k, _ := range c.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (c *Cache) Del(key string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	delete(c.data, key)
	return nil
}

// Marshal serializes cache data
func (c *Cache) Marshal() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(c.data)
}

// UnMarshal deserializes cache data
func (c *Cache) UnMarshal(serialized io.ReadCloser) error {
	var newData map[string]string
	var err error
	if err = json.NewDecoder(serialized).Decode(&newData); err != nil {
		return err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.data = newData
	return err
}
