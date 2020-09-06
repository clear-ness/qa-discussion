package searchlayer

import (
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/search"
	"github.com/clear-ness/qa-discussion/store"
)

type SearchVoteStore struct {
	store.VoteStore
	rootStore *SearchStore
}

func (s SearchVoteStore) IndexVote(vote *model.Vote) {
	esVote := search.ESVoteFromVote(vote)
	go func() {
		s.rootStore.esBackend.IndexESVote(esVote)
	}()
}

func (s *SearchVoteStore) DeleteVote(vote *model.Vote) {
	esVote := search.ESVoteFromVote(vote)
	go func() {
		s.rootStore.esBackend.DeleteESVote(esVote)
	}()
}

func (s *SearchVoteStore) SetupIndex() {
	voteMapping := `{
    "mappings": {
      "_doc": {
        "properties": {
          "post_id":        { "type": "keyword" },
          "user_id":        { "type": "keyword" },
          "type":           { "type": "keyword" },
          "tags":           { "type": "text" },
          "team_id":        { "type": "keyword" },
          "first_post_rev": { "type": "integer" },
          "last_post_rev":  { "type": "integer" },
		  "create_at":      { "type": "date", "format": "epoch_millis" },
		  "invalidate_at":  { "type": "date", "format": "epoch_millis" },
		  "completed_at":   { "type": "date", "format": "epoch_millis" },
		  "completed_by":   { "type": "keyword" },
		  "rejected_at":    { "type": "date", "format": "epoch_millis" },
		  "rejected_by":    { "type": "keyword" },
        }
      }
    }
		}`
	s.rootStore.esBackend.CreateIndex(voteMapping, search.INDEX_NAME_VOTES)
}
