package sqlstore

import (
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlUserPointHistoryStore struct {
	store.Store
}

func NewSqlUserPointHistoryStore(sqlStore store.Store) store.UserPointHistoryStore {
	s := &SqlUserPointHistoryStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.UserPointHistory{}, "UserPointHistory").SetKeys(false, "Id")
	}

	return s
}

func (s *SqlUserPointHistoryStore) GetUserPointHistoryBeforeTime(time int64, userId string, page, perPage int, teamId string) ([]*model.UserPointHistory, *model.AppError) {
	offset := page * perPage

	query := s.GetQueryBuilder().Select("u.*")
	query = query.From("UserPointHistory u").
		Where(sq.And{
			sq.Expr(`UserId = ?`, userId),
			sq.Expr(`CreateAt <= ?`, time),
		})

	// TODO: 問題無い？
	if teamId != "" {
		query = query.Where(sq.And{
			sq.Expr(`TeamId = ?`, teamId),
		})
	} else {
		query = query.Where("TeamId IS NULL")
	}

	query = query.OrderBy("CreateAt DESC").
		Limit(uint64(perPage)).
		Offset(uint64(offset))

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlUserPointHistoryStore.GetUserPointHistoryBeforeTime", "store.sql_user_point_history.get_user_point_history_before_time.app_error", nil, "", http.StatusInternalServerError)
	}

	var history []*model.UserPointHistory
	_, err = s.GetReplica().Select(&history, queryString, args...)
	if err != nil {
		return nil, model.NewAppError("SqlUserPointHistoryStore.GetUserPointHistoryBeforeTime", "store.sql_user_point_history.get_user_point_history_before_time.app_error", nil, "", http.StatusInternalServerError)
	}

	return history, nil
}

func (s *SqlUserPointHistoryStore) TopAskersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopUserByTagResult, *model.AppError) {
	return nil, nil
}

func (s *SqlUserPointHistoryStore) TopAnswerersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopUserByTagResult, *model.AppError) {
	return nil, nil
}

func (s *SqlUserPointHistoryStore) TopAnswersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopPostByTagResult, *model.AppError) {
	return nil, nil
}
