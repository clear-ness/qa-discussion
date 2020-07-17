package sqlstore

import (
	"database/sql"
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlNotificationSettingStore struct {
	store.Store
}

func NewSqlNotificationSettingStore(sqlStore store.Store) store.NotificationSettingStore {
	s := &SqlNotificationSettingStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.NotificationSetting{}, "NotificationSettings").SetKeys(false, "Id")
	}

	return s
}

func (s *SqlNotificationSettingStore) Get(userId string) (*model.NotificationSetting, *model.AppError) {
	var setting *model.NotificationSetting

	if err := s.GetReplica().SelectOne(&setting, "SELECT * FROM NotificationSettings WHERE UserId = :UserId", map[string]interface{}{"UserId": userId}); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlNotificationSettingStore.get", "store.sql_notification_setting.get.select.app_error", nil, err.Error(), http.StatusNotFound)
		}

		return nil, model.NewAppError("SqlNotificationSettingStore.get", "store.sql_notification_setting.get.select.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return setting, nil
}

func (s *SqlNotificationSettingStore) Save(userId, inboxInterval string) *model.AppError {
	var setting *model.NotificationSetting

	if err := s.GetReplica().SelectOne(&setting, "SELECT * FROM NotificationSettings WHERE UserId = :UserId", map[string]interface{}{"UserId": userId}); err != nil {
		if err == sql.ErrNoRows {
			setting := &model.NotificationSetting{
				UserId:        userId,
				InboxInterval: inboxInterval,
			}

			setting.PreSave()
			if err := setting.IsValid(); err != nil {
				return err
			}

			if err := s.GetMaster().Insert(setting); err != nil {
				return model.NewAppError("SqlNotificationSettingStore.save", "store.sql_notification_setting.save.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
			}

			return nil
		}

		return model.NewAppError("SqlNotificationSettingStore.save", "store.sql_notification_setting.save.select.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if setting.InboxInterval == inboxInterval {
		return model.NewAppError("SqlNotificationSettingStore.save", "store.sql_notification_setting.save.same_value.app_error", nil, "", http.StatusBadRequest)
	}

	setting.InboxInterval = inboxInterval
	if err := setting.IsValid(); err != nil {
		return err
	}

	setting.PreUpdate()
	if _, err := s.GetMaster().Update(setting); err != nil {
		return model.NewAppError("SqlNotificationSettingStore.save", "store.sql_notification_setting.save.update.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
