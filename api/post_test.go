package api

import (
	"strconv"
	"strings"
	"testing"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateQuestionPost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	post := &model.Post{}

	_, resp := Client.CreateQuestion(post)
	CheckBadRequestStatus(t, resp)

	post.Title = "title1"
	_, resp = Client.CreateQuestion(post)
	CheckBadRequestStatus(t, resp)

	post.Content = "content1"
	rpost, resp := Client.CreateQuestion(post)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)
	require.Equal(t, post.Content, rpost.Content, "content didn't match")
	require.Equal(t, th.BasicUser.Id, rpost.UserId, "author id didn't match")

	actual, resp := Client.GetPost(rpost.Id)
	CheckNoError(t, resp)
	assert.Equal(t, post.Content, actual.Content, "failed to get post")

	post.ParentId = model.NewId()
	rpost, resp = Client.CreateQuestion(post)
	CheckNoError(t, resp)
	require.Empty(t, rpost.ParentId, "question should have no parent")

	post.Type = model.POST_TYPE_ANSWER
	_, resp = Client.CreateQuestion(post)
	CheckNoError(t, resp)
	require.Equal(t, model.POST_TYPE_QUESTION, rpost.Type, "invalid post type")

	t.Run("set tags", func(t *testing.T) {
		post.Tags = "tag1"
		rpost, resp = Client.CreateQuestion(post)
		CheckNoError(t, resp)
		assert.Equal(t, "tag1", rpost.Tags, "failed to create tags")

		tags := ""
		for i := 0; i < model.MAX_PARSE_TAG_COUNT; i++ {
			tags = tags + "tag" + strconv.Itoa(i)
			if i < (model.MAX_PARSE_TAG_COUNT - 1) {
				tags = tags + " "
			}
		}

		post.Tags = tags
		rpost, resp = Client.CreateQuestion(post)
		CheckNoError(t, resp)
		assert.Equal(t, tags, rpost.Tags, "failed to create tags")

		post.Tags = tags + " tagExtra"
		rpost, resp = Client.CreateQuestion(post)
		CheckBadRequestStatus(t, resp)
	})

	Client.Logout()

	_, resp = Client.CreateQuestion(post)
	CheckUnauthorizedStatus(t, resp)
}

func TestCreateAnswerPost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	post := &model.Post{}

	_, resp := Client.CreateAnswer(post)
	CheckBadRequestStatus(t, resp)

	post.Content = "content1"
	_, resp = Client.CreateAnswer(post)
	CheckBadRequestStatus(t, resp)

	post.ParentId = model.NewId()
	_, resp = Client.CreateAnswer(post)
	CheckNotFoundStatus(t, resp)

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	post.ParentId = rpost.Id
	rpost2, resp := Client.CreateAnswer(post)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)
	require.Equal(t, post.Content, rpost2.Content, "content didn't match")
	require.Equal(t, th.BasicUser.Id, rpost2.UserId, "author id didn't match")

	t.Run("answer's parent is answer", func(t *testing.T) {
		post.ParentId = rpost2.Id
		_, resp = Client.CreateAnswer(post)
		CheckNotFoundStatus(t, resp)
	})

	Client.Logout()

	post.ParentId = rpost.Id
	_, resp = Client.CreateAnswer(post)
	CheckUnauthorizedStatus(t, resp)
}

func TestCreateCommentPost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	post := &model.Post{}

	_, resp := Client.CreateComment(post)
	CheckNotFoundStatus(t, resp)

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "question content"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	t.Run("question's comment", func(t *testing.T) {
		post.ParentId = rpost.Id
		_, resp = Client.CreateComment(post)
		CheckBadRequestStatus(t, resp)

		post.Content = "question's comment content"
		rpost2, resp := Client.CreateComment(post)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)
		require.Equal(t, post.Content, rpost2.Content, "content didn't match")
		require.Equal(t, th.BasicUser.Id, rpost2.UserId, "author id didn't match")
	})

	t.Run("answer's comment", func(t *testing.T) {
		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = "answer content"
		rpost2, resp := Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)
		require.Equal(t, answer.Content, rpost2.Content, "content didn't match")
		require.Equal(t, th.BasicUser.Id, rpost2.UserId, "author id didn't match")

		post.ParentId = rpost2.Id
		post.Content = "answer's comment content"
		_, resp = Client.CreateComment(post)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)
	})

	Client.Logout()

	_, resp = Client.CreateComment(post)
	CheckUnauthorizedStatus(t, resp)
}

