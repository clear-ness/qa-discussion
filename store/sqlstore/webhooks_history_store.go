package sqlstore

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlWebhooksHistoryStore struct {
	store.Store
}

func NewSqlWebhooksHistoryStore(sqlStore store.Store) store.WebhooksHistoryStore {
	s := &SqlWebhooksHistoryStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.WebhooksHistory{}, "WebhooksHistory").SetKeys(false, "Id")
	}

	return s
}

func (s SqlWebhooksHistoryStore) LogWebhookEvent(history *model.WebhooksHistory) error {
	if err := s.GetMaster().Insert(history); err != nil {
		return errors.Wrapf(err, "LogWebhookEvent Id=%s", history.Id)
	}

	return nil
}

func (s SqlWebhooksHistoryStore) GetWebhooksHistoriesPage(teamId string, offset, limit int) ([]*model.WebhooksHistory, *model.AppError) {
	var histories []*model.WebhooksHistory
	_, err := s.GetReplica().Select(histories, `
		SELECT
			WebhooksHistory.*
		FROM
			WebhooksHistory
		WHERE
			WebhooksHistory.TeamId = :TeamId
		LIMIT :Limit
		OFFSET :Offset`, map[string]interface{}{"TeamId": teamId, "Limit": limit, "Offset": offset})
	if err != nil {
		return nil, model.NewAppError("SqlWebhooksHistoryStore.GetWebhooksHistoriesPage", "store.sql_webhooks_history.get_page.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return histories, nil
}
