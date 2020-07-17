package model

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	HEADER_REQUEST_ID         = "X-Request-ID"
	HEADER_VERSION_ID         = "X-Version-ID"
	HEADER_REQUESTED_WITH     = "X-Requested-With"
	HEADER_REQUESTED_WITH_XML = "XMLHttpRequest"
	HEADER_TOKEN              = "token"
	HEADER_CSRF_TOKEN         = "X-CSRF-Token"
	HEADER_FORWARDED          = "X-Forwarded-For"
	HEADER_REAL_IP            = "X-Real-IP"
	HEADER_FORWARDED_PROTO    = "X-Forwarded-Proto"
	HEADER_AUTH               = "Authorization"
	HEADER_BEARER             = "BEARER"

	STATUS      = "status"
	STATUS_OK   = "OK"
	STATUS_FAIL = "FAIL"

	API_URL_SUFFIX = "/api/v1"
)

type Response struct {
	StatusCode    int
	Error         *AppError
	RequestId     string
	ServerVersion string
	Header        http.Header
}

type Client struct {
	Url           string
	ApiUrl        string
	HttpClient    *http.Client
	AuthToken     string
	AuthType      string
	HttpHeader    map[string]string
	SessionCookie string
	UserCookie    string
	CsrfCookie    string

	trueString  string
	falseString string
}

func NewAPIClient(url string) *Client {
	return &Client{url, url + API_URL_SUFFIX, &http.Client{}, "", "", map[string]string{}, "", "", "", "", ""}
}

func (c *Client) SetBoolString(value bool, valueStr string) {
	if value {
		c.trueString = valueStr
	} else {
		c.falseString = valueStr
	}
}

func BuildErrorResponse(r *http.Response, err *AppError) *Response {
	var statusCode int
	var header http.Header
	if r != nil {
		statusCode = r.StatusCode
		header = r.Header
	} else {
		statusCode = 0
		header = make(http.Header)
	}

	return &Response{
		StatusCode: statusCode,
		Error:      err,
		Header:     header,
	}
}

func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = io.Copy(ioutil.Discard, r.Body)
		_ = r.Body.Close()
	}
}

func BuildResponse(r *http.Response) *Response {
	return &Response{
		StatusCode:    r.StatusCode,
		RequestId:     r.Header.Get(HEADER_REQUEST_ID),
		ServerVersion: r.Header.Get(HEADER_VERSION_ID),
		Header:        r.Header,
	}
}

func (c *Client) GetPostsRoute() string {
	return "/posts"
}

func (c *Client) CreateQuestionRoute() string {
	return fmt.Sprintf(c.GetPostsRoute() + "/question")
}

func (c *Client) CreateAnswerRoute() string {
	return fmt.Sprintf(c.GetPostsRoute() + "/answer")
}

func (c *Client) CreateCommentRoute() string {
	return fmt.Sprintf(c.GetPostsRoute() + "/comment")
}

func (c *Client) SearchPostsRoute() string {
	return fmt.Sprintf(c.GetPostsRoute() + "/search")
}

func (c *Client) AdvancedSearchPostsRoute() string {
	return fmt.Sprintf(c.GetPostsRoute() + "/advanced_search")
}

func (c *Client) CreateQuestion(post *Post) (*Post, *Response) {
	r, err := c.DoApiPost(c.CreateQuestionRoute(), post.ToUnsanitizedJson())
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostFromJson(r.Body), BuildResponse(r)
}

func (c *Client) CreateAnswer(post *Post) (*Post, *Response) {
	r, err := c.DoApiPost(c.CreateAnswerRoute(), post.ToUnsanitizedJson())
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostFromJson(r.Body), BuildResponse(r)
}

func (c *Client) CreateComment(post *Post) (*Post, *Response) {
	r, err := c.DoApiPost(c.CreateCommentRoute(), post.ToUnsanitizedJson())
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostFromJson(r.Body), BuildResponse(r)
}

func (c *Client) SearchPosts(requestBody map[string]string, sort string, noAnswers string) (*PostsWithCount, *Response) {
	query := fmt.Sprintf("?sort=%v&no_answers=%v", sort, noAnswers)
	r, err := c.DoApiPost(c.SearchPostsRoute()+query, MapToJson(requestBody))
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostsWithCountFromJson(r.Body), BuildResponse(r)
}

func (c *Client) AdvancedSearchPosts(requestBody map[string]string, sort string) (*PostsWithCount, *Response) {
	query := fmt.Sprintf("?sort=%v", sort)
	r, err := c.DoApiPost(c.AdvancedSearchPostsRoute()+query, MapToJson(requestBody))
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostsWithCountFromJson(r.Body), BuildResponse(r)
}

func (c *Client) FavoritePost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/favorite", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) CancelFavoritePost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/cancel_favorite", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) UpdatePost(postId string, post *Post) (*Post, *Response) {
	r, err := c.DoApiPut(c.GetPostRoute(postId), post.ToUnsanitizedJson())
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostFromJson(r.Body), BuildResponse(r)
}

