package sqlstore

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/go-gorp/gorp"
)

type SqlUserGroupStore struct {
	store.Store

	membersQuery sq.SelectBuilder
}

func NewSqlUserGroupStore(sqlStore store.Store) store.UserGroupStore {
	s := &SqlUserGroupStore{
		Store: sqlStore,
	}

	s.membersQuery = s.GetQueryBuilder().Select("GroupMembers.*").From("GroupMembers")

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.UserGroup{}, "UserGroups").SetKeys(false, "Id")

		db.AddTableWithName(model.GroupMember{}, "GroupMembers").SetKeys(false, "GroupId", "UserId")
	}

	return s
}

func groupMemberSliceColumns() []string {
	return []string{"GroupId", "UserId", "Type"}
}

func groupMemberToSlice(member *model.GroupMember) []interface{} {
	resultSlice := []interface{}{}
	resultSlice = append(resultSlice, member.GroupId)
	resultSlice = append(resultSlice, member.UserId)
	resultSlice = append(resultSlice, member.Type)

	return resultSlice
}

func (s SqlUserGroupStore) GetTeamGroups(teamId string) (*model.UserGroupList, *model.AppError) {
	data := &model.UserGroupList{}
	_, err := s.GetReplica().Select(data, "SELECT * FROM UserGroups WHERE TeamId = :TeamId ORDER BY Name", map[string]interface{}{"TeamId": teamId})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlUserGroupStore.GetGroups", store.MISSING_GROUPS_ERROR, nil, "", http.StatusNotFound)
		}

		return nil, model.NewAppError("SqlUserGroupStore.GetTeamGroups", "store.sql_group.get_groups.get.app_error", nil, "teamId="+teamId+",  err="+err.Error(), http.StatusInternalServerError)
	}

	return data, nil
}

