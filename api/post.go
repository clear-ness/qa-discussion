package api

import (
	"math"
	"net/http"
	"strconv"

	"github.com/clear-ness/qa-discussion/model"
)

func (api *API) InitPost() {
	// GUIの左サイドバーで特定Teamを選んだ状態で「private ask」すると、
	// そのteamに紐づくプライベートなpostとなる。
	// Team無選択状態でaskした場合は、teamに紐付かない。teamに紐づくpostは常にプライベート。
	// teamのメンバー以外はteam全体にaskする事は出来ないし、teamに紐づくpostを一切CRUD出来ない。
	// teamのメンバーなら(groupでなく)team全体、
	// または private groupにもask出来る。(結局teamに紐づく)
	// https://www.stackoverflow.help/support/solutions/articles/36000213146-manage-user-groups
	//
	// 左サイドバーのTeam無選択状態でpublic teamかつpublic groupにaskした場合
	// (つまりr.body.post.teamIdが無いがc.Params.groupIdが存在する場合)、
	// public group内ユーザーへのinboxMessageが作成される。(inboxMessageはgroup.teamIdに紐づく)
	// それらの投稿はteamに紐付かない、groupはあくまでも通知目的。
	api.BaseRoutes.Posts.Handle("/question", api.ApiSessionRequired(createQuestionPost)).Methods("POST")
	api.BaseRoutes.Posts.Handle("/answer", api.ApiSessionRequired(createAnswerPost)).Methods("POST")
	api.BaseRoutes.Posts.Handle("/comment", api.ApiSessionRequired(createCommentPost)).Methods("POST")

	// with no comments, with authors
	api.BaseRoutes.Posts.Handle("/questions", api.ApiHandler(getQuestions)).Methods("POST")
	api.BaseRoutes.PostsForTeam.Handle("/questions", api.ApiSessionRequired(getQuestionsForTeam)).Methods("POST")

	api.BaseRoutes.Post.Handle("/answers", api.ApiHandler(getAnswersForPost)).Methods("GET")
	api.BaseRoutes.PostForTeam.Handle("/answers", api.ApiSessionRequired(getAnswersForTeamPost)).Methods("GET")

	api.BaseRoutes.Post.Handle("", api.ApiHandler(getPost)).Methods("GET")
	api.BaseRoutes.PostForTeam.Handle("", api.ApiSessionRequired(getTeamPost)).Methods("GET")

	api.BaseRoutes.Post.Handle("/view", api.ApiHandler(viewPost)).Methods("POST")

	api.BaseRoutes.Post.Handle("/comments", api.ApiHandler(getCommentsForPost)).Methods("GET")
	api.BaseRoutes.PostForTeam.Handle("/comments", api.ApiSessionRequired(getCommentsForTeamPost)).Methods("GET")

	api.BaseRoutes.Post.Handle("/linked", api.ApiHandler(getLinkedForPost)).Methods("GET")
	api.BaseRoutes.PostForTeam.Handle("/linked", api.ApiSessionRequired(getLinkedForTeamPost)).Methods("GET")

	api.BaseRoutes.RevisionsForPost.Handle("", api.ApiHandler(getRevisionsForPost)).Methods("GET")
	api.BaseRoutes.RevisionsForPost.Handle("/total_count", api.ApiHandler(getCurrentRevisionForPost)).Methods("GET")
	api.BaseRoutes.RevisionForPost.Handle("", api.ApiHandler(getRevisionPost)).Methods("GET")

	api.BaseRoutes.RevisionsForPostForTeam.Handle("", api.ApiHandler(getRevisionsForTeamPost)).Methods("GET")
	api.BaseRoutes.RevisionsForPostForTeam.Handle("/total_count", api.ApiHandler(getCurrentRevisionForTeamPost)).Methods("GET")
	api.BaseRoutes.RevisionForPostForTeam.Handle("", api.ApiHandler(getRevisionTeamPost)).Methods("GET")

	api.BaseRoutes.Post.Handle("", api.ApiSessionRequired(updatePost)).Methods("PUT")
	api.BaseRoutes.Post.Handle("", api.ApiSessionRequired(deletePost)).Methods("DELETE")

	api.BaseRoutes.Post.Handle("/best", api.ApiSessionRequired(selectBestAnswer)).Methods("POST")

	api.BaseRoutes.Post.Handle("/upvote", api.ApiSessionRequired(upvotePost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_upvote", api.ApiSessionRequired(cancelUpvotePost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/downvote", api.ApiSessionRequired(downvotePost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_downvote", api.ApiSessionRequired(cancelDownvotePost)).Methods("POST")

	api.BaseRoutes.Post.Handle("/flag", api.ApiSessionRequired(flagPost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_flag", api.ApiSessionRequired(cancelFlagPost)).Methods("POST")

	// TODO: sort by post.views
	api.BaseRoutes.Posts.Handle("/search", api.ApiHandler(searchPosts)).Methods("POST")
	api.BaseRoutes.PostsForTeam.Handle("/search", api.ApiSessionRequired(searchPostsForTeam)).Methods("POST")

	api.BaseRoutes.Posts.Handle("/advanced_search", api.ApiHandler(advancedSearchPosts)).Methods("POST")
	api.BaseRoutes.PostsForTeam.Handle("/advanced_search", api.ApiSessionRequired(advancedSearchPostsForTeam)).Methods("POST")

	api.BaseRoutes.Posts.Handle("/similar", api.ApiHandler(similarPosts)).Methods("POST")
	api.BaseRoutes.PostsForTeam.Handle("/similar", api.ApiSessionRequired(similarPostsForTeam)).Methods("POST")

	api.BaseRoutes.Posts.Handle("/related", api.ApiHandler(relatedPosts)).Methods("POST")

	// questions with the most views, answers, and votes over the last few days, or a week, or a month
	api.BaseRoutes.Posts.Handle("/hot", api.ApiHandler(hotPosts)).Methods("GET")
	api.BaseRoutes.PostsForTeam.Handle("/hot", api.ApiSessionRequired(hotPostsForTeam)).Methods("GET")

	api.BaseRoutes.PostsForUser.Handle("/questions", api.ApiHandler(getQuestionsForUser)).Methods("GET")
	api.BaseRoutes.TeamForUser.Handle("/questions", api.ApiSessionRequired(getQuestionsForTeamUser)).Methods("GET")

	api.BaseRoutes.PostsForUser.Handle("/answers", api.ApiHandler(getAnswersForUser)).Methods("GET")
	api.BaseRoutes.TeamForUser.Handle("/answers", api.ApiSessionRequired(getAnswersForTeamUser)).Methods("GET")

	// Moderators can lock quesions/answers.
	// Locked quesions/answers cannot be voted on or changed in any way.
	api.BaseRoutes.Post.Handle("/lock", api.ApiSessionRequired(lockPost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_lock", api.ApiSessionRequired(cancelLockPost)).Methods("POST")

	// Moderators can protect questions.
	// Protected questions only allow answers by users with more than ~ reputation.
	api.BaseRoutes.Post.Handle("/protect", api.ApiSessionRequired(protectPost)).Methods("POST")
	api.BaseRoutes.Post.Handle("/cancel_protect", api.ApiSessionRequired(cancelProtectPost)).Methods("POST")

	// TODO:
	// 1. inboxMessageが作成される際、postが新規作成及び更新される際、post vote countを更新する際にwebSocketでも相手に通知する
	// 2. statusテーブルを用意し、リアルタイムにユーザーのオンライン/オフライン状態を管理
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

func getQuestionsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION, TeamId: c.Params.TeamId}

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

func getAnswersForTeamPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_ANSWER, ParentId: c.Params.PostId, TeamId: c.Params.TeamId}
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

func getCommentsForTeamPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_COMMENT, ParentId: c.Params.PostId, TeamId: c.Params.TeamId}
	getPosts(c, w, r, options, false, false, false, false)
}

func getLinkedForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION}

	link := model.GetLink(c.App.GetSiteURL(), c.Params.PostId)
	options.Link = link
	getPosts(c, w, r, options, false, false, false, true)
}

func getLinkedForTeamPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION, TeamId: c.Params.TeamId}

	link := model.GetLink(c.App.GetSiteURL(), c.Params.PostId)
	options.Link = link
	getPosts(c, w, r, options, false, false, false, true)
}

func getRevisionsForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	sort := c.Params.SortType
	if sort != model.POST_SORT_TYPE_ACTIVE {
		c.SetInvalidUrlParam("sort")
		return
	}

	options := &model.GetPostsOptions{IncludeDeleted: true, OriginalId: c.Params.PostId}

	getPosts(c, w, r, options, false, false, false, false)
}

func getCurrentRevisionForPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}

	count, err := c.App.GetCurrentRevisionForPost(c.Params.PostId, "")
	if err != nil {
		c.Err = err
		return
	}

	countStr := strconv.FormatInt(count, 10)
	w.Write([]byte(model.MapToJson(map[string]string{"post_id": c.Params.PostId, "current_revision": countStr})))
}

func getRevisionPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId().RequireRevisionId()
	if c.Err != nil {
		return
	}

	revInt, err := strconv.Atoi(c.Params.RevisionId)
	if err != nil || revInt < 1 || revInt > math.MaxUint16 {
		c.SetInvalidUrlParam("revision_id")
		return
	}

	post, err2 := c.App.GetRevisionPost(c.Params.PostId, "", revInt-1)
	if err2 != nil {
		c.Err = err2
		return
	}

	w.Write([]byte(post.ToJson()))
}

func getRevisionsForTeamPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	sort := c.Params.SortType
	if sort != model.POST_SORT_TYPE_ACTIVE {
		c.SetInvalidUrlParam("sort")
		return
	}

	options := &model.GetPostsOptions{TeamId: c.Params.TeamId, IncludeDeleted: true, OriginalId: c.Params.PostId}

	getPosts(c, w, r, options, false, false, false, false)
}

func getCurrentRevisionForTeamPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	count, err := c.App.GetCurrentRevisionForPost(c.Params.PostId, c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	countStr := strconv.FormatInt(count, 10)
	w.Write([]byte(model.MapToJson(map[string]string{"post_id": c.Params.PostId, "current_revision": countStr})))
}

func getRevisionTeamPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequirePostId().RequireRevisionId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	revInt, err := strconv.Atoi(c.Params.RevisionId)
	if err != nil || revInt < 1 || revInt > math.MaxUint16 {
		c.SetInvalidUrlParam("revision_id")
		return
	}

	post, err2 := c.App.GetRevisionPost(c.Params.PostId, c.Params.TeamId, revInt-1)
	if err2 != nil {
		c.Err = err2
		return
	}

	w.Write([]byte(post.ToJson()))
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
		if sort != model.POST_SORT_TYPE_RELEVANCE || (sort == model.POST_SORT_TYPE_RELEVANCE && len(options.Title) <= 0) {
			c.SetInvalidUrlParam("sort")
			return
		}
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

func getTeamPost(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequirePostId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(post.ToJson()))
}

func viewPost(c *Context, w http.ResponseWriter, r *http.Request) {
	// TODO: このAPI自体がconfig.cacheEnableがtrueである前提。
	// そうで無ければ弾く。

	c.RequirePostId()
	if c.Err != nil {
		return
	}

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil {
		c.Err = err
		return
	}

	if post.Type != model.POST_TYPE_QUESTION {
		c.SetInvalidParam("post")
		return
	}

	userId := ""
	ipAddress := ""
	if len(c.App.Session.UserId) == 0 {
		ipAddress = c.App.IpAddress
	} else {
		userId = c.App.Session.UserId
	}

	c.App.ViewPost(post, userId, ipAddress)
}

