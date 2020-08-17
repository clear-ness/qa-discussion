package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitTag() {
	api.BaseRoutes.Tags.Handle("/autocomplete", api.ApiHandler(autocompleteTags)).Methods("GET")
	api.BaseRoutes.TagsForTeam.Handle("/autocomplete", api.ApiSessionRequired(autocompleteTagsForTeam)).Methods("GET")

	api.BaseRoutes.Tags.Handle("", api.ApiHandler(getTags)).Methods("GET")
	api.BaseRoutes.TagsForTeam.Handle("", api.ApiSessionRequired(getTagsForTeam)).Methods("GET")

	// TODO: /related
	// Including multiple tags in {tags} is equivalent to asking for "tags related to tag #1 and tag #2" not "tags related to tag #1 or tag #2".
	// count on tag objects returned is the number of question with that tag that also share all those in {tags}.
	// {tags} can contain up to 4 individual tags per request.
	//
	// → ElasticSearch のmore like this queryで実装？
	// https://meta.stackexchange.com/questions/20473/how-are-related-questions-selected

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
