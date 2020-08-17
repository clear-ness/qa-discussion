package cachelayer

import (
	"github.com/clear-ness/qa-discussion/cache"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type CacheStore struct {
	store.Store
	post   CachePostStore
	team   CacheTeamStore
	config *model.Config
}

func NewCacheLayer(baseStore store.Store, cfg *model.Config) CacheStore {
	cacheStore := CacheStore{
		Store:  baseStore,
		config: cfg,
	}

	cacheStore.post = CachePostStore{
		PostStore: baseStore.Post(),
		rootStore: &cacheStore,
	}

	cacheStore.team = CacheTeamStore{
		TeamStore: baseStore.Team(),
		rootStore: &cacheStore,
	}

	return cacheStore
}

func (s CacheStore) Post() store.PostStore {
	s.post
}

func (s CacheStore) Team() store.TeamStore {
	s.team
}

func (s CacheStore) DropAllTables() {
	s.Invalidate()
	s.Store.DropAllTables()
}

func (s *CacheStore) Invalidate() {
}

func (s *CacheStore) addToCache(key string, value interface{}, ttl *int) {
	cache.NewRedisBackend(s.config.CacheSettings).Set(key, value, ttl)
}

func (s *CacheStore) incrementBy(key string, count int) {
	cache.NewRedisBackend(s.config.CacheSettings).IncrBy(key, count)
}

func (s *CacheStore) readCache(key string) *string {
	val, err := cache.NewRedisBackend(s.config.CacheSettings).Get(key)
	if err == nil {
		return val
	}

	return nil
}

func (s *CacheStore) existsKey(key string) bool {
	count, err := cache.NewRedisBackend(config.CacheSettings).Exists(key)
	if err != nil || count <= 0 {
		return false
	}

	return true
}
