package app

import (
	"net/http"
	"sort"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetUserFavoritePostsForUser(toDate int64, userId string, page, perPage int, limitContent bool, teamId string) ([]*model.UserFavoritePostWithPost, int64, *model.AppError) {
	favoritePosts, totalCount, err := a.Srv.Store.UserFavoritePost().GetUserFavoritePostsBeforeTime(toDate, userId, page, perPage, true, teamId)
	if err != nil {
		return nil, int64(0), err
	}

	postIdsMaps := map[string]bool{}
	for _, favoritePost := range favoritePosts {
		postIdsMaps[favoritePost.PostId] = true
	}

	var postIds []string
	for key := range postIdsMaps {
		postIds = append(postIds, key)
	}

	posts, err := a.Srv.Store.Post().GetPostsByIds(postIds)
	if err != nil {
		return nil, 0, err
	}

	option := model.SetPostMetadataOptions{
		SetUser:       true,
		SetComments:   false,
		SetBestAnswer: false,
		SetParent:     false,
	}
	posts, err = a.SetPostMetadata(posts, option)
	if err != nil {
		return nil, 0, err
	}

	if limitContent {
		posts.LimitContentLength()
	}

	postMap := map[string]*model.Post{}
	for _, post := range posts {
		postMap[post.Id] = post
	}

	var uPosts []*model.UserFavoritePostWithPost
	for _, favoritePost := range favoritePosts {
		if post, ok := postMap[favoritePost.PostId]; ok {
			upost := &model.UserFavoritePostWithPost{
				favoritePost,
				post,
			}
			uPosts = append(uPosts, upost)
		}
	}

	sort.Slice(uPosts, func(i, j int) bool {
		return uPosts[i].CreateAt > uPosts[j].CreateAt
	})

	return uPosts, totalCount, err
}

func (a *App) GetUserFavoritePostForUser(userId string, postId string) (*model.UserFavoritePost, *model.AppError) {
	return a.Srv.Store.UserFavoritePost().GetByPostIdForUser(userId, postId)
}

func (a *App) GetUserFavoritePostsCountByPostId(postId string) (int64, *model.AppError) {
	return a.Srv.Store.UserFavoritePost().GetCountByPostId(postId)
}

func (a *App) CreateUserFavoritePost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId)
	if err != nil {
		mlog.Error("Couldn't create the favorite post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("CreateUserFavoritePost", "api.user_favorite_post.create.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.Type != model.POST_TYPE_QUESTION {
		return model.NewAppError("CreateUserFavoritePost", "api.user_favorite_post.create.get.app_error", nil, "", http.StatusInternalServerError)
	}

	return a.Srv.Store.UserFavoritePost().Save(postId, userId, post.TeamId)
}

func (a *App) DeleteUserFavoritePost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId)
	if err != nil {
		mlog.Error("Couldn't delete the favorite post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("DeleteUserFavoritePost", "api.user_favorite_post.delete.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.Type != model.POST_TYPE_QUESTION {
		return model.NewAppError("DeleteUserFavoritePost", "api.user_favorite_post.delete.get.app_error", nil, "", http.StatusInternalServerError)
	}

	return a.Srv.Store.UserFavoritePost().Delete(postId, userId)
}
