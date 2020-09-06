package search

import (
	"encoding/json"

	"github.com/clear-ness/qa-discussion/model"
)

type ESPostViewsHistory struct {
	Id         string `json:"id"`
	PostId     string `json:"post_id"`
	TeamId     string `json:"team_id"`
	UserId     string `json:"user_id"`
	IpAddress  string `json:"ip_address"`
	ViewsCount int    `json:"views_count"`
	CreateAt   int64  `json:"create_at"`
}

func ESPostViewsHistoryFromObj(history *model.PostViewsHistory) *ESPostViewsHistory {
	return &ESPostViewsHistory{
		Id:         history.Id,
		PostId:     history.PostId,
		TeamId:     history.TeamId,
		UserId:     history.UserId,
		IpAddress:  history.IpAddress,
		ViewsCount: history.ViewsCount,
		CreateAt:   history.CreateAt,
	}
}

func (b *ESBackend) IndexESPostViewsHistory(item *ESPostViewsHistory) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return b.Indexing(payload, item.Id, INDEX_NAME_POST_VIEWS_HISTORY)
}

func (b *ESBackend) HotESPostViewsHistory(interval string, teamId string, limit int) (*HotESPostSearchResults, error) {
	return b.HotESPosts(INDEX_NAME_POST_VIEWS_HISTORY, interval, teamId, limit)
}
