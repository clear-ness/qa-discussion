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

	Team() TeamStore
	TeamMemberHistory() TeamMemberHistoryStore
	UserGroup() UserGroupStore
	GroupMemberHistory() GroupMemberHistoryStore
	Collection() CollectionStore
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
	PostViewsHistory() PostViewsHistoryStore
	Webhook() WebhookStore
	WebhooksHistory() WebhooksHistoryStore
	Audit() AuditStore
	OAuth() OAuthStore
	Status() StatusStore
}

type TeamStore interface {
	Get(id string) (*model.Team, *model.AppError)
	GetByInviteId(inviteId string) (*model.Team, *model.AppError)
	Save(team *model.Team) (*model.Team, *model.AppError)
	GetMember(teamId string, userId string) (*model.TeamMember, *model.AppError)
	GetMembers(teamId string, offset int, limit int, teamMembersGetOptions *model.TeamMembersGetOptions) ([]*model.TeamMember, *model.AppError)
	GetMembersByIds(teamId string, userIds []string) ([]*model.TeamMember, *model.AppError)
	SaveMember(member *model.TeamMember, maxUsersPerTeam int) (*model.TeamMember, *model.AppError)
	SaveMultipleMembers(members []*model.TeamMember, maxUsersPerTeam int) ([]*model.TeamMember, *model.AppError)
	GetActiveMemberCount(teamId string) (int64, *model.AppError)
	UpdateMember(member *model.TeamMember) (*model.TeamMember, *model.AppError)
	UpdateMultipleMembers(members []*model.TeamMember) ([]*model.TeamMember, *model.AppError)
	GetTeamsByUserId(userId string) ([]*model.Team, *model.AppError)
	GetTeamsForUser(userId string) ([]*model.TeamMember, *model.AppError)
	Update(team *model.Team) (*model.Team, *model.AppError)
	PermanentDelete(teamId string) *model.AppError
	RemoveAllMembersByTeam(teamId string) *model.AppError
	UpdateLastTeamIconUpdate(teamId string, curTime int64) *model.AppError
	AutocompletePublic(name string) ([]*model.Team, *model.AppError)
}

type TeamMemberHistoryStore interface {
	LogJoinEvent(userId string, teamId string, joinTime int64) error
	LogLeaveEvent(userId string, teamId string, leaveTime int64) error
}

type UserGroupStore interface {
	GetTeamGroups(teamId string) (*model.UserGroupList, *model.AppError)
	Save(group *model.UserGroup, maxGroupsPerTeam int64) (*model.UserGroup, *model.AppError)
	SaveMember(member *model.GroupMember) (*model.GroupMember, *model.AppError)
	SaveMultipleMembers(members []*model.GroupMember) ([]*model.GroupMember, *model.AppError)
	GetGroupsForTeam(teamId string, groupType string, offset int, limit int) (*model.UserGroupList, *model.AppError)
	AutocompleteInTeam(teamId string, term string, groupType string, includeDeleted bool) (*model.UserGroupList, *model.AppError)
	GetGroups(teamId string, userId string, includeDeleted bool) (*model.UserGroupList, *model.AppError)
	Get(id string) (*model.UserGroup, *model.AppError)
	GetAllGroupMembersForUser(userId string) (map[string]string, *model.AppError)
	Update(group *model.UserGroup) (*model.UserGroup, *model.AppError)
	Delete(groupId string, time int64) *model.AppError
	GetMembers(groupId string, memberType string, offset, limit int) (*model.GroupMembers, *model.AppError)
	GetMember(groupId string, userId string) (*model.GroupMember, *model.AppError)
	UpdateMember(member *model.GroupMember) (*model.GroupMember, *model.AppError)
	UpdateMultipleMembers(members []*model.GroupMember) ([]*model.GroupMember, *model.AppError)
	RemoveMembers(groupId string, userIds []string) *model.AppError
	RemoveMember(groupId string, userId string) *model.AppError
}

type GroupMemberHistoryStore interface {
	LogJoinEvent(userId string, groupId string, joinTime int64) error
	LogLeaveEvent(userId string, groupId string, leaveTime int64) error
}

type CollectionStore interface {
	Get(id string) (*model.Collection, *model.AppError)
	GetPost(collectionId string, postId string) (*model.CollectionPost, *model.AppError)
	GetPosts(collectionId string, offset, limit int) (*model.CollectionPosts, *model.AppError)
	GetCollectionsForTeam(teamId string, offset int, limit int, title string) (*model.CollectionList, *model.AppError)
	GetTeamCollections(teamId string) (*model.CollectionList, *model.AppError)
	Save(collection *model.Collection, maxCollectionsPerTeam int64) (*model.Collection, *model.AppError)
	SavePost(colPost *model.CollectionPost) (*model.CollectionPost, *model.AppError)
	SaveMultiplePosts(colPosts []*model.CollectionPost) ([]*model.CollectionPost, *model.AppError)
	RemovePost(collectionId string, postId string) *model.AppError
	RemovePosts(collectionId string, postIds []string) *model.AppError
	Delete(collectionId string, time int64) *model.AppError
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
	GetSessions(userId string) ([]*model.Session, *model.AppError)
}