func TestSearchPosts(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	_, resp := Client.SearchPosts(make(map[string]string), "", "")
	CheckBadRequestStatus(t, resp)

	requestBody := map[string]string{"post_type": model.POST_TYPE_COMMENT}
	_, resp = Client.SearchPosts(requestBody, "", "")
	CheckBadRequestStatus(t, resp)

	requestBody = map[string]string{"post_type": model.POST_TYPE_QUESTION}
	data, resp := Client.SearchPosts(requestBody, "", "")
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 0, "invalid search")

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "question content"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	t.Run("search question", func(t *testing.T) {
		requestBody = map[string]string{"post_type": model.POST_TYPE_QUESTION}
		data, resp = Client.SearchPosts(requestBody, "", "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")
		require.Equal(t, rpost.Id, data.Posts[0].Id, "searched posts didn't match")
		require.Equal(t, int64(1), data.TotalCount, "searched posts didn't match")

		requestBody = map[string]string{"post_type": model.POST_TYPE_QUESTION, "title": rpost.Title}
		data, resp = Client.SearchPosts(requestBody, "", "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")
		require.Equal(t, rpost.Id, data.Posts[0].Id, "searched posts didn't match")
		require.Equal(t, int64(1), data.TotalCount, "searched posts didn't match")

		question2 := &model.Post{}
		question2.Title = "title2"
		question2.Content = model.NewId()
		question2.Tags = "tag1 tag2 tag3"
		rpost2, resp := Client.CreateQuestion(question2)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		requestBody = map[string]string{"post_type": model.POST_TYPE_QUESTION, "tagged": "tag1"}
		data, resp = Client.SearchPosts(requestBody, "", "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")
		require.Equal(t, data.Posts[0].Id, rpost2.Id, "searched posts didn't match")

		requestBody = map[string]string{"post_type": model.POST_TYPE_QUESTION, "user_id": th.BasicUser.Id}
		data, resp = Client.SearchPosts(requestBody, "", "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 2, "invalid search")
		require.Equal(t, data.Posts[0].Id, rpost2.Id, "searched posts didn't match")
		require.Equal(t, data.Posts[1].Id, rpost.Id, "searched posts didn't match")

		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = "answer content"
		_, resp = Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		requestBody = map[string]string{"post_type": model.POST_TYPE_QUESTION, "user_id": th.BasicUser.Id}
		data, resp = Client.SearchPosts(requestBody, model.POST_SORT_TYPE_ANSWERS, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 2, "invalid search")
		require.Equal(t, data.Posts[0].Id, rpost.Id, "searched posts didn't match")
		require.Equal(t, data.Posts[1].Id, rpost2.Id, "searched posts didn't match")
	})

	t.Run("search answer", func(t *testing.T) {
		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = model.NewId()
		rpost2, resp := Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)
		require.Equal(t, answer.Content, rpost2.Content, "content didn't match")
		require.Equal(t, th.BasicUser.Id, rpost2.UserId, "author id didn't match")

		requestBody = map[string]string{"post_type": model.POST_TYPE_ANSWER}
		data, resp = Client.SearchPosts(requestBody, "", "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 2, "invalid search")
		require.Equal(t, data.Posts[0].Id, rpost2.Id, "searched posts didn't match")
	})
}

