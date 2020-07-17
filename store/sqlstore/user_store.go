package sqlstore

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlUserStore struct {
	store.Store

	usersQuery sq.SelectBuilder
}

func NewSqlUserStore(sqlStore store.Store) store.UserStore {
	us := &SqlUserStore{
		Store: sqlStore,
	}

	us.usersQuery = us.GetQueryBuilder().Select("Users.*").From("Users")

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.User{}, "Users").SetKeys(false, "Id")
	}

	return us
}

func (us SqlUserStore) Save(user *model.User) (*model.User, *model.AppError) {
	if len(user.Id) > 0 {
		return nil, model.NewAppError("SqlUserStore.Save", "store.sql_user.save.existing.app_error", nil, "user_id="+user.Id, http.StatusBadRequest)
	}

	user.PreSave()
	if err := user.IsValid(); err != nil {
		return nil, err
	}

	if err := us.GetMaster().Insert(user); err != nil {
		if IsUniqueConstraintError(err, []string{"Email", "users_email_key", "idx_users_email_unique"}) {
			return nil, model.NewAppError("SqlUserStore.Save", "store.sql_user.save.email_exists.app_error", nil, "user_id="+user.Id+", "+err.Error(), http.StatusBadRequest)
		}
		return nil, model.NewAppError("SqlUserStore.Save", "store.sql_user.save.app_error", nil, "user_id="+user.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	return user, nil
}

func (us SqlUserStore) Update(user *model.User, trustedUpdateData bool) (*model.UserUpdate, *model.AppError) {
	user.PreUpdate()

	if err := user.IsValid(); err != nil {
		return nil, err
	}

	oldUserResult, err := us.GetMaster().Get(model.User{}, user.Id)
	if err != nil {
		return nil, model.NewAppError("SqlUserStore.Update", "store.sql_user.update.finding.app_error", nil, "user_id="+user.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	if oldUserResult == nil {
		return nil, model.NewAppError("SqlUserStore.Update", "store.sql_user.update.find.app_error", nil, "user_id="+user.Id, http.StatusBadRequest)
	}

	oldUser := oldUserResult.(*model.User)
	user.CreateAt = oldUser.CreateAt
	user.SuspendTime = oldUser.SuspendTime
	user.Password = oldUser.Password
	user.Props = oldUser.Props
	user.EmailVerified = oldUser.EmailVerified
	user.FailedAttempts = oldUser.FailedAttempts
	user.Points = oldUser.Points
	user.LastInboxMessageViewed = oldUser.LastInboxMessageViewed

	if !trustedUpdateData {
		user.Type = oldUser.Type
		user.DeleteAt = oldUser.DeleteAt
	}

	if user.Email != oldUser.Email {
		user.EmailVerified = false
	}

	count, err := us.GetMaster().Update(user)
	if err != nil {
		if IsUniqueConstraintError(err, []string{"Email", "users_email_key", "idx_users_email_unique"}) {
			return nil, model.NewAppError("SqlUserStore.Update", "store.sql_user.update.email_taken.app_error", nil, "user_id="+user.Id+", "+err.Error(), http.StatusBadRequest)
		}
		return nil, model.NewAppError("SqlUserStore.Update", "store.sql_user.update.updating.app_error", nil, "user_id="+user.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	if count != 1 {
		return nil, model.NewAppError("SqlUserStore.Update", "store.sql_user.update.app_error", nil, fmt.Sprintf("user_id=%v, count=%v", user.Id, count), http.StatusInternalServerError)
	}

	user.Sanitize(map[string]bool{})
	oldUser.Sanitize(map[string]bool{})
	return &model.UserUpdate{New: user, Old: oldUser}, nil
}

func (us SqlUserStore) Get(id string) (*model.User, *model.AppError) {
	query := us.usersQuery.Where("Id = ?", id).Where("DeleteAt = ?", 0)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlUserStore.Get", "store.sql_user.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	user := &model.User{}
	if err := us.GetReplica().SelectOne(user, queryString, args...); err == sql.ErrNoRows {
		return nil, model.NewAppError("SqlUserStore.Get", store.MISSING_ACCOUNT_ERROR, nil, "user_id="+id, http.StatusNotFound)
	} else if err != nil {
		return nil, model.NewAppError("SqlUserStore.Get", "store.sql_user.get.app_error", nil, "user_id="+id+", "+err.Error(), http.StatusInternalServerError)
	}

	return user, nil
}

func (us SqlUserStore) GetByIds(userIds []string) ([]*model.User, *model.AppError) {
	query := us.usersQuery.
		Where(map[string]interface{}{
			"Id":       userIds,
			"DeleteAt": 0,
		})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlUserStore.Get", "store.sql_user.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	var users []*model.User
	if _, err := us.GetReplica().Select(&users, queryString, args...); err != nil {
		return nil, model.NewAppError("SqlUserStore.Get", "store.sql_user.get.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return users, nil
}

func (us SqlUserStore) GetByEmail(email string) (*model.User, *model.AppError) {
	email = strings.ToLower(email)

	query := us.usersQuery.Where("Email = ?", email)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlUserStore.GetByEmail", "store.sql_user.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	user := model.User{}
	if err := us.GetReplica().SelectOne(&user, queryString, args...); err != nil {
		return nil, model.NewAppError("SqlUserStore.GetByEmail", store.MISSING_ACCOUNT_ERROR, nil, "email="+email+", "+err.Error(), http.StatusInternalServerError)
	}

	return &user, nil
}

func (us SqlUserStore) GetUsersByDates(options *model.GetUsersOptions) ([]*model.User, *model.AppError) {
	orderBy := "CreateAt DESC"
	if options.SortType == "votes" {
		orderBy = "Points DESC"
	}

	offset := options.Page * options.PerPage

	query := us.usersQuery.Where("DeleteAt = ?", 0).
		OrderBy(orderBy).
		Limit(uint64(options.PerPage)).
		Offset(uint64(offset))

	if options.FromDate != 0 {
		query = query.Where("CreateAt >= ?", options.FromDate)
	}

	if options.ToDate != 0 {
		query = query.Where("CreateAt <= ?", options.ToDate)
	}

	if options.SortType == "votes" && options.Min != nil {
		query = query.Where("Points >= ?", *options.Min)
	}

	if options.SortType == "votes" && options.Max != nil {
		query = query.Where("Points <= ?", *options.Max)
	}

	prefix := options.Username
	if len(prefix) > 0 {
		query = query.Where("Username LIKE ?", prefix+"%")
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlUserStore.GetUsersByDates", "store.sql_user.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	var users []*model.User
	if _, err := us.GetReplica().Select(&users, queryString, args...); err != nil {
		return nil, model.NewAppError("SqlUserStore.GetUsersByDates", "store.sql_user.get.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return users, nil
}

func (us SqlUserStore) GetForLogin(loginId string) (*model.User, *model.AppError) {
	query := us.usersQuery
	query = query.Where("Email = ?", loginId)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlUserStore.GetForLogin", "store.sql_user.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	users := []*model.User{}
	if _, err := us.GetReplica().Select(&users, queryString, args...); err != nil {
		return nil, model.NewAppError("SqlUserStore.GetForLogin", "store.sql_user.get_for_login.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if len(users) == 0 {
		return nil, model.NewAppError("SqlUserStore.GetForLogin", "store.sql_user.get_for_login.app_error", nil, "", http.StatusInternalServerError)
	}
	if len(users) > 1 {
		return nil, model.NewAppError("SqlUserStore.GetForLogin", "store.sql_user.get_for_login.multiple_users", nil, "", http.StatusInternalServerError)
	}

	return users[0], nil
}

func (us SqlUserStore) VerifyEmail(userId, email string) (string, *model.AppError) {
	curTime := model.GetMillis()
	if _, err := us.GetMaster().Exec("UPDATE Users SET Email = :email, EmailVerified = true, UpdateAt = :Time WHERE Id = :UserId", map[string]interface{}{"email": email, "Time": curTime, "UserId": userId}); err != nil {
		return "", model.NewAppError("SqlUserStore.VerifyEmail", "store.sql_user.verify_email.app_error", nil, "userId="+userId+", "+err.Error(), http.StatusInternalServerError)
	}

	return userId, nil
}

func (us SqlUserStore) GetByInboxInterval(fromUserId string, inboxInterval string, limit int) ([]*model.User, *model.AppError) {
	var settings []*model.NotificationSetting
	if _, err := us.GetReplica().Select(&settings, `
		SELECT *
		FROM NotificationSettings
		WHERE InboxInterval = :InboxInterval
		AND UserId > :FromUserId
		ORDER BY UserId
		LIMIT :Limit
		`, map[string]interface{}{
		"InboxInterval": inboxInterval,
		"FromUserId":    fromUserId,
		"Limit":         limit,
	}); err != nil {
		return nil, model.NewAppError("SqlUserStore.GetByInboxInterval", "store.sql_user.get_by_inbox_interval.get.app_error", nil, "fromUserId="+fromUserId+", err="+err.Error(), http.StatusInternalServerError)
	}

	var userIds []string
	for _, setting := range settings {
		userIds = append(userIds, setting.UserId)
	}

	var users []*model.User
	users, err := us.GetByIds(userIds)
	if err != nil {
		return nil, model.NewAppError("SqlUserStore.GetByInboxInterval", "store.sql_user.get_by_inbox_interval.get.app_error", nil, "fromUserId="+fromUserId+", err="+err.Error(), http.StatusInternalServerError)
	}

	return users, nil
}

func (us SqlUserStore) UpdateLastInboxMessageViewed(message *model.InboxMessage, userId string) *model.AppError {
	lastMessageViewed, err := us.GetReplica().SelectInt(`
		SELECT
			LastInboxMessageViewed
		FROM
			Users
		WHERE
			Users.Id = :UserId`, map[string]interface{}{"UserId": userId})
	if err != nil {
		return model.NewAppError("SqlUserStore.UpdateLastInboxMessageViewed", "store.sql_user.update_last_inbox_message_viewed.get_last_message_viewed.app_error", nil, "userId="+userId+", "+err.Error(), http.StatusInternalServerError)
	}

	if lastMessageViewed >= message.CreateAt {
		return model.NewAppError("SqlUserStore.UpdateLastInboxMessageViewed", "store.sql_user.update_last_inbox_message_viewed.already_read_message.app_error", nil, "userId="+userId, http.StatusBadRequest)
	}

	curTime := model.GetMillis()
	if _, err := us.GetMaster().Exec("UPDATE Users SET LastInboxMessageViewed = :LastMessageViewed, UpdateAt= :Time WHERE Id = :UserId", map[string]interface{}{"LastMessageViewed": message.CreateAt, "Time": curTime, "UserId": userId}); err != nil {
		return model.NewAppError("SqlUserStore.UpdateLastInboxMessageViewed", "store.sql_user.update_last_inbox_message_viewed.app_error", nil, "userId="+userId+", "+err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (us SqlUserStore) SuspendUser(userId string, suspendSpan string, moderatorId string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlUserStore.SuspendUser", "store.sql_user.suspend_user.app_error", nil, "id="+userId+", err="+errMsg, http.StatusInternalServerError)
	}

	var user *model.User
	if err := us.GetReplica().SelectOne(&user, "SELECT * FROM Users WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": userId}); err != nil {
		return appErr(err.Error())
	}

	user.AddProp(model.USER_PROPS_SUSPEND_BY, moderatorId)

	curTime := model.GetMillis()

	suspendTime := model.GetSuspendTimeBySpan(suspendSpan, curTime)

	if _, err := us.GetMaster().Exec("UPDATE Users SET SuspendTime = :SuspendTime, UpdateAt= :UpdateAt, Props = :Props WHERE Id = :UserId", map[string]interface{}{"SuspendTime": suspendTime, "UpdateAt": curTime, "Props": model.MapToJson(user.Props), "UserId": userId}); err != nil {
		return model.NewAppError("SqlUserStore.SuspendUser", "store.sql_user.suspend_user.app_error", nil, "userId="+userId+", "+err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (us SqlUserStore) Delete(userId string, time int64, deleteById string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlUserStore.DeleteUser", "store.sql_user.delete_user.app_error", nil, "id="+userId+", err="+errMsg, http.StatusInternalServerError)
	}

	var user *model.User
	err := us.GetReplica().SelectOne(&user, "SELECT * FROM Users WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": userId})
	if err != nil {
		return appErr(err.Error())
	}

	user.AddProp(model.USER_PROPS_DELETE_BY, deleteById)

	if _, err := us.GetMaster().Exec("UPDATE Users SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"DeleteAt": time, "UpdateAt": time, "Id": userId, "Props": model.MapToJson(user.Props)}); err != nil {
		return model.NewAppError("SqlUserStore.DeleteUser", "store.sql_user.delete_user.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (us SqlUserStore) UpdatePassword(userId, hashedPassword string) *model.AppError {
	updateAt := model.GetMillis()

	if _, err := us.GetMaster().Exec("UPDATE Users SET Password = :Password, UpdateAt = :UpdateAt, FailedAttempts = 0 WHERE Id = :UserId", map[string]interface{}{"Password": hashedPassword, "UpdateAt": updateAt, "UserId": userId}); err != nil {
		return model.NewAppError("SqlUserStore.UpdatePassword", "store.sql_user.update_password.app_error", nil, "id="+userId+", "+err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (us SqlUserStore) UpdateFailedPasswordAttempts(userId string, attempts int) *model.AppError {
	if _, err := us.GetMaster().Exec("UPDATE Users SET FailedAttempts = :FailedAttempts WHERE Id = :UserId", map[string]interface{}{"FailedAttempts": attempts, "UserId": userId}); err != nil {
		return model.NewAppError("SqlUserStore.UpdateFailedPasswordAttempts", "store.sql_user.update_failed_password_attempts.app_error", nil, "user_id="+userId, http.StatusInternalServerError)
	}

	return nil
}

func (us SqlUserStore) Count(options *model.UserCountOptions) (int64, *model.AppError) {
	query := us.GetQueryBuilder().Select("COUNT(DISTINCT u.Id)").From("Users AS u")

	if !options.IncludeDeleted {
		query = query.Where("u.DeleteAt = 0")
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return int64(0), model.NewAppError("SqlUserStore.Get", "store.sql_user.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	count, err := us.GetReplica().SelectInt(queryString, args...)
	if err != nil {
		return int64(0), model.NewAppError("SqlUserStore.Count", "store.sql_user.get_total_users_count.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return count, nil
}

func (us SqlUserStore) UpdateLastPictureUpdate(userId string, time int64) *model.AppError {
	if _, err := us.GetMaster().Exec("UPDATE Users SET LastPictureUpdate = :Time, UpdateAt = :Time WHERE Id = :UserId", map[string]interface{}{"Time": time, "UserId": userId}); err != nil {
		return model.NewAppError("SqlUserStore.UpdateLastPictureUpdate", "store.sql_user.update_last_picture_update.app_error", nil, "user_id="+userId, http.StatusInternalServerError)
	}

	return nil
}
