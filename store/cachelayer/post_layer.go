package cachelayer

import (
	"github.com/clear-ness/qa-discussion/store"
)

type CachePostStore struct {
	store.PostStore
	rootStore *CacheStore
}

// (どちらかと言うと)views countを意図的に増やす事を防止する方向に倒す。
func (s CachePostStore) ViewPost(postId string, userId string, ipAddress string, count int) error {
	viewKey := postId
	if userId != "" {
		viewKey += userId
	} else if ipAddress != "" {
		viewKey += ipAddress
	} else {
		return nil
	}

	exists := s.rootStore.existsKey(viewKey)
	if exists {
		return nil
	} else {
		// バッファーA
		s.rootStore.addToCache(viewKey, "", 15*60)
	}

	counterKey := postId + "counters"
	counter := s.rootStore.doStandardReadCache(counterKey)
	if counter == nil {
		counter = 0
		s.rootStore.addToCache(counterKey, 1, 3*60)
	}

	counter = counter + 1

	if counter == 1 {
		err := s.PostStore.ViewPost(postId, userId, ipAddress, 1)
		if err != nil {
			return err
		}
	} else if counter >= 5 {
		// counterバッファーを初期化する(ttlも変更)
		s.rootStore.addToCache(counterKey, 0, 3*60)

		err := s.PostStore.ViewPost(postId, userId, ipAddress, 5)
		if err != nil {
			return err
		}

	} else {
		// ttlは変更させ無い
		s.rootStore.incrementBy(counterKey, 1)
	}

	return nil
}
