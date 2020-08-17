package model

import (
	"encoding/json"
	"io"
	"net/http"
	"unicode/utf8"
)

const (
	POST_TYPE_QUESTION = "question"
	POST_TYPE_ANSWER   = "answer"
	POST_TYPE_COMMENT  = "comment"
	// take care of mysql's ft_min_word_len
	POST_TITLE_MIN_RUNES = 5
	POST_TITLE_MAX_RUNES = 1000
	// take care of mysql's ft_min_word_len
	POST_CONTENT_MIN_RUNES  = 5
	POST_CONTENT_MAX_RUNES  = 4000
	POST_PROPS_MAX_RUNES    = 4000
	POST_SEARCH_TERMS_MAX   = 200
	POST_SEARCH_MAX_COUNT   = 500
	POST_COMMENT_LIMIT      = 20
	POST_PROPS_DELETE_BY    = "deleteBy"
	POST_PROPS_LOCKED_BY    = "lockedBy"
	POST_PROPS_PROTECTED_BY = "protectedBy"

	POST_SORT_TYPE_VOTES     = "votes"
	POST_SORT_TYPE_ANSWERS   = "answers"
	POST_SORT_TYPE_CREATION  = "creation"
	POST_SORT_TYPE_ACTIVE    = "active"
	POST_SORT_TYPE_NAME      = "name"
	POST_SORT_TYPE_POPULAR   = "popular"
	POST_SORT_TYPE_RELEVANCE = "relevance"
)

// TODO: question's views count:
// There is a buffer in memory which accumulates counter hit data
// then periodically writes it to the database where it updates the [Views] column of each question involved.
// The buffer has some fixed size. It's flushed when either it's filled up or when a predetermined time interval elapses.
//
// キャッシュはアプリとDBの中間レイヤー。
// 計3種類用意する。
//
// もしバッファーAが無かったなら、
// (IP or UserId) + QuestionId をキーとし、
// ttl時間(15分)設定したバッファーAを用意する。
// もしバッファーAが有ったなら、以下の(1),(2)の手順に従う。
//
// QuestionId + "previous" をキーとし、1日の特定の15分間隔を示すハッシュ値 を管理する別のバッファーB と
// QuestionId + "counters" をキーとし、counter値 を管理する別のバッファーC
// を用意しておき、
// (1)現在時刻がバッファーBのハッシュ値と同じ期間だった場合、またはバッファーBが無かった場合は、
// バッファーCのcounter値をインクリメントする。
// (2)現在時刻がバッファーBのハッシュ値と異なる期間だった場合、
// またはバッファーCのcounter値が一定数を超えた場合は、Post.Viewsをcounter値の分だけインクリメント更新し、バッファーB,Cを削除する。
//
// → QuestionId + 15_min_hashをキーにしてredisのKEYSコマンドで検索すれば、バッファーBは不要かも。
//
// ElasticCache for redis:
// When the amount of data exceeds the configured maxmemory setting, Redis has different ways of responding depending on the selected eviction policy. By default, ElastiCache for Redis is configured to remove from memory the least recently used keys with a ttl set. The eviction policy parameter is called maxmemory-policy, and the default value in ElastiCache is volatile-lru. Another interesting option for this use case is the volatile-ttl policy, which instructs Redis to reclaim memory by removing those keys with the shortest ttl.
type Post struct {
	Id          string          `db:"Id, primarykey" json:"id"`
	Type        string          `db:"Type" json:"type"`
	RootId      string          `db:"RootId" json:"root_id"`
	ParentId    string          `db:"ParentId" json:"parent_id"`
	OriginalId  string          `db:"OriginalId" json:"original_id,omitempty"`
	BestId      string          `db:"BestId" json:"best_id,omitempty"`
	UserId      string          `db:"UserId" json:"user_id"`
	TeamId      string          `db:"TeamId" json:"team_id"`
	Title       string          `db:"Title" json:"title,omitempty"`
	Content     string          `db:"Content" json:"content"`
	Tags        string          `db:"Tags" json:"tags,omitempty"`
	Props       StringInterface `db:"Props" json:"-"`
	UpVotes     int             `db:"UpVotes" json:"up_votes,omitempty"`
	DownVotes   int             `db:"DownVotes" json:"down_votes,omitempty"`
	Points      int             `db:"Points" json:"points,omitempty"`
	AnswerCount int             `db:"AnswerCount" json:"answer_count,omitempty"`
	FlagCount   int             `db:"FlagCount" json:"flag_count,omitempty"`
	ProtectedAt int64           `db:"ProtectedAt" json:"protected_at,omitempty"`
	LockedAt    int64           `db:"LockedAt" json:"locked_at,omitempty"`
	CreateAt    int64           `db:"CreateAt" json:"create_at"`
	UpdateAt    int64           `db:"UpdateAt" json:"update_at"`
	EditAt      int64           `db:"EditAt" json:"edit_at"`
	DeleteAt    int64           `db:"DeleteAt" json:"delete_at"`

	// whether my favorite post or not
	Favorited     bool  `json:"favorited,omitempty" db:"-"`
	FavoriteCount int64 `json:"favorite_count,omitempty" db:"-"`
	UpVoted       bool  `json:"up_voted,omitempty" db:"-"`
	DownVoted     bool  `json:"down_voted,omitempty" db:"-"`
	Flagged       bool  `json:"flagged,omitempty" db:"-"`

	Metadata *PostMetadata `json:"metadata,omitempty" db:"-"`
}

func (o *Post) Clone() *Post {
	copy := *o
	return &copy
}

