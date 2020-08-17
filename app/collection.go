package app

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

func (a *App) CreateCollectionWithUser(collection *model.Collection, userId string) (*model.Collection, *model.AppError) {
	if len(collection.TeamId) == 0 {
		return nil, model.NewAppError("CreateCollectionWithUser", "app.collection.create_collection.no_team_id.app_error", nil, "", http.StatusBadRequest)
	}

	count, err := a.GetNumberOfCollectionsOnTeam(collection.TeamId)
	if err != nil {
		return nil, err
	}

	if int64(count+1) > *a.Config().TeamSettings.MaxCollectionsPerTeam {
		return nil, model.NewAppError("CreateCollectionWithUser", "api.collection.create_collection.max_collection_limit.app_error", map[string]interface{}{"MaxCollectionsPerTeam": *a.Config().TeamSettings.MaxCollectionsPerTeam}, "", http.StatusBadRequest)
	}

	collection.UserId = userId

	rcollection, err := a.CreateCollection(collection, true)
	if err != nil {
		return nil, err
	}

	return rcollection, nil
}

func (a *App) GetNumberOfCollectionsOnTeam(teamId string) (int, *model.AppError) {
	list, err := a.Srv.Store.Collection().GetTeamCollections(teamId)
	if err != nil {
		return 0, err
	}

	return len(*list), nil
}

func (a *App) CreateCollection(collection *model.Collection, addPost bool) (*model.Collection, *model.AppError) {
	// TODO:collection.Titleを事前に(regex)整形しておく

	col, err := a.Srv.Store.Collection().Save(collection, *a.Config().TeamSettings.MaxCollectionsPerTeam)
	if err != nil {
		return nil, err
	}

	return col, nil
}

func (a *App) GetCollection(collectionId string) (*model.Collection, *model.AppError) {
	return a.Srv.Store.Collection().Get(collectionId)
}

func (a *App) GetCollectionPost(collectionId string, postId string) (*model.CollectionPost, *model.AppError) {
	return a.Srv.Store.Collection().GetPost(collectionId, postId)
}

func (a *App) AddCollectionPost(postId string, collection *model.Collection) (*model.CollectionPost, *model.AppError) {
	if colPost, err := a.Srv.Store.Collection().GetPost(collection.Id, postId); err != nil {
		if err.Id != store.MISSING_COLLECTION_POST_ERROR {
			return nil, err
		}
	} else {
		return colPost, nil
	}

	if collection.DeleteAt > 0 {
		return nil, model.NewAppError("AddCollectionPost", "api.collection.add_collection_post.deleted_collection.app_error", nil, "", http.StatusBadRequest)
	}

	var post *model.Post
	var err *model.AppError
	if post, err = a.GetPost(postId); err != nil {
		return nil, err
	}

	if post.TeamId != collection.TeamId {
		return nil, model.NewAppError("AddCollectionPost", "api.collection.add_collection_post.different_team_id.app_error", nil, "", http.StatusBadRequest)
	}

	colPost, err := a.AddPostToCollection(post, collection)
	if err != nil {
		return nil, err
	}

	return colPost, nil
}

func (a *App) AddPostToCollection(post *model.Post, collection *model.Collection) (*model.CollectionPost, *model.AppError) {
	newColPost := &model.CollectionPost{
		CollectionId: collection.Id,
		PostId:       post.Id,
	}

	newColPost, err := a.Srv.Store.Collection().SavePost(newColPost)
	if err != nil {
		return nil, model.NewAppError("AddPostToCollection", "api.collection.add_post.failed.app_error", nil, "", http.StatusInternalServerError)
	}

	return newColPost, nil
}

func (a *App) GetCollectionPostsPage(collectionId string, page, perPage int) (*model.CollectionPosts, *model.AppError) {
	return a.Srv.Store.Collection().GetPosts(collectionId, page*perPage, perPage)
}

func (a *App) GetCollectionsForTeam(teamId string, offset int, limit int) (*model.CollectionList, *model.AppError) {
	return a.Srv.Store.Collection().GetCollectionsForTeam(teamId, offset, limit)
}

func (a *App) RemovePostFromCollection(collectionId string, postId string) *model.AppError {
	if _, err := a.Srv.Store.Collection().GetPost(collectionId, postId); err != nil {
		return err
	}

	collection, err := a.GetCollection(collectionId)
	if err != nil {
		return err
	}

	if collection.DeleteAt > 0 {
		return model.NewAppError("RemoveCollectionPost", "api.collection.removecollection_post.deleted_collection.app_error", nil, "", http.StatusBadRequest)
	}

	var post *model.Post
	if post, err = a.GetPost(postId); err != nil {
		return err
	}

	if post.TeamId != collection.TeamId {
		return model.NewAppError("RemoveCollectionPost", "api.collection.removecollection_post.different_team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if err := a.Srv.Store.Collection().RemovePost(collection.Id, postId); err != nil {
		return err
	}

	return nil
}

func (a *App) DeleteCollection(collection *model.Collection) *model.AppError {
	if collection.DeleteAt > 0 {
		err := model.NewAppError("deleteCollection", "api.collection.delete_collection.deleted.app_error", nil, "", http.StatusBadRequest)
		return err
	}

	deleteAt := model.GetMillis()
	if err := a.Srv.Store.Collection().Delete(collection.Id, deleteAt); err != nil {
		return model.NewAppError("DeleteCollection", "app.collection.delete.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