func TestGetAnswersForPost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	data, resp := Client.GetAnswersForPost(model.NewId())
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 0, "invalid answers")

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	answer := &model.Post{}
	answer.ParentId = rpost.Id
	answer.Content = "answer content"
	rpost2, resp := Client.CreateAnswer(answer)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	data, resp = Client.GetAnswersForPost(rpost.Id)
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 1, "invalid answers")
	require.Equal(t, rpost2.Id, data.Posts[0].Id, "answer posts didn't match")
	require.Equal(t, th.BasicUser.Id, data.Posts[0].UserId, "author id didn't match")

	t.Run("check up voted", func(t *testing.T) {
		_, resp := Client.UpvotePost(rpost2.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		data, resp = Client.GetAnswersForPost(rpost.Id)
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid answers")
		require.Equal(t, rpost2.Id, data.Posts[0].Id, "answer posts didn't match")
		require.True(t, data.Posts[0].UpVoted, "my voted should be seen")
	})

	t.Run("check down voted", func(t *testing.T) {
		_, resp := Client.DownvotePost(rpost2.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		data, resp = Client.GetAnswersForPost(rpost.Id)
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid answers")
		require.Equal(t, rpost2.Id, data.Posts[0].Id, "answer posts didn't match")
		require.True(t, data.Posts[0].DownVoted, "my voted should be seen")
	})

	t.Run("check flagged", func(t *testing.T) {
		_, resp := Client.FlagPost(rpost2.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		data, resp = Client.GetAnswersForPost(rpost.Id)
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid answers")
		require.Equal(t, rpost2.Id, data.Posts[0].Id, "answer posts didn't match")
		require.True(t, data.Posts[0].Flagged, "my voted should be seen")
	})

	t.Run("get commented answer", func(t *testing.T) {
		comment := &model.Post{}
		comment.ParentId = rpost2.Id
		comment.Content = "answer's comment content"
		rpost3, resp := Client.CreateComment(comment)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		data, resp = Client.GetAnswersForPost(rpost.Id)
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid answers")
		require.Equal(t, rpost2.Id, data.Posts[0].Id, "answer posts didn't match")
		require.Len(t, data.Posts[0].Metadata.Comments, 1, "answer's comments didn't match")
		require.Equal(t, rpost3.Id, data.Posts[0].Metadata.Comments[0].Id, "answer's comments didn't match")
		require.Equal(t, th.BasicUser.Id, data.Posts[0].Metadata.Comments[0].UserId, "author id didn't match")

		Client.Logout()
		th.LoginBasic2()

		comment = &model.Post{}
		comment.ParentId = rpost2.Id
		comment.Content = "answer's comment content2"
		rpost4, resp := Client.CreateComment(comment)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		Client.Logout()

		data, resp = Client.GetAnswersForPost(rpost.Id)
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid answers")
		require.Equal(t, rpost2.Id, data.Posts[0].Id, "answer posts didn't match")
		require.Len(t, data.Posts[0].Metadata.Comments, 2, "answer's comments didn't match")
		require.Equal(t, rpost4.Id, data.Posts[0].Metadata.Comments[0].Id, "answer's comments didn't match")
		require.Equal(t, th.BasicUser2.Id, data.Posts[0].Metadata.Comments[0].UserId, "author id didn't match")
		require.Equal(t, rpost3.Id, data.Posts[0].Metadata.Comments[1].Id, "answer's comments didn't match")
		require.Equal(t, th.BasicUser.Id, data.Posts[0].Metadata.Comments[1].UserId, "author id didn't match")
	})
}

func TestGetPost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	_, resp := Client.GetPost(model.NewId())
	CheckNotFoundStatus(t, resp)

	question := &model.Post{}
	question.Title = "title1"
	question.Content = model.NewId()
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	post, resp := Client.GetPost(rpost.Id)
	CheckNoError(t, resp)
	assert.Equal(t, question.Content, post.Content, "failed to get post")

	t.Run("check up voted", func(t *testing.T) {
		_, resp := Client.UpvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp = Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		require.Equal(t, rpost.Id, post.Id, "question posts didn't match")
		require.True(t, post.UpVoted, "my voted should be seen")
		require.Empty(t, post.DownVoted, "shold not be seen voted")
		require.Empty(t, post.Flagged, "shold not be seen voted")
	})

	t.Run("check down voted", func(t *testing.T) {
		_, resp := Client.DownvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp = Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		require.Equal(t, rpost.Id, post.Id, "question posts didn't match")
		require.True(t, post.UpVoted, "my voted should be seen")
		require.True(t, post.DownVoted, "my voted should be seen")
		require.Empty(t, post.Flagged, "shold not be seen voted")
	})

	t.Run("check flagged", func(t *testing.T) {
		_, resp := Client.FlagPost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp = Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		require.Equal(t, rpost.Id, post.Id, "question posts didn't match")
		require.True(t, post.UpVoted, "my voted should be seen")
		require.True(t, post.DownVoted, "my voted should be seen")
		require.True(t, post.Flagged, "my voted should be seen")
	})

	Client.Logout()
	th.LoginBasic2()

	comment := &model.Post{}
	comment.ParentId = rpost.Id
	comment.Content = "question's comment content"
	rpost2, resp := Client.CreateComment(comment)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	post, resp = Client.GetPost(rpost.Id)
	CheckNoError(t, resp)
	assert.Equal(t, question.Content, post.Content, "failed to get post")
	require.Len(t, post.Metadata.Comments, 1, "answer's comments didn't match")
	require.Equal(t, rpost2.Id, post.Metadata.Comments[0].Id, "answer's comments didn't match")
	require.Equal(t, th.BasicUser2.Id, post.Metadata.Comments[0].UserId, "author id didn't match")

	th.LoginBasic()

	t.Run("get my favorite post", func(t *testing.T) {
		_, resp = Client.FavoritePost(rpost.Id)
		CheckNoError(t, resp)

		post, resp = Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, question.Content, post.Content, "failed to get post")
		require.True(t, post.Favorited, "not favorited")
		assert.Equal(t, int64(1), post.FavoriteCount, "favorite count doesn't match")
	})

	Client.Logout()

	post, resp = Client.GetPost(rpost.Id)
	CheckNoError(t, resp)
	assert.Equal(t, question.Content, post.Content, "failed to get post")
	require.Empty(t, post.Favorited, "shold not favorited")
	assert.Equal(t, int64(1), post.FavoriteCount, "favorite count doesn't match")
}

