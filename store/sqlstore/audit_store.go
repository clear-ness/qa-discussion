package sqlstore

import (
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlAuditStore struct {
	store.Store
}

func NewSqlAuditStore(sqlStore store.Store) store.AuditStore {
	s := &SqlAuditStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Audit{}, "Audits").SetKeys(false, "Id")
	}

	return s
}

func (s SqlAuditStore) Save(audit *model.Audit) *model.AppError {
	audit.Id = model.NewId()
	audit.CreateAt = model.GetMillis()

	if err := s.GetMaster().Insert(audit); err != nil {
		return model.NewAppError("SqlAuditStore.Save", "store.sql_audit.save.saving.app_error", nil, "user_id="+audit.UserId+" action="+audit.Action, http.StatusInternalServerError)
	}
	return nil
}

func (s SqlAuditStore) Get(user_id string, offset int, limit int) (model.Audits, *model.AppError) {
	if limit > 1000 {
		return nil, model.NewAppError("SqlAuditStore.Get", "store.sql_audit.get.limit.app_error", nil, "user_id="+user_id, http.StatusBadRequest)
	}

	query := s.GetQueryBuilder().
		Select("*").
		From("Audits").
		OrderBy("CreateAt DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	if len(user_id) != 0 {
		query = query.Where(sq.Eq{"UserId": user_id})
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlAuditStore.Get", "store.sql_audit.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	var audits model.Audits
	if _, err := s.GetReplica().Select(&audits, queryString, args...); err != nil {
		return nil, model.NewAppError("SqlAuditStore.Get", "store.sql_audit.get.finding.app_error", nil, "user_id="+user_id, http.StatusInternalServerError)
	}

	return audits, nil
}

func (s SqlAuditStore) PermanentDeleteByUser(userId string) *model.AppError {
	if _, err := s.GetMaster().Exec("DELETE FROM Audits WHERE UserId = :userId",
		map[string]interface{}{"userId": userId}); err != nil {
		return model.NewAppError("SqlAuditStore.Delete", "store.sql_audit.permanent_delete_by_user.app_error", nil, "user_id="+userId, http.StatusInternalServerError)
	}
	return nil
}
