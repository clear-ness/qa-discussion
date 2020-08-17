package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

func (api *API) InitCollection() {
	api.BaseRoutes.Collections.Handle("", api.ApiSessionRequired(createCollection)).Methods("POST")

	api.BaseRoutes.CollectionsForTeam.Handle("", api.ApiSessionRequired(getCollectionsForTeam)).Methods("GET")
	// TODO: collection.titleでも検索出来る様に(full text)

	//api.BaseRoutes.Collection.Handle("", api.ApiSessionRequired(getCollection)).Methods("GET")
	//api.BaseRoutes.Collection.Handle("", api.ApiSessionRequired(updateCollection)).Methods("PUT")
	api.BaseRoutes.Collection.Handle("", api.ApiSessionRequired(deleteCollection)).Methods("DELETE")

	// Add a post to a collection by creating a collection-post object.
	api.BaseRoutes.CollectionPosts.Handle("", api.ApiSessionRequired(addCollectionPost)).Methods("POST")
	// 特定collectionに所属するpostsを取得
	api.BaseRoutes.CollectionPosts.Handle("", api.ApiSessionRequired(getCollectionPosts)).Methods("GET")
	// 特定collectionに所属する特定postを削除
	api.BaseRoutes.CollectionPost.Handle("", api.ApiSessionRequired(removeCollectionPost)).Methods("DELETE")
}

func createCollection(c *Context, w http.ResponseWriter, r *http.Request) {
	collection := model.CollectionFromJson(r.Body)
	if collection == nil {
		c.SetInvalidParam("collection")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, collection.TeamId, model.PERMISSION_CREATE_COLLECTION) {
		c.SetPermissionError(model.PERMISSION_CREATE_COLLECTION)
		return
	}

	col, err := c.App.CreateCollectionWithUser(collection, c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(col.ToJson()))
}

func addCollectionPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId().RequireCollectionId()
	if c.Err != nil {
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil {
		c.Err = err
		return
	}

	if post.TeamId == "" || !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_MANAGE_COLLECTION_POSTS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_COLLECTION_POSTS)
		return
	}

	col, err := c.App.GetCollection(c.Params.CollectionId)
	if err != nil {
		c.Err = err
		return
	}

	if _, err = c.App.GetCollectionPost(col.Id, post.Id); err != nil {
		if err.Id != store.MISSING_COLLECTION_POST_ERROR {
			c.Err = err
			return
		}
	}

	cp, err := c.App.AddCollectionPost(post.Id, col)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(cp.ToJson()))
}

func getCollectionPosts(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireCollectionId()
	if c.Err != nil {
		return
	}

	col, err := c.App.GetCollection(c.Params.CollectionId)
	if err != nil {
		c.Err = err
		return
	}

	team, err := c.App.GetTeam(col.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, team.Id, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	colPosts, err := c.App.GetCollectionPostsPage(c.Params.CollectionId, c.Params.Page, c.Params.PerPage)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(colPosts.ToJson()))
}

func getCollectionsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	team, err := c.App.GetTeam(c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, team.Id, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	cols, err := c.App.GetCollectionsForTeam(c.Params.TeamId, c.Params.Page*c.Params.PerPage, c.Params.PerPage)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(cols.ToJson()))
}

func removeCollectionPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireCollectionId().RequirePostId()
	if c.Err != nil {
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil {
		c.Err = err
		return
	}

	if post.TeamId == "" || !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_MANAGE_COLLECTION_POSTS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_COLLECTION_POSTS)
		return
	}

	if err = c.App.RemovePostFromCollection(c.Params.CollectionId, c.Params.PostId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func deleteCollection(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireCollectionId()
	if c.Err != nil {
		return
	}

	col, err := c.App.GetCollection(c.Params.CollectionId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, col.TeamId, model.PERMISSION_DELETE_COLLECTION) {
		c.SetPermissionError(model.PERMISSION_DELETE_COLLECTION)
		return
	}

	err = c.App.DeleteCollection(col)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
