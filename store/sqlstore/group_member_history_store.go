package sqlstore

import (
	"github.com/pkg/errors"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlGroupMemberHistoryStore struct {
	store.Store
}

func NewSqlGroupMemberHistoryStore(sqlStore store.Store) store.GroupMemberHistoryStore {
	s := &SqlGroupMemberHistoryStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.GroupMemberHistory{}, "GroupMemberHistory").SetKeys(false, "GroupId", "UserId", "JoinTime")
	}

	return s
}

func (s SqlGroupMemberHistoryStore) LogJoinEvent(userId string, groupId string, joinTime int64) error {
	groupMemberHistory := &model.GroupMemberHistory{
		GroupId:  groupId,
		UserId:   userId,
		JoinTime: joinTime,
	}

	if err := s.GetMaster().Insert(groupMemberHistory); err != nil {
		return errors.Wrapf(err, "LogJoinEvent userId=%s groupId=%s joinTime=%d", userId, groupId, joinTime)
	}

	return nil
}

func (s SqlGroupMemberHistoryStore) LogLeaveEvent(userId string, groupId string, leaveTime int64) error {
	query := `
		UPDATE GroupMemberHistory
		SET LeaveTime = :LeaveTime
		WHERE UserId = :UserId
		AND GroupId = :GroupId
		AND LeaveTime IS NULL`

	params := map[string]interface{}{"UserId": userId, "GroupId": groupId, "LeaveTime": leaveTime}
	sqlResult, err := s.GetMaster().Exec(query, params)
	if err != nil {
		return errors.Wrapf(err, "LogLeaveEvent userId=%s groupId=%s leaveTime=%d", userId, groupId, leaveTime)
	}

	if rows, err := sqlResult.RowsAffected(); err == nil && rows != 1 {
		mlog.Warn("Group join event for user and group not found", mlog.String("user", userId), mlog.String("group", groupId))
	}

	return nil
}
