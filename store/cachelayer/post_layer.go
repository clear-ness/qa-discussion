package cachelayer

import (
	"strconv"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

const (
	POST_VIEW_KEY_TTL    = 15 * 60
	POST_COUNTER_KEY_TTL = 3 * 60
)

type CachePostStore struct {
	store.PostStore
	rootStore *CacheStore
}

// (どちらかと言うと)views countを意図的に増やす事を防止する方向に倒す。
func (s CachePostStore) ViewPost(postId string, teamId string, userId string, ipAddress string, count int) *model.AppError {
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
		s.rootStore.addToCache(viewKey, "", POST_VIEW_KEY_TTL)
	}

	counterKey := postId + "counters"

	countNum := 0
	countStr := s.rootStore.readCache(counterKey)
	if countStr == nil {
		s.rootStore.addToCache(counterKey, 1, POST_COUNTER_KEY_TTL)
	} else {
		if val, err := strconv.Atoi(*countStr); err == nil {
			countNum = val
		} else {
			return nil
		}
	}

	countNum = countNum + 1

	if countNum >= model.POST_COUNTER_MAX {
		// counterバッファーを初期化する(ttlも変更)
		s.rootStore.addToCache(counterKey, 0, POST_COUNTER_KEY_TTL)

		err := s.PostStore.ViewPost(postId, teamId, userId, ipAddress, model.POST_COUNTER_MAX)
		if err != nil {
			return err
		}

	} else {
		// ttlは変更させ無い
		s.rootStore.incrementBy(counterKey, 1)
	}

	return nil
}

// TODO: keyをconst定義
func (s CachePostStore) InvalidateVoteTypesForPost(userId string, postId string) {
	key := userId + postId + "votes"

	s.rootStore.deleteCache([]string{key})
}

func (s CachePostStore) UpVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.UpVotePost(postId, userId)
	if err != nil {
		return nil, err
	}

	s.InvalidateVoteTypesForPost(userId, postId)

	return vote, nil
}

func (s CachePostStore) CancelUpVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.CancelUpVotePost(postId, userId)
	if err != nil {
		return nil, err
	}

	s.InvalidateVoteTypesForPost(userId, postId)

	return vote, nil
}

func (s CachePostStore) DownVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.DownVotePost(postId, userId)
	if err != nil {
		return nil, err
	}

	s.InvalidateVoteTypesForPost(userId, postId)

	return vote, nil
}

func (s CachePostStore) CancelDownVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.CancelDownVotePost(postId, userId)
	if err != nil {
		return nil, err
	}

	s.InvalidateVoteTypesForPost(userId, postId)

	return vote, nil
}

func (s CachePostStore) FlagPost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.FlagPost(postId, userId)
	if err != nil {
		return nil, err
	}

	s.InvalidateVoteTypesForPost(userId, postId)

	return vote, nil
}

func (s CachePostStore) CancelFlagPost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.CancelFlagPost(postId, userId)
	if err != nil {
		return nil, err
	}

	s.InvalidateVoteTypesForPost(userId, postId)

	return vote, nil
}

// TODO: チームの投稿リストの最初の1ページ目をキャッシュ
// Contentが巨大な場合やViewsを考慮し、idsのみをキャッシュ管理して
// 内部的に GetPostsByIds を呼ぶ様にする？
// もしくはsizeをlimitContentしてキャッシュ管理？

// TODO: GetPostCount をキャッシュ
