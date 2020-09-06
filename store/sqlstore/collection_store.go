package sqlstore

import (
	"database/sql"
	"net/http"

	"github.com/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlCollectionStore struct {
	store.Store

	postsQuery sq.SelectBuilder
}

func NewSqlCollectionStore(sqlStore store.Store) store.CollectionStore {
	s := &SqlCollectionStore{
		Store: sqlStore,
	}

	s.postsQuery = s.GetQueryBuilder().Select("CollectionPosts.*").From("CollectionPosts")

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Collection{}, "Collections").SetKeys(false, "Id")

		db.AddTableWithName(model.CollectionPost{}, "CollectionPosts").SetKeys(false, "CollectionId", "PostId")
	}

	return s
}

func collectionPostSliceColumns() []string {
	return []string{"CollectionId", "PostId"}
}

func collectionPostToSlice(colPost *model.CollectionPost) []interface{} {
	resultSlice := []interface{}{}
	resultSlice = append(resultSlice, colPost.CollectionId)
	resultSlice = append(resultSlice, colPost.PostId)

	return resultSlice
}

func (s SqlCollectionStore) GetTeamCollections(teamId string) (*model.CollectionList, *model.AppError) {
	data := &model.CollectionList{}
	_, err := s.GetReplica().Select(data, "SELECT * FROM Collections WHERE TeamId = :TeamId", map[string]interface{}{"TeamId": teamId})
	if err != nil {
		return nil, model.NewAppError("SqlCollectionStore.GetTeamCollections", "store.sql_collection.get_collections.get.app_error", nil, "teamId="+teamId+",  err="+err.Error(), http.StatusInternalServerError)
	}

	if len(*data) == 0 {
		return nil, model.NewAppError("SqlCollectionStore.GetTeamCollections", "store.sql_collection.get_collections.not_found.app_error", nil, "teamId="+teamId, http.StatusNotFound)
	}

	return data, nil
}

func (s SqlCollectionStore) Save(collection *model.Collection, maxCollectionsPerTeam int64) (*model.Collection, *model.AppError) {
	if collection.DeleteAt != 0 {
		return nil, model.NewAppError("SqlCollectionStore.Save", "store.sql_collection.save.already_deleted.app_error", nil, "", http.StatusInternalServerError)
	}

	if len(collection.Id) > 0 {
		return nil, model.NewAppError("SqlCollectionStore.Save", "store.sql_collection.save.existing.app_error", nil, "", http.StatusBadRequest)
	}

	collection.PreSave()
	if err := collection.IsValid(); err != nil {
		return nil, err
	}

	if maxCollectionsPerTeam >= 0 {
		if count, err := s.GetReplica().SelectInt(`SELECT count(*) FROM Collections WHERE TeamId = :TeamId AND DeleteAt = 0`, map[string]interface{}{"TeamId": collection.TeamId}); err != nil {
			return nil, model.NewAppError("SqlCollectionStore.Save", "store.sql_collection.save.get_count.app_error", nil, "", http.StatusInternalServerError)
		} else if count >= maxCollectionsPerTeam {
			return nil, model.NewAppError("SqlCollectionStore.Save", "store.sql_collection.save.too_many.app_error", nil, "", http.StatusBadRequest)
		}
	}

	if err := s.GetMaster().Insert(collection); err != nil {
		if IsUniqueConstraintError(err, []string{"PRIMARY"}) {
			return nil, model.NewAppError("SqlCollectionStore.Save", "store.sql_collection.save.exists.app_error", nil, err.Error(), http.StatusBadRequest)
		}

		return nil, model.NewAppError("SqlCollectionStore.Save", "store.sql_collection.save.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return collection, nil
}

func (s SqlCollectionStore) Get(id string) (*model.Collection, *model.AppError) {
	obj, err := s.GetReplica().Get(model.Collection{}, id)
	if err != nil {
		return nil, model.NewAppError("SqlCollectionStore.Get", "store.sql_collection.get.get.app_error", nil, "id="+id, http.StatusInternalServerError)
	}
	if obj == nil {
		return nil, model.NewAppError("SqlCollectionStore.Get", "store.sql_collection.get.not_found.app_error", nil, "id="+id, http.StatusNotFound)
	}

	col := obj.(*model.Collection)
	return col, nil
}

func (s SqlCollectionStore) GetPost(collectionId string, postId string) (*model.CollectionPost, *model.AppError) {
	var colPost *model.CollectionPost
	if err := s.GetReplica().SelectOne(&colPost, "SELECT * FROM CollectionPosts WHERE CollectionPosts.CollectionId = :CollectionId AND CollectionPosts.PostId = :PostId", map[string]interface{}{"CollectionId": collectionId, "PostId": postId}); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlCollectionStore.GetPost", store.MISSING_COLLECTION_POST_ERROR, nil, "collection_id="+collectionId+"post_id="+postId+","+err.Error(), http.StatusNotFound)
		}

		return nil, model.NewAppError("SqlCollectionStore.GetPost", "store.sql_collection.get_post.app_error", nil, "collection_id="+collectionId+"post_id="+postId+","+err.Error(), http.StatusInternalServerError)
	}

	return colPost, nil
}

