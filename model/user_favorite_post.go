package model

import (
	"encoding/json"
	"io"
	"net/http"
)

type UserFavoritePost struct {
	PostId   string `db:"PostId" json:"post_id"`
	UserId   string `db:"UserId" json:"user_id"`
	TeamId   string `db:"TeamId" json:"team_id"`
	CreateAt int64  `db:"CreateAt" json:"create_at"`
}

func (u *UserFavoritePost) PreSave() {
	if u.CreateAt == 0 {
		u.CreateAt = GetMillis()
	}
}

func (u *UserFavoritePost) IsValid() *AppError {
	if len(u.PostId) != 26 {
		return NewAppError("UserFavoritePost.IsValid", "model.user_favorite_post.is_valid.post_id.app_error", nil, "", http.StatusBadRequest)
	}

	if len(u.UserId) != 26 {
		return NewAppError("UserFavoritePost.IsValid", "model.user_favorite_post.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if u.TeamId != "" && len(u.TeamId) != 26 {
		return NewAppError("UserFavoritePost.IsValid", "model.user_favorite_post.is_valid.team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if u.CreateAt == 0 {
		return NewAppError("UserFavoritePost.IsValid", "model.user_favorite_post.is_valid.create_at.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type UserFavoritePostWithPost struct {
	*UserFavoritePost
	Post *Post `json:"post"`
}

type UserFavoritePostsWithCount struct {
	UserFavoritePosts []*UserFavoritePostWithPost `json:"user_favorite_posts"`
	TotalCount        int64                       `json:"total_count"`
}

func (o *UserFavoritePostsWithCount) ToJson() []byte {
	b, _ := json.Marshal(o)
	return b
}

func UserFavoritePostsWithCountFromJson(data io.Reader) *UserFavoritePostsWithCount {
	var o *UserFavoritePostsWithCount
	json.NewDecoder(data).Decode(&o)
	return o
}
