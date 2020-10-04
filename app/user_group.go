package app

import (
	"net/http"
	"strings"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

func (a *App) CreateGroupWithUser(group *model.UserGroup, userId string) (*model.UserGroup, *model.AppError) {
	if len(group.TeamId) == 0 {
		return nil, model.NewAppError("CreateGroupWithUser", "app.group.create_group.no_team_id.app_error", nil, "", http.StatusBadRequest)
	}

	count, err := a.GetNumberOfGroupsOnTeam(group.TeamId)
	if err != nil {
		return nil, err
	}

	if int64(count+1) > *a.Config().TeamSettings.MaxGroupsPerTeam {
		return nil, model.NewAppError("CreateGroupWithUser", "api.group.create_group.max_group_limit.app_error", map[string]interface{}{"MaxGroupsPerTeam": *a.Config().TeamSettings.MaxGroupsPerTeam}, "", http.StatusBadRequest)
	}

	group.UserId = userId

	rgroup, err := a.CreateGroup(group, true)
	if err != nil {
		return nil, err
	}

	return rgroup, nil
}

func (a *App) GetNumberOfGroupsOnTeam(teamId string) (int, *model.AppError) {
	list, err := a.Srv.Store.UserGroup().GetTeamGroups(teamId)
	if err != nil {
		if err.Id == store.MISSING_GROUPS_ERROR {
			return 0, nil
		}

		return 0, err
	}

	return len(*list), nil
}

func (a *App) CreateGroup(group *model.UserGroup, addMember bool) (*model.UserGroup, *model.AppError) {
	group.Name = strings.TrimSpace(group.Name)

	sg, err := a.Srv.Store.UserGroup().Save(group, *a.Config().TeamSettings.MaxGroupsPerTeam)
	if err != nil {
		return nil, err
	}

	if addMember {
		user, err := a.Srv.Store.User().Get(group.UserId)
		if err != nil {
			return nil, err
		}

		// グループの作者をgroup adminとする。
		gm := &model.GroupMember{
			GroupId: sg.Id,
			UserId:  user.Id,
			Type:    model.GROUP_MEMBER_TYPE_ADMIN,
		}
		if _, err := a.Srv.Store.UserGroup().SaveMember(gm); err != nil {
			return nil, err
		}

		if err := a.Srv.Store.GroupMemberHistory().LogJoinEvent(group.UserId, sg.Id, model.GetMillis()); err != nil {
			mlog.Error("Failed to update GroupMemberHistory table", mlog.Err(err))
			return nil, model.NewAppError("CreateGroup", "app.group_member_history.log_join_event.internal_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	return sg, nil
}

func (a *App) GetGroupsForTeam(teamId string, groupType string, offset int, limit int) (*model.UserGroupList, *model.AppError) {
	return a.Srv.Store.UserGroup().GetGroupsForTeam(teamId, groupType, offset, limit)
}

func (a *App) AutocompleteGroups(teamId string, term string, groupType string) (*model.UserGroupList, *model.AppError) {
	term = strings.TrimSpace(term)
	return a.Srv.Store.UserGroup().AutocompleteInTeam(teamId, term, groupType, false)
}

func (a *App) GetGroupsForUser(teamId string, userId string, includeDeleted bool) (*model.UserGroupList, *model.AppError) {
	return a.Srv.Store.UserGroup().GetGroups(teamId, userId, includeDeleted)
}

func (a *App) GetGroup(groupId string) (*model.UserGroup, *model.AppError) {
	return a.Srv.Store.UserGroup().Get(groupId)
}

func (a *App) UpdateGroup(group *model.UserGroup) (*model.UserGroup, *model.AppError) {
	rgroup, err := a.Srv.Store.UserGroup().Update(group)
	if err != nil {
		return nil, err
	}

	return rgroup, nil
}

func (a *App) DeleteGroup(group *model.UserGroup) *model.AppError {
	if group.DeleteAt > 0 {
		err := model.NewAppError("deleteGroup", "api.group.delete_group.deleted.app_error", nil, "", http.StatusBadRequest)
		return err
	}

	deleteAt := model.GetMillis()
	if err := a.Srv.Store.UserGroup().Delete(group.Id, deleteAt); err != nil {
		return model.NewAppError("DeleteGroup", "app.group.delete.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) GetGroupMembersPage(groupId string, memberType string, page, perPage int) (*model.GroupMembers, *model.AppError) {
	return a.Srv.Store.UserGroup().GetMembers(groupId, memberType, page*perPage, perPage)
}

func (a *App) GetGroupMember(groupId string, userId string) (*model.GroupMember, *model.AppError) {
	return a.Srv.Store.UserGroup().GetMember(groupId, userId)
}

func (a *App) AddGroupMember(userId string, group *model.UserGroup) (*model.GroupMember, *model.AppError) {
	if member, err := a.Srv.Store.UserGroup().GetMember(group.Id, userId); err != nil {
		if err.Id != store.MISSING_GROUP_MEMBER_ERROR {
			return nil, err
		}
	} else {
		return member, nil
	}

	var user *model.User
	var err *model.AppError
	if user, err = a.GetUser(userId); err != nil {
		return nil, err
	}

	gm, err := a.AddUserToGroup(user, group)
	if err != nil {
		return nil, err
	}

	return gm, nil
}

func (a *App) addUserToGroup(user *model.User, group *model.UserGroup, teamMember *model.TeamMember) (*model.GroupMember, *model.AppError) {
	groupMember, err := a.Srv.Store.UserGroup().GetMember(group.Id, user.Id)
	if err != nil {
		if err.Id != store.MISSING_GROUP_MEMBER_ERROR {
			return nil, err
		}
	} else {
		return groupMember, nil
	}

	// デフォルトではnormal user
	// 後からtypeを更新出来る形式。
	newMember := &model.GroupMember{
		GroupId: group.Id,
		UserId:  user.Id,
		Type:    model.GROUP_MEMBER_TYPE_NORMAL,
	}

	newMember, err = a.Srv.Store.UserGroup().SaveMember(newMember)
	if err != nil {
		mlog.Error("Failed to add member", mlog.String("user_id", user.Id), mlog.String("group_id", group.Id), mlog.Err(err))
		return nil, model.NewAppError("AddUserToGroup", "api.group.add_user.to.group.failed.app_error", nil, "", http.StatusInternalServerError)
	}

	if nErr := a.Srv.Store.GroupMemberHistory().LogJoinEvent(user.Id, group.Id, model.GetMillis()); nErr != nil {
		mlog.Error("Failed to update GroupMemberHistory table", mlog.Err(nErr))
		return nil, model.NewAppError("AddUserToGroup", "app.group_member_history.log_join_event.internal_error", nil, nErr.Error(), http.StatusInternalServerError)
	}

	return newMember, nil
}

func (a *App) AddUserToGroup(user *model.User, group *model.UserGroup) (*model.GroupMember, *model.AppError) {
	teamMember, err := a.Srv.Store.Team().GetMember(group.TeamId, user.Id)
	// ユーザーをグループに追加する場合、関連するteamにそのユーザーが所蔵していることが前提。
	if err != nil {
		return nil, err
	}
	if teamMember.DeleteAt > 0 {
		return nil, model.NewAppError("AddUserToGroup", "api.group.add_user.to.group.failed.deleted.app_error", nil, "", http.StatusBadRequest)
	}

	newMember, err := a.addUserToGroup(user, group, teamMember)
	if err != nil {
		return nil, err
	}

	return newMember, nil
}

func (a *App) UpdateGroupMemberType(groupId string, userId string, newType string) (*model.GroupMember, *model.AppError) {
	var member *model.GroupMember
	var err *model.AppError
	if member, err = a.GetGroupMember(groupId, userId); err != nil {
		return nil, err
	}

	if member.Type == newType {
		return nil, model.NewAppError("UpdateGroupMemberType", "api.group.update_group_member_type.same_type.app_error", nil, "", http.StatusBadRequest)
	}

	// group adminの数が1人以下ならadmin → normalに変更不能にする
	members, err := a.GetGroupMembersPage(groupId, model.GROUP_MEMBER_TYPE_ADMIN, 0, model.GROUP_MEMBER_SEARCH_DEFAULT_LIMIT)
	if member.Type == model.GROUP_MEMBER_TYPE_ADMIN && newType != model.GROUP_MEMBER_TYPE_ADMIN && len(*members) <= 1 {
		return nil, model.NewAppError("UpdateGroupMemberType", "api.group.update_group_member_type.missing_admin.app_error", nil, "", http.StatusBadRequest)
	}

	member.Type = newType
	member, err = a.Srv.Store.UserGroup().UpdateMember(member)
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (a *App) removeUserFromGroup(userIdToRemove string, removerUserId string, group *model.UserGroup) *model.AppError {
	groupMember, err := a.GetGroupMember(group.Id, userIdToRemove)
	if err != nil && err.Id == store.MISSING_GROUP_MEMBER_ERROR {
		return model.NewAppError("removeUserFromGroup", "app.group.remove_user_from_group.missing.internal_error", nil, err.Error(), http.StatusBadRequest)
	} else if err != nil {
		return model.NewAppError("removeUserFromGroup", "app.group.remove_user_from_group.missing.internal_error", nil, err.Error(), http.StatusInternalServerError)
	}

	// group adminはいきなりはgroupから削除不能。一旦normalに戻す必要あり。
	if groupMember.Type == model.GROUP_MEMBER_TYPE_ADMIN {
		return model.NewAppError("removeUserFromGroup", "api.group.remove_user_from_group.admin.app_error", nil, "", http.StatusBadRequest)
	}

	if err := a.Srv.Store.UserGroup().RemoveMember(group.Id, userIdToRemove); err != nil {
		return err
	}

	if err := a.Srv.Store.GroupMemberHistory().LogLeaveEvent(userIdToRemove, group.Id, model.GetMillis()); err != nil {
		return model.NewAppError("removeUserFromGroup", "app.group_member_history.log_leave_event.internal_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) RemoveUserFromGroup(userIdToRemove string, removerUserId string, group *model.UserGroup) *model.AppError {
	var err *model.AppError
	if err = a.removeUserFromGroup(userIdToRemove, removerUserId, group); err != nil {
		return err
	}

	return nil
}
