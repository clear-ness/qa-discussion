package searchlayer

import (
	"github.com/clear-ness/qa-discussion/model"
)

type SearchPostStore struct {
	store.PostStore
	rootStore *SearchStore
}

func (s SearchPostStore) indexPost(post *model.Post) {
	s.rootStore.indexPost(post)
}

func (s *SearchPostStore) SaveQuestion(post *model.Post) (*model.Post, *model.AppError) {
	post, err := s.PostStore.SaveQuestion(post)
	if err == nil {
		s.indexPost(post)
	}

	return post, err
}

func (s *SearchPostStore) Update(newPost *model.Post, oldPost *model.Post) (*model.Post, *model.AppError) {
	post, err := s.PostStore.Update(newPost, oldPost)
	if err == nil {
		s.indexPost(post)
	}

	return post, err
}

func (s *SearchPostStore) deletePost(post *model.Post) {
	s.rootStore.deletePost(post)
}

func (s *SearchPostStore) DeleteQuestion(postId string, time int64, deleteById string) *model.AppError {
	err := s.PostStore.DeleteQuestion(postId, time, deleteByID)

	if err == nil {
		// TODO: postIdからdeleted postを取得

		s.deletePost(post)
	}
	return err
}

// TODO: related post search

func (s *SearchPostStore) setupIndex() error {
	mapping := `{
    "mappings": {
      "_doc": {
        "properties": {
          "id":         { "type": "keyword" },
          "image_url":  { "type": "keyword" },
          "title":      { "type": "text", "analyzer": "english" },
          "alt":        { "type": "text", "analyzer": "english" },
          "transcript": { "type": "text", "analyzer": "english" },
          "published":  { "type": "date" },
          "link":       { "type": "keyword" },
          "news":       { "type": "text", "analyzer": "english" }
        }
      }
    }
		}`

	s.rootStore.setupIndex(mapping, INDEX_NAME_POSTS)
}