func TestUpdatePost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	t.Run("update question post", func(t *testing.T) {
		post := &model.Post{
			Type:    model.POST_TYPE_QUESTION,
			Content: question.Content,
			Title:   question.Title,
		}
		_, resp := Client.UpdatePost(rpost.Id, post)
		CheckBadRequestStatus(t, resp)

		post.Id = rpost.Id
		uppost, resp := Client.UpdatePost(rpost.Id, post)
		CheckNoError(t, resp)
		assert.Equal(t, int64(0), uppost.EditAt, "EditAt should be zero")

		post.Content = "updated content"
		uppost, resp = Client.UpdatePost(rpost.Id, post)
		CheckNoError(t, resp)
		assert.Equal(t, post.Content, uppost.Content, "failed to updates")
		assert.NotEqual(t, int64(0), uppost.EditAt, "EditAt should not be zero")
		editAt1 := uppost.EditAt

		post.Title = "updated title"
		uppost, resp = Client.UpdatePost(rpost.Id, post)
		CheckNoError(t, resp)
		assert.Equal(t, post.Title, uppost.Title, "failed to updates")
		assert.NotEqual(t, int64(0), uppost.EditAt, "EditAt should not be zero")
		editAt2 := uppost.EditAt
		require.LessOrEqual(t, editAt1, editAt2, "EditAt should be updated")

		post.Tags = "tag1"
		uppost, resp = Client.UpdatePost(rpost.Id, post)
		CheckNoError(t, resp)
		uppost, resp = Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, "tag1", uppost.Tags, "failed to updates")
		editAt3 := uppost.EditAt
		require.LessOrEqual(t, editAt2, editAt3, "EditAt should be updated")

		post.Tags = ""
		uppost, resp = Client.UpdatePost(rpost.Id, post)
		CheckNoError(t, resp)
		uppost, resp = Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, "", uppost.Tags, "failed to updates")

		tags := ""
		for i := 0; i < model.MAX_PARSE_TAG_COUNT; i++ {
			tags = tags + "tag" + strconv.Itoa(i)
			if i < (model.MAX_PARSE_TAG_COUNT - 1) {
				tags = tags + " "
			}
		}

		post.Tags = tags + " tagExtra"
		uppost, resp = Client.UpdatePost(rpost.Id, post)
		CheckBadRequestStatus(t, resp)
	})

	t.Run("update answer post", func(t *testing.T) {
		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = "answer content"
		rpost2, resp := Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		post := &model.Post{
			Id:      rpost2.Id,
			Content: answer.Content,
		}
		_, resp = Client.UpdatePost(rpost2.Id, post)
		CheckBadRequestStatus(t, resp)

		post.Type = model.POST_TYPE_ANSWER
		uppost, resp := Client.UpdatePost(rpost2.Id, post)
		CheckNoError(t, resp)
		assert.Equal(t, int64(0), uppost.EditAt, "EditAt should be zero")

		post.Content = "updated content"
		uppost, resp = Client.UpdatePost(rpost2.Id, post)
		CheckNoError(t, resp)
		assert.Equal(t, post.Content, uppost.Content, "failed to updates")
		assert.NotEqual(t, int64(0), uppost.EditAt, "EditAt should not be zero")
	})

	t.Run("update comment post", func(t *testing.T) {
		comment := &model.Post{}
		comment.ParentId = rpost.Id
		comment.Content = "question's comment content"
		rpost2, resp := Client.CreateComment(comment)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		post := &model.Post{
			Id:      rpost2.Id,
			Content: comment.Content,
		}
		_, resp = Client.UpdatePost(rpost2.Id, post)
		CheckBadRequestStatus(t, resp)

		post.Type = model.POST_TYPE_COMMENT
		uppost, resp := Client.UpdatePost(rpost2.Id, post)
		CheckNoError(t, resp)
		assert.Equal(t, int64(0), uppost.EditAt, "EditAt should be zero")

		post.Content = model.NewId()
		uppost, resp = Client.UpdatePost(rpost2.Id, post)
		CheckNoError(t, resp)
		assert.Equal(t, post.Content, uppost.Content, "failed to updates")
		assert.NotEqual(t, int64(0), uppost.EditAt, "EditAt should not be zero")
	})

	t.Run("update other user's post", func(t *testing.T) {
		Client.Logout()
		th.LoginBasic2()

		question2 := &model.Post{}
		question2.Title = "title2"
		question2.Content = "content2"
		rpost, resp := Client.CreateQuestion(question2)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		th.LoginBasic()

		post := &model.Post{
			Id:      rpost.Id,
			Type:    model.POST_TYPE_QUESTION,
			Content: question2.Content,
			Title:   question2.Title,
		}
		_, resp = Client.UpdatePost(rpost.Id, post)
		CheckForbiddenStatus(t, resp)
	})
}

