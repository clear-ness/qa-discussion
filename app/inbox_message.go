package app

import (
	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetSingleInboxMessage(messageId string) (*model.InboxMessage, *model.AppError) {
	return a.Srv.Store.InboxMessage().GetSingle(messageId)
}

func (a *App) GetInboxMessagesForUserToDate(toDate int64, userId string, page, perPage int, teamId string) ([]*model.InboxMessage, *model.AppError) {
	return a.Srv.Store.InboxMessage().GetInboxMessages(toDate, userId, "<=", page, perPage, teamId)
}

func (a *App) GetInboxMessagesUnreadCountForUser(userId string, teamId string) (int64, *model.AppError) {
	return a.Srv.Store.InboxMessage().GetInboxMessagesUnreadCount(userId, 0, teamId)
}
