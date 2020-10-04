package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitWebhook() {
	api.BaseRoutes.Hooks.Handle("", api.ApiSessionRequired(createHook)).Methods("POST")
	api.BaseRoutes.Hooks.Handle("", api.ApiSessionRequired(getHooks)).Methods("GET")

	api.BaseRoutes.Hook.Handle("", api.ApiSessionRequired(getHook)).Methods("GET")
	api.BaseRoutes.Hook.Handle("", api.ApiSessionRequired(updateHook)).Methods("PUT")
	api.BaseRoutes.Hook.Handle("", api.ApiSessionRequired(deleteHook)).Methods("DELETE")
	api.BaseRoutes.Hook.Handle("/regen_token", api.ApiSessionRequired(regenHookToken)).Methods("POST")

	api.BaseRoutes.Hooks.Handle("/history", api.ApiSessionRequired(getHooksHistory)).Methods("GET")
}

func createHook(c *Context, w http.ResponseWriter, r *http.Request) {
	hook := model.WebhookFromJson(r.Body)
	if hook == nil {
		c.SetInvalidParam("webhook")
		return
	}

	hook.UserId = c.App.Session.UserId

	if !c.App.SessionHasPermissionToTeam(c.App.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	rhook, err := c.App.CreateWebhook(hook)
	if err != nil {
		c.Err = err
		return
	}
	// TODO: SanitizeInput

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(rhook.ToJson()))
}

func getHooks(c *Context, w http.ResponseWriter, r *http.Request) {
	teamId := r.URL.Query().Get("team_id")
	if len(teamId) <= 0 {
		c.SetInvalidParam("team_id")
	}

	var hooks []*model.Webhook
	var err *model.AppError

	userId := c.App.Session.UserId
	if !c.App.SessionHasPermissionToTeam(c.App.Session, teamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	// team adminは複数存在しうるが、
	// team admin同士なら互いにteamに対して設定したwebhooksを見れる仕様。
	if c.App.SessionHasPermissionToTeam(c.App.Session, teamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		userId = ""
	}

	hooks, err = c.App.GetWebhooksForTeamPage(teamId, userId, c.Params.Page, c.Params.PerPage)

	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.WebhookListToJson(hooks)))
}

func getHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hook, err := c.App.GetWebhook(c.Params.HookId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.App.Session.UserId != hook.UserId && !c.App.SessionHasPermissionToTeam(c.App.Session, hook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	w.Write([]byte(hook.ToJson()))
}

func updateHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	updatedHook := model.WebhookFromJson(r.Body)
	if updatedHook == nil {
		c.SetInvalidParam("webhook")
		return
	}

	if updatedHook.Id != c.Params.HookId {
		c.SetInvalidParam("hook_id")
		return
	}

	oldHook, err := c.App.GetWebhook(c.Params.HookId)
	if err != nil {
		c.Err = err
		return
	}

	if updatedHook.TeamId == "" {
		updatedHook.TeamId = oldHook.TeamId
	}

	if updatedHook.TeamId != oldHook.TeamId {
		c.Err = model.NewAppError("updateHook", "api.webhook.team_mismatch.app_error", nil, "user_id="+c.App.Session.UserId, http.StatusBadRequest)
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, updatedHook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.App.Session.UserId != oldHook.UserId && !c.App.SessionHasPermissionToTeam(c.App.Session, updatedHook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	updatedHook.UserId = c.App.Session.UserId

	rhook, err := c.App.UpdateWebhook(oldHook, updatedHook)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(rhook.ToJson()))
}

func deleteHook(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hook, err := c.App.GetWebhook(c.Params.HookId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.App.Session.UserId != hook.UserId && !c.App.SessionHasPermissionToTeam(c.App.Session, hook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	if err := c.App.DeleteWebhook(hook.Id); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func regenHookToken(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireHookId()
	if c.Err != nil {
		return
	}

	hook, err := c.App.GetWebhook(c.Params.HookId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, hook.TeamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	if c.App.Session.UserId != hook.UserId && !c.App.SessionHasPermissionToTeam(c.App.Session, hook.TeamId, model.PERMISSION_MANAGE_OTHERS_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_OTHERS_WEBHOOKS)
		return
	}

	rhook, err := c.App.RegenWebhookToken(hook)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(rhook.ToJson()))
}

func getHooksHistory(c *Context, w http.ResponseWriter, r *http.Request) {
	teamId := r.URL.Query().Get("team_id")
	if len(teamId) <= 0 {
		c.SetInvalidParam("team_id")
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, teamId, model.PERMISSION_MANAGE_WEBHOOKS) {
		c.SetPermissionError(model.PERMISSION_MANAGE_WEBHOOKS)
		return
	}

	histories, err := c.App.GetWebhooksHistoriesPage(teamId, c.Params.Page, c.Params.PerPage)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.WebhooksHistoryListToJson(histories)))
}
