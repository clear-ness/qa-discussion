package api

import (
	"net/http"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitReview() {
	api.BaseRoutes.ReviewsForPost.Handle("", api.ApiSessionRequired(createReviewVote)).Methods("POST")

	api.BaseRoutes.Reviews.Handle("", api.ApiSessionRequired(searchReviews)).Methods("POST")
	api.BaseRoutes.ReviewsForPost.Handle("", api.ApiSessionRequired(getReviewsForPost)).Methods("GET")
	api.BaseRoutes.ReviewsForUser.Handle("", api.ApiSessionRequired(getReviewsForUser)).Methods("GET")

	api.BaseRoutes.ReviewsForPost.Handle("/reject", api.ApiSessionRequired(rejectReviewsForPost)).Methods("POST")

	api.BaseRoutes.ReviewsForPost.Handle("/complete", api.ApiSessionRequired(completeReviewsForPost)).Methods("POST")
}

func getReviewsForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_REVIEW_VOTES) {
		c.SetPermissionError(model.PERMISSION_CREATE_REVIEW_VOTES)
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil || post.TeamId != "" {
		c.SetPermissionError(model.PERMISSION_CREATE_REVIEW_VOTES)
		return
	}

	options := &model.SearchReviewsOptions{PostId: post.Id, IncludeInvalidated: true, IncludeCompleted: true, IncludeRejected: true, ReviewType: model.VOTE_TYPE_REVIEW}

	getReviews(c, w, r, options)
}

func getReviewsForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_REVIEW_VOTES) {
		c.SetPermissionError(model.PERMISSION_CREATE_REVIEW_VOTES)
		return
	}

	options := &model.SearchReviewsOptions{UserId: c.Params.UserId, IncludeInvalidated: true, IncludeCompleted: true, IncludeRejected: true, ReviewType: model.VOTE_TYPE_REVIEW}

	getReviews(c, w, r, options)
}

func searchReviews(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_REVIEW_VOTES) {
		c.SetPermissionError(model.PERMISSION_CREATE_REVIEW_VOTES)
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err == nil && post.TeamId != "" {
		c.SetPermissionError(model.PERMISSION_CREATE_REVIEW_VOTES)
		return
	}

	options := &model.SearchReviewsOptions{}

	if post != nil && post.TeamId == "" {
		options.PostId = post.Id
	}

	if c.Params.UserId != "" {
		options.UserId = c.Params.UserId
	}

	if c.Params.ReviewType != "" {
		options.ReviewType = c.Params.ReviewType
	}

	m := model.MapFromJson(r.Body)
	if len(m["tagged"]) > 0 {
		options.Tagged = m["tagged"]
	}

	getReviews(c, w, r, options)
}

func getReviews(c *Context, w http.ResponseWriter, r *http.Request, options *model.SearchReviewsOptions) {
	if c.Params.FromDate != 0 && c.Params.ToDate != 0 && c.Params.FromDate > c.Params.ToDate {
		c.SetInvalidUrlParam("from_to_dates")
		return
	}

	options.FromDate = c.Params.FromDate
	options.ToDate = c.Params.ToDate
	options.Page = c.Params.Page
	options.PerPage = c.Params.PerPage

	reviews, totalCount, err := c.App.GetReviews(options, true)
	if err != nil {
		c.Err = err
		return
	}

	data := model.ReviewsWithCount{Reviews: reviews, TotalCount: totalCount}

	w.Write([]byte(data.ToJson()))
}

func createReviewVote(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	user, err := c.App.GetUser(c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_REVIEW_VOTES) && user.Points < model.MIN_USER_POINT_FOR_VOTE_REVIEW {
		c.Err = model.NewAppError("createReviewVote", "api.review.create_review_vote.low_points.app_error", nil, "", http.StatusBadRequest)
		return
	}

	// TODO: post毎に上限有り。type-flagは一旦上限無し。
	m := model.MapFromJson(r.Body)
	tagContents := m["tags"]

	tagContents = model.ParseTags(tagContents)
	tags := strings.Fields(tagContents)

	if len(tags) <= 0 {
		c.SetInvalidParam("tags")
		return
	}

	vote, err := c.App.CreateReviewVote(c.Params.PostId, c.Params.UserId, tagContents)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(vote.ToJson()))
}

func rejectReviewsForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_COMPLETE_REVIEW_VOTES) {
		c.SetPermissionError(model.PERMISSION_COMPLETE_REVIEW_VOTES)
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil {
		c.Err = err
		return
	}

	if post.TeamId != "" {
		c.SetPermissionError(model.PERMISSION_COMPLETE_REVIEW_VOTES)
		return
	}

	err = c.App.RejectReviewsForPost(post.Id, c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func completeReviewsForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_COMPLETE_REVIEW_VOTES) {
		c.SetPermissionError(model.PERMISSION_COMPLETE_REVIEW_VOTES)
		return
	}

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_COMPLETE_REVIEW_VOTES)
		return
	}

	if post.TeamId != "" {
		c.SetPermissionError(model.PERMISSION_COMPLETE_REVIEW_VOTES)
		return
	}

	err = c.App.CompleteReviewsForPost(post, c.App.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
