package l1cache

import "time"

// CacheOptions contains options for initializaing a cache
type CacheOptions struct {
	Size                   int
	DefaultExpiry          time.Duration
	Name                   string
	InvalidateClusterEvent string
}

// Provider is a provider for Cache
type Provider interface {
	// NewCache creates a new cache with given options.
	NewCache(opts *CacheOptions) Cache
}

type cacheProvider struct {
}

// NewProvider creates a new CacheProvider
func NewProvider() Provider {
	return &cacheProvider{}
}

// NewCache creates a new cache with given opts
func (c *cacheProvider) NewCache(opts *CacheOptions) Cache {
	return NewLRU(&LRUOptions{
		Size:                   opts.Size,
		DefaultExpiry:          opts.DefaultExpiry,
		InvalidateClusterEvent: opts.InvalidateClusterEvent,
	})
}
