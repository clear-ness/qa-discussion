package cachelayer

import (
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type CacheVoteStore struct {
	store.VoteStore
	rootStore *CacheStore
}

func (s CacheVoteStore) GetVoteTypesForPost(userId string, postId string) ([]string, *model.AppError) {
	key := userId + postId + "votes"

	// TODO: voteが空の場合にエラーなはず
	if types := s.rootStore.readSetCache(key); types != nil {
		return *types, nil
	}

	types, err := s.VoteStore.GetVoteTypesForPost(userId, postId)
	if err != nil {
		return []string{}, err
	}

	s.rootStore.addToSetCache(key, types)

	return types, nil
}
