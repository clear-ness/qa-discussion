package api

import (
	"regexp"
	"testing"

	"github.com/clear-ness/qa-discussion/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	user := &model.User{
		Email:    th.GenerateTestEmail(),
		Username: GenerateTestUsername(),
		Password: "Password1@",
	}

	ruser, resp := Client.CreateUser(user)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	ruser, resp = Client.GetUser(ruser.Id)
	require.Equal(t, user.Username, ruser.Username, "username didn't match")

	_, _ = Client.Login(user.Email, user.Password)

	_, resp = Client.CreateUser(ruser)
	CheckBadRequestStatus(t, resp)
}

func TestVerifyUserEmail(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	email := th.GenerateTestEmail()
	user := &model.User{
		Email:    email,
		Username: GenerateTestUsername(),
		Password: "hello1",
	}

	_, _ = Client.CreateUser(user)

	_, resp := Client.VerifyUserEmail(model.NewId())
	CheckBadRequestStatus(t, resp)

	_, resp = Client.VerifyUserEmail("")
	CheckBadRequestStatus(t, resp)
}

func TestSendVerificationEmail(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	pass, resp := Client.SendVerificationEmail(th.BasicUser.Email)
	CheckNoError(t, resp)

	require.True(t, pass, "should have passed")

	_, resp = Client.SendVerificationEmail("")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.SendVerificationEmail(th.GenerateTestEmail())
	CheckNoError(t, resp)

	Client.Logout()
	_, resp = Client.SendVerificationEmail(th.BasicUser.Email)
	CheckNoError(t, resp)
}

func TestLogin(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	t.Run("missing password", func(t *testing.T) {
		_, resp := Client.Login(th.BasicUser.Email, "")
		CheckBadRequestStatus(t, resp)
	})

	t.Run("unknown user", func(t *testing.T) {
		_, resp := Client.Login("unknown", th.BasicUser.Password)
		CheckBadRequestStatus(t, resp)
	})

	t.Run("valid login", func(t *testing.T) {
		user, resp := Client.Login(th.BasicUser.Email, th.BasicUser.Password)
		CheckNoError(t, resp)
		assert.Equal(t, user.Id, th.BasicUser.Id)
	})

	t.Run("should return cookies with X-Requested-With header", func(t *testing.T) {
		Client.HttpHeader[model.HEADER_REQUESTED_WITH] = model.HEADER_REQUESTED_WITH_XML

		user, resp := Client.Login(th.BasicUser.Email, th.BasicUser.Password)

		sessionCookie := ""
		userCookie := ""
		csrfCookie := ""

		for _, cookie := range resp.Header["Set-Cookie"] {
			if match := regexp.MustCompile("^" + model.SESSION_COOKIE_TOKEN + "=([a-z0-9]+)").FindStringSubmatch(cookie); match != nil {
				sessionCookie = match[1]
			} else if match := regexp.MustCompile("^" + model.SESSION_COOKIE_USER + "=([a-z0-9]+)").FindStringSubmatch(cookie); match != nil {
				userCookie = match[1]
			} else if match := regexp.MustCompile("^" + model.SESSION_COOKIE_CSRF + "=([a-z0-9]+)").FindStringSubmatch(cookie); match != nil {
				csrfCookie = match[1]
			}
		}

		session, _ := th.App.GetSession(Client.AuthToken)

		assert.Equal(t, Client.AuthToken, sessionCookie)
		assert.Equal(t, user.Id, userCookie)
		assert.Equal(t, session.GetCSRF(), csrfCookie)
	})
}

func TestGetUser(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	ruser, resp := Client.GetUser(th.BasicUser.Id)
	CheckNoError(t, resp)
	assert.Equal(t, th.BasicUser.Id, ruser.Id)
	assert.Equal(t, int64(0), ruser.QuestionCount)
	assert.Equal(t, int64(0), ruser.AnswerCount)

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	ruser, resp = Client.GetUser(th.BasicUser.Id)
	CheckNoError(t, resp)
	assert.Equal(t, th.BasicUser.Id, ruser.Id)
	assert.Equal(t, int64(0), ruser.AnswerCount)
	assert.Equal(t, int64(1), ruser.QuestionCount)

	answer := &model.Post{}
	answer.ParentId = rpost.Id
	answer.Content = "answer content"
	_, resp = Client.CreateAnswer(answer)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()

	ruser, resp = Client.GetUser(th.BasicUser.Id)
	CheckNoError(t, resp)
	assert.Equal(t, th.BasicUser.Id, ruser.Id)
	assert.Equal(t, int64(1), ruser.AnswerCount)
	assert.Equal(t, int64(1), ruser.QuestionCount)
}