func (o *Post) ToJson() string {
	copy := o.Clone()
	b, _ := json.Marshal(copy)
	return string(b)
}

func (o *Post) ToUnsanitizedJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func PostFromJson(data io.Reader) *Post {
	var o *Post
	json.NewDecoder(data).Decode(&o)
	return o
}

func (o *Post) IsValid(maxPostSize int) *AppError {
	if len(o.Id) != 26 {
		return NewAppError("Post.IsValid", "model.post.is_valid.id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.CreateAt == 0 {
		return NewAppError("Post.IsValid", "model.post.is_valid.create_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if o.UpdateAt == 0 {
		return NewAppError("Post.IsValid", "model.post.is_valid.update_at.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if len(o.UserId) != 26 {
		return NewAppError("Post.IsValid", "model.post.is_valid.user_id.app_error", nil, "", http.StatusBadRequest)
	}

	if o.TeamId != "" && len(o.TeamId) != 26 {
		return NewAppError("Post.IsValid", "model.post.is_valid.team_id.app_error", nil, "", http.StatusBadRequest)
	}

	if !(len(o.OriginalId) == 26 || len(o.OriginalId) == 0) {
		return NewAppError("Post.IsValid", "model.post.is_valid.original_id.app_error", nil, "", http.StatusBadRequest)
	}

	if utf8.RuneCountInString(o.Content) > maxPostSize || utf8.RuneCountInString(o.Content) < POST_CONTENT_MIN_RUNES {
		return NewAppError("Post.IsValid", "model.post.is_valid.content.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	if utf8.RuneCountInString(StringInterfaceToJson(o.Props)) > POST_PROPS_MAX_RUNES {
		return NewAppError("Post.IsValid", "model.post.is_valid.props.app_error", nil, "id="+o.Id, http.StatusBadRequest)
	}

	switch o.Type {
	case
		POST_TYPE_QUESTION:
		if utf8.RuneCountInString(o.Title) > POST_TITLE_MAX_RUNES || utf8.RuneCountInString(o.Title) < POST_TITLE_MIN_RUNES {
			return NewAppError("Post.IsValid", "model.post.is_valid.title.app_error", nil, "id="+o.Id, http.StatusBadRequest)
		}

		if len(o.Tags) > 0 && (utf8.RuneCountInString(o.Tags) > TAG_MAX_RUNES || utf8.RuneCountInString(o.Tags) < TAG_MIN_RUNES) {
			return NewAppError("Post.IsValid", "model.post.is_valid.tags.app_error", nil, "id="+o.Id, http.StatusBadRequest)
		}
	case
		POST_TYPE_ANSWER:
		if len(o.ParentId) != 26 || len(o.RootId) != 26 {
			return NewAppError("Post.IsValid", "model.post.is_valid.parents_id.app_error", nil, "", http.StatusBadRequest)
		}
	case
		POST_TYPE_COMMENT:
		if len(o.ParentId) != 26 || len(o.RootId) != 26 {
			return NewAppError("Post.IsValid", "model.post.is_valid.parents_id.app_error", nil, "", http.StatusBadRequest)
		}
	default:
		return NewAppError("Post.IsValid", "model.post.is_valid.type.app_error", nil, "id="+o.Type, http.StatusBadRequest)
	}

	return nil
}

func (o *Post) PreSave() {
	if o.Id == "" {
		o.Id = NewId()
	}

	o.OriginalId = ""

	if o.CreateAt == 0 {
		o.CreateAt = GetMillis()
	}

	o.UpdateAt = o.CreateAt

	o.MakeNonNil()
}

func (o *Post) IsLocked() bool {
	return o.LockedAt > 0
}

func (o *Post) IsProtected() bool {
	return o.ProtectedAt > 0
}

func (o *Post) MakeNonNil() {
	if o.Props == nil {
		o.Props = make(map[string]interface{})
	}
}

func (o *Post) AddProp(key string, value interface{}) {
	o.MakeNonNil()
	o.Props[key] = value
}

func GetLink(siteURL string, postId string) string {
	return siteURL + "/questions/" + postId
}

func IsQuestionOrAnswer(postType string) bool {
	if postType == POST_TYPE_QUESTION || postType == POST_TYPE_ANSWER {
		return true
	}

	return false
}

// tag or title or link can be used as TermsType
type GetPostsOptions struct {
	FromDate  int64
	ToDate    int64
	PostType  string
	ParentId  string
	UserId    string
	SortType  string
	Min       *int
	Max       *int
	Tagged    string
	Title     string
	Link      string
	NoAnswers bool
	Page      int
	PerPage   int
	TeamId    string
}

type SearchPostsOptions struct {
	Terms         string
	ExcludedTerms string
	TermsType     string
	UserId        string
	SortType      string
	MinVotes      *int
	MaxVotes      *int
	MinAnswers    *int
	MaxAnswers    *int
	PostType      string
	Ids           []string
	ParentId      string
	FromDate      int64
	ToDate        int64
	Page          int
	PerPage       int
	TeamId        string
}

func (o *GetPostsOptions) GetPostsOptionsToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	}

	return string(b)
}

type AdvancedSearchParameter struct {
	Terms *string `json:"terms"`
}

func (o *AdvancedSearchParameter) AdvancedSearchParameterToJson() string {
	b, err := json.Marshal(o)
	if err != nil {
		return ""
	}

	return string(b)
}

func AdvancedSearchParameterFromJson(data io.Reader) *AdvancedSearchParameter {
	decoder := json.NewDecoder(data)
	var searchParam AdvancedSearchParameter
	err := decoder.Decode(&searchParam)
	if err != nil {
		return nil
	}

	return &searchParam
}
