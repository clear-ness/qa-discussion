package searchlayer

import (
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/search"
	"github.com/clear-ness/qa-discussion/store"
)

type SearchPostViewsHistoryStore struct {
	store.PostViewsHistoryStore
	rootStore *SearchStore
}

func (s SearchPostViewsHistoryStore) IndexPostViewsHistory(history *model.PostViewsHistory) {
	esHistory := search.ESPostViewsHistoryFromObj(history)
	go func() {
		s.rootStore.esBackend.IndexESPostViewsHistory(esHistory)
	}()
}

func (s *SearchPostViewsHistoryStore) SetupIndex() {
	historyMapping := `{
    "mappings": {
      "_doc": {
        "properties": {
          "id":           { "type": "keyword" },
          "post_id":      { "type": "keyword" },
          "team_id":      { "type": "keyword" },
          "user_id":      { "type": "keyword" },
          "ip_address":   { "type": "text" },
          "views_count":  { "type": "integer" },
		  "create_at":    { "type": "date", "format": "epoch_millis" },
        }
      }
    }
		}`
	s.rootStore.esBackend.CreateIndex(historyMapping, search.INDEX_NAME_POST_VIEWS_HISTORY)
}
