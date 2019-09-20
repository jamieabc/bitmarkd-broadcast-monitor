package cache

import (
	"errors"
	"time"

	goCache "github.com/patrickmn/go-cache"
)

const (
	defaultExpireTime  = 5 * time.Minute
	defaultCleanupTime = 10 * time.Minute
)

type implementation struct {
	cache *goCache.Cache
}

// Set - set value of key to cache
func (i *implementation) Set(key string, value interface{}) {
	i.cache.Set(key, value, goCache.DefaultExpiration)
}

// Get - get value of key from cache
func (i *implementation) Get(key string) (interface{}, bool) {
	return i.cache.Get(key)
}

// NewCache - new cache
// if no arguments given default expiration is 5 minutes, clean-up time is 10 minutes
func NewCache(args ...interface{}) (Cache, error) {
	if 0 == len(args) {
		return &implementation{
			cache: goCache.New(defaultExpireTime, defaultCleanupTime),
		}, nil
	}

	if 2 != len(args) {
		return &implementation{}, errors.New("wrong parameters, expect expire-time & cleanup-time")

	}

	expireTime := args[0].(time.Duration)
	cleanupTime := args[1].(time.Duration)

	return &implementation{
		cache: goCache.New(expireTime, cleanupTime),
	}, nil
}