type PostStore interface {
	SaveQuestion(post *model.Post) (*model.Post, *model.AppError)
	SaveAnswer(post *model.Post) (*model.Post, *model.AppError)
	SaveComment(post *model.Post) (*model.Post, *model.AppError)
	Update(newPost *model.Post, oldPost *model.Post) (*model.Post, *model.AppError)
	GetSingle(id string, includeDeleted bool) (*model.Post, *model.AppError)
	GetSingleByType(id string, postType string) (*model.Post, *model.AppError)
	GetPostCount(postType string, userId string, teamId string, fromDate int64, toDate int64) (int64, *model.AppError)
	GetPostsByIds(postIds []string) (model.Posts, *model.AppError)
	GetPosts(options *model.GetPostsOptions, getCount bool) (model.Posts, int64, *model.AppError)
	SearchPosts(paramsList []*model.SearchParams, sortType string, page, perPage int, teamId string) (model.Posts, int64, *model.AppError)
	GetChildPostsCount(id string) (int64, *model.AppError)
	GetCommentsForPost(postId string, limit int) ([]*model.Post, *model.AppError)
	DeleteQuestion(postId string, time int64, deleteById string) *model.AppError
	DeleteAnswer(postId string, time int64, deleteById string) *model.AppError
	DeleteComment(postId string, time int64, deleteById string) *model.AppError
	SelectBestAnswer(postId, bestId string) *model.AppError
	UpVotePost(postId string, userId string) (*model.Vote, *model.AppError)
	CancelUpVotePost(postId string, userId string) (*model.Vote, *model.AppError)
	DownVotePost(postId string, userId string) (*model.Vote, *model.AppError)
	CancelDownVotePost(postId string, userId string) (*model.Vote, *model.AppError)
	FlagPost(postId string, userId string) (*model.Vote, *model.AppError)
	CancelFlagPost(postId string, userId string) (*model.Vote, *model.AppError)
	LockPost(postId string, time int64, userId string) *model.AppError
	CancelLockPost(postId string, userId string) *model.AppError
	ProtectPost(postId string, time int64, userId string) *model.AppError
	CancelProtectPost(postId string, userId string) *model.AppError
	ViewPost(postId string, teamId string, userId string, ipAddress string, count int) *model.AppError
	SavePostViewsHistory(postId string, teamId string, userId string, ipAddress string, count int, time int64) (*model.PostViewsHistory, *model.AppError)
	RelatedSearch(term string, limit int) ([]*model.RelatedPostSearchResult, *model.AppError)
	HotSearch(interval string, teamId string, limit int) ([]string, *model.AppError)
	GetCurrentRevisionForPost(postId, teamId string) (int64, *model.AppError)
	GetRevisionPost(postId, teamId string, offset int) (*model.Post, *model.AppError)
	GetAnsweredRate(teamId string) (float64, *model.AppError)
	AnalyticsPostCounts(teamId string) (model.Analytics, *model.AppError)
	AnalyticsActiveAuthorCounts(teamId string) (model.Analytics, *model.AppError)
	SaveUserPointHistory(history *model.UserPointHistory) (*model.UserPointHistory, *model.AppError)
}

type TagStore interface {
	GetTags(options *model.GetTagsOptions) (model.Tags, *model.AppError)
	GetTagsCount(options *model.GetTagsOptions) (int64, *model.AppError)
	CreateTags(addedTags []string, time int64, teamId string, tagType string) *model.AppError
}

type VoteStore interface {
	GetVotesBeforeTime(time int64, userId string, page, perPage int, excludeFlag bool, getCount bool, teamId string) ([]*model.Vote, int64, *model.AppError)
	GetByPostIdForUser(userId string, postId string, voteType string) (*model.Vote, *model.AppError)
	GetVoteTypesForPost(userId string, postId string) ([]string, *model.AppError)
	CreateReviewVote(post *model.Post, userId string, tagContents string, revision int64) (*model.Vote, *model.AppError)
	GetRejectedReviewsCount(postId string, currentRevision int64) (int64, *model.AppError)
	RejectReviewsForPost(postId string, rejectedBy string, revision int64) *model.AppError
	CompleteReviewsForPost(postId string, completedBy string, revision int64) *model.AppError
	GetReviews(options *model.SearchReviewsOptions, getCount bool) ([]*model.Vote, int64, *model.AppError)
	AnalyticsVoteCounts(teamId string, voteType string) (model.Analytics, *model.AppError)
}

