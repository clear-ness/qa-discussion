package api

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitPost() {
	api.BaseRoutes.Posts.Handle("/question", api.ApiSessionRequired(createQuestionPost)).Methods("POST")
	api.BaseRoutes.Posts.Handle("/answer", api.ApiSessionRequired(createAnswerPost)).Methods("POST")
	api.BaseRoutes.Posts.Handle("/comment", api.ApiSessionRequired(createCommentPost)).Methods("POST")

	// with no comments, with authors
	api.BaseRoutes.Posts.Handle("/questions", api.ApiHandler(getQuestions)).Methods("POST")

	api.BaseRoutes.Post.Handle("/answers", api.ApiHandler(getAnswersForPost)).Methods("GET")

	api.BaseRoutes.Post.Handle("", api.ApiHandler(getPost)).Methods("GET")

	api.BaseRoutes.Post.Handle("/comments", api.ApiHandler(getCommentsForPost)).Methods("GET")

	api.BaseRoutes.Post.Handle("", api.ApiSessionRequired(updatePost)).Methods("PUT")
	api.BaseRoutes.Post.Handle("", api.ApiSessionRequired(deletePost)).Methods("DELETE")

	api.BaseRoutes.Post.Handle("/best", api.ApiSessionRequired(selectBestAnswer)).Methods("POST")

	api.BaseRoutes.Post.Handle("/upvote", api.ApiSessionRequired(upvotePost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_upvote", api.ApiSessionRequired(cancelUpvotePost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/downvote", api.ApiSessionRequired(downvotePost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_downvote", api.ApiSessionRequired(cancelDownvotePost)).Methods("POST")

	api.BaseRoutes.Post.Handle("/flag", api.ApiSessionRequired(flagPost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_flag", api.ApiSessionRequired(cancelFlagPost)).Methods("POST")

	api.BaseRoutes.Posts.Handle("/search", api.ApiHandler(searchPosts)).Methods("POST")
	api.BaseRoutes.Posts.Handle("/advanced_search", api.ApiHandler(advancedSearchPosts)).Methods("POST")

	api.BaseRoutes.PostsForUser.Handle("/questions", api.ApiHandler(getQuestionsForUser)).Methods("GET")
	api.BaseRoutes.PostsForUser.Handle("/answers", api.ApiHandler(getAnswersForUser)).Methods("GET")

	// Moderators can lock quesions/answers.
	// Locked quesions/answers cannot be voted on or changed in any way.
	api.BaseRoutes.Post.Handle("/lock", api.ApiSessionRequired(lockPost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_lock", api.ApiSessionRequired(cancelLockPost)).Methods("POST")

	// Moderators can protect questions.
	// Protected questions only allow answers by users with more than ~ reputation.
	api.BaseRoutes.Post.Handle("/protect", api.ApiSessionRequired(protectPost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_protect", api.ApiSessionRequired(cancelProtectPost)).Methods("POST")
}

func getQuestions(c *Context, w http.ResponseWriter, r *http.Request) {
	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION}

	if c.Params.NoAnswers {
		options.NoAnswers = true
	}

	m := model.MapFromJson(r.Body)
	if len(m["tagged"]) > 0 {
		options.Tagged = m["tagged"]
	}

	getPosts(c, w, r, options, false, false, false, true)
}

func getAnswersForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_ANSWER, ParentId: c.Params.PostId}
	getPosts(c, w, r, options, true, false, true, false)
}

func getCommentsForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_COMMENT, ParentId: c.Params.PostId}
	getPosts(c, w, r, options, false, false, false, false)
}

func getPosts(c *Context, w http.ResponseWriter, r *http.Request, options *model.GetPostsOptions, getComments bool, getParent bool, checkVoted bool, limitContent bool) {
	if c.Params.FromDate != 0 && c.Params.ToDate != 0 && c.Params.FromDate > c.Params.ToDate {
		c.SetInvalidUrlParam("from_to_dates")
		return
	}

	if c.Params.Min != nil && c.Params.Max != nil && *c.Params.Min > *c.Params.Max {
		c.SetInvalidUrlParam("min_max")
		return
	}

	sort := c.Params.SortType
	if len(sort) > 0 && sort != model.POST_SORT_TYPE_CREATION && sort != model.POST_SORT_TYPE_ACTIVE && sort != model.POST_SORT_TYPE_VOTES && sort != model.POST_SORT_TYPE_ANSWERS {
		c.SetInvalidUrlParam("sort")
		return
	}
	if sort == model.POST_SORT_TYPE_ANSWERS && options.PostType != model.POST_TYPE_QUESTION {
		c.SetInvalidUrlParam("sort")
		return
	}

	// TODO: sortable when type is comment
	if len(sort) > 0 && options.PostType != model.POST_TYPE_COMMENT {
		options.SortType = sort
	}

	options.FromDate = c.Params.FromDate
	options.ToDate = c.Params.ToDate
	options.Page = c.Params.Page
	options.PerPage = c.Params.PerPage

	if c.Params.Min != nil {
		options.Min = c.Params.Min
	}
	if c.Params.Max != nil {
		options.Max = c.Params.Max
	}

	posts, totalCount, err := c.App.GetPosts(options, getComments, getParent, checkVoted, limitContent)
	if err != nil {
		c.Err = err
		return
	}

	data := model.PostsWithCount{Posts: posts, TotalCount: totalCount}

	w.Write([]byte(data.ToJson()))
}

func getPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(post.ToJson()))
}

func createQuestionPost(c *Context, w http.ResponseWriter, r *http.Request) {
	post := model.PostFromJson(r.Body)
	if post == nil {
		c.SetInvalidParam("post")
		return
	}

	post.UserId = c.App.Session.UserId

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_POST) {
		c.SetPermissionError(model.PERMISSION_CREATE_POST)
		return
	}

	rp, err := c.App.CreateQuestion(post)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(rp.ToJson()))
}

