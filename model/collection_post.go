package model

import (
	"encoding/json"
	"net/http"
)

type CollectionPost struct {
	CollectionId string `db:"CollectionId" json:"collection_id"`
	PostId       string `db:"PostId" json:"post_id"`
}

func (o *CollectionPost) IsValid() *AppError {
	if len(o.CollectionId) != 26 {
		return NewAppError("CollectionPost.IsValid", "model.collection_post.is_valid.collection_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(o.PostId) != 26 {
		return NewAppError("CollectionPost.IsValid", "model.collection_post.is_valid.post_id.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

func (o *CollectionPost) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

type CollectionPosts []CollectionPost

func (o *CollectionPosts) ToJson() string {
	if b, err := json.Marshal(o); err != nil {
		return "[]"
	} else {
		return string(b)
	}
}
