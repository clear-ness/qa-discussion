package cachelayer

import (
	"strconv"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type CacheUserFavoritePostStore struct {
	store.UserFavoritePostStore
	rootStore *CacheStore
}

func (s CacheUserFavoritePostStore) GetCountByPostId(postId string) (int64, *model.AppError) {
	favoriteKey := postId + "favorites"

	if countStr := s.rootStore.readCache(favoriteKey); countStr != nil {
		if val, err := strconv.Atoi(*countStr); err == nil {
			return int64(val), nil
		}
	}

	count, err := s.UserFavoritePostStore.GetCountByPostId(postId)
	if err != nil {
		return 0, err
	}

	s.rootStore.addToCache(favoriteKey, count, 0)

	return count, nil
}

func (s CacheUserFavoritePostStore) InvalidateFavoriteCount(postId string) {
	favoriteKey := postId + "favorites"
	s.rootStore.deleteCache([]string{favoriteKey})
}

func (s CacheUserFavoritePostStore) Save(postId string, userId string, teamId string) *model.AppError {
	err := s.UserFavoritePostStore.Save(postId, userId, teamId)
	if err != nil {
		return err
	}

	s.InvalidateFavoriteCount(postId)

	return nil
}

func (s CacheUserFavoritePostStore) Delete(postId string, userId string) *model.AppError {
	err := s.UserFavoritePostStore.Delete(postId, userId)
	if err != nil {
		return err
	}

	s.InvalidateFavoriteCount(postId)

	return nil
}
