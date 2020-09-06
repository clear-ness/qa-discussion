package cachelayer

import (
	"strconv"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type CacheNotificationSettingStore struct {
	store.NotificationSettingStore
	rootStore *CacheStore
}

func NotificationSettingHashFromObj(obj *model.NotificationSetting) map[string]interface{} {
	createAt := strconv.FormatInt(obj.CreateAt, 10)
	updateAt := strconv.FormatInt(obj.UpdateAt, 10)

	hash := make(map[string]interface{})
	hash["id"] = obj.Id
	hash["userid"] = obj.UserId
	hash["inboxInterval"] = obj.InboxInterval
	hash["createat"] = createAt
	hash["updateat"] = updateAt

	return hash
}

func NotificationSettingFromHash(hash map[string]string) *model.NotificationSetting {
	createAt := int64(0)
	if val, err := strconv.Atoi(hash["createat"]); err == nil {
		createAt = int64(val)
	}
	updateAt := int64(0)
	if val, err := strconv.Atoi(hash["updateat"]); err == nil {
		updateAt = int64(val)
	}

	return &model.NotificationSetting{
		Id:            hash["id"],
		UserId:        hash["userid"],
		InboxInterval: hash["inboxinterval"],
		CreateAt:      createAt,
		UpdateAt:      updateAt,
	}
}

func (s CacheNotificationSettingStore) Get(userId string) (*model.NotificationSetting, *model.AppError) {
	notificationKey := userId + "notification"

	if hash := s.rootStore.readHashCache(notificationKey); hash != nil {
		return NotificationSettingFromHash(hash), nil
	}

	setting, err := s.NotificationSettingStore.Get(userId)
	if err != nil {
		return nil, err
	}

	hash := NotificationSettingHashFromObj(setting)
	s.rootStore.addToHashCache(notificationKey, hash)

	return setting, nil
}

func (s CacheNotificationSettingStore) InvalidateNotificationSetting(userId string) {
	notificationKey := userId + "notification"

	s.rootStore.deleteCache([]string{notificationKey})
}

func (s CacheNotificationSettingStore) Save(userId, inboxInterval string) *model.AppError {
	err := s.NotificationSettingStore.Save(userId, inboxInterval)
	if err != nil {
		return err
	}

	s.InvalidateNotificationSetting(userId)

	return nil
}