func TestUpvoteDownvotePost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	post, resp := Client.GetPost(rpost.Id)
	CheckNoError(t, resp)
	assert.Equal(t, 0, post.UpVotes, "post upvotes count should be zero initially")

	Client.Logout()
	th.LoginBasic2()

	t.Run("upvote question post", func(t *testing.T) {
		_, resp := Client.UpvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp := Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 1, post.UpVotes, "post upvotes count invalid")
		assert.Equal(t, 0, post.DownVotes, "post downvotes count invalid")
		assert.Equal(t, 1, post.Points, "post points count invalid")
	})

	t.Run("cancel upvote question post", func(t *testing.T) {
		_, resp := Client.CancelUpvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp := Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 0, post.UpVotes, "post upvotes count invalid")
		assert.Equal(t, 0, post.DownVotes, "post downvotes count invalid")
		assert.Equal(t, 0, post.Points, "post points count invalid")
	})

	t.Run("downvote question post", func(t *testing.T) {
		_, resp := Client.DownvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp := Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 0, post.UpVotes, "post upvotes count invalid")
		assert.Equal(t, 1, post.DownVotes, "post downvotes count invalid")
		assert.Equal(t, -1, post.Points, "post points count invalid")
	})

	t.Run("cancel downvote question post", func(t *testing.T) {
		_, resp := Client.CancelDownvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp := Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 0, post.UpVotes, "post upvotes count invalid")
		assert.Equal(t, 0, post.DownVotes, "post downvotes count invalid")
		assert.Equal(t, 0, post.Points, "post points count invalid")
	})

	t.Run("upvote answer post", func(t *testing.T) {
		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = "answer content"
		rpost2, resp := Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		post, resp := Client.GetPost(rpost2.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 0, post.UpVotes, "post upvotes count should be zero initially")

		_, resp = Client.UpvotePost(rpost2.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp = Client.GetPost(rpost2.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 1, post.UpVotes, "post upvotes count invalid")
		assert.Equal(t, 0, post.DownVotes, "post downvotes count invalid")
		assert.Equal(t, 1, post.Points, "post points count invalid")
	})

	Client.Logout()

	_, resp = Client.UpvotePost(rpost.Id)
	CheckUnauthorizedStatus(t, resp)
}