func (c *Client) UpvotePost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/upvote", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) CancelUpvotePost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/cancel_upvote", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) DownvotePost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/downvote", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) CancelDownvotePost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/cancel_downvote", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) FlagPost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/flag", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) CancelFlagPost(postId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetPostRoute(postId)+"/cancel_flag", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) DoApiPost(url string, data string) (*http.Response, *AppError) {
	return c.DoApiRequest(http.MethodPost, c.ApiUrl+url, data)
}

func (c *Client) DoApiPut(url string, data string) (*http.Response, *AppError) {
	return c.DoApiRequest(http.MethodPut, c.ApiUrl+url, data)
}

func (c *Client) DoApiRequest(method, url, data string) (*http.Response, *AppError) {
	return c.doApiRequestReader(method, url, strings.NewReader(data))
}

func (c *Client) doApiRequestReader(method, url string, data io.Reader) (*http.Response, *AppError) {
	rq, err := http.NewRequest(method, url, data)

	if err != nil {
		return nil, NewAppError(url, "model.client.connecting.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	if len(c.AuthToken) > 0 {
		rq.Header.Set(HEADER_AUTH, c.AuthType+" "+c.AuthToken)
	}

	rq.Header.Set(HEADER_REQUESTED_WITH, HEADER_REQUESTED_WITH_XML)

	cookie := &http.Cookie{
		Name:  SESSION_COOKIE_USER,
		Value: c.UserCookie,
	}
	cookie2 := &http.Cookie{
		Name:  SESSION_COOKIE_TOKEN,
		Value: c.SessionCookie,
	}
	cookie3 := &http.Cookie{
		Name:  SESSION_COOKIE_CSRF,
		Value: c.CsrfCookie,
	}

	rq.AddCookie(cookie)
	rq.AddCookie(cookie2)
	rq.AddCookie(cookie3)
	rq.Header.Add(HEADER_CSRF_TOKEN, c.CsrfCookie)

	if c.HttpHeader != nil && len(c.HttpHeader) > 0 {
		for k, v := range c.HttpHeader {
			rq.Header.Set(k, v)
		}
	}

	rp, err := c.HttpClient.Do(rq)
	if err != nil || rp == nil {
		return nil, NewAppError(url, "model.client.connecting.app_error", nil, err.Error(), 0)
	}

	if rp.StatusCode == 304 {
		return rp, nil
	}

	if rp.StatusCode >= 300 {
		defer closeBody(rp)
		return rp, AppErrorFromJson(rp.Body)
	}

	return rp, nil
}

func (c *Client) login(m map[string]string) (*User, *Response) {
	r, err := c.DoApiPost("/users/login", MapToJson(m))
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	c.AuthToken = r.Header.Get(HEADER_TOKEN)
	c.AuthType = HEADER_BEARER

	for _, cookie := range r.Header["Set-Cookie"] {
		if match := regexp.MustCompile("^" + SESSION_COOKIE_TOKEN + "=([a-z0-9]+)").FindStringSubmatch(cookie); match != nil {
			c.SessionCookie = match[1]
		} else if match := regexp.MustCompile("^" + SESSION_COOKIE_USER + "=([a-z0-9]+)").FindStringSubmatch(cookie); match != nil {
			c.UserCookie = match[1]
		} else if match := regexp.MustCompile("^" + SESSION_COOKIE_CSRF + "=([a-z0-9]+)").FindStringSubmatch(cookie); match != nil {
			c.CsrfCookie = match[1]
		}
	}

	return UserFromJson(r.Body), BuildResponse(r)
}

// Login authenticates a user by login id, which can be username, email or some sort
// of SSO identifier based on server configuration, and a password.
func (c *Client) Login(loginId string, password string) (*User, *Response) {
	m := make(map[string]string)
	m["login_id"] = loginId
	m["password"] = password
	return c.login(m)
}

func (c *Client) GetUsersRoute() string {
	return "/users"
}

func (c *Client) GetUserRoute(userId string) string {
	return fmt.Sprintf(c.GetUsersRoute()+"/%v", userId)
}

func (c *Client) GetTagsRoute() string {
	return "/tags"
}

func (c *Client) GetInboxMessagesRoute() string {
	return "/inbox_messages"
}

func (c *Client) GetInboxMessageRoute(inboxMessageId string) string {
	return fmt.Sprintf(c.GetInboxMessagesRoute()+"/%v", inboxMessageId)
}

func (c *Client) GetQuestionsForUser(userId string) (*PostsWithCount, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId) + "/posts/questions")
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)

	return PostsWithCountFromJson(r.Body), BuildResponse(r)
}

func (c *Client) GetAnswersForUser(userId string) (*PostsWithCount, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId) + "/posts/answers")
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)

	return PostsWithCountFromJson(r.Body), BuildResponse(r)
}

func (c *Client) CreateUser(user *User) (*User, *Response) {
	r, err := c.DoApiPost(c.GetUsersRoute(), user.ToJson())
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return UserFromJson(r.Body), BuildResponse(r)
}

