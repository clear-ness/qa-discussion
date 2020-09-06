package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/utils"
)

type ESPost struct {
	Id          string `json:"id"`
	Type        string `json:"type"`
	ParentId    string `json:"parent_id"`
	UserId      string `json:"user_id"`
	TeamId      string `json:"team_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Tags        string `json:"tags"`
	Points      int    `json:"points"`
	AnswerCount int    `json:"answer_count"`
	Views       int    `json:"views"`
	CreateAt    int64  `json:"create_at"`
	UpdateAt    int64  `json:"update_at"`
	DeleteAt    int64  `json:"delete_at"`
}

func ESPostFromPost(post *model.Post) *ESPost {
	return &ESPost{
		Id:          post.Id,
		Type:        post.Type,
		ParentId:    post.ParentId,
		UserId:      post.UserId,
		TeamId:      post.TeamId,
		Title:       post.Title,
		Content:     post.Content,
		Tags:        post.Tags,
		Points:      post.Points,
		AnswerCount: post.AnswerCount,
		Views:       post.Views,
		CreateAt:    post.CreateAt,
		UpdateAt:    post.UpdateAt,
		DeleteAt:    post.DeleteAt,
	}
}

type ESPostSearchResults struct {
	Total int       `json:"total"`
	Hits  []*ESPost `json:"hits"`
}

func (b *ESBackend) IndexESPost(item *ESPost) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return b.Indexing(payload, item.Id, INDEX_NAME_POSTS)
}

func (b *ESBackend) DeleteESPost(item *ESPost) error {
	return b.DeleteIndex(item.Id, INDEX_NAME_POSTS)
}

// postViewsHistory, vote, answerのES検索結果に利用される
type HotESPostSearchResult struct {
	PostId   string
	DocCount int
}

type HotESPostSearchResults struct {
	Total int                      `json:"total"`
	Hits  []*HotESPostSearchResult `json:"hits"`
}

func (b *ESBackend) RelatedESPosts(term string, limit int) (*ESPostSearchResults, error) {
	var results ESPostSearchResults

	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"more_like_this": map[string]interface{}{
				"fields":        []string{"title", "tags", "content"},
				"like":          term,
				"min_term_freq": 1,
				//"max_query_terms": 12,
			},
		},
		"size": limit,
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := b.es.Search(
		b.es.Search.WithIndex(INDEX_NAME_POSTS),
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
	}

	var r envelopeResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return &results, err
	}

	results.Total = r.Hits.Total.Value

	if len(r.Hits.Hits) < 1 {
		results.Hits = []*ESPost{}
		return &results, nil
	}

	for _, hit := range r.Hits.Hits {
		var h ESPost
		h.Id = hit.Id

		// これでESPostの各属性がhに付与される
		if err := json.Unmarshal(hit.Source, &h); err != nil {
			return &results, err
		}

		results.Hits = append(results.Hits, &h)
	}

	return &results, nil
}

func (b *ESBackend) HotESVotes(interval string, teamId string, limit int) (*HotESPostSearchResults, error) {
	return b.HotESPosts(INDEX_NAME_VOTES, interval, teamId, limit)
}

func (b *ESBackend) HotESAnswerPosts(interval string, teamId string, limit int) (*HotESPostSearchResults, error) {
	return b.HotESPosts(INDEX_NAME_POSTS, interval, teamId, limit)
}

func (b *ESBackend) HotESPosts(indexName string, interval string, teamId string, limit int) (*HotESPostSearchResults, error) {
	// TODO: yesterdayから、を考慮
	curTime := model.GetMillis()

	var gte int64
	switch interval {
	case model.HOT_POSTS_INTERVAL_DAYS:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 3 * time.Hour))
	case model.HOT_POSTS_INTERVAL_WEEK:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 7 * time.Hour))
	case model.HOT_POSTS_INTERVAL_MONTH:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 30 * time.Hour))
	default:
		gte = utils.MillisFromTime(time.Now().Add(-24 * 3 * time.Hour))
	}

	var results HotESPostSearchResults

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
	}

	var groupByField string

	if indexName == INDEX_NAME_POST_VIEWS_HISTORY {
		groupByField = "post_id"
	} else if indexName == INDEX_NAME_VOTES {
		typeTerm := map[string]interface{}{
			"term": map[string]interface{}{
				"type": model.VOTE_TYPE_UP_VOTE,
			},
		}
		filter = append(filter, typeTerm)

		groupByField = "post_id"
	} else if indexName == INDEX_NAME_POSTS {
		typeTerm := map[string]interface{}{
			"term": map[string]interface{}{
				"type": model.POST_TYPE_ANSWER,
			},
		}
		filter = append(filter, typeTerm)

		groupByField = "parent_id"
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
				},
			},
		},
		"size": limit,
	}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := b.es.Search(
		b.es.Search.WithIndex(indexName),
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
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
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
		results.Hits = []*HotESPostSearchResult{}
		return &results, nil
	}

	for _, bucket := range r.Aggregations.GroupById.Buckets {
		var h HotESPostSearchResult
		h.PostId = bucket.Key
		h.DocCount = bucket.DocCount

		results.Hits = append(results.Hits, &h)
	}

	return &results, nil
}
