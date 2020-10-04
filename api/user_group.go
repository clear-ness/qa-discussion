package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

func (api *API) InitUserGroup() {
	api.BaseRoutes.Groups.Handle("", api.ApiSessionRequired(createGroup)).Methods("POST")

	api.BaseRoutes.GroupsForTeam.Handle("", api.ApiSessionRequired(getGroupsForTeam)).Methods("GET")
	api.BaseRoutes.GroupsForTeam.Handle("/autocomplete", api.ApiSessionRequired(autocompleteGroupsForTeam)).Methods("GET")
	// 特定ユーザーの特定チームのグループの一覧を取得
	api.BaseRoutes.User.Handle("/teams/{team_id:[A-Za-z0-9]+}/groups", api.ApiSessionRequired(getGroupsForTeamForUser)).Methods("GET")

	api.BaseRoutes.Group.Handle("", api.ApiSessionRequired(getGroup)).Methods("GET")
	api.BaseRoutes.Group.Handle("", api.ApiSessionRequired(updateGroup)).Methods("PUT")
	api.BaseRoutes.Group.Handle("", api.ApiSessionRequired(deleteGroup)).Methods("DELETE")

	// Add a user to a group by creating a group member object.
	api.BaseRoutes.GroupMember.Handle("", api.ApiSessionRequired(addGroupMember)).Methods("POST")

	// 特定グループに所属する特定メンバーを取得
	api.BaseRoutes.GroupMember.Handle("", api.ApiSessionRequired(getGroupMember)).Methods("GET")
	// 特定グループに所属するメンバー達を取得
	api.BaseRoutes.GroupMembers.Handle("", api.ApiSessionRequired(getGroupMembers)).Methods("GET")

	// group adminをいきなりgroupから削除する事は出来ない。
	// 他にadminが1人以上いる前提で、一旦normalに降格させてから、groupから削除する事なら出来る。
	//
	// 特定グループに所属する特定メンバーのtypeを変更
	api.BaseRoutes.GroupMember.Handle("/type", api.ApiSessionRequired(updateGroupMemberType)).Methods("PUT")
	// 特定グループに所属する特定メンバーを削除
	api.BaseRoutes.GroupMember.Handle("", api.ApiSessionRequired(removeGroupMember)).Methods("DELETE")
}

func createGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	group := model.UserGroupFromJson(r.Body)
	if group == nil {
		c.SetInvalidParam("group")
		return
	}
	// TODO: SanitizeInput

	team, err := c.App.GetTeam(group.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, group.TeamId, model.PERMISSION_CREATE_GROUP) {
		c.SetPermissionError(model.PERMISSION_CREATE_GROUP)
		return
	}

	if team.Type == model.TEAM_TYPE_PRIVATE && group.Type != model.GROUP_TYPE_PRIVATE {
		// private teamかつpublic groupは許可しない、弾く。
		c.SetInvalidParam("type")
		return
	}

	sg, err := c.App.CreateGroupWithUser(group, c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(sg.ToJson()))
}

func getGroupsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	team, err := c.App.GetTeam(c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	groups := &model.UserGroupList{}
	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_LIST_TEAM_GROUPS) {
		if team.Type == model.TEAM_TYPE_PUBLIC {
			groups, err = c.App.GetGroupsForTeam(c.Params.TeamId, model.GROUP_TYPE_PUBLIC, c.Params.Page*c.Params.PerPage, c.Params.PerPage)
			if err != nil {
				c.Err = err
				return
			}
		} else {
			c.SetPermissionError(model.PERMISSION_LIST_TEAM_GROUPS)
			return
		}
	} else {
		groups, err = c.App.GetGroupsForTeam(c.Params.TeamId, "", c.Params.Page*c.Params.PerPage, c.Params.PerPage)
		if err != nil {
			c.Err = err
			return
		}
	}

	w.Write([]byte(groups.ToJson()))
}

func autocompleteGroupsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	team, err := c.App.GetTeam(c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	name := r.URL.Query().Get("name")

	groups := &model.UserGroupList{}
	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_LIST_TEAM_GROUPS) {
		if team.Type == model.TEAM_TYPE_PUBLIC {
			groups, err = c.App.AutocompleteGroups(c.Params.TeamId, name, model.GROUP_TYPE_PUBLIC)
			if err != nil {
				c.Err = err
				return
			}

		} else {
			c.SetPermissionError(model.PERMISSION_LIST_TEAM_GROUPS)
			return
		}
	} else {
		groups, err = c.App.AutocompleteGroups(c.Params.TeamId, name, "")
		if err != nil {
			c.Err = err
			return
		}
	}

	w.Write([]byte(groups.ToJson()))
}

func getGroupsForTeamForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId().RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToUser(c.App.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	groups, err := c.App.GetGroupsForUser(c.Params.TeamId, c.Params.UserId, false)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(groups.ToJson()))
}

func getGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireGroupId()
	if c.Err != nil {
		return
	}

	group, err := c.App.GetGroup(c.Params.GroupId)
	if err != nil {
		c.Err = err
		return
	}

	team, err := c.App.GetTeam(group.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	if !(team.Type == model.TEAM_TYPE_PUBLIC && group.Type == model.GROUP_TYPE_PUBLIC) && !c.App.SessionHasPermissionToTeam(c.App.Session, group.TeamId, model.PERMISSION_READ_GROUP) {
		c.SetPermissionError(model.PERMISSION_READ_GROUP)
		return
	}

	w.Write([]byte(group.ToJson()))
}

func updateGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireGroupId()
	if c.Err != nil {
		return
	}

	group := model.UserGroupFromJson(r.Body)
	if group == nil {
		c.SetInvalidParam("group")
		return
	}
	if group.Id != c.Params.GroupId {
		c.SetInvalidParam("group_id")
		return
	}

	originalOldGroup, err := c.App.GetGroup(group.Id)
	if err != nil {
		c.Err = err
		return
	}
	oldGroup := originalOldGroup.DeepCopy()

	if !c.App.SessionHasPermissionToGroup(c.App.Session, c.Params.GroupId, model.PERMISSION_MANAGE_GROUP_PROPERTIES) {
		c.SetPermissionError(model.PERMISSION_MANAGE_GROUP_PROPERTIES)
		return
	}

	if oldGroup.DeleteAt > 0 {
		c.Err = model.NewAppError("updateGroup", "api.group.update_group.deleted.app_error", nil, "", http.StatusBadRequest)
		return
	}

	if len(group.Name) > 0 {
		oldGroup.Name = group.Name
	}

	if len(group.Description) > 0 {
		oldGroup.Description = group.Description
	}

	_, err = c.App.UpdateGroup(oldGroup)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(oldGroup.ToJson()))
}

func deleteGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireGroupId()
	if c.Err != nil {
		return
	}

	group, err := c.App.GetGroup(c.Params.GroupId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToGroup(c.App.Session, group.Id, model.PERMISSION_MANAGE_GROUP) {
		c.SetPermissionError(model.PERMISSION_MANAGE_GROUP)
		return
	}

	err = c.App.DeleteGroup(group)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func getGroupMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireGroupId().RequireUserId()
	if c.Err != nil {
		return
	}

	group, err := c.App.GetGroup(c.Params.GroupId)
	if err != nil {
		c.Err = err
		return
	}

	team, err := c.App.GetTeam(group.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	// public teamかつpublic groupのメンバーは誰でも見れる仕様
	if !(team.Type == model.TEAM_TYPE_PUBLIC && group.Type == model.GROUP_TYPE_PUBLIC) && !c.App.SessionHasPermissionToGroup(c.App.Session, c.Params.GroupId, model.PERMISSION_READ_GROUP) {
		c.SetPermissionError(model.PERMISSION_READ_GROUP)
		return
	}

	member, err := c.App.GetGroupMember(c.Params.GroupId, c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(member.ToJson()))
}

func getGroupMembers(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireGroupId()
	if c.Err != nil {
		return
	}

	group, err := c.App.GetGroup(c.Params.GroupId)
	if err != nil {
		c.Err = err
		return
	}

	team, err := c.App.GetTeam(group.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	// public teamかつpublic groupのメンバーは誰でも見れる仕様
	if !(team.Type == model.TEAM_TYPE_PUBLIC && group.Type == model.GROUP_TYPE_PUBLIC) && !c.App.SessionHasPermissionToGroup(c.App.Session, c.Params.GroupId, model.PERMISSION_READ_GROUP) {
		c.SetPermissionError(model.PERMISSION_READ_GROUP)
		return
	}

	members, err := c.App.GetGroupMembersPage(c.Params.GroupId, "", c.Params.Page, c.Params.PerPage)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(members.ToJson()))
}

func addGroupMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId().RequireGroupId()
	if c.Err != nil {
		return
	}

	member := &model.GroupMember{
		GroupId: c.Params.GroupId,
		UserId:  c.Params.UserId,
	}

	group, err := c.App.GetGroup(member.GroupId)
	if err != nil {
		c.Err = err
		return
	}

	isNewMembership := false
	if _, err = c.App.GetGroupMember(member.GroupId, member.UserId); err != nil {
		if err.Id == store.MISSING_GROUP_MEMBER_ERROR {
			isNewMembership = true
		} else {
			c.Err = err
			return
		}
	}

	isSelfAdd := member.UserId == c.App.Session.UserId

	if isSelfAdd && isNewMembership {
		// self joinはnormal teamメンバーには禁止する、team adminなら出来る。
		if !c.App.SessionHasPermissionToTeam(c.App.Session, group.TeamId, model.PERMISSION_JOIN_GROUPS) {
			c.SetPermissionError(model.PERMISSION_JOIN_GROUPS)
			return
		}
	} else if isSelfAdd && !isNewMembership {
		// already in the group
	} else if !isSelfAdd {
		// group adminならそのチームのメンバーをgroupに追加出来る。
		if !c.App.SessionHasPermissionToGroup(c.App.Session, group.Id, model.PERMISSION_MANAGE_GROUP_MEMBERS) {
			c.SetPermissionError(model.PERMISSION_MANAGE_GROUP_MEMBERS)
			return
		}
	}

	gm, err := c.App.AddGroupMember(member.UserId, group)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(gm.ToJson()))
}

func updateGroupMemberType(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireGroupId().RequireUserId()
	if c.Err != nil {
		return
	}

	props := model.MapFromJson(r.Body)

	newType := props["type"]
	if newType != model.GROUP_MEMBER_TYPE_NORMAL && newType != model.GROUP_MEMBER_TYPE_ADMIN {
		c.SetInvalidParam("type")
		return
	}

	if !c.App.SessionHasPermissionToGroup(c.App.Session, c.Params.GroupId, model.PERMISSION_MANAGE_GROUP_MEMBER_TYPE) {
		c.SetPermissionError(model.PERMISSION_MANAGE_GROUP_MEMBER_TYPE)
		return
	}

	if _, err := c.App.UpdateGroupMemberType(c.Params.GroupId, c.Params.UserId, newType); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func removeGroupMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireGroupId().RequireUserId()
	if c.Err != nil {
		return
	}

	group, err := c.App.GetGroup(c.Params.GroupId)
	if err != nil {
		c.Err = err
		return
	}

	if c.Params.UserId != c.App.Session.UserId {
		if !c.App.SessionHasPermissionToGroup(c.App.Session, group.Id, model.PERMISSION_MANAGE_GROUP_MEMBERS) {
			c.SetPermissionError(model.PERMISSION_MANAGE_GROUP_MEMBERS)
			return
		}
	}
	if err = c.App.RemoveUserFromGroup(c.Params.UserId, c.App.Session.UserId, group); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