func TestFlagPost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	post, resp := Client.GetPost(rpost.Id)
	CheckNoError(t, resp)
	assert.Equal(t, 0, post.FlagCount, "post flag count should be zero initially")

	Client.Logout()
	th.LoginBasic2()

	t.Run("flag & cancel flag question post", func(t *testing.T) {
		_, resp := Client.FlagPost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp := Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 1, post.FlagCount, "post flag count should be updated")

		_, resp = Client.CancelFlagPost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		post, resp = Client.GetPost(rpost.Id)
		CheckNoError(t, resp)
		assert.Equal(t, 0, post.FlagCount, "post flag count should be updated")
	})

	Client.Logout()

	_, resp = Client.FlagPost(rpost.Id)
	CheckUnauthorizedStatus(t, resp)
}

func TestGetQuestionsForUser(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	data, resp := Client.GetQuestionsForUser(model.NewId())
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 0, "invalid questions")

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	data, resp = Client.GetQuestionsForUser(th.BasicUser.Id)
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 1, "invalid questions")
	assert.Equal(t, rpost.Id, data.Posts[0].Id, "failed to get post")
	require.Equal(t, int64(1), data.TotalCount, "invalid questions")

	question2 := &model.Post{}
	question2.Title = "title2"
	question2.Content = "content2"
	rpost2, resp := Client.CreateQuestion(question2)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()

	data, resp = Client.GetQuestionsForUser(th.BasicUser.Id)
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 2, "invalid questions")
	assert.Equal(t, rpost2.Id, data.Posts[0].Id, "failed to get post")
	assert.Equal(t, rpost.Id, data.Posts[1].Id, "failed to get post")
	require.Equal(t, int64(2), data.TotalCount, "invalid questions")
}

func TestGetAnswersForUser(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	data, resp := Client.GetAnswersForUser(model.NewId())
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 0, "invalid questions")

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

	data, resp = Client.GetAnswersForUser(th.BasicUser2.Id)
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 1, "invalid answers")
	assert.Equal(t, rpost2.Id, data.Posts[0].Id, "failed to get post")
	require.Equal(t, int64(1), data.TotalCount, "invalid answers")

	answer = &model.Post{}
	answer.ParentId = rpost.Id
	answer.Content = "answer content2"
	rpost3, resp := Client.CreateAnswer(answer)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()

	data, resp = Client.GetAnswersForUser(th.BasicUser2.Id)
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 2, "invalid answers")
	assert.Equal(t, rpost3.Id, data.Posts[0].Id, "failed to get post")
	assert.Equal(t, rpost2.Id, data.Posts[1].Id, "failed to get post")
	require.Equal(t, int64(2), data.TotalCount, "invalid answers")
}

