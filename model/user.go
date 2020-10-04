package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	USER_TYPE_NORMAL    = "normal"
	USER_TYPE_MODERATOR = "moderator"
	USER_TYPE_ADMIN     = "admin"

	ME                       = "me"
	USER_NAME_MAX_LENGTH     = 64
	USER_NAME_MIN_LENGTH     = 3
	USER_EMAIL_MAX_LENGTH    = 128
	USER_PASSWORD_MAX_LENGTH = 72

	SUSPEND_SPAN_TYPE_WEEK      = "week"
	SUSPEND_SPAN_TYPE_MONTH     = "month"
	SUSPEND_SPAN_TYPE_QUARTER   = "quarter"
	SUSPEND_SPAN_TYPE_HALF_YEAR = "half_year"
	SUSPEND_SPAN_TYPE_YEAR      = "year"

	USER_PROPS_SUSPEND_BY = "suspendBy"
	USER_PROPS_DELETE_BY  = "deleteBy"
)

// TODO: user's views count:
// The number of unique visitors to user profile.
type User struct {
	Id                     string    `db:"Id, primarykey" json:"id"`
	Type                   string    `db:"Type" json:"type"`
	CreateAt               int64     `db:"CreateAt" json:"create_at"`
	UpdateAt               int64     `db:"UpdateAt" json:"update_at"`
	DeleteAt               int64     `db:"DeleteAt" json:"delete_at"`
	SuspendTime            int64     `db:"SuspendTime" json:"suspend_time,omitempty"`
	Username               string    `db:"Username" json:"username"`
	Password               string    `db:"Password" json:"password,omitempty"`
	Props                  StringMap `db:"Props" json:"-"`
	Email                  string    `db:"Email" json:"email,omitempty"`
	EmailVerified          bool      `db:"EmailVerified" json:"email_verified,omitempty"`
	Points                 int       `db:"Points" json:"points,omitempty"`
	LastInboxMessageViewed int64     `db:"LastInboxMessageViewed" json:"last_inbox_message_viewed,omitempty"`
	LastPictureUpdate      int64     `db:"LastPictureUpdate" json:"last_picture_update,omitempty"`
	FailedAttempts         int       `db:"FailedAttempts" json:"failed_attempts,omitempty"`

	QuestionCount    int64  `db:"-" json:"question_count,omitempty"`
	AnswerCount      int64  `db:"-" json:"answer_count,omitempty"`
	ProfileImageLink string `db:"-" json:"profile_image_link,omitempty`
}

type UserUpdate struct {
	Old *User
	New *User
}

func (u *User) IsValid() *AppError {
	if len(u.Id) != 26 {
		return InvalidUserError("id", "")
	}

	if u.Type != USER_TYPE_NORMAL && u.Type != USER_TYPE_MODERATOR && u.Type != USER_TYPE_ADMIN {
		return InvalidUserError("type", u.Id)
	}

	if u.CreateAt == 0 {
		return InvalidUserError("create_at", u.Id)
	}

	if u.UpdateAt == 0 {
		return InvalidUserError("update_at", u.Id)
	}

	if !IsValidUsername(u.Username) {
		return InvalidUserError("username", u.Id)
	}

	if len(u.Email) > USER_EMAIL_MAX_LENGTH || len(u.Email) == 0 || !IsValidEmail(u.Email) {
		return InvalidUserError("email", u.Id)
	}

	if len(u.Password) > USER_PASSWORD_MAX_LENGTH {
		return InvalidUserError("password_limit", u.Id)
	}

	return nil
}

func (u *User) DeepCopy() *User {
	copyUser := *u
	if u.Props != nil {
		copyUser.Props = CopyStringMap(u.Props)
	}
	return &copyUser
}

func InvalidUserError(fieldName string, userId string) *AppError {
	id := fmt.Sprintf("model.user.is_valid.%s.app_error", fieldName)
	details := ""
	if userId != "" {
		details = "user_id=" + userId
	}
	return NewAppError("User.IsValid", id, nil, details, http.StatusBadRequest)
}

var validUsernameChars = regexp.MustCompile(`^[a-z0-9\.\-_]+$`)