func (c *Client) GetMe() (*User, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(ME))
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return UserFromJson(r.Body), BuildResponse(r)
}

func (c *Client) GetUser(userId string) (*User, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId))
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return UserFromJson(r.Body), BuildResponse(r)
}

func (c *Client) VerifyUserEmail(token string) (bool, *Response) {
	requestBody := map[string]string{"token": token}
	r, err := c.DoApiPost(c.GetUsersRoute()+"/email/verify", MapToJson(requestBody))
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) SendVerificationEmail(email string) (bool, *Response) {
	requestBody := map[string]string{"email": email}
	r, err := c.DoApiPost(c.GetUsersRoute()+"/email/verify/send", MapToJson(requestBody))
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) AutocompleteTags(tagName string) (Tags, *Response) {
	query := fmt.Sprintf("?tag_name=%v", tagName)

	r, err := c.DoApiGet(c.GetTagsRoute() + "/autocomplete" + query)
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)

	tags, _ := TagsFromJson(r.Body)

	return tags, BuildResponse(r)
}

func (c *Client) GetInboxMessagesForUser(userId string) (InboxMessages, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId) + c.GetInboxMessagesRoute())
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return InboxMessagesFromJson(r.Body), BuildResponse(r)
}

func (c *Client) GetInboxMessagesUnreadCountForUser(userId string) (int, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId) + c.GetInboxMessagesRoute() + "/unread_count")
	if err != nil {
		return 0, BuildErrorResponse(r, err)
	}
	defer closeBody(r)

	responseData := struct {
		UserId                   string `json:"user_id"`
		InboxMessagesUnreadCount string `json:"inbox_messages_unread_count"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&responseData); err != nil {
		appErr := NewAppError("Api.GetInboxMessagesUnreadCountForUser", "api.marshal_error", nil, err.Error(), http.StatusInternalServerError)
		return 0, BuildErrorResponse(r, appErr)
	}

	count, _ := strconv.Atoi(responseData.InboxMessagesUnreadCount)

	return count, BuildResponse(r)
}

func (c *Client) SetInboxMessageRead(userId, inboxMessageId string) (bool, *Response) {
	r, err := c.DoApiPost(c.GetUserRoute(userId)+c.GetInboxMessageRoute(inboxMessageId)+"/set_read", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

func (c *Client) GetUserFavoritePosts(userId string) (*UserFavoritePostsWithCount, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId) + "/user_favorite_posts")
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return UserFavoritePostsWithCountFromJson(r.Body), BuildResponse(r)
}

func (c *Client) GetUserVotes(userId string) (*VotesWithCount, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId) + "/votes")
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return VotesWithCountFromJson(r.Body), BuildResponse(r)
}

func (c *Client) GetNotificationSettingForUser(userId string) (*NotificationSetting, *Response) {
	r, err := c.DoApiGet(c.GetUserRoute(userId) + "/notification_setting")
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return NotificationSettingFromJson(r.Body), BuildResponse(r)
}

func (c *Client) UpdateNotificationSettingForUser(userId, interval string) (bool, *Response) {
	query := fmt.Sprintf("?inbox_interval=%v", interval)
	r, err := c.DoApiPut(c.GetUserRoute(userId)+"/notification_setting"+query, "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return CheckStatusOK(r), BuildResponse(r)
}

// Logout terminates the current user's session.
func (c *Client) Logout() (bool, *Response) {
	r, err := c.DoApiPost("/users/logout", "")
	if err != nil {
		return false, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	c.AuthToken = ""
	c.AuthType = HEADER_BEARER
	c.SessionCookie = ""
	c.UserCookie = ""
	c.CsrfCookie = ""
	return CheckStatusOK(r), BuildResponse(r)
}

// CheckStatusOK is a convenience function for checking the standard OK response
// from the web service.
func CheckStatusOK(r *http.Response) bool {
	m := MapFromJson(r.Body)
	defer closeBody(r)

	if m != nil && m[STATUS] == STATUS_OK {
		return true
	}

	return false
}

func (c *Client) GetPostRoute(postId string) string {
	return fmt.Sprintf(c.GetPostsRoute()+"/%v", postId)
}

func (c *Client) DoApiGet(url string) (*http.Response, *AppError) {
	return c.DoApiRequest(http.MethodGet, c.ApiUrl+url, "")
}

func (c *Client) GetPost(postId string) (*Post, *Response) {
	r, err := c.DoApiGet(c.GetPostRoute(postId))
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostFromJson(r.Body), BuildResponse(r)
}

func (c *Client) GetAnswersForPost(postId string) (*PostsWithCount, *Response) {
	r, err := c.DoApiGet(c.GetPostRoute(postId) + "/answers")
	if err != nil {
		return nil, BuildErrorResponse(r, err)
	}
	defer closeBody(r)
	return PostsWithCountFromJson(r.Body), BuildResponse(r)
}
