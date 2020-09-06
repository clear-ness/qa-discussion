package web

import (
	"net/http"
	"strings"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils"
)

type Context struct {
	App           *app.App
	Log           *mlog.Logger
	Params        *Params
	Err           *model.AppError
	siteURLHeader string
}

func (c *Context) SetInvalidParam(parameter string) {
	c.Err = NewInvalidParamError(parameter)
}

func NewInvalidParamError(parameter string) *model.AppError {
	err := model.NewAppError("Context", "api.context.invalid_body_param.app_error", map[string]interface{}{"Name": parameter}, "", http.StatusBadRequest)
	return err
}

func (c *Context) SetInvalidUrlParam(parameter string) {
	c.Err = NewInvalidUrlParamError(parameter)
}

func NewInvalidUrlParamError(parameter string) *model.AppError {
	err := model.NewAppError("Context", "api.context.invalid_url_param.app_error", map[string]interface{}{"Name": parameter}, "", http.StatusBadRequest)
	return err
}

func (c *Context) RemoveSessionCookie(w http.ResponseWriter, r *http.Request) {
	subpath, _ := utils.GetSubpathFromConfig(c.App.Config())

	cookie := &http.Cookie{
		Name:     model.SESSION_COOKIE_TOKEN,
		Value:    "",
		Path:     subpath,
		MaxAge:   -1,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
}

func (c *Context) SetPermissionError(permission *model.Permission) {
	c.Err = c.App.MakePermissionError(permission)
}

func (c *Context) SetSiteURLHeader(url string) {
	c.siteURLHeader = strings.TrimRight(url, "/")
}

func (c *Context) SessionRequired() {
	if len(c.App.Session.UserId) == 0 {
		c.Err = model.NewAppError("", "api.context.session_expired.app_error", nil, "UserRequired", http.StatusUnauthorized)
		return
	}
}

func (c *Context) RequirePostId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.PostId) != 26 {
		c.SetInvalidUrlParam("post_id")
	}

	return c
}

func (c *Context) RequireUserId() *Context {
	if c.Err != nil {
		return c
	}

	if c.Params.UserId == model.ME {
		c.Params.UserId = c.App.Session.UserId
	}

	if len(c.Params.UserId) != 26 {
		c.SetInvalidUrlParam("user_id")
	}

	return c
}

func (c *Context) RequireBestId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.BestId) != 26 {
		c.SetInvalidUrlParam("best_id")
	}

	return c
}

func (c *Context) RequireFromDate() *Context {
	if c.Err != nil {
		return c
	}

	if c.Params.FromDate == 0 {
		c.SetInvalidUrlParam("from_date")
	}

	return c
}

func (c *Context) RequireToDate() *Context {
	if c.Err != nil {
		return c
	}

	if c.Params.ToDate == 0 {
		c.SetInvalidUrlParam("to_date")
	}

	return c
}

func (c *Context) RequireInboxMessageId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.InboxMessageId) != 26 {
		c.SetInvalidUrlParam("inbox_message_id")
	}

	return c
}

func (c *Context) RequireSuspendSpanType() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.SuspendSpanType) == 0 {
		c.SetInvalidUrlParam("suspend_span_type")
	}

	return c
}

func (c *Context) RequireUserType() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.UserType) == 0 {
		c.SetInvalidUrlParam("user_type")
	}

	return c
}

func (c *Context) RequireFilename() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.Filename) == 0 {
		c.SetInvalidUrlParam("filename")
	}

	return c
}

func (c *Context) RequireFileId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.FileId) != 26 {
		c.SetInvalidUrlParam("file_id")
	}

	return c
}

func (c *Context) RequireTeamId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.TeamId) != 26 {
		c.SetInvalidUrlParam("team_id")
	}

	return c
}

func (c *Context) RequireGroupId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.GroupId) != 26 {
		c.SetInvalidUrlParam("group_id")
	}

	return c
}

func (c *Context) RequireCollectionId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.CollectionId) != 26 {
		c.SetInvalidUrlParam("collection_id")
	}

	return c
}

func (c *Context) RequireHotPostsInterval() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.HotPostsInterval) == 0 {
		c.SetInvalidUrlParam("hot_posts_interval")
	}

	return c
}

func (c *Context) RequireRevisionId() *Context {
	if c.Err != nil {
		return c
	}

	if !model.IsValidNumberString(c.Params.RevisionId) {
		c.SetInvalidUrlParam("revision_id")
	}

	return c
}

func (c *Context) RequireTopInterval() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.TopUsersOrPostsInterval) == 0 {
		c.SetInvalidUrlParam("top_interval")
	}

	return c
}

func (c *Context) RequireHookId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.HookId) != 26 {
		c.SetInvalidUrlParam("hook_id")
	}

	return c
}

func (c *Context) RequireAppId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.AppId) != 26 {
		c.SetInvalidUrlParam("app_id")
	}
	return c
}
