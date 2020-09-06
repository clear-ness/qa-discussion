package searchlayer

import (
	"net/http"
	"sort"
	"sync"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/services/search"
	"github.com/clear-ness/qa-discussion/store"
)

const (
	HOT_POST_SEARCH_MAX_COUNT = 10
)

type SearchPostStore struct {
	store.PostStore
	rootStore *SearchStore
}

func (s SearchPostStore) IndexPost(post *model.Post) {
	esPost := search.ESPostFromPost(post)
	go func() {
		s.rootStore.esBackend.IndexESPost(esPost)
	}()
}

func (s *SearchPostStore) SaveQuestion(post *model.Post) (*model.Post, *model.AppError) {
	post, err := s.PostStore.SaveQuestion(post)
	if err == nil {
		s.IndexPost(post)
	}

	return post, err
}

func (s *SearchPostStore) SaveAnswer(post *model.Post) (*model.Post, *model.AppError) {
	post, err := s.PostStore.SaveAnswer(post)
	if err == nil {
		s.IndexPost(post)
	}

	return post, err
}

func (s *SearchPostStore) SaveUserPointHistory(history *model.UserPointHistory) (*model.UserPointHistory, *model.AppError) {
	history, err := s.PostStore.SaveUserPointHistory(history)
	if err == nil {
		s.rootStore.userPointHistory.IndexUserPointHistory(history)
	}

	return history, err
}

func (s *SearchPostStore) SavePostViewsHistory(postId string, teamId string, userId string, ipAddress string, count int, time int64) (*model.PostViewsHistory, *model.AppError) {
	history, err := s.PostStore.SavePostViewsHistory(postId, teamId, userId, ipAddress, count, time)
	if err == nil {
		s.rootStore.postViewsHistory.IndexPostViewsHistory(history)
	}

	return history, err
}

func (s *SearchPostStore) Update(newPost *model.Post, oldPost *model.Post) (*model.Post, *model.AppError) {
	post, err := s.PostStore.Update(newPost, oldPost)
	if err == nil {
		s.IndexPost(post)
	}

	return post, err
}

func (s *SearchPostStore) DeletePost(post *model.Post) {
	esPost := search.ESPostFromPost(post)
	go func() {
		s.rootStore.esBackend.DeleteESPost(esPost)
	}()
}

func (s *SearchPostStore) DeleteQuestion(postId string, time int64, deleteById string) *model.AppError {
	err := s.PostStore.DeleteQuestion(postId, time, deleteById)
	if err == nil {
		post, err := s.PostStore.GetSingle(postId, true)
		if err != nil {
			return err
		}

		s.DeletePost(post)
	}

	return err
}

func (s *SearchPostStore) DeleteAnswer(postId string, time int64, deleteById string) *model.AppError {
	err := s.PostStore.DeleteAnswer(postId, time, deleteById)
	if err == nil {
		post, err := s.PostStore.GetSingle(postId, true)
		if err != nil {
			return err
		}

		s.DeletePost(post)
	}

	return err
}

func (s *SearchPostStore) UpVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.UpVotePost(postId, userId)
	if err == nil {
		s.rootStore.vote.IndexVote(vote)
	}

	return vote, err
}

func (s *SearchPostStore) CancelUpVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.CancelUpVotePost(postId, userId)
	if err == nil {
		s.rootStore.vote.DeleteVote(vote)
	}

	return vote, err
}

func (s *SearchPostStore) DownVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.DownVotePost(postId, userId)
	if err == nil {
		s.rootStore.vote.IndexVote(vote)
	}

	return vote, err
}

func (s *SearchPostStore) CancelDownVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.CancelDownVotePost(postId, userId)
	if err == nil {
		s.rootStore.vote.DeleteVote(vote)
	}

	return vote, err
}

func (s *SearchPostStore) FlagPost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.FlagPost(postId, userId)
	if err == nil {
		s.rootStore.vote.IndexVote(vote)
	}

	return vote, err
}

func (s *SearchPostStore) CancelFlagPost(postId string, userId string) (*model.Vote, *model.AppError) {
	vote, err := s.PostStore.CancelFlagPost(postId, userId)
	if err == nil {
		s.rootStore.vote.DeleteVote(vote)
	}

	return vote, err
}

