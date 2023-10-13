package caches

type Serializer interface {
	Serialize(v any) ([]byte, error)

	Deserialize(data []byte, v any) error
}