func IsValidUsername(s string) bool {
	if len(s) < USER_NAME_MIN_LENGTH || len(s) > USER_NAME_MAX_LENGTH {
		return false
	}

	if !validUsernameChars.MatchString(s) {
		return false
	}

	return true
}

func NormalizeUsername(username string) string {
	return strings.ToLower(username)
}

func NormalizeEmail(email string) string {
	return strings.ToLower(email)
}

func (u *User) PreSave() {
	if u.Id == "" {
		u.Id = NewId()
	}

	if u.Username == "" {
		u.Username = NewId()
	}

	u.Username = NormalizeUsername(u.Username)
	u.Email = NormalizeEmail(u.Email)

	u.CreateAt = GetMillis()
	u.UpdateAt = u.CreateAt

	if len(u.Password) > 0 {
		u.Password = HashPassword(u.Password)
	}

	u.MakeNonNil()
}

func HashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		panic(err)
	}

	return string(hash)
}

func ComparePassword(hash string, password string) bool {
	if len(password) == 0 || len(hash) == 0 {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (u *User) PreUpdate() {
	u.Username = NormalizeUsername(u.Username)
	u.Email = NormalizeEmail(u.Email)
	u.UpdateAt = GetMillis()
}

func (u *User) ToJson() string {
	b, _ := json.Marshal(u)
	return string(b)
}

func (u *User) Sanitize(options map[string]bool) {
	u.Password = ""
	u.Props = make(map[string]string)
	u.LastInboxMessageViewed = 0
	u.Type = ""
	u.LastPictureUpdate = 0
	u.FailedAttempts = 0

	if len(options) != 0 && !options["email"] {
		u.Email = ""
	}
}

func (u *User) ClearNonProfileFields() {
	u.Password = ""
	u.EmailVerified = false
	u.FailedAttempts = 0
}

func (u *User) SanitizeProfile(options map[string]bool) {
	u.ClearNonProfileFields()
	u.Sanitize(options)
}

func (u *User) SanitizeInput() {
	u.Type = USER_TYPE_NORMAL
	u.EmailVerified = false
	u.Points = 0
	u.LastInboxMessageViewed = 0
	u.LastPictureUpdate = 0
	u.FailedAttempts = 0
}

func UserFromJson(data io.Reader) *User {
	var user *User
	json.NewDecoder(data).Decode(&user)
	return user
}

func GetSuspendTimeBySpan(suspendSpan string, currentTime int64) int64 {
	days := 0

	switch suspendSpan {
	case SUSPEND_SPAN_TYPE_WEEK:
		days = 7
	case SUSPEND_SPAN_TYPE_MONTH:
		days = 30
	case SUSPEND_SPAN_TYPE_QUARTER:
		days = 90
	case SUSPEND_SPAN_TYPE_HALF_YEAR:
		days = 180
	case SUSPEND_SPAN_TYPE_YEAR:
		days = 360
	}

	milliSeconds := days * 60 * 60 * 24 * 1000

	return currentTime + int64(milliSeconds)
}

func (u *User) IsSuspending() bool {
	return u.SuspendTime > 0 && u.SuspendTime > GetMillis()
}

type GetUsersOptions struct {
	FromDate int64
	ToDate   int64
	SortType string
	Username string
	// Min and Max specify the range of a field being specified by SortType
	Min     *int
	Max     *int
	Page    int
	PerPage int
}

func UserListToJson(u []*User) string {
	b, _ := json.Marshal(u)
	return string(b)
}

func (u *User) MakeNonNil() {
	if u.Props == nil {
		u.Props = make(map[string]string)
	}
}

func (u *User) AddProp(key string, value string) {
	u.MakeNonNil()
	u.Props[key] = value
}

func CreateProfileImageKey(userId string, time int64) string {
	etag := strconv.FormatInt(time, 10)
	path := "/users/" + userId + "/" + etag + ".jpg"
	return path
}

func (u *User) GetProfileImageLink(settings *FileSettings) string {
	if u.LastPictureUpdate == 0 || *settings.AmazonCloudFrontURL == "" {
		return ""
	}

	path := (*settings.AmazonCloudFrontURL + CreateProfileImageKey(u.Id, u.LastPictureUpdate))
	return path
}
