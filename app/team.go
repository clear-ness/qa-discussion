package app

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"

	"github.com/disintegration/imaging"
)

func (a *App) CreateTeam(team *model.Team) (*model.Team, *model.AppError) {
	team.InviteId = ""
	// team_store内でPreSave()によりinvite idがセットされる
	rteam, err := a.Srv.Store.Team().Save(team)
	if err != nil {
		return nil, err
	}

	return rteam, nil
}

func (a *App) CreateTeamWithUser(team *model.Team, userId string) (*model.Team, *model.AppError) {
	user, err := a.GetUser(userId)
	if err != nil {
		return nil, err
	}
	// チームの作者のメアドをチームのメアドとする
	team.Email = user.Email

	if !a.isTeamEmailAllowed(user, team) {
		return nil, model.NewAppError("isTeamEmailAllowed", "api.team.is_team_creation_allowed.domain.app_error", nil, "", http.StatusBadRequest)
	}

	rteam, err := a.CreateTeam(team)
	if err != nil {
		return nil, err
	}

	if err = a.JoinUserToTeam(rteam, user, ""); err != nil {
		return nil, err
	}

	return rteam, nil
}

func (a *App) normalizeDomains(domains string) []string {
	return strings.Fields(strings.TrimSpace(strings.ToLower(strings.Replace(strings.Replace(domains, "@", " ", -1), ",", " ", -1))))
}

