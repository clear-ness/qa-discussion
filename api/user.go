package api

import (
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitUser() {
	api.BaseRoutes.Users.Handle("", api.ApiHandler(createUser)).Methods("POST")
	api.BaseRoutes.Users.Handle("/email/verify", api.ApiHandler(verifyUserEmail)).Methods("POST")
	api.BaseRoutes.Users.Handle("/email/verify/send", api.ApiHandler(sendVerificationEmail)).Methods("POST")
	api.BaseRoutes.Users.Handle("/login", api.ApiHandler(login)).Methods("POST")
	api.BaseRoutes.Users.Handle("/logout", api.ApiHandler(logout)).Methods("POST")
	api.BaseRoutes.Users.Handle("", api.ApiHandler(getUsers)).Methods("GET")
	api.BaseRoutes.Users.Handle("/ids", api.ApiHandler(getUsersByIds)).Methods("POST")

	api.BaseRoutes.User.Handle("", api.ApiHandler(getUser)).Methods("GET")
	api.BaseRoutes.UserForTeam.Handle("", api.ApiSessionRequired(getTeamUser)).Methods("GET")

	api.BaseRoutes.User.Handle("", api.ApiSessionRequired(updateUser)).Methods("PUT")
	api.BaseRoutes.User.Handle("", api.ApiSessionRequired(deleteUser)).Methods("DELETE")

	// Moderators can place users in timed suspension,
	// account will be unable to vote, ask, answer, or comment.
	// At the end of this timed suspension period, your account will resume as normal.
	api.BaseRoutes.User.Handle("/suspend", api.ApiSessionRequired(suspendUser)).Methods("POST")
	api.BaseRoutes.User.Handle("/type", api.ApiSessionRequired(updateUserType)).Methods("PUT")
	api.BaseRoutes.User.Handle("/password", api.ApiSessionRequired(updatePassword)).Methods("PUT")

	api.BaseRoutes.Users.Handle("/password/reset", api.ApiHandler(resetPassword)).Methods("POST")
	api.BaseRoutes.Users.Handle("/password/reset/send", api.ApiHandler(sendPasswordReset)).Methods("POST")

	api.BaseRoutes.InboxMessagesForUser.Handle("", api.ApiSessionRequired(getInboxMessagesForUser)).Methods("GET")
	api.BaseRoutes.InboxMessagesForUser.Handle("/unread_count", api.ApiSessionRequired(getInboxMessagesUnreadCountForUser)).Methods("GET")
	api.BaseRoutes.InboxMessageForUser.Handle("/set_read", api.ApiSessionRequired(setInboxMessageRead)).Methods("POST")

	api.BaseRoutes.UserPointHistoryForUser.Handle("", api.ApiSessionRequired(getUserPointHistoryForUser)).Methods("GET")

	api.BaseRoutes.VotesForUser.Handle("", api.ApiSessionRequired(getVotesForUser)).Methods("GET")

	api.BaseRoutes.User.Handle("/image", api.ApiSessionRequired(setProfileImage)).Methods("POST")

	// TODO: search users

}

func createUser(c *Context, w http.ResponseWriter, r *http.Request) {
	user := model.UserFromJson(r.Body)
	if user == nil {
		c.SetInvalidParam("user")
		return
	}

	user.SanitizeInput()

	// 以下、いずれの場合もユーザー新規作成だけでなく、事前にチームにも参加させておく
	//
	// inviteUsersToTeamでemail link招待された場合
	tokenId := r.URL.Query().Get("t")
	// チームのlink共有されて 作成する場合 (teamのlinkを知ってさえいればいい)
	inviteId := r.URL.Query().Get("iid")

	var ruser *model.User
	var err *model.AppError

	if len(tokenId) > 0 {
		var token *model.Token
		token, err = c.App.Srv.Store.Token().GetByToken(tokenId)
		if err != nil {
			c.Err = model.NewAppError("CreateUserWithToken", "api.user.create_user.signup_link_invalid.app_error", nil, err.Error(), http.StatusBadRequest)
			return
		}

		ruser, err = c.App.CreateUserWithToken(user, token)
	} else if len(inviteId) > 0 {
		ruser, err = c.App.CreateUserWithInviteId(user, inviteId)
	} else {
		ruser, err = c.App.CreateUserFromSignup(user)
	}

	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(ruser.ToJson()))
}

func verifyUserEmail(c *Context, w http.ResponseWriter, r *http.Request) {
	props := model.MapFromJson(r.Body)

	token := props["token"]
	if len(token) != model.TOKEN_SIZE {
		c.SetInvalidParam("token")
		return
	}

	if err := c.App.VerifyEmailFromToken(token); err != nil {
		c.Err = model.NewAppError("verifyUserEmail", "api.user.verify_email.bad_link.app_error", nil, err.Error(), http.StatusBadRequest)
		return
	}

	ReturnStatusOK(w)
}

