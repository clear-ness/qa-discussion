package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils"
)

const (
	TOP_USERS_OR_POSTS_BY_TAG_FOR_USER = "user"
	TOP_USERS_OR_POSTS_BY_TAG_FOR_POST = "post"
)

type ESUserPointHistory struct {
	Id       string `json:"id"`
	TeamId   string `json:"team_id"`
	UserId   string `json:"user_id"`
	Type     string `json:"type"`
	PostId   string `json:"post_id"`
	PostType string `json:"post_type"`
	Tags     string `json:"tags"`
	Points   int    `json:"points"`
	CreateAt int64  `json:"create_at"`
}

func ESUserPointHistoryFromObj(history *model.UserPointHistory) *ESUserPointHistory {
	return &ESUserPointHistory{
		Id:       history.Id,
		TeamId:   history.TeamId,
		UserId:   history.UserId,
		Type:     history.Type,
		PostId:   history.PostId,
		PostType: history.PostType,
		Tags:     history.Tags,
		Points:   history.Points,
		CreateAt: history.CreateAt,
	}
}

func (b *ESBackend) IndexESUserPointHistory(item *ESUserPointHistory) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return b.Indexing(payload, item.Id, INDEX_NAME_USER_POINT_HISTORY)
}

type TopUserOrPostByTag struct {
	UserId      string
	PostId      string
	DocCount    int
	TotalPoints int
}

type TopUsersOrPostsByTag struct {
	Total int                   `json:"total"`
	Hits  []*TopUserOrPostByTag `json:"hits"`
}

// most asker per tag:
// select userId, sum(points) from UserPointHistory where type in (~) and postType = 'question' and tags like 'tag1%' and createAt > 0 and createAt < 1000 group by userId
//
// most answerer per tag:
// select userId, sum(points) from UserPointHistory where type in (~) and postType = 'answer' and tags like 'tag1%' and createAt > 0 and createAt < 1000 group by userId
//
// hot answers per tag:
// select postId, sum(points) from UserPointHistory where type in (~) and postType = 'answer' and tags like 'tag1%' and createAt > 0 and createAt < 1000 group by postId
func (b *ESBackend) TopUsersByTag(interval string, teamId string, postType string, tag string, limit int) (*TopUsersOrPostsByTag, error) {
	return b.topUsersOrPostsByTag(interval, teamId, postType, tag, limit, TOP_USERS_OR_POSTS_BY_TAG_FOR_USER)
}

func (b *ESBackend) TopPostsByTag(interval string, teamId string, postType string, tag string, limit int) (*TopUsersOrPostsByTag, error) {
	return b.topUsersOrPostsByTag(interval, teamId, postType, tag, limit, TOP_USERS_OR_POSTS_BY_TAG_FOR_POST)
}

func (b *ESBackend) topUsersOrPostsByTag(interval string, teamId string, postType string, tag string, limit int, groupBy string) (*TopUsersOrPostsByTag, error) {
	var results TopUsersOrPostsByTag

	curTime := model.GetMillis()

	var gte int64
	switch interval {
	case model.USER_POINT_HISTORY_INTERVAL_DAY:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 1 * time.Hour))
	case model.USER_POINT_HISTORY_INTERVAL_WEEK:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 7 * time.Hour))
	case model.USER_POINT_HISTORY_INTERVAL_MONTH:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 30 * time.Hour))
	default:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 1 * time.Hour))
	}

	// TODO: teamIdが空の場合を考慮
	filter := []map[string]interface{}{
		map[string]interface{}{
			"range": map[string]interface{}{
				"create_at": map[string]interface{}{
					"gte": gte,
					"lte": curTime,
				},
			},
		},
		map[string]interface{}{
			"term": map[string]interface{}{
				"team_id": teamId,
			},
		},
		map[string]interface{}{
			"term": map[string]interface{}{
				"post_type": postType,
			},
		},
		map[string]interface{}{
			"match": map[string]interface{}{
				"tags": tag,
			},
		},
		map[string]interface{}{
			"terms": map[string]interface{}{
				"type": []string{model.USER_POINT_TYPE_VOTED, model.USER_POINT_TYPE_VOTED_CANCELED, model.USER_POINT_TYPE_DOWN_VOTED, model.USER_POINT_TYPE_DOWN_VOTED_CANCELED},
			},
		},
	}

	var groupByField string
	if groupBy == TOP_USERS_OR_POSTS_BY_TAG_FOR_POST {
		groupByField = "post_id"
	} else if groupBy == TOP_USERS_OR_POSTS_BY_TAG_FOR_USER {
		groupByField = "user_id"
	}

	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filter,
			},
		},
		"aggs": map[string]interface{}{
			"group_by_id": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": groupByField,
					"size":  limit,
					"order": map[string]interface{}{
						"total_points": "desc",
					},
				},
				"aggs": map[string]interface{}{
					"total_points": map[string]interface{}{
						"sum": map[string]interface{}{
							"field": "points",
						},
					},
				},
			},
		},
		"size": limit,
	}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := b.es.Search(
		b.es.Search.WithIndex(INDEX_NAME_USER_POINT_HISTORY),
		b.es.Search.WithBody(&buf),
	)
	if err != nil {
		return &results, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return &results, err
		}
		return &results, fmt.Errorf("[%s] %s: %s", res.Status(), e["error"].(map[string]interface{})["type"], e["error"].(map[string]interface{})["reason"])
	}

	type envelopeResponse struct {
		Took int
		Hits struct {
			Total struct {
				Value int
			}
			Hits []struct {
				Id     string          `json:"_id"`
				Source json.RawMessage `json:"_source"`
			}
		}
		Aggregations struct {
			GroupById struct {
				Buckets []struct {
					Key         string `json:"key"`
					DocCount    int    `json:"doc_count"`
					TotalPoints struct {
						Value int `json:"value"`
					}
				}
			}
		}
	}

	var r envelopeResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return &results, err
	}

	results.Total = len(r.Aggregations.GroupById.Buckets)

	if len(r.Hits.Hits) <= 0 {
		results.Hits = []*TopUserOrPostByTag{}
		return &results, nil
	}

	for _, bucket := range r.Aggregations.GroupById.Buckets {
		var h TopUserOrPostByTag
		if groupBy == TOP_USERS_OR_POSTS_BY_TAG_FOR_POST {
			h.PostId = bucket.Key
		} else if groupBy == TOP_USERS_OR_POSTS_BY_TAG_FOR_USER {
			h.UserId = bucket.Key
		}
		h.DocCount = bucket.DocCount
		h.TotalPoints = bucket.TotalPoints.Value

		results.Hits = append(results.Hits, &h)
	}

	return &results, nil
}
