package app

import (
	"sort"

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