func sendVerificationEmail(c *Context, w http.ResponseWriter, r *http.Request) {
	props := model.MapFromJson(r.Body)

	email := props["email"]
	if len(email) == 0 {
		c.SetInvalidParam("email")
		return
	}

	user, err := c.App.GetUserForLogin(email)
	if err != nil {
		ReturnStatusOK(w)
		return
	}

	if err = c.App.SendEmailVerification(user, user.Email); err != nil {
		mlog.Error(err.Error())
		ReturnStatusOK(w)
		return
	}

	ReturnStatusOK(w)
}

func login(c *Context, w http.ResponseWriter, r *http.Request) {
	props := model.MapFromJson(r.Body)

	loginId := props["login_id"]
	password := props["password"]

	user, err := c.App.AuthenticateUserForLogin(loginId, password)
	if err != nil {
		c.Err = err
		return
	}

	err = c.App.DoLogin(w, r, user)
	if err != nil {
		c.Err = err
		return
	}

	if r.Header.Get(model.HEADER_REQUESTED_WITH) == model.HEADER_REQUESTED_WITH_XML {
		c.App.AttachSessionCookies(w, r)
	}

	user.Sanitize(map[string]bool{})

	w.Write([]byte(user.ToJson()))
}

func logout(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RemoveSessionCookie(w, r)
	if c.App.Session.Id != "" {
		if err := c.App.RevokeSessionById(c.App.Session.Id); err != nil {
			c.Err = err
			return
		}
	}

	ReturnStatusOK(w)
}

func getUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()

	user, err := c.App.GetUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	if user == nil {
		c.SetInvalidParam("user_id")
		return
	}

	questionCount, err := c.App.GetPostCount(user.Id, model.POST_TYPE_QUESTION, "")
	if err != nil {
		mlog.Error(err.Error())
	} else {
		user.QuestionCount = questionCount
	}

	answerCount, err := c.App.GetPostCount(user.Id, model.POST_TYPE_ANSWER, "")
	if err != nil {
		mlog.Error(err.Error())
	} else {
		user.AnswerCount = answerCount
	}

	options := map[string]bool{}
	options["email"] = false
	user.SanitizeProfile(options)

	w.Write([]byte(user.ToJson()))
}

func getTeamUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	user, err := c.App.GetUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	if user == nil {
		c.SetInvalidParam("user_id")
		return
	}

	member, err := c.App.GetTeamMember(c.Params.TeamId, user.Id)
	if err != nil {
		c.Err = err
		return
	}

	questionCount, err := c.App.GetPostCount(user.Id, model.POST_TYPE_QUESTION, member.TeamId)
	if err != nil {
		mlog.Error(err.Error())
	} else {
		user.QuestionCount = questionCount
	}

	answerCount, err := c.App.GetPostCount(user.Id, model.POST_TYPE_ANSWER, member.TeamId)
	if err != nil {
		mlog.Error(err.Error())
	} else {
		user.AnswerCount = answerCount
	}

	// TODO: memberを返す？getUsersも同様に考慮。
	user.Points = member.Points

	options := map[string]bool{}
	options["email"] = false
	user.SanitizeProfile(options)

	w.Write([]byte(user.ToJson()))
}

func getUsers(c *Context, w http.ResponseWriter, r *http.Request) {
	if c.Params.FromDate != 0 && c.Params.ToDate != 0 && c.Params.FromDate > c.Params.ToDate {
		c.SetInvalidUrlParam("from_to_dates")
		return
	}

	options := &model.GetUsersOptions{FromDate: c.Params.FromDate, ToDate: c.Params.ToDate}
	options.Page = c.Params.Page
	options.PerPage = c.Params.PerPage

	sort := c.Params.SortType
	if len(sort) > 0 && sort != model.POST_SORT_TYPE_CREATION && sort != model.POST_SORT_TYPE_VOTES {
		c.SetInvalidUrlParam("sort")
		return
	} else {
		options.SortType = sort
	}

	if c.Params.Min != nil && c.Params.Max != nil && *c.Params.Min > *c.Params.Max {
		c.SetInvalidUrlParam("min_max")
		return
	}
	options.Min = c.Params.Min
	options.Max = c.Params.Max

	options.Username = c.Params.UserName

	users, err := c.App.GetUsersByDates(options)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.UserListToJson(users)))
}

