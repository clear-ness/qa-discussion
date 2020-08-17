package api

import (
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitTeam() {
	api.BaseRoutes.Teams.Handle("", api.ApiSessionRequired(createTeam)).Methods("POST")
	// (左サイドバーでチーム無選択状態で) public teamは検索出来る。
	api.BaseRoutes.Teams.Handle("/autocomplete", api.ApiHandler(autocompletePublicTeams)).Methods("POST")
	api.BaseRoutes.TeamsForUser.Handle("", api.ApiSessionRequired(getTeamsForUser)).Methods("GET")
	//api.BaseRoutes.TeamsForUser.Handle("/unread", api.ApiSessionRequired(getTeamsUnreadForUser)).Methods("GET")

	api.BaseRoutes.Team.Handle("", api.ApiSessionRequired(getTeam)).Methods("GET")
	api.BaseRoutes.Team.Handle("", api.ApiSessionRequired(updateTeam)).Methods("PUT")
	api.BaseRoutes.Team.Handle("", api.ApiSessionRequired(deleteTeam)).Methods("DELETE")

	// Regenerates the invite id used in invite links of a team
	api.BaseRoutes.Team.Handle("/regenerate_invite_id", api.ApiSessionRequired(regenerateTeamInviteId)).Methods("POST")
	api.BaseRoutes.Team.Handle("/image", api.ApiSessionRequired(setTeamIcon)).Methods("POST")
	api.BaseRoutes.Team.Handle("/image", api.ApiSessionRequired(removeTeamIcon)).Methods("DELETE")

	// invite linkで招待する場合:
	// Invite users to the existing team using the user's email.
	api.BaseRoutes.Team.Handle("/invite/email", api.ApiSessionRequired(inviteUsersToTeam)).Methods("POST")
	// Using either an invite id or hash/data pair from an email invite link, add a user to a team.
	api.BaseRoutes.Teams.Handle("/members/invite", api.ApiSessionRequired(addUserToTeamFromInvite)).Methods("POST")

	// public teamだったとしてもteamに所属するユーザーは公開されない。
	// 特定ユーザーが所属するteamsは公開されない、本人とsystem adminのみ見れる。
	// ただし、public teamかつpublic groupに所属するユーザーに限っては公開される。
	api.BaseRoutes.TeamMember.Handle("", api.ApiSessionRequired(getTeamMember)).Methods("GET")
	// Get a page team members list based on query string parameters - team id, page and per page.
	api.BaseRoutes.TeamMembers.Handle("", api.ApiSessionRequired(getTeamMembers)).Methods("GET")
	// Get a list of team members based on a provided array of user ids.
	api.BaseRoutes.TeamMembers.Handle("/ids", api.ApiSessionRequired(getTeamMembersByIds)).Methods("POST")
	// Get a list of team members for a user. Useful for getting the ids of teams the user is on and the types in those teams.
	api.BaseRoutes.TeamMembersForUser.Handle("", api.ApiSessionRequired(getTeamMembersForUser)).Methods("GET")

	// team adminをいきなりteamから削除する事は出来ない。
	// 他にadminが1人以上いる前提で、一旦normalに降格させてから、teamから削除する事なら出来る。
	api.BaseRoutes.TeamMember.Handle("/type", api.ApiSessionRequired(updateTeamMemberType)).Methods("PUT")
	// Delete the team member object for a user, effectively removing them from a team.
	api.BaseRoutes.TeamMember.Handle("", api.ApiSessionRequired(removeTeamMember)).Methods("DELETE")
}

func createTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	team := model.TeamFromJson(r.Body)
	if team == nil {
		c.SetInvalidParam("team")
		return
	}
	// チームのメアドはチームの作者のメアド
	team.Email = strings.ToLower(team.Email)

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_TEAM) {
		c.Err = model.NewAppError("createTeam", "api.team.is_team_creation_allowed.disabled.app_error", nil, "", http.StatusForbidden)
		return
	}

	rteam, err := c.App.CreateTeamWithUser(team, c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(rteam.ToJson()))
}

func autocompletePublicTeams(c *Context, w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if len(name) <= 0 {
		c.SetInvalidParam("name")
		return
	}

	teams, err := c.App.AutocompletePublicTeams(name)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.TeamListToJson(teams)))
}

func getTeamsForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if c.App.Session.UserId != c.Params.UserId && !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_READ_OTHERS_TEAMS) {
		c.SetPermissionError(model.PERMISSION_READ_OTHERS_TEAMS)
		return
	}

	teams, err := c.App.GetTeamsForUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	c.App.SanitizeTeams(c.App.Session, teams)

	w.Write([]byte(model.TeamListToJson(teams)))
}

func getTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	team, err := c.App.GetTeam(c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	if team.Type != model.TEAM_TYPE_PUBLIC && !c.App.SessionHasPermissionToTeam(c.App.Session, team.Id, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	c.App.SanitizeTeam(c.App.Session, team)

	w.Write([]byte(team.ToJson()))
}

func updateTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	team := model.TeamFromJson(r.Body)
	if team == nil {
		c.SetInvalidParam("team")
		return
	}
	team.Email = strings.ToLower(team.Email)

	if team.Id != c.Params.TeamId {
		c.SetInvalidParam("id")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	updatedTeam, err := c.App.UpdateTeam(team)
	if err != nil {
		c.Err = err
		return
	}

	c.App.SanitizeTeam(c.App.Session, updatedTeam)

	w.Write([]byte(updatedTeam.ToJson()))
}

func deleteTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	var err *model.AppError
	if c.Params.Permanent {
		err = c.App.PermanentDeleteTeamId(c.Params.TeamId)
	} else {
		err = c.App.SoftDeleteTeam(c.Params.TeamId)
	}

	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func regenerateTeamInviteId(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	patchedTeam, err := c.App.RegenerateTeamInviteId(c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	c.App.SanitizeTeam(c.App.Session, patchedTeam)

	w.Write([]byte(patchedTeam.ToJson()))
}

func inviteUsersToTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_INVITE_USER_TO_TEAM) {
		c.SetPermissionError(model.PERMISSION_INVITE_USER_TO_TEAM)
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_ADD_USER_TO_TEAM) {
		c.SetPermissionError(model.PERMISSION_INVITE_USER_TO_TEAM)
		return
	}

	emailList := model.ArrayFromJson(r.Body)
	for i := range emailList {
		emailList[i] = strings.ToLower(emailList[i])
	}

	if len(emailList) == 0 {
		c.SetInvalidParam("user_email")
		return
	}

	err := c.App.InviteNewUsersToTeam(emailList, c.Params.TeamId, c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}
	ReturnStatusOK(w)
}

// こちらは既存のユーザーをチームに参加させる場合。
// つまりsessionが必要なリクエスト。
func addUserToTeamFromInvite(c *Context, w http.ResponseWriter, r *http.Request) {
	// inviteUsersToTeam経由 とは異なるので注意。
	tokenId := r.URL.Query().Get("token")
	// チームのlink id 経由で参加する場合 (teamのlinkを知ってさえいれば参加出来る)
	inviteId := r.URL.Query().Get("invite_id")

	var member *model.TeamMember
	var err *model.AppError

	if len(tokenId) > 0 {
		member, err = c.App.AddTeamMemberByToken(c.App.Session.UserId, tokenId)
	} else if len(inviteId) > 0 {
		member, err = c.App.AddTeamMemberByInviteId(inviteId, c.App.Session.UserId)
	} else {
		err = model.NewAppError("addTeamMember", "api.team.add_user_to_team.missing_parameter.app_error", nil, "", http.StatusBadRequest)
	}

	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(member.ToJson()))
}

func getTeamMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	team, err := c.App.GetTeamMember(c.Params.TeamId, c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(team.ToJson()))
}

func getTeamMembers(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	sort := r.URL.Query().Get("sort")

	excludeDeletedUsers := r.URL.Query().Get("exclude_deleted_users")
	excludeDeletedUsersBool, _ := strconv.ParseBool(excludeDeletedUsers)

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	teamMembersGetOptions := &model.TeamMembersGetOptions{
		Sort:                sort,
		ExcludeDeletedUsers: excludeDeletedUsersBool,
	}

	members, err := c.App.GetTeamMembers(c.Params.TeamId, c.Params.Page*c.Params.PerPage, c.Params.PerPage, teamMembersGetOptions)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.TeamMembersToJson(members)))
}

func getTeamMembersByIds(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	userIds := model.ArrayFromJson(r.Body)

	if len(userIds) == 0 {
		c.SetInvalidParam("user_ids")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	members, err := c.App.GetTeamMembersByIds(c.Params.TeamId, userIds)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.TeamMembersToJson(members)))
}

func getTeamMembersForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToUser(c.App.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	members, err := c.App.GetTeamMembersForUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.TeamMembersToJson(members)))
}

func setTeamIcon(c *Context, w http.ResponseWriter, r *http.Request) {
	defer io.Copy(ioutil.Discard, r.Body)

	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	if r.ContentLength > *c.App.Config().FileSettings.MaxFileSize {
		c.Err = model.NewAppError("setTeamIcon", "api.team.set_team_icon.too_large.app_error", nil, "", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(*c.App.Config().FileSettings.MaxFileSize); err != nil {
		c.Err = model.NewAppError("setTeamIcon", "api.team.set_team_icon.parse.app_error", nil, err.Error(), http.StatusBadRequest)
		return
	}

	m := r.MultipartForm

	imageArray, ok := m.File["image"]
	if !ok {
		c.Err = model.NewAppError("setTeamIcon", "api.team.set_team_icon.no_file.app_error", nil, "", http.StatusBadRequest)
		return
	}

	if len(imageArray) <= 0 {
		c.Err = model.NewAppError("setTeamIcon", "api.team.set_team_icon.array.app_error", nil, "", http.StatusBadRequest)
		return
	}

	imageData := imageArray[0]

	if err := c.App.SetTeamIcon(c.Params.TeamId, imageData); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func removeTeamIcon(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	if err := c.App.RemoveTeamIcon(c.Params.TeamId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func updateTeamMemberType(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	props := model.MapFromJson(r.Body)

	newType := props["type"]
	if newType != model.TEAM_MEMBER_TYPE_NORMAL && newType != model.TEAM_MEMBER_TYPE_ADMIN {
		c.SetInvalidParam("type")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	if _, err := c.App.UpdateTeamMemberType(c.Params.TeamId, c.Params.UserId, newType); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func removeTeamMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	if c.App.Session.UserId != c.Params.UserId {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_REMOVE_USER_FROM_TEAM) {
			c.SetPermissionError(model.PERMISSION_REMOVE_USER_FROM_TEAM)
			return
		}
	}
	if err := c.App.RemoveUserFromTeam(c.Params.TeamId, c.Params.UserId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
