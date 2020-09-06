package searchlayer

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/search"
	"github.com/clear-ness/qa-discussion/store"
)

type SearchUserPointHistoryStore struct {
	store.UserPointHistoryStore
	rootStore *SearchStore
}

func (s SearchUserPointHistoryStore) IndexUserPointHistory(history *model.UserPointHistory) {
	esHistory := search.ESUserPointHistoryFromObj(history)
	go func() {
		s.rootStore.esBackend.IndexESUserPointHistory(esHistory)
	}()
}

func (s *SearchUserPointHistoryStore) TopAskersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopUserByTagResult, *model.AppError) {
	results, err := s.rootStore.esBackend.TopUsersByTag(interval, teamId, model.POST_TYPE_QUESTION, tag, limit)
	if err != nil {
		return nil, model.NewAppError("SearchUserPointHistoryStore.TopAskersByTag", "search_user_point_history.top_askers_by_tag.search.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if results.Total <= 0 {
		return []*model.TopUserByTagResult{}, nil
	}

	res := []*model.TopUserByTagResult{}
	for _, hit := range results.Hits {
		r := &model.TopUserByTagResult{
			UserId:     hit.UserId,
			TotalScore: hit.TotalPoints,
		}

		res = append(res, r)
	}

	return res, nil
}

func (s *SearchUserPointHistoryStore) TopAnswerersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopUserByTagResult, *model.AppError) {
	results, err := s.rootStore.esBackend.TopUsersByTag(interval, teamId, model.POST_TYPE_ANSWER, tag, limit)
	if err != nil {
		return nil, model.NewAppError("SearchUserPointHistoryStore.TopAnswerersByTag", "search_user_point_history.top_answerers_by_tag.search.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if results.Total <= 0 {
		return []*model.TopUserByTagResult{}, nil
	}

	res := []*model.TopUserByTagResult{}
	for _, hit := range results.Hits {
		r := &model.TopUserByTagResult{
			UserId:     hit.UserId,
			TotalScore: hit.TotalPoints,
		}

		res = append(res, r)
	}

	return res, nil
}

func (s *SearchUserPointHistoryStore) TopAnswersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopPostByTagResult, *model.AppError) {
	results, err := s.rootStore.esBackend.TopPostsByTag(interval, teamId, model.POST_TYPE_ANSWER, tag, limit)
	if err != nil {
		return nil, model.NewAppError("SearchUserPointHistoryStore.TopAnswersByTag", "search_user_point_history.top_answers_by_tag.search.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if results.Total <= 0 {
		return []*model.TopPostByTagResult{}, nil
	}

	res := []*model.TopPostByTagResult{}
	for _, hit := range results.Hits {
		r := &model.TopPostByTagResult{
			PostId:     hit.PostId,
			TotalScore: hit.TotalPoints,
		}

		res = append(res, r)
	}

	return res, nil
}

func (s *SearchUserPointHistoryStore) SetupIndex() {
	historyMapping := `{
    "mappings": {
      "_doc": {
        "properties": {
          "id":           { "type": "keyword" },
          "user_id":      { "type": "keyword" },
          "team_id":      { "type": "keyword" },
          "type":         { "type": "keyword" },
          "post_id":      { "type": "keyword" },
          "post_type":    { "type": "keyword" },
          "tags":         { "type": "text" },
          "points":       { "type": "integer" },
		  "create_at":    { "type": "date", "format": "epoch_millis" },
        }
      }
    }
		}`
	s.rootStore.esBackend.CreateIndex(historyMapping, search.INDEX_NAME_USER_POINT_HISTORY)
}
