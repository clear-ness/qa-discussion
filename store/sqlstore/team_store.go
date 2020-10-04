package sqlstore

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/go-gorp/gorp"
)

const (
	TEAM_MEMBER_EXISTS_ERROR = "store.sql_team.save_member.exists.app_error"
)

type SqlTeamStore struct {
	store.Store

	membersQuery sq.SelectBuilder
}

func teamMemberSliceColumns() []string {
	return []string{"TeamId", "UserId", "Type", "Points", "DeleteAt"}
}

func teamMemberToSlice(member *model.TeamMember) []interface{} {
	resultSlice := []interface{}{}
	resultSlice = append(resultSlice, member.TeamId)
	resultSlice = append(resultSlice, member.UserId)
	resultSlice = append(resultSlice, member.Type)
	resultSlice = append(resultSlice, member.Points)
	resultSlice = append(resultSlice, member.DeleteAt)

	return resultSlice
}

func NewSqlTeamStore(sqlStore store.Store) store.TeamStore {
	s := &SqlTeamStore{
		Store: sqlStore,
	}

	s.membersQuery = s.GetQueryBuilder().Select("TeamMembers.*").From("TeamMembers")

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Team{}, "Teams").SetKeys(false, "Id")
		db.AddTableWithName(model.TeamMember{}, "TeamMembers").SetKeys(false, "TeamId", "UserId")
	}

	return s
}

func (s SqlTeamStore) Save(team *model.Team) (*model.Team, *model.AppError) {
	if len(team.Id) > 0 {
		return nil, model.NewAppError("SqlTeamStore.Save",
			"store.sql_team.save.existing.app_error", nil, "id="+team.Id, http.StatusBadRequest)
	}

	// これでteamのinvite idがセットされる
	team.PreSave()

	if err := team.IsValid(); err != nil {
		return nil, err
	}

	if err := s.GetMaster().Insert(team); err != nil {
		if IsUniqueConstraintError(err, []string{"Name", "teams_name_key"}) {
			return nil, model.NewAppError("SqlTeamStore.Save", "store.sql_team.save.domain_exists.app_error", nil, "id="+team.Id+", "+err.Error(), http.StatusBadRequest)
		}
		return nil, model.NewAppError("SqlTeamStore.Save", "store.sql_team.save.app_error", nil, "id="+team.Id+", "+err.Error(), http.StatusInternalServerError)
	}
	return team, nil
}