func TestGetInboxMessagesForUser(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	messages, resp := Client.GetInboxMessagesForUser(model.ME)
	require.Len(t, messages, 0, "inbox messages should be empty initially")
	CheckNoError(t, resp)

	count, resp := Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckNoError(t, resp)
	assert.Equal(t, 0, count)

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()
	th.LoginBasic2()

	answer := &model.Post{}
	answer.ParentId = rpost.Id
	answer.Content = "answer content"
	rpost2, resp := Client.CreateAnswer(answer)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()
	th.LoginBasic()

	messages, resp = Client.GetInboxMessagesForUser(model.ME)
	require.Len(t, messages, 1, "inbox messages should be updated")
	CheckNoError(t, resp)
	assert.Equal(t, model.INBOX_MESSAGE_TYPE_ANSWER, messages[0].Type)
	assert.Equal(t, answer.Content, messages[0].Content)
	assert.Equal(t, th.BasicUser.Id, messages[0].UserId)
	assert.Equal(t, th.BasicUser2.Id, messages[0].SenderId)
	assert.Equal(t, rpost.Id, messages[0].QuestionId)
	assert.Equal(t, rpost2.Id, messages[0].AnswerId)
	assert.Equal(t, question.Title, messages[0].Title)
	assert.Equal(t, question.Title, messages[0].Title)
	require.True(t, messages[0].IsUnread, "should be unread")

	count, resp = Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckNoError(t, resp)
	assert.Equal(t, 1, count)

	comment := &model.Post{}
	comment.ParentId = rpost2.Id
	comment.Content = "answer's comment content"
	rpost3, resp := Client.CreateComment(comment)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	messages, resp = Client.GetInboxMessagesForUser(model.ME)
	require.Len(t, messages, 1, "inbox messages should not be updated")
	CheckNoError(t, resp)

	reply1 := "@" + th.BasicUser.Username
	reply2 := "@" + th.BasicUser2.Username
	comment = &model.Post{}
	comment.ParentId = rpost2.Id
	comment.Content = reply1 + " " + reply2 + " hello"
	rpost4, resp := Client.CreateComment(comment)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	messages, resp = Client.GetInboxMessagesForUser(model.ME)
	require.Len(t, messages, 2, "inbox messages should be updated")
	CheckNoError(t, resp)
	assert.Equal(t, model.INBOX_MESSAGE_TYPE_COMMENT_REPLY, messages[0].Type)
	assert.Equal(t, rpost4.Id, messages[0].CommentId)
	assert.Equal(t, rpost2.Id, messages[1].AnswerId)
	assert.Equal(t, comment.Content, messages[0].Content)
	assert.Equal(t, th.BasicUser.Id, messages[0].UserId)
	assert.Equal(t, th.BasicUser.Id, messages[0].SenderId)

	count, resp = Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckNoError(t, resp)
	assert.Equal(t, 2, count)

	Client.Logout()
	th.LoginBasic2()

	messages, resp = Client.GetInboxMessagesForUser(model.ME)
	require.Len(t, messages, 2, "inbox messages should be updated")
	CheckNoError(t, resp)
	assert.Equal(t, model.INBOX_MESSAGE_TYPE_COMMENT_REPLY, messages[0].Type)
	assert.Equal(t, rpost4.Id, messages[0].CommentId)
	assert.Equal(t, comment.Content, messages[0].Content)
	assert.Equal(t, th.BasicUser2.Id, messages[0].UserId)
	assert.Equal(t, th.BasicUser.Id, messages[0].SenderId)
	assert.Equal(t, model.INBOX_MESSAGE_TYPE_COMMENT, messages[1].Type)
	assert.Equal(t, rpost3.Id, messages[1].CommentId)
	assert.Equal(t, th.BasicUser2.Id, messages[1].UserId)
	assert.Equal(t, th.BasicUser.Id, messages[1].SenderId)

	count, resp = Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckNoError(t, resp)
	assert.Equal(t, 2, count)

	Client.Logout()

	_, resp = Client.GetInboxMessagesForUser(model.ME)
	CheckUnauthorizedStatus(t, resp)

	_, resp = Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckUnauthorizedStatus(t, resp)
}