func createAnswerPost(c *Context, w http.ResponseWriter, r *http.Request) {
	post := model.PostFromJson(r.Body)
	if post == nil {
		c.SetInvalidParam("post")
		return
	}

	post.UserId = c.App.Session.UserId

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_POST) {
		c.SetPermissionError(model.PERMISSION_CREATE_POST)
		return
	}

	rp, err := c.App.CreateAnswer(post)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(rp.ToJson()))
}

func createCommentPost(c *Context, w http.ResponseWriter, r *http.Request) {
	post := model.PostFromJson(r.Body)
	if post == nil {
		c.SetInvalidParam("post")
		return
	}

	post.UserId = c.App.Session.UserId

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_POST) {
		c.SetPermissionError(model.PERMISSION_CREATE_POST)
		return
	}

	rp, err := c.App.CreateComment(post)
	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(rp.ToJson()))
}

func updatePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	post := model.PostFromJson(r.Body)

	if post == nil {
		c.SetInvalidParam("post")
		return
	}

	if post.Id != c.Params.PostId {
		c.SetInvalidParam("id")
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_POST) {
		c.SetPermissionError(model.PERMISSION_EDIT_POST)
		return
	}

	originalPost, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_EDIT_POST)
		return
	}

	if c.App.Session.UserId != originalPost.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_OTHERS_POSTS) {
			c.SetPermissionError(model.PERMISSION_EDIT_OTHERS_POSTS)
			return
		}
	}

	if originalPost.IsLocked() {
		c.SetPermissionError(model.PERMISSION_EDIT_POST)
		return
	}

	post.Id = c.Params.PostId

	rpost, err := c.App.UpdatePost(post)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(rpost.ToJson()))
}

func deletePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	post, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_DELETE_POST)
		return
	}

	if c.App.Session.UserId == post.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_DELETE_POST) {
			c.SetPermissionError(model.PERMISSION_DELETE_POST)
			return
		}
	} else {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_DELETE_OTHERS_POSTS) {
			return
		}
	}

	if _, err := c.App.DeletePost(post, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func selectBestAnswer(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId().RequireBestId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_POST) {
		c.SetPermissionError(model.PERMISSION_EDIT_POST)
		return
	}

	post, err := c.App.GetSinglePostByType(c.Params.PostId, model.POST_TYPE_QUESTION)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_EDIT_POST)
		return
	}

	if c.App.Session.UserId != post.UserId {
		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_OTHERS_POSTS) {
			c.SetPermissionError(model.PERMISSION_EDIT_OTHERS_POSTS)
			return
		}
	}

	if err := c.App.SelectBestAnswer(c.Params.PostId, c.Params.BestId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func upvotePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_VOTE_POST) {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	_, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if err := c.App.UpVotePost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func cancelUpvotePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_VOTE_POST) {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	_, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if err := c.App.CancelUpVotePost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func downvotePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_VOTE_POST) {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	_, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if err := c.App.DownVotePost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func cancelDownvotePost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_VOTE_POST) {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	_, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if err := c.App.CancelDownVotePost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func flagPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_FLAG_POST) {
		c.SetPermissionError(model.PERMISSION_FLAG_POST)
		return
	}

	_, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_FLAG_POST)
		return
	}

	if err := c.App.FlagPost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func cancelFlagPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_FLAG_POST) {
		c.SetPermissionError(model.PERMISSION_FLAG_POST)
		return
	}

	_, err := c.App.GetSinglePost(c.Params.PostId)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_FLAG_POST)
		return
	}

	if err := c.App.CancelFlagPost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func searchPosts(c *Context, w http.ResponseWriter, r *http.Request) {
	m := model.MapFromJson(r.Body)

	postType := m["post_type"]
	if !model.IsQuestionOrAnswer(postType) {
		c.SetInvalidParam("post_type")
		return
	}

	options := &model.GetPostsOptions{PostType: postType}

	if len(m["user_id"]) == 26 {
		options.UserId = m["user_id"]
	}

	if postType == model.POST_TYPE_QUESTION {
		if c.Params.NoAnswers {
			options.NoAnswers = true
		}

		if len(m["tagged"]) > 0 {
			options.Tagged = m["tagged"]
		}
	}

	getPosts(c, w, r, options, false, false, false, true)
}

func advancedSearchPosts(c *Context, w http.ResponseWriter, r *http.Request) {
	params := model.AdvancedSearchParameterFromJson(r.Body)
	if params.Terms == nil || len(*params.Terms) == 0 || len(*params.Terms) > model.POST_SEARCH_TERMS_MAX {
		c.SetInvalidParam("terms")
		return
	}

	terms := params.Terms

	sort := c.Params.SortType
	if len(sort) > 0 && sort != model.POST_SORT_TYPE_CREATION && sort != model.POST_SORT_TYPE_ACTIVE && sort != model.POST_SORT_TYPE_VOTES {
		c.SetInvalidUrlParam("sort")
		return
	}

	timeZoneOffset := 0
	if c.Params.TimeZoneOffset != nil {
		timeZoneOffset = *c.Params.TimeZoneOffset
	}

	posts, totalCount, err := c.App.SearchPosts(*terms, sort, true, true, c.Params.Page, c.Params.PerPage, timeZoneOffset)
	if err != nil {
		c.Err = err
		return
	}

	data := model.PostsWithCount{Posts: posts, TotalCount: totalCount}

	//w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write([]byte(data.ToJson()))
}

func getQuestionsForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return

	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION, UserId: c.Params.UserId}

	if c.Params.NoAnswers {
		options.NoAnswers = true
	}

	getPosts(c, w, r, options, false, false, false, true)
}

func getAnswersForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return

	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_ANSWER, UserId: c.Params.UserId}
	getPosts(c, w, r, options, false, true, false, true)
}

func lockPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_LOCK_POST) {
		c.SetPermissionError(model.PERMISSION_LOCK_POST)
		return
	}

	if err := c.App.LockPost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func cancelLockPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_LOCK_POST) {
		c.SetPermissionError(model.PERMISSION_LOCK_POST)
		return
	}

	if err := c.App.CancelLockPost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func protectPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_PROTECT_POST) {
		c.SetPermissionError(model.PERMISSION_PROTECT_POST)
		return
	}

	if err := c.App.ProtectPost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func cancelProtectPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_PROTECT_POST) {
		c.SetPermissionError(model.PERMISSION_PROTECT_POST)
		return
	}

	if err := c.App.CancelProtectPost(c.Params.PostId, c.App.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}