func (s SqlTeamStore) GetMember(teamId string, userId string) (*model.TeamMember, *model.AppError) {
	query := s.membersQuery.
		Where(sq.Eq{"TeamMembers.TeamId": teamId}).
		Where(sq.Eq{"TeamMembers.UserId": userId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetMember", "store.sql_team.get_member.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	member := &model.TeamMember{}
	err = s.GetReplica().SelectOne(&member, queryString, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlTeamStore.GetMember", "store.sql_team.get_member.missing.app_error", nil, "teamId="+teamId+" userId="+userId+" "+err.Error(), http.StatusNotFound)
		}
		return nil, model.NewAppError("SqlTeamStore.GetMember", "store.sql_team.get_member.app_error", nil, "teamId="+teamId+" userId="+userId+" "+err.Error(), http.StatusInternalServerError)
	}

	return member, nil
}

func (s SqlTeamStore) GetActiveMemberCount(teamId string) (int64, *model.AppError) {
	query := s.GetQueryBuilder().
		Select("count(DISTINCT TeamMembers.UserId)").
		From("TeamMembers, Users").
		Where("TeamMembers.DeleteAt = 0").
		Where("TeamMembers.UserId = Users.Id").
		Where("Users.DeleteAt = 0").
		Where(sq.Eq{"TeamMembers.TeamId": teamId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return 0, model.NewAppError("SqlTeamStore.GetActiveMemberCount", "store.sql_team.get_active_member_count.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	count, err := s.GetReplica().SelectInt(queryString, args...)
	if err != nil {
		return 0, model.NewAppError("SqlTeamStore.GetActiveMemberCount", "store.sql_team.get_active_member_count.app_error", nil, "teamId="+teamId+" "+err.Error(), http.StatusInternalServerError)
	}

	return count, nil
}

func (s SqlTeamStore) SaveMember(member *model.TeamMember, maxUsersPerTeam int) (*model.TeamMember, *model.AppError) {
	members, err := s.SaveMultipleMembers([]*model.TeamMember{member}, maxUsersPerTeam)
	if err != nil {
		return nil, err
	}
	return members[0], nil
}

func (s SqlTeamStore) SaveMultipleMembers(members []*model.TeamMember, maxUsersPerTeam int) ([]*model.TeamMember, *model.AppError) {
	newTeamMembers := map[string]int{}
	users := map[string]bool{}
	for _, member := range members {
		newTeamMembers[member.TeamId] = 0
	}

	for _, member := range members {
		newTeamMembers[member.TeamId]++
		users[member.UserId] = true

		if err := member.IsValid(); err != nil {
			return nil, err
		}
	}

	teams := []string{}
	for team := range newTeamMembers {
		teams = append(teams, team)
	}

	if maxUsersPerTeam >= 0 {
		queryCount := s.GetQueryBuilder().
			Select(
				"COUNT(0) as Count, TeamMembers.TeamId as TeamId",
			).
			From("TeamMembers").
			Join("Users ON TeamMembers.UserId = Users.Id").
			Where(sq.Eq{"TeamMembers.TeamId": teams}).
			Where(sq.Eq{"TeamMembers.DeleteAt": 0}).
			Where(sq.Eq{"Users.DeleteAt": 0}).
			GroupBy("TeamMembers.TeamId")

		sqlCountQuery, argsCount, errCount := queryCount.ToSql()
		if errCount != nil {
			return nil, model.NewAppError("SqlUserStore.Save", "store.sql_user.save.member_count.app_error", nil, errCount.Error(), http.StatusInternalServerError)
		}

		// 使い捨て用のstructを定義
		var counters []struct {
			Count  int    `db:"Count"`
			TeamId string `db:"TeamId"`
		}
		_, err := s.GetMaster().Select(&counters, sqlCountQuery, argsCount...)
		if err != nil {
			return nil, model.NewAppError("SqlUserStore.Save", "store.sql_user.save.member_count.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		for teamId, newMembers := range newTeamMembers {
			existingMembers := 0
			for _, counter := range counters {
				if counter.TeamId == teamId {
					existingMembers = counter.Count
				}
			}
			if existingMembers+newMembers > maxUsersPerTeam {
				return nil, model.NewAppError("SqlUserStore.Save", "store.sql_user.save.max_accounts.app_error", nil, "", http.StatusBadRequest)
			}
		}
	}

	// bulk insert
	query := s.GetQueryBuilder().Insert("TeamMembers").Columns(teamMemberSliceColumns()...)
	for _, member := range members {
		query = query.Values(teamMemberToSlice(member)...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.SaveMember", "store.sql_team.save_member.save.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := s.GetMaster().Exec(sql, args...); err != nil {
		if IsUniqueConstraintError(err, []string{"TeamId", "PRIMARY"}) {
			return nil, model.NewAppError("SqlTeamStore.SaveMember", TEAM_MEMBER_EXISTS_ERROR, nil, err.Error(), http.StatusBadRequest)
		}
		return nil, model.NewAppError("SqlTeamStore.SaveMember", "store.sql_team.save_member.save.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	newMembers := []*model.TeamMember{}
	for _, member := range members {
		newMember := *member
		newMembers = append(newMembers, &newMember)
	}

	return newMembers, nil
}

func (s SqlTeamStore) GetTeamsByUserId(userId string) ([]*model.Team, *model.AppError) {
	var teams []*model.Team
	if _, err := s.GetReplica().Select(&teams, "SELECT Teams.* FROM Teams, TeamMembers WHERE TeamMembers.TeamId = Teams.Id AND TeamMembers.UserId = :UserId AND TeamMembers.DeleteAt = 0 AND Teams.DeleteAt = 0", map[string]interface{}{"UserId": userId}); err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetTeamsByUserId", "store.sql_team.get_all.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return teams, nil
}

func (s SqlTeamStore) GetTeamsForUser(userId string) ([]*model.TeamMember, *model.AppError) {
	query := s.membersQuery.
		Where(sq.Eq{"TeamMembers.UserId": userId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetMembers", "store.sql_team.get_members.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	members := []*model.TeamMember{}
	_, err = s.GetReplica().Select(&members, queryString, args...)
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetMembers", "store.sql_team.get_members.app_error", nil, "userId="+userId+" "+err.Error(), http.StatusInternalServerError)
	}

	return members, nil
}

func (s SqlTeamStore) Get(id string) (*model.Team, *model.AppError) {
	obj, err := s.GetReplica().Get(model.Team{}, id)
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.Get", "store.sql_team.get.finding.app_error", nil, "id="+id+", "+err.Error(), http.StatusInternalServerError)
	}
	if obj == nil {
		return nil, model.NewAppError("SqlTeamStore.Get", "store.sql_team.get.find.app_error", nil, "id="+id, http.StatusNotFound)
	}

	return obj.(*model.Team), nil
}

func (s SqlTeamStore) Update(team *model.Team) (*model.Team, *model.AppError) {
	team.PreUpdate()

	if err := team.IsValid(); err != nil {
		return nil, err
	}

	oldResult, err := s.GetMaster().Get(model.Team{}, team.Id)
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.Update", "store.sql_team.update.finding.app_error", nil, "id="+team.Id+", "+err.Error(), http.StatusInternalServerError)

	}
	if oldResult == nil {
		return nil, model.NewAppError("SqlTeamStore.Update", "store.sql_team.update.find.app_error", nil, "id="+team.Id, http.StatusBadRequest)
	}

	oldTeam := oldResult.(*model.Team)
	team.CreateAt = oldTeam.CreateAt
	team.UpdateAt = model.GetMillis()

	count, err := s.GetMaster().Update(team)
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.Update", "store.sql_team.update.updating.app_error", nil, "id="+team.Id+", "+err.Error(), http.StatusInternalServerError)
	}
	if count != 1 {
		return nil, model.NewAppError("SqlTeamStore.Update", "store.sql_team.update.app_error", nil, "id="+team.Id, http.StatusInternalServerError)
	}

	return team, nil
}

func (s SqlTeamStore) RemoveAllMembersByTeam(teamId string) *model.AppError {
	_, err := s.GetMaster().Exec("DELETE FROM TeamMembers WHERE TeamId = :TeamId", map[string]interface{}{"TeamId": teamId})
	if err != nil {
		return model.NewAppError("SqlTeamStore.RemoveMember", "store.sql_team.remove_member.app_error", nil, "team_id="+teamId+", "+err.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (s SqlTeamStore) PermanentDelete(teamId string) *model.AppError {
	if _, err := s.GetMaster().Exec("DELETE FROM Teams WHERE Id = :TeamId", map[string]interface{}{"TeamId": teamId}); err != nil {
		return model.NewAppError("SqlTeamStore.Delete", "store.sql_team.permanent_delete.app_error", nil, "teamId="+teamId+", "+err.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (s SqlTeamStore) GetByInviteId(inviteId string) (*model.Team, *model.AppError) {
	team := model.Team{}

	err := s.GetReplica().SelectOne(&team, "SELECT * FROM Teams WHERE InviteId = :InviteId", map[string]interface{}{"InviteId": inviteId})
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetByInviteId", "store.sql_team.get_by_invite_id.finding.app_error", nil, "inviteId="+inviteId+", "+err.Error(), http.StatusNotFound)
	}

	if len(inviteId) == 0 || team.InviteId != inviteId {
		return nil, model.NewAppError("SqlTeamStore.GetByInviteId", "store.sql_team.get_by_invite_id.find.app_error", nil, "inviteId="+inviteId, http.StatusNotFound)
	}
	return &team, nil
}

func (s SqlTeamStore) GetMembers(teamId string, offset int, limit int, teamMembersGetOptions *model.TeamMembersGetOptions) ([]*model.TeamMember, *model.AppError) {
	query := s.membersQuery.
		Where(sq.Eq{"TeamMembers.TeamId": teamId}).
		Where(sq.Eq{"TeamMembers.DeleteAt": 0}).
		Limit(uint64(limit)).
		Offset(uint64(offset))

	if teamMembersGetOptions == nil || teamMembersGetOptions.Sort == "" {
		query = query.OrderBy("UserId")
	}

	if teamMembersGetOptions != nil {
		if teamMembersGetOptions.Type == model.TEAM_MEMBER_TYPE_NORMAL || teamMembersGetOptions.Type == model.TEAM_MEMBER_TYPE_ADMIN {
			query = query.Where(sq.Eq{"TeamMembers.Type": teamMembersGetOptions.Type})
		}

		if teamMembersGetOptions.Sort == model.TEAM_MEMBER_SORT_TYPE_USERNAME || teamMembersGetOptions.ExcludeDeletedUsers {
			query = query.LeftJoin("Users ON TeamMembers.UserId = Users.Id")
		}

		if teamMembersGetOptions.ExcludeDeletedUsers {
			query = query.Where(sq.Eq{"Users.DeleteAt": 0})
		}

		if teamMembersGetOptions.Sort == model.TEAM_MEMBER_SORT_TYPE_USERNAME {
			query = query.OrderBy(model.TEAM_MEMBER_SORT_TYPE_USERNAME)
		}
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetMembers", "store.sql_team.get_members.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	members := []*model.TeamMember{}
	_, err = s.GetReplica().Select(&members, queryString, args...)
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetMembers", "store.sql_team.get_members.app_error", nil, "teamId="+teamId+" "+err.Error(), http.StatusInternalServerError)
	}

	return members, nil
}

func (s SqlTeamStore) GetMembersByIds(teamId string, userIds []string) ([]*model.TeamMember, *model.AppError) {
	if len(userIds) == 0 {
		return nil, model.NewAppError("SqlTeamStore.GetMembersByIds", "store.sql_team.get_members_by_ids.app_error", nil, "Invalid list of user ids", http.StatusInternalServerError)
	}

	query := s.membersQuery.
		Where(sq.Eq{"TeamMembers.TeamId": teamId}).
		Where(sq.Eq{"TeamMembers.UserId": userIds}).
		Where(sq.Eq{"TeamMembers.DeleteAt": 0})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetMembersByIds", "store.sql_team.get_members_by_ids.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	members := []*model.TeamMember{}
	if _, err := s.GetReplica().Select(&members, queryString, args...); err != nil {
		return nil, model.NewAppError("SqlTeamStore.GetMembersByIds", "store.sql_team.get_members_by_ids.app_error", nil, "teamId="+teamId+" "+err.Error(), http.StatusInternalServerError)
	}

	return members, nil
}

func (s SqlTeamStore) UpdateLastTeamIconUpdate(teamId string, curTime int64) *model.AppError {
	if _, err := s.GetMaster().Exec("UPDATE Teams SET LastPictureUpdate = :Time, UpdateAt = :Time WHERE Id = :teamId", map[string]interface{}{"Time": curTime, "teamId": teamId}); err != nil {
		return model.NewAppError("SqlTeamStore.UpdateLastTeamIconUpdate", "store.sql_team.update_last_team_icon_update.app_error", nil, "team_id="+teamId, http.StatusInternalServerError)
	}
	return nil
}

func (s SqlTeamStore) AutocompletePublic(name string) ([]*model.Team, *model.AppError) {
	likeTerm := name + "%"

	queryFormat := `
		SELECT
			Teams.*
		FROM
			Teams
		WHERE
			Teams.Type = :Type
			AND Teams.Name LIKE :Name
			AND Teams.DeleteAt = 0
		LIMIT ` + strconv.Itoa(model.TEAM_SEARCH_DEFAULT_LIMIT)

	var teams []*model.Team
	if _, err := s.GetReplica().Select(&teams, queryFormat, map[string]interface{}{"Type": model.TEAM_TYPE_PUBLIC, "Name": likeTerm}); err != nil {
		return nil, model.NewAppError("SqlTeamStore.AutocompletePublic", "store.sql_team.search.app_error", nil, "term="+likeTerm+", "+", "+err.Error(), http.StatusInternalServerError)
	}

	sort.Slice(teams, func(a, b int) bool {
		return strings.ToLower(teams[a].Name) < strings.ToLower(teams[b].Name)
	})

	return teams, nil
}

func (s SqlTeamStore) UpdateMultipleMembers(members []*model.TeamMember) ([]*model.TeamMember, *model.AppError) {
	for _, member := range members {
		if err := member.IsValid(); err != nil {
			return nil, err
		}
	}

	var transaction *gorp.Transaction
	var err error
	if transaction, err = s.GetMaster().Begin(); err != nil {
		return nil, model.NewAppError("SqlTeamStore.MigrateTeamMembers", "store.sql_team.migrate_team_members.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	defer finalizeTransaction(transaction)

	updatedMembers := []*model.TeamMember{}
	for _, member := range members {
		if _, err := transaction.Update(member); err != nil {
			return nil, model.NewAppError("SqlTeamStore.UpdateMember", "store.sql_team.update_member.app_error", nil, "team_id="+member.TeamId+", "+"user_id="+member.UserId+", "+err.Error(), http.StatusInternalServerError)
		}

		res := model.TeamMember{}
		if err := transaction.SelectOne(&res, "SELECT * FROM TeamMembers WHERE TeamMembers.TeamId = :TeamId AND TeamMembers.UserId = :UserId", map[string]interface{}{"TeamId": member.TeamId, "UserId": member.UserId}); err != nil {
			if err == sql.ErrNoRows {
				return nil, model.NewAppError("SqlTeamStore.GetMember", store.MISSING_TEAM_MEMBER_ERROR, nil, "team_id="+member.TeamId+"user_id="+member.UserId+","+err.Error(), http.StatusNotFound)
			}

			return nil, model.NewAppError("SqlTeamStore.GetMember", "store.sql_team.get_member.app_error", nil, "team_id="+member.TeamId+"user_id="+member.UserId+","+err.Error(), http.StatusInternalServerError)
		}

		updatedMembers = append(updatedMembers, &res)
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlTeamStore.MigrateTeamMembers", "store.sql_team.migrate_team_members.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return updatedMembers, nil
}

func (s SqlTeamStore) UpdateMember(member *model.TeamMember) (*model.TeamMember, *model.AppError) {
	updatedMembers, err := s.UpdateMultipleMembers([]*model.TeamMember{member})
	if err != nil {
		return nil, err
	}

	return updatedMembers[0], nil
}
