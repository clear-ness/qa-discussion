package cachelayer

import (
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/cache"
	"github.com/clear-ness/qa-discussion/store"
)

type CacheStore struct {
	store.Store
	post                CachePostStore
	team                CacheTeamStore
	userFavoritePost    CacheUserFavoritePostStore
	notificationSetting CacheNotificationSettingStore
	vote                CacheVoteStore
	config              *model.Config
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

	cacheStore.userFavoritePost = CacheUserFavoritePostStore{
		UserFavoritePostStore: baseStore.UserFavoritePost(),
		rootStore:             &cacheStore,
	}

	cacheStore.notificationSetting = CacheNotificationSettingStore{
		NotificationSettingStore: baseStore.NotificationSetting(),
		rootStore:                &cacheStore,
	}

	cacheStore.vote = CacheVoteStore{
		VoteStore: baseStore.Vote(),
		rootStore: &cacheStore,
	}

	return cacheStore
}

func (s CacheStore) Post() store.PostStore {
	return s.post
}

func (s CacheStore) Team() store.TeamStore {
	return s.team
}

func (s CacheStore) UserFavoritePost() store.UserFavoritePostStore {
	return s.userFavoritePost
}

func (s CacheStore) NotificationSetting() store.NotificationSettingStore {
	return s.notificationSetting
}

func (s CacheStore) Vote() store.VoteStore {
	return s.vote
}

func (s CacheStore) DropAllTables() {
	s.Invalidate()
	s.Store.DropAllTables()
}

func (s *CacheStore) Invalidate() {
	// deletes all keys from all databases
	cache.NewRedisBackend(&s.config.CacheSettings).FlushAll()
}

func (s *CacheStore) addToCache(key string, value interface{}, ttl int) {
	// TODO: 非同期実行？
	cache.NewRedisBackend(&s.config.CacheSettings).Set(key, value, ttl)
}

func (s *CacheStore) incrementBy(key string, count int) {
	cache.NewRedisBackend(&s.config.CacheSettings).IncrBy(key, count)
}

func (s *CacheStore) readCache(key string) *string {
	val, err := cache.NewRedisBackend(&s.config.CacheSettings).Get(key)
	// キーが無ければerrが返る
	if err == nil {
		return &val
	}

	return nil
}

func (s *CacheStore) addToHashCache(key string, values map[string]interface{}) {
	cache.NewRedisBackend(&s.config.CacheSettings).HSet(key, values)
}

func (s *CacheStore) readHashCache(key string) map[string]string {
	val, err := cache.NewRedisBackend(&s.config.CacheSettings).HGetAll(key)
	// キーが無ければerrが返る
	if err == nil {
		return val
	}

	return nil
}

func (s *CacheStore) addToSetCache(key string, members []string) {
	cache.NewRedisBackend(&s.config.CacheSettings).SAdd(key, members)
}

func (s *CacheStore) readSetCache(key string) *[]string {
	members, err := cache.NewRedisBackend(&s.config.CacheSettings).SMembers(key)
	if err == nil {
		return &members
	}

	return nil
}

func (s *CacheStore) existsKey(key string) bool {
	count, err := cache.NewRedisBackend(&s.config.CacheSettings).Exists(key)
	if err != nil || count <= 0 {
		return false
	}

	return true
}

func (s *CacheStore) deleteCache(keys []string) (int64, error) {
	// 実際に消された数が返る
	return cache.NewRedisBackend(&s.config.CacheSettings).Del(keys)
}
