package app

import (
	"net/http"
	"strings"

	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store/searchlayer"
	"github.com/clear-ness/qa-discussion/utils"
)

func (a *App) GetSinglePost(postId string, includeDeleted bool) (*model.Post, *model.AppError) {
	return a.Srv.Store.Post().GetSingle(postId, includeDeleted)
}

func (a *App) GetSinglePostByType(postId string, postType string) (*model.Post, *model.AppError) {
	return a.Srv.Store.Post().GetSingleByType(postId, postType)
}

func (a *App) GetPostCount(userId string, postType string, teamId string) (int64, *model.AppError) {
	return a.Srv.Store.Post().GetPostCount(postType, userId, teamId, 0, 0)
}

func (a *App) CreateQuestion(post *model.Post, group *model.UserGroup) (*model.Post, *model.AppError) {
	user, err := a.GetUser(post.UserId)
	if err != nil || user == nil {
		return nil, model.NewAppError("CreateQuestion", "api.post.create_question.post_user.app_error", nil, "", http.StatusBadRequest)
	}

	if len(post.TeamId) == 0 && user.IsSuspending() {
		return nil, model.NewAppError("CreateQuestion", "api.post.create_question.user_suspending.app_error", nil, "", http.StatusBadRequest)
	}

	if post.Tags != "" {
		tags := model.ParseTags(post.Tags)
		if len(strings.Fields(tags)) != len(strings.Fields(post.Tags)) {
			return nil, model.NewAppError("CreateQuestion", "api.post.create_question.parse_tags.app_error", nil, "", http.StatusBadRequest)
		}

		post.Tags = tags
	}

	post = &model.Post{
		Type:        model.POST_TYPE_QUESTION,
		RootId:      "",
		ParentId:    "",
		BestId:      "",
		UserId:      post.UserId,
		TeamId:      post.TeamId,
		Title:       post.Title,
		Content:     post.Content,
		Tags:        post.Tags,
		UpVotes:     0,
		DownVotes:   0,
		AnswerCount: 0,
		FlagCount:   0,
		Views:       0,
		DeleteAt:    0,
	}

	rpost, err := a.Srv.Store.Post().SaveQuestion(post)
	if err != nil {
		mlog.Error("Couldn't save the question", mlog.Err(err))
		return nil, err
	}

	if group != nil {
		max := len(post.Content)
		if max > model.INBOX_MESSAGE_CONTENT_MAX_LENGTH {
			max = model.INBOX_MESSAGE_CONTENT_MAX_LENGTH
		}
		content := post.Content[0:max]

		curTime := model.GetMillis()
		var messages []*model.InboxMessage

		members, err := a.GetGroupMembersPage(group.Id, "", 0, model.GROUP_MEMBER_SEARCH_DEFAULT_LIMIT)
		for _, member := range *members {
			message := &model.InboxMessage{
				Content:    content,
				SenderId:   post.UserId,
				QuestionId: post.Id,
				Title:      post.Title,
				AnswerId:   "",
				CommentId:  "",
				TeamId:     group.TeamId,
				CreateAt:   curTime,
			}
			message.Type = model.INBOX_MESSAGE_TYPE_QUESTION
			message.UserId = member.UserId

			messages = append(messages, message)
		}

		_, err = a.Srv.Store.InboxMessage().SaveMultipleInboxMessages(messages)
		if err != nil {
			return nil, model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.save_multiple_inbox_messages.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		a.PublishInboxMessages(messages)
	}

	if post.TeamId != "" {
		a.tryWebhook(post, user)
	}

	return rpost, nil
}

func (a *App) PublishInboxMessages(messages []*model.InboxMessage) {
	for _, message := range messages {
		event := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_INBOX_MESSAGE, "", message.UserId, nil)
		event.Add("user_id", message.UserId)

		a.Srv.Publish(event)
	}
}

func (a *App) tryWebhook(post *model.Post, user *model.User) {
	a.Srv.Go(func() {
		team, err := a.Srv.Store.Team().Get(post.TeamId)
		if err != nil || team.DeleteAt > 0 {
			return
		}

		if err := a.handleWebhookEvents(post, team, user); err != nil {
			mlog.Error(err.Error())
		}
	})
}