func getUsersByIds(c *Context, w http.ResponseWriter, r *http.Request) {
	userIds := model.ArrayFromJson(r.Body)

	if len(userIds) == 0 {
		c.SetInvalidParam("user_ids")
		return
	}

	users, err := c.App.GetUsers(userIds)
	if err != nil {
		c.Err = err
		return
	}

	options := map[string]bool{}
	options["email"] = false
	for _, user := range users {
		user.SanitizeProfile(options)
	}

	w.Write([]byte(model.UserListToJson(users)))
}

func getInboxMessagesForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if c.App.Session.UserId != c.Params.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_READ_OTHERS_INBOX_MESSAGES) {
			c.SetPermissionError(model.PERMISSION_READ_OTHERS_INBOX_MESSAGES)
			return
		}
	}

	teamId := ""
	if len(c.Params.TeamId) > 0 {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
			c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
			return
		}

		teamId = c.Params.TeamId
	}

	curTime := model.GetMillis()

	messages, err := c.App.GetInboxMessagesForUserToDate(curTime, c.Params.UserId, c.Params.Page, c.Params.PerPage, teamId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.InboxMessageListToJson(messages)))
}

func getUserPointHistoryForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if c.App.Session.UserId != c.Params.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_READ_OTHERS_USER_POINT_HISTORY) {
			c.SetPermissionError(model.PERMISSION_READ_OTHERS_USER_POINT_HISTORY)
			return
		}
	}

	teamId := ""
	if len(c.Params.TeamId) > 0 {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
			c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
			return
		}

		teamId = c.Params.TeamId
	}

	toDate := model.GetMillis()
	if c.Params.ToDate != 0 {
		toDate = c.Params.ToDate
	}

	history, err := c.App.GetUserPointHistoryForUser(toDate, c.Params.UserId, c.Params.Page, c.Params.PerPage, teamId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.UserPointHistoryToJson(history)))
}

func getInboxMessagesUnreadCountForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if c.App.Session.UserId != c.Params.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_READ_OTHERS_INBOX_MESSAGES) {
			c.SetPermissionError(model.PERMISSION_READ_OTHERS_INBOX_MESSAGES)
			return
		}
	}

	teamId := ""
	if len(c.Params.TeamId) > 0 {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
			c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
			return
		}

		teamId = c.Params.TeamId
	}

	count, err := c.App.GetInboxMessagesUnreadCountForUser(c.Params.UserId, teamId)
	if err != nil {
		c.Err = err
		return
	}

	countStr := strconv.FormatInt(count, 10)
	w.Write([]byte(model.MapToJson(map[string]string{"user_id": c.Params.UserId, "inbox_messages_unread_count": countStr})))
}

func setInboxMessageRead(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId().RequireInboxMessageId()
	if c.Err != nil {
		return
	}

	if c.App.Session.UserId != c.Params.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_SET_READ_OTHERS_INBOX_MESSAGES) {
			c.SetPermissionError(model.PERMISSION_SET_READ_OTHERS_INBOX_MESSAGES)
			return
		}
	}

	message, err := c.App.GetSingleInboxMessage(c.Params.InboxMessageId)
	if err != nil {
		c.Err = err
		return
	}

	if message == nil {
		c.SetInvalidUrlParam("inbox_message_id")
		return
	}

	if message.UserId != c.Params.UserId {
		c.SetInvalidUrlParam("user_id")
		return
	}

	err = c.App.UpdateLastInboxMessageViewedForUser(message, c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func getVotesForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if c.App.Session.UserId != c.Params.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_READ_OTHERS_VOTES) {
			c.SetPermissionError(model.PERMISSION_READ_OTHERS_VOTES)
			return
		}
	}

	teamId := ""
	if len(c.Params.TeamId) > 0 {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
			c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
			return
		}
		teamId = c.Params.TeamId
	}

	curTime := model.GetMillis()

	votes, totalCount, err := c.App.GetVotesForUser(curTime, c.Params.UserId, c.Params.Page, c.Params.PerPage, true, true, teamId)
	if err != nil {
		c.Err = err
		return
	}

	data := model.VotesWithCount{Votes: votes, TotalCount: totalCount}
	w.Write([]byte(data.ToJson()))
}