func TestSetInboxMessageRead(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	pass, resp := Client.SetInboxMessageRead(model.ME, model.NewId())
	CheckNotFoundStatus(t, resp)
	require.False(t, pass, "should not set read")

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()
	th.LoginBasic2()

	answer := &model.Post{}
	answer.ParentId = rpost.Id
	answer.Content = "answer content"
	_, resp = Client.CreateAnswer(answer)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()
	th.LoginBasic()

	messages, resp := Client.GetInboxMessagesForUser(model.ME)
	require.Len(t, messages, 1, "inbox messages should be updated")
	CheckNoError(t, resp)

	count, resp := Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckNoError(t, resp)
	assert.Equal(t, 1, count)

	pass, resp = Client.SetInboxMessageRead(model.ME, messages[0].Id)
	CheckNoError(t, resp)
	require.True(t, pass, "should set read")

	count, resp = Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckNoError(t, resp)
	assert.Equal(t, 0, count)

	_, resp = Client.SetInboxMessageRead(model.ME, messages[0].Id)
	CheckBadRequestStatus(t, resp)

	count, resp = Client.GetInboxMessagesUnreadCountForUser(model.ME)
	CheckNoError(t, resp)
	assert.Equal(t, 0, count)

	Client.Logout()

	_, resp = Client.SetInboxMessageRead(model.ME, messages[0].Id)
	CheckUnauthorizedStatus(t, resp)
}

func TestGetUserVotes(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	data, resp := Client.GetUserVotes(model.ME)
	CheckNoError(t, resp)
	require.Len(t, data.Votes, 0, "invalid posts")

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	question2 := &model.Post{}
	question2.Title = "title2"
	question2.Content = "content2"
	rpost2, resp := Client.CreateQuestion(question2)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()
	th.LoginBasic2()

	_, resp = Client.UpvotePost(rpost.Id)
	CheckNoError(t, resp)

	_, resp = Client.UpvotePost(rpost2.Id)
	CheckNoError(t, resp)

	data, resp = Client.GetUserVotes(model.ME)
	CheckNoError(t, resp)
	require.Len(t, data.Votes, 2, "invalid posts")
	assert.Equal(t, int64(2), data.TotalCount, "failed to get total count")
	assert.Equal(t, rpost2.Id, data.Votes[0].Post.Id, "failed to get post")
	assert.Equal(t, rpost.Id, data.Votes[1].Post.Id, "failed to get post")
	assert.Equal(t, model.VOTE_TYPE_UP_VOTE, data.Votes[0].Vote.Type, "failed to get vote type")
	assert.Equal(t, model.VOTE_TYPE_UP_VOTE, data.Votes[1].Vote.Type, "failed to get vote type")

	answer := &model.Post{}
	answer.ParentId = rpost.Id
	answer.Content = "answer content"
	rpost3, resp := Client.CreateAnswer(answer)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	_, resp = Client.DownvotePost(rpost3.Id)
	CheckNoError(t, resp)

	_, resp = Client.FlagPost(rpost3.Id)
	CheckNoError(t, resp)

	data, resp = Client.GetUserVotes(model.ME)
	CheckNoError(t, resp)
	require.Len(t, data.Votes, 3, "invalid posts")
	assert.Equal(t, int64(3), data.TotalCount, "failed to get total count")
	assert.Equal(t, rpost3.Id, data.Votes[0].Post.Id, "failed to get post")
	assert.Equal(t, rpost2.Id, data.Votes[1].Post.Id, "failed to get post")
	assert.Equal(t, rpost.Id, data.Votes[2].Post.Id, "failed to get post")
	assert.Equal(t, model.VOTE_TYPE_DOWN_VOTE, data.Votes[0].Vote.Type, "failed to get vote type")
	assert.Equal(t, model.VOTE_TYPE_UP_VOTE, data.Votes[1].Vote.Type, "failed to get vote type")
	assert.Equal(t, model.VOTE_TYPE_UP_VOTE, data.Votes[2].Vote.Type, "failed to get vote type")

	Client.Logout()

	_, resp = Client.GetUserFavoritePosts(model.ME)
	CheckBadRequestStatus(t, resp)
}