func TestAdvancedSearchQuestionPosts(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	_, resp := Client.AdvancedSearchPosts(make(map[string]string), "")
	CheckBadRequestStatus(t, resp)

	terms := ""
	requestBody := map[string]string{"terms": terms}

	t.Run("invalid terms length", func(t *testing.T) {
		_, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckBadRequestStatus(t, resp)

		terms = strings.Repeat("A", model.POST_SEARCH_TERMS_MAX+1)
		requestBody["terms"] = terms
		_, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckBadRequestStatus(t, resp)
	})

	t.Run("invalid sort type", func(t *testing.T) {
		terms = "term"
		requestBody["terms"] = terms
		_, resp = Client.AdvancedSearchPosts(requestBody, model.POST_SORT_TYPE_ANSWERS)
		CheckBadRequestStatus(t, resp)
	})

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	question.Tags = "tag1 tag2 tag3"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	t.Run("search a question", func(t *testing.T) {
		terms = "title:" + "fake" + question.Title
		requestBody["terms"] = terms
		data, resp := Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")
		require.Equal(t, int64(0), data.TotalCount, "searched posts didn't match")

		terms = "title:" + question.Title
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")
		require.Equal(t, rpost.Id, data.Posts[0].Id, "searched posts didn't match")
		require.Equal(t, int64(1), data.TotalCount, "searched posts didn't match")

		terms = terms + " " + "-title:" + "fake" + question.Title
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		// with question's specific search property, forced for question's search
		appendTerms := " " + "is:" + model.POST_TYPE_ANSWER
		terms = terms + appendTerms
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = strings.Replace(terms, appendTerms, " "+"is:"+model.POST_TYPE_QUESTION, 1)
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = terms + " " + "body:" + question.Content
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = terms + " " + "\"" + question.Content + "\""
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = terms + " " + "#" + strings.Fields(question.Tags)[0]
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		appendTerms = " " + "from:" + "2050-1-1"
		terms = terms + appendTerms
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")

		terms = strings.Replace(terms, appendTerms, " "+"from:"+"2010-01-01", 1)
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		appendTerms = " " + "to:" + "2010-01-01"
		terms = terms + appendTerms
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")

		terms = strings.Replace(terms, appendTerms, " "+"to:"+"2050-01-01", 1)
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = terms + " " + "user:" + th.BasicUser.Id
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = terms + " " + "minvotes:" + "0"
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		_, resp = Client.DownvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")

		_, resp = Client.CancelDownvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		terms = terms + " " + "maxvotes:" + "0"
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		_, resp = Client.UpvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")

		_, resp = Client.CancelUpvotePost(rpost.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		appendTerms = " " + "minanswers:" + "1"
		terms = terms + appendTerms
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")

		terms = strings.Replace(terms, appendTerms, " "+"minanswers:"+"0", 1)
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		appendTerms = " " + "maxanswers:" + "0"
		terms = terms + appendTerms
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = "answer content"
		_, resp = Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")

		terms = strings.Replace(terms, appendTerms, " "+"maxanswers:"+"1", 1)
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		appendTerms = " " + "-#" + strings.Fields(question.Tags)[1]
		terms = terms + appendTerms
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")
		require.Equal(t, int64(0), data.TotalCount, "searched posts didn't match")

		terms = strings.Replace(terms, appendTerms, "", 1)
		terms = strings.Replace(terms, "#"+strings.Fields(question.Tags)[0], "#"+"fake"+strings.Fields(question.Tags)[0], 1)
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")
		require.Equal(t, int64(0), data.TotalCount, "searched posts didn't match")
	})
}

func TestAdvancedSearchAnswerPosts(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	terms := ""
	requestBody := map[string]string{"terms": terms}

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	question.Tags = "tag1 tag2 tag3"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	t.Run("search an answer", func(t *testing.T) {
		terms = "is:" + model.POST_TYPE_ANSWER
		requestBody["terms"] = terms
		data, resp := Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 0, "invalid search")

		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = "answer content"
		rpost2, resp := Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")
		require.Equal(t, rpost2.Id, data.Posts[0].Id, "searched posts didn't match")
		require.Equal(t, int64(1), data.TotalCount, "searched posts didn't match")

		terms = terms + " " + "body:" + answer.Content
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = terms + " " + "\"" + answer.Content + "\""
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		terms = terms + " " + "inquestion:" + rpost.Id
		requestBody["terms"] = terms
		data, resp = Client.AdvancedSearchPosts(requestBody, "")
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 1, "invalid search")

		answer2 := &model.Post{}
		answer2.ParentId = rpost.Id
		answer2.Content = "answer content"
		rpost3, resp := Client.CreateAnswer(answer2)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		_, resp = Client.UpvotePost(rpost2.Id)
		CheckNoError(t, resp)
		CheckOKStatus(t, resp)

		data, resp = Client.AdvancedSearchPosts(requestBody, model.POST_SORT_TYPE_CREATION)
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 2, "invalid search")
		require.Equal(t, rpost3.Id, data.Posts[0].Id, "searched posts didn't match")
		require.Equal(t, rpost2.Id, data.Posts[1].Id, "searched posts didn't match")
		require.Equal(t, int64(2), data.TotalCount, "searched posts didn't match")

		data, resp = Client.AdvancedSearchPosts(requestBody, model.POST_SORT_TYPE_VOTES)
		CheckNoError(t, resp)
		require.Len(t, data.Posts, 2, "invalid search")
		require.Equal(t, rpost2.Id, data.Posts[0].Id, "searched posts didn't match")
		require.Equal(t, rpost3.Id, data.Posts[1].Id, "searched posts didn't match")
		require.Equal(t, int64(2), data.TotalCount, "searched posts didn't match")
	})
}

