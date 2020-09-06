package sqlstore

import (
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlWebhookStore struct {
	store.Store
}

func NewSqlWebhookStore(sqlStore store.Store) store.WebhookStore {
	s := &SqlWebhookStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Webhook{}, "Webhooks").SetKeys(false, "Id")
	}

	return s
}

func (s SqlWebhookStore) GetByTeam(teamId string, userId string, offset, limit int) ([]*model.Webhook, *model.AppError) {
	var webhooks []*model.Webhook

	query := s.GetQueryBuilder().
		Select("*").
		From("Webhooks").
		Where(sq.And{
			sq.Eq{"TeamId": teamId},
			sq.Eq{"DeleteAt": int(0)},
		})

	if len(userId) > 0 {
		query = query.Where(sq.Eq{"UserId": userId})
	}
	if limit >= 0 && offset >= 0 {
		query = query.Limit(uint64(limit)).Offset(uint64(offset))
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlWebhookStore.GetByTeam", "store.sql_webhooks.get_by_team.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := s.GetReplica().Select(&webhooks, queryString, args...); err != nil {
		return nil, model.NewAppError("SqlWebhookStore.GetByTeam", "store.sql_webhooks.get_by_team.app_error", nil, "teamId="+teamId+", err="+err.Error(), http.StatusInternalServerError)
	}

	return webhooks, nil
}

func (s SqlWebhookStore) Save(webhook *model.Webhook) (*model.Webhook, *model.AppError) {
	if len(webhook.Id) > 0 {
		return nil, model.NewAppError("SqlWebhookStore.Save", "store.sql_webhooks.save.override.app_error", nil, "id="+webhook.Id, http.StatusBadRequest)
	}

	webhook.PreSave()
	if err := webhook.IsValid(); err != nil {
		return nil, err
	}

	if err := s.GetMaster().Insert(webhook); err != nil {
		return nil, model.NewAppError("SqlWebhookStore.Save", "store.sql_webhooks.save.app_error", nil, "id="+webhook.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	return webhook, nil
}

func (s SqlWebhookStore) Get(id string) (*model.Webhook, *model.AppError) {
	var webhook model.Webhook
	if err := s.GetReplica().SelectOne(&webhook, "SELECT * FROM Webhooks WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": id}); err != nil {
		return nil, model.NewAppError("SqlWebhookStore.Get", "store.sql_webhooks.get.app_error", nil, "id="+id+", err="+err.Error(), http.StatusInternalServerError)
	}

	return &webhook, nil
}

func (s SqlWebhookStore) Update(hook *model.Webhook) (*model.Webhook, *model.AppError) {
	hook.UpdateAt = model.GetMillis()
	if _, err := s.GetMaster().Update(hook); err != nil {
		return nil, model.NewAppError("SqlWebhookStore.Update", "store.sql_webhooks.update.app_error", nil, "id="+hook.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	return hook, nil
}

func (s SqlWebhookStore) Delete(webhookId string, time int64) *model.AppError {
	_, err := s.GetMaster().Exec("Update Webhooks SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"DeleteAt": time, "UpdateAt": time, "Id": webhookId})
	if err != nil {
		return model.NewAppError("SqlWebhookStore.Delete", "store.sql_webhooks.delete.app_error", nil, "id="+webhookId+", err="+err.Error(), http.StatusInternalServerError)
	}

	return nil
}
