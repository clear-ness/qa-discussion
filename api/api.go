package api

import (
	"fmt"
	"net/http"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/web"
	"github.com/gorilla/mux"
)

type Routes struct {
	Root    *mux.Router // ''
	ApiRoot *mux.Router // 'api/v1'

	Teams              *mux.Router // 'api/v1/teams'
	TeamsForUser       *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/teams'
	Team               *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}'
	TeamForUser        *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/teams/{team_id:[A-Za-z0-9]+}'
	TeamMembers        *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/members'
	TeamMember         *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/members/{user_id:[A-Za-z0-9]+}'
	TeamMembersForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/teams/members'

	Groups        *mux.Router // 'api/v1/groups'
	GroupsForTeam *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/groups'
	Group         *mux.Router // 'api/v1/groups/{group_id:[A-Za-z0-9]+}'
	GroupMembers  *mux.Router // 'api/v1/groups/{group_id:[A-Za-z0-9]+}/members'
	GroupMember   *mux.Router // 'api/v1/groups/{group_id:[A-Za-z0-9]+}/members/{user_id:[A-Za-z0-9]+}'

	Collections        *mux.Router // 'api/v1/collections'
	CollectionsForTeam *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/collections'
	Collection         *mux.Router // 'api/v1/collections/{collection_id:[A-Za-z0-9]+}'
	CollectionPosts    *mux.Router // 'api/v1/collections/{collection_id:[A-Za-z0-9]+}/posts'
	CollectionPost     *mux.Router // 'api/v1/collections/{collection_id:[A-Za-z0-9]+}/posts/{post_id:[A-Za-z0-9]+}'

	Users        *mux.Router // 'api/v1/users'
	User         *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}'
	UsersForTeam *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/users'
	UserForTeam  *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/users/{user_id:[A-Za-z0-9]+}'

	Tags        *mux.Router // 'api/v1/tags'
	TagsForTeam *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/tags'

	Posts        *mux.Router // 'api/v1/posts'
	Post         *mux.Router // 'api/v1/posts/{post_id:[A-Za-z0-9]+}'
	PostsForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/posts'
	PostsForTeam *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/posts'
	PostForTeam  *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/posts/{post_id:[A-Za-z0-9]+}'

	Files *mux.Router // 'api/v1/files'
	File  *mux.Router // 'api/v1/files/{file_id:[A-Za-z0-9]+}'

	InboxMessagesForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/inbox_messages'
	InboxMessageForUser  *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/inbox_messages/{inbox_message_id:[A-Za-z0-9]+}'

	UserPointHistoryForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/user_point_history'

	VotesForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/votes'

	UserFavoritePosts       *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/user_favorite_posts'
	TeamMemberFavoritePosts *mux.Router // 'api/v1/teams/{team_id:[A-Za-z0-9]+}/members/{user_id:[A-Za-z0-9]+}/user_favorite_posts'

	NotificationSettingForUser *mux.Router // 'api/v1/users/{user_id:[A-Za-z0-9]+}/notification_setting'
}

type API struct {
	GetGlobalAppOptions app.AppOptionCreator
	BaseRoutes          *Routes
}