func (s SqlCollectionStore) GetPosts(collectionId string, offset, limit int) (*model.CollectionPosts, *model.AppError) {
	colPosts := &model.CollectionPosts{}
	_, err := s.GetReplica().Select(colPosts, `
		SELECT
			CollectionPosts.*
		FROM
			CollectionPosts
		INNER JOIN
			Collections ON CollectionPosts.CollectionId = Collections.Id
		WHERE
			Collections.DeleteAt = 0 AND
			CollectionPosts.CollectionId = :CollectionId
		LIMIT :Limit
		OFFSET :Offset`, map[string]interface{}{"CollectionId": collectionId, "Limit": limit, "Offset": offset})
	if err != nil {
		return nil, model.NewAppError("SqlCollectionStore.GetPosts", "store.sql_collection.get_posts.app_error", nil, "collection_id="+collectionId+","+err.Error(), http.StatusInternalServerError)
	}

	return colPosts, nil
}

func (s SqlCollectionStore) SaveMultiplePosts(colPosts []*model.CollectionPost) ([]*model.CollectionPost, *model.AppError) {
	colPosts, err := s.saveMultiplePosts(colPosts)
	if err != nil {
		return nil, model.NewAppError("SaveMultiplePosts", "app.collection.save_multiple.internal_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return colPosts, nil
}

func (s SqlCollectionStore) SavePost(colPost *model.CollectionPost) (*model.CollectionPost, *model.AppError) {
	newColPosts, appErr := s.SaveMultiplePosts([]*model.CollectionPost{colPost})
	if appErr != nil {
		return nil, appErr
	}
	return newColPosts[0], nil
}

func (s SqlCollectionStore) saveMultiplePosts(colPosts []*model.CollectionPost) ([]*model.CollectionPost, error) {
	query := s.GetQueryBuilder().Insert("CollectionPosts").Columns(collectionPostSliceColumns()...)
	for _, colPost := range colPosts {
		if err := colPost.IsValid(); err != nil {
			return nil, err
		}
		query = query.Values(collectionPostToSlice(colPost)...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "collection_posts_tosql")
	}
	if _, err := s.GetMaster().Exec(sql, args...); err != nil {
		return nil, errors.Wrap(err, "collection_posts_save")
	}

	return colPosts, nil
}

func (s SqlCollectionStore) GetCollectionsForTeam(teamId string, offset int, limit int, title string) (*model.CollectionList, *model.AppError) {
	args := map[string]interface{}{
		"TeamId": teamId,
		"Limit":  limit,
		"Offset": offset,
	}

	fulltextClause := ""
	if title != "" {
		fulltextClause = "AND MATCH(Title) AGAINST (:Title IN BOOLEAN MODE)"
		args["Title"] = title
	}

	cols := &model.CollectionList{}
	_, err := s.GetReplica().Select(cols, `
		SELECT
			Collections.*
		FROM
			Collections
		WHERE
			Collections.TeamId = :TeamId
			AND Collections.DeleteAt = 0
			`+fulltextClause+`
		ORDER BY Collections.CreateAt DESC
		LIMIT :Limit
		OFFSET :Offset
		`, args)
	if err != nil {
		return nil, model.NewAppError("SqlCollectionStore.GetCollectionsForTeam", "store.sql_collection.get_collections_for_team.app_error", nil, "teamId="+teamId+", err="+err.Error(), http.StatusInternalServerError)
	}

	return cols, nil
}

func (s SqlCollectionStore) RemovePosts(collectionId string, postIds []string) *model.AppError {
	query := s.GetQueryBuilder().
		Delete("CollectionPosts").
		Where(sq.Eq{"CollectionId": collectionId}).
		Where(sq.Eq{"PostId": postIds})

	sql, args, err := query.ToSql()
	if err != nil {
		return model.NewAppError("SqlCollectionStore.RemovePosts", "store.sql_collection.remove_posts.app_error", nil, "collectionId="+collectionId+", err="+err.Error(), http.StatusInternalServerError)
	}

	_, err = s.GetMaster().Exec(sql, args...)
	if err != nil {
		return model.NewAppError("SqlCollectionStore.RemovePosts", "store.sql_collection.remove_posts.app_error", nil, "collectionId="+collectionId+", err="+err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s SqlCollectionStore) RemovePost(collectionId string, postId string) *model.AppError {
	return s.RemovePosts(collectionId, []string{postId})
}

func (s SqlCollectionStore) Delete(collectionId string, time int64) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlCollectionStore.DeleteCollection", "store.sql_collection.delete_collection.app_error", nil, "id="+collectionId+", err="+errMsg, http.StatusInternalServerError)
	}

	var col *model.Collection
	err := s.GetReplica().SelectOne(&col, "SELECT * FROM Collections WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": collectionId})
	if err != nil {
		return appErr(err.Error())
	}

	// TODO: collectionPostsは同時に削除しなくていい？
	if _, err := s.GetMaster().Exec("UPDATE Collections SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"DeleteAt": time, "UpdateAt": time, "Id": collectionId}); err != nil {
		return model.NewAppError("SqlCollectionStore.DeleteCollection", "store.sql_collection.delete_collection.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
