package cache

// Cache - interface for caching
type Cache interface {
	setter
	getter
}

type setter interface {
	Set(string, interface{})
}

type getter interface {
	Get(string) (interface{}, bool)
}
