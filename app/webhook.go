package app

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/httpservice"
	"github.com/clear-ness/qa-discussion/utils"
)

const (
	MaxResponseSize = 1024 * 1024
)

func (a *App) handleWebhookEvents(post *model.Post, team *model.Team, user *model.User) *model.AppError {
	hooks, err := a.Srv.Store.Webhook().GetByTeam(team.Id, "", -1, -1)
	if err != nil {
		return err
	}

	if len(hooks) == 0 {
		return nil
	}

	relevantHooks := []*model.Webhook{}
	for _, hook := range hooks {
		trigger := false
		switch post.Type {
		case model.POST_TYPE_QUESTION:
			trigger = hook.QuestionEvents
		case model.POST_TYPE_ANSWER:
			trigger = hook.AnswerEvents
		case model.POST_TYPE_COMMENT:
			trigger = hook.CommentEvents
		default:
			trigger = false
		}

		if trigger {
			relevantHooks = append(relevantHooks, hook)
		}
	}

	for _, hook := range relevantHooks {
		payload := &model.WebhookPayload{
			Token:     hook.Token,
			TeamId:    hook.TeamId,
			TeamName:  team.Name,
			Timestamp: post.CreateAt,
			UserId:    post.UserId,
			UserName:  user.Username,
			PostId:    post.Id,
			Content:   post.Content,
			Title:     post.Title,
			PostType:  post.Type,
		}

		a.Srv.Go(func(hook *model.Webhook) func() {
			return func() {
				a.TriggerWebhook(payload, hook, post)
			}
		}(hook))
	}

	return nil
}

func (a *App) TriggerWebhook(payload *model.WebhookPayload, hook *model.Webhook, post *model.Post) {
	var body io.Reader
	var contentType string
	if hook.ContentType == "application/json" {
		body = strings.NewReader(payload.ToJSON())
		contentType = "application/json"
	} else {
		body = strings.NewReader(payload.ToFormValues())
		contentType = "application/x-www-form-urlencoded"
	}

	for i := range hook.URLs {
		url := hook.URLs[i]

		a.Srv.Go(func() {
			_, err := a.doWebhookRequest(url, body, contentType, hook, payload)
			if err != nil {
				return
			}
		})
	}
}

func (a *App) doWebhookRequest(url string, body io.Reader, contentType string, hook *model.Webhook, payload *model.WebhookPayload) (*model.WebhookResponse, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := httpservice.MakeClient(false).Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
	}
	reqBody := string(bodyBytes)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
	}
	resBody := string(bodyBytes)

	history := &model.WebhooksHistory{
		Id:             model.NewId(),
		WebhookId:      hook.Id,
		PostId:         payload.PostId,
		TeamId:         payload.TeamId,
		WebhookName:    hook.Name,
		URL:            url,
		ContentType:    contentType,
		RequestBody:    reqBody,
		ResponseBody:   resBody,
		ResponseStatus: resp.StatusCode,
		CreateAt:       model.GetMillis(),
	}
	if err := a.Srv.Store.WebhooksHistory().LogWebhookEvent(history); err != nil {
		mlog.Error("Failed to log webhook history", mlog.Err(err))
	}

	return model.WebhookResponseFromJson(io.LimitReader(resp.Body, MaxResponseSize))
}

func (a *App) CreateWebhook(hook *model.Webhook) (*model.Webhook, *model.AppError) {
	if allHooks, err := a.Srv.Store.Webhook().GetByTeam(hook.TeamId, "", -1, -1); err != nil {
		return nil, err
	} else {
		for _, existingHook := range allHooks {
			urlIntersect := utils.StringArrayIntersection(existingHook.URLs, hook.URLs)
			if len(urlIntersect) != 0 {
				return nil, model.NewAppError("CreateWebhook", "api.webhook.create_webhook.intersect.app_error", nil, "", http.StatusInternalServerError)
			}
		}
	}

	webhook, err := a.Srv.Store.Webhook().Save(hook)
	if err != nil {
		return nil, err
	}

	return webhook, nil
}

func (a *App) GetWebhooksForTeamPage(teamId string, userId string, page, perPage int) ([]*model.Webhook, *model.AppError) {
	return a.Srv.Store.Webhook().GetByTeam(teamId, userId, page*perPage, perPage)
}

func (a *App) GetWebhook(hookId string) (*model.Webhook, *model.AppError) {
	return a.Srv.Store.Webhook().Get(hookId)
}

func (a *App) UpdateWebhook(oldHook, updatedHook *model.Webhook) (*model.Webhook, *model.AppError) {
	allHooks, err := a.Srv.Store.Webhook().GetByTeam(oldHook.TeamId, "", -1, -1)
	if err != nil {
		return nil, err
	}

	for _, existingHook := range allHooks {
		urlIntersect := utils.StringArrayIntersection(existingHook.URLs, updatedHook.URLs)
		if len(urlIntersect) != 0 && existingHook.Id != updatedHook.Id {
			return nil, model.NewAppError("UpdateWebhook", "api.webhook.update.intersect.app_error", nil, "", http.StatusBadRequest)
		}
	}

	updatedHook.Id = oldHook.Id
	updatedHook.Token = oldHook.Token
	updatedHook.UserId = oldHook.UserId
	updatedHook.TeamId = oldHook.TeamId
	updatedHook.CreateAt = oldHook.CreateAt
	updatedHook.UpdateAt = model.GetMillis()
	updatedHook.DeleteAt = oldHook.DeleteAt

	return a.Srv.Store.Webhook().Update(updatedHook)
}

func (a *App) DeleteWebhook(hookId string) *model.AppError {
	return a.Srv.Store.Webhook().Delete(hookId, model.GetMillis())
}

func (a *App) RegenWebhookToken(hook *model.Webhook) (*model.Webhook, *model.AppError) {
	hook.Token = model.NewId()
	return a.Srv.Store.Webhook().Update(hook)
}
