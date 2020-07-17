package api

import (
	"testing"
	"time"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/stretchr/testify/require"
)

func TestNotificationSettingForUser(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	data, resp := Client.GetNotificationSettingForUser(model.ME)
	CheckNotFoundStatus(t, resp)

	_, resp = Client.UpdateNotificationSettingForUser(model.ME, "")
	CheckNoError(t, resp)
	CheckOKStatus(t, resp)

	data, resp = Client.GetNotificationSettingForUser(model.ME)
	require.Equal(t, th.BasicUser.Id, data.UserId, "notification setting didn't match")
	require.Equal(t, "", data.InboxInterval, "notification setting didn't match")
	updateAt1 := data.UpdateAt

	time.Sleep(time.Millisecond * 10)

	_, resp = Client.UpdateNotificationSettingForUser(model.ME, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	CheckNoError(t, resp)
	CheckOKStatus(t, resp)

	data, resp = Client.GetNotificationSettingForUser(model.ME)
	require.Equal(t, th.BasicUser.Id, data.UserId, "notification setting didn't match")
	require.Equal(t, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR, data.InboxInterval, "notification setting didn't match")
	updateAt2 := data.UpdateAt
	require.Less(t, updateAt1, updateAt2, "UpdateAt should be updated")

	// same error
	_, resp = Client.UpdateNotificationSettingForUser(model.ME, model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR)
	CheckBadRequestStatus(t, resp)
}