// TODO: suspendされてもteam内では動けるのが前提
func suspendUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId().RequireSuspendSpanType()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_SUSPEND_USER) {
		c.SetPermissionError(model.PERMISSION_SUSPEND_USER)
		return
	}

	user, err := c.App.GetUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	if user == nil {
		c.SetInvalidUrlParam("user_id")
		return
	}

	err = c.App.SuspendUser(user.Id, c.Params.SuspendSpanType, c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func updateUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	user := model.UserFromJson(r.Body)
	if user == nil {
		c.SetInvalidParam("user")
		return
	}

	if user.Id != c.Params.UserId {
		c.SetInvalidParam("user_id")
		return
	}

	if !c.App.SessionHasPermissionToUser(c.App.Session, user.Id) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	oldUser, err := c.App.GetUser(user.Id)
	if err != nil {
		c.Err = err
		return
	}

	if c.App.Session.IsOAuth {
		if oldUser.Email != user.Email {
			c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
			return
		}
	}

	// If eMail update is attempted by the currently logged in user, check if correct password was provided
	if user.Email != "" && oldUser.Email != user.Email && c.App.Session.UserId == c.Params.UserId {
		err = c.App.DoubleCheckPassword(oldUser, user.Password)
		if err != nil {
			c.SetInvalidParam("password")
			return
		}
	}

	ruser, err := c.App.UpdateUser(user)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(ruser.ToJson()))
}

func updateUserType(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId().RequireUserType()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_USER_TYPE) {
		c.SetPermissionError(model.PERMISSION_EDIT_USER_TYPE)
		return
	}

	if _, err := c.App.UpdateUserType(c.Params.UserId, c.Params.UserType); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func updatePassword(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	props := model.MapFromJson(r.Body)
	newPassword := props["new_password"]

	var err *model.AppError
	if c.Params.UserId == c.App.Session.UserId {
		currentPassword := props["current_password"]
		if len(currentPassword) <= 0 {
			c.SetInvalidParam("current_password")
			return
		}

		err = c.App.UpdatePasswordAsUser(c.Params.UserId, currentPassword, newPassword)
	} else if c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_OTHER_USERS_PASSWORD) {
		err = c.App.UpdatePasswordByUserIdSendEmail(c.Params.UserId, newPassword, "api.user.reset_password.method")
	} else {
		err = model.NewAppError("updatePassword", "api.user.update_password.context.app_error", nil, "", http.StatusForbidden)
	}

	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func resetPassword(c *Context, w http.ResponseWriter, r *http.Request) {
	props := model.MapFromJson(r.Body)

	token := props["token"]
	if len(token) != model.TOKEN_SIZE {
		c.SetInvalidParam("token")
		return
	}

	newPassword := props["new_password"]

	if err := c.App.ResetPasswordFromToken(token, newPassword); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func sendPasswordReset(c *Context, w http.ResponseWriter, r *http.Request) {
	props := model.MapFromJson(r.Body)

	email := props["email"]
	email = strings.ToLower(email)
	if len(email) == 0 {
		c.SetInvalidParam("email")
		return
	}

	err := c.App.SendPasswordReset(email, c.App.GetSiteURL())
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func deleteUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	user, err := c.App.GetUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	if user == nil {
		c.SetInvalidUrlParam("user_id")
		return
	}

	if c.App.Session.UserId == user.Id {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_DELETE_USER) {
			c.SetPermissionError(model.PERMISSION_DELETE_USER)
			return
		}
	} else {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_DELETE_OTHER_USERS) {
			c.SetPermissionError(model.PERMISSION_DELETE_OTHER_USERS)
			return
		}
	}

	if err := c.App.DeleteUser(user.Id, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func setProfileImage(c *Context, w http.ResponseWriter, r *http.Request) {
	defer io.Copy(ioutil.Discard, r.Body)

	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToUser(c.App.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	if r.ContentLength > *c.App.Config().FileSettings.MaxFileSize {
		c.Err = model.NewAppError("uploadProfileImage", "api.user.upload_profile_user.too_large.app_error", nil, "", http.StatusRequestEntityTooLarge)
		return
	}

	if err := r.ParseMultipartForm(*c.App.Config().FileSettings.MaxFileSize); err != nil {
		c.Err = model.NewAppError("uploadProfileImage", "api.user.upload_profile_user.parse.app_error", nil, err.Error(), http.StatusInternalServerError)
		return
	}

	m := r.MultipartForm
	imageArray, ok := m.File["image"]
	if !ok {
		c.Err = model.NewAppError("uploadProfileImage", "api.user.upload_profile_user.no_file.app_error", nil, "", http.StatusBadRequest)
		return
	}

	if len(imageArray) <= 0 {
		c.Err = model.NewAppError("uploadProfileImage", "api.user.upload_profile_user.array.app_error", nil, "", http.StatusBadRequest)
		return
	}

	imageData := imageArray[0]
	if link, err := c.App.SetProfileImage(c.Params.UserId, imageData); err != nil {
		c.Err = err
		return
	} else {
		w.Write([]byte(model.MapToJson(map[string]string{"user_id": c.Params.UserId, "profile_image_link": link})))
	}
}
