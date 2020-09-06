package sqlstore

import (
	"net/http"

	sq "github.com/Masterminds/squirrel"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/clear-ness/qa-discussion/utils"
)

type SqlPostViewsHistoryStore struct {
	store.Store
}

func NewSqlPostViewsHistoryStore(sqlStore store.Store) store.PostViewsHistoryStore {
	s := &SqlPostViewsHistoryStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.PostViewsHistory{}, "PostViewsHistory").SetKeys(false, "Id")
	}

	return s
}

func (s *SqlPostViewsHistoryStore) GetViewsHistoryCount(teamId string, fromDate int64, toDate int64) (int64, *model.AppError) {
	query := s.GetQueryBuilder().Select("COUNT(*)")
	query = query.From("PostViewsHistory h")

	if teamId != "" {
		query = query.Where(sq.And{
			sq.Expr(`TeamId = ?`, teamId),
		})
	} else {
		query = query.Where("TeamId IS NULL")
	}

	if fromDate != 0 {
		query = query.Where(sq.And{
			sq.Expr(`CreateAt >= ?`, fromDate),
		})
	}

	if toDate != 0 {
		query = query.Where(sq.And{
			sq.Expr(`CreateAt <= ?`, toDate),
		})
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return int64(0), model.NewAppError("SqlPostViewsHistoryStore.GetViewsHistoryCount", "store.sql_post_views_history.get_views_history_count.app_error", nil, "", http.StatusInternalServerError)
	}

	count := int64(0)
	if count, err = s.GetReplica().SelectInt(queryString, args...); err != nil {
		return int64(0), model.NewAppError("SqlPostViewsHistoryStore.GetViewsHistoryCount", "store.sql_post_views_history.get_views_history_count.app_error", nil, "", http.StatusInternalServerError)
	}

	return count, nil
}

func (s *SqlPostViewsHistoryStore) AnalyticsPostViewsHistoryCounts(teamId string) (model.Analytics, *model.AppError) {
	query :=
		`SELECT
		        DATE(FROM_UNIXTIME(PostViewsHistory.CreateAt / 1000)) AS Name,
		        COUNT(PostViewsHistory.Id) AS Value
		    FROM PostViewsHistory`

	if len(teamId) > 0 {
		query += " WHERE TeamId = :TeamId AND"
	} else {
		query += " WHERE"
	}

	query += ` PostViewsHistory.CreateAt <= :EndTime
		            AND PostViewsHistory.CreateAt >= :StartTime
		GROUP BY DATE(FROM_UNIXTIME(PostViewsHistory.CreateAt / 1000))
		ORDER BY Name DESC
		LIMIT 30`

	end := utils.MillisFromTime(utils.EndOfDay(utils.Yesterday()))
	start := utils.MillisFromTime(utils.StartOfDay(utils.Yesterday().AddDate(0, 0, -31)))

	var rows model.Analytics
	_, err := s.GetReplica().Select(
		&rows,
		query,
		map[string]interface{}{"TeamId": teamId, "StartTime": start, "EndTime": end})
	if err != nil {
		return nil, model.NewAppError("SqlPostViewsHistoryStore.AnalyticsPostViewsHistoryCounts", "store.sql_post_views_history.analytics_post_views_history_counts.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	for _, row := range rows {
		row.Value = row.Value * model.POST_COUNTER_MAX
	}

	return rows, nil
}
