package sqlstore

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/clear-ness/qa-discussion/utils"
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

func (s *SqlVoteStore) GetVotesBeforeTime(time int64, userId string, page, perPage int, excludeFlag bool, getCount bool, teamId string) ([]*model.Vote, int64, *model.AppError) {
	queryString, args, err := s.getVotesBeforeTime(time, userId, page, perPage, excludeFlag, false, teamId).ToSql()
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
		queryString, args, err = s.getVotesBeforeTime(time, userId, page, perPage, excludeFlag, true, teamId).ToSql()
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

func (s *SqlVoteStore) getVotesBeforeTime(time int64, userId string, page, perPage int, excludeFlag bool, countQuery bool, teamId string) sq.SelectBuilder {
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

	// TODO: 問題無い？
	if teamId != "" {
		query = query.Where(sq.And{
			sq.Expr(`TeamId = ?`, teamId),
		})
	} else {
		query = query.Where("TeamId IS NULL")
	}

	if excludeFlag {
		query = query.Where(sq.And{
			sq.Expr(`Type IN (?, ?)`, model.VOTE_TYPE_UP_VOTE, model.VOTE_TYPE_DOWN_VOTE),
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

func (s *SqlVoteStore) GetVoteTypesForPost(userId string, postId string) ([]string, *model.AppError) {
	var votes []*model.Vote
	if _, err := s.GetReplica().Select(&votes, "SELECT * FROM Votes WHERE UserId = :UserId AND PostId = :PostId", map[string]interface{}{"UserId": userId, "PostId": postId}); err != nil {
		if err != sql.ErrNoRows {
			return nil, model.NewAppError("SqlVoteStore.GetAllVotesForPost", "store.sql_vote.get_all_votes_for_post.get.app_error", nil, "", http.StatusNotFound)
		}
	}

	types := []string{}
	for _, v := range votes {
		types = append(types, v.Type)
	}

	return types, nil
}

func (s *SqlVoteStore) CreateReviewVote(post *model.Post, userId string, tagContents string, revision int64) (*model.Vote, *model.AppError) {
	curTime := model.GetMillis()

	review := &model.Vote{
		PostId:       post.Id,
		UserId:       userId,
		Type:         model.VOTE_TYPE_REVIEW,
		Tags:         tagContents,
		TeamId:       post.TeamId,
		FirstPostRev: int(revision),
		CreateAt:     curTime,
	}

	if err := s.GetMaster().Insert(review); err != nil {
		return nil, model.NewAppError("SqlVoteStore.CreateReviewVote", "store.sql_vote.create_review_vote.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return review, nil
}

func (s *SqlVoteStore) GetRejectedReviewsCount(postId string, currentRevision int64) (int64, *model.AppError) {
	count, err := s.GetReplica().SelectInt(`
		SELECT
			count(*)
		FROM
			Votes
		WHERE
			Type IN (:Type1, :Type2, :Type3)
			AND PostId = :PostId
			AND LastPostRev = :LastPostRev
			AND RejectedAt >  0
		`,
		map[string]interface{}{"PostId": postId, "Type1": model.VOTE_TYPE_REVIEW, "Type2": model.VOTE_TYPE_FLAG, "Type3": model.VOTE_TYPE_SYSTEM, "LastPostRev": currentRevision})

	if err != nil {
		return 0, model.NewAppError("SqlUserFavoritePostStore.GetCountByPostId", "store.sql_user_favorite_post.get_count_by_post_id.app_error", nil, "", http.StatusNotFound)
	}

	return count, nil
}

func (s *SqlVoteStore) RejectReviewsForPost(postId string, rejectedBy string, revision int64) *model.AppError {
	curTime := model.GetMillis()

	if _, err := s.GetMaster().Exec("UPDATE Votes SET RejectedAt = :RejectedAt, RejectedBy = :RejectedBy, LastPostRev = :LastPostRev WHERE PostId = :PostId AND Type IN (:Type1, :Type2, :Type3) AND InvalidateAt = 0  AND CompletedAt = 0 AND RejectedAt = 0", map[string]interface{}{"RejectedAt": curTime, "RejectedBy": rejectedBy, "LastPostRev": revision, "PostId": postId, "Type1": model.VOTE_TYPE_REVIEW, "Type2": model.VOTE_TYPE_FLAG, "Type3": model.VOTE_TYPE_SYSTEM}); err != nil {
		return model.NewAppError("SqlVoteStore.RejectReviewsForPost", "store.sql_vote.reject_reviews_for_post.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlVoteStore) CompleteReviewsForPost(postId string, completedBy string, revision int64) *model.AppError {
	curTime := model.GetMillis()

	if _, err := s.GetMaster().Exec("UPDATE Votes SET CompletedAt = :CompletedAt, CompletedBy = :CompletedBy, LastPostRev = :LastPostRev WHERE PostId = :PostId AND Type IN (:Type1, :Type2, :Type3) AND InvalidateAt = 0  AND CompletedAt = 0 AND RejectedAt = 0", map[string]interface{}{"CompletedAt": curTime, "CompletedBy": completedBy, "LastPostRev": revision, "PostId": postId, "Type1": model.VOTE_TYPE_REVIEW, "Type2": model.VOTE_TYPE_FLAG, "Type3": model.VOTE_TYPE_SYSTEM}); err != nil {
		return model.NewAppError("SqlVoteStore.RejectReviewsForPost", "store.sql_vote.reject_reviews_for_post.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlVoteStore) GetReviews(options *model.SearchReviewsOptions, getCount bool) ([]*model.Vote, int64, *model.AppError) {
	queryString, args, err := s.searchReviews(options, false).ToSql()
	if err != nil {
		return nil, int64(0), model.NewAppError("SqlVoteStore.GetReviews", "store.sql_vote.get_reviews.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	var votes []*model.Vote
	_, err = s.GetMaster().Select(&votes, queryString, args...)
	if err != nil {
		return nil, int64(0), model.NewAppError("SqlVoteStore.GetReviews", "store.sql_vote.get_reviews.select.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	totalCount := int64(0)
	if getCount {
		queryString, args, err = s.searchReviews(options, true).ToSql()
		if err != nil {
			return nil, int64(0), model.NewAppError("SqlVoteStore.GetReviews", "store.sql_vote.get_reviews.get.app_error", nil, "", http.StatusInternalServerError)
		}
		if totalCount, err = s.GetMaster().SelectInt(queryString, args...); err != nil {
			return nil, int64(0), model.NewAppError("SqlVoteStore.GetReviews", "store.sql_vote.get_reviews.get.app_error", nil, "", http.StatusInternalServerError)
		}
	}

	return votes, totalCount, nil
}

func (s *SqlVoteStore) searchReviews(options *model.SearchReviewsOptions, countQuery bool) sq.SelectBuilder {
	offset := options.Page * options.PerPage

	var selectStr string
	if countQuery {
		selectStr = "count(*)"
	} else {
		selectStr = "v.*"
	}

	query := s.GetQueryBuilder().Select(selectStr)
	query = query.From("Votes v")

	if options.ReviewType != "" {
		query = query.Where(sq.And{
			sq.Expr(`Type = ?`, options.ReviewType),
		})
	} else {
		query = query.Where(sq.And{
			sq.Expr(`Type IN (?, ?, ?)`, model.VOTE_TYPE_FLAG, model.VOTE_TYPE_REVIEW, model.VOTE_TYPE_SYSTEM),
		})
	}

	if options.TeamId != "" {
		query = query.Where(sq.And{
			sq.Expr(`TeamId = ?`, options.TeamId),
		})
	} else {
		query = query.Where("TeamId IS NULL")
	}

	if options.FromDate != 0 {
		query = query.Where(sq.And{
			sq.Expr(`CreateAt >= ?`, options.FromDate),
		})
	}
	if options.ToDate != 0 {
		query = query.Where(sq.And{
			sq.Expr(`CreateAt <= ?`, options.ToDate),
		})
	}

	if !options.IncludeCompleted {
		query = query.Where(sq.And{
			sq.Eq{"CompletedAt": int(0)},
		})
	}

	if !options.IncludeRejected {
		query = query.Where(sq.And{
			sq.Eq{"RejectedAt": int(0)},
		})
	}

	if !options.IncludeInvalidated {
		query = query.Where(sq.And{
			sq.Eq{"InvalidateAt": int(0)},
		})
	}

	if options.PostId != "" {
		query = query.Where(sq.And{
			sq.Expr(`PostId = ?`, options.PostId),
		})
	}

	if options.UserId != "" {
		query = query.Where(sq.And{
			sq.Expr(`UserId = ?`, options.UserId),
		})
	}
	//} else {
	//    // for system vote
	//    query = query.Where(sq.And{
	//        sq.Eq{"UserId": ""},
	//    })
	//}

	orderBy := "CreateAt DESC"

	terms := options.Tagged
	for _, c := range specialSearchChar {
		terms = strings.Replace(terms, c, " ", -1)
	}

	if options.Tagged != "" {
		searchColumns := "Tags"
		fulltextClause := fmt.Sprintf("MATCH(%s) AGAINST (? IN BOOLEAN MODE)", searchColumns)

		splitTerms := []string{}
		for _, t := range strings.Fields(terms) {
			if len(t) >= model.TAG_MIN_RUNES {
				splitTerms = append(splitTerms, "+"+t)
			}
		}
		terms = strings.Join(splitTerms, " ")

		query = query.Where(sq.And{
			sq.Expr(fulltextClause, terms),
		})
	}

	if !countQuery {
		query = query.OrderBy(orderBy)
		query = query.Limit(uint64(options.PerPage)).Offset(uint64(offset))
	}

	return query
}

func (s *SqlVoteStore) AnalyticsVoteCounts(teamId string, voteType string) (model.Analytics, *model.AppError) {
	query :=
		`SELECT
		        DATE(FROM_UNIXTIME(Votes.CreateAt / 1000)) AS Name,
		        COUNT(Votes.PostId) AS Value
		    FROM Votes`

	if len(teamId) > 0 {
		query += " WHERE TeamId = :TeamId AND"
	} else {
		query += " WHERE"
	}

	if len(voteType) > 0 {
		query += " Type = :Type AND"
	}

	query += ` Votes.CreateAt <= :EndTime
		            AND Votes.CreateAt >= :StartTime
		GROUP BY DATE(FROM_UNIXTIME(Votes.CreateAt / 1000))
		ORDER BY Name DESC
		LIMIT 30`

	end := utils.MillisFromTime(utils.EndOfDay(utils.Yesterday()))
	start := utils.MillisFromTime(utils.StartOfDay(utils.Yesterday().AddDate(0, 0, -31)))

	var rows model.Analytics
	_, err := s.GetReplica().Select(
		&rows,
		query,
		map[string]interface{}{"TeamId": teamId, "StartTime": start, "EndTime": end, "Type": voteType})
	if err != nil {
		return nil, model.NewAppError("SqlVoteStore.AnalyticsVoteCounts", "store.sql_vote.analytics_vote_counts.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return rows, nil
}
