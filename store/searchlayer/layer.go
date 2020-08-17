package searchlayer

import (
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/search"
	"github.com/clear-ness/qa-discussion/store"
)

type SearchStore struct {
	store.Store
	post   *SearchPostStore
	config *model.Config
}

func NewSearchLayer(baseStore store.Store, cfg *model.Config) {
	searchStore := &SearchStore{
		Store:  baseStore,
		config: cfg,
	}

	searchStore.post = &SearchPostStore{
		PostStore: baseStore.Post(),
		rootStore: searchStore,
	}
}

func (s *SearchStore) Post() store.PostStore {
	return s.post
}

func (s *SearchStore) setupIndex(mapping string, indexName string) error {
	return NewESBackend(s.config.SearchSettings, indexName).CreateIndex(mapping)
}

func (s *SearchStore) SetupIndexes() {
	s.post.setupIndex()
	//s.user.setupIndex()
	//s.team.setupIndex()
}

func (s *SearchStore) indexPost(*model.Post post) error {
	esPost:= ESPostFromPost(post)
	return NewESBackend(s.config.SearchSettings, INDEX_NAME_POSTS).IndexESPost(esPost)
}

func (s *SearchStore) deletePost(*model.Post post) error {
	esPost:= ESPostFromPost(post)
	return NewESBackend(s.config.SearchSettings, INDEX_NAME_POSTS).deleteESPost(esPost)
}
