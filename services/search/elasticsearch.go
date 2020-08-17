package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/clear-ness/qa-discussion/model"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

const (
	INDEX_NAME_POSTS = "posts"
)

// question and answerの様な、親子関係のドキュメントをインデックスする場合:
// https://www.elastic.co/guide/en/elasticsearch/reference/7.x/parent-join.html
//
// 実装:
// Post.TeamId毎にESのシャード決定ルーティングを指定すると良さそう。
// (jobで)複数のpostをインデックシングする場合、bulk index APIを使うと良さそう。
type ESBackend struct {
	es        *elasticsearch.Client
	indexName string
}

type ESPostSearchResults struct {
	Total int          `json:"total"`
	Hits  []*ESPostHit `json:"hits"`
}

type ESPostHit struct {
	ESPost
	URL        string        `json:"url"`
	Sort       []interface{} `json:"sort"`
	Highlights *struct {
		Title      []string `json:"title"`
		Alt        []string `json:"alt"`
		Transcript []string `json:"transcript"`
	} `json:"highlights,omitempty"`
}

func NewESBackend(settings *model.SearchSettings, indexName string) (*ESBackend, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{*settings.SearchEndpoint},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	s := ESBackend{es: es, indexName: indexName}

	return &s, nil
}

// インデックス(rdbで言うデータベースに相当)をmapping付きで作成する
func (b *ESBackend) CreateIndex(mapping string) error {
	res, err := b.es.Indices.Exists([]string{b.indexName})
	if err != nil {
		return err
	}

	// A 404 means it does not exist, and 200 means it does.
	if res.StatusCode != 200 {
		res, err = b.es.Indices.Create(b.indexName, b.es.Indices.Create.WithBody(strings.NewReader(mapping)))
		if err != nil {
			return err
		}
	}

	return nil
}

// Create indexes a new document into store.
func (b *ESBackend) IndexESPost(item *ESPost) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}

	ctx := context.Background()
	res, err := esapi.CreateRequest{
		Index:      b.indexName,
		DocumentID: item.Id,
		Body:       bytes.NewReader(payload),
	}.Do(ctx, b.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return err
		}
		return fmt.Errorf("[%s] %s: %s", res.Status(), e["error"].(map[string]interface{})["type"], e["error"].(map[string]interface{})["reason"])
	}

	return nil
}

func (b *ESBackend) DeleteESPost(item *ESPost) error {
	ctx := context.Background()
	res, err := esapi.DeleteRequest{
		Index:      b.indexName,
		DocumentID: item.Id,
	}.Do(ctx, b.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return err
		}
		return fmt.Errorf("[%s] %s: %s", res.Status(), e["error"].(map[string]interface{})["type"], e["error"].(map[string]interface{})["reason"])
	}

	return nil
}

// Search returns results matching a query, paginated by after.
func (b *ESBackend) SearchESPosts(query string, after ...string) (*ESPostSearchResults, error) {
	var results ESPostSearchResults

	const searchMatch = `
	"query" : {
		"multi_match" : {
			"query" : %q,
			"fields" : ["title^100", "alt^10", "transcript"],
			"operator" : "and"
		}
	},
	"highlight" : {
		"fields" : {
			"title" : { "number_of_fragments" : 0 },
			"alt" : { "number_of_fragments" : 0 },
			"transcript" : { "number_of_fragments" : 5, "fragment_size" : 25 }
		}
	},
	"size" : 25,
	"sort" : [ { "_score" : "desc" }, { "_doc" : "asc" } ]`

	// ES検索
	res, err := b.es.Search(
		b.es.Search.WithIndex(b.indexName),
		b.es.Search.WithBody(b.buildQuery(query, searchMatch, after...)),
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
				ID         string          `json:"_id"`
				Source     json.RawMessage `json:"_source"`
				Highlights json.RawMessage `json:"highlight"`
				Sort       []interface{}   `json:"sort"`
			}
		}
	}

	var r envelopeResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return &results, err
	}

	results.Total = r.Hits.Total.Value

	if len(r.Hits.Hits) < 1 {
		results.Hits = []*ESPostHit{}
		return &results, nil
	}

	for _, hit := range r.Hits.Hits {
		var h ESPostHit
		h.Id = hit.Id
		h.Sort = hit.Sort
		h.URL = strings.Join([]string{baseURL, h.ID, ""}, "/")

		if err := json.Unmarshal(hit.Source, &h); err != nil {
			return &results, err
		}

		if len(hit.Highlights) > 0 {
			if err := json.Unmarshal(hit.Highlights, &h.Highlights); err != nil {
				return &results, err
			}
		}

		results.Hits = append(results.Hits, &h)
	}

	return &results, nil
}

func (b *ESBackend) buildQuery(query string, searchMatch string, after ...string) io.Reader {
	var builder strings.Builder

	builder.WriteString("{\n")
	builder.WriteString(fmt.Sprintf(searchMatch, query))

	if len(after) > 0 && after[0] != "" && after[0] != "null" {
		builder.WriteString(",\n")
		builder.WriteString(fmt.Sprintf(`	"search_after": %s`, after))
	}

	builder.WriteString("\n}")

	// fmt.Printf("%s\n", builder.String())
	return strings.NewReader(builder.String())
}