func (s SqlUserGroupStore) Save(group *model.UserGroup, maxGroupsPerTeam int64) (*model.UserGroup, *model.AppError) {
	if group.DeleteAt != 0 {
		return nil, model.NewAppError("SqlUserGroupStore.Save", "store.sql_group.save.already_deleted.app_error", nil, "", http.StatusInternalServerError)
	}

	if len(group.Id) > 0 {
		return nil, model.NewAppError("SqlUserGroupStore.Save", "store.sql_group.save.existing.app_error", nil, "", http.StatusBadRequest)
	}

	group.PreSave()
	if err := group.IsValid(); err != nil {
		return nil, err
	}

	if maxGroupsPerTeam >= 0 {
		if count, err := s.GetReplica().SelectInt(`SELECT count(*) FROM UserGroups WHERE TeamId = :TeamId AND DeleteAt = 0`, map[string]interface{}{"TeamId": group.TeamId}); err != nil {
			return nil, model.NewAppError("SqlUserGroupStore.Save", "store.sql_group.save.get_count.app_error", nil, "", http.StatusInternalServerError)
		} else if count >= maxGroupsPerTeam {
			return nil, model.NewAppError("SqlUserGroupStore.Save", "store.sql_group.save.too_many.app_error", nil, "", http.StatusBadRequest)
		}
	}

	if err := s.GetMaster().Insert(group); err != nil {
		if IsUniqueConstraintError(err, []string{"Name", "TeamId"}) {
			return nil, model.NewAppError("SqlGroupStore.Save", "store.sql_group.save.exists.app_error", nil, err.Error(), http.StatusBadRequest)
		}

		return nil, model.NewAppError("SqlGroupStore.Save", "store.sql_group.save.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return group, nil
}

func (s SqlUserGroupStore) SaveMultipleMembers(members []*model.GroupMember) ([]*model.GroupMember, *model.AppError) {
	members, err := s.saveMultipleMembers(members)
	if err != nil {
		return nil, model.NewAppError("SaveMultipleMembers", "app.group.save_multiple.internal_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return members, nil
}

func (s SqlUserGroupStore) SaveMember(member *model.GroupMember) (*model.GroupMember, *model.AppError) {
	newMembers, appErr := s.SaveMultipleMembers([]*model.GroupMember{member})
	if appErr != nil {
		return nil, appErr
	}
	return newMembers[0], nil
}

func (s SqlUserGroupStore) saveMultipleMembers(members []*model.GroupMember) ([]*model.GroupMember, error) {
	query := s.GetQueryBuilder().Insert("GroupMembers").Columns(groupMemberSliceColumns()...)
	for _, member := range members {
		if err := member.IsValid(); err != nil {
			return nil, err
		}
		query = query.Values(groupMemberToSlice(member)...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "group_members_tosql")
	}
	if _, err := s.GetMaster().Exec(sql, args...); err != nil {
		return nil, errors.Wrap(err, "group_members_save")
	}

	return members, nil
}

func (s SqlUserGroupStore) GetGroupsForTeam(teamId string, groupType string, offset int, limit int) (*model.UserGroupList, *model.AppError) {
	args := map[string]interface{}{
		"TeamId": teamId,
		"Limit":  limit,
		"Offset": offset,
	}

	typeFilter := "AND UserGroups.Type = :Type"
	if groupType == model.GROUP_TYPE_PUBLIC || groupType == model.GROUP_TYPE_PRIVATE {
		args["Type"] = groupType
	} else {
		typeFilter = ""
	}

	groups := &model.UserGroupList{}
	_, err := s.GetReplica().Select(groups, `
		SELECT
			UserGroups.*
		FROM
			UserGroups
		WHERE
			UserGroups.TeamId = :TeamId
			AND UserGroups.DeleteAt = 0
			`+typeFilter+`
		ORDER BY UserGroups.Name
		LIMIT :Limit
		OFFSET :Offset
		`, args)
	if err != nil {
		return nil, model.NewAppError("SqlUserGroupStore.GetGroupsForTeam", "store.sql_group.get_groups.get.app_error", nil, "teamId="+teamId+", err="+err.Error(), http.StatusInternalServerError)
	}

	return groups, nil
}

func (s SqlUserGroupStore) AutocompleteInTeam(teamId string, term string, groupType string, includeDeleted bool) (*model.UserGroupList, *model.AppError) {
	args := map[string]interface{}{
		"TeamId": teamId,
	}

	deleteFilter := "AND UserGroups.DeleteAt = 0"
	if includeDeleted {
		deleteFilter = ""
	}

	typeFilter := "AND UserGroups.Type = :Type"
	if groupType == model.GROUP_TYPE_PUBLIC || groupType == model.GROUP_TYPE_PRIVATE {
		args["Type"] = groupType
	} else {
		typeFilter = ""
	}

	queryFormat := `
		SELECT
			UserGroups.*
		FROM
			UserGroups
		WHERE
			UserGroups.TeamId = :TeamId
			` + deleteFilter + `
			` + typeFilter + `
			%v
		LIMIT ` + strconv.Itoa(model.GROUP_SEARCH_DEFAULT_LIMIT)

	var groups model.UserGroupList
	if likeClause, likeTerm := s.buildLIKEClause(term, "UserGroups.Name"); likeClause == "" {
		if _, err := s.GetReplica().Select(&groups, fmt.Sprintf(queryFormat, ""), args); err != nil {
			return nil, model.NewAppError("SqlUserGroupStore.AutocompleteInTeam", "store.sql_group.search.app_error", nil, "term="+term+", "+", "+err.Error(), http.StatusInternalServerError)
		}
	} else {
		query := fmt.Sprintf(queryFormat, "AND "+likeClause)
		args["LikeTerm"] = likeTerm
		if _, err := s.GetReplica().Select(&groups, query, args); err != nil {
			return nil, model.NewAppError("SqlUserGroupStore.AutocompleteInTeam", "store.sql_group.search.app_error", nil, "term="+term+", "+", "+err.Error(), http.StatusInternalServerError)
		}
	}

	sort.Slice(groups, func(a, b int) bool {
		return strings.ToLower(groups[a].Name) < strings.ToLower(groups[b].Name)
	})

	return &groups, nil
}

func (s SqlUserGroupStore) buildLIKEClause(term string, searchColumns string) (likeClause, likeTerm string) {
	likeTerm = sanitizeSearchTerm(term, "*")
	if likeTerm == "" {
		return
	}

	var searchFields []string
	for _, field := range strings.Split(searchColumns, ", ") {
		searchFields = append(searchFields, fmt.Sprintf("%s LIKE %s escape '*'", field, ":LikeTerm"))
	}

	likeClause = fmt.Sprintf("(%s)", strings.Join(searchFields, " OR "))
	likeTerm += "%"
	return
}

func (s SqlUserGroupStore) GetGroups(teamId string, userId string, includeDeleted bool) (*model.UserGroupList, *model.AppError) {
	query := "SELECT UserGroups.* FROM UserGroups, GroupMembers WHERE Id = GroupId AND UserId = :UserId AND DeleteAt = 0 AND TeamId = :TeamId ORDER BY Name"
	if includeDeleted {
		query = "SELECT UserGroups.* FROM UserGroups, GroupMembers WHERE UserGroups.Id = GroupMembers.GroupId AND GroupMembers.UserId = :UserId AND UserGroups.TeamId = :TeamId ORDER BY Name"
	}

	groups := &model.UserGroupList{}
	_, err := s.GetReplica().Select(groups, query, map[string]interface{}{"TeamId": teamId, "UserId": userId})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlUserGroupStore.GetGroups", store.MISSING_GROUPS_ERROR, nil, "", http.StatusNotFound)
		}

		return nil, model.NewAppError("SqlUserGroupStore.GetGroups", "store.sql_group.get_groups.get.app_error", nil, "teamId="+teamId+", userId="+userId+", err="+err.Error(), http.StatusInternalServerError)
	}

	if len(*groups) == 0 {
		return nil, model.NewAppError("SqlUserGroupStore.GetGroups", store.MISSING_GROUPS_ERROR, nil, "teamId="+teamId+", userId="+userId, http.StatusBadRequest)
	}

	return groups, nil
}

func (s SqlUserGroupStore) Get(id string) (*model.UserGroup, *model.AppError) {
	obj, err := s.GetReplica().Get(model.UserGroup{}, id)
	if err != nil {
		return nil, model.NewAppError("SqlUserGroupStore.Get", "store.sql_group.get.get.app_error", nil, "id="+id, http.StatusInternalServerError)
	}
	if obj == nil {
		return nil, model.NewAppError("SqlUserGroupStore.Get", "store.sql_group.get.not_found.app_error", nil, "id="+id, http.StatusNotFound)
	}

	gr := obj.(*model.UserGroup)
	return gr, nil
}

func (s SqlUserGroupStore) GetAllGroupMembersForUser(userId string) (map[string]string, *model.AppError) {
	members := &model.GroupMembers{}
	_, err := s.GetReplica().Select(members, `
			SELECT
				GroupMembers.*
			FROM
				GroupMembers
			INNER JOIN
				UserGroups ON GroupMembers.GroupId = UserGroups.Id
			WHERE
				UserGroups.DeleteAt = 0 AND
				GroupMembers.UserId = :UserId`, map[string]interface{}{"UserId": userId})

	if err != nil {
		return nil, model.NewAppError("SqlUserGroupStore.GetAllGroupMembersForUser", "store.sql_group.get_groups.get.app_error", nil, "userId="+userId+", err="+err.Error(), http.StatusInternalServerError)
	}

	result := make(map[string]string)
	for _, member := range *members {
		result[member.GroupId] = member.Type
	}

	return result, nil
}

func (s SqlUserGroupStore) Update(group *model.UserGroup) (*model.UserGroup, *model.AppError) {
	group.PreUpdate()

	if group.DeleteAt != 0 {
		return nil, model.NewAppError("SqlUserGroupStore.Update", "store.sql_group.update.deleted.app_error", nil, "", http.StatusBadRequest)
	}

	if err := group.IsValid(); err != nil {
		return nil, err
	}

	count, err := s.GetMaster().Update(group)
	if err != nil {
		if IsUniqueConstraintError(err, []string{"Name", "TeamId"}) {
			return nil, model.NewAppError("SqlUserGroupStore.Update", "store.sql_group.update.uniq.app_error", nil, err.Error(), http.StatusBadRequest)
		}

		return nil, model.NewAppError("SqlUserGroupStore.Update", "store.sql_group.update.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	if count != 1 {
		return nil, model.NewAppError("SqlUserGroupStore.Update", "store.sql_group.update.app_error", nil, "", http.StatusInternalServerError)
	}

	return group, nil
}

func (s SqlUserGroupStore) Delete(groupId string, time int64) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlUserGroupStore.DeleteGroup", "store.sql_group.delete_group.app_error", nil, "id="+groupId+", err="+errMsg, http.StatusInternalServerError)
	}

	var group *model.UserGroup
	err := s.GetReplica().SelectOne(&group, "SELECT * FROM UserGroups WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": groupId})
	if err != nil {
		return appErr(err.Error())
	}

	if _, err := s.GetMaster().Exec("UPDATE UserGroups SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"DeleteAt": time, "UpdateAt": time, "Id": groupId}); err != nil {
		return model.NewAppError("SqlUserGroupStore.DeleteGroup", "store.sql_group.delete_group.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s SqlUserGroupStore) GetMembers(groupId string, memberType string, offset, limit int) (*model.GroupMembers, *model.AppError) {
	args := map[string]interface{}{}
	args["GroupId"] = groupId
	args["Limit"] = limit
	args["Offset"] = offset

	typeClause := ""
	if memberType != "" {
		typeClause = "AND GroupMembers.Type = :MemberType"
		args["MemberType"] = memberType
	}

	members := &model.GroupMembers{}
	_, err := s.GetReplica().Select(members, `
		SELECT
			GroupMembers.*
		FROM
			GroupMembers
		INNER JOIN
			UserGroups ON GroupMembers.GroupId = UserGroups.Id
		WHERE
			UserGroups.DeleteAt = 0 AND
			GroupMembers.GroupId = :GroupId
			`+typeClause+`
		LIMIT :Limit
		OFFSET :Offset`, args)
	if err != nil {
		return nil, model.NewAppError("SqlUserGroupStore.GetMembers", "store.sql_group.get_members.app_error", nil, "group_id="+groupId+","+err.Error(), http.StatusInternalServerError)
	}

	return members, nil
}

func (s SqlUserGroupStore) GetMember(groupId string, userId string) (*model.GroupMember, *model.AppError) {
	var member *model.GroupMember
	if err := s.GetReplica().SelectOne(&member, "SELECT * FROM GroupMembers WHERE GroupMembers.GroupId = :GroupId AND GroupMembers.UserId = :UserId", map[string]interface{}{"GroupId": groupId, "UserId": userId}); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlUserGroupStore.GetMember", store.MISSING_GROUP_MEMBER_ERROR, nil, "group_id="+groupId+"user_id="+userId+","+err.Error(), http.StatusNotFound)
		}

		return nil, model.NewAppError("SqlUserGroupStore.GetMember", "store.sql_group.get_member.app_error", nil, "group_id="+groupId+"user_id="+userId+","+err.Error(), http.StatusInternalServerError)
	}

	return member, nil
}

