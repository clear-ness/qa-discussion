package api

import (
	"testing"

	"github.com/clear-ness/qa-discussion/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFavoritePost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	_, resp := Client.FavoritePost(model.NewId())
	CheckForbiddenStatus(t, resp)

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()
	th.LoginBasic2()

	t.Run("try to favorite answer post", func(t *testing.T) {
		answer := &model.Post{}
		answer.ParentId = rpost.Id
		answer.Content = "answer content"
		rpost2, resp := Client.CreateAnswer(answer)
		CheckNoError(t, resp)
		CheckCreatedStatus(t, resp)

		_, resp = Client.FavoritePost(rpost2.Id)
		CheckForbiddenStatus(t, resp)
	})

	t.Run("favorite question post", func(t *testing.T) {
		_, resp = Client.FavoritePost(rpost.Id)
		CheckNoError(t, resp)

		data, resp := Client.GetUserFavoritePosts(model.ME)
		CheckNoError(t, resp)
		require.Len(t, data.UserFavoritePosts, 1, "invalid posts")
		assert.Equal(t, rpost.Id, data.UserFavoritePosts[0].Post.Id, "failed to get post")
		assert.Equal(t, int64(1), data.TotalCount, "failed to get total count")
	})

	Client.Logout()

	_, resp = Client.FavoritePost(rpost.Id)
	CheckUnauthorizedStatus(t, resp)
}

func TestCancelFavoritePost(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	_, resp := Client.CancelFavoritePost(model.NewId())
	CheckForbiddenStatus(t, resp)

	question := &model.Post{}
	question.Title = "title1"
	question.Content = "content1"
	rpost, resp := Client.CreateQuestion(question)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.Logout()
	th.LoginBasic2()

	_, resp = Client.FavoritePost(rpost.Id)
	CheckNoError(t, resp)

	data, resp := Client.GetUserFavoritePosts(model.ME)
	CheckNoError(t, resp)
	require.Len(t, data.UserFavoritePosts, 1, "invalid posts")
	assert.Equal(t, rpost.Id, data.UserFavoritePosts[0].Post.Id, "failed to get post")
	assert.Equal(t, int64(1), data.TotalCount, "failed to get total count")

	pass, resp := Client.CancelFavoritePost(rpost.Id)
	CheckNoError(t, resp)
	require.True(t, pass, "should cancel favorite post")

	data, resp = Client.GetUserFavoritePosts(model.ME)
	CheckNoError(t, resp)
	require.Len(t, data.UserFavoritePosts, 0, "invalid posts")
	assert.Equal(t, int64(0), data.TotalCount, "failed to get total count")

	Client.Logout()

	_, resp = Client.CancelFavoritePost(rpost.Id)
	CheckUnauthorizedStatus(t, resp)
}

func TestGetUserFavoritePosts(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()
	Client := th.Client

	data, resp := Client.GetUserFavoritePosts(model.ME)
	CheckNoError(t, resp)
	require.Len(t, data.UserFavoritePosts, 0, "invalid posts")

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

	_, resp = Client.FavoritePost(rpost2.Id)
	CheckNoError(t, resp)

	_, resp = Client.FavoritePost(rpost.Id)
	CheckNoError(t, resp)

	data, resp = Client.GetUserFavoritePosts(model.ME)
	CheckNoError(t, resp)
	require.Len(t, data.UserFavoritePosts, 2, "invalid posts")
	assert.Equal(t, int64(2), data.TotalCount, "failed to get total count")
	assert.Equal(t, rpost.Id, data.UserFavoritePosts[0].Post.Id, "failed to get post")
	assert.Equal(t, rpost2.Id, data.UserFavoritePosts[1].Post.Id, "failed to get post")

	Client.Logout()

	_, resp = Client.GetUserFavoritePosts(model.ME)
	CheckBadRequestStatus(t, resp)
}
