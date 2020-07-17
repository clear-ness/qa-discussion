package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitUserFavoritePost() {
	api.BaseRoutes.UserFavoritePosts.Handle("", api.ApiHandler(getUserFavoritePosts)).Methods("GET")

	api.BaseRoutes.Post.Handle("/favorite", api.ApiSessionRequired(favoritePost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_favorite", api.ApiSessionRequired(cancelFavoritePost)).Methods("POST")
}

func getUserFavoritePosts(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	curTime := model.GetMillis()
	// TODO: sortable by added, creation, votes
	uPosts, totalCount, err := c.App.GetUserFavoritePostsForUser(curTime, c.Params.UserId, c.Params.Page, c.Params.PerPage, true)
	if err != nil {
		c.Err = err
		return
	}

	data := model.UserFavoritePostsWithCount{UserFavoritePosts: uPosts, TotalCount: totalCount}
	w.Write([]byte(data.ToJson()))
}

func favoritePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_FAVORITE_POST) {
		c.SetPermissionError(model.PERMISSION_FAVORITE_POST)
		return
	}

	_, err := c.App.GetSinglePostByType(c.Params.PostId, model.POST_TYPE_QUESTION)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_FAVORITE_POST)
		return
	}

	if err := c.App.CreateUserFavoritePost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func cancelFavoritePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_FAVORITE_POST) {
		c.SetPermissionError(model.PERMISSION_FAVORITE_POST)
		return
	}

	_, err := c.App.GetSinglePostByType(c.Params.PostId, model.POST_TYPE_QUESTION)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_FAVORITE_POST)
		return
	}

	if err := c.App.DeleteUserFavoritePost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
