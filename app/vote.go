package app

import (
	"net/http"
	"sort"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) GetVotesForUser(toDate int64, userId string, page, perPage int, excludeFlag bool, limitContent bool, teamId string) ([]*model.VoteWithPost, int64, *model.AppError) {
	votes, totalCount, err := a.Srv.Store.Vote().GetVotesBeforeTime(toDate, userId, page, perPage, excludeFlag, true, teamId)
	if err != nil {
		return nil, 0, err
	}

	postIdsMaps := map[string]bool{}
	for _, vote := range votes {
		postIdsMaps[vote.PostId] = true
	}

	var postIds []string
	for key := range postIdsMaps {
		postIds = append(postIds, key)
	}

	posts, err := a.Srv.Store.Post().GetPostsByIds(postIds)
	if err != nil {
		return nil, 0, err
	}

	option := model.SetPostMetadataOptions{
		SetUser:       true,
		SetComments:   false,
		SetBestAnswer: false,
		SetParent:     true,
	}
	posts, err = a.SetPostMetadata(posts, option)
	if err != nil {
		return nil, 0, err
	}

	if limitContent {
		posts.LimitContentLength()
	}

	postMap := map[string]*model.Post{}
	for _, post := range posts {
		postMap[post.Id] = post
	}

	var votesWithPost []*model.VoteWithPost
	for _, vote := range votes {
		if post, ok := postMap[vote.PostId]; ok {
			vpost := &model.VoteWithPost{
				vote,
				post,
			}
			votesWithPost = append(votesWithPost, vpost)
		}
	}

	sort.Slice(votesWithPost, func(i, j int) bool {
		return votesWithPost[i].CreateAt > votesWithPost[j].CreateAt
	})

	return votesWithPost, totalCount, nil
}

func (a *App) GetVote(userId string, postId string, voteType string) (*model.Vote, *model.AppError) {
	// TODO: teamIdの指定は不要？
	// インデックスの影響で必要そう..
	return a.Srv.Store.Vote().GetByPostIdForUser(userId, postId, voteType)
}

func (a *App) GetVoteTypesForPost(userId string, postId string) ([]string, *model.AppError) {
	// TODO: teamIdの指定は不要？
	// インデックスの影響で必要そう..
	return a.Srv.Store.Vote().GetVoteTypesForPost(userId, postId)
}

func (a *App) CreateReviewVote(postId string, userId string, tagContents string) (*model.Vote, *model.AppError) {
	post, err := a.Srv.Store.Post().GetSingle(postId, false)
	if err != nil {
		return nil, err
	}

	currentRevision, err := a.GetCurrentRevisionForPost(post.Id, "")

	count, err := a.Srv.Store.Vote().GetRejectedReviewsCount(postId, currentRevision)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, model.NewAppError("CreateReviewVote", "api.review.create_review_vote.rejected.app_error", nil, "", http.StatusBadRequest)
	}

	options := &model.GetTagsOptions{Type: model.TAG_TYPE_REVIEW}

	tags := strings.Fields(tagContents)
	for _, tag := range tags {
		options.Content = tag
		_, err := a.GetTags(options)
		if err != nil {
			return nil, err
		}
	}

	vote, err := a.Srv.Store.Vote().CreateReviewVote(post, userId, tagContents, currentRevision)
	if err != nil {
		return nil, err
	}

	return vote, nil
}

func (a *App) RejectReviewsForPost(postId string, rejectedBy string) *model.AppError {
	rev, err := a.Srv.Store.Post().GetCurrentRevisionForPost(postId, "")
	if err != nil {
		return err
	}

	return a.Srv.Store.Vote().RejectReviewsForPost(postId, rejectedBy, rev)
}

func (a *App) CompleteReviewsForPost(post *model.Post, completedBy string) *model.AppError {
	if _, err := a.DeletePostForcely(post, completedBy); err != nil {
		return err
	}

	rev, err := a.Srv.Store.Post().GetCurrentRevisionForPost(post.Id, "")
	if err != nil {
		return err
	}

	return a.Srv.Store.Vote().CompleteReviewsForPost(post.Id, completedBy, rev)
}

func (a *App) GetReviews(options *model.SearchReviewsOptions, getCount bool) ([]*model.Vote, int64, *model.AppError) {
	return a.Srv.Store.Vote().GetReviews(options, getCount)
}
