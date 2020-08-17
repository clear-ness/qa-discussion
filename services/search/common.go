package search

import (
	"github.com/clear-ness/qa-discussion/model"
)

// TODO: 構造を定義
// json定義必要？
type ESPost struct {
	Id          string
	Type        string
	ParentId    string
	UserId      string
	TeamId      string
	Title       string
	Content     string
	Tags        []string
	Points      int
	AnswerCount int
	CreateAt    int64
	UpdateAt    int64
	DeleteAt    int64
}

func ESPostFromPost(post *model.Post) *ESPost {
	//p := &model.PostForIndexing{
	//    TeamId: teamId,
	//}
	//post.ShallowCopy(&p.Post)
	return ESPostFromPostForIndexing(post)
}

func ESPostFromPostForIndexing(post *model.Post) *ESPost {
	return &ESPost{
		Id:       post.Id,
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		CreateAt: post.CreateAt,
		Content:  post.Content,
		Type:     post.Type,
		Tags:     strings.Fields(post.Tags),
	}
}