func (s *SearchPostStore) RelatedSearch(term string, limit int) ([]*model.RelatedPostSearchResult, *model.AppError) {
	results, err := s.rootStore.esBackend.RelatedESPosts(term, limit)

	if err != nil {
		return nil, model.NewAppError("SearchPostStore.RelatedSearch", "search_post.related_search.search.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if results.Total <= 0 {
		return []*model.RelatedPostSearchResult{}, nil
	}

	res := []*model.RelatedPostSearchResult{}
	for _, espost := range results.Hits {
		r := &model.RelatedPostSearchResult{
			Id:          espost.Id,
			TeamId:      espost.TeamId,
			Title:       espost.Title,
			AnswerCount: espost.AnswerCount,
		}

		res = append(res, r)
	}

	return res, nil
}

func (s *SearchPostStore) HotSearch(interval string, teamId string, limit int) ([]string, *model.AppError) {
	// TODO: 子供及び孫であるcacheLayer、sqlStoreのメソッドを実装しておく。
	// まずそれら子レイヤーから取得を試み、取得出来なかった場合はこれ以降で処理を進める。
	//err := s.PostStore.HotSearch(term, limit)

	var wg sync.WaitGroup
	pchan := make(chan store.StoreResult, 3)

	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := s.rootStore.esBackend.HotESPostViewsHistory(interval, teamId, limit)

		if err != nil {
			appErr := model.NewAppError("SearchPostStore.HotSearch", "store.search_post.hot_search.views.app_error", nil, err.Error(), http.StatusInternalServerError)
			pchan <- store.StoreResult{Data: results, Err: appErr}
			return
		}

		pchan <- store.StoreResult{Data: results, Err: nil}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := s.rootStore.esBackend.HotESVotes(interval, teamId, limit)

		if err != nil {
			appErr := model.NewAppError("SearchPostStore.HotSearch", "store.search_post.hot_search.votes.app_error", nil, err.Error(), http.StatusInternalServerError)
			pchan <- store.StoreResult{Data: results, Err: appErr}
			return
		}

		pchan <- store.StoreResult{Data: results, Err: nil}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		results, err := s.rootStore.esBackend.HotESAnswerPosts(interval, teamId, limit)
		if err != nil {
			appErr := model.NewAppError("SearchPostStore.HotSearch", "store.search_post.hot_search.answers.app_error", nil, err.Error(), http.StatusInternalServerError)
			pchan <- store.StoreResult{Data: results, Err: appErr}
			return
		}

		pchan <- store.StoreResult{Data: results, Err: nil}
	}()

	wg.Wait()
	close(pchan)

	scoreMap := make(map[string]float64)

	for result := range pchan {
		if result.Err != nil {
			return nil, result.Err
		}

		data := result.Data.(search.HotESPostSearchResults)

		totalDocCount := 0
		for _, h := range data.Hits {
			totalDocCount = totalDocCount + h.DocCount
		}

		for _, h := range data.Hits {
			scoreMap[h.PostId] = scoreMap[h.PostId] + float64(h.DocCount)/float64(totalDocCount)
		}
	}

	type score struct {
		postId string
		score  float64
	}
	var scores []score
	for k, v := range scoreMap {
		scores = append(scores, score{k, v})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if len(scores) > HOT_POST_SEARCH_MAX_COUNT {
		scores = scores[:HOT_POST_SEARCH_MAX_COUNT]
	}

	var postIds []string
	for _, score := range scores {
		postIds = append(postIds, score.postId)
	}

	return postIds, nil
}

func (s *SearchPostStore) SetupIndex() {
	postMapping := `{
    "mappings": {
      "_doc": {
        "properties": {
          "id":           { "type": "keyword" },
          "type":         { "type": "keyword" },
          "parent_id":    { "type": "keyword" },
          "user_id":      { "type": "keyword" },
          "team_id":      { "type": "keyword" },
          "title":        { "type": "text" },
          "content":      { "type": "text" },
          "tags":         { "type": "text" },
          "points":       { "type": "integer" },
          "answer_count": { "type": "integer" },
          "views":        { "type": "integer" },
		  "create_at":    { "type": "date", "format": "epoch_millis" },
		  "update_at":    { "type": "date", "format": "epoch_millis" },
		  "delete_at":    { "type": "date", "format": "epoch_millis" },
        }
      }
    }
		}`
	s.rootStore.esBackend.CreateIndex(postMapping, search.INDEX_NAME_POSTS)
}
