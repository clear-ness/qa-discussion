package sqlstore

import (
	"database/sql"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlVoteStore struct {
	store.Store
}

func NewSqlVoteStore(sqlStore store.Store) store.VoteStore {
	s := &SqlVoteStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Vote{}, "Votes").SetKeys(false, "UserId", "Type", "PostId")
	}

	return s
}

func (s *SqlVoteStore) GetVotesBeforeTime(time int64, userId string, page, perPage int, excludeFlag bool, getCount bool) ([]*model.Vote, int64, *model.AppError) {
	queryString, args, err := s.getVotesBeforeTime(time, userId, page, perPage, excludeFlag, false).ToSql()
	if err != nil {
		return nil, int64(0), model.NewAppError("SqlPostStore.GetVotesBeforeTime", "store.sql_post.get_votes_before_time.get.app_error", nil, "", http.StatusInternalServerError)
	}

	var votes []*model.Vote
	_, err = s.GetReplica().Select(&votes, queryString, args...)
	if err != nil {
		return nil, int64(0), model.NewAppError("SqlVoteStore.GetVotesBeforeTime", "store.sql_vote.get_votes_before_time.get.app_error", nil, "", http.StatusInternalServerError)
	}

	var totalCount int64

	if getCount {
		queryString, args, err = s.getVotesBeforeTime(time, userId, page, perPage, excludeFlag, true).ToSql()
		if err != nil {
			return nil, int64(0), model.NewAppError("SqlVoteStore.GetVotesBeforeTime", "store.sql_vote.get_votes_before_time.get.app_error", nil, "", http.StatusInternalServerError)
		}
		if totalCount, err = s.GetReplica().SelectInt(queryString, args...); err != nil {
			return nil, int64(0), model.NewAppError("SqlVoteStore.GetVotesBeforeTime", "store.sql_vote.get_votes_before_time.get.app_error", nil, "", http.StatusInternalServerError)
		}
	} else {
		totalCount = int64(0)
	}

	return votes, totalCount, nil
}

func (s *SqlVoteStore) getVotesBeforeTime(time int64, userId string, page, perPage int, excludeFlag bool, countQuery bool) sq.SelectBuilder {
	var selectStr string
	if countQuery {
		selectStr = "count(*)"
	} else {
		selectStr = "v.*"
	}

	query := s.GetQueryBuilder().Select(selectStr)
	query = query.From("Votes v").
		Where(sq.And{
			sq.Expr(`UserId = ?`, userId),
			sq.Expr(`CreateAt <= ?`, time),
		})

	if excludeFlag {
		query = query.Where(sq.And{
			sq.Expr(`Type != ?`, model.VOTE_TYPE_FLAG),
		})
	} else {
		// for index
		query = query.Where(sq.And{
			sq.Expr(`Type IN (?, ?, ?)`, model.VOTE_TYPE_UP_VOTE, model.VOTE_TYPE_DOWN_VOTE, model.VOTE_TYPE_FLAG),
		})
	}

	if !countQuery {
		offset := page * perPage
		query = query.OrderBy("CreateAt DESC").
			Limit(uint64(perPage)).
			Offset(uint64(offset))
	}

	return query
}

func (s *SqlVoteStore) GetByPostIdForUser(userId string, postId string, voteType string) (*model.Vote, *model.AppError) {
	var vote *model.Vote
	if err := s.GetReplica().SelectOne(&vote, "SELECT * FROM Votes WHERE UserId = :UserId AND PostId = :PostId AND Type = :Type", map[string]interface{}{"UserId": userId, "PostId": postId, "Type": voteType}); err != nil {
		if err != sql.ErrNoRows {
			return nil, model.NewAppError("SqlVoteStore.GetByPostIdForUser", "store.sql_vote.get_by_post_id_for_user.get.app_error", nil, "", http.StatusNotFound)
		}
	}

	return vote, nil
}
