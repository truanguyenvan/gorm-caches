package caches

import (
	"github.com/goccy/go-json"
)

type JSONSerializer struct{}

func (JSONSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONSerializer) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
