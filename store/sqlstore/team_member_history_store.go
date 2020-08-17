package sqlstore

import (
	"github.com/pkg/errors"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlTeamMemberHistoryStore struct {
	store.Store
}

func NewSqlTeamMemberHistoryStore(sqlStore store.Store) store.TeamMemberHistoryStore {
	s := &SqlTeamMemberHistoryStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.TeamMemberHistory{}, "TeamMemberHistory").SetKeys(false, "TeamId", "UserId", "JoinTime")
	}

	return s
}

func (s SqlTeamMemberHistoryStore) LogJoinEvent(userId string, teamId string, joinTime int64) error {
	teamMemberHistory := &model.TeamMemberHistory{
		TeamId:   teamId,
		UserId:   userId,
		JoinTime: joinTime,
	}

	if err := s.GetMaster().Insert(teamMemberHistory); err != nil {
		return errors.Wrapf(err, "LogJoinEvent userId=%s teamId=%s joinTime=%d", userId, teamId, joinTime)
	}

	return nil
}

func (s SqlTeamMemberHistoryStore) LogLeaveEvent(userId string, teamId string, leaveTime int64) error {
	query := `
		UPDATE TeamMemberHistory
		SET LeaveTime = :LeaveTime
		WHERE UserId = :UserId
		AND TeamId = :TeamId
		AND LeaveTime IS NULL`

	params := map[string]interface{}{"UserId": userId, "TeamId": teamId, "LeaveTime": leaveTime}
	sqlResult, err := s.GetMaster().Exec(query, params)
	if err != nil {
		return errors.Wrapf(err, "LogLeaveEvent userId=%s teamId=%s leaveTime=%d", userId, teamId, leaveTime)
	}

	if rows, err := sqlResult.RowsAffected(); err == nil && rows != 1 {
		mlog.Warn("Team join event for user and team not found", mlog.String("user", userId), mlog.String("team", teamId))
	}

	return nil
}
