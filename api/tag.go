package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitTag() {
	api.BaseRoutes.Tags.Handle("/autocomplete", api.ApiHandler(autocompleteTags)).Methods("GET")
	api.BaseRoutes.Tags.Handle("", api.ApiHandler(getTags)).Methods("GET")
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

func getTags(c *Context, w http.ResponseWriter, r *http.Request) {
	if c.Params.FromDate != 0 && c.Params.ToDate != 0 && c.Params.FromDate > c.Params.ToDate {
		c.SetInvalidUrlParam("from_to_dates")
		return
	}

	options := &model.GetTagsOptions{FromDate: c.Params.FromDate, ToDate: c.Params.ToDate}
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
