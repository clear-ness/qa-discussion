package sqlstore

import (
	"net/http"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlInboxMessageStore struct {
	store.Store
}

func inboxMessageSliceColumns() []string {
	return []string{"Id", "Type", "Content", "UserId", "SenderId", "QuestionId", "Title", "AnswerId", "CommentId", "TeamId", "CreateAt"}
}

func inboxMessageToSlice(inboxMessage *model.InboxMessage) []interface{} {
	return []interface{}{
		inboxMessage.Id,
		inboxMessage.Type,
		inboxMessage.Content,
		inboxMessage.UserId,
		inboxMessage.SenderId,
		inboxMessage.QuestionId,
		inboxMessage.Title,
		inboxMessage.AnswerId,
		inboxMessage.CommentId,
		inboxMessage.TeamId,
		inboxMessage.CreateAt,
	}
}

func NewSqlInboxMessageStore(sqlStore store.Store) store.InboxMessageStore {
	s := &SqlInboxMessageStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.InboxMessage{}, "InboxMessages").SetKeys(false, "Id")
	}

	return s
}

func (s SqlInboxMessageStore) GetSingle(id string) (*model.InboxMessage, *model.AppError) {
	var message *model.InboxMessage
	err := s.GetReplica().SelectOne(&message, "SELECT * FROM InboxMessages WHERE Id = :Id", map[string]interface{}{"Id": id})

	if err != nil {
		return nil, model.NewAppError("SqlInboxMessageStore.GetSingle", "store.sql_inbox_message.get_single.app_error", nil, "id="+id+err.Error(), http.StatusNotFound)
	}
	return message, nil
}

func (s SqlInboxMessageStore) GetInboxMessages(time int64, userId string, direction string, page, perPage int, teamId string) ([]*model.InboxMessage, *model.AppError) {
	offset := page * perPage

	if direction != ">" && direction != "<=" {
		return nil, model.NewAppError("SqlInboxMessageStore.GetInboxMessages", "store.sql_inbox_message.get_inbox_messages.app_error", nil, "", http.StatusInternalServerError)
	}

	query := s.GetQueryBuilder().Select("i.*")
	query = query.From("InboxMessages i").
		Where(sq.And{
			sq.Expr(`UserId = ?`, userId),
			sq.Expr(`CreateAt `+direction+` ?`, time),
		})

	// TODO: 問題無い？
	query = query.Where(sq.And{
		sq.Expr(`TeamId = ?`, teamId),
	})

	query = query.OrderBy("CreateAt DESC").
		Limit(uint64(perPage)).
		Offset(uint64(offset))

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlInboxMessageStore.GetInboxMessages", "store.sql_inbox_message.get_inbox_messages.app_error", nil, "", http.StatusInternalServerError)
	}

	var messages []*model.InboxMessage
	_, err = s.GetReplica().Select(&messages, queryString, args...)
	if err != nil {
		return nil, model.NewAppError("SqlInboxMessageStore.GetInboxMessages", "store.sql_inbox_message.get_inbox_messages.get.app_error", nil, "", http.StatusInternalServerError)
	}

	lastMessageViewed, err := s.GetReplica().SelectInt(`
		SELECT
			LastInboxMessageViewed
		FROM
			Users
		WHERE
			Users.Id = :UserId`, map[string]interface{}{"UserId": userId})
	if err != nil {
		return nil, model.NewAppError("SqlInboxMessageStore.GetInboxMessages", "store.sql_inbox_message.get_inbox_messages.get_last_message_viewed.app_error", nil, "", http.StatusInternalServerError)
	}

	for _, message := range messages {
		if message.CreateAt > lastMessageViewed {
			message.IsUnread = true
		}
	}

	return messages, nil
}

func (s SqlInboxMessageStore) GetInboxMessagesUnreadCount(userId string, fromDate int64, teamId string) (int64, *model.AppError) {
	createAtPart := "(SELECT LastInboxMessageViewed FROM Users WHERE Id = :UserId)"

	if fromDate != 0 {
		createAtPart = strconv.FormatInt(fromDate, 10)
	}

	// TODO: 問題無い？
	count, err := s.GetMaster().SelectInt(`
		SELECT
			count(*)
		FROM
			InboxMessages
		WHERE
			UserId = :UserId
			AND TeamId = :TeamId
			AND CreateAt > `+createAtPart,
		map[string]interface{}{"UserId": userId, "TeamId": teamId})

	if err != nil {
		return 0, model.NewAppError("SqlInboxMessageStore.GetInboxMessagesUnreadCount", "store.sql_inbox_message.get_inbox_messages_unread_count.get_unread_count.app_error", nil, "", http.StatusInternalServerError)
	}

	return count, nil
}

func (s *SqlInboxMessageStore) SaveInboxMessage(inboxMessage *model.InboxMessage) (*model.InboxMessage, *model.AppError) {
	if len(inboxMessage.Id) > 0 {
		return nil, model.NewAppError("SqlInboxMessageStore.SaveInboxMessage", "store.sql_inbox_message.save_inbox_message.existing.app_error", nil, "id="+inboxMessage.Id, http.StatusBadRequest)
	}

	inboxMessage.PreSave()
	if err := inboxMessage.IsValid(); err != nil {
		return nil, err
	}

	if err := s.GetMaster().Insert(inboxMessage); err != nil {
		return nil, model.NewAppError("SqlInboxMessageStore.SaveInboxMessage", "store.sql_inbox_message.save_inbox_message.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return inboxMessage, nil
}

func (s *SqlInboxMessageStore) SaveMultipleInboxMessages(inboxMessages []*model.InboxMessage) ([]*model.InboxMessage, *model.AppError) {
	query := s.GetQueryBuilder().Insert("InboxMessages").Columns(inboxMessageSliceColumns()...)

	for _, message := range inboxMessages {
		if len(message.Id) > 0 {
			return nil, model.NewAppError("SqlInboxMessageStore.SaveMultipleInboxMessages", "store.sql_inbox_message.save_multiple_inbox_messages.existing.app_error", nil, "id="+message.Id, http.StatusInternalServerError)
		}

		message.PreSave()

		if err := message.IsValid(); err != nil {
			return nil, model.NewAppError("SqlInboxMessageStore.SaveMultipleInboxMessages", "store.sql_inbox_message.save_multiple_inbox_messages.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		query = query.Values(inboxMessageToSlice(message)...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlInboxMessageStore.SaveMultipleInboxMessages", "store.sql_inbox_message.save_multiple_inbox_messages.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := s.GetMaster().Exec(sql, args...); err != nil {
		return nil, model.NewAppError("SqlInboxMessageStore.SaveMultipleInboxMessages", "store.sql_inbox_message.save_multiple_inbox_messages.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return inboxMessages, nil
}
