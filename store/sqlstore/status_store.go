package sqlstore

import (
	"database/sql"
	"net/http"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"

	"github.com/pkg/errors"
)

type SqlStatusStore struct {
	store.Store
}

func NewSqlStatusStore(sqlStore store.Store) store.StatusStore {
	s := &SqlStatusStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Status{}, "Status").SetKeys(false, "UserId")
	}

	return s
}

func (s SqlStatusStore) Get(userId string) (*model.Status, *model.AppError) {
	var status model.Status
	if err := s.GetReplica().SelectOne(&status,
		`SELECT
			*
		FROM
			Status
		WHERE
			UserId = :UserId`, map[string]interface{}{"UserId": userId}); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlStatusStore.Get", store.MISSING_STATUS_ERROR, nil, err.Error(), http.StatusNotFound)
		}

		return nil, model.NewAppError("SqlStatusStore.Get", "store.sql_status.get_post.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return &status, nil
}

func (s SqlStatusStore) GetByIds(userIds []string) ([]*model.Status, error) {
	query := s.GetQueryBuilder().
		Select("UserId, Status, Manual, LastActivityAt").
		From("Status").
		Where(sq.Eq{"UserId": userIds})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "status_tosql")
	}

	rows, err := s.GetReplica().Db.Query(queryString, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find Statuses")
	}

	var statuses []*model.Status
	defer rows.Close()
	for rows.Next() {
		var status model.Status
		if err = rows.Scan(&status.UserId, &status.Status, &status.Manual, &status.LastActivityAt); err != nil {
			return nil, errors.Wrap(err, "unable to scan from rows")
		}

		statuses = append(statuses, &status)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed while iterating over rows")
	}

	return statuses, nil
}

func (s SqlStatusStore) SaveOrUpdate(status *model.Status) error {
	if err := s.GetReplica().SelectOne(&model.Status{}, "SELECT * FROM Status WHERE UserId = :UserId", map[string]interface{}{"UserId": status.UserId}); err == nil {
		if _, err := s.GetMaster().Update(status); err != nil {
			return errors.Wrap(err, "failed to update Status")
		}
	} else {
		if err := s.GetMaster().Insert(status); err != nil {
			if !(strings.Contains(err.Error(), "for key 'PRIMARY'") && strings.Contains(err.Error(), "Duplicate entry")) {
				return errors.Wrap(err, "failed in save Status")
			}
		}
	}

	return nil
}

func (s SqlStatusStore) UpdateLastActivityAt(userId string, lastActivityAt int64) error {
	if _, err := s.GetMaster().Exec("UPDATE Status SET LastActivityAt = :Time WHERE UserId = :UserId", map[string]interface{}{"UserId": userId, "Time": lastActivityAt}); err != nil {
		return errors.Wrapf(err, "failed to update last activity for userId=%s", userId)
	}

	return nil
}
