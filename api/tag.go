package api

import (
	"net/http"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitTag() {
	api.BaseRoutes.Tags.Handle("/autocomplete", api.ApiHandler(autocompleteTags)).Methods("GET")
	api.BaseRoutes.TagsForTeam.Handle("/autocomplete", api.ApiSessionRequired(autocompleteTagsForTeam)).Methods("GET")

	api.BaseRoutes.Tags.Handle("", api.ApiHandler(getTags)).Methods("GET")
	api.BaseRoutes.TagsForTeam.Handle("", api.ApiSessionRequired(getTagsForTeam)).Methods("GET")

	api.BaseRoutes.Tags.Handle("/review", api.ApiSessionRequired(createReviewTags)).Methods("POST")
	api.BaseRoutes.Tags.Handle("/review", api.ApiSessionRequired(autocompleteReviewTags)).Methods("GET")

	api.BaseRoutes.Tags.Handle("/top_askers", api.ApiHandler(topAskersForTag)).Methods("POST")
	api.BaseRoutes.Tags.Handle("/top_answerers", api.ApiHandler(topAnswerersForTag)).Methods("POST")
	api.BaseRoutes.Tags.Handle("/top_answers", api.ApiHandler(topAnswersForTag)).Methods("POST")
}

func autocompleteReviewTags(c *Context, w http.ResponseWriter, r *http.Request) {
	user, err := c.App.GetUser(c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_REVIEW_VOTES) && user.Points < model.MIN_USER_POINT_FOR_VOTE_REVIEW {
		c.Err = model.NewAppError("autocompleteReviewTags", "api.tag.autocomplete_review_tags.low_points.app_error", nil, "", http.StatusBadRequest)
		return
	}

	options := &model.GetTagsOptions{Type: model.TAG_TYPE_REVIEW, SortType: model.POST_SORT_TYPE_NAME, Page: 0, PerPage: model.AUTOCOMPLETE_TAGS_LIMIT}

	tagName := c.Params.TagName
	if len(tagName) > 0 {
		options.InName = tagName
	} else {
		ReturnStatusOK(w)
		return
	}

	tags, err := c.App.GetTags(options)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(tags.ToJson()))
}

func autocompleteTags(c *Context, w http.ResponseWriter, r *http.Request) {
	options := &model.GetTagsOptions{SortType: model.POST_SORT_TYPE_NAME, Page: 0, PerPage: model.AUTOCOMPLETE_TAGS_LIMIT}

	tagName := c.Params.TagName
	if len(tagName) > 0 {
		options.InName = tagName
	} else {
		ReturnStatusOK(w)
		return
	}

	tags, err := c.App.GetTags(options)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(tags.ToJson()))
}

func autocompleteTagsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetTagsOptions{SortType: model.POST_SORT_TYPE_NAME, Page: 0, PerPage: model.AUTOCOMPLETE_TAGS_LIMIT, TeamId: c.Params.TeamId}

	tagName := c.Params.TagName
	if len(tagName) > 0 {
		options.InName = tagName
	} else {
		ReturnStatusOK(w)
		return
	}

	tags, err := c.App.GetTags(options)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(tags.ToJson()))
}

func getTagsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetTagsOptions{TeamId: c.Params.TeamId}
	getTagsWithOptions(c, w, r, options)
}

func getTags(c *Context, w http.ResponseWriter, r *http.Request) {
	options := &model.GetTagsOptions{}
	getTagsWithOptions(c, w, r, options)
}

func getTagsWithOptions(c *Context, w http.ResponseWriter, r *http.Request, options *model.GetTagsOptions) {
	if c.Params.FromDate != 0 && c.Params.ToDate != 0 && c.Params.FromDate > c.Params.ToDate {
		c.SetInvalidUrlParam("from_to_dates")
		return
	}

	options.FromDate = c.Params.FromDate
	options.ToDate = c.Params.ToDate
	options.Page = c.Params.Page
	options.PerPage = c.Params.PerPage

	sort := c.Params.SortType
	if len(sort) > 0 && sort != model.POST_SORT_TYPE_NAME && sort != model.POST_SORT_TYPE_POPULAR {
		c.SetInvalidUrlParam("sort")
		return
	} else {
		options.SortType = sort
	}

	if sort == model.POST_SORT_TYPE_POPULAR {
		if c.Params.Min != nil && c.Params.Max != nil && *c.Params.Min > *c.Params.Max {
			c.SetInvalidUrlParam("min_max")
			return
		}

		if c.Params.Min != nil {
			options.Min = c.Params.Min
		}

		if c.Params.Max != nil {
			options.Max = c.Params.Max
		}
	}

	tagName := c.Params.TagName
	if len(tagName) > 0 {
		options.InName = tagName
	}

	tags, err := c.App.GetTags(options)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(tags.ToJson()))
}

func createReviewTags(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_REVIEW_TAGS) {
		c.SetPermissionError(model.PERMISSION_CREATE_REVIEW_TAGS)
		return
	}

	m := model.MapFromJson(r.Body)
	tagContents := m["tags"]

	tagContents = model.ParseTags(tagContents)
	tags := strings.Fields(tagContents)

	if len(tags) <= 0 {
		c.SetInvalidParam("tags")
		return
	}

	curTime := model.GetMillis()

	err := c.App.CreateTags(tags, curTime, "", model.TAG_TYPE_REVIEW)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	ReturnStatusOK(w)
}

func topAskersForTag(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTopInterval()
	if c.Err != nil {
		return
	}

	m := model.MapFromJson(r.Body)
	tagContents := m["tags"]
	tagContents = model.ParseTags(tagContents)
	tags := strings.Fields(tagContents)

	if len(tags) <= 0 {
		c.SetInvalidParam("tags")
		return
	}

	var results []*model.TopUserByTagResult
	var err *model.AppError
	results, err = c.App.TopAskersForTag(c.Params.TopUsersOrPostsInterval, "", tags[0])
	if err != nil {
		c.Err = err
		return
	}

	data := model.TopUserByTagResultsWithCount{TopUserByTagResults: results, TotalCount: int64(len(results))}
	w.Write([]byte(data.ToJson()))
}

func topAnswerersForTag(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTopInterval()
	if c.Err != nil {
		return
	}

	m := model.MapFromJson(r.Body)
	tagContents := m["tags"]
	tagContents = model.ParseTags(tagContents)
	tags := strings.Fields(tagContents)

	if len(tags) <= 0 {
		c.SetInvalidParam("tags")
		return
	}

	var results []*model.TopUserByTagResult
	var err *model.AppError
	results, err = c.App.TopAnswerersForTag(c.Params.TopUsersOrPostsInterval, "", tags[0])
	if err != nil {
		c.Err = err
		return
	}

	data := model.TopUserByTagResultsWithCount{TopUserByTagResults: results, TotalCount: int64(len(results))}
	w.Write([]byte(data.ToJson()))
}

func topAnswersForTag(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTopInterval()
	if c.Err != nil {
		return
	}

	m := model.MapFromJson(r.Body)
	tagContents := m["tags"]
	tagContents = model.ParseTags(tagContents)
	tags := strings.Fields(tagContents)

	if len(tags) <= 0 {
		c.SetInvalidParam("tags")
		return
	}

	var results []*model.TopPostByTagResult
	var err *model.AppError
	results, err = c.App.TopAnswersForTag(c.Params.TopUsersOrPostsInterval, "", tags[0])
	if err != nil {
		c.Err = err
		return
	}

	data := model.TopPostByTagResultsWithCount{TopPostByTagResults: results, TotalCount: int64(len(results))}
	w.Write([]byte(data.ToJson()))
}
