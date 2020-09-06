package search

import (
	"bytes"
	"context"
	"strings"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

const (
	INDEX_NAME_POSTS              = "posts"
	INDEX_NAME_POST_VIEWS_HISTORY = "post_views_history"
	INDEX_NAME_USER_POINT_HISTORY = "user_point_history"
	INDEX_NAME_VOTES              = "votes"
)

type ESBackend struct {
	es *elasticsearch.Client
}

func NewESBackend(settings *model.SearchSettings) (*ESBackend, *error) {
	cfg := elasticsearch.Config{
		Addresses: []string{*settings.SearchEndpoint},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, &err
	}

	s := ESBackend{es: es}

	return &s, nil
}

// インデックス(rdbで言うデータベースに相当)をmapping付きで作成する
func (b *ESBackend) CreateIndex(mapping string, indexName string) error {
	res, err := b.es.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}

	// A 404 means it does not exist, and 200 means it does.
	if res.StatusCode != 200 {
		res, err = b.es.Indices.Create(indexName, b.es.Indices.Create.WithBody(strings.NewReader(mapping)))
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO: (jobで定期的に？)もはや検索範囲外になったindexing達を削除
func (b *ESBackend) Indexing(payload []byte, id string, indexName string) error {
	ctx := context.Background()
	res, err := esapi.CreateRequest{
		Index:      indexName,
		DocumentID: id,
		Body:       bytes.NewReader(payload),
	}.Do(ctx, b.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func (b *ESBackend) DeleteIndex(itemId string, indexName string) error {
	ctx := context.Background()
	res, err := esapi.DeleteRequest{
		Index:      indexName,
		DocumentID: itemId,
	}.Do(ctx, b.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