func (s SqlUserGroupStore) UpdateMultipleMembers(members []*model.GroupMember) ([]*model.GroupMember, *model.AppError) {
	for _, member := range members {
		if err := member.IsValid(); err != nil {
			return nil, err
		}
	}

	var transaction *gorp.Transaction
	var err error
	if transaction, err = s.GetMaster().Begin(); err != nil {
		return nil, model.NewAppError("SqlUserGroupStore.MigrateGroupMembers", "store.sql_group.migrate_group_members.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	defer finalizeTransaction(transaction)

	updatedMembers := []*model.GroupMember{}
	for _, member := range members {
		if _, err := transaction.Update(member); err != nil {
			return nil, model.NewAppError("SqlUserGroupStore.UpdateMember", "store.sql_group.update_member.app_error", nil, "group_id="+member.GroupId+", "+"user_id="+member.UserId+", "+err.Error(), http.StatusInternalServerError)
		}

		res := model.GroupMember{}
		if err := transaction.SelectOne(&res, "SELECT * FROM GroupMembers WHERE GroupMembers.GroupId = :GroupId AND GroupMembers.UserId = :UserId", map[string]interface{}{"GroupId": member.GroupId, "UserId": member.UserId}); err != nil {
			if err == sql.ErrNoRows {
				return nil, model.NewAppError("SqlUserGroupStore.GetMember", store.MISSING_GROUP_MEMBER_ERROR, nil, "group_id="+member.GroupId+"user_id="+member.UserId+","+err.Error(), http.StatusNotFound)
			}
			return nil, model.NewAppError("SqlUserGroupStore.GetMember", "store.sql_group.get_member.app_error", nil, "group_id="+member.GroupId+"user_id="+member.UserId+","+err.Error(), http.StatusInternalServerError)
		}
		updatedMembers = append(updatedMembers, &res)
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlUserGroupStore.MigrateGroupMembers", "store.sql_group.migrate_group_members.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return updatedMembers, nil
}

func (s SqlUserGroupStore) UpdateMember(member *model.GroupMember) (*model.GroupMember, *model.AppError) {
	updatedMembers, err := s.UpdateMultipleMembers([]*model.GroupMember{member})
	if err != nil {
		return nil, err
	}

	return updatedMembers[0], nil
}

func (s SqlUserGroupStore) RemoveMembers(groupId string, userIds []string) *model.AppError {
	query := s.GetQueryBuilder().
		Delete("GroupMembers").
		Where(sq.Eq{"GroupId": groupId}).
		Where(sq.Eq{"UserId": userIds})

	sql, args, err := query.ToSql()
	if err != nil {
		return model.NewAppError("SqlUserGroupStore.RemoveMember", "store.sql_group.remove_member.app_error", nil, "group_id="+groupId+", "+err.Error(), http.StatusInternalServerError)
	}

	_, err = s.GetMaster().Exec(sql, args...)
	if err != nil {
		return model.NewAppError("SqlUserGroupStore.RemoveMember", "store.sql_group.remove_member.app_error", nil, "group_id="+groupId+", "+err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s SqlUserGroupStore) RemoveMember(groupId string, userId string) *model.AppError {
	return s.RemoveMembers(groupId, []string{userId})
}
