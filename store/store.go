package store

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/go-gorp/gorp"
)

type StoreResult struct {
	Data interface{}
	Err  *model.AppError
}

type Store interface {
	DriverName() string
	GetMaster() *gorp.DbMap
	//LockToMaster()
	//UnlockFromMaster()
	GetReplica() *gorp.DbMap
	TotalMasterDbConnections() int
	TotalReadDbConnections() int
	Close()
	GetAllConns() []*gorp.DbMap
	GetQueryBuilder() sq.StatementBuilderType
	DropAllTables()

	User() UserStore
	Token() TokenStore
	Session() SessionStore
	Post() PostStore
	Tag() TagStore
	Vote() VoteStore
	UserPointHistory() UserPointHistoryStore
	InboxMessage() InboxMessageStore
	UserFavoritePost() UserFavoritePostStore
	FileInfo() FileInfoStore
	NotificationSetting() NotificationSettingStore
}

type UserStore interface {
	Save(user *model.User) (*model.User, *model.AppError)
	Update(user *model.User, trustedUpdateData bool) (*model.UserUpdate, *model.AppError)
	Get(id string) (*model.User, *model.AppError)
	GetByIds(userIds []string) ([]*model.User, *model.AppError)
	GetByEmail(email string) (*model.User, *model.AppError)
	GetUsersByDates(options *model.GetUsersOptions) ([]*model.User, *model.AppError)
	GetForLogin(loginId string) (*model.User, *model.AppError)
	GetByInboxInterval(fromUserId string, inboxInterval string, limit int) ([]*model.User, *model.AppError)
	VerifyEmail(userId, email string) (string, *model.AppError)
	UpdateLastInboxMessageViewed(message *model.InboxMessage, userId string) *model.AppError
	SuspendUser(userId string, suspendSpan string, moderatorId string) *model.AppError
	Delete(userId string, time int64, deleteById string) *model.AppError
	UpdatePassword(userId, hashedPassword string) *model.AppError
	UpdateFailedPasswordAttempts(userId string, attempts int) *model.AppError
	Count(options *model.UserCountOptions) (int64, *model.AppError)
	UpdateLastPictureUpdate(userId string, time int64) *model.AppError
}

type TokenStore interface {
	Save(recovery *model.Token) *model.AppError
	GetByToken(token string) (*model.Token, *model.AppError)
	Delete(token string) *model.AppError
}

type SessionStore interface {
	Get(sessionIdOrToken string) (*model.Session, *model.AppError)
	Save(session *model.Session) (*model.Session, *model.AppError)
	Remove(sessionIdOrToken string) *model.AppError
	RemoveByUserId(userId string) *model.AppError
}

type PostStore interface {
	SaveQuestion(post *model.Post) (*model.Post, *model.AppError)
	SaveAnswer(post *model.Post) (*model.Post, *model.AppError)
	SaveComment(post *model.Post) (*model.Post, *model.AppError)
	Update(newPost *model.Post, oldPost *model.Post) (*model.Post, *model.AppError)
	GetSingle(id string) (*model.Post, *model.AppError)
	GetSingleByType(id string, postType string) (*model.Post, *model.AppError)
	GetPostCountByUserId(postType string, userId string) (int64, *model.AppError)
	GetPostsByIds(postIds []string) (model.Posts, *model.AppError)
	GetPosts(options *model.GetPostsOptions, getCount bool) (model.Posts, int64, *model.AppError)
	SearchPosts(paramsList []*model.SearchParams, sortType string, page, perPage int) (model.Posts, int64, *model.AppError)
	GetChildPostsCount(id string) (int64, *model.AppError)
	GetCommentsForPost(postId string, limit int) ([]*model.Post, *model.AppError)
	DeleteQuestion(postId string, time int64, deleteById string) *model.AppError
	DeleteAnswer(postId string, time int64, deleteById string) *model.AppError
	DeleteComment(postId string, time int64, deleteById string) *model.AppError
	SelectBestAnswer(postId, bestId string) *model.AppError
	UpVotePost(postId string, userId string) *model.AppError
	CancelUpVotePost(postId string, userId string) *model.AppError
	DownVotePost(postId string, userId string) *model.AppError
	CancelDownVotePost(postId string, userId string) *model.AppError
	FlagPost(postId string, userId string) *model.AppError
	CancelFlagPost(postId string, userId string) *model.AppError
	LockPost(postId string, time int64, userId string) *model.AppError
	CancelLockPost(postId string, userId string) *model.AppError
	ProtectPost(postId string, time int64, userId string) *model.AppError
	CancelProtectPost(postId string, userId string) *model.AppError
}

type TagStore interface {
	GetTags(options *model.GetTagsOptions) (model.Tags, *model.AppError)
}

type VoteStore interface {
	GetVotesBeforeTime(time int64, userId string, page, perPage int, excludeFlag bool, getCount bool) ([]*model.Vote, int64, *model.AppError)
	GetByPostIdForUser(userId string, postId string, voteType string) (*model.Vote, *model.AppError)
}

type UserPointHistoryStore interface {
	GetUserPointHistoryBeforeTime(time int64, userId string, page, perPage int) ([]*model.UserPointHistory, *model.AppError)
}

type InboxMessageStore interface {
	GetSingle(id string) (*model.InboxMessage, *model.AppError)
	GetInboxMessages(time int64, userId string, direction string, page, perPage int) ([]*model.InboxMessage, *model.AppError)
	GetInboxMessagesUnreadCount(userId string, fromDate int64) (int64, *model.AppError)
	SaveInboxMessage(inboxMessage *model.InboxMessage) (*model.InboxMessage, *model.AppError)
	SaveMultipleInboxMessages(inboxMessages []*model.InboxMessage) ([]*model.InboxMessage, *model.AppError)
}

type UserFavoritePostStore interface {
	GetByPostIdForUser(userId string, postId string) (*model.UserFavoritePost, *model.AppError)
	GetCountByPostId(postId string) (int64, *model.AppError)
	GetUserFavoritePostsBeforeTime(time int64, userId string, page, perPage int, getCount bool) ([]*model.UserFavoritePost, int64, *model.AppError)
	Save(postId string, userId string) *model.AppError
	Delete(postId string, userId string) *model.AppError
}

type FileInfoStore interface {
	Save(info *model.FileInfo) (*model.FileInfo, *model.AppError)
	DeleteForPost(postId string) (string, *model.AppError)
	Get(id string) (*model.FileInfo, *model.AppError)
	AttachToPost(fileId, postId, userId string) *model.AppError
}

type NotificationSettingStore interface {
	Get(userId string) (*model.NotificationSetting, *model.AppError)
	Save(userId, inboxInterval string) *model.AppError
}