type UserPointHistoryStore interface {
	GetUserPointHistoryBeforeTime(time int64, userId string, page, perPage int, teamId string) ([]*model.UserPointHistory, *model.AppError)
	TopAskersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopUserByTagResult, *model.AppError)
	TopAnswerersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopUserByTagResult, *model.AppError)
	TopAnswersByTag(interval string, teamId string, tag string, limit int) ([]*model.TopPostByTagResult, *model.AppError)
}

type InboxMessageStore interface {
	GetSingle(id string) (*model.InboxMessage, *model.AppError)
	GetInboxMessages(time int64, userId string, direction string, page, perPage int, teamId string) ([]*model.InboxMessage, *model.AppError)
	GetInboxMessagesUnreadCount(userId string, fromDate int64, teamId string) (int64, *model.AppError)
	SaveInboxMessage(inboxMessage *model.InboxMessage) (*model.InboxMessage, *model.AppError)
	SaveMultipleInboxMessages(inboxMessages []*model.InboxMessage) ([]*model.InboxMessage, *model.AppError)
}

type UserFavoritePostStore interface {
	GetByPostIdForUser(userId string, postId string) (*model.UserFavoritePost, *model.AppError)
	GetCountByPostId(postId string) (int64, *model.AppError)
	GetUserFavoritePostsBeforeTime(time int64, userId string, page, perPage int, getCount bool, teamId string) ([]*model.UserFavoritePost, int64, *model.AppError)
	Save(postId string, userId string, teamId string) *model.AppError
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

type PostViewsHistoryStore interface {
	GetViewsHistoryCount(teamId string, fromDate int64, toDate int64) (int64, *model.AppError)
	AnalyticsPostViewsHistoryCounts(teamId string) (model.Analytics, *model.AppError)
}

type WebhookStore interface {
	GetByTeam(teamId string, userId string, offset, limit int) ([]*model.Webhook, *model.AppError)
	Save(webhook *model.Webhook) (*model.Webhook, *model.AppError)
	Get(id string) (*model.Webhook, *model.AppError)
	Update(hook *model.Webhook) (*model.Webhook, *model.AppError)
	Delete(webhookId string, time int64) *model.AppError
}

type WebhooksHistoryStore interface {
	LogWebhookEvent(history *model.WebhooksHistory) error
	GetWebhooksHistoriesPage(teamId string, offset, limit int) ([]*model.WebhooksHistory, *model.AppError)
}

type AuditStore interface {
	Get(user_id string, offset int, limit int) (model.Audits, *model.AppError)
	Save(audit *model.Audit) *model.AppError
	PermanentDeleteByUser(userId string) *model.AppError
}

type OAuthStore interface {
	SaveApp(app *model.OAuthApp) (*model.OAuthApp, *model.AppError)
	UpdateApp(app *model.OAuthApp) (*model.OAuthApp, *model.AppError)
	GetApp(id string) (*model.OAuthApp, *model.AppError)
	GetAppByUserId(userId string, offset, limit int) ([]*model.OAuthApp, *model.AppError)
	DeleteApp(id string) *model.AppError
	GetAuthorizedApps(userId string, offset, limit int) ([]*model.OAuthApp, *model.AppError)
	SaveAuthData(authData *model.AuthData) (*model.AuthData, *model.AppError)
	SaveAuthorizedApp(app *model.OAuthAuthorizedApp) *model.AppError
	GetAccessData(token string) (*model.AccessData, *model.AppError)
	RemoveAccessData(token string) *model.AppError
	SaveAccessData(accessData *model.AccessData) (*model.AccessData, *model.AppError)
	GetAuthData(code string) (*model.AuthData, *model.AppError)
	RemoveAuthData(code string) *model.AppError
	GetPreviousAccessData(userId, clientId string) (*model.AccessData, *model.AppError)
	UpdateAccessData(accessData *model.AccessData) (*model.AccessData, *model.AppError)
	GetAccessDataByRefreshToken(token string) (*model.AccessData, *model.AppError)
	GetAccessDataByUserForApp(userId, clientId string) ([]*model.AccessData, *model.AppError)
	DeleteAuthorizedApp(userId string, clientId string) *model.AppError
}

type StatusStore interface {
	Get(userId string) (*model.Status, *model.AppError)
	GetByIds(userIds []string) ([]*model.Status, error)
	SaveOrUpdate(status *model.Status) error
	UpdateLastActivityAt(userId string, lastActivityAt int64) error
}