func (a *App) CreateAnswer(post *model.Post) (*model.Post, *model.AppError) {
	if post.ParentId == "" {
		return nil, model.NewAppError("CreateAnswer", "api.post.create_answer.parent.app_error", nil, "", http.StatusBadRequest)
	}

	parentQuestion, err := a.Srv.Store.Post().GetSingleByType(post.ParentId, model.POST_TYPE_QUESTION)
	if err != nil {
		mlog.Error("Couldn't save the answer", mlog.Err(err))
		return nil, err
	}

	if parentQuestion == nil {
		return nil, model.NewAppError("CreateAnswer", "api.post.create_answer.parent.app_error", nil, "", http.StatusBadRequest)
	}

	user, err := a.GetUser(post.UserId)
	if err != nil || user == nil {
		return nil, model.NewAppError("CreateAnswer", "api.post.create_answer.post_user.app_error", nil, "", http.StatusBadRequest)
	}

	if len(post.TeamId) == 0 && user.IsSuspending() {
		return nil, model.NewAppError("CreateAnswer", "api.post.create_answer.user_suspending.app_error", nil, "", http.StatusBadRequest)
	}

	if len(post.TeamId) == 0 && parentQuestion.IsProtected() && user.Points < model.MIN_USER_POINT_FOR_ANSWER_FOR_PROTECTED_POST {
		return nil, model.NewAppError("CreateAnswer", "api.post.create_answer.protected.app_error", nil, "", http.StatusBadRequest)
	}

	post = &model.Post{
		Type:        model.POST_TYPE_ANSWER,
		RootId:      parentQuestion.Id,
		ParentId:    parentQuestion.Id,
		BestId:      "",
		UserId:      post.UserId,
		TeamId:      post.TeamId,
		Title:       "",
		Content:     post.Content,
		Tags:        "",
		UpVotes:     0,
		DownVotes:   0,
		AnswerCount: 0,
		FlagCount:   0,
		Views:       0,
		DeleteAt:    0,
	}

	_, err = a.Srv.Store.Post().SaveAnswer(post)
	if err != nil {
		mlog.Error("Couldn't save the answer", mlog.Err(err))
		return nil, err
	}

	curTime := model.GetMillis()

	max := len(post.Content)
	if max > model.INBOX_MESSAGE_CONTENT_MAX_LENGTH {
		max = model.INBOX_MESSAGE_CONTENT_MAX_LENGTH
	}
	content := post.Content[0:max]

	message := &model.InboxMessage{
		Type:       model.INBOX_MESSAGE_TYPE_ANSWER,
		Content:    content,
		UserId:     parentQuestion.UserId,
		SenderId:   post.UserId,
		QuestionId: parentQuestion.Id,
		Title:      parentQuestion.Title,
		AnswerId:   post.Id,
		TeamId:     post.TeamId,
		CreateAt:   curTime,
	}

	_, err = a.Srv.Store.InboxMessage().SaveInboxMessage(message)
	if err != nil {
		return nil, model.NewAppError("CreateAnswer", "api.post.create_answer.save_inbox_message.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	a.PublishInboxMessages([]*model.InboxMessage{message})

	if post.TeamId != "" {
		a.tryWebhook(post, user)
	}

	return post, nil
}

func (a *App) CreateComment(post *model.Post) (*model.Post, *model.AppError) {
	parent, err := a.Srv.Store.Post().GetSingle(post.ParentId, false)
	if err != nil {
		mlog.Error("Couldn't save the comment", mlog.Err(err))
		return nil, err
	}
	if parent == nil {
		return nil, model.NewAppError("CreateComment", "api.post.create_comment.parent.app_error", nil, "", http.StatusInternalServerError)
	}

	user, err := a.GetUser(post.UserId)
	if err != nil || user == nil {
		return nil, model.NewAppError("CreateComment", "api.post.create_comment.post_user.app_error", nil, "", http.StatusBadRequest)
	}

	if len(post.TeamId) == 0 && user.IsSuspending() {
		return nil, model.NewAppError("CreateComment", "api.post.create_comment.user_suspending.app_error", nil, "", http.StatusBadRequest)
	}

	var rootId string
	switch parent.Type {
	case model.POST_TYPE_QUESTION:
		rootId = parent.Id
	case model.POST_TYPE_ANSWER:
		rootId = parent.ParentId
	default:
		return nil, model.NewAppError("CreateComment", "api.post.create_comment.parent_type.app_error", nil, "", http.StatusInternalServerError)
	}

	post = &model.Post{
		Type:        model.POST_TYPE_COMMENT,
		RootId:      rootId,
		ParentId:    parent.Id,
		BestId:      "",
		UserId:      post.UserId,
		TeamId:      post.TeamId,
		Title:       "",
		Content:     post.Content,
		Tags:        "",
		UpVotes:     0,
		DownVotes:   0,
		AnswerCount: 0,
		FlagCount:   0,
		Views:       0,
		DeleteAt:    0,
	}

	rpost, err := a.Srv.Store.Post().SaveComment(post)
	if err != nil {
		mlog.Error("Couldn't save the comment", mlog.Err(err))
		return nil, err
	}

	rpost, err = a.Srv.Store.Post().GetSingle(rpost.Id, false)
	if err != nil {
		mlog.Error("Couldn't get post for inbox messages", mlog.Err(err))
		return rpost, nil
	}

	err = a.saveInboxMessagesForComment(rpost, true)
	if err != nil {
		mlog.Error("Couldn't save inbox messages for comment", mlog.Err(err))
		return rpost, nil
	}

	if post.TeamId != "" {
		a.tryWebhook(post, user)
	}

	return rpost, nil
}

// users can comment reply to participants of a comment thread or the author of the post
func (a *App) saveInboxMessagesForComment(post *model.Post, forceInformAuthor bool) *model.AppError {
	parent, err := a.Srv.Store.Post().GetSingle(post.ParentId, false)
	if err != nil {
		return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.get_single.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	if parent == nil {
		return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.get_single.app_error", nil, "", http.StatusInternalServerError)
	}

	root, err := a.Srv.Store.Post().GetSingle(post.RootId, false)
	if err != nil {
		return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.get_single.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	if root == nil {
		return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.get_single.app_error", nil, "", http.StatusInternalServerError)
	}

	curTime := model.GetMillis()

	answerId := ""
	if parent.Type == model.POST_TYPE_ANSWER {
		answerId = parent.Id
	}
	max := len(post.Content)
	if max > model.INBOX_MESSAGE_CONTENT_MAX_LENGTH {
		max = model.INBOX_MESSAGE_CONTENT_MAX_LENGTH
	}
	content := post.Content[0:max]

	repliedNames := model.ParseReplies(post.Content)

	if len(repliedNames) <= 0 {
		message := &model.InboxMessage{
			Content:    content,
			SenderId:   post.UserId,
			QuestionId: root.Id,
			Title:      root.Title,
			AnswerId:   answerId,
			CommentId:  post.Id,
			TeamId:     post.TeamId,
			CreateAt:   curTime,
		}
		message.Type = model.INBOX_MESSAGE_TYPE_COMMENT
		message.UserId = parent.UserId
		_, err := a.Srv.Store.InboxMessage().SaveInboxMessage(message)
		if err != nil {
			return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.save_inbox_message.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		a.PublishInboxMessages([]*model.InboxMessage{message})

		return nil
	}

	commentsForPost, err := a.Srv.Store.Post().GetCommentsForPost(parent.Id, model.POST_COMMENT_LIMIT)
	if err != nil {
		return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.get_comments.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	userIdsMaps := map[string]bool{}
	userIdsMaps[parent.UserId] = true

	for _, comment := range commentsForPost {
		userIdsMaps[comment.UserId] = true
	}
	var userIds []string
	for key := range userIdsMaps {
		userIds = append(userIds, key)
	}
	users, err := a.GetUsers(userIds)
	if err != nil {
		return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.get_users.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	usersMap := make(map[string][]*model.User)
	options := map[string]bool{}
	options["email"] = false
	for _, user := range users {
		user.SanitizeProfile(options)

		usersForName := usersMap[user.Username]
		usersForName = append(usersForName, user)
		usersMap[user.Username] = usersForName
	}

	var messages []*model.InboxMessage
	replyToParentAuthor := false

	// TODO: User Groups can also be referred to in comments or posts using @
	// in the same way you would notify any individual member of your Team.
	for _, name := range repliedNames {
		if usersForName, ok := usersMap[name]; ok {
			for _, user := range usersForName {
				message := &model.InboxMessage{
					Content:    content,
					SenderId:   post.UserId,
					QuestionId: root.Id,
					Title:      root.Title,
					AnswerId:   answerId,
					CommentId:  post.Id,
					TeamId:     post.TeamId,
					CreateAt:   curTime,
				}
				message.Type = model.INBOX_MESSAGE_TYPE_COMMENT_REPLY
				message.UserId = user.Id

				messages = append(messages, message)

				if user.Id == parent.UserId {
					replyToParentAuthor = true
				}
			}
		}
	}

	if !replyToParentAuthor && forceInformAuthor {
		message := &model.InboxMessage{
			Content:    content,
			SenderId:   post.UserId,
			QuestionId: root.Id,
			Title:      root.Title,
			AnswerId:   answerId,
			CommentId:  post.Id,
			TeamId:     post.TeamId,
			CreateAt:   curTime,
		}
		message.Type = model.INBOX_MESSAGE_TYPE_COMMENT
		message.UserId = parent.UserId

		messages = append(messages, message)
	}

	_, err = a.Srv.Store.InboxMessage().SaveMultipleInboxMessages(messages)
	if err != nil {
		return model.NewAppError("saveInboxMessagesForComment", "api.post.save_inbox_messages_for_comment.save_multiple_inbox_messages.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	a.PublishInboxMessages(messages)

	return nil
}

func (a *App) GetPostWithMetadata(postId string) (*model.Post, *model.AppError) {
	post, err := a.GetSinglePost(postId, false)
	if err != nil {
		return nil, err
	}

	var posts model.Posts
	posts = append(posts, post)

	// comments of a comment does not exists
	getComments := false
	if model.IsQuestionOrAnswer(post.Type) {
		getComments = true
	}

	getBestAnswer := false
	if post.Type == model.POST_TYPE_QUESTION {
		getBestAnswer = true
	}

	option := model.SetPostMetadataOptions{
		SetUser:       true,
		SetComments:   getComments,
		SetBestAnswer: getBestAnswer,
		SetParent:     false,
	}
	posts, err = a.SetPostMetadata(posts, option)
	if err != nil {
		return nil, err
	}

	return posts[0], nil
}

func (a *App) GetPosts(options *model.GetPostsOptions, getComments bool, getParent bool, checkVoted bool, limitContent bool) (model.Posts, int64, *model.AppError) {
	posts, totalCount, err := a.Srv.Store.Post().GetPosts(options, true)
	if err != nil {
		return nil, 0, err
	}

	option := model.SetPostMetadataOptions{
		SetUser:       true,
		SetComments:   getComments,
		SetBestAnswer: false,
		SetParent:     getParent,
	}
	posts, err = a.SetPostMetadata(posts, option)
	if err != nil {
		return nil, 0, err
	}

	if checkVoted {
		posts, err = a.CheckVoted(posts)
		if err != nil {
			return nil, 0, err
		}
	}

	if limitContent {
		posts.LimitContentLength()
	}

	return posts, totalCount, err
}

func (a *App) CheckVoted(posts model.Posts) (model.Posts, *model.AppError) {
	if a.Session.UserId == "" {
		return posts, nil
	}

	// TODO: use GetVoteTypesForPost
	for _, post := range posts {
		if model.IsQuestionOrAnswer(post.Type) {
			upVote, err := a.GetVote(a.Session.UserId, post.Id, model.VOTE_TYPE_UP_VOTE)
			if err != nil {
				mlog.Error(err.Error())
			}

			if upVote != nil {
				post.UpVoted = true
			}

			downVote, err := a.GetVote(a.Session.UserId, post.Id, model.VOTE_TYPE_DOWN_VOTE)
			if err != nil {
				mlog.Error(err.Error())
			}

			if downVote != nil {
				post.DownVoted = true
			}
		}

		flagged, err := a.GetVote(a.Session.UserId, post.Id, model.VOTE_TYPE_FLAG)
		if err != nil {
			mlog.Error(err.Error())
		}

		if flagged != nil {
			post.Flagged = true
		}
	}

	return posts, nil
}

func (a *App) GetPost(postId string) (*model.Post, *model.AppError) {
	post, err := a.GetPostWithMetadata(postId)
	if err != nil {
		return nil, err
	}

	if post.Type == model.POST_TYPE_ANSWER || post.Type == model.POST_TYPE_COMMENT {
		return post, nil
	}

	if a.Session.UserId != "" {
		favoritePost, err := a.GetUserFavoritePostForUser(a.Session.UserId, post.Id)
		if err != nil {
			mlog.Error(err.Error())
		}

		if favoritePost != nil {
			post.Favorited = true
		}
	}

	count, err := a.GetUserFavoritePostsCountByPostId(post.Id)
	if err != nil {
		return nil, err
	}

	post.FavoriteCount = count

	var posts model.Posts
	posts = append(posts, post)
	posts, err = a.CheckVoted(posts)

	return posts[0], err
}

// only questions or answers may have comments
func (a *App) GetCommentsAndCommentUserForPosts(postIds []string) (map[string][]*model.Post, *model.AppError) {
	comments := make(map[string][]*model.Post)
	userIdsMaps := map[string]bool{}

	for _, postId := range postIds {
		commentsForPost, err := a.Srv.Store.Post().GetCommentsForPost(postId, model.POST_COMMENT_LIMIT)
		if err != nil {
			return nil, err
		}
		comments[postId] = commentsForPost

		for _, comment := range commentsForPost {
			userIdsMaps[comment.UserId] = true
		}
	}

	var userIds []string
	for key := range userIdsMaps {
		userIds = append(userIds, key)
	}

	users, err := a.GetUsers(userIds)
	if err != nil {
		return nil, err
	}

	userMap := map[string]*model.User{}
	options := map[string]bool{}
	options["email"] = false
	for _, user := range users {
		user.SanitizeProfile(options)
		userMap[user.Id] = user
	}

	for _, postId := range postIds {
		if commentsForPost, ok := comments[postId]; ok {
			for _, comment := range commentsForPost {
				comment.Metadata = &model.PostMetadata{}
				if user, ok := userMap[comment.UserId]; ok {
					comment.Metadata.User = user
				}
			}
		}
	}

	comments = populateEmptyComments(postIds, comments)
	return comments, nil
}

func populateEmptyComments(postIds []string, comments map[string][]*model.Post) map[string][]*model.Post {
	for _, postId := range postIds {
		if _, ok := comments[postId]; !ok {
			comments[postId] = []*model.Post{}
		}
	}
	return comments
}

func (a *App) SearchPosts(terms string, sortType string, getParent bool, limitContent bool, page int, perPage int, timeZoneOffset int, teamId string) (model.Posts, int64, *model.AppError) {
	paramsList := model.ParseSearchParams(strings.TrimSpace(terms), timeZoneOffset)

	finalParamsList := []*model.SearchParams{}
	for _, params := range paramsList {
		if params.Terms != "*" {
			finalParamsList = append(finalParamsList, params)
		}
	}
	if len(finalParamsList) == 0 {
		err := model.NewAppError("SearchPosts", "api.post.search_posts.no_params.app_error", nil, "", http.StatusBadRequest)
		return nil, int64(0), err
	}

	posts, totalCount, err := a.Srv.Store.Post().SearchPosts(finalParamsList, sortType, page, perPage, teamId)
	if err != nil {
		return nil, int64(0), err
	}

	option := model.SetPostMetadataOptions{
		SetUser:       true,
		SetComments:   false,
		SetBestAnswer: false,
		SetParent:     getParent,
	}
	posts, err = a.SetPostMetadata(posts, option)
	if err != nil {
		return nil, int64(0), err
	}

	if limitContent {
		posts.LimitContentLength()
	}

	return posts, totalCount, nil
}

func (a *App) UpdatePost(post *model.Post) (*model.Post, *model.AppError) {
	oldPost, err := a.Srv.Store.Post().GetSingle(post.Id, false)
	if err != nil {
		return nil, err
	}

	if oldPost == nil || oldPost.Type != post.Type {
		err = model.NewAppError("UpdatePost", "api.post.update_post.find.app_error", nil, "id="+post.Id, http.StatusBadRequest)
		return nil, err
	}

	if oldPost.DeleteAt != 0 {
		err = model.NewAppError("UpdatePost", "api.post.update_post.permissions_details.app_error", map[string]interface{}{"PostId": post.Id}, "", http.StatusBadRequest)
		return nil, err
	}

	newPost := &model.Post{}
	*newPost = *oldPost

	var edited = false
	if newPost.Type == model.POST_TYPE_QUESTION {
		if newPost.Title != post.Title {
			newPost.Title = post.Title
			edited = true
		}

		tags := model.ParseTags(post.Tags)
		if post.Tags != "" && len(strings.Fields(tags)) != len(strings.Fields(post.Tags)) {
			err = model.NewAppError("UpdatePost", "api.post.update_post.parse_tags.app_error", map[string]interface{}{"PostId": post.Id}, "", http.StatusBadRequest)
			return nil, err
		}
		post.Tags = tags

		removed := utils.StringSliceDiff(strings.Fields(oldPost.Tags), strings.Fields(post.Tags))
		added := utils.StringSliceDiff(strings.Fields(post.Tags), strings.Fields(oldPost.Tags))

		if len(removed) > 0 || len(added) > 0 {
			newPost.Tags = post.Tags
			edited = true
		}
	}

	if newPost.Content != post.Content {
		newPost.Content = post.Content
		edited = true
	}

	if edited {
		newPost.EditAt = model.GetMillis()
	}

	rpost, err := a.Srv.Store.Post().Update(newPost, oldPost)
	if err != nil {
		return nil, err
	}

	if edited && rpost.Type == model.POST_TYPE_COMMENT {
		rpost, err = a.Srv.Store.Post().GetSingle(rpost.Id, false)
		if err != nil {
			mlog.Error("Couldn't get post for inbox messages", mlog.Err(err))
			return rpost, nil
		}

		err := a.saveInboxMessagesForComment(rpost, false)
		if err != nil {
			mlog.Error("Couldn't save inbox messages for comment", mlog.Err(err))
			return rpost, nil
		}
	}

	return rpost, nil
}

func (a *App) DeletePost(post *model.Post, deleteByID string) (*model.Post, *model.AppError) {
	switch post.Type {
	case model.POST_TYPE_QUESTION:
		// with one more child posts, then prevent deleting the post
		count, err := a.Srv.Store.Post().GetChildPostsCount(post.Id)
		if err != nil {
			return nil, err
		}
		if count >= 1 {
			return nil, model.NewAppError("DeletePost", "api.post.delete_question.child.app_error", nil, "", http.StatusInternalServerError)
		}

		if err := a.Srv.Store.Post().DeleteQuestion(post.Id, model.GetMillis(), deleteByID); err != nil {
			return nil, err
		}
	case model.POST_TYPE_ANSWER:
		count, err := a.Srv.Store.Post().GetChildPostsCount(post.Id)
		if err != nil {
			return nil, err
		}
		if count >= 1 {
			return nil, model.NewAppError("DeletePost", "api.post.delete_answer.child.app_error", nil, "", http.StatusInternalServerError)
		}

		parent, err := a.Srv.Store.Post().GetSingleByType(post.ParentId, model.POST_TYPE_QUESTION)
		if err != nil {
			return nil, err
		}
		if parent.BestId == post.Id {
			return nil, model.NewAppError("DeletePost", "api.post.delete_answer.parent.app_error", nil, "", http.StatusInternalServerError)
		}

		if err := a.Srv.Store.Post().DeleteAnswer(post.Id, model.GetMillis(), deleteByID); err != nil {
			return nil, err
		}
	case model.POST_TYPE_COMMENT:
		if err := a.Srv.Store.Post().DeleteComment(post.Id, model.GetMillis(), deleteByID); err != nil {
			return nil, err
		}
	default:
		return nil, model.NewAppError("DeletePost", "api.post.delete.type.app_error", nil, "", http.StatusInternalServerError)
	}

	a.Srv.Go(func() {
		a.DeletePostFiles(post)
	})

	return post, nil
}

func (a *App) DeletePostForcely(post *model.Post, deleteByID string) (*model.Post, *model.AppError) {
	switch post.Type {
	case model.POST_TYPE_QUESTION:
		if err := a.Srv.Store.Post().DeleteQuestion(post.Id, model.GetMillis(), deleteByID); err != nil {
			return nil, err
		}
	case model.POST_TYPE_ANSWER:
		if err := a.Srv.Store.Post().DeleteAnswer(post.Id, model.GetMillis(), deleteByID); err != nil {
			return nil, err
		}
	case model.POST_TYPE_COMMENT:
		if err := a.Srv.Store.Post().DeleteComment(post.Id, model.GetMillis(), deleteByID); err != nil {
			return nil, err
		}
	default:
		return nil, model.NewAppError("DeletePost", "api.post.delete.type.app_error", nil, "", http.StatusInternalServerError)
	}

	a.Srv.Go(func() {
		a.DeletePostFiles(post)
	})

	return post, nil
}

func (a *App) SelectBestAnswer(postId, bestId string) *model.AppError {
	return a.Srv.Store.Post().SelectBestAnswer(postId, bestId)
}

func (a *App) UpVotePost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't upvote the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("UpVotePost", "api.post.upvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.Type != model.POST_TYPE_QUESTION && post.Type != model.POST_TYPE_ANSWER {
		return model.NewAppError("UpVotePost", "api.post.upvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.TeamId == "" && post.IsLocked() {
		return model.NewAppError("UpVotePost", "api.post.upvote.get.app_error", nil, "", http.StatusBadRequest)
	}

	user, err := a.GetUser(userId)
	if err != nil || user == nil {
		return model.NewAppError("UpVotePost", "api.post.upvote.session_user.app_error", nil, "", http.StatusBadRequest)
	}

	if post.TeamId == "" && user.IsSuspending() {
		return model.NewAppError("UpVotePost", "api.post.upvote.user_suspending.app_error", nil, "", http.StatusBadRequest)
	}

	_, err = a.Srv.Store.Post().UpVotePost(postId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) CancelUpVotePost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't cancel upvote the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("CancelUpVotePost", "api.post.cancel_upvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.TeamId == "" && post.IsLocked() {
		return model.NewAppError("CancelUpVotePost", "api.post.cancel_upvote.get.app_error", nil, "", http.StatusBadRequest)
	}

	if post.Type != model.POST_TYPE_QUESTION && post.Type != model.POST_TYPE_ANSWER {
		return model.NewAppError("CancelUpVotePost", "api.post.cancel_upvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	user, err := a.GetUser(userId)
	if err != nil || user == nil {
		return model.NewAppError("CancelUpVotePost", "api.post.cancel_upvote.session_user.app_error", nil, "", http.StatusBadRequest)
	}

	if post.TeamId == "" && user.IsSuspending() {
		return model.NewAppError("CancelUpVotePost", "api.post.cancel_upvote.user_suspending.app_error", nil, "", http.StatusBadRequest)
	}

	_, err = a.Srv.Store.Post().CancelUpVotePost(postId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) DownVotePost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't downvote the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("DownVotePost", "api.post.downvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.Type != model.POST_TYPE_QUESTION && post.Type != model.POST_TYPE_ANSWER {
		return model.NewAppError("DownVotePost", "api.post.downvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.TeamId == "" && post.IsLocked() {
		return model.NewAppError("DownVotePost", "api.post.downvote.get.app_error", nil, "", http.StatusBadRequest)
	}

	user, err := a.GetUser(userId)
	if err != nil || user == nil {
		return model.NewAppError("DownVotePost", "api.post.downvote.session_user.app_error", nil, "", http.StatusBadRequest)
	}

	if post.TeamId == "" && user.IsSuspending() {
		return model.NewAppError("DownVotePost", "api.post.downvote.user_suspending.app_error", nil, "", http.StatusBadRequest)
	}

	_, err = a.Srv.Store.Post().DownVotePost(postId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) CancelDownVotePost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't cancel cancel downvote the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("CancelDownVotePost", "api.post.cancel_downvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.Type != model.POST_TYPE_QUESTION && post.Type != model.POST_TYPE_ANSWER {
		return model.NewAppError("CancelDownVotePost", "api.post.cancel_downvote.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.TeamId == "" && post.IsLocked() {
		return model.NewAppError("CancelDownVotePost", "api.post.cancel_downvote.get.app_error", nil, "", http.StatusBadRequest)
	}

	user, err := a.GetUser(userId)
	if err != nil || user == nil {
		return model.NewAppError("CancelDownVotePost", "api.post.cancel_downvote.session_user.app_error", nil, "", http.StatusBadRequest)
	}

	if post.TeamId == "" && user.IsSuspending() {
		return model.NewAppError("CancelDownVotePost", "api.post.cancel_downvote.user_suspending.app_error", nil, "", http.StatusBadRequest)
	}

	_, err = a.Srv.Store.Post().CancelDownVotePost(postId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) FlagPost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't flag the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("FlagPost", "api.post.flag.get.app_error", nil, "", http.StatusInternalServerError)
	}

	_, err = a.Srv.Store.Post().FlagPost(postId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) CancelFlagPost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't cancel flag the post", mlog.Err(err))
		return err
	}
	if post == nil {
		return model.NewAppError("CancelFlagPost", "api.post.cancel_flag.get.app_error", nil, "", http.StatusInternalServerError)
	}

	_, err = a.Srv.Store.Post().CancelFlagPost(postId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) LockPost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't lock the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("LockPost", "api.post.lock.get.app_error", nil, "", http.StatusNotFound)
	}

	if post.Type != model.POST_TYPE_QUESTION && post.Type != model.POST_TYPE_ANSWER {
		return model.NewAppError("LockPost", "api.post.lock.get.app_error", nil, "", http.StatusBadRequest)
	}

	// TODO: teamの場合も対応
	if post.TeamId != "" {
		return model.NewAppError("LockPost", "api.post.lock.team.app_error", nil, "", http.StatusBadRequest)
	}

	return a.Srv.Store.Post().LockPost(postId, model.GetMillis(), userId)
}

func (a *App) CancelLockPost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't cancel lock the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("CancelLockPost", "api.post.cancel_lock.get.app_error", nil, "", http.StatusNotFound)
	}

	if post.LockedAt == 0 {
		return model.NewAppError("CancelLockPost", "api.post.cancel_lock.get.app_error", nil, "", http.StatusBadRequest)
	}

	if post.Type != model.POST_TYPE_QUESTION && post.Type != model.POST_TYPE_ANSWER {
		return model.NewAppError("CancelLockPost", "api.post.cancel_lock.get.app_error", nil, "", http.StatusBadRequest)
	}

	// TODO: teamの場合も対応
	if post.TeamId != "" {
		return model.NewAppError("CancelLockPost", "api.post.cancel_lock.team.app_error", nil, "", http.StatusBadRequest)
	}

	return a.Srv.Store.Post().CancelLockPost(postId, userId)
}

func (a *App) ProtectPost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't protect the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("ProtectPost", "api.post.protect.get.app_error", nil, "", http.StatusNotFound)
	}

	if post.Type != model.POST_TYPE_QUESTION {
		return model.NewAppError("ProtectPost", "api.post.protect.get.app_error", nil, "", http.StatusBadRequest)
	}

	// TODO: teamの場合も対応
	if post.TeamId != "" {
		return model.NewAppError("ProtectPost", "api.post.protect.team.app_error", nil, "", http.StatusBadRequest)
	}

	return a.Srv.Store.Post().ProtectPost(postId, model.GetMillis(), userId)
}

func (a *App) CancelProtectPost(postId string, userId string) *model.AppError {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		mlog.Error("Couldn't cancel protect the post", mlog.Err(err))
		return err
	}

	if post == nil {
		return model.NewAppError("CancelProtectPost", "api.post.cancel_protect.get.app_error", nil, "", http.StatusInternalServerError)
	}

	if post.Type != model.POST_TYPE_QUESTION {
		return model.NewAppError("CancelProtectPost", "api.post.cancel_protect.get.app_error", nil, "", http.StatusBadRequest)
	}

	// TODO: teamの場合も対応
	if post.TeamId != "" {
		return model.NewAppError("CancelProtectPost", "api.post.cancel_protect.team.app_error", nil, "", http.StatusBadRequest)
	}

	return a.Srv.Store.Post().CancelProtectPost(postId, userId)
}

func (a *App) DeletePostFiles(post *model.Post) {
	if _, err := a.Srv.Store.FileInfo().DeleteForPost(post.Id); err != nil {
		mlog.Warn("Encountered error when deleting files for post", mlog.String("post_id", post.Id), mlog.Err(err))
	}
}

func (a *App) ViewPost(post *model.Post, userId string, ipAddress string) *model.AppError {
	return a.Srv.Store.Post().ViewPost(post.Id, post.TeamId, userId, ipAddress, 1)
}

func (a *App) RelatedPosts(post *model.Post) ([]*model.RelatedPostSearchResult, *model.AppError) {
	term := post.Title + " " + post.Tags

	return a.Srv.Store.Post().RelatedSearch(term, 10)
}

func (a *App) HotPosts(interval string, teamId string) (model.Posts, *model.AppError) {
	postIds, err := a.Srv.Store.Post().HotSearch(interval, teamId, searchlayer.HOT_POST_SEARCH_MAX_COUNT*2)
	if err != nil {
		return nil, model.NewAppError("HotPosts", "api.post.hot_posts.hot_search.app_error", nil, "", http.StatusInternalServerError)
	}

	posts, err := a.Srv.Store.Post().GetPostsByIds(postIds)
	if err != nil {
		return nil, err
	}

	var results []*model.Post
	for _, postId := range postIds {
		for _, post := range posts {
			if postId == post.Id {
				results = append(results, post)
			}
		}
	}

	return results, nil
}

func (a *App) GetCurrentRevisionForPost(postId string, teamId string) (int64, *model.AppError) {
	return a.Srv.Store.Post().GetCurrentRevisionForPost(postId, teamId)
}

func (a *App) GetRevisionPost(postId string, teamId string, offset int) (*model.Post, *model.AppError) {
	return a.Srv.Store.Post().GetRevisionPost(postId, teamId, offset)
}
