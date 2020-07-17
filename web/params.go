package web

import (
	"net/http"
	"strconv"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/gorilla/mux"
)

const (
	PAGE_DEFAULT     = 0
	PER_PAGE_DEFAULT = 20
	PER_PAGE_MAXIMUM = 100
)

type Params struct {
	UserId          string
	PostId          string
	BestId          string
	Page            int
	PerPage         int
	FromDate        int64
	ToDate          int64
	Min             *int
	Max             *int
	NoAnswers       bool
	SortType        string
	Body            string
	InboxMessageId  string
	SuspendSpanType string
	UserType        string
	UserName        string
	Filename        string
	FileId          string
	InboxInterval   string
	TagName         string
	TimeZoneOffset  *int
}

func ParamsFromRequest(r *http.Request) *Params {
	params := &Params{}

	props := mux.Vars(r)

	query := r.URL.Query()

	if val, ok := props["user_id"]; ok {
		params.UserId = val
	}

	if val, ok := props["post_id"]; ok {
		params.PostId = val
	}

	params.BestId = query.Get("best_id")

	if val, err := strconv.ParseInt(query.Get("from_date"), 10, 64); err == nil && val > 0 {
		params.FromDate = val
	}

	if val, err := strconv.ParseInt(query.Get("to_date"), 10, 64); err == nil && val > 0 {
		params.ToDate = val
	}

	if val, err := strconv.Atoi(query.Get("min")); err == nil {
		params.Min = &val
	}

	if val, err := strconv.Atoi(query.Get("max")); err == nil {
		params.Max = &val
	}

	if val, err := strconv.Atoi(query.Get("page")); err != nil || val < 0 {
		params.Page = PAGE_DEFAULT
	} else {
		params.Page = val
	}

	if val, err := strconv.Atoi(query.Get("per_page")); err != nil || val < 0 {
		params.PerPage = PER_PAGE_DEFAULT
	} else if val > PER_PAGE_MAXIMUM {
		params.PerPage = PER_PAGE_MAXIMUM
	} else {
		params.PerPage = val
	}

	if val := query.Get("sort"); len(val) > 0 {
		switch val {
		case model.POST_SORT_TYPE_VOTES:
			params.SortType = model.POST_SORT_TYPE_VOTES
		case model.POST_SORT_TYPE_CREATION:
			params.SortType = model.POST_SORT_TYPE_CREATION
		case model.POST_SORT_TYPE_ACTIVE:
			params.SortType = model.POST_SORT_TYPE_ACTIVE
		case model.POST_SORT_TYPE_ANSWERS:
			params.SortType = model.POST_SORT_TYPE_ANSWERS
		case model.POST_SORT_TYPE_NAME:
			params.SortType = model.POST_SORT_TYPE_NAME
		case model.POST_SORT_TYPE_POPULAR:
			params.SortType = model.POST_SORT_TYPE_POPULAR
		}
	}

	if val, err := strconv.ParseBool(query.Get("no_answers")); err == nil {
		params.NoAnswers = val
	}

	if val, ok := props["body"]; ok {
		params.Body = val
	}

	if val, ok := props["inbox_message_id"]; ok {
		params.InboxMessageId = val
	}

	if val := query.Get("suspend_span_type"); len(val) > 0 {
		switch val {
		case model.SUSPEND_SPAN_TYPE_WEEK:
			params.SuspendSpanType = model.SUSPEND_SPAN_TYPE_WEEK
		case model.SUSPEND_SPAN_TYPE_MONTH:
			params.SuspendSpanType = model.SUSPEND_SPAN_TYPE_MONTH
		case model.SUSPEND_SPAN_TYPE_QUARTER:
			params.SuspendSpanType = model.SUSPEND_SPAN_TYPE_QUARTER
		case model.SUSPEND_SPAN_TYPE_HALF_YEAR:
			params.SuspendSpanType = model.SUSPEND_SPAN_TYPE_HALF_YEAR
		case model.SUSPEND_SPAN_TYPE_YEAR:
			params.SuspendSpanType = model.SUSPEND_SPAN_TYPE_YEAR
		}
	}

	if val := query.Get("user_type"); len(val) > 0 {
		switch val {
		case model.USER_TYPE_NORMAL:
			params.UserType = model.USER_TYPE_NORMAL
		case model.USER_TYPE_MODERATOR:
			params.UserType = model.USER_TYPE_MODERATOR
		case model.USER_TYPE_ADMIN:
			params.UserType = model.USER_TYPE_ADMIN
		}
	}

	if val := query.Get("user_name"); len(val) > 0 {
		params.UserName = val
	}

	params.Filename = query.Get("filename")

	if val, ok := props["file_id"]; ok {
		params.FileId = val
	}

	if val := query.Get("inbox_interval"); len(val) > 0 {
		switch val {
		case model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR:
			params.InboxInterval = model.NOTIFICATION_INBOX_INTERVAL_THREE_HOUR
		case model.NOTIFICATION_INBOX_INTERVAL_DAY:
			params.InboxInterval = model.NOTIFICATION_INBOX_INTERVAL_DAY
		case model.NOTIFICATION_INBOX_INTERVAL_WEEK:
			params.InboxInterval = model.NOTIFICATION_INBOX_INTERVAL_WEEK
		}
	}

	if val := query.Get("tag_name"); len(val) > 0 {
		params.TagName = val
	}

	if val, err := strconv.Atoi(query.Get("timezone_offset")); err == nil {
		params.TimeZoneOffset = &val
	}

	return params
}
