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
		db.AddTableWithName(model.Tag{}, "Tags").SetKeys(false, "Content")
	}

	return s
}

func (s *SqlTagStore) GetTags(options *model.GetTagsOptions) (model.Tags, *model.AppError) {
	offset := options.Page * options.PerPage
	var tags model.Tags

	query := s.GetQueryBuilder().Select("t.*")
	query = query.From("Tags t")

	if options.InName != "" {
		prefix := options.InName
		query = query.Where("Content LIKE ?", prefix+"%")
	}

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

	var orderBy = "CreateAt DESC"
	if options.SortType == model.POST_SORT_TYPE_NAME {
		orderBy = "Content ASC"
	} else if options.SortType == model.POST_SORT_TYPE_POPULAR {
		orderBy = "PostCount DESC"
	}

	query = query.OrderBy(orderBy).
		Limit(uint64(options.PerPage)).
		Offset(uint64(offset))

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlTagStore.GetTags", "store.sql_tag.get_tags.get.app_error", nil, "", http.StatusInternalServerError)
	}

	_, err = s.GetReplica().Select(&tags, queryString, args...)
	if err != nil {
		return nil, model.NewAppError("SqlTagStore.GetTags", "store.sql_tag.get_tags.select.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return tags, nil
}
