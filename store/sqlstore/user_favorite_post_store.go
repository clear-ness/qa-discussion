package sqlstore

import (
	"database/sql"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlUserFavoritePostStore struct {
	store.Store
}

func NewSqlUserFavoritePostStore(sqlStore store.Store) store.UserFavoritePostStore {
	s := &SqlUserFavoritePostStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.UserFavoritePost{}, "UserFavoritePosts").SetKeys(false, "PostId", "UserId")
	}

	return s
}

func (s *SqlUserFavoritePostStore) GetByPostIdForUser(userId string, postId string) (*model.UserFavoritePost, *model.AppError) {
	var favoritePost *model.UserFavoritePost
	if err := s.GetReplica().SelectOne(&favoritePost, "SELECT * FROM UserFavoritePosts WHERE UserId = :UserId AND PostId = :PostId", map[string]interface{}{"UserId": userId, "PostId": postId}); err != nil {
		if err != sql.ErrNoRows {
			return nil, model.NewAppError("SqlUserFavoritePostStore.GetByPostIdForUser", "store.sql_user_favorite_post.get_by_post_id_for_me.app_error", nil, "", http.StatusInternalServerError)
		}
	}

	return favoritePost, nil
}

func (s *SqlUserFavoritePostStore) GetCountByPostId(postId string) (int64, *model.AppError) {
	count, err := s.GetReplica().SelectInt(`
		SELECT
			count(*)
		FROM
			UserFavoritePosts
		WHERE
			PostId = :PostId`,
		map[string]interface{}{"PostId": postId})

	if err != nil {
		return 0, model.NewAppError("SqlUserFavoritePostStore.GetCountByPostId", "store.sql_user_favorite_post.get_count_by_post_id.app_error", nil, "", http.StatusNotFound)
	}

	return count, nil
}

func (s *SqlUserFavoritePostStore) GetUserFavoritePostsBeforeTime(time int64, userId string, page, perPage int, getCount bool, teamId string) ([]*model.UserFavoritePost, int64, *model.AppError) {
	queryString, args, err := s.getUserFavoritePostsBeforeTime(time, userId, page, perPage, false, teamId).ToSql()
	if err != nil {
		return nil, int64(0), model.NewAppError("SqlUserFavoritePostStore.GetUserFavoritePostsBeforeTime", "store.sql_user_favorite_post.get_user_favorite_posts_before_time.get.app_error", nil, "", http.StatusInternalServerError)
	}

	var favoritePosts []*model.UserFavoritePost
	if _, err = s.GetReplica().Select(&favoritePosts, queryString, args...); err != nil {
		return nil, int64(0), model.NewAppError("SqlUserFavoritePostStore.GetUserFavoritePostsBeforeTime", "store.sql_user_favorite_post.get_user_favorite_posts_before_time.get.app_error", nil, "userId="+userId+", err="+err.Error(), http.StatusInternalServerError)
	}

	var totalCount int64

	if getCount {
		queryString, args, err = s.getUserFavoritePostsBeforeTime(time, userId, page, perPage, true, teamId).ToSql()
		if err != nil {
			return nil, int64(0), model.NewAppError("SqlUserFavoritePostStore.GetUserFavoritePostsBeforeTime", "store.sql_user_favorite_post.get_user_favorite_posts_before_time.get.app_error", nil, "", http.StatusInternalServerError)
		}
		if totalCount, err = s.GetReplica().SelectInt(queryString, args...); err != nil {
			return nil, int64(0), model.NewAppError("SqlUserFavoritePostStore.GetUserFavoritePostsBeforeTime", "store.sql_user_favorite_post.get_user_favorite_posts_before_time.get.app_error", nil, "userId="+userId+", err="+err.Error(), http.StatusInternalServerError)
		}
	} else {
		totalCount = int64(0)
	}

	return favoritePosts, totalCount, nil
}

func (s *SqlUserFavoritePostStore) getUserFavoritePostsBeforeTime(time int64, userId string, page, perPage int, countQuery bool, teamId string) sq.SelectBuilder {
	var selectStr string
	if countQuery {
		selectStr = "count(*)"
	} else {
		selectStr = "u.*"
	}

	query := s.GetQueryBuilder().Select(selectStr)
	query = query.From("UserFavoritePosts u").
		Where(sq.And{
			sq.Expr(`UserId = ?`, userId),
			sq.Expr(`CreateAt <= ?`, time),
		})

	// TODO: teamIdが""の場合は問題無い？
	query = query.Where(sq.And{
		sq.Expr(`TeamId = ?`, teamId),
	})

	if !countQuery {
		offset := page * perPage
		query = query.OrderBy("CreateAt DESC").
			Limit(uint64(perPage)).
			Offset(uint64(offset))
	}

	return query
}

func (s *SqlUserFavoritePostStore) Save(postId string, userId string, teamId string) *model.AppError {
	favoritePost := &model.UserFavoritePost{
		PostId: postId,
		UserId: userId,
		TeamId: teamId,
	}

	favoritePost.PreSave()
	if err := favoritePost.IsValid(); err != nil {
		return err
	}

	if err := s.GetMaster().Insert(favoritePost); err != nil {
		return model.NewAppError("SqlUserFavoritePostStore.FavoritePost", "store.sql_user_favorite_post.favoritePost.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlUserFavoritePostStore) Delete(postId string, userId string) *model.AppError {
	if _, err := s.GetMaster().Exec("DELETE FROM UserFavoritePosts WHERE UserId = :UserId AND PostId = :PostId", map[string]interface{}{"UserId": userId, "PostId": postId}); err != nil {
		return model.NewAppError("SqlUserFavoritePostStore.Delete", "store.sql_user_favorite_post.delete.app_error", nil, "", http.StatusInternalServerError)
	}

	return nil
}