func createQuestionPost(c *Context, w http.ResponseWriter, r *http.Request) {
	// 自分が所属するteamIdがある場合、private askの場合であり、postはチームに紐付く。
	// さらにgroupIdがある場合、(private or public) groupユーザー達に対してinboxを作成。
	// (private groupに対しての場合、自分が所属するteamに紐付くgroupIdである事が前提)
	// groupIdが無い場合、inboxは作成され無い。
	//
	// 自分が所属するteamIdが無い場合、public askの場合であり、postはチームに紐付か無い。
	// さらにgroupIdがある場合、(public) groupユーザー達に対してinboxを作成。
	// groupIdが無い場合、inboxは作成され無い。最も一般的なケース。
	post := model.PostFromJson(r.Body)
	if post == nil {
		c.SetInvalidParam("post")
		return
	}

	post.UserId = c.App.Session.UserId

	var group *model.UserGroup

	if len(post.TeamId) == 26 {
		// private askの場合
		team, err := c.App.GetTeam(post.TeamId)
		if err != nil {
			c.Err = err
			return
		}

		if !c.App.SessionHasPermissionToTeam(c.App.Session, team.Id, model.PERMISSION_CREATE_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_CREATE_TEAM_POST)
			return
		}

		if len(c.Params.GroupId) == 26 {
			group, err = c.App.GetGroup(c.Params.GroupId)
			if err != nil {
				c.Err = err
				return
			}

			if group.TeamId != team.Id {
				c.SetInvalidParam("group_id")
				return
			}
		}
	} else {
		// public askの場合
		post.TeamId = ""

		if len(c.Params.GroupId) == 26 {
			group, err := c.App.GetGroup(c.Params.GroupId)
			if err != nil {
				c.Err = err
				return
			}

			team, err := c.App.GetTeam(group.TeamId)
			if err != nil {
				c.Err = err
				return
			}

			if !(team.Type == model.TEAM_TYPE_PUBLIC && group.Type == model.GROUP_TYPE_PUBLIC) && !c.App.SessionHasPermissionToTeam(c.App.Session, group.TeamId, model.PERMISSION_CREATE_TEAM_POST) {
				c.SetPermissionError(model.PERMISSION_CREATE_TEAM_POST)
				return
			}
		} else if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_POST) {
			// 最も一般的なケース
			c.SetPermissionError(model.PERMISSION_CREATE_POST)
			return
		}
	}

	rp, err := c.App.CreateQuestion(post, group)
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

	if len(post.TeamId) == 26 {
		team, err := c.App.GetTeam(post.TeamId)
		if err != nil {
			c.Err = err
			return
		}

		if !c.App.SessionHasPermissionToTeam(c.App.Session, team.Id, model.PERMISSION_CREATE_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_CREATE_TEAM_POST)
			return
		}
	} else {
		post.TeamId = ""

		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_POST) {
			c.SetPermissionError(model.PERMISSION_CREATE_POST)
			return
		}
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

	if len(post.TeamId) == 26 {
		team, err := c.App.GetTeam(post.TeamId)
		if err != nil {
			c.Err = err
			return
		}

		if !c.App.SessionHasPermissionToTeam(c.App.Session, team.Id, model.PERMISSION_CREATE_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_CREATE_TEAM_POST)
			return
		}
	} else {
		post.TeamId = ""

		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_CREATE_POST) {
			c.SetPermissionError(model.PERMISSION_CREATE_POST)
			return
		}
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

	originalPost, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_EDIT_POST)
		return
	}

	if originalPost.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, originalPost.TeamId, model.PERMISSION_EDIT_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_EDIT_TEAM_POST)
			return
		}
	} else {
		post.TeamId = ""

		if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_POST) {
			c.SetPermissionError(model.PERMISSION_EDIT_POST)
			return
		}
	}

	if c.App.Session.UserId != originalPost.UserId {
		if originalPost.TeamId != "" {
			if !c.App.SessionHasPermissionToTeam(c.App.Session, originalPost.TeamId, model.PERMISSION_EDIT_OTHERS_TEAM_POSTS) {
				c.SetPermissionError(model.PERMISSION_EDIT_OTHERS_TEAM_POSTS)
				return
			}
		} else {
			if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_EDIT_OTHERS_POSTS) {
				c.SetPermissionError(model.PERMISSION_EDIT_OTHERS_POSTS)
				return
			}
		}
	}

	if originalPost.TeamId == "" && originalPost.IsLocked() {
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

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_DELETE_POST)
		return
	}

	if c.App.Session.UserId == post.UserId {
		if post.TeamId != "" {
			if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_DELETE_TEAM_POST) {
				c.SetPermissionError(model.PERMISSION_DELETE_TEAM_POST)
				return
			}
		} else {
			if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_DELETE_POST) {
				c.SetPermissionError(model.PERMISSION_DELETE_POST)
				return
			}
		}
	} else {
		if post.TeamId != "" {
			if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_DELETE_OTHERS_TEAM_POSTS) {
				return
			}
		} else {
			if !c.App.SessionHasPermissionTo(c.App.Session, model.PERMISSION_DELETE_OTHERS_POSTS) {
				return
			}
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

	if post.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_EDIT_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_EDIT_TEAM_POST)
			return
		}
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

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if post.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_VOTE_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_VOTE_TEAM_POST)
			return
		}
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

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if post.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_VOTE_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_VOTE_TEAM_POST)
			return
		}
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

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if post.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_VOTE_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_VOTE_TEAM_POST)
			return
		}
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

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_VOTE_POST)
		return
	}

	if post.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_VOTE_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_VOTE_TEAM_POST)
			return
		}
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

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_FLAG_POST)
		return
	}

	if post.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_FLAG_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_FLAG_TEAM_POST)
			return
		}
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

	post, err := c.App.GetSinglePost(c.Params.PostId, false)
	if err != nil {
		c.SetPermissionError(model.PERMISSION_FLAG_POST)
		return
	}

	if post.TeamId != "" {
		if !c.App.SessionHasPermissionToTeam(c.App.Session, post.TeamId, model.PERMISSION_FLAG_TEAM_POST) {
			c.SetPermissionError(model.PERMISSION_FLAG_TEAM_POST)
			return
		}
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

func searchPostsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	m := model.MapFromJson(r.Body)

	postType := m["post_type"]
	if !model.IsQuestionOrAnswer(postType) {
		c.SetInvalidParam("post_type")
		return
	}

	options := &model.GetPostsOptions{PostType: postType, TeamId: c.Params.TeamId}

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

	posts, totalCount, err := c.App.SearchPosts(*terms, sort, true, true, c.Params.Page, c.Params.PerPage, timeZoneOffset, "")
	if err != nil {
		c.Err = err
		return
	}

	data := model.PostsWithCount{Posts: posts, TotalCount: totalCount}

	//w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write([]byte(data.ToJson()))
}

func advancedSearchPostsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

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

	posts, totalCount, err := c.App.SearchPosts(*terms, sort, true, true, c.Params.Page, c.Params.PerPage, timeZoneOffset, c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	data := model.PostsWithCount{Posts: posts, TotalCount: totalCount}

	//w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write([]byte(data.ToJson()))
}

func similarPosts(c *Context, w http.ResponseWriter, r *http.Request) {
	m := model.MapFromJson(r.Body)

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION}

	if len(m["title"]) <= 0 {
		c.SetInvalidParam("title")
		return
	}
	options.Title = m["title"]

	if len(m["tagged"]) > 0 {
		options.Tagged = m["tagged"]
	}

	getPosts(c, w, r, options, false, false, false, true)
}

func similarPostsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	m := model.MapFromJson(r.Body)

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION, TeamId: c.Params.TeamId}

	if len(m["title"]) <= 0 {
		c.SetInvalidParam("title")
		return
	}
	options.Title = m["title"]

	if len(m["tagged"]) > 0 {
		options.Tagged = m["tagged"]
	}

	getPosts(c, w, r, options, false, false, false, true)
}

func relatedPosts(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequirePostId()
	if c.Err != nil {
		return
	}
	// TODO: ES検索が有効な場合のみ、related search APIを生やす。

	post, err := c.App.GetPost(c.Params.PostId)
	if err != nil {
		c.Err = err
		return
	}

	if post.Type != model.POST_TYPE_QUESTION || post.Title == "" || post.TeamId != "" {
		c.SetInvalidParam("post")
		return
	}

	var results []*model.RelatedPostSearchResult
	if results, err = c.App.RelatedPosts(post); err != nil {
		c.Err = err
		return
	}

	data := model.RelatedPostSearchResultsWithCount{RelatedPostSearchResults: results, TotalCount: int64(len(results))}
	w.Write([]byte(data.ToJson()))
}

func hotPosts(c *Context, w http.ResponseWriter, r *http.Request) {
	// TODO: このAPI自体がconfig.cacheEnableがtrueである前提。
	// そうで無ければ弾く。

	c.RequireHotPostsInterval()
	if c.Err != nil {
		return
	}

	posts, err := c.App.HotPosts(c.Params.HotPostsInterval, "")
	if err != nil {
		c.Err = err
		return
	}

	data := model.PostsWithCount{Posts: posts, TotalCount: int64(len(posts))}
	w.Write([]byte(data.ToJson()))
}

func hotPostsForTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	// TODO: このAPI自体がconfig.cacheEnableがtrueである前提。
	// そうで無ければ弾く。

	c.RequireHotPostsInterval().RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	posts, err := c.App.HotPosts(c.Params.HotPostsInterval, c.Params.TeamId)
	if err != nil {
		c.Err = err
		return
	}

	data := model.PostsWithCount{Posts: posts, TotalCount: int64(len(posts))}
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

func getQuestionsForTeamUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return

	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_QUESTION, UserId: c.Params.UserId, TeamId: c.Params.TeamId}

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

func getAnswersForTeamUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return

	}

	if !c.App.SessionHasPermissionToTeam(c.App.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM_POST) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM_POST)
		return
	}

	options := &model.GetPostsOptions{PostType: model.POST_TYPE_ANSWER, UserId: c.Params.UserId, TeamId: c.Params.TeamId}
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
