package sqlstore

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/mlog"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/clear-ness/qa-discussion/utils"
	"github.com/go-gorp/gorp"
)

// TODO: このファイル全体でteam,group,member再考
type SqlPostStore struct {
	store.Store
	maxPostSizeOnce   sync.Once
	maxPostSizeCached int
}

func tagSliceColumns() []string {
	return []string{"Content", "TeamId", "Type", "PostCount", "CreateAt", "UpdateAt"}
}

func tagToSlice(tag *model.Tag) []interface{} {
	return []interface{}{
		tag.Content,
		tag.TeamId,
		tag.Type,
		tag.PostCount,
		tag.CreateAt,
		tag.UpdateAt,
	}
}

func NewSqlPostStore(sqlStore store.Store) store.PostStore {
	s := &SqlPostStore{
		Store:             sqlStore,
		maxPostSizeCached: model.POST_CONTENT_MAX_RUNES,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Post{}, "Posts").SetKeys(false, "Id")
	}

	return s
}

func (s *SqlPostStore) CreateSystemReview(post *model.Post, userId string, tagContents string, time int64) *model.AppError {
	review := &model.Vote{
		PostId:       post.Id,
		UserId:       userId,
		Type:         model.VOTE_TYPE_SYSTEM,
		Tags:         tagContents,
		TeamId:       post.TeamId,
		FirstPostRev: 1,
		CreateAt:     time,
	}

	if err := s.GetMaster().Insert(review); err != nil {
		return model.NewAppError("SqlPostStore.CreateSystemReview", "store.sql_post.create_system_review.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlPostStore) SaveQuestion(post *model.Post) (*model.Post, *model.AppError) {
	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.SaveQuestion", "store.sql_post.save_question.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()

	defer finalizeTransaction(transaction)
	if upsertErr := s.saveQuestion(transaction, post, curTime); upsertErr != nil {
		return nil, upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.SaveQuestion", "store.sql_post.save_question.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if count, err := s.GetPostCount("", post.UserId, post.TeamId, 0, 0); err == nil && count <= 1 {
		s.CreateSystemReview(post, "", model.SYSTEM_TAG_FIRST_POSTS, curTime)
	}

	return post, nil
}

func (s *SqlPostStore) saveQuestion(transaction *gorp.Transaction, post *model.Post, curTime int64) *model.AppError {
	if len(post.Id) > 0 {
		return model.NewAppError("SqlPostStore.saveQuestion", "store.sql_post.save_question.existing.app_error", nil, "id="+post.Id, http.StatusBadRequest)
	}

	maxPostSize := s.GetMaxPostSize()

	post.PreSave()
	if err := post.IsValid(maxPostSize); err != nil {
		return err
	}

	if err := transaction.Insert(post); err != nil {
		return model.NewAppError("SqlPostStore.saveQuestion", "store.sql_post.save_question.app_error", nil, "id="+post.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	addedTags := strings.Fields(post.Tags)

	if len(addedTags) > 0 {
		sql, args, err := s.buildInsertTagsQuery(addedTags, curTime, post.TeamId)
		if err != nil {
			return model.NewAppError("SqlPostStore.saveQuestion", "store.sql_post.save_question.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		if _, err := transaction.Exec(sql, args...); err != nil {
			return model.NewAppError("SqlPostStore.saveQuestion", "store.sql_post.save_question.insert_tags.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points + :PointForCreateQuestion, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForCreateQuestion": model.USER_POINT_FOR_CREATE_QUESTION, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.saveQuestion", "store.sql_post.save_question.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points + :PointForCreateQuestion WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForCreateQuestion": model.USER_POINT_FOR_CREATE_QUESTION, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.saveQuestion", "store.sql_post.save_question.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_CREATE_QUESTION,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     post.Tags,
		Points:   model.USER_POINT_FOR_CREATE_QUESTION,
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return nil
}

func (s *SqlPostStore) SaveUserPointHistory(history *model.UserPointHistory) (*model.UserPointHistory, *model.AppError) {
	if err := s.GetMaster().Insert(history); err != nil {
		return nil, model.NewAppError("SqlPostStore.SaveUserPointHistory", "store.sql_post.save_user_point_history.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return history, nil
}

func (s *SqlPostStore) SaveAnswer(post *model.Post) (*model.Post, *model.AppError) {
	var parent *model.Post
	if err := s.GetReplica().SelectOne(&parent, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": post.ParentId}); err != nil {
		return nil, model.NewAppError("SqlPostStore.SaveAnswer", "store.sql_post.save_answer.parent.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.SaveAnswer", "store.sql_post.save_answer.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()

	defer finalizeTransaction(transaction)
	if upsertErr := s.saveAnswer(transaction, post, parent, curTime); upsertErr != nil {
		return nil, upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.SaveAnswer", "store.sql_post.save_answer.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	tags := []string{}
	if count, err := s.GetPostCount("", post.UserId, post.TeamId, 0, 0); err == nil && count <= 1 {
		tags = append(tags, model.SYSTEM_TAG_FIRST_POSTS)
	}
	if (curTime - parent.CreateAt) > model.LATE_ANSWERS_MILLIS {
		tags = append(tags, model.SYSTEM_TAG_LATE_ANSWERS)
	}

	if len(tags) > 0 {
		tagContents := strings.Join(tags, " ")
		s.CreateSystemReview(post, "", tagContents, curTime)
	}

	return post, nil
}

func (s *SqlPostStore) saveAnswer(transaction *gorp.Transaction, post *model.Post, parent *model.Post, curTime int64) *model.AppError {
	if len(post.Id) > 0 {
		return model.NewAppError("SqlPostStore.saveAnswer", "store.sql_post.save_answer.existing.app_error", nil, "id="+post.Id, http.StatusBadRequest)
	}

	maxPostSize := s.GetMaxPostSize()

	post.PreSave()
	if err := post.IsValid(maxPostSize); err != nil {
		return err
	}

	if err := transaction.Insert(post); err != nil {
		return model.NewAppError("SqlPostStore.saveAnswer", "store.sql_post.save_answer.app_error", nil, "id="+post.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	if _, err := transaction.Exec(
		`
		UPDATE Posts
		SET
			AnswerCount = AnswerCount + 1,
			UpdateAt = :UpdateAt
		WHERE
			Id = :Id
		`,
		map[string]interface{}{
			"Id":       post.ParentId,
			"UpdateAt": curTime,
		},
	); err != nil {
		return model.NewAppError("SqlPostStore.saveAnswer", "store.sql_post.save_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	// prevent self point gain
	if post.UserId == parent.UserId {
		return nil
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points + :PointForCreateAnswer, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForCreateAnswer": model.USER_POINT_FOR_CREATE_ANSWER, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.saveAnswer", "store.sql_post.save_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points + :PointForCreateAnswer WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForCreateAnswer": model.USER_POINT_FOR_CREATE_ANSWER, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.saveAnswer", "store.sql_post.save_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_CREATE_ANSWER,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     parent.Tags,
		Points:   model.USER_POINT_FOR_CREATE_ANSWER,
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return nil
}

func (s *SqlPostStore) SaveComment(post *model.Post) (*model.Post, *model.AppError) {
	if len(post.Id) > 0 {
		return nil, model.NewAppError("SqlPostStore.SaveComment", "store.sql_post.save_comment.existing.app_error", nil, "id="+post.Id, http.StatusBadRequest)
	}

	maxPostSize := s.GetMaxPostSize()

	post.PreSave()
	if err := post.IsValid(maxPostSize); err != nil {
		return nil, err
	}

	comments, err := s.GetCommentsForPost(post.ParentId, model.POST_COMMENT_LIMIT)
	if err != nil {
		return nil, err
	}

	if len(comments) >= model.POST_COMMENT_LIMIT {
		return nil, model.NewAppError("SqlPostStore.saveComment", "store.sql_post.save_comment.max_limit.app_error", nil, "id="+post.Id, http.StatusBadRequest)
	}

	curTime := model.GetMillis()

	if err := s.GetMaster().Insert(post); err != nil {
		return nil, model.NewAppError("SqlPostStore.saveComment", "store.sql_post.save_comment.inserting.app_error", nil, "id="+post.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	if count, err := s.GetPostCount("", post.UserId, post.TeamId, 0, 0); err == nil && count <= 1 {
		s.CreateSystemReview(post, "", model.SYSTEM_TAG_FIRST_POSTS, curTime)
	}

	return post, nil
}

func (s *SqlPostStore) Update(newPost *model.Post, oldPost *model.Post) (*model.Post, *model.AppError) {
	removedTags := []string{}
	addedTags := []string{}

	if model.POST_TYPE_QUESTION == newPost.Type {
		removedTags = utils.StringSliceDiff(strings.Fields(oldPost.Tags), strings.Fields(newPost.Tags))
		addedTags = utils.StringSliceDiff(strings.Fields(newPost.Tags), strings.Fields(oldPost.Tags))
	}

	newPost.UpdateAt = model.GetMillis()

	oldPost.DeleteAt = newPost.UpdateAt
	oldPost.UpdateAt = newPost.UpdateAt
	oldPost.OriginalId = oldPost.Id
	oldPost.Id = model.NewId()

	maxPostSize := s.GetMaxPostSize()

	if err := newPost.IsValid(maxPostSize); err != nil {
		return nil, err
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.Update", "store.sql_post.update.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	defer finalizeTransaction(transaction)

	if upsertErr := s.update(transaction, newPost, removedTags, addedTags); upsertErr != nil {
		return nil, upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.Update", "store.sql_post.update.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	s.GetMaster().Insert(oldPost)

	return newPost, nil
}

func (s *SqlPostStore) update(transaction *gorp.Transaction, post *model.Post, removedTags []string, addedTags []string) *model.AppError {
	curTime := model.GetMillis()

	if _, err := transaction.Update(post); err != nil {
		return model.NewAppError("SqlPostStore.update", "store.sql_post.update.app_error", nil, "id="+post.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	for _, tagContent := range removedTags {
		if _, err := transaction.Exec("UPDATE Tags SET PostCount = PostCount - 1, UpdateAt = :UpdateAt WHERE Content = :Content AND TeamId = :TeamId", map[string]interface{}{"UpdateAt": curTime, "Content": tagContent, "TeamId": post.TeamId}); err != nil {
			return model.NewAppError("SqlPostStore.update", "store.sql_post.update.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	if len(addedTags) > 0 {
		sql, args, err := s.buildInsertTagsQuery(addedTags, curTime, post.TeamId)
		if err != nil {
			return model.NewAppError("SqlPostStore.update", "store.sql_post.update.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		if _, err := transaction.Exec(sql, args...); err != nil {
			return model.NewAppError("SqlPostStore.update", "store.sql_post.update.insert_tags.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	return nil
}

func (s *SqlPostStore) buildInsertTagsQuery(addedTags []string, time int64, teamId string) (string, []interface{}, error) {
	query := s.GetQueryBuilder().Insert("Tags").Columns(tagSliceColumns()...)

	for _, tagContent := range addedTags {
		tag := &model.Tag{
			Content:   tagContent,
			TeamId:    teamId,
			Type:      "",
			PostCount: 1,
			CreateAt:  time,
			UpdateAt:  time,
		}

		tag.PreSave()
		if err := tag.IsValid(); err != nil {
			return "", nil, err
		}

		query = query.Values(tagToSlice(tag)...)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return "", args, model.NewAppError("SqlPostStore.buildInsertTagsQuery", "store.sql_post.inset_tags.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return sql + " ON DUPLICATE KEY UPDATE PostCount = VALUES(PostCount) + 1, UpdateAt = VALUES(UpdateAt)", args, nil
}

func (s *SqlPostStore) GetSingle(id string, includeDeleted bool) (*model.Post, *model.AppError) {
	var deletedClause string
	if !includeDeleted {
		deletedClause = "DeleteAt = 0 AND"
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, `
	SELECT *
	FROM
		Posts
	WHERE
		`+deletedClause+`
		Id = :Id
	`, map[string]interface{}{"Id": id})

	if err != nil {
		return nil, model.NewAppError("SqlPostStore.GetSingle", "store.sql_post.get.app_error", nil, "id="+id+err.Error(), http.StatusNotFound)
	}
	return post, nil
}

func (s *SqlPostStore) GetSingleByType(id string, postType string) (*model.Post, *model.AppError) {
	if postType != model.POST_TYPE_QUESTION && postType != model.POST_TYPE_ANSWER && postType != model.POST_TYPE_COMMENT {
		return nil, model.NewAppError("SqlPostStore.GetSingleByType", "store.sql_post.get_by_type.app_error", nil, "id="+id, http.StatusNotFound)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND Type = :Type AND DeleteAt = 0", map[string]interface{}{"Id": id, "Type": postType})

	if err != nil {
		return nil, model.NewAppError("SqlPostStore.GetSingleByType", "store.sql_post.get_by_type.app_error", nil, "id="+id+err.Error(), http.StatusNotFound)
	}
	return post, nil
}

func (s *SqlPostStore) GetPostCount(postType string, userId string, teamId string, fromDate int64, toDate int64) (int64, *model.AppError) {
	args := map[string]interface{}{}

	teamFilter := ""
	if teamId == "" {
		teamFilter = "AND TeamId IS NULL"
	} else {
		teamFilter = "AND TeamId = :TeamId"
		args["TeamId"] = teamId
	}

	typeFilter := ""
	if postType != "" {
		typeFilter = "AND Type = :Type"
		args["Type"] = postType
	}

	userFilter := ""
	if userId != "" {
		userFilter = "AND UserId = :UserId"
		args["UserId"] = userId
	}

	fromFilter := ""
	if fromDate != int64(0) {
		fromFilter = "AND CreateAt >= :FromDate"
		args["FromDate"] = fromDate
	}

	toFilter := ""
	if toDate != int64(0) {
		toFilter = "AND CreateAt <= :ToDate"
		args["ToDate"] = toDate
	}

	count, err := s.GetReplica().SelectInt(`
		SELECT
			count(*)
		FROM
			Posts
		WHERE
			DeleteAt = 0
			`+typeFilter+`
			`+teamFilter+`
			`+userFilter+`
			`+fromFilter+`
			`+toFilter+`
			`, args)

	if err != nil {
		return 0, model.NewAppError("SqlPostStore.GetPostCount", "store.sql_post.get_post_count_by_user_id.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return count, nil
}

func (s *SqlPostStore) GetPostsByIds(postIds []string) (model.Posts, *model.AppError) {
	keys, params := MapStringsToQueryParams(postIds, "Post")
	query := `SELECT p.* FROM Posts p WHERE p.Id IN ` + keys + ` AND DeleteAt = 0 ORDER BY CreateAt DESC`

	var posts model.Posts
	_, err := s.GetReplica().Select(&posts, query, params)
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.GetPostsByIds", "store.sql_post.get_posts_by_ids.app_error", nil, "", http.StatusInternalServerError)
	}
	return posts, nil
}

// https://stackoverflow.com/questions/25088183/mysql-fulltext-search-with-symbol-produces-error-syntax-error-unexpected
var specialSearchChar = []string{
	"<",
	">",
	"+",
	"-",
	"(",
	")",
	"~",
	"@",
	":",
	".",
}

func (s *SqlPostStore) GetPosts(options *model.GetPostsOptions, getCount bool) (model.Posts, int64, *model.AppError) {
	searchOptions := &model.SearchPostsOptions{
		UserId:         options.UserId,
		SortType:       options.SortType,
		PostType:       options.PostType,
		Ids:            []string{},
		ParentId:       options.ParentId,
		FromDate:       options.FromDate,
		ToDate:         options.ToDate,
		Page:           options.Page,
		PerPage:        options.PerPage,
		TeamId:         options.TeamId,
		IncludeDeleted: options.IncludeDeleted,
		OriginalId:     options.OriginalId,
	}

	if options.Title != "" {
		searchOptions.Terms = options.Title
		if options.Tagged != "" {
			searchOptions.Terms += " " + options.Tagged
		}
		searchOptions.TermsType = model.TERMS_TYPE_SIMILAR
	} else if options.Link != "" {
		searchOptions.Terms = options.Link
		searchOptions.TermsType = model.TERMS_TYPE_LINK
	} else if options.Tagged != "" {
		searchOptions.Terms = options.Tagged
		searchOptions.TermsType = model.TERMS_TYPE_TAG
	}

	if options.SortType == model.POST_SORT_TYPE_VOTES {
		if options.Min != nil {
			searchOptions.MinVotes = options.Min
		}
		if options.Max != nil {
			searchOptions.MaxVotes = options.Max
		}
	} else if options.SortType == model.POST_SORT_TYPE_ANSWERS {
		if options.Min != nil {
			searchOptions.MinAnswers = options.Min
		}
		if options.Max != nil {
			searchOptions.MaxAnswers = options.Max
		}
	}

	if options.NoAnswers {
		zero := 0
		searchOptions.MinAnswers = &zero
		searchOptions.MaxAnswers = &zero
	}

	queryString, args, err := s.searchPosts(searchOptions, false).ToSql()
	if err != nil {
		return nil, int64(0), model.NewAppError("SqlPostStore.GetPostContext", "store.sql_post.get_posts.get.app_error", nil, "", http.StatusInternalServerError)
	}

	var posts []*model.Post
	_, err = s.GetMaster().Select(&posts, queryString, args...)
	if err != nil {
		return nil, int64(0), model.NewAppError("SqlPostStore.GetPostContext", "store.sql_post.get_posts.select.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	totalCount := int64(0)
	if getCount {
		queryString, args, err = s.searchPosts(searchOptions, true).ToSql()
		if err != nil {
			return nil, int64(0), model.NewAppError("SqlPostStore.GetPostContext", "store.sql_post.get_posts.get.app_error", nil, "", http.StatusInternalServerError)
		}
		if totalCount, err = s.GetMaster().SelectInt(queryString, args...); err != nil {
			return nil, int64(0), model.NewAppError("SqlPostStore.GetPostContext", "store.sql_post.get_posts.get.app_error", nil, "", http.StatusInternalServerError)
		}
	}

	return posts, totalCount, nil
}

// advanced search
func (s *SqlPostStore) SearchPosts(paramsList []*model.SearchParams, sortType string, page, perPage int, teamId string) (model.Posts, int64, *model.AppError) {
	options := []*model.SearchPostsOptions{}
	for _, params := range paramsList {
		fromDate := int64(0)
		if params.FromDate != "" {
			fromDate = params.GetFromDateMillis()
		}
		toDate := int64(0)
		if params.ToDate != "" {
			toDate = params.GetToDateMillis()
		}

		option := &model.SearchPostsOptions{
			Terms:         params.Terms,
			ExcludedTerms: params.ExcludedTerms,
			TermsType:     params.TermsType,
			UserId:        params.User,
			SortType:      sortType,
			MinVotes:      params.MinVotes,
			MaxVotes:      params.MaxVotes,
			MinAnswers:    params.MinAnswers,
			MaxAnswers:    params.MaxAnswers,
			PostType:      params.PostType,
			Ids:           params.Ids,
			ParentId:      params.Parent,
			FromDate:      fromDate,
			ToDate:        toDate,
			Page:          0,
			PerPage:       model.POST_SEARCH_MAX_COUNT * 5,
			TeamId:        teamId,
		}

		options = append(options, option)
	}

	var wg sync.WaitGroup

	pchan := make(chan store.StoreResult, len(paramsList))

	for _, option := range options {
		if option.Terms == "*" {
			continue
		}

		wg.Add(1)

		go func(option *model.SearchPostsOptions) {
			defer wg.Done()

			var posts model.Posts
			queryString, args, err := s.searchPosts(option, false).ToSql()
			if err != nil {
				appErr := model.NewAppError("SqlPostStore.SearchPosts", "store.sql_post.search_posts.app_error", nil, err.Error(), http.StatusInternalServerError)
				pchan <- store.StoreResult{Data: posts, Err: appErr}
				return
			}

			_, err = s.GetReplica().Select(&posts, queryString, args...)
			if err != nil {
				appErr := model.NewAppError("SqlPostStore.SearchPosts", "store.sql_post.search_posts.app_error", nil, err.Error(), http.StatusInternalServerError)
				pchan <- store.StoreResult{Data: posts, Err: appErr}
				return
			}

			pchan <- store.StoreResult{Data: posts, Err: nil}

		}(option)
	}

	wg.Wait()
	close(pchan)

	// get conjunction of results
	finalPostMap := map[string]*model.Post{}
	firstChan := true
	for result := range pchan {
		if result.Err != nil {
			return nil, int64(0), result.Err
		}

		dupPostMap := map[string]*model.Post{}
		data := result.Data.(model.Posts)
		for _, p := range data {
			if firstChan || finalPostMap[p.Id] != nil {
				dupPostMap[p.Id] = p
			}
		}

		finalPostMap = dupPostMap
		firstChan = false
	}

	var posts model.Posts
	for _, p := range finalPostMap {
		posts = append(posts, p)
	}

	sort.Slice(posts, func(i, j int) bool {
		switch sortType {
		case model.POST_SORT_TYPE_ACTIVE:
			return posts[i].UpdateAt > posts[j].UpdateAt
		case model.POST_SORT_TYPE_VOTES:
			return posts[i].Points > posts[j].Points
		default:
			return posts[i].CreateAt > posts[j].CreateAt
		}
	})

	if len(posts) > model.POST_SEARCH_MAX_COUNT {
		posts = posts[:model.POST_SEARCH_MAX_COUNT]
	}

	totalCount := int64(len(posts))

	if len(posts) > page*perPage {
		start := page * perPage
		end := len(posts)

		if len(posts) > (page+1)*perPage {
			end = (page + 1) * perPage
		}

		posts = posts[start:end]

		return posts, totalCount, nil
	}

	return nil, totalCount, nil
}

func (s *SqlPostStore) searchPosts(options *model.SearchPostsOptions, countQuery bool) sq.SelectBuilder {
	offset := options.Page * options.PerPage

	var selectStr string
	if countQuery {
		selectStr = "count(*)"
	} else {
		selectStr = "p.*"
	}

	query := s.GetQueryBuilder().Select(selectStr)
	query = query.From("Posts p")

	if !options.IncludeDeleted {
		query = query.Where(sq.And{
			sq.Eq{"DeleteAt": int(0)},
		})
	}

	if options.PostType != "" {
		query = query.Where(sq.And{
			sq.Expr(`Type = ?`, options.PostType),
		})
	} else {
		// search questions and answers when no PostType
		query = query.Where(sq.And{
			sq.Expr(`Type IN (?, ?)`, model.POST_TYPE_QUESTION, model.POST_TYPE_ANSWER),
		})
	}

	if options.TeamId != "" {
		query = query.Where(sq.And{
			sq.Expr(`TeamId = ?`, options.TeamId),
		})
	} else {
		query = query.Where("TeamId = ''")
	}

	if len(options.Ids) > 0 {
		query = query.Where(sq.And{
			sq.Expr(`Id IN (?)`, options.Ids[0]),
		})
	}

	if options.FromDate != 0 {
		query = query.Where(sq.And{
			sq.Expr(`CreateAt >= ?`, options.FromDate),
		})
	}

	if options.ToDate != 0 {
		query = query.Where(sq.And{
			sq.Expr(`CreateAt <= ?`, options.ToDate),
		})
	}

	if (options.PostType == model.POST_TYPE_ANSWER || options.PostType == model.POST_TYPE_COMMENT || options.PostType == "") && options.ParentId != "" {
		query = query.Where(sq.And{
			sq.Expr(`ParentId = ?`, options.ParentId),
		})
	}

	if options.OriginalId != "" {
		query = query.Where(sq.And{
			sq.Expr(`OriginalId = ?`, options.OriginalId),
		})
	}

	if options.UserId != "" {
		query = query.Where(sq.And{
			sq.Expr(`UserId = ?`, options.UserId),
		})
	}

	if options.MinVotes != nil {
		query = query.Where(sq.And{
			sq.Expr(`Points >= ?`, *options.MinVotes),
		})
	}
	if options.MaxVotes != nil {
		query = query.Where(sq.And{
			sq.Expr(`Points <= ?`, *options.MaxVotes),
		})
	}

	if options.PostType == model.POST_TYPE_QUESTION && options.MinAnswers != nil {
		query = query.Where(sq.And{
			sq.Expr(`AnswerCount >= ?`, *options.MinAnswers),
		})
	}
	if options.PostType == model.POST_TYPE_QUESTION && options.MaxAnswers != nil {
		query = query.Where(sq.And{
			sq.Expr(`AnswerCount <= ?`, *options.MaxAnswers),
		})
	}

	var orderBy = "CreateAt DESC"
	if options.SortType == model.POST_SORT_TYPE_ACTIVE {
		orderBy = "UpdateAt DESC"
	} else if options.SortType == model.POST_SORT_TYPE_VOTES {
		orderBy = "Points DESC"
	} else if options.SortType == model.POST_SORT_TYPE_ANSWERS {
		orderBy = "AnswerCount DESC"
	} else if options.TermsType == model.TERMS_TYPE_SIMILAR && options.SortType == model.POST_SORT_TYPE_RELEVANCE {
		// mysqlではデフォルトで似ている順になるため。
		orderBy = ""
	}

	terms := options.Terms
	excludedTerms := options.ExcludedTerms

	for _, c := range specialSearchChar {
		if options.TermsType != model.TERMS_TYPE_LINK {
			terms = strings.Replace(terms, c, " ", -1)
		}
		excludedTerms = strings.Replace(excludedTerms, c, " ", -1)
	}

	if options.TermsType != "" {
		searchColumns := ""
		if options.TermsType == model.TERMS_TYPE_TAG {
			searchColumns = "Tags"
		} else if options.TermsType == model.TERMS_TYPE_SIMILAR {
			searchColumns = "Title, Tags"
		} else if options.TermsType == model.TERMS_TYPE_PLAIN {
			searchColumns = "Title, Tags, Content"
		} else if options.TermsType == model.TERMS_TYPE_TITLE {
			searchColumns = "Title"
		} else if options.TermsType == model.TERMS_TYPE_BODY || options.TermsType == model.TERMS_TYPE_LINK {
			searchColumns = "Content"
		} else {
			searchColumns = "Title, Tags, Content"
		}
		fulltextClause := fmt.Sprintf("MATCH(%s) AGAINST (? IN BOOLEAN MODE)", searchColumns)

		excludeClause := ""
		if excludedTerms != "" {
			excludeClause = " -(" + excludedTerms + ")"
		}

		splitTerms := []string{}
		for _, t := range strings.Fields(terms) {
			if len(t) >= model.TAG_MIN_RUNES {
				if options.TermsType == model.TERMS_TYPE_SIMILAR {
					splitTerms = append(splitTerms, t)
				} else if options.TermsType == model.TERMS_TYPE_LINK {
					splitTerms = append(splitTerms, "\""+t+"\"")
				} else {
					splitTerms = append(splitTerms, "+"+t)
				}
			}
		}
		terms = strings.Join(splitTerms, " ") + excludeClause

		query = query.Where(sq.And{
			sq.Expr(fulltextClause, terms),
		})
	}

	if !countQuery {
		if orderBy != "" {
			query = query.OrderBy(orderBy)
		}
		query = query.Limit(uint64(options.PerPage)).Offset(uint64(offset))
	}

	return query
}

func (s *SqlPostStore) DeleteQuestion(postId string, time int64, deleteById string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.DeleteQuestion", "store.sql_post.delete_question.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return model.NewAppError("SqlPostStore.DeleteQuestion", "store.sql_post.delete_question.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)
	if upsertErr := s.deleteQuestion(transaction, post, time, deleteById); upsertErr != nil {
		return upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return model.NewAppError("SqlPostStore.DeleteQuestion", "store.sql_post.delete_question.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlPostStore) deleteQuestion(transaction *gorp.Transaction, post *model.Post, time int64, deleteById string) *model.AppError {
	post.AddProp(model.POST_PROPS_DELETE_BY, deleteById)

	if _, err := transaction.Exec("UPDATE Posts SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"DeleteAt": time, "UpdateAt": time, "Id": post.Id, "Props": model.StringInterfaceToJson(post.Props)}); err != nil {
		return model.NewAppError("SqlPostStore.deleteQuestion", "store.sql_post.delete_question.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	tagContents := strings.Fields(post.Tags)
	for _, tagContent := range tagContents {
		if _, err := transaction.Exec("UPDATE Tags SET PostCount = PostCount - 1 WHERE Content = :Content AND TeamId = :TeamId",
			map[string]interface{}{"Content": tagContent, "TeamId": post.TeamId}); err != nil {
			return model.NewAppError("SqlPostStore.deleteQuestion", "store.sql_post.delete_question.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points - :PointForCreateQuestion, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForCreateQuestion": model.USER_POINT_FOR_CREATE_QUESTION, "UpdateAt": time, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.deleteQuestion", "store.sql_post.delete_question.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points - :PointForCreateQuestion WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForCreateQuestion": model.USER_POINT_FOR_CREATE_QUESTION, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.deleteQuestion", "store.sql_post.delete_question.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	if err := s.invalidateReviewsForPost(transaction, post.Id, time, post.TeamId); err != nil {
		return err
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_DELETE_QUESTION,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     post.Tags,
		Points:   -(model.USER_POINT_FOR_CREATE_QUESTION),
		CreateAt: time,
	}
	s.SaveUserPointHistory(user_point_history)

	return nil
}

func (s *SqlPostStore) invalidateReviewsForPost(transaction *gorp.Transaction, postId string, time int64, teamId string) *model.AppError {
	var rev int64
	var err *model.AppError
	if rev, err = s.GetCurrentRevisionForPost(postId, teamId); err != nil {
		return model.NewAppError("SqlPostStore.invalidateReviewsForPost", "store.sql_post.invalidate_reviews_for_post.get_revision.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := s.GetMaster().Exec("UPDATE Votes SET InvalidateAt = :InvalidateAt, LastPostRev = :LastPostRev WHERE PostId = :PostId AND Type IN (:Type1, :Type2, :Type3) AND InvalidateAt = 0  AND CompletedAt = 0 AND RejectedAt = 0", map[string]interface{}{"InvalidateAt": time, "LastPostRev": rev, "PostId": postId, "Type1": model.VOTE_TYPE_REVIEW, "Type2": model.VOTE_TYPE_FLAG, "Type3": model.VOTE_TYPE_SYSTEM}); err != nil {
		return model.NewAppError("SqlPostStore.invalidateReviewsForPost", "store.sql_post.invalidate_reviews_for_post.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlPostStore) DeleteAnswer(postId string, time int64, deleteById string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.DeleteAnswer", "store.sql_post.delete_answer.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	var parent *model.Post
	if err := s.GetReplica().SelectOne(&parent, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": post.ParentId}); err != nil {
		return appErr(err.Error())
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return model.NewAppError("SqlPostStore.DeleteAnswer", "store.sql_post.delete_answer.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)
	if upsertErr := s.deleteAnswer(transaction, post, parent, time, deleteById); upsertErr != nil {
		return upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return model.NewAppError("SqlPostStore.DeleteAnswer", "store.sql_post.delete_answer.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlPostStore) deleteAnswer(transaction *gorp.Transaction, post *model.Post, parent *model.Post, time int64, deleteById string) *model.AppError {
	post.AddProp(model.POST_PROPS_DELETE_BY, deleteById)

	if _, err := transaction.Exec("UPDATE Posts SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"DeleteAt": time, "UpdateAt": time, "Id": post.Id, "Props": model.StringInterfaceToJson(post.Props)}); err != nil {
		return model.NewAppError("SqlPostStore.deleteAnswer", "store.sql_post.delete_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := transaction.Exec("UPDATE Posts SET AnswerCount = AnswerCount - 1 WHERE Id = :Id AND Type = :Type",
		map[string]interface{}{"Id": post.ParentId, "Type": model.POST_TYPE_QUESTION}); err != nil {
		return model.NewAppError("SqlPostStore.deleteAnswer", "store.sql_post.delete_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points - :PointForCreateAnswer, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForCreateAnswer": model.USER_POINT_FOR_CREATE_ANSWER, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.deleteAnswer", "store.sql_post.delete_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points - :PointForCreateAnswer WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForCreateAnswer": model.USER_POINT_FOR_CREATE_ANSWER, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.deleteAnswer", "store.sql_post.delete_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	if err := s.invalidateReviewsForPost(transaction, post.Id, time, post.TeamId); err != nil {
		return err
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_DELETE_ANSWER,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     parent.Tags,
		Points:   -(model.USER_POINT_FOR_CREATE_ANSWER),
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return nil
}

func (s *SqlPostStore) DeleteComment(postId string, time int64, deleteById string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.DeleteComment", "store.sql_post.delete_comment.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	post.AddProp(model.POST_PROPS_DELETE_BY, deleteById)

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return model.NewAppError("SqlPostStore.DeleteComment", "store.sql_post.delete_comment.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)

	if _, err := transaction.Exec("UPDATE Posts SET DeleteAt = :DeleteAt, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"DeleteAt": time, "UpdateAt": time, "Id": postId, "Props": model.StringInterfaceToJson(post.Props)}); err != nil {
		return model.NewAppError("SqlPostStore.DeleteComment", "store.sql_post.delete_comment.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if err := s.invalidateReviewsForPost(transaction, postId, time, post.TeamId); err != nil {
		return err
	}

	if err := transaction.Commit(); err != nil {
		return model.NewAppError("SqlPostStore.DeleteComment", "store.sql_post.delete_comment.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlPostStore) SelectBestAnswer(postId, bestId string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.SelectBestAnswer", "store.sql_post.select_best_answer.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	if post.BestId != "" {
		return model.NewAppError("SqlPostStore.SelectBestAnswer", "store.sql_post.select_best_answer.best_exists.app_error", nil, "", http.StatusInternalServerError)
	}

	var ans *model.Post
	err = s.GetReplica().SelectOne(&ans, "SELECT * FROM Posts WHERE Id = :Id AND Type = :Type AND DeleteAt = 0", map[string]interface{}{"Id": bestId, "Type": model.POST_TYPE_ANSWER})
	if err != nil {
		return appErr(err.Error())
	}

	if ans.ParentId != post.Id {
		return model.NewAppError("SqlPostStore.SelectBestAnswer", "store.sql_post.select_best_answer.invalid_answer.app_error", nil, "", http.StatusInternalServerError)
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return model.NewAppError("SqlPostStore.SelectBestAnswer", "store.sql_post.select_best_answer.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)
	if upsertErr := s.selectBestAnswer(transaction, post, ans); upsertErr != nil {
		return upsertErr
	}

	if err = transaction.Commit(); err != nil {
		return model.NewAppError("SqlPostStore.SelectBestAnswer", "store.sql_post.select_best_answer.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *SqlPostStore) selectBestAnswer(transaction *gorp.Transaction, post *model.Post, ans *model.Post) *model.AppError {
	curTime := model.GetMillis()

	if _, err := transaction.Exec("UPDATE Posts SET BestId = :BestId, UpdateAt = :UpdateAt WHERE Id = :Id AND Type = :Type",
		map[string]interface{}{"BestId": ans.Id, "UpdateAt": curTime, "Id": post.Id, "Type": model.POST_TYPE_QUESTION}); err != nil {
		return model.NewAppError("SqlPostStore.selectBestAnswer", "store.sql_post.select_best_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	// prevent self point gain
	if post.UserId == ans.UserId {
		return nil
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points + :PointForSelectAnswer, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForSelectAnswer": model.USER_POINT_FOR_SELECT_ANSWER, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.selectBestAnswer", "store.sql_post.select_best_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		if _, err := transaction.Exec("UPDATE Users SET Points = Points + :PointForSelectedAnswer, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForSelectedAnswer": model.USER_POINT_FOR_SELECTED_ANSWER, "UpdateAt": curTime, "Id": ans.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.selectBestAnswer", "store.sql_post.select_best_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points + :PointForSelectAnswer WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForSelectAnswer": model.USER_POINT_FOR_SELECT_ANSWER, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.selectBestAnswer", "store.sql_post.select_best_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}

		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points + :PointForSelectedAnswer WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForSelectedAnswer": model.USER_POINT_FOR_SELECTED_ANSWER, "TeamId": ans.TeamId, "UserId": ans.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.selectBestAnswer", "store.sql_post.select_best_answer.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_SELECT_ANSWER,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     post.Tags,
		Points:   model.USER_POINT_FOR_SELECT_ANSWER,
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	user_point_history2 := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   ans.TeamId,
		UserId:   ans.UserId,
		Type:     model.USER_POINT_TYPE_SELECTED_ANSWER,
		PostId:   ans.Id,
		PostType: ans.Type,
		Tags:     post.Tags,
		Points:   model.USER_POINT_FOR_SELECTED_ANSWER,
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history2)

	return nil
}

func (s *SqlPostStore) GetMaxPostSize() int {
	s.maxPostSizeOnce.Do(func() {
		s.maxPostSizeCached = s.determineMaxPostSize()
	})

	return s.maxPostSizeCached
}

func (s *SqlPostStore) determineMaxPostSize() int {
	var maxPostSizeBytes int32

	if err := s.GetReplica().SelectOne(&maxPostSizeBytes, `
		SELECT
			COALESCE(CHARACTER_MAXIMUM_LENGTH, 0)
		FROM
			INFORMATION_SCHEMA.COLUMNS
		WHERE
			table_schema = DATABASE()
		AND table_name = 'Posts'
		AND column_name = 'Content'
		LIMIT 1
	`); err != nil {
		mlog.Error("Unable to determine the maximum supported post size", mlog.Err(err))
	}

	maxPostSize := int(maxPostSizeBytes) / 4

	if maxPostSize < model.POST_CONTENT_MAX_RUNES {
		maxPostSize = model.POST_CONTENT_MAX_RUNES
	}

	mlog.Info("Post.Content has size restrictions", mlog.Int("max_characters", maxPostSize), mlog.Int32("max_bytes", maxPostSizeBytes))

	return maxPostSize
}

func (s *SqlPostStore) GetCommentsForPost(postId string, limit int) ([]*model.Post, *model.AppError) {
	var comments []*model.Post
	if _, err := s.GetReplica().Select(&comments, `
		SELECT
			*
		FROM
			Posts
		WHERE
			ParentId = :ParentId AND Type = :Type AND DeleteAt = 0
		ORDER BY
			CreateAt DESC
		LIMIT
			:Limit
		`,
		map[string]interface{}{"ParentId": postId, "Type": model.POST_TYPE_COMMENT, "Limit": limit}); err != nil {
		if err != sql.ErrNoRows {
			return nil, model.NewAppError("SqlPostStore.GetCommentsForPost", "store.sql_post.get_comments_for_post.app_error", nil, "", http.StatusInternalServerError)
		}
	}

	return comments, nil
}

func (s *SqlPostStore) GetChildPostsCount(id string) (int64, *model.AppError) {
	count, err := s.GetReplica().SelectInt(`
		SELECT
			count(*)
		FROM
			Posts
		WHERE
			ParentId = :ParentId	
		AND (Type = :Type1 OR Type = :Type2)
		AND DeleteAt = 0`,
		map[string]interface{}{"ParentId": id, "Type1": model.POST_TYPE_ANSWER, "Type2": model.POST_TYPE_COMMENT})

	if err != nil {
		return 0, model.NewAppError("SqlPostStore.GetChildPostsCount", "store.sql_post.get_child_posts_count.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return count, nil
}

func (s *SqlPostStore) UpVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.UpVotePost", "store.sql_post.upvote_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return nil, appErr(err.Error())
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.UpVotePost", "store.sql_post.upvote_post.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)

	var vote *model.Vote
	var upsertErr *model.AppError
	if vote, upsertErr = s.upvotePost(transaction, post, userId); upsertErr != nil {
		return nil, upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.UpVotePost", "store.sql_post.upvote_post.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return vote, nil
}

func (s *SqlPostStore) upvotePost(transaction *gorp.Transaction, post *model.Post, userId string) (*model.Vote, *model.AppError) {
	curTime := model.GetMillis()

	vote := &model.Vote{
		PostId:   post.Id,
		UserId:   userId,
		Type:     model.VOTE_TYPE_UP_VOTE,
		TeamId:   post.TeamId,
		CreateAt: curTime,
	}

	if err := transaction.Insert(vote); err != nil {
		return nil, model.NewAppError("SqlPostStore.upvotePost", "store.sql_post.upvotePost.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := transaction.Exec("UPDATE Posts SET UpVotes = UpVotes + 1, Points = Points + 1, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"UpdateAt": curTime, "Id": post.Id}); err != nil {
		return nil, model.NewAppError("SqlPostStore.upvotePost", "store.sql_post.upvotePost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	// prevent self point gain
	if userId == post.UserId {
		return vote, nil
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points + :PointForVoted, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForVoted": model.USER_POINT_FOR_VOTED, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return nil, model.NewAppError("SqlPostStore.upvotePost", "store.sql_post.upvotePost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points + :PointForVoted WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForVoted": model.USER_POINT_FOR_VOTED, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return nil, model.NewAppError("SqlPostStore.upvotePost", "store.sql_post.upvotePost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	var tags string
	var appErr *model.AppError
	if tags, appErr = s.getTagsForPost(post); appErr != nil {
		return nil, appErr
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_VOTED,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     tags,
		Points:   model.USER_POINT_FOR_VOTED,
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return vote, nil
}

func (s *SqlPostStore) getTagsForPost(post *model.Post) (string, *model.AppError) {
	tags := post.Tags

	if post.Type == model.POST_TYPE_ANSWER {
		var parent *model.Post
		if err := s.GetReplica().SelectOne(&parent, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": post.ParentId}); err != nil {
			return "", model.NewAppError("SqlPostStore.getTagsForPost", "store.sql_post.get_tags_for_post.parent.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
		tags = parent.Tags
	} else if post.Type == model.POST_TYPE_COMMENT {
		var root *model.Post
		if err := s.GetReplica().SelectOne(&root, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": post.RootId}); err != nil {
			return "", model.NewAppError("SqlPostStore.getTagsForPost", "store.sql_post.get_tags_for_post.root.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
		tags = root.Tags
	}

	return tags, nil
}

func (s *SqlPostStore) CancelUpVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.CancelUpVotePost", "store.sql_post.cancel_upvote_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var vote *model.Vote
	err := s.GetReplica().SelectOne(&vote, "SELECT * FROM Votes WHERE PostId = :PostId AND Type = :Type AND UserId = :UserId", map[string]interface{}{"PostId": postId, "Type": model.VOTE_TYPE_UP_VOTE, "UserId": userId})
	if err != nil {
		return nil, appErr(err.Error())
	}
	if vote == nil {
		return nil, model.NewAppError("SqlPostStore.CancelUpvotePost", "store.sql_post.cancel_upvote_post.select.app_error", nil, "", http.StatusInternalServerError)
	}

	var post *model.Post
	err = s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return nil, appErr(err.Error())
	}
	if post == nil {
		return nil, model.NewAppError("SqlPostStore.CancelUpvotePost", "store.sql_post.cancel_upvote_post.select.app_error", nil, "", http.StatusInternalServerError)
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.CancelUpVotePost", "store.sql_post.cancel_upvote_post.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)

	var upsertErr *model.AppError
	if upsertErr = s.cancelUpvotePost(transaction, vote, post, userId); upsertErr != nil {
		return nil, upsertErr
	}

	if err = transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.CancelUpVotePost", "store.sql_post.cancel_upvote_post.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return vote, nil
}

func (s *SqlPostStore) cancelUpvotePost(transaction *gorp.Transaction, vote *model.Vote, post *model.Post, userId string) *model.AppError {
	if _, err := transaction.Delete(vote); err != nil {
		return model.NewAppError("SqlPostStore.CancelUpVotePost", "store.sql_post.cancel_upvote_post.deleting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()

	post.UpVotes = post.UpVotes - 1
	if post.UpVotes < 0 {
		post.UpVotes = 0
	}
	if _, err := transaction.Exec("UPDATE Posts SET UpVotes = :UpVotes, Points = Points - 1, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"UpVotes": post.UpVotes, "UpdateAt": curTime, "Id": post.Id}); err != nil {
		return model.NewAppError("SqlPostStore.CancelUpVotePost", "store.sql_post.cancel_upvote_post.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	// prevent self point gain
	if userId == post.UserId {
		return nil
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points - :PointForVoted, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForVoted": model.USER_POINT_FOR_VOTED, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.CancelUpvotePost", "store.sql_post.cancel_upvote_post.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points - :PointForVoted WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForVoted": model.USER_POINT_FOR_VOTED, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.CancelUpvotePost", "store.sql_post.cancel_upvote_post.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	var tags string
	var appErr *model.AppError
	if tags, appErr = s.getTagsForPost(post); appErr != nil {
		return appErr
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_VOTED_CANCELED,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     tags,
		Points:   -(model.USER_POINT_FOR_VOTED),
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return nil
}

func (s *SqlPostStore) DownVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.DownVotePost", "store.sql_post.downvote_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return nil, appErr(err.Error())
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.DownVotePost", "store.sql_post.downvote_post.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)

	var vote *model.Vote
	var upsertErr *model.AppError
	if vote, upsertErr = s.downvotePost(transaction, post, userId); upsertErr != nil {
		return nil, upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.DownVotePost", "store.sql_post.downvote_post.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return vote, nil
}

func (s *SqlPostStore) downvotePost(transaction *gorp.Transaction, post *model.Post, userId string) (*model.Vote, *model.AppError) {
	curTime := model.GetMillis()

	vote := &model.Vote{
		PostId:   post.Id,
		UserId:   userId,
		Type:     model.VOTE_TYPE_DOWN_VOTE,
		TeamId:   post.TeamId,
		CreateAt: curTime,
	}

	if err := transaction.Insert(vote); err != nil {
		return nil, model.NewAppError("SqlPostStore.downvotePost", "store.sql_post.downvotePost.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := transaction.Exec("UPDATE Posts SET DownVotes = DownVotes + 1, Points = Points - 1, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"UpdateAt": curTime, "Id": post.Id}); err != nil {
		return nil, model.NewAppError("SqlPostStore.downvotePost", "store.sql_post.downvotePost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	// prevent self point gain
	if userId == post.UserId {
		return vote, nil
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points + :PointForDownVoted, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForDownVoted": model.USER_POINT_FOR_DOWN_VOTED, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return nil, model.NewAppError("SqlPostStore.downvotePost", "store.sql_post.downvotePost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points + :PointForDownVoted WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForDownVoted": model.USER_POINT_FOR_DOWN_VOTED, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return nil, model.NewAppError("SqlPostStore.downvotePost", "store.sql_post.downvotePost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	var tags string
	var appErr *model.AppError
	if tags, appErr = s.getTagsForPost(post); appErr != nil {
		return nil, appErr
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_DOWN_VOTED,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     tags,
		Points:   model.USER_POINT_FOR_DOWN_VOTED,
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return vote, nil
}

func (s *SqlPostStore) CancelDownVotePost(postId string, userId string) (*model.Vote, *model.AppError) {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.CancelDownVotePost", "store.sql_post.cancel_downvote_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var vote *model.Vote
	err := s.GetReplica().SelectOne(&vote, "SELECT * FROM Votes WHERE PostId = :PostId AND Type = :Type AND UserId = :UserId", map[string]interface{}{"PostId": postId, "Type": model.VOTE_TYPE_DOWN_VOTE, "UserId": userId})
	if err != nil {
		return nil, appErr(err.Error())
	}
	if vote == nil {
		return nil, model.NewAppError("SqlPostStore.CancelDownvotePost", "store.sql_post.cancel_downvote_post.select.app_error", nil, "", http.StatusInternalServerError)
	}

	var post *model.Post
	err = s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return nil, appErr(err.Error())
	}
	if post == nil {
		return nil, model.NewAppError("SqlPostStore.CancelDownvotePost", "store.sql_post.cancel_downvote_post.select.app_error", nil, "", http.StatusInternalServerError)
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.CancelDownVotePost", "store.sql_post.cancel_downvote_post.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)
	if upsertErr := s.cancelDownvotePost(transaction, vote, post, userId); upsertErr != nil {
		return nil, upsertErr
	}

	if err = transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.CancelDownVotePost", "store.sql_post.cancel_downvote_post.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return vote, nil
}

func (s *SqlPostStore) cancelDownvotePost(transaction *gorp.Transaction, vote *model.Vote, post *model.Post, userId string) *model.AppError {
	if _, err := transaction.Delete(vote); err != nil {
		return model.NewAppError("SqlPostStore.CancelDownVotePost", "store.sql_post.cancel_downvote_post.deleting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()

	post.DownVotes = post.DownVotes - 1
	if post.DownVotes < 0 {
		post.DownVotes = 0
	}
	if _, err := transaction.Exec("UPDATE Posts SET DownVotes = :DownVotes, Points = Points + 1, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"DownVotes": post.DownVotes, "UpdateAt": curTime, "Id": post.Id}); err != nil {
		return model.NewAppError("SqlPostStore.CancelDownVotePost", "store.sql_post.cancel_downvote_post.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	// prevent self point gain
	if userId == post.UserId {
		return nil
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points - :PointForDownVoted, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForDownVoted": model.USER_POINT_FOR_DOWN_VOTED, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.CancelDownvotePost", "store.sql_post.cancel_downvote_post.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points - :PointForDownVoted WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForDownVoted": model.USER_POINT_FOR_DOWN_VOTED, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.CancelDownvotePost", "store.sql_post.cancel_downvote_post.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	var tags string
	var appErr *model.AppError
	if tags, appErr = s.getTagsForPost(post); appErr != nil {
		return appErr
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_DOWN_VOTED_CANCELED,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     tags,
		Points:   -(model.USER_POINT_FOR_DOWN_VOTED),
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return nil
}

func (s *SqlPostStore) FlagPost(postId string, userId string) (*model.Vote, *model.AppError) {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.FlagPost", "store.sql_post.flag_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return nil, appErr(err.Error())
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.FlagPost", "store.sql_post.flag_post.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)

	var vote *model.Vote
	var upsertErr *model.AppError
	if vote, upsertErr = s.flagPost(transaction, post, userId); upsertErr != nil {
		return nil, upsertErr
	}

	if err := transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.FlagPost", "store.sql_post.flag_post.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return vote, nil
}

func (s *SqlPostStore) flagPost(transaction *gorp.Transaction, post *model.Post, userId string) (*model.Vote, *model.AppError) {
	curTime := model.GetMillis()

	var rev int64
	var err *model.AppError
	if rev, err = s.GetCurrentRevisionForPost(post.Id, post.TeamId); err != nil {
		return nil, model.NewAppError("SqlPostStore.flagPost", "store.sql_post.flagPost.get_revision.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	flag := &model.Vote{
		PostId:       post.Id,
		UserId:       userId,
		Type:         model.VOTE_TYPE_FLAG,
		TeamId:       post.TeamId,
		CreateAt:     curTime,
		FirstPostRev: int(rev),
	}

	if err := transaction.Insert(flag); err != nil {
		return nil, model.NewAppError("SqlPostStore.flagPost", "store.sql_post.flagPost.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if _, err := transaction.Exec("UPDATE Posts SET FlagCount = FlagCount + 1, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"UpdateAt": curTime, "Id": post.Id}); err != nil {
		return nil, model.NewAppError("SqlPostStore.flagPost", "store.sql_post.flagPost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points + :PointForFlagged, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForFlagged": model.USER_POINT_FOR_FLAGGED, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return nil, model.NewAppError("SqlPostStore.flagPost", "store.sql_post.flagPost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points + :PointForFlagged WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForFlagged": model.USER_POINT_FOR_FLAGGED, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return nil, model.NewAppError("SqlPostStore.flagPost", "store.sql_post.flagPost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	var tags string
	var appErr *model.AppError
	if tags, appErr = s.getTagsForPost(post); appErr != nil {
		return nil, appErr
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_FLAGGED,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     tags,
		Points:   model.USER_POINT_FOR_FLAGGED,
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return flag, nil
}

func (s *SqlPostStore) CancelFlagPost(postId string, userId string) (*model.Vote, *model.AppError) {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.CancelFlagPost", "store.sql_post.cancel_flag_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var flag *model.Vote
	err := s.GetReplica().SelectOne(&flag, "SELECT * FROM Votes WHERE PostId = :PostId AND Type = :Type AND UserId = :UserId", map[string]interface{}{"PostId": postId, "Type": model.VOTE_TYPE_FLAG, "UserId": userId})
	if err != nil {
		return nil, appErr(err.Error())
	}
	if flag == nil {
		return nil, model.NewAppError("SqlPostStore.CancelFlagPost", "store.sql_post.cancel_flag_post.select.app_error", nil, "", http.StatusInternalServerError)
	}

	var post *model.Post
	err = s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return nil, appErr(err.Error())
	}
	if post == nil {
		return nil, model.NewAppError("SqlPostStore.CancelFlagPost", "store.sql_post.cancel_flag_post.select.app_error", nil, "", http.StatusInternalServerError)
	}

	transaction, err := s.GetMaster().Begin()
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.CancelFlagPost", "store.sql_post.cancel_flag_post.open_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	defer finalizeTransaction(transaction)
	if upsertErr := s.cancelFlagPost(transaction, flag, post, userId); upsertErr != nil {
		return nil, upsertErr
	}

	if err = transaction.Commit(); err != nil {
		return nil, model.NewAppError("SqlPostStore.CancelFlagPost", "store.sql_post.cancel_flag_post.commit_transaction.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return flag, nil
}

func (s *SqlPostStore) cancelFlagPost(transaction *gorp.Transaction, flag *model.Vote, post *model.Post, userId string) *model.AppError {
	if _, err := transaction.Delete(flag); err != nil {
		return model.NewAppError("SqlPostStore.CancelFlagPost", "store.sql_post.cancel_flag_post.deleting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	curTime := model.GetMillis()

	if _, err := transaction.Exec("UPDATE Posts SET FlagCount = FlagCount - 1, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"UpdateAt": curTime, "Id": post.Id}); err != nil {
		return model.NewAppError("SqlPostStore.cancelFlagPost", "store.sql_post.cancelFlagPost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	if len(post.TeamId) == 0 {
		if _, err := transaction.Exec("UPDATE Users SET Points = Points - :PointForFlagged, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"PointForFlagged": model.USER_POINT_FOR_FLAGGED, "UpdateAt": curTime, "Id": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.cancelFlagPost", "store.sql_post.cancelFlagPost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if _, err := transaction.Exec("UPDATE TeamMembers SET Points = Points - :PointForFlagged WHERE TeamId = :TeamId AND UserId = :UserId AND DeleteAt = 0", map[string]interface{}{"PointForFlagged": model.USER_POINT_FOR_FLAGGED, "TeamId": post.TeamId, "UserId": post.UserId}); err != nil {
			return model.NewAppError("SqlPostStore.cancelFlagPost", "store.sql_post.cancelFlagPost.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	var tags string
	var appErr *model.AppError
	if tags, appErr = s.getTagsForPost(post); appErr != nil {
		return appErr
	}

	user_point_history := &model.UserPointHistory{
		Id:       model.NewId(),
		TeamId:   post.TeamId,
		UserId:   post.UserId,
		Type:     model.USER_POINT_TYPE_FLAGGED_CANCELED,
		PostId:   post.Id,
		PostType: post.Type,
		Tags:     tags,
		Points:   -(model.USER_POINT_FOR_FLAGGED),
		CreateAt: curTime,
	}
	s.SaveUserPointHistory(user_point_history)

	return nil
}

func (s *SqlPostStore) LockPost(postId string, time int64, userId string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.LockPost", "store.sql_post.lock_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	post.AddProp(model.POST_PROPS_LOCKED_BY, userId)

	if _, err := s.GetMaster().Exec("UPDATE Posts SET LockedAt = :LockedAt, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"LockedAt": time, "UpdateAt": time, "Id": postId, "Props": model.StringInterfaceToJson(post.Props)}); err != nil {
		return appErr(err.Error())
	}

	return nil
}

func (s *SqlPostStore) CancelLockPost(postId string, userId string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.CancelLockPost", "store.sql_post.cancel_lock_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	post.AddProp(model.POST_PROPS_LOCKED_BY, "")

	curTime := model.GetMillis()
	if _, err := s.GetMaster().Exec("UPDATE Posts SET LockedAt = 0, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"UpdateAt": curTime, "Id": postId, "Props": model.StringInterfaceToJson(post.Props)}); err != nil {
		return appErr(err.Error())
	}

	return nil
}

func (s *SqlPostStore) ProtectPost(postId string, time int64, userId string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.ProtectPost", "store.sql_post.protect_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	post.AddProp(model.POST_PROPS_PROTECTED_BY, userId)

	if _, err := s.GetMaster().Exec("UPDATE Posts SET ProtectedAt = :ProtectedAt, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"ProtectedAt": time, "UpdateAt": time, "Id": postId, "Props": model.StringInterfaceToJson(post.Props)}); err != nil {
		return appErr(err.Error())
	}

	return nil
}

func (s *SqlPostStore) CancelProtectPost(postId string, userId string) *model.AppError {
	appErr := func(errMsg string) *model.AppError {
		return model.NewAppError("SqlPostStore.CancelProtectPost", "store.sql_post.cancel_protect_post.app_error", nil, "id="+postId+", err="+errMsg, http.StatusInternalServerError)
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, "SELECT * FROM Posts WHERE Id = :Id AND DeleteAt = 0", map[string]interface{}{"Id": postId})
	if err != nil {
		return appErr(err.Error())
	}

	post.AddProp(model.POST_PROPS_PROTECTED_BY, "")

	curTime := model.GetMillis()
	if _, err := s.GetMaster().Exec("UPDATE Posts SET ProtectedAt = 0, UpdateAt = :UpdateAt, Props = :Props WHERE Id = :Id", map[string]interface{}{"UpdateAt": curTime, "Id": postId, "Props": model.StringInterfaceToJson(post.Props)}); err != nil {
		return appErr(err.Error())
	}

	return nil
}

func (s *SqlPostStore) ViewPost(postId string, teamId string, userId string, ipAddress string, count int) *model.AppError {
	curTime := model.GetMillis()

	if _, err := s.GetMaster().Exec("UPDATE Posts SET Views = Views + :ViewsCount, UpdateAt = :UpdateAt WHERE Id = :Id", map[string]interface{}{"ViewsCount": count, "UpdateAt": curTime, "Id": postId}); err != nil {
		return model.NewAppError("SqlPostStore.ViewPost", "store.sql_post.view_post.updating.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	s.SavePostViewsHistory(postId, teamId, userId, ipAddress, count, curTime)

	return nil
}

func (s *SqlPostStore) SavePostViewsHistory(postId string, teamId string, userId string, ipAddress string, count int, time int64) (*model.PostViewsHistory, *model.AppError) {
	post_views_history := &model.PostViewsHistory{
		Id:         model.NewId(),
		PostId:     postId,
		TeamId:     teamId,
		UserId:     userId,
		IpAddress:  ipAddress,
		ViewsCount: count,
		CreateAt:   time,
	}

	if err := s.GetMaster().Insert(post_views_history); err != nil {
		return nil, model.NewAppError("SqlPostStore.SavePostViewsHistory", "store.sql_post.save_post_views_history.inserting.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return post_views_history, nil
}

func (s *SqlPostStore) RelatedSearch(term string, limit int) ([]*model.RelatedPostSearchResult, *model.AppError) {
	return nil, nil
}

func (s *SqlPostStore) HotSearch(interval string, teamId string, limit int) ([]string, *model.AppError) {
	return nil, nil
}

func (s *SqlPostStore) GetCurrentRevisionForPost(postId, teamId string) (int64, *model.AppError) {
	args := map[string]interface{}{"OriginalId": postId}

	teamFilter := ""
	if teamId == "" {
		teamFilter = "AND TeamId IS NULL"
	} else {
		teamFilter = "AND TeamId = :TeamId"
		args["TeamId"] = teamId
	}

	count, err := s.GetReplica().SelectInt(`
		SELECT
			count(*)
		FROM
			Posts
		WHERE
			OriginalId = :OriginalId
			`+teamFilter+`
			`, args)

	if err != nil {
		return 0, model.NewAppError("SqlPostStore.GetCurrentRevisionForPost", "store.sql_post.get_revisions_total_count_for_post.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return count + 1, nil
}

func (s *SqlPostStore) GetRevisionPost(postId, teamId string, offset int) (*model.Post, *model.AppError) {
	args := map[string]interface{}{"OriginalId": postId, "Limit": 1, "Offset": offset}

	teamFilter := ""
	if teamId == "" {
		teamFilter = "AND TeamId IS NULL"
	} else {
		teamFilter = "AND TeamId = :TeamId"
		args["TeamId"] = teamId
	}

	var post *model.Post
	err := s.GetReplica().SelectOne(&post, `
		SELECT
			*
		FROM
			Posts
		WHERE
			OriginalId = :OriginalId
			`+teamFilter+`
		ORDER BY
			UpdateAt ASC
		LIMIT
			:Limit
		OFFSET
			:Offset
			`, args)

	if err != nil {
		return nil, model.NewAppError("SqlPostStore.GetRevisionPost", "store.sql_post.get_revision_post.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return post, nil
}

func (s *SqlPostStore) GetAnsweredRate(teamId string) (float64, *model.AppError) {
	args := map[string]interface{}{"Type": model.POST_TYPE_ANSWER}

	teamFilter := ""
	if teamId == "" {
		teamFilter = "AND TeamId IS NULL"
	} else {
		teamFilter = "AND TeamId = :TeamId"
		args["TeamId"] = teamId
	}

	var answeredCount int64
	answeredCount, err := s.GetReplica().SelectInt(`
		SELECT
			COUNT(DISTINCT ParentId)
		FROM
			Posts
		WHERE
			Type = :Type
			`+teamFilter+`
			AND DeleteAt = 0
			`, args)

	if err != nil {
		return float64(0), model.NewAppError("SqlPostStore.GetAnsweredRate", "store.sql_post.get_answered_rate.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	var totalCount int64
	var appErr *model.AppError
	if totalCount, appErr = s.GetPostCount(model.POST_TYPE_QUESTION, "", teamId, 0, 0); appErr != nil {
		return float64(0), appErr
	}

	rate := float64(answeredCount) / float64(totalCount)

	return rate, nil
}

func (s *SqlPostStore) AnalyticsPostCounts(teamId string) (model.Analytics, *model.AppError) {
	query :=
		`SELECT
		        DATE(FROM_UNIXTIME(Posts.CreateAt / 1000)) AS Name,
		        COUNT(Posts.Id) AS Value
		    FROM Posts`

	if len(teamId) > 0 {
		query += " WHERE TeamId = :TeamId AND"
	} else {
		query += " WHERE"
	}

	query += ` Posts.CreateAt <= :EndTime
		            AND Posts.CreateAt >= :StartTime
		GROUP BY DATE(FROM_UNIXTIME(Posts.CreateAt / 1000))
		ORDER BY Name DESC
		LIMIT 30`

	end := utils.MillisFromTime(utils.EndOfDay(utils.Yesterday()))
	start := utils.MillisFromTime(utils.StartOfDay(utils.Yesterday().AddDate(0, 0, -31)))

	var rows model.Analytics
	_, err := s.GetReplica().Select(
		&rows,
		query,
		map[string]interface{}{"TeamId": teamId, "StartTime": start, "EndTime": end})
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.AnalyticsPostCounts", "store.sql_post.analytics_post_counts.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return rows, nil
}

func (s *SqlPostStore) AnalyticsActiveAuthorCounts(teamId string) (model.Analytics, *model.AppError) {
	query :=
		`SELECT
		        DATE(FROM_UNIXTIME(Posts.CreateAt / 1000)) AS Name,
		        COUNT(DISTINCT Posts.UserId) AS Value
		    FROM Posts`

	if len(teamId) > 0 {
		query += " WHERE TeamId = :TeamId AND"
	} else {
		query += " WHERE"
	}

	query += ` Posts.CreateAt <= :EndTime
		            AND Posts.CreateAt >= :StartTime
		GROUP BY DATE(FROM_UNIXTIME(Posts.CreateAt / 1000))
		ORDER BY Name DESC
		LIMIT 30`

	end := utils.MillisFromTime(utils.EndOfDay(utils.Yesterday()))
	start := utils.MillisFromTime(utils.StartOfDay(utils.Yesterday().AddDate(0, 0, -31)))

	var rows model.Analytics
	_, err := s.GetReplica().Select(
		&rows,
		query,
		map[string]interface{}{"TeamId": teamId, "StartTime": start, "EndTime": end})
	if err != nil {
		return nil, model.NewAppError("SqlPostStore.AnalyticsActiveAuthorCounts", "store.sql_post.analytics_active_author_count.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return rows, nil
}
