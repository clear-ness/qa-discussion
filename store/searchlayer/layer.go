package searchlayer

import (
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/search"
	"github.com/clear-ness/qa-discussion/store"
)

type SearchStore struct {
	store.Store
	post             *SearchPostStore
	vote             *SearchVoteStore
	userPointHistory *SearchUserPointHistoryStore
	postViewsHistory *SearchPostViewsHistoryStore
	config           *model.Config
	esBackend        *search.ESBackend
}

func NewSearchLayer(baseStore store.Store, cfg *model.Config) *SearchStore {
	searchStore := &SearchStore{
		Store:  baseStore,
		config: cfg,
	}

	searchStore.post = &SearchPostStore{
		PostStore: baseStore.Post(),
		rootStore: searchStore,
	}

	searchStore.vote = &SearchVoteStore{
		VoteStore: baseStore.Vote(),
		rootStore: searchStore,
	}

	searchStore.userPointHistory = &SearchUserPointHistoryStore{
		UserPointHistoryStore: baseStore.UserPointHistory(),
		rootStore:             searchStore,
	}

	searchStore.postViewsHistory = &SearchPostViewsHistoryStore{
		PostViewsHistoryStore: baseStore.PostViewsHistory(),
		rootStore:             searchStore,
	}

	setting := *cfg
	esBackend, err := search.NewESBackend(&setting.SearchSettings)
	if err != nil {
		return nil
	}
	searchStore.esBackend = esBackend

	return searchStore
}

func (s *SearchStore) Post() store.PostStore {
	return s.post
}

func (s *SearchStore) Vote() store.VoteStore {
	return s.vote
}

func (s *SearchStore) UserPointHistory() store.UserPointHistoryStore {
	return s.userPointHistory
}

func (s *SearchStore) PostViewsHistory() store.PostViewsHistoryStore {
	return s.postViewsHistory
}

func (s *SearchStore) SetupIndexes() {
	s.post.SetupIndex()
	s.vote.SetupIndex()
	s.userPointHistory.SetupIndex()
	s.postViewsHistory.SetupIndex()
}