func (a *App) isEmailAddressAllowed(email string, allowedDomains []string) bool {
	for _, restriction := range allowedDomains {
		domains := a.normalizeDomains(restriction)
		if len(domains) <= 0 {
			continue
		}
		matched := false
		for _, d := range domains {
			if strings.HasSuffix(email, "@"+d) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func (a *App) isTeamEmailAllowed(user *model.User, team *model.Team) bool {
	email := strings.ToLower(user.Email)
	allowedDomains := a.getAllowedDomains(user, team)
	return a.isEmailAddressAllowed(email, allowedDomains)
}

func (a *App) getAllowedDomains(user *model.User, team *model.Team) []string {
	return []string{team.AllowedDomains}
}

// Returns three values:
// 1. a pointer to the team member, if successful
// 2. a boolean: true if the user has a non-deleted team member for that team already, otherwise false.
// 3. a pointer to an AppError if something went wrong.
func (a *App) joinUserToTeam(team *model.Team, user *model.User) (*model.TeamMember, bool, *model.AppError) {
	tm := &model.TeamMember{
		TeamId: team.Id,
		UserId: user.Id,
		Type:   model.TEAM_MEMBER_TYPE_NORMAL,
	}

	rtm, err := a.Srv.Store.Team().GetMember(team.Id, user.Id)
	if err != nil {
		var tmr *model.TeamMember
		tmr, err = a.Srv.Store.Team().SaveMember(tm, *a.Config().TeamSettings.MaxUsersPerTeam)
		if err != nil {
			return nil, false, err
		}
		return tmr, false, nil
	}

	if rtm.DeleteAt == 0 {
		return rtm, true, nil
	}

	// これ以降は該当team memberが既に削除済み状態で存在する場合

	membersCount, err := a.Srv.Store.Team().GetActiveMemberCount(tm.TeamId)
	if err != nil {
		return nil, false, err
	}

	if membersCount >= int64(*a.Config().TeamSettings.MaxUsersPerTeam) {
		return nil, false, model.NewAppError("joinUserToTeam", "app.team.join_user_to_team.max_accounts.app_error", nil, "teamId="+tm.TeamId, http.StatusBadRequest)
	}

	// 以前の削除前の状態を考慮し、未削除状態に更新する
	tm.Points = rtm.Points
	member, err := a.Srv.Store.Team().UpdateMember(tm)
	if err != nil {
		return nil, false, err
	}

	return member, false, nil
}

func (a *App) JoinUserToTeam(team *model.Team, user *model.User, userRequestorId string) *model.AppError {
	if !a.isTeamEmailAllowed(user, team) {
		return model.NewAppError("JoinUserToTeam", "api.team.join_user_to_team.allowed_domains.app_error", nil, "", http.StatusBadRequest)
	}
	_, alreadyAdded, err := a.joinUserToTeam(team, user)
	if err != nil {
		return err
	}
	if alreadyAdded {
		return nil
	}

	return nil
}

func (a *App) GetTeamsForUser(userId string) ([]*model.Team, *model.AppError) {
	teams, err := a.Srv.Store.Team().GetTeamsByUserId(userId)
	for _, team := range teams {
		team.TeamImageLink = team.GetTeamImageLink(&a.Config().FileSettings)
	}

	return teams, err
}

func (a *App) SanitizeTeam(session model.Session, team *model.Team) *model.Team {
	if a.SessionHasPermissionToTeam(session, team.Id, model.PERMISSION_MANAGE_TEAM) {
		return team
	}

	team.Sanitize()

	return team
}

func (a *App) SanitizeTeams(session model.Session, teams []*model.Team) []*model.Team {
	for _, team := range teams {
		a.SanitizeTeam(session, team)
	}

	return teams
}

func (a *App) GetTeam(teamId string) (*model.Team, *model.AppError) {
	team, err := a.Srv.Store.Team().Get(teamId)
	team.TeamImageLink = team.GetTeamImageLink(&a.Config().FileSettings)
	return team, err
}

func (a *App) UpdateTeam(team *model.Team) (*model.Team, *model.AppError) {
	oldTeam, err := a.GetTeam(team.Id)
	if err != nil {
		return nil, err
	}

	oldTeam.Description = team.Description
	oldTeam.AllowedDomains = team.AllowedDomains

	oldTeam, err = a.updateTeamUnsanitized(oldTeam)
	if err != nil {
		return team, err
	}

	return oldTeam, nil
}

func (a *App) updateTeamUnsanitized(team *model.Team) (*model.Team, *model.AppError) {
	return a.Srv.Store.Team().Update(team)
}

func (a *App) PermanentDeleteTeamId(teamId string) *model.AppError {
	team, err := a.GetTeam(teamId)
	if err != nil {
		return err
	}

	return a.PermanentDeleteTeam(team)
}

func (a *App) PermanentDeleteTeam(team *model.Team) *model.AppError {
	team.DeleteAt = model.GetMillis()
	if _, err := a.Srv.Store.Team().Update(team); err != nil {
		return err
	}

	if err := a.Srv.Store.Team().RemoveAllMembersByTeam(team.Id); err != nil {
		return err
	}

	if err := a.Srv.Store.Team().PermanentDelete(team.Id); err != nil {
		return err
	}

	return nil
}

func (a *App) SoftDeleteTeam(teamId string) *model.AppError {
	team, err := a.GetTeam(teamId)
	if err != nil {
		return err
	}

	team.DeleteAt = model.GetMillis()
	if team, err = a.Srv.Store.Team().Update(team); err != nil {
		return err
	}

	return nil
}

func (a *App) RegenerateTeamInviteId(teamId string) (*model.Team, *model.AppError) {
	team, err := a.GetTeam(teamId)
	if err != nil {
		return nil, err
	}

	team.InviteId = model.NewId()

	updatedTeam, err := a.Srv.Store.Team().Update(team)
	if err != nil {
		return nil, err
	}

	return updatedTeam, nil
}

func (a *App) prepareInviteNewUsersToTeam(teamId, senderId string) (*model.User, *model.Team, *model.AppError) {
	tchan := make(chan store.StoreResult, 1)
	go func() {
		team, err := a.Srv.Store.Team().Get(teamId)
		tchan <- store.StoreResult{Data: team, Err: err}
		close(tchan)
	}()

	uchan := make(chan store.StoreResult, 1)
	go func() {
		user, err := a.Srv.Store.User().Get(senderId)
		uchan <- store.StoreResult{Data: user, Err: err}
		close(uchan)
	}()

	result := <-tchan
	if result.Err != nil {
		return nil, nil, result.Err
	}
	team := result.Data.(*model.Team)

	result = <-uchan
	if result.Err != nil {
		return nil, nil, result.Err
	}
	user := result.Data.(*model.User)

	return user, team, nil
}

func (a *App) InviteNewUsersToTeam(emailList []string, teamId, senderId string) *model.AppError {
	if len(emailList) == 0 {
		err := model.NewAppError("InviteNewUsersToTeam", "api.team.invite_members.no_one.app_error", nil, "", http.StatusBadRequest)
		return err
	}

	user, team, err := a.prepareInviteNewUsersToTeam(teamId, senderId)
	if err != nil {
		return err
	}

	allowedDomains := a.getAllowedDomains(user, team)
	var invalidEmailList []string
	for _, email := range emailList {
		if !a.isEmailAddressAllowed(email, allowedDomains) {
			invalidEmailList = append(invalidEmailList, email)
		}
	}

	if len(invalidEmailList) > 0 {
		s := strings.Join(invalidEmailList, ", ")
		err := model.NewAppError("InviteNewUsersToTeam", "api.team.invite_members.invalid_email.app_error", map[string]interface{}{"Addresses": s}, "", http.StatusBadRequest)
		return err
	}

	a.SendInviteEmails(team, user.Username, user.Id, emailList, a.GetSiteURL())

	return nil
}

func (a *App) AddTeamMemberByToken(userId, tokenId string) (*model.TeamMember, *model.AppError) {
	team, err := a.AddUserToTeamByToken(userId, tokenId)
	if err != nil {
		return nil, err
	}

	teamMember, err := a.GetTeamMember(team.Id, userId)
	if err != nil {
		return nil, err
	}

	return teamMember, nil
}

func (a *App) AddTeamMemberByInviteId(inviteId, userId string) (*model.TeamMember, *model.AppError) {
	team, err := a.AddUserToTeamByInviteId(inviteId, userId)
	if err != nil {
		return nil, err
	}

	teamMember, err := a.GetTeamMember(team.Id, userId)
	if err != nil {
		return nil, err
	}
	return teamMember, nil
}

func (a *App) AddUserToTeamByToken(userId string, tokenId string) (*model.Team, *model.AppError) {
	token, err := a.Srv.Store.Token().GetByToken(tokenId)
	if err != nil {
		return nil, model.NewAppError("AddUserToTeamByToken", "api.user.create_user.signup_link_invalid.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	if token.Type != TOKEN_TYPE_TEAM_INVITATION {
		return nil, model.NewAppError("AddUserToTeamByToken", "api.user.create_user.signup_link_invalid.app_error", nil, "", http.StatusBadRequest)
	}

	if model.GetMillis()-token.CreateAt >= INVITATION_EXPIRY_TIME {
		a.DeleteToken(token)
		return nil, model.NewAppError("AddUserToTeamByToken", "api.user.create_user.signup_link_expired.app_error", nil, "", http.StatusBadRequest)
	}

	tokenData := model.MapFromJson(strings.NewReader(token.Extra))

	tchan := make(chan store.StoreResult, 1)
	go func() {
		team, err := a.Srv.Store.Team().Get(tokenData["teamId"])
		tchan <- store.StoreResult{Data: team, Err: err}
		close(tchan)
	}()

	uchan := make(chan store.StoreResult, 1)
	go func() {
		user, err := a.Srv.Store.User().Get(userId)
		uchan <- store.StoreResult{Data: user, Err: err}
		close(uchan)
	}()

	result := <-tchan
	if result.Err != nil {
		return nil, result.Err
	}
	team := result.Data.(*model.Team)

	result = <-uchan
	if result.Err != nil {
		return nil, result.Err
	}
	user := result.Data.(*model.User)

	if err := a.JoinUserToTeam(team, user, ""); err != nil {
		return nil, err
	}

	if err := a.DeleteToken(token); err != nil {
		return nil, err
	}

	return team, nil
}

func (a *App) AddUserToTeamByInviteId(inviteId string, userId string) (*model.Team, *model.AppError) {
	tchan := make(chan store.StoreResult, 1)
	go func() {
		team, err := a.Srv.Store.Team().GetByInviteId(inviteId)
		tchan <- store.StoreResult{Data: team, Err: err}
		close(tchan)
	}()

	uchan := make(chan store.StoreResult, 1)
	go func() {
		user, err := a.Srv.Store.User().Get(userId)
		uchan <- store.StoreResult{Data: user, Err: err}
		close(uchan)
	}()

	result := <-tchan
	if result.Err != nil {
		return nil, result.Err
	}
	team := result.Data.(*model.Team)

	result = <-uchan
	if result.Err != nil {
		return nil, result.Err
	}
	user := result.Data.(*model.User)

	if err := a.JoinUserToTeam(team, user, ""); err != nil {
		return nil, err
	}

	return team, nil
}

func (a *App) GetTeamMember(teamId, userId string) (*model.TeamMember, *model.AppError) {
	return a.Srv.Store.Team().GetMember(teamId, userId)
}

func (a *App) GetTeamMembers(teamId string, offset int, limit int, teamMembersGetOptions *model.TeamMembersGetOptions) ([]*model.TeamMember, *model.AppError) {
	return a.Srv.Store.Team().GetMembers(teamId, offset, limit, teamMembersGetOptions)
}

func (a *App) GetTeamMembersByIds(teamId string, userIds []string) ([]*model.TeamMember, *model.AppError) {
	return a.Srv.Store.Team().GetMembersByIds(teamId, userIds)
}

func (a *App) GetTeamMembersForUser(userId string) ([]*model.TeamMember, *model.AppError) {
	return a.Srv.Store.Team().GetTeamsForUser(userId)
}

func (a *App) RemoveUserFromTeam(teamId string, userId string, requestorId string) *model.AppError {
	tchan := make(chan store.StoreResult, 1)
	go func() {
		team, err := a.Srv.Store.Team().Get(teamId)
		tchan <- store.StoreResult{Data: team, Err: err}
		close(tchan)
	}()

	uchan := make(chan store.StoreResult, 1)
	go func() {
		user, err := a.Srv.Store.User().Get(userId)
		uchan <- store.StoreResult{Data: user, Err: err}
		close(uchan)
	}()

	result := <-tchan
	if result.Err != nil {
		return result.Err
	}
	team := result.Data.(*model.Team)

	result = <-uchan
	if result.Err != nil {
		return result.Err
	}
	user := result.Data.(*model.User)

	if err := a.LeaveTeam(team, user, requestorId); err != nil {
		return err
	}

	return nil
}

func (a *App) LeaveTeam(team *model.Team, user *model.User, requestorId string) *model.AppError {
	teamMember, err := a.GetTeamMember(team.Id, user.Id)
	if err != nil {
		return model.NewAppError("LeaveTeam", "api.team.remove_user_from_team.missing.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	// team adminはいきなりはteamから削除不能。一旦normalに戻す必要あり。
	if teamMember.Type == model.TEAM_MEMBER_TYPE_ADMIN {
		return model.NewAppError("LeaveTeam", "api.team.remove_user_from_team.admin.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	// そのteamに関連する、現在所属しているgroupから離脱させる(物理削除)
	var groupList *model.GroupList
	if groupList, err = a.Srv.Store.Group().GetGroups(team.Id, user.Id, true); err != nil {
		if err.Id == "store.sql_group.get_groups.not_found.app_error" {
			groupList = &model.GroupList{}
		} else {
			return err
		}
	}

	for _, group := range *groupList {
		if err = a.Srv.Store.Group().RemoveMember(group.Id, user.Id); err != nil {
			return err
		}
	}

	// そのteamから離脱させる(論理削除)
	err = a.RemoveTeamMemberFromTeam(teamMember, requestorId)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) RemoveTeamMemberFromTeam(teamMember *model.TeamMember, requestorId string) *model.AppError {
	teamMember.Type = ""
	teamMember.DeleteAt = model.GetMillis()

	if _, err := a.Srv.Store.Team().UpdateMember(teamMember); err != nil {
		return err
	}

	return nil
}

func (a *App) SetTeamIcon(teamId string, imageData *multipart.FileHeader) *model.AppError {
	file, err := imageData.Open()
	if err != nil {
		return model.NewAppError("SetTeamIcon", "api.team.set_team_icon.open.app_error", nil, err.Error(), http.StatusBadRequest)
	}
	defer file.Close()
	return a.SetTeamIconFromMultiPartFile(teamId, file)
}

func (a *App) SetTeamIconFromMultiPartFile(teamId string, file multipart.File) *model.AppError {
	team, getTeamErr := a.GetTeam(teamId)
	if getTeamErr != nil {
		return model.NewAppError("SetTeamIcon", "api.team.set_team_icon.get_team.app_error", nil, getTeamErr.Error(), http.StatusBadRequest)
	}

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return model.NewAppError("SetTeamIcon", "api.team.set_team_icon.decode_config.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	if config.Width*config.Height > model.MaxImageSize {
		return model.NewAppError("SetTeamIcon", "api.team.set_team_icon.too_large.app_error", nil, "", http.StatusBadRequest)
	}

	file.Seek(0, 0)

	return a.SetTeamIconFromFile(team, file)
}

func (a *App) SetTeamIconFromFile(team *model.Team, file io.Reader) *model.AppError {
	img, _, err := image.Decode(file)
	if err != nil {
		return model.NewAppError("SetTeamIcon", "api.team.set_team_icon.decode.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	orientation, _ := getImageOrientation(file)
	img = makeImageUpright(img, orientation)

	teamIconWidthAndHeight := 128
	img = imaging.Fill(img, teamIconWidthAndHeight, teamIconWidthAndHeight, imaging.Center, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err = png.Encode(buf, img)
	if err != nil {
		return model.NewAppError("SetTeamIcon", "api.team.set_team_icon.encode.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()
	path := model.CreateTeamImageKey(team.Id, curTime)

	if err := a.WriteFile(buf, path); err != nil {
		return model.NewAppError("SetTeamIcon", "api.team.set_team_icon.write_file.app_error", nil, "", http.StatusInternalServerError)
	}

	if err := a.Srv.Store.Team().UpdateLastTeamIconUpdate(team.Id, curTime); err != nil {
		return model.NewAppError("SetTeamIcon", "api.team.team_icon.update.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	return nil
}

func (a *App) RemoveTeamIcon(teamId string) *model.AppError {
	_, err := a.GetTeam(teamId)
	if err != nil {
		return model.NewAppError("RemoveTeamIcon", "api.team.remove_team_icon.get_team.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	if err := a.Srv.Store.Team().UpdateLastTeamIconUpdate(teamId, 0); err != nil {
		return model.NewAppError("RemoveTeamIcon", "api.team.team_icon.update.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	return nil
}

func (a *App) AutocompletePublicTeams(name string) ([]*model.Team, *model.AppError) {
	return a.Srv.Store.Team().AutocompletePublic(name)
}

func (a *App) UpdateTeamMemberType(teamId string, userId string, newType string) (*model.TeamMember, *model.AppError) {
	var member *model.TeamMember
	var err *model.AppError
	if member, err = a.GetTeamMember(teamId, userId); err != nil {
		return nil, err
	}

	if member.Type == newType {
		return nil, model.NewAppError("UpdateTeamMemberType", "api.team.update_team_member_type.same_type.app_error", nil, "", http.StatusBadRequest)
	}

	teamMembersGetOptions := &model.TeamMembersGetOptions{
		ExcludeDeletedUsers: true,
		Type:                model.TEAM_MEMBER_TYPE_ADMIN,
	}
	// team adminの数が1人以下ならadmin → normalに変更不能にする
	members, err := a.GetTeamMembers(teamId, 0, 10, teamMembersGetOptions)
	if member.Type == model.TEAM_MEMBER_TYPE_ADMIN && newType != model.TEAM_MEMBER_TYPE_ADMIN && len(members) <= 1 {
		return nil, model.NewAppError("UpdateTeamMemberType", "api.team.update_team_member_type.missing_admin.app_error", nil, "", http.StatusBadRequest)
	}

	member.Type = newType
	member, err = a.Srv.Store.Team().UpdateMember(member)
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (a *App) GetAnalytics(analyticKey string, teamId string) (model.Analytics, *model.AppError) {
	if analyticKey == "answered_rate" {
		var rows model.Analytics = make([]*model.Analytic, 1)
		rows[0] = &model.Analytic{Name: "answered_rate", Value: 0}

		rate, err := a.Srv.Store.Post().GetAnsweredRate(teamId)
		if err != nil {
			return nil, err
		}
		rows[0].Value = rate

		return rows, nil
	} else if analyticKey == "post_counts_day" {
		return a.Srv.Store.Post().AnalyticsPostCounts(teamId)
	} else if analyticKey == "active_author_counts_day" {
		return a.Srv.Store.Post().AnalyticsActiveAuthorCounts(teamId)
	} else if analyticKey == "post_views_counts_day" {
		return a.Srv.Store.PostViewsHistory().AnalyticsPostViewsHistoryCounts(teamId)
	} else if analyticKey == "up_vote_counts_day" {
		return a.Srv.Store.Vote().AnalyticsVoteCounts(teamId, model.VOTE_TYPE_UP_VOTE)
	}

	return nil, nil
}
