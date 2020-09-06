package sqlstore

import (
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlTagStore struct {
	store.Store
}

func NewSqlTagStore(sqlStore store.Store) store.TagStore {
	s := &SqlTagStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Tag{}, "Tags").SetKeys(false, "Content", "TeamId")
	}

	return s
}

func (s *SqlTagStore) getTags(options *model.GetTagsOptions, countQuery bool) sq.SelectBuilder {
	offset := options.Page * options.PerPage

	var selectStr string
	if countQuery {
		selectStr = "count(*)"
	} else {
		selectStr = "t.*"
	}

	query := s.GetQueryBuilder().Select(selectStr)
	query = query.From("Tags t")

	if options.Content != "" {
		query = query.Where(sq.And{
			sq.Expr(`Content = ?`, options.Content),
		})
	} else if options.InName != "" {
		prefix := options.InName
		query = query.Where("Content LIKE ?", prefix+"%")
	}

	// TODO: teamId, typeが""の場合は問題無い？
	query = query.Where(sq.And{
		sq.Expr(`TeamId = ?`, options.TeamId),
	})

	query = query.Where(sq.And{
		sq.Expr(`Type = ?`, options.Type),
	})

	if options.SortType == model.POST_SORT_TYPE_POPULAR && options.Min != nil {
		query = query.Where(sq.And{
			sq.Expr(`PostCount >= ?`, *options.Min),
		})
	}

	if options.SortType == model.POST_SORT_TYPE_POPULAR && options.Max != nil {
		query = query.Where(sq.And{
			sq.Expr(`PostCount <= ?`, *options.Max),
		})
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

	if !countQuery {
		var orderBy = "CreateAt DESC"
		if options.SortType == model.POST_SORT_TYPE_NAME {
			orderBy = "Content ASC"
		} else if options.SortType == model.POST_SORT_TYPE_POPULAR {
			orderBy = "PostCount DESC"
		}
		query = query.OrderBy(orderBy).
			Limit(uint64(options.PerPage)).
			Offset(uint64(offset))
	}

	return query
}

func (s *SqlTagStore) GetTags(options *model.GetTagsOptions) (model.Tags, *model.AppError) {
	queryString, args, err := s.getTags(options, false).ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlTagStore.GetTags", "store.sql_tag.get_tags.get.app_error", nil, "", http.StatusInternalServerError)
	}

	var tags model.Tags
	_, err = s.GetReplica().Select(&tags, queryString, args...)
	if err != nil {
		return nil, model.NewAppError("SqlTagStore.GetTags", "store.sql_tag.get_tags.get.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return tags, nil
}

func (s *SqlTagStore) GetTagsCount(options *model.GetTagsOptions) (int64, *model.AppError) {
	queryString, args, err := s.getTags(options, true).ToSql()
	if err != nil {
		return int64(0), model.NewAppError("SqlTagStore.GetTagsCount", "store.sql_tag.get_tags_count.get.app_error", nil, "", http.StatusInternalServerError)
	}

	count := int64(0)
	if count, err = s.GetReplica().SelectInt(queryString, args...); err != nil {
		return int64(0), model.NewAppError("SqlTagStore.GetTagsCount", "store.sql_tag.get_tags_count.get.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return count, nil
}

func (s *SqlTagStore) CreateTags(addedTags []string, time int64, teamId string, tagType string) *model.AppError {
	if len(addedTags) <= 0 {
		return model.NewAppError("SqlTagStore.CreateTags", "store.sql_tag.create_tags.no_tags.app_error", nil, "", http.StatusInternalServerError)
	}

	sql, args, err := s.buildInsertTagsQuery(addedTags, time, teamId, tagType, false)
	if err != nil {
		return model.NewAppError("SqlTagStore.CreateTags", "store.sql_tag.create_tags.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := s.GetMaster().Exec(sql, args...); err != nil {
		return model.NewAppError("SqlTagStore.CreateTags", "store.sql_tag.create_tags.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlTagStore) buildInsertTagsQuery(addedTags []string, time int64, teamId string, tagType string, updateOnDuplicate bool) (string, []interface{}, error) {
	query := s.GetQueryBuilder().Insert("Tags").Columns(tagSliceColumns()...)

	for _, tagContent := range addedTags {
		tag := &model.Tag{
			Content:   tagContent,
			TeamId:    teamId,
			Type:      tagType,
			PostCount: 1,
			CreateAt:  time,
			UpdateAt:  time,
		}

		tag.PreSave()
		if err := tag.IsValid(); err != nil {
			return "", nil, err
		}

		query = query.Values(tagToSlice(tag)...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return "", args, model.NewAppError("SqlTagStore.buildInsertTagsQuery", "store.sql_tag.inset_tags.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if updateOnDuplicate {
		return sql + " ON DUPLICATE KEY UPDATE PostCount = VALUES(PostCount) + 1, UpdateAt = VALUES(UpdateAt)", args, nil
	} else {
		return sql, args, nil
	}
}
