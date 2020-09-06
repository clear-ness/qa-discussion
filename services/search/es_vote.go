package search

import (
	"encoding/json"

	"github.com/clear-ness/qa-discussion/model"
)

type ESVote struct {
	PostId   string `json:"post_id"`
	UserId   string `json:"user_id"`
	Type     string `json:"type"`
	TeamId   string `json:"team_id"`
	CreateAt int64  `json:"create_at"`
}

func ESVoteFromVote(vote *model.Vote) *ESVote {
	return &ESVote{
		PostId:   vote.PostId,
		UserId:   vote.UserId,
		Type:     vote.Type,
		TeamId:   vote.TeamId,
		CreateAt: vote.CreateAt,
	}
}

func (b *ESBackend) IndexESVote(item *ESVote) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return b.Indexing(payload, item.UserId+item.Type+item.PostId, INDEX_NAME_VOTES)
}

func (b *ESBackend) DeleteESVote(item *ESVote) error {
	return b.DeleteIndex(item.UserId+item.Type+item.PostId, INDEX_NAME_VOTES)
}