func Init(globalOptionsFunc app.AppOptionCreator, root *mux.Router) *API {
	api := &API{
		GetGlobalAppOptions: globalOptionsFunc,
		BaseRoutes:          &Routes{},
	}

	api.BaseRoutes.Root = root
	api.BaseRoutes.ApiRoot = root.PathPrefix(model.API_URL_SUFFIX).Subrouter()

	api.BaseRoutes.Teams = api.BaseRoutes.ApiRoot.PathPrefix("/teams").Subrouter()
	api.BaseRoutes.TeamsForUser = api.BaseRoutes.User.PathPrefix("/teams").Subrouter()
	api.BaseRoutes.Team = api.BaseRoutes.Teams.PathPrefix("/{team_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.TeamForUser = api.BaseRoutes.TeamsForUser.PathPrefix("/{team_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.TeamMembers = api.BaseRoutes.Team.PathPrefix("/members").Subrouter()
	api.BaseRoutes.TeamMember = api.BaseRoutes.TeamMembers.PathPrefix("/{user_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.TeamMembersForUser = api.BaseRoutes.User.PathPrefix("/teams/members").Subrouter()

	api.BaseRoutes.Groups = api.BaseRoutes.ApiRoot.PathPrefix("/groups").Subrouter()
	api.BaseRoutes.GroupsForTeam = api.BaseRoutes.Team.PathPrefix("/groups").Subrouter()
	api.BaseRoutes.Group = api.BaseRoutes.Groups.PathPrefix("/{group_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.GroupMembers = api.BaseRoutes.Group.PathPrefix("/members").Subrouter()
	api.BaseRoutes.GroupMember = api.BaseRoutes.GroupMembers.PathPrefix("/{user_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.Collections = api.BaseRoutes.ApiRoot.PathPrefix("/collections").Subrouter()
	api.BaseRoutes.CollectionsForTeam = api.BaseRoutes.Team.PathPrefix("/collections").Subrouter()
	api.BaseRoutes.Collection = api.BaseRoutes.Collections.PathPrefix("/{collection_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.CollectionPosts = api.BaseRoutes.Collection.PathPrefix("/posts").Subrouter()
	api.BaseRoutes.CollectionPost = api.BaseRoutes.CollectionPosts.PathPrefix("/{post_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.Users = api.BaseRoutes.ApiRoot.PathPrefix("/users").Subrouter()
	api.BaseRoutes.User = api.BaseRoutes.ApiRoot.PathPrefix("/users/{user_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.UsersForTeam = api.BaseRoutes.Team.PathPrefix("/users").Subrouter()
	api.BaseRoutes.UserForTeam = api.BaseRoutes.UsersForTeam.PathPrefix("/{user_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.Tags = api.BaseRoutes.ApiRoot.PathPrefix("/tags").Subrouter()
	api.BaseRoutes.TagsForTeam = api.BaseRoutes.Team.PathPrefix("/tags").Subrouter()

	api.BaseRoutes.Posts = api.BaseRoutes.ApiRoot.PathPrefix("/posts").Subrouter()
	api.BaseRoutes.Post = api.BaseRoutes.Posts.PathPrefix("/{post_id:[A-Za-z0-9]+}").Subrouter()
	api.BaseRoutes.PostsForUser = api.BaseRoutes.User.PathPrefix("/posts").Subrouter()
	api.BaseRoutes.PostsForTeam = api.BaseRoutes.Team.PathPrefix("/posts").Subrouter()
	api.BaseRoutes.PostForTeam = api.BaseRoutes.PostsForTeam.PathPrefix("/{post_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.Files = api.BaseRoutes.ApiRoot.PathPrefix("/files").Subrouter()
	api.BaseRoutes.File = api.BaseRoutes.ApiRoot.PathPrefix("/files/{file_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.InboxMessagesForUser = api.BaseRoutes.User.PathPrefix("/inbox_messages").Subrouter()
	api.BaseRoutes.InboxMessageForUser = api.BaseRoutes.InboxMessagesForUser.PathPrefix("/{inbox_message_id:[A-Za-z0-9]+}").Subrouter()

	api.BaseRoutes.UserPointHistoryForUser = api.BaseRoutes.User.PathPrefix("/user_point_history").Subrouter()

	api.BaseRoutes.VotesForUser = api.BaseRoutes.User.PathPrefix("/votes").Subrouter()

	api.BaseRoutes.UserFavoritePosts = api.BaseRoutes.User.PathPrefix("/user_favorite_posts").Subrouter()
	api.BaseRoutes.TeamMemberFavoritePosts = api.BaseRoutes.TeamMember.PathPrefix("/user_favorite_posts").Subrouter()

	api.BaseRoutes.NotificationSettingForUser = api.BaseRoutes.User.PathPrefix("/notification_setting").Subrouter()

	api.InitTeam()
	api.InitGroup()
	api.InitCollection()
	api.InitUser()
	api.InitPost()
	api.InitTag()
	api.InitUserFavoritePost()
	api.InitFile()
	api.InitNotificationSetting()

	root.Handle("/api/v1/{anything:.*}", http.HandlerFunc(hello))

	return api
}

func hello(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "hello")
}

var ReturnStatusOK = web.ReturnStatusOK