func TestAdvancedSearchPosts(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	terms := ""
	requestBody := map[string]string{"terms": terms}

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	question.Tags = "tag1 tag2 tag3"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	answer := &model.Post{}
	answer.ParentId = rpost.Id
	answer.Content = question.Content
	rpost2, resp := Client.CreateAnswer(answer)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	terms = terms + " " + "body:" + question.Content
	requestBody["terms"] = terms
	data, resp := Client.AdvancedSearchPosts(requestBody, "")
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 2, "invalid search")
	require.Equal(t, rpost2.Id, data.Posts[0].Id, "searched posts didn't match")
	require.Equal(t, rpost.Id, data.Posts[1].Id, "searched posts didn't match")
	require.Equal(t, int64(2), data.TotalCount, "searched posts didn't match")

	// special case for search by post ids
	appendTerms := " " + "is:" + model.POST_TYPE_QUESTION + " " + "inquestion:" + rpost.Id
	terms = terms + appendTerms
	requestBody["terms"] = terms
	data, resp = Client.AdvancedSearchPosts(requestBody, "")
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 1, "invalid search")
	require.Equal(t, rpost.Id, data.Posts[0].Id, "searched posts didn't match")

	terms = strings.Replace(terms, appendTerms, "", 1)

	// with question's specific search property, forced for question's search
	terms = terms + " " + "minanswers:" + "0"
	requestBody["terms"] = terms
	data, resp = Client.AdvancedSearchPosts(requestBody, "")
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 1, "invalid search")
	require.Equal(t, rpost.Id, data.Posts[0].Id, "searched posts didn't match")

	terms = terms + " " + "minanswers:" + "2"
	requestBody["terms"] = terms
	data, resp = Client.AdvancedSearchPosts(requestBody, "")
	CheckNoError(t, resp)
	require.Len(t, data.Posts, 0, "invalid search")
}
